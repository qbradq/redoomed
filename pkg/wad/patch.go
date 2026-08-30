package wad

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	// ErrInvalidPatch is returned when patch data cannot be parsed.
	ErrInvalidPatch = errors.New("invalid Doom picture patch format")
	// ErrInvalidPalette is returned when palette slice is less than 768 bytes.
	ErrInvalidPalette = errors.New("palette must be at least 768 bytes")
)

type rawPatchHeader struct {
	Width      uint16
	Height     uint16
	LeftOffset int16
	TopOffset  int16
}

// Patch represents a decoded Doom Picture format graphic patch.
type Patch struct {
	Width      int
	Height     int
	LeftOffset int
	TopOffset  int
	// Pixels holds palette indices (0-255) for each (x, y) coordinate.
	// Indexed by y*Width + x.
	Pixels []byte
	// Mask indicates whether a pixel is opaque (true) or transparent (false).
	Mask []bool
}

// ParsePatch parses raw bytes into a Doom Picture Patch structure.
func ParsePatch(data []byte) (*Patch, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: header too short (%d bytes)", ErrInvalidPatch, len(data))
	}

	var header rawPatchHeader
	reader := bytes.NewReader(data)
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("%w: failed to read patch header: %w", ErrInvalidPatch, err)
	}

	w := int(header.Width)
	h := int(header.Height)
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("%w: invalid dimensions %dx%d", ErrInvalidPatch, w, h)
	}

	if len(data) < 8+w*4 {
		return nil, fmt.Errorf("%w: data too short for column table", ErrInvalidPatch)
	}

	colOffsets := make([]uint32, w)
	if err := binary.Read(reader, binary.LittleEndian, &colOffsets); err != nil {
		return nil, fmt.Errorf("%w: failed to read column offsets: %w", ErrInvalidPatch, err)
	}

	totalPixels := w * h
	pixels := make([]byte, totalPixels)
	mask := make([]bool, totalPixels)

	for x := 0; x < w; x++ {
		offset := int(colOffsets[x])
		if offset < 0 || offset >= len(data) {
			continue
		}

		pos := offset
		for pos < len(data) {
			topDelta := data[pos]
			if topDelta == 0xFF {
				// 0xFF terminates the column
				break
			}

			if pos+1 >= len(data) {
				break
			}
			length := int(data[pos+1])

			// Post format:
			// pos:   topDelta (1 byte)
			// pos+1: length   (1 byte)
			// pos+2: dummy    (1 byte)
			// pos+3: pixels   (length bytes)
			// pos+3+length: dummy (1 byte)
			pixelStart := pos + 3
			pixelEnd := pixelStart + length
			if pixelEnd > len(data) {
				break
			}

			yOffset := int(topDelta)
			for i := 0; i < length; i++ {
				y := yOffset + i
				if y >= 0 && y < h {
					idx := y*w + x
					pixels[idx] = data[pixelStart+i]
					mask[idx] = true
				}
			}

			// Advance to next post (4 header/dummy bytes + length bytes)
			pos += 4 + length
		}
	}

	return &Patch{
		Width:      w,
		Height:     h,
		LeftOffset: int(header.LeftOffset),
		TopOffset:  int(header.TopOffset),
		Pixels:     pixels,
		Mask:       mask,
	}, nil
}

// PixelAt returns the palette index and opacity at (x, y).
func (p *Patch) PixelAt(x, y int) (byte, bool) {
	if x < 0 || x >= p.Width || y < 0 || y >= p.Height {
		return 0, false
	}
	idx := y*p.Width + x
	return p.Pixels[idx], p.Mask[idx]
}

// ToImage converts the patch to an *ebiten.Image using the provided 768-byte RGB palette.
func (p *Patch) ToImage(palette []byte) (*ebiten.Image, error) {
	if len(palette) < 768 {
		return nil, ErrInvalidPalette
	}
	rgba := image.NewRGBA(image.Rect(0, 0, p.Width, p.Height))
	for y := 0; y < p.Height; y++ {
		for x := 0; x < p.Width; x++ {
			palIdx, opaque := p.PixelAt(x, y)
			if !opaque {
				continue
			}
			pr := palette[int(palIdx)*3]
			pg := palette[int(palIdx)*3+1]
			pb := palette[int(palIdx)*3+2]
			rgba.SetRGBA(x, y, color.RGBA{R: pr, G: pg, B: pb, A: 255})
		}
	}
	return ebiten.NewImageFromImage(rgba), nil
}
