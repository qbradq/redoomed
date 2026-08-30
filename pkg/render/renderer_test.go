package render

import (
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/wad"
)

func TestRendererCreation(t *testing.T) {
	r := NewRenderer(320, 168, 200)
	if r == nil {
		t.Fatal("expected non-nil Renderer")
	}
	if r.viewWidth != 320 || r.viewHeight != 168 || r.bufHeight != 200 {
		t.Errorf("unexpected dimensions: %dx%d (%d)", r.viewWidth, r.viewHeight, r.bufHeight)
	}
	if len(r.pixels) != 320*200*4 {
		t.Errorf("expected pixel buffer length %d, got %d", 320*200*4, len(r.pixels))
	}
}

func TestRendererRenderMap(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			t.Skip("freedoom2.wad not found in test path, skipping")
		}
	}

	w, err := wad.Open(wadPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer w.Close()

	mapData, err := w.LoadMap("MAP01")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	p1, ok := mapData.Player1Start()
	if !ok {
		t.Fatal("player 1 start not found in MAP01")
	}

	cam := &Camera{
		X:         float64(p1.X),
		Y:         float64(p1.Y),
		Angle:     float64(p1.Angle),
		EyeHeight: DefaultPlayerEyeHeight,
	}

	target := ebiten.NewImage(320, 200)
	renderer := NewRenderer(320, 168, 200)

	renderer.Render(target, mapData, cam)

	// Verify that non-zero pixels were rendered into target image
	hasPixels := false
	for _, b := range renderer.pixels {
		if b != 0 {
			hasPixels = true
			break
		}
	}
	if !hasPixels {
		t.Error("expected non-black rendered pixels in frame buffer")
	}
}

func TestSkyRendering(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			t.Skip("freedoom2.wad not found in test path, skipping")
		}
	}

	w, err := wad.Open(wadPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer w.Close()

	texMgr, err := wad.NewTextureManager(w)
	if err != nil {
		t.Fatalf("NewTextureManager failed: %v", err)
	}

	skyTex := texMgr.GetSkyTexture("MAP01")
	if skyTex == nil {
		t.Fatal("expected sky texture for MAP01")
	}
	if skyTex.Width <= 0 || skyTex.Height <= 0 {
		t.Errorf("invalid sky texture dimensions: %dx%d", skyTex.Width, skyTex.Height)
	}

	renderer := NewRenderer(320, 168, 200)
	cam := &Camera{X: 0, Y: 0, Z: 41, Angle: 90}
	palette := texMgr.PaletteRGBA()

	// Test drawSkySpan directly
	renderer.Clear(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	for x := 0; x < 320; x++ {
		renderer.drawSkySpan(cam, x, 0, 84, skyTex, texMgr, palette)
	}

	// Verify non-zero pixels rendered in sky area
	nonZero := 0
	for y := 0; y < 85; y++ {
		for x := 0; x < 320; x++ {
			idx := (y*320 + x) * 4
			if renderer.pixels[idx] != 0 || renderer.pixels[idx+1] != 0 || renderer.pixels[idx+2] != 0 {
				nonZero++
			}
		}
	}
	if nonZero == 0 {
		t.Error("expected non-zero pixels rendered for sky span")
	}
}
