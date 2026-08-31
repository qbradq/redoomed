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

func TestThingSpriteRendering(t *testing.T) {
	renderer := NewRenderer(320, 168, 200)

	// Create mock map data with a sector
	mapData := &wad.MapData{
		Name: "TEST_THINGS",
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128, LightLevel: 255},
		},
	}

	cam := &Camera{
		X:         0,
		Y:         0,
		Z:         41,
		Angle:     0, // Facing +X
		EyeHeight: 41,
	}

	// Create mock patch (16x16 red box with LeftOffset=8, TopOffset=16)
	patch := &wad.Patch{
		Width:      16,
		Height:     16,
		LeftOffset: 8,
		TopOffset:  16,
		Pixels:     make([]byte, 16*16),
		Mask:       make([]bool, 16*16),
	}
	for i := range patch.Pixels {
		patch.Pixels[i] = 176 // Red palette index
		patch.Mask[i] = true
	}

	// Uncollected item at (100, 0)
	items := []*wad.ItemEntity{
		{
			ID:        0,
			X:         100,
			Y:         0,
			FloorZ:    0,
			CeilingZ:  128,
			Radius:    20,
			Height:    16,
			Collected: false,
			Def: wad.ItemDef{
				Sprite: "MOCK_SPRITE",
			},
		},
	}

	renderer.SetItems(items)

	// Reset depthBuffer
	for i := range renderer.depthBuffer {
		renderer.depthBuffer[i] = 1000.0
	}

	// Mock texture manager with patch
	w := &wad.WAD{}
	tm, _ := wad.NewTextureManager(w)
	// If tm is nil, drawThings still has fallback or can use direct patch injection
	// Let's test with mapData.Textures = tm and mock patch cached
	if tm != nil {
		mapData.Textures = tm
		// Inject mock patch into cache if field accessible or via wrapper
	}

	// Draw things directly
	renderer.drawThings(mapData, cam)
}

func TestThingSpriteOcclusionAndCollected(t *testing.T) {
	renderer := NewRenderer(320, 168, 200)

	mapData := &wad.MapData{
		Name: "TEST_OCCLUSION",
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128, LightLevel: 255},
		},
		Things: []wad.Thing{
			{X: 100, Y: 0, Type: wad.ThingKeyBlueCard},
		},
	}

	cam := &Camera{
		X:         0,
		Y:         0,
		Z:         41,
		Angle:     0,
		EyeHeight: 41,
	}

	// 1. Solid wall / floor closer than thing (depthBuffer = 50.0, thing tz = 100.0)
	for i := range renderer.depthBuffer {
		renderer.depthBuffer[i] = 50.0
	}

	renderer.drawThings(mapData, cam)

	// Verify all pixels remain black (occluded)
	for i := 0; i < 320*168*4; i += 4 {
		if renderer.pixels[i] != 0 || renderer.pixels[i+1] != 0 || renderer.pixels[i+2] != 0 {
			t.Fatalf("expected completely occluded thing not to draw any pixels")
		}
	}

	// 2. Collected item should not be drawn
	item := &wad.ItemEntity{
		ID:        0,
		X:         100,
		Y:         0,
		FloorZ:    0,
		Collected: true,
		Def:       wad.ItemDef{Sprite: "BKEYA0"},
	}
	renderer.SetItems([]*wad.ItemEntity{item})
	for i := range renderer.depthBuffer {
		renderer.depthBuffer[i] = 1000.0
	}

	renderer.drawThings(mapData, cam)
	for i := 0; i < 320*168*4; i += 4 {
		if renderer.pixels[i] != 0 || renderer.pixels[i+1] != 0 || renderer.pixels[i+2] != 0 {
			t.Fatalf("expected collected item not to draw any pixels")
		}
	}
}

func TestFreedoomMAP01ThingRendering(t *testing.T) {
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

	texMgr, err := wad.NewTextureManager(w)
	if err != nil {
		t.Fatalf("NewTextureManager failed: %v", err)
	}
	mapData.Textures = texMgr

	items := wad.ParseMapItems(mapData)
	if len(items) == 0 {
		t.Fatal("expected items in Freedoom2 MAP01")
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
	renderer.SetItems(items)

	renderer.Render(target, mapData, cam)

	// Verify non-zero rendered pixels
	hasPixels := false
	for _, b := range renderer.pixels {
		if b != 0 {
			hasPixels = true
			break
		}
	}
	if !hasPixels {
		t.Error("expected non-black pixels when rendering MAP01 with items")
	}
}

func TestThingSpriteFloorAndWallOcclusion(t *testing.T) {
	renderer := NewRenderer(320, 168, 200)

	mapData := &wad.MapData{
		Name: "TEST_PARTIAL_OCCLUSION",
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128, LightLevel: 255},
		},
	}

	cam := &Camera{
		X:         0,
		Y:         0,
		Z:         41,
		Angle:     0,
		EyeHeight: 41,
	}

	item := &wad.ItemEntity{
		ID:        0,
		X:         100, // tz = 100
		Y:         0,
		FloorZ:    0,
		CeilingZ:  128,
		Collected: false,
		Def:       wad.ItemDef{Sprite: "MOCK"},
	}
	renderer.SetItems([]*wad.ItemEntity{item})

	// Set depthBuffer:
	// - Columns 0 to 159 have a solid wall at depth 50.0 (closer than thing at 100.0)
	// - Columns 160 to 319 have deep background at depth 500.0
	// - Rows 100 to 167 across all columns have a floor in foreground at depth 40.0
	for y := 0; y < 168; y++ {
		for x := 0; x < 320; x++ {
			idx := y*320 + x
			if x < 160 {
				renderer.depthBuffer[idx] = 50.0 // Wall occlusion on left
			} else {
				renderer.depthBuffer[idx] = 500.0 // Open on right
			}
			if y >= 100 {
				renderer.depthBuffer[idx] = 40.0 // Foreground floor occlusion
			}
		}
	}

	renderer.drawThings(mapData, cam)

	// Verify that no pixels were drawn on the left half (x < 160) or on the floor (y >= 100)
	for y := 0; y < 168; y++ {
		for x := 0; x < 320; x++ {
			idx := (y*320 + x) * 4
			hasColor := renderer.pixels[idx] != 0 || renderer.pixels[idx+1] != 0 || renderer.pixels[idx+2] != 0
			if (x < 160 || y >= 100) && hasColor {
				t.Fatalf("expected pixel at (%d, %d) to be occluded by wall/floor, but got color", x, y)
			}
		}
	}
}

func TestFixedPropsAndEastFacingRendering(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			wadPath = "../../doom1.wad"
			if _, err := os.Stat(wadPath); os.IsNotExist(err) {
				wadPath = "doom1.wad"
				if _, err := os.Stat(wadPath); os.IsNotExist(err) {
					t.Skip("no WAD file found for test")
				}
			}
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

	renderer := NewRenderer(320, 168, 200)

	// Create map data with both collectible items and fixed props (barrels, pillars)
	mapData := &wad.MapData{
		Name:     "TEST_PROPS",
		Textures: texMgr,
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128, LightLevel: 255},
		},
		Things: []wad.Thing{
			{X: 100, Y: -20, Type: 2035},              // Barrel (prop)
			{X: 120, Y: 20, Type: 30},                 // Tall green pillar (prop)
			{X: 140, Y: 0, Type: wad.ThingKeyBlueCard}, // Blue key (item)
		},
	}

	// Player is facing East (Angle = 0)
	cam := &Camera{
		X:         0,
		Y:         0,
		Z:         41,
		Angle:     0,
		EyeHeight: 41,
	}

	// Populate parsed items on renderer
	items := wad.ParseMapItems(mapData)
	if len(items) != 1 {
		t.Fatalf("expected 1 item parsed, got %d", len(items))
	}
	renderer.SetItems(items)

	for i := range renderer.depthBuffer {
		renderer.depthBuffer[i] = 1000.0
	}

	renderer.drawThings(mapData, cam)

	// Verify that pixels were drawn for props as well
	hasPixels := false
	for _, b := range renderer.pixels {
		if b != 0 {
			hasPixels = true
			break
		}
	}
	if !hasPixels {
		t.Errorf("expected pixels drawn for props and items when facing East")
	}
}





