package mode

import (
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/wad"
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

	// In default state without loaded map:
	// Mini map and Level view are invisible (false).
	// Intermission and HUD are visible (true).
	if gm.MiniMapLayer().IsVisible() {
		t.Error("expected MiniMapLayer to be invisible by default")
	}
	if gm.LevelViewLayer().IsVisible() {
		t.Error("expected LevelViewLayer to be invisible by default without WAD")
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
	name           string
	visible        bool
	preventsLower  bool
	updateConsumed bool
	updateCalled   bool
	drawCalled     bool
}

func (l *testCustomLayer) Name() string               { return l.name }
func (l *testCustomLayer) IsVisible() bool           { return l.visible }
func (l *testCustomLayer) SetVisible(v bool)         { l.visible = v }
func (l *testCustomLayer) PreventsLowerDrawing() bool { return l.preventsLower }
func (l *testCustomLayer) Update() (bool, error)     { l.updateCalled = true; return l.updateConsumed, nil }
func (l *testCustomLayer) Draw(screen *ebiten.Image) { l.drawCalled = true }

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

func TestLevelViewLayerAndMovement(t *testing.T) {
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

	gm := NewGameMode("MAP01", w, nil)

	// Verify LevelViewLayer was set visible by loading map
	if !gm.LevelViewLayer().IsVisible() {
		t.Error("expected LevelViewLayer to be visible after map load")
	}
	if gm.MiniMapLayer().IsVisible() {
		t.Error("expected MiniMapLayer to be hidden by default after map load")
	}
	if gm.IntermissionLayer().IsVisible() {
		t.Error("expected IntermissionLayer to be hidden after map load")
	}

	cam := gm.LevelViewLayer().Camera()
	if cam == nil {
		t.Fatal("expected non-nil Camera in LevelViewLayer")
	}
	origX, origY, origAngle := cam.X, cam.Y, cam.Angle

	// Move player
	gm.LevelViewLayer().MovePlayer(10.0, 0.0, 15.0)
	if cam.Angle != origAngle+15.0 {
		t.Errorf("expected angle %f, got %f", origAngle+15.0, cam.Angle)
	}
	if cam.X == origX && cam.Y == origY {
		t.Error("expected player position to change after MovePlayer")
	}

	// Render frame
	screen := ebiten.NewImage(1280, 800)
	gm.Draw(screen)
}

func TestMiniMapDrawingAndFlags(t *testing.T) {
	mapData := &wad.MapData{
		Name: "TESTMAP",
		Vertexes: []wad.Vertex{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
			{X: 100, Y: 100},
			{X: 0, Y: 100},
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, Flags: wad.LinedefBlocking}, // 1-sided wall (Red)
			{V1: 1, V2: 2, Flags: wad.LinedefTwoSided}, // 2-sided line (Brown)
			{V1: 2, V2: 3, Flags: wad.LinedefSecret},   // Secret line (Red)
			{V1: 3, V2: 0, Flags: wad.LinedefDontDraw}, // Don't draw (Skipped)
		},
		Things: []wad.Thing{
			{X: 50, Y: 50, Angle: 90, Type: wad.ThingPlayer1Start},
		},
	}

	miniMap := NewMiniMapLayer(mapData)
	if !miniMap.HasPlayer() {
		t.Fatal("expected player to be detected from ThingPlayer1Start")
	}

	miniMap.SetVisible(true)
	screen := ebiten.NewImage(320, 200)
	miniMap.Draw(screen)

	// Test dynamic player position update
	miniMap.SetPlayer(75, 25, 180)
	x, y, angle := miniMap.PlayerPosition()
	if x != 75 || y != 25 || angle != 180 {
		t.Errorf("player position mismatch: (%f, %f, %f)", x, y, angle)
	}

	miniMap.Draw(screen)
}

func TestGameControlsTabToggle(t *testing.T) {
	toggled := false
	controls := NewGameControlsLayer(func() {
		toggled = true
	})

	if controls.Name() != "Game Controls" {
		t.Errorf("expected name 'Game Controls', got %q", controls.Name())
	}
	if !controls.IsVisible() {
		t.Error("expected controls to be visible")
	}
	if controls.onToggleMiniMap == nil {
		t.Error("expected onToggleMiniMap to be non-nil")
	}

	// Trigger callback directly
	controls.onToggleMiniMap()
	if !toggled {
		t.Error("expected toggled to be true after callback invocation")
	}
}

func TestMiniMapZoomControls(t *testing.T) {
	miniMap := NewMiniMapLayer(nil)

	if miniMap.Zoom() != 1.0 {
		t.Errorf("expected initial zoom 1.0, got %f", miniMap.Zoom())
	}

	// Test ZoomIn
	miniMap.ZoomIn(2.0)
	if miniMap.Zoom() != 2.0 {
		t.Errorf("expected zoom 2.0, got %f", miniMap.Zoom())
	}

	// Test ZoomOut
	miniMap.ZoomOut(2.0)
	if miniMap.Zoom() != 1.0 {
		t.Errorf("expected zoom 1.0, got %f", miniMap.Zoom())
	}

	// Test SetZoom bounds clamping
	miniMap.SetZoom(100.0)
	if miniMap.Zoom() != maxMiniMapZoom {
		t.Errorf("expected zoom clamped to max %f, got %f", maxMiniMapZoom, miniMap.Zoom())
	}

	miniMap.SetZoom(0.01)
	if miniMap.Zoom() != minMiniMapZoom {
		t.Errorf("expected zoom clamped to min %f, got %f", minMiniMapZoom, miniMap.Zoom())
	}

	// Test Update when not visible (no-op)
	miniMap.SetVisible(false)
	consumed, err := miniMap.Update()
	if err != nil || consumed {
		t.Errorf("expected invisible layer Update to return false, nil; got %v, %v", consumed, err)
	}
}

func TestGameControlsOnUse(t *testing.T) {
	controls := NewGameControlsLayer(nil)
	used := false
	controls.SetOnUse(func() {
		used = true
	})

	if controls.onUse == nil {
		t.Fatal("expected onUse to be set")
	}

	controls.onUse()
	if !used {
		t.Error("expected used to be true after onUse invocation")
	}
}

func TestGameModeUse(t *testing.T) {
	gm := NewGameMode("", nil, nil)
	var loggedMsg string
	gm.SetOnLog(func(msg string) {
		loggedMsg = msg
	})

	// Without map data, Use returns false and no line
	idx, ld, hit := gm.Use()
	if hit || idx != -1 || ld != nil {
		t.Errorf("expected no hit without map, got %v, %d, %v", hit, idx, ld)
	}

	// Setup mock map data and player
	mapData := &wad.MapData{
		Vertexes: []wad.Vertex{
			{X: 0, Y: 30},
			{X: 100, Y: 30},
		},
		Linedefs: []wad.Linedef{
			{
				V1:      0,
				V2:      1,
				Flags:   wad.LinedefBlocking,
				Special: 11, // S1 Exit
				Tag:     42,
			},
		},
	}
	gm.LevelViewLayer().SetMapData(mapData)
	gm.LevelViewLayer().SetCamera(50, 0, 41, 90) // Facing North towards line at Y=30

	idx, ld, hit = gm.Use()
	if !hit {
		t.Fatalf("expected gm.Use() to hit linedef")
	}
	if idx != 0 {
		t.Errorf("expected line idx 0, got %d", idx)
	}
	if ld == nil || ld.Special != 11 || ld.Tag != 42 {
		t.Errorf("expected Special=11, Tag=42, got %+v", ld)
	}
	if loggedMsg == "" {
		t.Errorf("expected loggedMsg to be populated via SetOnLog callback")
	}
}

func TestGameModeTriggerLineSpecial(t *testing.T) {
	gm := NewGameMode("", nil, nil)
	var triggeredSpecial, triggeredLine, triggeredTag int
	gm.SetOnTriggerLineSpecial(func(special, lineID, secID, thingID, tag int) {
		triggeredSpecial = special
		triggeredLine = lineID
		triggeredTag = tag
	})

	mapData := &wad.MapData{
		Vertexes: []wad.Vertex{
			{X: 0, Y: 20},
			{X: 100, Y: 20},
		},
		Linedefs: []wad.Linedef{
			{
				V1:      0,
				V2:      1,
				Flags:   wad.LinedefBlocking,
				Special: 1, // DR Door
				Tag:     5,
			},
		},
	}
	gm.LevelViewLayer().SetMapData(mapData)
	gm.LevelViewLayer().SetCamera(50, 0, 41, 90)

	_, _, hit := gm.Use()
	if !hit {
		t.Fatalf("expected Use to hit line")
	}

	if triggeredSpecial != 1 || triggeredLine != 0 || triggeredTag != 5 {
		t.Errorf("expected special=1, line=0, tag=5, got special=%d, line=%d, tag=%d",
			triggeredSpecial, triggeredLine, triggeredTag)
	}
}


