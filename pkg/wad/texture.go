package wad

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"
	"strings"
)

var (
	// ErrTextureNotFound is returned when a requested texture name is not found in TEXTURE1/TEXTURE2.
	ErrTextureNotFound = errors.New("texture not found")
	// ErrFlatNotFound is returned when a requested flat name is not found.
	ErrFlatNotFound = errors.New("flat not found")
)

// Texture represents a composite wall texture formed from one or more patches.
type Texture struct {
	Name   string
	Width  int
	Height int
	Pixels []byte // Palette indices (0-255), length = Width * Height
	Mask   []bool // true if opaque, false if transparent
}

// PixelAt returns the palette index and opacity at (x, y) with horizontal and vertical wrapping.
func (t *Texture) PixelAt(x, y int) (byte, bool) {
	if t.Width <= 0 || t.Height <= 0 || len(t.Pixels) == 0 {
		return 0, false
	}
	wx := ((x % t.Width) + t.Width) % t.Width
	wy := ((y % t.Height) + t.Height) % t.Height
	idx := wy*t.Width + wx
	return t.Pixels[idx], t.Mask[idx]
}

// Flat represents a 64x64 raw floor/ceiling texture lump.
type Flat struct {
	Name   string
	Width  int
	Height int
	Pixels []byte // 4096 palette indices
}

// PixelAt returns the palette index for flat coordinate (x, y) with 64x64 wrapping.
func (f *Flat) PixelAt(x, y int) byte {
	if len(f.Pixels) < 4096 {
		return 0
	}
	u := x & 63
	v := y & 63
	return f.Pixels[v*64+u]
}

type rawMapPatch struct {
	OriginX  int16
	OriginY  int16
	Patch    int16
	StepDir  int16
	ColorMap int16
}

type rawTextureDef struct {
	Name            [8]byte
	Masked          uint32
	Width           int16
	Height          int16
	ColumnDirectory int32
	NumPatches      int16
}

// TextureDef holds the definition of a texture from TEXTURE1/2 before patches are resolved.
type TextureDef struct {
	Name    string
	Width   int
	Height  int
	Patches []TexturePatchDef
}

// TexturePatchDef defines a patch placement within a texture.
type TexturePatchDef struct {
	OriginX    int
	OriginY    int
	PatchIndex int
}

// TextureManager manages loading, compositing, and caching of wall textures, flats, palettes, and colormaps.
type TextureManager struct {
	wad         *WAD
	palette     []byte          // 768 bytes from PLAYPAL palette 0
	paletteRGBA [256]color.RGBA // fast RGBA lookup
	colormap    []byte          // 34 * 256 bytes from COLORMAP
	pnames      []string        // Patch names from PNAMES lump
	textureDefs map[string]TextureDef
	textures    map[string]*Texture
	flats       map[string]*Flat
}

// NewTextureManager creates and initializes a TextureManager from the given WAD.
func NewTextureManager(w *WAD) (*TextureManager, error) {
	if w == nil {
		return nil, errors.New("cannot create TextureManager with nil WAD")
	}

	tm := &TextureManager{
		wad:         w,
		textureDefs: make(map[string]TextureDef),
		textures:    make(map[string]*Texture),
		flats:       make(map[string]*Flat),
	}

	// 1. Load PLAYPAL (palette 0)
	if playpal, err := w.GetLump("PLAYPAL"); err == nil && len(playpal) >= 768 {
		tm.palette = playpal[:768]
		for i := 0; i < 256; i++ {
			tm.paletteRGBA[i] = color.RGBA{
				R: tm.palette[i*3],
				G: tm.palette[i*3+1],
				B: tm.palette[i*3+2],
				A: 255,
			}
		}
	} else {
		// Fallback default grayscale palette
		tm.palette = make([]byte, 768)
		for i := 0; i < 256; i++ {
			tm.palette[i*3] = byte(i)
			tm.palette[i*3+1] = byte(i)
			tm.palette[i*3+2] = byte(i)
			tm.paletteRGBA[i] = color.RGBA{R: byte(i), G: byte(i), B: byte(i), A: 255}
		}
	}

	// 2. Load COLORMAP (34 tables of 256 bytes)
	if colormap, err := w.GetLump("COLORMAP"); err == nil && len(colormap) >= 34*256 {
		tm.colormap = colormap
	}

	// 3. Load PNAMES
	if pnamesData, err := w.GetLump("PNAMES"); err == nil && len(pnamesData) >= 4 {
		numPatches := binary.LittleEndian.Uint32(pnamesData[:4])
		tm.pnames = make([]string, numPatches)
		for i := 0; i < int(numPatches); i++ {
			offset := 4 + i*8
			if offset+8 <= len(pnamesData) {
				nameBytes := pnamesData[offset : offset+8]
				name := strings.ToUpper(string(bytes.TrimRight(nameBytes, "\x00")))
				tm.pnames[i] = name
			}
		}
	}

	// 4. Load TEXTURE1 and TEXTURE2 definitions
	for _, lumpName := range []string{"TEXTURE1", "TEXTURE2"} {
		if texData, err := w.GetLump(lumpName); err == nil {
			_ = tm.parseTextureLump(texData)
		}
	}

	return tm, nil
}

func (tm *TextureManager) parseTextureLump(data []byte) error {
	if len(data) < 4 {
		return errors.New("texture lump too short")
	}

	numTextures := int(binary.LittleEndian.Uint32(data[:4]))
	if len(data) < 4+numTextures*4 {
		return errors.New("texture lump too short for offset table")
	}

	for i := 0; i < numTextures; i++ {
		offset := int(binary.LittleEndian.Uint32(data[4+i*4 : 8+i*4]))
		if offset < 0 || offset >= len(data) {
			continue
		}

		reader := bytes.NewReader(data[offset:])
		var rawHeader rawTextureDef
		if err := binary.Read(reader, binary.LittleEndian, &rawHeader); err != nil {
			continue
		}

		texName := strings.ToUpper(string(bytes.TrimRight(rawHeader.Name[:], "\x00")))
		w := int(rawHeader.Width)
		h := int(rawHeader.Height)
		numP := int(rawHeader.NumPatches)

		patchDefs := make([]TexturePatchDef, 0, numP)
		for p := 0; p < numP; p++ {
			var rawP rawMapPatch
			if err := binary.Read(reader, binary.LittleEndian, &rawP); err != nil {
				break
			}
			patchDefs = append(patchDefs, TexturePatchDef{
				OriginX:    int(rawP.OriginX),
				OriginY:    int(rawP.OriginY),
				PatchIndex: int(rawP.Patch),
			})
		}

		tm.textureDefs[texName] = TextureDef{
			Name:    texName,
			Width:   w,
			Height:  h,
			Patches: patchDefs,
		}
	}

	return nil
}

// Palette returns the 768-byte RGB palette 0.
func (tm *TextureManager) Palette() []byte {
	return tm.palette
}

// PaletteRGBA returns the precomputed 256 RGBA color array.
func (tm *TextureManager) PaletteRGBA() [256]color.RGBA {
	return tm.paletteRGBA
}

// ColorMap returns the colormap data slice (34 colormaps of 256 bytes each).
func (tm *TextureManager) ColorMap() []byte {
	return tm.colormap
}

// MapColor maps a palette index through a colormap level (0-31 for standard light levels).
func (tm *TextureManager) MapColor(colormapIndex int, palIndex byte) byte {
	if len(tm.colormap) < 32*256 {
		return palIndex
	}
	if colormapIndex < 0 {
		colormapIndex = 0
	} else if colormapIndex > 31 {
		colormapIndex = 31
	}
	return tm.colormap[colormapIndex*256+int(palIndex)]
}

// LightToColormap maps Doom sector light level (0-255) and distance to a colormap level (0-31).
func (tm *TextureManager) LightToColormap(lightLevel int, distance float64) int {
	// Doom light level diminishing formula approximation
	if lightLevel < 0 {
		lightLevel = 0
	} else if lightLevel > 255 {
		lightLevel = 255
	}

	// Base colormap from sector light level: 255 -> 0, 0 -> 31
	baseMap := (255 - lightLevel) / 8

	// Distance attenuation: light diminishes as distance increases
	distOffset := int(distance / 64.0)
	cm := baseMap + distOffset
	if cm < 0 {
		return 0
	}
	if cm > 31 {
		return 31
	}
	return cm
}

// GetTexture returns the composite wall texture with the given name, loading and caching it if needed.
func (tm *TextureManager) GetTexture(name string) (*Texture, error) {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "" || upper == "-" {
		return nil, ErrTextureNotFound
	}

	if tex, ok := tm.textures[upper]; ok {
		return tex, nil
	}

	def, ok := tm.textureDefs[upper]
	if !ok {
		// Fallback: check if the texture name directly exists as a single patch lump
		if patch, err := tm.wad.GetPatch(upper); err == nil {
			tex := &Texture{
				Name:   upper,
				Width:  patch.Width,
				Height: patch.Height,
				Pixels: patch.Pixels,
				Mask:   patch.Mask,
			}
			tm.textures[upper] = tex
			return tex, nil
		}
		return nil, fmt.Errorf("%w: %s", ErrTextureNotFound, upper)
	}

	// Composite texture from patches
	w := def.Width
	h := def.Height
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid texture dimensions %dx%d for %s", w, h, upper)
	}

	pixels := make([]byte, w*h)
	mask := make([]bool, w*h)

	for _, pdef := range def.Patches {
		if pdef.PatchIndex < 0 || pdef.PatchIndex >= len(tm.pnames) {
			continue
		}
		patchName := tm.pnames[pdef.PatchIndex]
		patch, err := tm.wad.GetPatch(patchName)
		if err != nil {
			continue
		}

		for px := 0; px < patch.Width; px++ {
			tx := pdef.OriginX + px
			if tx < 0 || tx >= w {
				continue
			}

			for py := 0; py < patch.Height; py++ {
				ty := pdef.OriginY + py
				if ty < 0 || ty >= h {
					continue
				}

				palIdx, opaque := patch.PixelAt(px, py)
				if opaque {
					idx := ty*w + tx
					pixels[idx] = palIdx
					mask[idx] = true
				}
			}
		}
	}

	tex := &Texture{
		Name:   upper,
		Width:  w,
		Height: h,
		Pixels: pixels,
		Mask:   mask,
	}

	tm.textures[upper] = tex
	return tex, nil
}

// GetFlat returns the 64x64 flat texture with the given lump name, loading and caching it if needed.
func (tm *TextureManager) GetFlat(name string) (*Flat, error) {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "" || upper == "-" {
		return nil, ErrFlatNotFound
	}

	if flat, ok := tm.flats[upper]; ok {
		return flat, nil
	}

	data, err := tm.wad.GetLump(upper)
	if err != nil {
		return nil, fmt.Errorf("%w: %s (%v)", ErrFlatNotFound, upper, err)
	}

	if len(data) < 4096 {
		// If shorter than 4096 bytes, pad to 4096 bytes
		padded := make([]byte, 4096)
		copy(padded, data)
		data = padded
	}

	flat := &Flat{
		Name:   upper,
		Width:  64,
		Height: 64,
		Pixels: data[:4096],
	}

	tm.flats[upper] = flat
	return flat, nil
}

// PreloadMap preloads and caches all wall textures and flats referenced by a MapData.
func (tm *TextureManager) PreloadMap(md *MapData) {
	if md == nil {
		return
	}

	for _, s := range md.Sidedefs {
		if s.UpperTexture != "" && s.UpperTexture != "-" {
			_, _ = tm.GetTexture(s.UpperTexture)
		}
		if s.MiddleTexture != "" && s.MiddleTexture != "-" {
			_, _ = tm.GetTexture(s.MiddleTexture)
		}
		if s.LowerTexture != "" && s.LowerTexture != "-" {
			_, _ = tm.GetTexture(s.LowerTexture)
		}
	}

	for _, sec := range md.Sectors {
		if sec.FloorPic != "" && sec.FloorPic != "-" {
			_, _ = tm.GetFlat(sec.FloorPic)
		}
		if sec.CeilingPic != "" && sec.CeilingPic != "-" {
			_, _ = tm.GetFlat(sec.CeilingPic)
		}
	}
}
