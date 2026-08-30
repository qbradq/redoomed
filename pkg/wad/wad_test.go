package wad

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestWADLoading(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			t.Skip("freedoom2.wad not found in test path, skipping")
		}
	}

	w, err := Open(wadPath)
	if err != nil {
		t.Fatalf("Open(%s) failed: %v", wadPath, err)
	}
	defer w.Close()

	if w.Type() != "IWAD" {
		t.Errorf("expected IWAD type, got %s", w.Type())
	}

	if w.NumLumps() == 0 {
		t.Error("expected non-zero lumps in WAD")
	}

	if !w.HasLump("PLAYPAL") {
		t.Error("expected PLAYPAL lump to exist")
	}

	playpal, err := w.GetLump("PLAYPAL")
	if err != nil {
		t.Fatalf("GetLump(PLAYPAL) failed: %v", err)
	}
	if len(playpal) < 768 {
		t.Errorf("expected PLAYPAL size >= 768, got %d", len(playpal))
	}
}

func TestHUDFont(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			t.Skip("freedoom2.wad not found in test path, skipping")
		}
	}

	w, err := Open(wadPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer w.Close()

	font, err := NewHUDFont(w)
	if err != nil {
		t.Fatalf("NewHUDFont failed: %v", err)
	}

	// Verify 'A' and lowercase 'a' (via uppercase fallback)
	glyphA := font.GetGlyph('A')
	if glyphA == nil {
		t.Fatal("expected glyph for 'A'")
	}
	if glyphA.Patch.Width <= 0 || glyphA.Patch.Height <= 0 {
		t.Errorf("invalid glyph dimensions for 'A': %dx%d", glyphA.Patch.Width, glyphA.Patch.Height)
	}
	if glyphA.Image == nil {
		t.Fatal("expected glyph Image to be non-nil")
	}

	glypha := font.GetGlyph('a')
	if glypha == nil {
		t.Fatal("expected fallback glyph for 'a'")
	}

	// Verify 'y' maps to uppercase 'Y'
	glyphY := font.GetGlyph('Y')
	glyphy := font.GetGlyph('y')
	if glyphy == nil || glyphY == nil {
		t.Fatal("expected glyphs for 'y' and 'Y'")
	}
	if glyphy != glyphY {
		t.Errorf("expected 'y' to resolve to 'Y' glyph, got different glyph pointer")
	}

	// Test text measurement
	wPx, hPx := font.MeasureText("Hello, World!")
	if wPx <= 0 || hPx <= 0 {
		t.Errorf("invalid text measurement: %dx%d", wPx, hPx)
	}

	// Test DrawText
	dst := ebiten.NewImage(320, 200)
	font.DrawText(dst, "Hello, World!", 0, 0)
}
