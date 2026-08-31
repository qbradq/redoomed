package mode

import (
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/data"
	"github.com/qbradq/redoomed/pkg/font"
	"github.com/qbradq/redoomed/pkg/wad"
)

func TestEditorModeDefaults(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)
	if ed == nil {
		t.Fatal("expected NewEditorMode to return non-nil")
	}

	// Verify defaults
	if ed.GridSize() != 64 {
		t.Errorf("expected default grid size 64, got %d", ed.GridSize())
	}
	if ed.ZoomLevel() != 3 {
		t.Errorf("expected default zoom level 3 (0.125 scale), got %d", ed.ZoomLevel())
	}
	if ed.ZoomScale() != 0.125 {
		t.Errorf("expected default zoom scale 0.125, got %f", ed.ZoomScale())
	}
	if ed.CamX() != 0 || ed.CamY() != 0 {
		t.Errorf("expected camera at (0, 0), got (%f, %f)", ed.CamX(), ed.CamY())
	}
}

func TestLoadEditorIcons(t *testing.T) {
	img, err := loadEditorIcons()
	if err != nil {
		t.Fatalf("loadEditorIcons failed: %v", err)
	}
	if img == nil {
		t.Fatal("expected non-nil editor icons image")
	}
	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 128 {
		t.Errorf("expected 128x128 icons atlas, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	f, err := data.FS.Open("gfx/editor_icons.png")
	if err != nil {
		t.Fatalf("open editor_icons.png: %v", err)
	}
	defer f.Close()
	rawImg, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode editor_icons.png: %v", err)
	}
	// Check non-empty pixels in icon 0 (0..15, 0..15)
	var nonZeroCount int
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			r, g, b, a := rawImg.At(x, y).RGBA()
			if a > 0 {
				nonZeroCount++
				t.Logf("Pixel (%d, %d): R=%d G=%d B=%d A=%d", x, y, r>>8, g>>8, b>>8, a>>8)
			}
		}
	}
	t.Logf("Icon 0 non-zero alpha pixels: %d / 256", nonZeroCount)
}

func TestEditorGridSizeDoublingAndHalving(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)
	ed.SetGridSize(1)

	// Increase grid size: 1 -> 2 -> 4 -> 8 -> 16 -> 32 -> 64 -> 128 -> 256 -> 256 (capped)
	expectedIncreases := []int{2, 4, 8, 16, 32, 64, 128, 256, 256, 256}
	for _, expected := range expectedIncreases {
		ed.IncreaseGridSize()
		if ed.GridSize() != expected {
			t.Errorf("expected grid size %d after increase, got %d", expected, ed.GridSize())
		}
	}

	// Decrease grid size: 256 -> 128 -> 64 -> 32 -> 16 -> 8 -> 4 -> 2 -> 1 -> 1 (capped)
	expectedDecreases := []int{128, 64, 32, 16, 8, 4, 2, 1, 1, 1}
	for _, expected := range expectedDecreases {
		ed.DecreaseGridSize()
		if ed.GridSize() != expected {
			t.Errorf("expected grid size %d after decrease, got %d", expected, ed.GridSize())
		}
	}

	// Test SetGridSize with clamping
	ed.SetGridSize(0)
	if ed.GridSize() != 1 {
		t.Errorf("expected grid size clamped to 1, got %d", ed.GridSize())
	}

	ed.SetGridSize(1000)
	if ed.GridSize() != 256 {
		t.Errorf("expected grid size clamped to 256, got %d", ed.GridSize())
	}

	ed.SetGridSize(64)
	if ed.GridSize() != 64 {
		t.Errorf("expected grid size 64, got %d", ed.GridSize())
	}
}

func TestEditorZoomLevelDoublingAndHalving(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)

	// Initial zoom level is 3 (scale 0.125)
	if ed.ZoomLevel() != 3 {
		t.Fatalf("expected initial zoom level 3, got %d", ed.ZoomLevel())
	}

	// Increase zoom level: 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 11 (capped)
	expectedIncreases := []int{4, 5, 6, 7, 8, 9, 10, 11, 11, 11}
	for _, expected := range expectedIncreases {
		ed.IncreaseZoom()
		if ed.ZoomLevel() != expected {
			t.Errorf("expected zoom level %d after increase, got %d", expected, ed.ZoomLevel())
		}
	}
	if ed.ZoomScale() != 32.0 {
		t.Errorf("expected max zoom level 11 to have scale 32.0, got %f", ed.ZoomScale())
	}

	// Decrease zoom level: 11 -> 10 -> 9 -> 8 -> 7 -> 6 -> 5 -> 4 -> 3 -> 2 -> 1 -> 0 -> 0 (capped)
	expectedDecreases := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 0, 0}
	for _, expected := range expectedDecreases {
		ed.DecreaseZoom()
		if ed.ZoomLevel() != expected {
			t.Errorf("expected zoom level %d after decrease, got %d", expected, ed.ZoomLevel())
		}
	}
	// Level 0 is 64 map units per pixel -> scale 1/64
	if math.Abs(ed.ZoomScale()-(1.0/64.0)) > 1e-9 {
		t.Errorf("expected zoom level 0 to have scale 1/64 (0.015625), got %f", ed.ZoomScale())
	}

	// Test SetZoomLevel clamping
	ed.SetZoomLevel(-5)
	if ed.ZoomLevel() != 0 {
		t.Errorf("expected zoom level clamped to 0, got %d", ed.ZoomLevel())
	}

	ed.SetZoomLevel(50)
	if ed.ZoomLevel() != 11 {
		t.Errorf("expected zoom level clamped to 11, got %d", ed.ZoomLevel())
	}
}

func TestEditorCoordinateTransformations(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)
	ed.SetCamPosition(100, 200)
	ed.SetZoomLevel(8) // scale 4.0

	// Test WorldToScreen and ScreenToWorld round-trip
	origWx, origWy := 150.0, 250.0
	sx, sy := ed.WorldToScreen(origWx, origWy)
	wx, wy := ed.ScreenToWorld(sx, sy)

	if math.Abs(wx-origWx) > 1e-6 || math.Abs(wy-origWy) > 1e-6 {
		t.Errorf("round-trip coordinate transform failed: got (%f, %f), want (%f, %f)", wx, wy, origWx, origWy)
	}

	// At zoom level 8 (scale 4.0), 8 map units = 32 pixels
	sx1, _ := ed.WorldToScreen(0, 0)
	sx2, _ := ed.WorldToScreen(8, 0)
	pixelDistance := sx2 - sx1
	if pixelDistance != 32.0 {
		t.Errorf("expected 8 units at zoom level 8 (scale 4.0) to be 32 pixels, got %f", pixelDistance)
	}

	// At zoom level 0 (scale 1/64), 64 map units = 1 pixel
	ed.SetZoomLevel(0)
	sxA, _ := ed.WorldToScreen(0, 0)
	sxB, _ := ed.WorldToScreen(64, 0)
	distAtZero := sxB - sxA
	if distAtZero != 1.0 {
		t.Errorf("expected 64 units at zoom level 0 (scale 1/64) to be 1 pixel, got %f", distAtZero)
	}
}

func TestEditorButtonsAndCallbacks(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)

	var toggled, consoleToggled bool
	ed.SetOnToggle(func() {
		toggled = true
	})
	ed.SetOnToggleConsole(func() {
		consoleToggled = true
	})

	if ed.onToggle == nil || ed.onToggleConsole == nil {
		t.Fatal("expected callbacks to be set")
	}

	ed.onToggle()
	if !toggled {
		t.Error("expected onToggle callback to have been called")
	}

	ed.onToggleConsole()
	if !consoleToggled {
		t.Error("expected onToggleConsole callback to have been called")
	}

	// Verify icon button array length
	buttons := ed.Buttons()
	if len(buttons) != NumIconButtons {
		t.Fatalf("expected %d buttons, got %d", NumIconButtons, len(buttons))
	}
}

func TestEditorButtonsPromptSpecifications(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)

	// Button 0: New map button. Icon 0. Disabled.
	b0 := ed.Button(0)
	if b0.Icon != 0 || b0.Enabled != false || !b0.Visible {
		t.Errorf("button 0 mismatch: %+v", b0)
	}

	// Button 1: Open map button. Icon 1. Disabled.
	b1 := ed.Button(1)
	if b1.Icon != 1 || b1.Enabled != false || !b1.Visible {
		t.Errorf("button 1 mismatch: %+v", b1)
	}

	// Button 2: Save map button. Icon 2. Disabled.
	b2 := ed.Button(2)
	if b2.Icon != 2 || b2.Enabled != false || !b2.Visible {
		t.Errorf("button 2 mismatch: %+v", b2)
	}

	// Button 3: Vertex mode button. Icon 3. Enabled, on by default.
	b3 := ed.Button(3)
	if b3.Icon != 3 || b3.Enabled != true || !b3.IsToggle || !b3.Active || !b3.Visible {
		t.Errorf("button 3 mismatch: %+v", b3)
	}

	// Button 4: Line mode button. Icon 4. Enabled, off by default.
	b4 := ed.Button(4)
	if b4.Icon != 4 || b4.Enabled != true || !b4.IsToggle || b4.Active || !b4.Visible {
		t.Errorf("button 4 mismatch: %+v", b4)
	}

	// Button 5: Sector mode button. Icon 5. Enabled, off by default.
	b5 := ed.Button(5)
	if b5.Icon != 5 || b5.Enabled != true || !b5.IsToggle || b5.Active || !b5.Visible {
		t.Errorf("button 5 mismatch: %+v", b5)
	}

	// Button 6: Thing mode button. Icon 6. Enabled, off by default.
	b6 := ed.Button(6)
	if b6.Icon != 6 || b6.Enabled != true || !b6.IsToggle || b6.Active || !b6.Visible {
		t.Errorf("button 6 mismatch: %+v", b6)
	}

	// Button 7: Zoom In button. Icon 7. Enabled. Zooms in.
	b7 := ed.Button(7)
	if b7.Icon != 7 || b7.Enabled != true || !b7.Visible || b7.OnClick == nil {
		t.Errorf("button 7 mismatch: %+v", b7)
	}
	initialZoom := ed.ZoomLevel()
	b7.OnClick(ed)
	if ed.ZoomLevel() != initialZoom+1 {
		t.Errorf("expected zoom level %d after zoom in, got %d", initialZoom+1, ed.ZoomLevel())
	}

	// Button 8: Zoom Out button. Icon 8. Enabled. Zooms out.
	b8 := ed.Button(8)
	if b8.Icon != 8 || b8.Enabled != true || !b8.Visible || b8.OnClick == nil {
		t.Errorf("button 8 mismatch: %+v", b8)
	}
	b8.OnClick(ed)
	if ed.ZoomLevel() != initialZoom {
		t.Errorf("expected zoom level %d after zoom out, got %d", initialZoom, ed.ZoomLevel())
	}

	// Button 9: Increase grid size. Icon 9. Enabled.
	b9 := ed.Button(9)
	if b9.Icon != 9 || b9.Enabled != true || !b9.Visible || b9.OnClick == nil {
		t.Errorf("button 9 mismatch: %+v", b9)
	}
	initialGrid := ed.GridSize()
	b9.OnClick(ed)
	if ed.GridSize() != initialGrid*2 {
		t.Errorf("expected grid size %d after increase, got %d", initialGrid*2, ed.GridSize())
	}

	// Button 10: Decrease grid size. Icon 10. Enabled.
	b10 := ed.Button(10)
	if b10.Icon != 10 || b10.Enabled != true || !b10.Visible || b10.OnClick == nil {
		t.Errorf("button 10 mismatch: %+v", b10)
	}
	b10.OnClick(ed)
	if ed.GridSize() != initialGrid {
		t.Errorf("expected grid size %d after decrease, got %d", initialGrid, ed.GridSize())
	}

	// Buttons 11..14: Unused, not visible
	for i := 11; i < NumIconButtons; i++ {
		btn := ed.Button(i)
		if btn.Visible || btn.Icon != -1 {
			t.Errorf("expected unused button %d to be invisible with icon -1, got %+v", i, btn)
		}
	}
}

func TestEditorToggleGroupBehavior(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)

	// Default mode: Vertex
	if ed.EditMode() != EditModeVertex {
		t.Errorf("expected default mode Vertex, got %v", ed.EditMode())
	}
	if !ed.Button(3).Active || ed.Button(4).Active || ed.Button(5).Active || ed.Button(6).Active {
		t.Error("expected only Vertex mode button to be active")
	}

	// Switch to Line Mode
	ed.SetEditMode(EditModeLine)
	if ed.EditMode() != EditModeLine {
		t.Errorf("expected mode Line, got %v", ed.EditMode())
	}
	if ed.Button(3).Active || !ed.Button(4).Active || ed.Button(5).Active || ed.Button(6).Active {
		t.Error("expected only Line mode button to be active")
	}

	// Switch to Sector Mode
	ed.SetEditMode(EditModeSector)
	if ed.EditMode() != EditModeSector {
		t.Errorf("expected mode Sector, got %v", ed.EditMode())
	}
	if ed.Button(3).Active || ed.Button(4).Active || !ed.Button(5).Active || ed.Button(6).Active {
		t.Error("expected only Sector mode button to be active")
	}

	// Switch to Thing Mode
	ed.SetEditMode(EditModeThing)
	if ed.EditMode() != EditModeThing {
		t.Errorf("expected mode Thing, got %v", ed.EditMode())
	}
	if ed.Button(3).Active || ed.Button(4).Active || ed.Button(5).Active || !ed.Button(6).Active {
		t.Error("expected only Thing mode button to be active")
	}
}

func TestEditorDrawAndStatusBar(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	// Test with nil WAD
	ed := NewEditorMode(cf, nil)
	screen := ebiten.NewImage(1280, 800)

	// Draw should succeed without panic
	ed.Draw(screen)

	// Test with a mock WAD
	w := &wad.WAD{}
	w.SetFilename("freedoom2.wad")
	ed.SetWAD(w)
	if ed.WAD() != w {
		t.Error("expected WAD to match assigned instance")
	}

	ed.Draw(screen)

	// Test Update
	if err := ed.Update(); err != nil {
		t.Errorf("Update() returned error: %v", err)
	}

	// Test status bar strings
	rightText1 := fmt.Sprintf("GRID: %3d  ZOOM: %2d", ed.GridSize(), ed.ZoomLevel())
	ed.SetGridSize(256)
	ed.SetZoomLevel(11)
	rightText2 := fmt.Sprintf("GRID: %3d  ZOOM: %2d", ed.GridSize(), ed.ZoomLevel())

	if len(rightText1) != len(rightText2) {
		t.Errorf("expected right-justified text lengths to match for fixed-width formatting: %q vs %q", rightText1, rightText2)
	}
}

func TestEditorLayoutDimensions(t *testing.T) {
	if EditorBufferWidth != 640 || EditorBufferHeight != 400 {
		t.Errorf("expected 640x400 base resolution, got %dx%d", EditorBufferWidth, EditorBufferHeight)
	}
	if UserPanelWidth != 240 || UserPanelHeight != 392 {
		t.Errorf("expected 240x392 user panel, got %dx%d", UserPanelWidth, UserPanelHeight)
	}
	if EditingWindowWidth != 400 || EditingWindowHeight != 392 {
		t.Errorf("expected 400x392 editing window, got %dx%d", EditingWindowWidth, EditingWindowHeight)
	}
	if StatusBarWidth != 640 || StatusBarHeight != 8 || StatusBarY != 392 {
		t.Errorf("expected 640x8 status bar at Y=392, got %dx%d at Y=%d", StatusBarWidth, StatusBarHeight, StatusBarY)
	}
	if NumIconButtons != 15 {
		t.Errorf("expected 15 icon buttons, got %d", NumIconButtons)
	}
}

func TestEditorMapDataRendering(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)

	// Create mock map data with linedefs and vertices
	mockMap := &wad.MapData{
		Name: "TESTMAP",
		Vertexes: []wad.Vertex{
			{X: -64, Y: -64},
			{X: 64, Y: -64},
			{X: 64, Y: 64},
			{X: -64, Y: 64},
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1},
			{V1: 1, V2: 2},
			{V1: 2, V2: 3},
			{V1: 3, V2: 0},
		},
	}

	ed.SetMapData(mockMap)
	if ed.MapData() != mockMap {
		t.Fatal("expected MapData to match set mockMap")
	}

	screen := ebiten.NewImage(1280, 800)
	ed.Draw(screen)

	// Test map data provider
	ed.SetMapData(nil)
	ed.SetMapDataProvider(func() *wad.MapData {
		return mockMap
	})
	if ed.MapData() != mockMap {
		t.Fatal("expected MapData to match dynamic provider")
	}

	ed.Draw(screen)
}

func TestEditorThingsRendering(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)

	mockMap := &wad.MapData{
		Name: "TESTMAP",
		Things: []wad.Thing{
			{X: 0, Y: 0, Type: 1},      // Player 1 start (radius 16)
			{X: 100, Y: 100, Type: 3004}, // Zombieman (radius 20)
		},
	}
	ed.SetMapData(mockMap)

	screen := ebiten.NewImage(1280, 800)
	ed.Draw(screen)
}

func TestEditorHoverTrackingAndHighlighting(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)
	ed.SetZoomLevel(6) // 1.0 scale (1 map unit = 1 pixel)
	ed.SetCamPosition(0, 0)

	// Map center is at editing window center (200, 196)
	mockMap := &wad.MapData{
		Name: "TESTMAP",
		Vertexes: []wad.Vertex{
			{X: -50, Y: 0},
			{X: 50, Y: 0},
			{X: 50, Y: 50},
			{X: -50, Y: 50},
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 1, V2: 2, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 2, V2: 3, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 3, V2: 0, RightSide: 0, LeftSide: 0xFFFF},
		},
		Sidedefs: []wad.Sidedef{
			{Sector: 0},
		},
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128},
		},
		Things: []wad.Thing{
			{X: 0, Y: 0, Type: 1}, // Center at (200, 196)
		},
	}
	ed.SetMapData(mockMap)
	ed.SetCamPosition(0, 0) // Re-center at (0, 0)

	// Screen position of vertex 0 (-50, 0): (200 - 50, 196) = (150, 196)
	// Screen position of vertex 1 (50, 0): (200 + 50, 196) = (250, 196)

	// 1. Vertex Mode
	ed.SetEditMode(EditModeVertex)

	// Cursor at (152, 196) -> distance = 2 pixels <= 5px -> should highlight vertex 0
	ed.updateHoverTargets(152, 196)
	if ed.HoveredVertex() != 0 {
		t.Errorf("expected hovered vertex 0, got %d", ed.HoveredVertex())
	}

	// Cursor at (160, 196) -> distance = 10 pixels > 5px -> no hover
	ed.updateHoverTargets(160, 196)
	if ed.HoveredVertex() != -1 {
		t.Errorf("expected no hovered vertex (>5px), got %d", ed.HoveredVertex())
	}

	// 2. Line Mode
	ed.SetEditMode(EditModeLine)

	// Line 0 is from (150, 196) to (250, 196). Cursor at (200, 198) -> dist = 2px <= 5px -> should highlight line 0
	ed.updateHoverTargets(200, 198)
	if ed.HoveredLinedef() != 0 {
		t.Errorf("expected hovered linedef 0, got %d", ed.HoveredLinedef())
	}

	// Cursor at (200, 210) -> dist = 14px > 5px -> no hover
	ed.updateHoverTargets(200, 210)
	if ed.HoveredLinedef() != -1 {
		t.Errorf("expected no hovered linedef (>5px), got %d", ed.HoveredLinedef())
	}

	// 3. Sector Mode (only highlights when cursor is in a sector)
	ed.SetEditMode(EditModeSector)

	// Mock map without subsectors returns -1 (not inside a valid BSP sector)
	ed.updateHoverTargets(200, 198)
	if ed.HoveredSector() != -1 {
		t.Errorf("expected no hovered sector without subsectors, got %d", ed.HoveredSector())
	}

	// 4. Thing Mode
	ed.SetEditMode(EditModeThing)
	// Thing 0 is at (200, 196) with radius 16. Cursor at (205, 196) -> dist = 5px <= radius+5px -> hovered
	ed.updateHoverTargets(205, 196)
	if ed.HoveredThing() != 0 {
		t.Errorf("expected hovered thing 0, got %d", ed.HoveredThing())
	}

	// Cursor outside editing window (e.g. at user panel mx=450)
	ed.updateHoverTargets(450, 100)
	if ed.HoveredThing() != -1 || ed.HoveredVertex() != -1 || ed.HoveredLinedef() != -1 || ed.HoveredSector() != -1 {
		t.Error("expected all hover targets to reset outside editing window")
	}

	// Drawing with hover targets set
	screen := ebiten.NewImage(1280, 800)
	ed.Draw(screen)
}

func TestEditorSectorModeHighlightFreedoom2(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	w, err := wad.Open("../../freedoom2.wad")
	if err != nil {
		t.Skipf("skipping freedoom2 test: %v", err)
	}
	defer w.Close()

	md, err := w.LoadMap("MAP01")
	if err != nil {
		t.Fatalf("LoadMap failed: %v", err)
	}

	ed := NewEditorMode(cf, w)
	ed.SetMapData(md)
	ed.SetEditMode(EditModeSector)

	p1, ok := md.Player1Start()
	if !ok {
		t.Fatal("expected player 1 start in MAP01")
	}

	// Center on player start and test cursor at center of screen (200, 196)
	ed.SetCamPosition(float64(p1.X), float64(p1.Y))
	ed.updateHoverTargets(200, 196)

	if ed.HoveredSector() < 0 {
		t.Errorf("expected player starting sector to be highlighted, got %d", ed.HoveredSector())
	}

	// Move cursor to outer void (e.g. at far coordinates outside the map)
	ed.SetCamPosition(-50000, -50000)
	ed.updateHoverTargets(200, 196)
	if ed.HoveredSector() != -1 {
		t.Errorf("expected no sector highlighted in void, got %d", ed.HoveredSector())
	}
}

func TestEditorSelectionTracking(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)

	// 1. Vertex Selection
	ed.SetEditMode(EditModeVertex)
	if ed.SelectionCount() != 0 {
		t.Errorf("expected 0 selections initially, got %d", ed.SelectionCount())
	}

	// Single select vertex 2
	ed.SelectVertex(2, false)
	if !ed.IsVertexSelected(2) || ed.IsVertexSelected(1) || ed.SelectionCount() != 1 {
		t.Errorf("expected vertex 2 selected, got %+v", ed.SelectedVertexes())
	}

	// Single select vertex 5 (replaces vertex 2)
	ed.SelectVertex(5, false)
	if !ed.IsVertexSelected(5) || ed.IsVertexSelected(2) || ed.SelectionCount() != 1 {
		t.Errorf("expected vertex 5 selected, got %+v", ed.SelectedVertexes())
	}

	// Multi-select: toggle vertex 2 (adds)
	ed.SelectVertex(2, true)
	if !ed.IsVertexSelected(5) || !ed.IsVertexSelected(2) || ed.SelectionCount() != 2 {
		t.Errorf("expected vertex 5 and 2 selected, got %+v", ed.SelectedVertexes())
	}

	// Multi-select: toggle vertex 5 (removes)
	ed.SelectVertex(5, true)
	if ed.IsVertexSelected(5) || !ed.IsVertexSelected(2) || ed.SelectionCount() != 1 {
		t.Errorf("expected only vertex 2 selected after toggle off, got %+v", ed.SelectedVertexes())
	}

	// Clear selection
	ed.ClearSelection()
	if ed.SelectionCount() != 0 {
		t.Errorf("expected 0 selections after ClearSelection, got %d", ed.SelectionCount())
	}

	// 2. Line Selection
	ed.SetEditMode(EditModeLine)
	ed.SelectLine(10, false)
	ed.SelectLine(20, true)
	if !ed.IsLineSelected(10) || !ed.IsLineSelected(20) || ed.SelectionCount() != 2 {
		t.Errorf("expected lines 10 and 20 selected, got %+v", ed.SelectedLines())
	}
	ed.ClearSelection()
	if ed.SelectionCount() != 0 {
		t.Errorf("expected 0 selections after line ClearSelection, got %d", ed.SelectionCount())
	}

	// 3. Sector Selection
	ed.SetEditMode(EditModeSector)
	ed.SelectSector(3, false)
	ed.SelectSector(7, true)
	if !ed.IsSectorSelected(3) || !ed.IsSectorSelected(7) || ed.SelectionCount() != 2 {
		t.Errorf("expected sectors 3 and 7 selected, got %+v", ed.SelectedSectors())
	}
	ed.ClearSelection()
	if ed.SelectionCount() != 0 {
		t.Errorf("expected 0 selections after sector ClearSelection, got %d", ed.SelectionCount())
	}

	// 4. Thing Selection
	ed.SetEditMode(EditModeThing)
	ed.SelectThing(1, false)
	ed.SelectThing(4, true)
	if !ed.IsThingSelected(1) || !ed.IsThingSelected(4) || ed.SelectionCount() != 2 {
		t.Errorf("expected things 1 and 4 selected, got %+v", ed.SelectedThings())
	}
	ed.ClearSelection()
	if ed.SelectionCount() != 0 {
		t.Errorf("expected 0 selections after thing ClearSelection, got %d", ed.SelectionCount())
	}
}

func TestEditorInspectorPropertiesSingleAndMulti(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)
	mockMap := &wad.MapData{
		Name: "TESTMAP",
		Vertexes: []wad.Vertex{
			{X: 100, Y: 200},
			{X: 100, Y: 300},
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, Flags: 1, Special: 0, Tag: 5, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 1, V2: 0, Flags: 1, Special: 11, Tag: 5, RightSide: 0, LeftSide: 1},
		},
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128, FloorPic: "FLOOR0_1", CeilingPic: "CEIL1_1", LightLevel: 160, Special: 0, Tag: 5},
			{FloorHeight: 64, CeilingHeight: 128, FloorPic: "FLOOR0_1", CeilingPic: "CEIL1_2", LightLevel: 160, Special: 0, Tag: 5},
		},
		Things: []wad.Thing{
			{X: 100, Y: 200, Angle: 90, Type: 1, Flags: 7},
			{X: 150, Y: 200, Angle: 90, Type: 1, Flags: 7},
		},
	}
	ed.SetMapData(mockMap)

	// 1. Vertex inspector
	ed.SetEditMode(EditModeVertex)

	// Single vertex 0
	propsV0 := ed.InspectorProperties([]int{0})
	expectedV0 := map[string]string{
		"Index": "0",
		"X":     "100",
		"Y":     "200",
	}
	for _, p := range propsV0 {
		if expectedV0[p.Name] != p.Value {
			t.Errorf("vertex 0 prop %s mismatch: got %s, want %s", p.Name, p.Value, expectedV0[p.Name])
		}
	}

	// Multi vertex [0, 1] -> X matches (100), Y differs ([many]), Index differs ([many])
	propsVMulti := ed.InspectorProperties([]int{0, 1})
	expectedVMulti := map[string]string{
		"Index": "[many]",
		"X":     "100",
		"Y":     "[many]",
	}
	for _, p := range propsVMulti {
		if expectedVMulti[p.Name] != p.Value {
			t.Errorf("vertex multi prop %s mismatch: got %s, want %s", p.Name, p.Value, expectedVMulti[p.Name])
		}
	}

	// 2. Line inspector
	ed.SetEditMode(EditModeLine)
	propsLMulti := ed.InspectorProperties([]int{0, 1})
	expectedLMulti := map[string]string{
		"Index":        "[many]",
		"V1":           "[many]",
		"V2":           "[many]",
		"Flags":        "1",
		"Special":      "[many]",
		"Tag":          "5",
		"Right Sidedef": "0",
		"Left Sidedef":  "[many]",
	}
	for _, p := range propsLMulti {
		if expectedLMulti[p.Name] != p.Value {
			t.Errorf("line multi prop %s mismatch: got %s, want %s", p.Name, p.Value, expectedLMulti[p.Name])
		}
	}

	// 3. Sector inspector
	ed.SetEditMode(EditModeSector)
	propsSMulti := ed.InspectorProperties([]int{0, 1})
	expectedSMulti := map[string]string{
		"Index":          "[many]",
		"Floor Height":   "[many]",
		"Ceiling Height": "128",
		"Floor Pic":      "FLOOR0_1",
		"Ceiling Pic":    "[many]",
		"Light Level":    "160",
		"Special":        "0",
		"Tag":            "5",
	}
	for _, p := range propsSMulti {
		if expectedSMulti[p.Name] != p.Value {
			t.Errorf("sector multi prop %s mismatch: got %s, want %s", p.Name, p.Value, expectedSMulti[p.Name])
		}
	}

	// 4. Thing inspector
	ed.SetEditMode(EditModeThing)
	propsTMulti := ed.InspectorProperties([]int{0, 1})
	expectedTMulti := map[string]string{
		"Index": "[many]",
		"Type":  "1",
		"Name":  "Player 1 Start",
		"X":     "[many]",
		"Y":     "200",
		"Angle": "90",
		"Flags": "7",
	}
	for _, p := range propsTMulti {
		if expectedTMulti[p.Name] != p.Value {
			t.Errorf("thing multi prop %s mismatch: got %s, want %s", p.Name, p.Value, expectedTMulti[p.Name])
		}
	}
}

func TestEditorRenderingSelectedAndHovered(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)
	mockMap := &wad.MapData{
		Name: "TESTMAP",
		Vertexes: []wad.Vertex{
			{X: -64, Y: -64},
			{X: 64, Y: -64},
			{X: 64, Y: 64},
			{X: -64, Y: 64},
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 1, V2: 2, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 2, V2: 3, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 3, V2: 0, RightSide: 0, LeftSide: 0xFFFF},
		},
		Sidedefs: []wad.Sidedef{
			{Sector: 0},
		},
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128},
		},
		Things: []wad.Thing{
			{X: 0, Y: 0, Type: 1},
		},
	}
	ed.SetMapData(mockMap)
	ed.SetCamPosition(0, 0)
	ed.SetZoomLevel(6)

	screen := ebiten.NewImage(1280, 800)

	// 1. Vertex Mode: Select vertex 0, Hover vertex 1
	ed.SetEditMode(EditModeVertex)
	ed.SelectVertex(0, false)
	ed.hoveredVertex = 1
	ed.Draw(screen)

	// Vertex 0 hovered AND selected -> should remain selected (yellow)
	ed.hoveredVertex = 0
	ed.Draw(screen)

	// 2. Line Mode: Select line 0, Hover line 1
	ed.SetEditMode(EditModeLine)
	ed.SelectLine(0, false)
	ed.hoveredLinedef = 1
	ed.Draw(screen)

	// Line 0 hovered AND selected
	ed.hoveredLinedef = 0
	ed.Draw(screen)

	// 3. Sector Mode: Select sector 0
	ed.SetEditMode(EditModeSector)
	ed.SelectSector(0, false)
	ed.hoveredSector = -1
	ed.Draw(screen)

	// 4. Thing Mode: Select thing 0
	ed.SetEditMode(EditModeThing)
	ed.SelectThing(0, false)
	ed.hoveredThing = 0
	ed.Draw(screen)
}

