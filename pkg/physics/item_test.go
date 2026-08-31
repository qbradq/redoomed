package physics

import (
	"testing"

	"github.com/qbradq/redoomed/pkg/wad"
)

func TestCheckItemTouch(t *testing.T) {
	player := NewPlayerActor(100, 100, 41, 0)
	player.FloorZ = 0
	player.CeilingZ = 128
	player.Radius = 16.0
	player.Height = 56.0

	item := &wad.ItemEntity{
		ID:        0,
		X:         110,
		Y:         100,
		FloorZ:    0,
		CeilingZ:  128,
		Radius:    20.0,
		Height:    16.0,
		Collected: false,
	}

	// 1. In range horizontally and vertically
	if !CheckItemTouch(player, item) {
		t.Errorf("expected player to touch item at (110, 100)")
	}

	// 2. Far away horizontally
	item.X = 200
	if CheckItemTouch(player, item) {
		t.Errorf("expected player not to touch distant item at (200, 100)")
	}

	// 3. Already collected
	item.X = 110
	item.Collected = true
	if CheckItemTouch(player, item) {
		t.Errorf("expected collected item not to be touched")
	}

	// 4. Vertical separation (item far above player)
	item.Collected = false
	item.FloorZ = 200
	item.CeilingZ = 300
	if CheckItemTouch(player, item) {
		t.Errorf("expected item on high ledge (FloorZ=200) not to touch player on ground (FloorZ=0)")
	}
}

func TestTouchItems(t *testing.T) {
	player := NewPlayerActor(100, 100, 41, 0)
	player.FloorZ = 0
	player.CeilingZ = 128

	items := []*wad.ItemEntity{
		{ID: 0, X: 110, Y: 100, FloorZ: 0, Radius: 20, Height: 16},
		{ID: 1, X: 500, Y: 500, FloorZ: 0, Radius: 20, Height: 16},
		{ID: 2, X: 105, Y: 105, FloorZ: 0, Radius: 20, Height: 16},
	}

	touched := TouchItems(player, items)
	if len(touched) != 2 {
		t.Fatalf("expected 2 touched items, got %d", len(touched))
	}
	if touched[0].ID != 0 || touched[1].ID != 2 {
		t.Errorf("touched items mismatch: %+v", touched)
	}
}
