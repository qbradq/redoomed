package mode

import (
	"fmt"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

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

	// Verify icon button array
	if len(ed.buttons) != NumIconButtons {
		t.Fatalf("expected %d buttons, got %d", NumIconButtons, len(ed.buttons))
	}
	for i := 0; i < NumIconButtons; i++ {
		if ed.buttons[i].Index != i {
			t.Errorf("button %d index mismatch: %d", i, ed.buttons[i].Index)
		}
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

