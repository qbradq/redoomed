package mode

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestGameModeLayerStackOrdering(t *testing.T) {
	gm := NewGameMode("MAP01", nil, nil)
	if gm == nil {
		t.Fatal("expected NewGameMode to return non-nil")
	}

	layers := gm.Layers()
	if len(layers) != 7 {
		t.Fatalf("expected 7 layers in GameMode stack, got %d", len(layers))
	}

	expectedOrder := []string{
		"Common",
		"Game Menu",
		"Game Controls",
		"HUD",
		"Mini map",
		"Level view",
		"Intermission Screen",
	}

	for i, expected := range expectedOrder {
		if layers[i].Name() != expected {
			t.Errorf("layer[%d] name = %q, want %q", i, layers[i].Name(), expected)
		}
	}

	if gm.CommonLayer() == nil || gm.CommonLayer().Name() != "Common" {
		t.Error("expected CommonLayer getter to return valid layer")
	}
	if gm.GameMenuLayer() == nil || gm.GameMenuLayer().Name() != "Game Menu" {
		t.Error("expected GameMenuLayer getter to return valid layer")
	}
	if gm.GameControlsLayer() == nil || gm.GameControlsLayer().Name() != "Game Controls" {
		t.Error("expected GameControlsLayer getter to return valid layer")
	}
	if gm.HUDLayer() == nil || gm.HUDLayer().Name() != "HUD" {
		t.Error("expected HUDLayer getter to return valid layer")
	}
	if gm.MiniMapLayer() == nil || gm.MiniMapLayer().Name() != "Mini map" {
		t.Error("expected MiniMapLayer getter to return valid layer")
	}
	if gm.LevelViewLayer() == nil || gm.LevelViewLayer().Name() != "Level view" {
		t.Error("expected LevelViewLayer getter to return valid layer")
	}
	if gm.IntermissionLayer() == nil || gm.IntermissionLayer().Name() != "Intermission Screen" {
		t.Error("expected IntermissionLayer getter to return valid layer")
	}
}

func TestGameModeOcclusion(t *testing.T) {
	gm := NewGameMode("MAP01", nil, nil)

	// In default state:
	// Mini map and Level view are invisible (false).
	// Intermission and HUD are visible (true).
	if gm.MiniMapLayer().IsVisible() {
		t.Error("expected MiniMapLayer to be invisible by default")
	}
	if gm.LevelViewLayer().IsVisible() {
		t.Error("expected LevelViewLayer to be invisible by default")
	}
	if !gm.IntermissionLayer().IsVisible() {
		t.Error("expected IntermissionLayer to be visible by default")
	}
	if !gm.HUDLayer().IsVisible() {
		t.Error("expected HUDLayer to be visible by default")
	}

	// Verify occlusion properties
	if !gm.MiniMapLayer().PreventsLowerDrawing() {
		t.Error("expected MiniMapLayer.PreventsLowerDrawing() to be true")
	}
	if !gm.LevelViewLayer().PreventsLowerDrawing() {
		t.Error("expected LevelViewLayer.PreventsLowerDrawing() to be true")
	}
	if gm.IntermissionLayer().PreventsLowerDrawing() {
		t.Error("expected IntermissionLayer.PreventsLowerDrawing() to be false")
	}
	if gm.HUDLayer().PreventsLowerDrawing() {
		t.Error("expected HUDLayer.PreventsLowerDrawing() to be false")
	}
}

type testCustomLayer struct {
	name            string
	visible         bool
	preventsLower   bool
	updateConsumed  bool
	updateCalled    bool
	drawCalled      bool
	drawnToBuffer   bool
}

func (l *testCustomLayer) Name() string                     { return l.name }
func (l *testCustomLayer) IsVisible() bool                 { return l.visible }
func (l *testCustomLayer) SetVisible(v bool)               { l.visible = v }
func (l *testCustomLayer) PreventsLowerDrawing() bool      { return l.preventsLower }
func (l *testCustomLayer) Update() (bool, error)           { l.updateCalled = true; return l.updateConsumed, nil }
func (l *testCustomLayer) Draw(screen *ebiten.Image)       { l.drawCalled = true }

func TestGameModeInputPropagation(t *testing.T) {
	gm := &GameMode{
		buffer: ebiten.NewImage(GameBufferWidth, GameBufferHeight),
	}

	topLayer := &testCustomLayer{name: "Top", visible: true, updateConsumed: true}
	bottomLayer := &testCustomLayer{name: "Bottom", visible: true, updateConsumed: false}

	gm.layers = []Layer{topLayer, bottomLayer}

	if err := gm.Update(); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	if !topLayer.updateCalled {
		t.Error("expected topLayer.Update() to be called")
	}
	if bottomLayer.updateCalled {
		t.Error("expected bottomLayer.Update() to be skipped because topLayer consumed input")
	}

	// Now test when top layer does not consume input
	topLayer.updateCalled = false
	bottomLayer.updateCalled = false
	topLayer.updateConsumed = false

	if err := gm.Update(); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	if !topLayer.updateCalled {
		t.Error("expected topLayer.Update() to be called")
	}
	if !bottomLayer.updateCalled {
		t.Error("expected bottomLayer.Update() to be called when not consumed")
	}
}

func TestGameModeDrawingWithOcclusion(t *testing.T) {
	gm := &GameMode{
		buffer: ebiten.NewImage(GameBufferWidth, GameBufferHeight),
	}

	top := &testCustomLayer{name: "Top", visible: true}
	middle := &testCustomLayer{name: "Middle", visible: true, preventsLower: true}
	bottom := &testCustomLayer{name: "Bottom", visible: true}

	gm.layers = []Layer{top, middle, bottom}

	screen := ebiten.NewImage(1280, 800)
	gm.Draw(screen)

	if !middle.drawCalled {
		t.Error("expected middle.Draw() to be called")
	}
	if !top.drawCalled {
		t.Error("expected top.Draw() to be called")
	}
	if bottom.drawCalled {
		t.Error("expected bottom.Draw() to be skipped due to middle layer occlusion")
	}
}

func TestGameModeDrawWithTextures(t *testing.T) {
	titleImg := ebiten.NewImage(320, 200)
	titleImg.Fill(color.RGBA{R: 200, G: 0, B: 0, A: 255})

	stbarImg := ebiten.NewImage(320, 32)
	stbarImg.Fill(color.RGBA{R: 0, G: 200, B: 0, A: 255})

	gm := NewGameMode("MAP01", nil, nil)
	gm.IntermissionLayer().SetTitleImage(titleImg)
	gm.HUDLayer().SetSTBARImage(stbarImg)

	screen := ebiten.NewImage(1280, 800)
	gm.Draw(screen)
}
