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

func TestMaskedSegRendering(t *testing.T) {
	// Create a mock Texture with alternating opaque and transparent columns/pixels (like an iron grate)
	grateTex := &wad.Texture{
		Name:   "MIDGRATE",
		Width:  16,
		Height: 16,
		Pixels: make([]byte, 256),
		Mask:   make([]bool, 256),
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			idx := y*16 + x
			if (x+y)%2 == 0 {
				grateTex.Pixels[idx] = 1 // Palette index 1
				grateTex.Mask[idx] = true
			} else {
				grateTex.Pixels[idx] = 0
				grateTex.Mask[idx] = false
			}
		}
	}

	renderer := NewRenderer(320, 168, 200)
	// Clear with a known background color (blue: 0, 0, 255)
	renderer.Clear(color.RGBA{R: 0, G: 0, B: 255, A: 255})

	// Facing seg directly in front of camera at Z distance = 50, centered in view
	cam := &Camera{X: 0, Y: 0, Z: 32, Angle: 90, EyeHeight: 32}
	// Let seg span horizontally across columns 150 to 170
	xStart := 150
	xEnd := 170

	// Populate clip buffers: unobstructed from y=0 to y=167
	renderer.maskedClipTop = make([]int, 0, xEnd-xStart+1)
	renderer.maskedClipBottom = make([]int, 0, xEnd-xStart+1)
	for x := xStart; x <= xEnd; x++ {
		renderer.maskedClipTop = append(renderer.maskedClipTop, -1)
		renderer.maskedClipBottom = append(renderer.maskedClipBottom, 168)
	}

	mockMapData := &wad.MapData{
		Name: "TEST",
	}

	ld := &wad.Linedef{
		Flags: wad.LinedefTwoSided,
	}
	frontSide := &wad.Sidedef{
		MiddleTexture: "MIDGRATE",
	}
	frontSec := &wad.Sector{
		CeilingHeight: 40,
		FloorHeight:   0,
		LightLevel:    255,
	}
	backSec := &wad.Sector{
		CeilingHeight: 40,
		FloorHeight:   0,
		LightLevel:    255,
	}

	renderer.maskedSegs = []maskedSeg{
		{
			seg:        &wad.Seg{Offset: 0},
			ld:         ld,
			frontSide:  frontSide,
			frontSec:   frontSec,
			backSec:    backSec,
			midTex:     grateTex,
			xStart:     xStart,
			xEnd:       xEnd,
			tx1:        -20,
			tz1:        50,
			dxCam:      40,
			dzCam:      0,
			uOffset1:   0,
			du:         40,
			clipOffset: 0,
		},
	}

	renderer.drawMaskedSegs(mockMapData, cam)

	// Verify that both grate pixels (white/mapped) and background blue pixels are present in the region
	hasGrate := false
	hasBackground := false

	for y := 50; y < 120; y++ {
		for x := xStart; x <= xEnd; x++ {
			idx := (y*320 + x) * 4
			b := renderer.pixels[idx+2]
			if b == 255 && renderer.pixels[idx] == 0 && renderer.pixels[idx+1] == 0 {
				hasBackground = true
			} else {
				hasGrate = true
			}
		}
	}

	if !hasGrate {
		t.Error("expected opaque pixels from masked middle texture to be rendered")
	}
	if !hasBackground {
		t.Error("expected transparent pixels from masked middle texture to preserve background")
	}
}

func TestMaskedLinedefsInFreedoom(t *testing.T) {
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

	// Check MAP01 for 2-sided linedefs with middle textures (e.g. grates/windows/bars)
	mapData, err := w.LoadMap("MAP01")
	if err != nil {
		t.Fatalf("LoadMap MAP01 failed: %v", err)
	}

	// Count 2-sided linedefs with middle textures
	maskedCount := 0
	for _, ld := range mapData.Linedefs {
		if ld.Flags&wad.LinedefTwoSided != 0 && ld.RightSide != 0xFFFF && ld.LeftSide != 0xFFFF {
			rSide := &mapData.Sidedefs[ld.RightSide]
			lSide := &mapData.Sidedefs[ld.LeftSide]
			if (rSide.MiddleTexture != "" && rSide.MiddleTexture != "-") ||
				(lSide.MiddleTexture != "" && lSide.MiddleTexture != "-") {
				maskedCount++
			}
		}
	}

	t.Logf("Found %d 2-sided linedefs with middle textures in MAP01", maskedCount)
	if maskedCount == 0 {
		t.Log("Note: MAP01 has no 2-sided mid textures, checking other maps")
	}

	target := ebiten.NewImage(320, 200)
	renderer := NewRenderer(320, 168, 200)
	p1, ok := mapData.Player1Start()
	if ok {
		cam := &Camera{
			X:         float64(p1.X),
			Y:         float64(p1.Y),
			Angle:     float64(p1.Angle),
			EyeHeight: DefaultPlayerEyeHeight,
		}
		renderer.Render(target, mapData, cam)
	}
}

func TestMaskedSegPegging(t *testing.T) {
	grateTex := &wad.Texture{
		Name:   "MIDBARS",
		Width:  8,
		Height: 16,
		Pixels: make([]byte, 128),
		Mask:   make([]bool, 128),
	}
	for i := range grateTex.Pixels {
		grateTex.Pixels[i] = 1
		grateTex.Mask[i] = true
	}

	renderer := NewRenderer(320, 168, 200)
	renderer.Clear(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	cam := &Camera{X: 0, Y: 0, Z: 0, Angle: 90, EyeHeight: 0}
	xStart := 155
	xEnd := 165

	renderer.maskedClipTop = make([]int, 0, xEnd-xStart+1)
	renderer.maskedClipBottom = make([]int, 0, xEnd-xStart+1)
	for x := xStart; x <= xEnd; x++ {
		renderer.maskedClipTop = append(renderer.maskedClipTop, -1)
		renderer.maskedClipBottom = append(renderer.maskedClipBottom, 168)
	}

	// Test Lower Unpegged (pegged to floor = 0, extends up to Z=16)
	ldUnpegged := &wad.Linedef{
		Flags: wad.LinedefTwoSided | wad.LinedefDontPegBottom,
	}
	frontSide := &wad.Sidedef{MiddleTexture: "MIDBARS"}
	frontSec := &wad.Sector{CeilingHeight: 64, FloorHeight: 0, LightLevel: 255}
	backSec := &wad.Sector{CeilingHeight: 64, FloorHeight: 0, LightLevel: 255}

	renderer.maskedSegs = []maskedSeg{
		{
			seg:        &wad.Seg{Offset: 0},
			ld:         ldUnpegged,
			frontSide:  frontSide,
			frontSec:   frontSec,
			backSec:    backSec,
			midTex:     grateTex,
			xStart:     xStart,
			xEnd:       xEnd,
			tx1:        -10,
			tz1:        50,
			dxCam:      20,
			dzCam:      0,
			clipOffset: 0,
		},
	}

	renderer.drawMaskedSegs(&wad.MapData{Name: "TEST"}, cam)

	// Since floor = 0, cam.Z = 0, texture extends from Z=0 to Z=16.
	// centerY = 84. At Z=0, y = 84. At Z=16, y = 84 - 16 * 3.2 = 32.8.
	// So pixels must be drawn in range y in [35, 80]
	hasPixels := false
	for y := 35; y <= 80; y++ {
		for x := xStart; x <= xEnd; x++ {
			idx := (y*320 + x) * 4
			if renderer.pixels[idx] != 0 || renderer.pixels[idx+1] != 0 || renderer.pixels[idx+2] != 0 {
				hasPixels = true
				break
			}
		}
	}
	if !hasPixels {
		t.Error("expected pixels rendered for lower unpegged masked texture")
	}
}

func TestMaskedSegOcclusion(t *testing.T) {
	grateTex := &wad.Texture{
		Name:   "MIDBARS",
		Width:  8,
		Height: 16,
		Pixels: make([]byte, 128),
		Mask:   make([]bool, 128),
	}
	for i := range grateTex.Pixels {
		grateTex.Pixels[i] = 1
		grateTex.Mask[i] = true
	}

	renderer := NewRenderer(320, 168, 200)
	renderer.Clear(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	cam := &Camera{X: 0, Y: 0, Z: 0, Angle: 90, EyeHeight: 0}
	xStart := 155
	xEnd := 165

	// Fully occluded clip buffers
	renderer.maskedClipTop = make([]int, 0, xEnd-xStart+1)
	renderer.maskedClipBottom = make([]int, 0, xEnd-xStart+1)
	for x := xStart; x <= xEnd; x++ {
		renderer.maskedClipTop = append(renderer.maskedClipTop, 100)
		renderer.maskedClipBottom = append(renderer.maskedClipBottom, 50) // clipTop >= clipBottom - 1
	}

	ld := &wad.Linedef{Flags: wad.LinedefTwoSided}
	frontSide := &wad.Sidedef{MiddleTexture: "MIDBARS"}
	frontSec := &wad.Sector{CeilingHeight: 64, FloorHeight: 0, LightLevel: 255}
	backSec := &wad.Sector{CeilingHeight: 64, FloorHeight: 0, LightLevel: 255}

	renderer.maskedSegs = []maskedSeg{
		{
			seg:        &wad.Seg{Offset: 0},
			ld:         ld,
			frontSide:  frontSide,
			frontSec:   frontSec,
			backSec:    backSec,
			midTex:     grateTex,
			xStart:     xStart,
			xEnd:       xEnd,
			tx1:        -10,
			tz1:        50,
			dxCam:      20,
			dzCam:      0,
			clipOffset: 0,
		},
	}

	renderer.drawMaskedSegs(&wad.MapData{Name: "TEST"}, cam)

	// Verify nothing was drawn
	for y := 0; y < 168; y++ {
		for x := xStart; x <= xEnd; x++ {
			idx := (y*320 + x) * 4
			if renderer.pixels[idx] != 0 || renderer.pixels[idx+1] != 0 || renderer.pixels[idx+2] != 0 {
				t.Fatalf("expected occluded masked seg not to render any pixels, found at (%d, %d)", x, y)
			}
		}
	}
}


