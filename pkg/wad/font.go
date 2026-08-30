package wad

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// DefaultSpaceWidth is the horizontal pixel advance for the space character.
	DefaultSpaceWidth = 4
	// DefaultLineHeight is the vertical line advance for multi-line text.
	DefaultLineHeight = 8
)

// Glyph holds a single character patch and its authentic palette-rendered Ebitengine texture.
type Glyph struct {
	Char  rune
	Patch *Patch
	Image *ebiten.Image
}

// HUDFont represents the Doom HUD message font loaded from WAD lumps (STCFNxxx)
// and rendered using the WAD's PLAYPAL palette.
type HUDFont struct {
	glyphs     map[rune]*Glyph
	lineHeight int
	spaceWidth int
}

// NewHUDFont loads all available HUD font glyphs (STCFN033 - STCFN125) from the provided WAD,
// rendering each glyph in its authentic PLAYPAL palette colors.
func NewHUDFont(w *WAD) (*HUDFont, error) {
	playpalData, err := w.GetLump("PLAYPAL")
	if err != nil {
		return nil, fmt.Errorf("failed to read PLAYPAL lump: %w", err)
	}
	if len(playpalData) < 768 {
		return nil, fmt.Errorf("PLAYPAL lump too short (%d bytes, expected >= 768)", len(playpalData))
	}

	glyphs := make(map[rune]*Glyph)
	maxHeight := 0

	for r := rune(33); r <= 126; r++ {
		lumpName := fmt.Sprintf("STCFN%03d", r)
		data, err := w.GetLump(lumpName)
		if err != nil {
			continue
		}

		patch, err := ParsePatch(data)
		if err != nil {
			continue
		}

		// Convert patch to RGBA using the normal PLAYPAL palette (palette 0)
		rgba := image.NewRGBA(image.Rect(0, 0, patch.Width, patch.Height))
		for y := 0; y < patch.Height; y++ {
			for x := 0; x < patch.Width; x++ {
				palIdx, opaque := patch.PixelAt(x, y)
				if !opaque {
					continue
				}
				pr := playpalData[int(palIdx)*3]
				pg := playpalData[int(palIdx)*3+1]
				pb := playpalData[int(palIdx)*3+2]
				rgba.SetRGBA(x, y, color.RGBA{R: pr, G: pg, B: pb, A: 255})
			}
		}

		glyphImg := ebiten.NewImageFromImage(rgba)

		glyphs[r] = &Glyph{
			Char:  r,
			Patch: patch,
			Image: glyphImg,
		}
		if patch.Height > maxHeight {
			maxHeight = patch.Height
		}
	}

	if len(glyphs) == 0 {
		return nil, fmt.Errorf("no STCFN font lumps found in WAD")
	}

	lineH := maxHeight + 1
	if lineH < DefaultLineHeight {
		lineH = DefaultLineHeight
	}

	return &HUDFont{
		glyphs:     glyphs,
		lineHeight: lineH,
		spaceWidth: DefaultSpaceWidth,
	}, nil
}

// GetGlyph returns the glyph for a character. Lower-case 'y' is specially mapped
// to upper-case 'Y' to account for a quirk in the WAD font lumps (where STCFN121
// contains an unintended graphic).
func (f *HUDFont) GetGlyph(r rune) *Glyph {
	// Quirk fix: always render 'y' as upper-case 'Y'
	if r == 'y' {
		if g, ok := f.glyphs['Y']; ok {
			return g
		}
	}
	if g, ok := f.glyphs[r]; ok {
		return g
	}
	if r >= 'a' && r <= 'z' {
		upper := r - ('a' - 'A')
		if g, ok := f.glyphs[upper]; ok {
			return g
		}
	}
	return nil
}

// HasGlyph returns true if the font can render the given character.
func (f *HUDFont) HasGlyph(r rune) bool {
	return f.GetGlyph(r) != nil
}

// LineHeight returns the line height in pixels.
func (f *HUDFont) LineHeight() int {
	return f.lineHeight
}

// SpaceWidth returns the space advance in pixels.
func (f *HUDFont) SpaceWidth() int {
	return f.spaceWidth
}

// MeasureText calculates the width and height required to render the string.
func (f *HUDFont) MeasureText(text string) (width, height int) {
	lines := strings.Split(text, "\n")
	maxWidth := 0

	for _, line := range lines {
		lineWidth := 0
		for _, r := range line {
			if r == ' ' {
				lineWidth += f.spaceWidth
				continue
			}
			if r == '\t' {
				lineWidth += f.spaceWidth * 4
				continue
			}
			glyph := f.GetGlyph(r)
			if glyph != nil {
				lineWidth += glyph.Patch.Width
			}
		}
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
	}

	totalHeight := len(lines) * f.lineHeight
	return maxWidth, totalHeight
}

// DrawText renders text onto target at (x, y) using the font's authentic normal palette colors.
func (f *HUDFont) DrawText(target *ebiten.Image, text string, x, y int) {
	lines := strings.Split(text, "\n")
	currY := y

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest

	for _, line := range lines {
		currX := x
		for _, r := range line {
			if r == ' ' {
				currX += f.spaceWidth
				continue
			}
			if r == '\t' {
				currX += f.spaceWidth * 4
				continue
			}

			glyph := f.GetGlyph(r)
			if glyph == nil || glyph.Image == nil {
				continue
			}

			op.GeoM.Reset()
			op.GeoM.Translate(float64(currX), float64(currY))
			target.DrawImage(glyph.Image, op)

			currX += glyph.Patch.Width
		}
		currY += f.lineHeight
	}
}
