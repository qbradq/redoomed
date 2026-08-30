package font

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/qbradq/redoomed/pkg/data"
)

const (
	// FontSize is 8px for the unscii-8 font.
	FontSize = 8
)

// ConsoleFont wraps text.Face for rendering 8x8 fixed-width text.
type ConsoleFont struct {
	face *text.GoTextFace
}

// NewConsoleFont loads unscii-8.ttf from embedded data.FS.
func NewConsoleFont() (*ConsoleFont, error) {
	dataBytes, err := data.FS.ReadFile("fonts/unscii-8.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read unscii-8.ttf from embedded FS: %w", err)
	}

	source, err := text.NewGoTextFaceSource(bytes.NewReader(dataBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse unscii-8.ttf: %w", err)
	}

	face := &text.GoTextFace{
		Source: source,
		Size:   FontSize,
	}

	return &ConsoleFont{
		face: face,
	}, nil
}

// LineHeight returns the line advance (8px).
func (f *ConsoleFont) LineHeight() int {
	return FontSize
}

// GlyphWidth returns the fixed width per character (8px).
func (f *ConsoleFont) GlyphWidth() int {
	return FontSize
}

// MeasureText calculates the width and height of the given text in pixels.
func (f *ConsoleFont) MeasureText(str string) (int, int) {
	if f == nil || f.face == nil {
		return len(str) * 8, 8
	}
	w, h := text.Measure(str, f.face, 0)
	return int(w), int(h)
}

// DrawText draws text at (x, y) on the target image in the given color (or white by default).
func (f *ConsoleFont) DrawText(target *ebiten.Image, str string, x, y int, clr color.Color) {
	if f == nil || f.face == nil || str == "" {
		return
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	if clr != nil {
		op.ColorScale.ScaleWithColor(clr)
	}
	text.Draw(target, str, f.face, op)
}
