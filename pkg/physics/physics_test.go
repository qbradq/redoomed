package physics

import (
	"testing"

	"github.com/qbradq/redoomed/pkg/wad"
)

// helper to build a test map with 2 sectors and a linedef between them
func createTwoSectorMap(floor1, ceil1, floor2, ceil2 int16, lineFlags uint16, twoSided bool) *wad.MapData {
	vertexes := []wad.Vertex{
		{X: -100, Y: -100}, // 0: bottom-left
		{X: 0, Y: -100},    // 1: bottom-mid
		{X: 100, Y: -100},  // 2: bottom-right
		{X: 100, Y: 100},   // 3: top-right
		{X: 0, Y: 100},     // 4: top-mid
		{X: -100, Y: 100},  // 5: top-left
	}

	sectors := []wad.Sector{
		{FloorHeight: floor1, CeilingHeight: ceil1}, // Sector 0 (Left: x < 0)
		{FloorHeight: floor2, CeilingHeight: ceil2}, // Sector 1 (Right: x > 0)
	}

	sidedefs := []wad.Sidedef{
		{Sector: 0}, // Sidedef 0 -> Sector 0
		{Sector: 1}, // Sidedef 1 -> Sector 1
	}

	leftSide := uint16(0xFFFF)
	if twoSided {
		leftSide = 1
	}

	linedefs := []wad.Linedef{
		// Dividing line between Sector 0 and 1 (from (0,-100) to (0,100))
		{V1: 1, V2: 4, Flags: lineFlags, RightSide: 0, LeftSide: leftSide},
		// Sector 0 outer boundary
		{V1: 0, V2: 1, Flags: wad.LinedefBlocking, RightSide: 0, LeftSide: 0xFFFF},
		{V1: 4, V2: 5, Flags: wad.LinedefBlocking, RightSide: 0, LeftSide: 0xFFFF},
		{V1: 5, V2: 0, Flags: wad.LinedefBlocking, RightSide: 0, LeftSide: 0xFFFF},
		// Sector 1 outer boundary
		{V1: 1, V2: 2, Flags: wad.LinedefBlocking, RightSide: 1, LeftSide: 0xFFFF},
		{V1: 2, V2: 3, Flags: wad.LinedefBlocking, RightSide: 1, LeftSide: 0xFFFF},
		{V1: 3, V2: 4, Flags: wad.LinedefBlocking, RightSide: 1, LeftSide: 0xFFFF},
	}

	segs := []wad.Seg{
		{V1: 1, V2: 4, Linedef: 0, Direction: 0}, // Seg 0 in Subsector 0 (Sector 0)
		{V1: 4, V2: 1, Linedef: 0, Direction: 1}, // Seg 1 in Subsector 1 (Sector 1)
	}

	subsectors := []wad.Subsector{
		{NumSegs: 1, FirstSeg: 0}, // Subsector 0
		{NumSegs: 1, FirstSeg: 1}, // Subsector 1
	}

	nodes := []wad.Node{
		{
			PartitionX: 0,
			PartitionY: -100,
			ChangeX:    0,
			ChangeY:    200,
			RightChild: 0x8000 | 1, // Subsector 1 (x > 0)
			LeftChild:  0x8000 | 0, // Subsector 0 (x < 0)
		},
	}

	return &wad.MapData{
		Name:       "TEST",
		Vertexes:   vertexes,
		Linedefs:   linedefs,
		Sidedefs:   sidedefs,
		Sectors:    sectors,
		Segs:       segs,
		Subsectors: subsectors,
		Nodes:      nodes,
	}
}

func TestSolidWallCollision(t *testing.T) {
	// 1-sided wall at x=0
	mapData := createTwoSectorMap(0, 128, 0, 128, 0, false)

	player := NewPlayerActor(-50, 0, 41, 0) // Facing East (towards wall at x=0)

	// Try moving into the solid wall at x = -5
	// Since player radius = 16, x=-5 is within 16 units of x=0 wall -> should be blocked
	valid, _, _ := CheckPosition(mapData, player, -5, 0)
	if valid {
		t.Error("expected CheckPosition to block movement into solid 1-sided wall")
	}

	// Try moving to safe position x=-40 (dist=40 > 16)
	valid, _, _ = CheckPosition(mapData, player, -40, 0)
	if !valid {
		t.Error("expected CheckPosition to allow movement at x=-40")
	}

	// Test 2-sided wall with LinedefBlocking flag
	mapDataBlocking := createTwoSectorMap(0, 128, 0, 128, wad.LinedefBlocking, true)
	valid, _, _ = CheckPosition(mapDataBlocking, player, -5, 0)
	if valid {
		t.Error("expected CheckPosition to block movement into 2-sided LinedefBlocking line")
	}
}

func TestStepUpLimit(t *testing.T) {
	// 1. Step up within limit (e.g. 16 units <= 24)
	mapDataStepOk := createTwoSectorMap(0, 128, 16, 128, wad.LinedefTwoSided, true)
	player := NewPlayerActor(-30, 0, 41, 0)
	player.FloorZ = 0

	valid, floorZ, _ := CheckPosition(mapDataStepOk, player, 20, 0)
	if !valid {
		t.Error("expected step up of 16 units to be permitted")
	}
	if floorZ != 16 {
		t.Errorf("expected floorZ 16, got %f", floorZ)
	}

	// 2. Step up exceeding limit (e.g. 32 units > 24)
	mapDataStepTooHigh := createTwoSectorMap(0, 128, 32, 128, wad.LinedefTwoSided, true)
	valid, _, _ = CheckPosition(mapDataStepTooHigh, player, 20, 0)
	if valid {
		t.Error("expected step up of 32 units to be BLOCKED (exceeds MaxStepHeight 24)")
	}
}

func TestLowCeilingCollision(t *testing.T) {
	// Sector 1 has ceiling of 40 (height = 40 - 0 = 40 < 56)
	mapDataLowCeiling := createTwoSectorMap(0, 128, 0, 40, wad.LinedefTwoSided, true)
	player := NewPlayerActor(-30, 0, 41, 0)
	player.FloorZ = 0

	valid, _, _ := CheckPosition(mapDataLowCeiling, player, 20, 0)
	if valid {
		t.Error("expected movement under low ceiling (40 units < 56 units) to be BLOCKED")
	}
}

func TestSmallFloorCrackPrevention(t *testing.T) {
	// 3 sectors: Left floor=0, Middle crack floor=-100 (width 8 units), Right floor=0
	vertexes := []wad.Vertex{
		{X: -50, Y: -50},
		{X: -4, Y: -50},
		{X: 4, Y: -50},
		{X: 50, Y: -50},
		{X: 50, Y: 50},
		{X: 4, Y: 50},
		{X: -4, Y: 50},
		{X: -50, Y: 50},
	}

	sectors := []wad.Sector{
		{FloorHeight: 0, CeilingHeight: 128},    // Sector 0: Left
		{FloorHeight: -100, CeilingHeight: 128}, // Sector 1: Narrow Crack (-4 to +4)
		{FloorHeight: 0, CeilingHeight: 128},    // Sector 2: Right
	}

	sidedefs := []wad.Sidedef{
		{Sector: 0},
		{Sector: 1},
		{Sector: 2},
	}

	linedefs := []wad.Linedef{
		// Left boundary of crack: x = -4
		{V1: 1, V2: 6, Flags: wad.LinedefTwoSided, RightSide: 0, LeftSide: 1},
		// Right boundary of crack: x = 4
		{V1: 2, V2: 5, Flags: wad.LinedefTwoSided, RightSide: 1, LeftSide: 2},
	}

	mapData := &wad.MapData{
		Name:       "CRACKMAP",
		Vertexes:   vertexes,
		Linedefs:   linedefs,
		Sidedefs:   sidedefs,
		Sectors:    sectors,
		Subsectors: []wad.Subsector{{NumSegs: 0, FirstSeg: 0}},
	}

	player := NewPlayerActor(-20, 0, 41, 0)
	player.FloorZ = 0

	// Player steps directly over the narrow 8-unit crack at x=0
	// Because player radius = 16, bounding box [-16, +16] spans into both Sector 0 and Sector 2
	valid, floorZ, _ := CheckPosition(mapData, player, 0, 0)
	if !valid {
		t.Fatal("expected position over small crack to be valid")
	}
	if floorZ != 0 {
		t.Errorf("expected player to remain on floor height 0 (spanning crack), got %f", floorZ)
	}
}

func TestSlideMove(t *testing.T) {
	// Wall at x=0 (1-sided)
	mapData := createTwoSectorMap(0, 128, 0, 128, 0, false)

	player := NewPlayerActor(-20, 0, 41, 0)
	player.FloorZ = 0

	// Moving diagonally (+10 in X towards wall, +10 in Y parallel to wall)
	// Full move (+10, +10) would reach x=-10 (within radius 16 of x=0) -> blocked
	// SlideMove should slide along Y: resulting in player staying at x=-20 and moving to y=+10
	moved := SlideMove(mapData, player, 10, 10)
	if !moved {
		t.Fatal("expected SlideMove to successfully slide along wall")
	}
	if player.X != -20 {
		t.Errorf("expected player X to remain at -20 (blocked by wall), got %f", player.X)
	}
	if player.Y != 10 {
		t.Errorf("expected player Y to slide to 10, got %f", player.Y)
	}
}

func TestMonsterBlockingLinedef(t *testing.T) {
	mapData := createTwoSectorMap(0, 128, 0, 128, wad.LinedefBlockMonsters|wad.LinedefTwoSided, true)

	player := NewPlayerActor(-30, 0, 41, 0)
	player.FloorZ = 0
	monster := NewActor(-30, 0, 41, 0, 16, 56, 41, 24, true)
	monster.FloorZ = 0

	// Player should NOT be blocked by LinedefBlockMonsters
	validPlayer, _, _ := CheckPosition(mapData, player, -5, 0)
	if !validPlayer {
		t.Error("expected player to pass through LinedefBlockMonsters line")
	}

	// Monster SHOULD be blocked by LinedefBlockMonsters
	validMonster, _, _ := CheckPosition(mapData, monster, -5, 0)
	if validMonster {
		t.Error("expected monster to be BLOCKED by LinedefBlockMonsters line")
	}
}

func TestMove(t *testing.T) {
	mapData := createTwoSectorMap(0, 128, 0, 128, wad.LinedefBlocking, true)
	player := NewPlayerActor(-50, 0, 41, 0) // Facing East towards blocking line at x=0

	// Move forward by 10 units (from -50 to -40) -> allowed
	Move(mapData, player, 10, 0, 0)
	if player.X != -40 || player.Y != 0 {
		t.Errorf("expected player position (-40, 0), got (%f, %f)", player.X, player.Y)
	}

	// Turn 90 degrees left (facing North)
	Move(mapData, player, 0, 0, 90)
	if player.Angle != 90 {
		t.Errorf("expected player angle 90, got %f", player.Angle)
	}

	// Move forward (North) by 15 units -> should change Y
	Move(mapData, player, 15, 0, 0)
	if player.Y != 15 {
		t.Errorf("expected player Y 15, got %f", player.Y)
	}
}

func TestFreedoomMAP01Collision(t *testing.T) {
	w, err := wad.Open("../../freedoom2.wad")
	if err != nil {
		t.Skip("freedoom2.wad not found, skipping map test")
	}
	defer w.Close()

	mapData, err := w.LoadMap("MAP01")
	if err != nil {
		t.Fatalf("failed to load MAP01: %v", err)
	}

	p1, ok := mapData.Player1Start()
	if !ok {
		t.Fatal("expected Player1Start in MAP01")
	}

	player := NewPlayerActor(float64(p1.X), float64(p1.Y), 41, float64(p1.Angle))
	if sec, ok := mapData.SectorAt(player.X, player.Y); ok && sec != nil {
		player.FloorZ = float64(sec.FloorHeight)
		player.Z = player.FloorZ + player.EyeHeight
	}

	// Player start position must be valid
	valid, _, _ := CheckPosition(mapData, player, player.X, player.Y)
	if !valid {
		t.Errorf("Player 1 start position (%f, %f) in MAP01 should be valid", player.X, player.Y)
	}

	// Slight movement forward from spawn should be valid
	origX, origY := player.X, player.Y
	Move(mapData, player, 5.0, 0, 0)
	if player.X == origX && player.Y == origY {
		t.Error("expected player to move forward in open spawn room")
	}
}

func TestLiftRidingAndMovement(t *testing.T) {
	// Sector 0 (left: x < 0) is a lift starting at floor 0.
	// Sector 1 (right: x > 0) is the upper landing at floor 128.
	mapData := createTwoSectorMap(0, 256, 128, 256, wad.LinedefTwoSided, true)

	player := NewPlayerActor(-50, 0, 41, 0) // Standing on lift in Sector 0, facing East (towards Sector 1)
	player.FloorZ = 0
	player.Z = 41

	// 1. Verify player can move around on the lift at floor 0
	Move(mapData, player, 10, 0, 0)
	if player.X != -40 {
		t.Errorf("expected player X -40, got %f", player.X)
	}

	// 2. The lift raises from floor 0 up to floor 128 (while player stands on it)
	mapData.Sectors[0].FloorHeight = 128

	// Verify UpdateActorFloor updates player floor and eye height
	UpdateActorFloor(mapData, player)
	if player.FloorZ != 128 {
		t.Errorf("expected player.FloorZ 128 after lift rises, got %f", player.FloorZ)
	}
	if player.Z != 128+player.EyeHeight {
		t.Errorf("expected player.Z %f, got %f", 128+player.EyeHeight, player.Z)
	}

	// 3. Player moves forward off the lift into Sector 1 (floor 128)
	// Because both sectors are now at floor 128, movement should NOT be blocked by MaxStepHeight!
	oldX := player.X
	Move(mapData, player, 30, 0, 0) // Moves East across x=0 into Sector 1
	if player.X <= oldX {
		t.Errorf("expected player to successfully step off the lift at floor 128, but remained at X %f", player.X)
	}
	if player.FloorZ != 128 {
		t.Errorf("expected player to remain at floor height 128, got %f", player.FloorZ)
	}
}

func TestGravityFallingAndLanding(t *testing.T) {
	mapData := createTwoSectorMap(0, 256, 0, 256, wad.LinedefTwoSided, true)

	// Spawn actor 100 units above floor (floor=0, eyeHeight=41, so groundZ=41)
	actor := NewPlayerActor(0, 0, 141, 0)
	actor.FloorZ = 0
	actor.Z = 141 // 100 units above ground
	actor.OnGround = false

	// Tick 1: VelZ should become -1.0, Z should become 140.0
	ApplyGravity(mapData, actor)
	if actor.VelZ != -1.0 {
		t.Errorf("expected VelZ -1.0 on tick 1, got %f", actor.VelZ)
	}
	if actor.Z != 140.0 {
		t.Errorf("expected Z 140.0 on tick 1, got %f", actor.Z)
	}
	if actor.OnGround {
		t.Error("expected actor to still be in air")
	}

	// Tick 2: VelZ should become -2.0, Z should become 138.0
	ApplyGravity(mapData, actor)
	if actor.VelZ != -2.0 {
		t.Errorf("expected VelZ -2.0 on tick 2, got %f", actor.VelZ)
	}
	if actor.Z != 138.0 {
		t.Errorf("expected Z 138.0 on tick 2, got %f", actor.Z)
	}

	// Simulate remaining ticks until actor lands on floor
	for i := 0; i < 50; i++ {
		ApplyGravity(mapData, actor)
		if actor.OnGround {
			break
		}
	}

	if !actor.OnGround {
		t.Errorf("expected actor to have landed on ground, Z=%f, VelZ=%f", actor.Z, actor.VelZ)
	}
	if actor.Z != 41.0 {
		t.Errorf("expected actor Z to land exactly at 41.0, got %f", actor.Z)
	}
	if actor.VelZ != 0.0 {
		t.Errorf("expected VelZ to be reset to 0 upon landing, got %f", actor.VelZ)
	}
}

func TestWalkingOffLedge(t *testing.T) {
	// Sector 0 (left: x < 0) has floor 100.
	// Sector 1 (right: x > 0) has floor 0 (100-unit drop).
	mapData := createTwoSectorMap(100, 256, 0, 256, wad.LinedefTwoSided, true)

	player := NewPlayerActor(-20, 0, 141, 0) // Standing on Sector 0 at floor 100, facing East
	player.FloorZ = 100
	player.Z = 141

	// Move forward across x=0 ledge into Sector 1
	oldX := player.X
	Move(mapData, player, 40, 0, 0)
	if player.X <= oldX {
		t.Fatalf("expected player to move across ledge, remained at X %f", player.X)
	}
	if player.FloorZ != 0 {
		t.Errorf("expected new FloorZ to be 0 beneath player, got %f", player.FloorZ)
	}
	// Player should still be at Z=141 in mid-air right after the horizontal move step
	if player.Z != 141 {
		t.Errorf("expected player to remain in mid-air at Z 141 immediately after walking off ledge, got %f", player.Z)
	}
	if player.OnGround {
		t.Error("expected player to be in air (OnGround=false)")
	}

	// Apply gravity ticks to fall down to groundZ=41
	for i := 0; i < 30; i++ {
		ApplyGravity(mapData, player)
		if player.OnGround {
			break
		}
	}

	if !player.OnGround {
		t.Errorf("expected player to have landed at floor 0 after falling, Z=%f", player.Z)
	}
	if player.Z != 41 {
		t.Errorf("expected player to land at Z 41, got %f", player.Z)
	}
}

func TestNoGravityFlag(t *testing.T) {
	mapData := createTwoSectorMap(0, 256, 0, 256, wad.LinedefTwoSided, true)

	// Floating entity (e.g. Cacodemon or flying projectile) at Z=100
	caco := NewActor(0, 0, 100, 0, 30, 56, 28, 24, true)
	caco.NoGravity = true

	// Tick gravity several times
	for i := 0; i < 10; i++ {
		ApplyGravity(mapData, caco)
	}

	if caco.Z != 100 {
		t.Errorf("expected NoGravity entity to remain at Z 100, got %f", caco.Z)
	}
	if caco.VelZ != 0 {
		t.Errorf("expected NoGravity entity to have VelZ 0, got %f", caco.VelZ)
	}
}

func TestCeilingCollision(t *testing.T) {
	// Sector with ceiling at 100, floor at 0
	mapData := createTwoSectorMap(0, 100, 0, 100, wad.LinedefTwoSided, true)

	// Player with height=56, eyeHeight=41 (head is 15 units above eye level)
	player := NewPlayerActor(0, 0, 41, 0)
	player.FloorZ = 0
	player.CeilingZ = 100
	player.VelZ = 20.0 // Jumping / launched upward

	// Apply gravity
	ApplyGravity(mapData, player)

	// At eyeHeight=41, head is at Z + 15 = 41 + 20 + 15 = 76 <= 100 (still climbing)
	// Next jump impulse to exceed ceiling
	player.VelZ = 30.0
	ApplyGravity(mapData, player)

	// Max head position is ceiling (100), so max eye level is 100 - (56 - 41) = 85
	if player.Z > 85.0 {
		t.Errorf("expected player eye level to be stopped by ceiling at <= 85.0, got %f", player.Z)
	}
	if player.VelZ != 0 {
		t.Errorf("expected upward velocity to be zeroed on ceiling impact, got %f", player.VelZ)
	}
}

func TestMaxFallSpeed(t *testing.T) {
	mapData := createTwoSectorMap(0, 10000, 0, 10000, wad.LinedefTwoSided, true)

	// Drop from extreme height
	player := NewPlayerActor(0, 0, 5000, 0)
	player.FloorZ = 0
	player.Z = 5000

	// Tick 100 times to exceed terminal velocity
	for i := 0; i < 100; i++ {
		ApplyGravity(mapData, player)
	}

	if player.VelZ < -DefaultMaxFallSpeed {
		t.Errorf("expected VelZ to be clamped to -%f, got %f", DefaultMaxFallSpeed, player.VelZ)
	}
}



