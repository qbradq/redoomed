package physics

import (
	"testing"

	"github.com/qbradq/redoomed/pkg/wad"
)

func TestUseLineBasic(t *testing.T) {
	// Map layout:
	// A horizontal line from (0, 30) to (100, 30)
	// Player at (50, 0) facing North (Angle = 90)
	// Distance to line is 30, which is <= DefaultUseRange (64)
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
				Special: 1, // Manual door
				Tag:     10,
			},
		},
	}

	actor := NewPlayerActor(50, 0, 41, 90)

	lineIdx, ld, dist, hit := UseLine(mapData, actor, DefaultUseRange)
	if !hit {
		t.Fatalf("expected UseLine to hit linedef, but got hit=false")
	}
	if lineIdx != 0 {
		t.Errorf("expected lineIdx 0, got %d", lineIdx)
	}
	if ld == nil || ld.Special != 1 || ld.Tag != 10 {
		t.Errorf("expected ld Special=1, Tag=10, got %+v", ld)
	}
	if dist < 29.9 || dist > 30.1 {
		t.Errorf("expected dist ~30.0, got %f", dist)
	}

	// Facing away (South, Angle = 270): should not hit
	actor.Angle = 270
	_, _, _, hitAway := UseLine(mapData, actor, DefaultUseRange)
	if hitAway {
		t.Errorf("expected facing away not to hit linedef")
	}

	// Out of range (Player at 50, -70, facing North): distance is 100 > 64
	actor.Y = -70
	actor.Angle = 90
	_, _, _, hitOutOfRange := UseLine(mapData, actor, DefaultUseRange)
	if hitOutOfRange {
		t.Errorf("expected out of range not to hit linedef")
	}
}

func TestUseLineMultipleLinesClosest(t *testing.T) {
	// Line 0 at Y = 50
	// Line 1 at Y = 25
	mapData := &wad.MapData{
		Vertexes: []wad.Vertex{
			{X: 0, Y: 50},   // 0
			{X: 100, Y: 50}, // 1
			{X: 0, Y: 25},   // 2
			{X: 100, Y: 25}, // 3
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, Special: 5, Tag: 1},
			{V1: 2, V2: 3, Special: 9, Tag: 2},
		},
	}

	actor := NewPlayerActor(50, 0, 41, 90)

	lineIdx, ld, dist, hit := UseLine(mapData, actor, DefaultUseRange)
	if !hit {
		t.Fatalf("expected hit, got false")
	}
	if lineIdx != 1 {
		t.Errorf("expected closest lineIdx 1, got %d", lineIdx)
	}
	if ld == nil || ld.Special != 9 {
		t.Errorf("expected Special 9, got %+v", ld)
	}
	if dist < 24.9 || dist > 25.1 {
		t.Errorf("expected dist ~25.0, got %f", dist)
	}
}

func TestUseLineNilOrEmpty(t *testing.T) {
	actor := NewPlayerActor(0, 0, 0, 0)
	if _, _, _, hit := UseLine(nil, actor, 64); hit {
		t.Errorf("expected no hit for nil map")
	}
	if _, _, _, hit := UseLine(&wad.MapData{}, nil, 64); hit {
		t.Errorf("expected no hit for nil actor")
	}
	if _, _, _, hit := UseLine(&wad.MapData{}, actor, 64); hit {
		t.Errorf("expected no hit for empty map")
	}
}

func TestUseLineTraverseTwoSidedNonSpecial(t *testing.T) {
	// Simulates doorway recess like E1M1 line 310 / line 340:
	// Line 0 (door recess opening) at Y = 20, 2-sided, Special = 0
	// Line 1 (actual door) at Y = 36, 2-sided, Special = 1
	mapData := &wad.MapData{
		Vertexes: []wad.Vertex{
			{X: 0, Y: 20},   // 0
			{X: 100, Y: 20}, // 1
			{X: 0, Y: 36},   // 2
			{X: 100, Y: 36}, // 3
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, Flags: wad.LinedefTwoSided, Special: 0, Tag: 0, RightSide: 0, LeftSide: 1}, // Line 0 (recess opening)
			{V1: 2, V2: 3, Flags: wad.LinedefTwoSided, Special: 1, Tag: 0, RightSide: 2, LeftSide: 3}, // Line 1 (door)
		},
	}

	actor := NewPlayerActor(50, 0, 41, 90) // Player at Y=0 facing North

	lineIdx, ld, dist, hit := UseLine(mapData, actor, DefaultUseRange)
	if !hit {
		t.Fatalf("expected hit, got false")
	}
	if lineIdx != 1 {
		t.Errorf("expected use ray to pass through line 0 and hit door line 1, got lineIdx %d", lineIdx)
	}
	if ld == nil || ld.Special != 1 {
		t.Errorf("expected Special 1, got %+v", ld)
	}
	if dist < 35.9 || dist > 36.1 {
		t.Errorf("expected dist ~36.0, got %f", dist)
	}
}


func TestCheckCrossedLines(t *testing.T) {
	mapData := &wad.MapData{
		Vertexes: []wad.Vertex{
			{X: 0, Y: 50},   // 0
			{X: 100, Y: 50}, // 1
			{X: 50, Y: 0},   // 2
			{X: 50, Y: 100}, // 3
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, Special: 88, Tag: 2}, // Line 0: Horizontal line at Y=50
			{V1: 2, V2: 3, Special: 36, Tag: 1}, // Line 1: Vertical line at X=50
		},
	}

	// Move from (25, 10) to (25, 90) crosses Line 0
	crossed := CheckCrossedLines(mapData, 25, 10, 25, 90)
	if len(crossed) != 1 || crossed[0] != 0 {
		t.Errorf("expected crossing line 0, got %v", crossed)
	}

	// Move from (10, 25) to (90, 25) crosses Line 1
	crossed = CheckCrossedLines(mapData, 10, 25, 90, 25)
	if len(crossed) != 1 || crossed[0] != 1 {
		t.Errorf("expected crossing line 1, got %v", crossed)
	}

	// Move from (10, 10) to (90, 90) crosses both Line 0 and Line 1
	crossed = CheckCrossedLines(mapData, 10, 10, 90, 90)
	if len(crossed) != 2 {
		t.Errorf("expected crossing 2 lines, got %v", crossed)
	}

	// Move without crossing (from 10, 10 to 20, 20)
	crossed = CheckCrossedLines(mapData, 10, 10, 20, 20)
	if len(crossed) != 0 {
		t.Errorf("expected no crossings, got %v", crossed)
	}

	// Nil / empty checks
	if CheckCrossedLines(nil, 0, 0, 10, 10) != nil {
		t.Errorf("expected nil for nil mapData")
	}
	if CheckCrossedLines(mapData, 10, 10, 10, 10) != nil {
		t.Errorf("expected nil for stationary point")
	}
}

