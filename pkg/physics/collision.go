package physics

import (
	"math"

	"github.com/qbradq/redoomed/pkg/wad"
)

// SampleFloorCeiling calculates the highest floor and lowest ceiling within the actor's 2D bounding box at (x, y).
// It samples 9 points across the actor bounding box and includes adjacent sectors from intersecting 2-sided linedefs.
func SampleFloorCeiling(mapData *wad.MapData, actor *Actor, x, y float64) (floorZ, ceilingZ float64, ok bool) {
	if mapData == nil || actor == nil {
		return 0, 0, false
	}

	r := actor.Radius
	boxMinX := x - r
	boxMaxX := x + r
	boxMinY := y - r
	boxMaxY := y + r

	// 1. Gather all sectors covered by the actor's 2D bounding box
	samplePoints := [9][2]float64{
		{x, y},             // Center
		{boxMinX, boxMinY}, // Bottom-left
		{boxMaxX, boxMinY}, // Bottom-right
		{boxMinX, boxMaxY}, // Top-left
		{boxMaxX, boxMaxY}, // Top-right
		{boxMinX, y},       // Mid-left
		{boxMaxX, y},       // Mid-right
		{x, boxMinY},       // Mid-bottom
		{x, boxMaxY},       // Mid-top
	}

	highestFloor := -math.MaxFloat64
	lowestCeiling := math.MaxFloat64
	sectorFound := false

	for _, pt := range samplePoints {
		if sec, found := mapData.SectorAt(pt[0], pt[1]); found && sec != nil {
			sectorFound = true
			fh := float64(sec.FloorHeight)
			ch := float64(sec.CeilingHeight)
			if fh > highestFloor {
				highestFloor = fh
			}
			if ch < lowestCeiling {
				lowestCeiling = ch
			}
		}
	}

	// Also check sectors from 2-sided linedefs intersecting the actor's bounding box
	for i := range mapData.Linedefs {
		ld := &mapData.Linedefs[i]
		if int(ld.V1) >= len(mapData.Vertexes) || int(ld.V2) >= len(mapData.Vertexes) {
			continue
		}
		v1 := mapData.Vertexes[ld.V1]
		v2 := mapData.Vertexes[ld.V2]

		if !lineBoxOverlap(v1.X, v1.Y, v2.X, v2.Y, boxMinX, boxMaxX, boxMinY, boxMaxY) {
			continue
		}

		if ld.LeftSide != 0xFFFF {
			if int(ld.RightSide) < len(mapData.Sidedefs) && int(ld.LeftSide) < len(mapData.Sidedefs) {
				rs := &mapData.Sidedefs[ld.RightSide]
				ls := &mapData.Sidedefs[ld.LeftSide]
				if int(rs.Sector) < len(mapData.Sectors) && int(ls.Sector) < len(mapData.Sectors) {
					s1 := &mapData.Sectors[rs.Sector]
					s2 := &mapData.Sectors[ls.Sector]
					sectorFound = true
					if float64(s1.FloorHeight) > highestFloor {
						highestFloor = float64(s1.FloorHeight)
					}
					if float64(s2.FloorHeight) > highestFloor {
						highestFloor = float64(s2.FloorHeight)
					}
					if float64(s1.CeilingHeight) < lowestCeiling {
						lowestCeiling = float64(s1.CeilingHeight)
					}
					if float64(s2.CeilingHeight) < lowestCeiling {
						lowestCeiling = float64(s2.CeilingHeight)
					}
				}
			}
		}
	}

	if !sectorFound {
		return actor.FloorZ, actor.CeilingZ, false
	}
	return highestFloor, lowestCeiling, true
}

// ApplyGravity applies gravitational acceleration and vertical collision handling (floor landing, ceiling bumping) to the actor for one simulation tick.
func ApplyGravity(mapData *wad.MapData, actor *Actor) {
	if mapData == nil || actor == nil {
		return
	}

	// 1. Refresh floor and ceiling boundaries at current actor position
	if floorZ, ceilingZ, ok := SampleFloorCeiling(mapData, actor, actor.X, actor.Y); ok {
		actor.FloorZ = floorZ
		actor.CeilingZ = ceilingZ
	}

	targetGroundZ := actor.FloorZ + actor.EyeHeight
	headOffset := actor.Height - actor.EyeHeight

	gravity := actor.Gravity
	if gravity <= 0 {
		gravity = DefaultGravity
	}
	maxFall := actor.MaxFallSpeed
	if maxFall <= 0 {
		maxFall = DefaultMaxFallSpeed
	}

	if actor.NoGravity {
		// Flying / floating entity: enforce floor and ceiling limits without gravity
		if actor.Z < targetGroundZ {
			actor.Z = targetGroundZ
			actor.VelZ = 0
			actor.OnGround = true
		}
		if actor.Z+headOffset > actor.CeilingZ {
			actor.Z = actor.CeilingZ - headOffset
			if actor.VelZ > 0 {
				actor.VelZ = 0
			}
		}
		return
	}

	// Dynamic gravity simulation
	if actor.Z > targetGroundZ {
		// In mid-air: accelerate downwards
		actor.OnGround = false
		actor.VelZ -= gravity
		if actor.VelZ < -maxFall {
			actor.VelZ = -maxFall
		}

		actor.Z += actor.VelZ

		// Check landing on floor
		if actor.Z <= targetGroundZ {
			actor.Z = targetGroundZ
			actor.VelZ = 0
			actor.OnGround = true
		}

		// Check ceiling bump
		if actor.Z+headOffset > actor.CeilingZ {
			actor.Z = actor.CeilingZ - headOffset
			if actor.VelZ > 0 {
				actor.VelZ = 0
			}
		}
	} else if actor.Z < targetGroundZ {
		// Floor rose under the actor (e.g. lift/platform rising)
		actor.Z = targetGroundZ
		actor.VelZ = 0
		actor.OnGround = true
	} else {
		// Resting on floor (actor.Z == targetGroundZ)
		if actor.VelZ > 0 {
			// Upward momentum (e.g. jump)
			actor.Z += actor.VelZ
			actor.OnGround = false

			// Check ceiling bump
			if actor.Z+headOffset > actor.CeilingZ {
				actor.Z = actor.CeilingZ - headOffset
				actor.VelZ = 0
			}
		} else {
			actor.VelZ = 0
			actor.OnGround = true
		}
	}
}

// UpdateActorFloor updates the actor's FloorZ, CeilingZ, and vertical Z based on current map sector heights at (actor.X, actor.Y).
// It applies gravity to simulate falling or riding moving floors.
func UpdateActorFloor(mapData *wad.MapData, actor *Actor) {
	ApplyGravity(mapData, actor)
}

// CheckPosition tests whether the actor can legally stand at (targetX, targetY).
// It verifies:
// 1. Solid wall linedef collisions (1-sided and blocking 2-sided).
// 2. Step up height (cannot step up higher than MaxStepHeight).
// 3. Low ceiling clearance (cannot squeeze under ceilings lower than actor Height).
// 4. Multi-sector bounding box sampling for floor and ceiling heights (preventing falling into small cracks).
func CheckPosition(mapData *wad.MapData, actor *Actor, targetX, targetY float64) (valid bool, floorZ float64, ceilingZ float64) {
	if mapData == nil {
		return true, actor.FloorZ, actor.CeilingZ
	}

	// 1. Resync actor's current floor/ceiling state to handle moving sectors (lifts, crushers)
	if curFloor, curCeil, ok := SampleFloorCeiling(mapData, actor, actor.X, actor.Y); ok {
		actor.FloorZ = curFloor
		actor.CeilingZ = curCeil
		if actor.OnGround && actor.Z < curFloor+actor.EyeHeight {
			actor.Z = curFloor + actor.EyeHeight
		}
	}

	// 2. Sample target location floor and ceiling
	highestFloor, lowestCeiling, sectorFound := SampleFloorCeiling(mapData, actor, targetX, targetY)
	if !sectorFound {
		return false, actor.FloorZ, actor.CeilingZ
	}

	// 3. Prevent actor from stepping up too high
	if highestFloor-actor.FloorZ > actor.MaxStepHeight {
		return false, highestFloor, lowestCeiling
	}

	// 4. Prevent actor from squeezing under low ceilings
	if lowestCeiling-highestFloor < actor.Height {
		return false, highestFloor, lowestCeiling
	}
	if lowestCeiling-actor.FloorZ < actor.Height {
		return false, highestFloor, lowestCeiling
	}

	// 5. Check collisions against all linedefs
	r := actor.Radius
	boxMinX := targetX - r
	boxMaxX := targetX + r
	boxMinY := targetY - r
	boxMaxY := targetY + r
	for i := range mapData.Linedefs {
		ld := &mapData.Linedefs[i]
		if int(ld.V1) >= len(mapData.Vertexes) || int(ld.V2) >= len(mapData.Vertexes) {
			continue
		}
		v1 := mapData.Vertexes[ld.V1]
		v2 := mapData.Vertexes[ld.V2]

		if !lineBoxOverlap(v1.X, v1.Y, v2.X, v2.Y, boxMinX, boxMaxX, boxMinY, boxMaxY) {
			continue
		}

		dist := pointToSegmentDistance(targetX, targetY, v1.X, v1.Y, v2.X, v2.Y)
		if dist >= r {
			continue
		}

		// Line penetrates actor's collision radius. Check passability:
		// A. 1-sided line: always solid
		if ld.LeftSide == 0xFFFF {
			return false, highestFloor, lowestCeiling
		}

		// B. Blocking flags
		if ld.Flags&wad.LinedefBlocking != 0 {
			return false, highestFloor, lowestCeiling
		}
		if actor.IsMonster && (ld.Flags&wad.LinedefBlockMonsters != 0) {
			return false, highestFloor, lowestCeiling
		}

		// C. 2-sided line opening checks
		if int(ld.RightSide) >= len(mapData.Sidedefs) || int(ld.LeftSide) >= len(mapData.Sidedefs) {
			return false, highestFloor, lowestCeiling
		}
		frontSide := &mapData.Sidedefs[ld.RightSide]
		backSide := &mapData.Sidedefs[ld.LeftSide]
		if int(frontSide.Sector) >= len(mapData.Sectors) || int(backSide.Sector) >= len(mapData.Sectors) {
			return false, highestFloor, lowestCeiling
		}

		frontSec := &mapData.Sectors[frontSide.Sector]
		backSec := &mapData.Sectors[backSide.Sector]

		openFloor := math.Max(float64(frontSec.FloorHeight), float64(backSec.FloorHeight))
		openCeil := math.Min(float64(frontSec.CeilingHeight), float64(backSec.CeilingHeight))

		// Check vertical opening clearance
		if openCeil-openFloor < actor.Height {
			return false, highestFloor, lowestCeiling
		}
		if openCeil-actor.FloorZ < actor.Height {
			return false, highestFloor, lowestCeiling
		}

		// Check step up over the line
		if openFloor-actor.FloorZ > actor.MaxStepHeight {
			return false, highestFloor, lowestCeiling
		}
	}

	return true, highestFloor, lowestCeiling
}

// TryMove attempts to move the actor directly to (targetX, targetY).
// If the move is valid, the actor's position, FloorZ, and CeilingZ are updated, returning true.
// Vertical elevation (Z) is preserved when walking off ledges so gravity can take effect.
func TryMove(mapData *wad.MapData, actor *Actor, targetX, targetY float64) bool {
	if mapData == nil {
		actor.X = targetX
		actor.Y = targetY
		return true
	}

	valid, floorZ, ceilingZ := CheckPosition(mapData, actor, targetX, targetY)
	if !valid {
		return false
	}

	oldFloorZ := actor.FloorZ
	actor.X = targetX
	actor.Y = targetY
	actor.FloorZ = floorZ
	actor.CeilingZ = ceilingZ

	groundZ := floorZ + actor.EyeHeight
	if floorZ > oldFloorZ {
		// Stepped up onto higher floor
		if actor.Z < groundZ {
			actor.Z = groundZ
			actor.OnGround = true
			if actor.VelZ < 0 {
				actor.VelZ = 0
			}
		}
	} else if floorZ < oldFloorZ {
		// Stepped down or walked off a ledge
		// If actor was grounded, they are now in the air; let gravity accelerate them downwards
		if actor.Z <= groundZ {
			actor.Z = groundZ
			actor.OnGround = true
		} else {
			actor.OnGround = false
		}
	} else {
		// Floor height unchanged
		if actor.Z <= groundZ {
			actor.Z = groundZ
			actor.OnGround = true
		}
	}

	return true
}

// SlideMove attempts full movement (dx, dy). If blocked, it attempts axis-aligned sliding along X or Y.
func SlideMove(mapData *wad.MapData, actor *Actor, dx, dy float64) bool {
	if dx == 0 && dy == 0 {
		return false
	}

	// 1. Try full direct move
	if TryMove(mapData, actor, actor.X+dx, actor.Y+dy) {
		return true
	}

	// 2. Try X-only slide
	moved := false
	if dx != 0 {
		if TryMove(mapData, actor, actor.X+dx, actor.Y) {
			moved = true
		}
	}

	// 3. Try Y-only slide
	if dy != 0 {
		if TryMove(mapData, actor, actor.X, actor.Y+dy) {
			moved = true
		}
	}

	return moved
}

// Move calculates displacement vectors from forward, strafe, and turn deltas, then executes sliding movement.
func Move(mapData *wad.MapData, actor *Actor, forward, strafe, turn float64) {
	if actor == nil {
		return
	}

	actor.Angle += turn
	for actor.Angle < 0 {
		actor.Angle += 360
	}
	for actor.Angle >= 360 {
		actor.Angle -= 360
	}

	rad := actor.Angle * math.Pi / 180.0
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	// Forward movement (+cos, +sin)
	dx := forward * cosA
	dy := forward * sinA

	// Strafe movement (+sin, -cos)
	dx += strafe * sinA
	dy += strafe * (-cosA)

	SlideMove(mapData, actor, dx, dy)
}

// lineBoxOverlap tests if line segment (x1, y1)-(x2, y2) bounding box overlaps AABB [minX, maxX, minY, maxY].
func lineBoxOverlap(x1, y1, x2, y2 int16, minX, maxX, minY, maxY float64) bool {
	lMinX := math.Min(float64(x1), float64(x2))
	lMaxX := math.Max(float64(x1), float64(x2))
	lMinY := math.Min(float64(y1), float64(y2))
	lMaxY := math.Max(float64(y1), float64(y2))
	return !(lMaxX < minX || lMinX > maxX || lMaxY < minY || lMinY > maxY)
}

// pointToSegmentDistance computes the shortest Euclidean distance from (px, py) to segment (x1, y1)-(x2, y2).
func pointToSegmentDistance(px, py float64, x1, y1, x2, y2 int16) float64 {
	fx1, fy1 := float64(x1), float64(y1)
	fx2, fy2 := float64(x2), float64(y2)
	dx := fx2 - fx1
	dy := fy2 - fy1
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(px-fx1, py-fy1)
	}
	t := ((px-fx1)*dx + (py-fy1)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	projX := fx1 + t*dx
	projY := fy1 + t*dy
	return math.Hypot(px-projX, py-projY)
}
