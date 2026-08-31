package mode

import (
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/player"
	"github.com/qbradq/redoomed/pkg/render"
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

func TestLevelViewLayerFloorUpdate(t *testing.T) {
	mapData := &wad.MapData{
		Vertexes: []wad.Vertex{
			{X: -100, Y: -100},
			{X: 100, Y: -100},
			{X: 100, Y: 100},
			{X: -100, Y: 100},
		},
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128},
		},
		Sidedefs: []wad.Sidedef{
			{Sector: 0},
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 1, V2: 2, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 2, V2: 3, RightSide: 0, LeftSide: 0xFFFF},
			{V1: 3, V2: 0, RightSide: 0, LeftSide: 0xFFFF},
		},
		Segs: []wad.Seg{
			{V1: 0, V2: 1, Linedef: 0, Direction: 0},
		},
		Subsectors: []wad.Subsector{
			{NumSegs: 1, FirstSeg: 0},
		},
		Nodes: []wad.Node{},
	}

	layer := NewLevelViewLayer()
	layer.SetMapData(mapData)
	layer.SetVisible(true)
	layer.SetCamera(0, 0, 41, 0)

	if layer.Camera().Z != 41 {
		t.Fatalf("expected initial camera Z 41, got %f", layer.Camera().Z)
	}

	// Change sector floor height (e.g. lift rises from 0 to 96)
	mapData.Sectors[0].FloorHeight = 96

	// Run Update()
	_, _ = layer.Update()

	// Verify player and camera Z adapted to new floor height
	if layer.Player().FloorZ != 96 {
		t.Errorf("expected player FloorZ 96, got %f", layer.Player().FloorZ)
	}
	expectedZ := 96.0 + render.DefaultPlayerEyeHeight
	if layer.Camera().Z != expectedZ {
		t.Errorf("expected camera Z %f, got %f", expectedZ, layer.Camera().Z)
	}
}

func TestGameModeSimulatesGravityDuringMovement(t *testing.T) {
	// 2-sector map with a high ledge (x < 0 has floor 100, x > 0 has floor 0)
	mapData := &wad.MapData{
		Vertexes: []wad.Vertex{
			{X: -100, Y: -100},
			{X: 0, Y: -100},
			{X: 100, Y: -100},
			{X: 100, Y: 100},
			{X: 0, Y: 100},
			{X: -100, Y: 100},
		},
		Sectors: []wad.Sector{
			{FloorHeight: 100, CeilingHeight: 256}, // Sector 0 (x < 0)
			{FloorHeight: 0, CeilingHeight: 256},   // Sector 1 (x > 0)
		},
		Sidedefs: []wad.Sidedef{
			{Sector: 0},
			{Sector: 1},
		},
		Linedefs: []wad.Linedef{
			{V1: 1, V2: 4, RightSide: 0, LeftSide: 1, Flags: wad.LinedefTwoSided},
			{V1: 0, V2: 1, RightSide: 0, LeftSide: 0xFFFF, Flags: wad.LinedefBlocking},
			{V1: 4, V2: 5, RightSide: 0, LeftSide: 0xFFFF, Flags: wad.LinedefBlocking},
			{V1: 5, V2: 0, RightSide: 0, LeftSide: 0xFFFF, Flags: wad.LinedefBlocking},
			{V1: 1, V2: 2, RightSide: 1, LeftSide: 0xFFFF, Flags: wad.LinedefBlocking},
			{V1: 2, V2: 3, RightSide: 1, LeftSide: 0xFFFF, Flags: wad.LinedefBlocking},
			{V1: 3, V2: 4, RightSide: 1, LeftSide: 0xFFFF, Flags: wad.LinedefBlocking},
		},
		Segs: []wad.Seg{
			{V1: 1, V2: 4, Linedef: 0, Direction: 0},
			{V1: 4, V2: 1, Linedef: 0, Direction: 1},
		},
		Subsectors: []wad.Subsector{
			{NumSegs: 1, FirstSeg: 0},
			{NumSegs: 1, FirstSeg: 1},
		},
		Nodes: []wad.Node{
			{
				PartitionX: 0,
				PartitionY: -100,
				ChangeX:    0,
				ChangeY:    200,
				RightChild: 0x8000 | 1,
				LeftChild:  0x8000 | 0,
			},
		},
		Things: []wad.Thing{
			{X: -20, Y: 0, Angle: 0, Type: wad.ThingPlayer1Start},
		},
	}

	gm := NewGameMode("TEST", nil, nil)
	gm.levelViewLayer.SetMapData(mapData)
	gm.levelViewLayer.SetVisible(true)
	gm.miniMapLayer.SetVisible(false)
	gm.intermissionLayer.SetVisible(false)

	// Step player off the ledge horizontally into Sector 1 (moving East by 50 units from X=-20 to X=30)
	gm.levelViewLayer.MovePlayer(50, 0, 0)

	// Right after horizontal move, player is fully clear of the ledge in mid-air over sector 1 (Z = 141)
	if gm.levelViewLayer.Player().Z != 141 {
		t.Fatalf("expected player to be in mid-air at Z=141, got %f", gm.levelViewLayer.Player().Z)
	}

	// Now run GameMode.Update() while movement occurs or controls update
	// GameMode.Update() must execute levelViewLayer.Update() to apply gravity even if controls layer updated
	err := gm.Update()
	if err != nil {
		t.Fatalf("unexpected error in gm.Update(): %v", err)
	}

	// Z should have fallen by 1.0 (from 141 down to 140) in this single tick
	if gm.levelViewLayer.Player().Z != 140.0 {
		t.Errorf("expected player Z to fall to 140.0 under gravity during GameMode.Update(), got %f", gm.levelViewLayer.Player().Z)
	}
	if gm.levelViewLayer.Camera().Z != 140.0 {
		t.Errorf("expected camera Z to follow gravity to 140.0 during GameMode.Update(), got %f", gm.levelViewLayer.Camera().Z)
	}
}

func TestHUDLayerWithPlayerStats(t *testing.T) {
	gm := NewGameMode("", nil, nil)
	if gm.PlayerStats() == nil {
		t.Fatal("expected PlayerStats to be initialized in GameMode")
	}

	hud := gm.HUDLayer()
	if hud == nil {
		t.Fatal("expected HUDLayer to not be nil")
	}
	if hud.PlayerStats() != gm.PlayerStats() {
		t.Error("expected HUDLayer.PlayerStats to match GameMode.PlayerStats")
	}

	// Create test screen
	screen := ebiten.NewImage(GameBufferWidth, GameBufferHeight)

	// Create HUDAssets with placeholder images
	assets := &HUDAssets{
		STBAR:       ebiten.NewImage(320, 32),
		STARMS:      ebiten.NewImage(40, 32),
		TallMinus:   ebiten.NewImage(9, 16),
		TallPercent: ebiten.NewImage(9, 16),
		Faces:       make(map[string]*HUDPatch),
	}
	for i := 0; i <= 9; i++ {
		assets.TallNums[i] = ebiten.NewImage(9, 16)
		assets.SmallNums[i] = ebiten.NewImage(4, 6)
		assets.GrayNums[i] = ebiten.NewImage(4, 6)
	}
	for i := 0; i <= 8; i++ {
		assets.Keys[i] = ebiten.NewImage(8, 5)
	}
	assets.Faces["STFST00"] = &HUDPatch{Image: ebiten.NewImage(24, 29), LeftOffset: -5, TopOffset: -2}
	assets.Faces["STFOUCH0"] = &HUDPatch{Image: ebiten.NewImage(24, 29), LeftOffset: -5, TopOffset: -2}
	assets.Faces["STFEVL0"] = &HUDPatch{Image: ebiten.NewImage(24, 29), LeftOffset: -5, TopOffset: -2}

	hud.SetAssets(assets)
	if hud.Assets() != assets {
		t.Error("expected SetAssets to update assets")
	}

	// Modify player stats
	ps := gm.PlayerStats()
	ps.Health = 75
	ps.Armor = 50
	ps.ArmorType = 1
	ps.GiveWeapon(player.WeaponShotgun)
	ps.GiveKey(player.KeyBlueCard)
	ps.GiveKey(player.KeyRedSkull)

	// Draw HUD layer onto screen
	hud.Draw(screen)

	// Test Update
	_, err := hud.Update()
	if err != nil {
		t.Fatalf("unexpected error updating HUDLayer: %v", err)
	}
	if ps.TotalTics == 0 {
		t.Error("expected HUDLayer.Update to advance player stats TotalTics")
	}
}

func TestGameControlsWeaponSwitching(t *testing.T) {
	gm := NewGameMode("", nil, nil)
	ps := gm.PlayerStats()

	// Initially player has Fist (1) and Pistol (2)
	if ps.ReadyWeapon != player.WeaponPistol {
		t.Errorf("expected initial weapon Pistol, got %v", ps.ReadyWeapon)
	}

	// Give Shotgun (slot 3) and Plasma (slot 6)
	ps.GiveWeapon(player.WeaponShotgun)
	ps.GiveWeapon(player.WeaponPlasma)

	if gm.gameControlsLayer.onSelectWeaponSlot == nil {
		t.Fatal("expected onSelectWeaponSlot callback to be registered")
	}

	// Switch to slot 3
	gm.gameControlsLayer.onSelectWeaponSlot(3)
	if ps.ReadyWeapon != player.WeaponShotgun {
		t.Errorf("expected slot 3 switch to select Shotgun, got %v", ps.ReadyWeapon)
	}

	// Switch to slot 6
	gm.gameControlsLayer.onSelectWeaponSlot(6)
	if ps.ReadyWeapon != player.WeaponPlasma {
		t.Errorf("expected slot 6 switch to select Plasma, got %v", ps.ReadyWeapon)
	}

	// Switch to slot 2 (Pistol)
	gm.gameControlsLayer.onSelectWeaponSlot(2)
	if ps.ReadyWeapon != player.WeaponPistol {
		t.Errorf("expected slot 2 switch to select Pistol, got %v", ps.ReadyWeapon)
	}
}





