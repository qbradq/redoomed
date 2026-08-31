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
	if ed.GridSize() != 16 {
		t.Errorf("expected default grid size 16, got %d", ed.GridSize())
	}
	if ed.ZoomLevel() != 1.0 {
		t.Errorf("expected default zoom level 1.0, got %f", ed.ZoomLevel())
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

	// Increase grid size: 16 -> 32 -> 64 -> 128 -> 256 -> 256 (capped)
	expectedIncreases := []int{32, 64, 128, 256, 256, 256}
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

	// Increase zoom level: 1.0 -> 2.0 -> 4.0 -> 8.0 -> 16.0 -> 32.0 -> 32.0 (capped)
	expectedIncreases := []float64{2.0, 4.0, 8.0, 16.0, 32.0, 32.0}
	for _, expected := range expectedIncreases {
		ed.IncreaseZoom()
		if ed.ZoomLevel() != expected {
			t.Errorf("expected zoom level %f after increase, got %f", expected, ed.ZoomLevel())
		}
	}

	// Decrease zoom level: 32 -> 16 -> 8 -> 4 -> 2 -> 1 -> 0.5 -> 0.25 -> 0.25 (capped)
	expectedDecreases := []float64{16.0, 8.0, 4.0, 2.0, 1.0, 0.5, 0.25, 0.25, 0.25}
	for _, expected := range expectedDecreases {
		ed.DecreaseZoom()
		if ed.ZoomLevel() != expected {
			t.Errorf("expected zoom level %f after decrease, got %f", expected, ed.ZoomLevel())
		}
	}

	// Test SetZoomLevel clamping
	ed.SetZoomLevel(0.1)
	if ed.ZoomLevel() != 0.25 {
		t.Errorf("expected zoom level clamped to 0.25, got %f", ed.ZoomLevel())
	}

	ed.SetZoomLevel(50.0)
	if ed.ZoomLevel() != 32.0 {
		t.Errorf("expected zoom level clamped to 32.0, got %f", ed.ZoomLevel())
	}
}

func TestEditorCoordinateTransformations(t *testing.T) {
	cf, err := font.NewConsoleFont()
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	ed := NewEditorMode(cf, nil)
	ed.SetCamPosition(100, 200)
	ed.SetZoomLevel(4.0)

	// Test WorldToScreen and ScreenToWorld round-trip
	origWx, origWy := 150.0, 250.0
	sx, sy := ed.WorldToScreen(origWx, origWy)
	wx, wy := ed.ScreenToWorld(sx, sy)

	if math.Abs(wx-origWx) > 1e-6 || math.Abs(wy-origWy) > 1e-6 {
		t.Errorf("round-trip coordinate transform failed: got (%f, %f), want (%f, %f)", wx, wy, origWx, origWy)
	}

	// At zoom 4, 8 map units = 32 pixels
	sx1, _ := ed.WorldToScreen(0, 0)
	sx2, _ := ed.WorldToScreen(8, 0)
	pixelDistance := sx2 - sx1
	if pixelDistance != 32.0 {
		t.Errorf("expected 8 units at zoom 4.0 to be 32 pixels, got %f", pixelDistance)
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
	rightText1 := fmt.Sprintf("GRID: %3d  ZOOM: %5.2fx", ed.GridSize(), ed.ZoomLevel())
	ed.SetGridSize(256)
	ed.SetZoomLevel(32.0)
	rightText2 := fmt.Sprintf("GRID: %3d  ZOOM: %5.2fx", ed.GridSize(), ed.ZoomLevel())

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
	if NumIconButtons*IconButtonSize != UserPanelWidth {
		t.Errorf("expected 15 buttons of 16px to equal user panel width of %d, got %d", UserPanelWidth, NumIconButtons*IconButtonSize)
	}
}

