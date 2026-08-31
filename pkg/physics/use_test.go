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
