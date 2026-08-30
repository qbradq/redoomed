package font

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestConsoleFont(t *testing.T) {
	cf, err := NewConsoleFont()
	if err != nil {
		t.Fatalf("NewConsoleFont failed: %v", err)
	}

	if cf.LineHeight() != 8 {
		t.Errorf("expected line height 8, got %d", cf.LineHeight())
	}

	if cf.GlyphWidth() != 8 {
		t.Errorf("expected glyph width 8, got %d", cf.GlyphWidth())
	}

	w, h := cf.MeasureText("Hello, World!")
	if w <= 0 || h <= 0 {
		t.Errorf("invalid text measurement: %dx%d", w, h)
	}

	dst := ebiten.NewImage(320, 200)
	cf.DrawText(dst, "Hello, World!", 10, 10, color.White)
}
