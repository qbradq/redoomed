package script

import (
	"testing"
	"time"

	"github.com/qbradq/redoomed/pkg/wad"
)

func TestLineSpecial001Embedded(t *testing.T) {
	mapData := createTestMapData()
	// Initial state: sector 1 ceiling is 0
	if mapData.Sectors[1].CeilingHeight != 0 {
		t.Fatalf("expected initial sector 1 ceiling height 0, got %d", mapData.Sectors[1].CeilingHeight)
	}

	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	if !repl.HasLineSpecial(1) {
		t.Fatal("expected line special 1 (line_001.tengo) to be discovered in embedded FS")
	}

	// Trigger special 1 for line 0 (which connects to sector 1 on back side)
	repl.TriggerLineSpecial(1, 0, 0, 0, 0)

	// Wait for door to start opening
	time.Sleep(150 * time.Millisecond)

	// Ceiling should have raised
	h := mapData.Sectors[1].CeilingHeight
	if h <= 0 {
		t.Errorf("expected sector 1 ceiling height to increase (> 0), got %d", h)
	}
}

func TestLineSpecial001TargetCalculations(t *testing.T) {
	mapData := createTestMapData()
	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	// Direct test of door opening math via map module
	code := `
m := import("map")
sec_id := 1
dest_ceil := m.get_lowest_adjacent_ceiling(sec_id) - 4
`
	_, err := repl.Eval(code)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	res, err := repl.Eval("dest_ceil")
	if err != nil || res != "124" {
		t.Errorf("expected destination ceiling 124 (128 - 4), got %q", res)
	}
}
