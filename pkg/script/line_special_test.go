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

func TestLineSpecial011_ExitLevel(t *testing.T) {
	mapData := createTestMapData()
	mapData.Name = "E1M1"

	var startedMap string
	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})
	repl.SetStartMapFunc(func(m string) {
		startedMap = m
	})

	if !repl.HasLineSpecial(11) {
		t.Fatal("expected line special 11 to exist")
	}

	// Trigger special 11 on line 0
	err := repl.TriggerLineSpecialSync(11, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("TriggerLineSpecialSync failed: %v", err)
	}

	if startedMap != "E1M2" {
		t.Errorf("expected next map 'E1M2', got %q", startedMap)
	}

	// Linedef special should be cleared (S1)
	if mapData.Linedefs[0].Special != 0 {
		t.Errorf("expected line special cleared to 0 (S1), got %d", mapData.Linedefs[0].Special)
	}

	// Test Doom 2 map naming: MAP01 -> MAP02
	mapData.Name = "MAP01"
	mapData.Linedefs[0].Special = 11
	err = repl.TriggerLineSpecialSync(11, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("TriggerLineSpecialSync failed: %v", err)
	}
	if startedMap != "MAP02" {
		t.Errorf("expected next map 'MAP02', got %q", startedMap)
	}
}

func TestLineSpecial036_LowerFloorTurbo(t *testing.T) {
	mapData := createTestMapData()
	// Initial state: sector 0 has floor 0, sector 2 has floor 16 and Tag 2.
	// We'll set sector 2 floor height to 128 and test lowering it down to 0 (lowest adjacent).
	mapData.Sectors[2].FloorHeight = 128

	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	if !repl.HasLineSpecial(36) {
		t.Fatal("expected line special 36 to exist")
	}

	// Trigger special 36 with Tag = 2 on line 2 (Special 36, Tag 2)
	mapData.Linedefs[2].Special = 36
	repl.TriggerLineSpecial(36, 2, 2, 0, 2)

	// Wait for turbo floor lower
	time.Sleep(200 * time.Millisecond)

	// Floor should have lowered towards 0
	h := mapData.Sectors[2].FloorHeight
	if h >= 128 {
		t.Errorf("expected sector 2 floor to lower below 128, got %d", h)
	}

	// Line special on line 2 should be cleared to 0 (W1)
	if mapData.Linedefs[2].Special != 0 {
		t.Errorf("expected line 2 special cleared to 0 (W1), got %d", mapData.Linedefs[2].Special)
	}
}

func TestLineSpecial048_ScrollTextureLeft(t *testing.T) {
	mapData := createTestMapData()
	mapData.Sidedefs[0].XOffset = 0
	mapData.Linedefs[0].RightSide = 0

	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	if !repl.HasLineSpecial(48) {
		t.Fatal("expected line special 48 to exist")
	}

	repl.TriggerLineSpecial(48, 0, 0, 0, 0)

	// Wait for scroller to increment x offset
	time.Sleep(100 * time.Millisecond)

	if mapData.Sidedefs[0].XOffset <= 0 {
		t.Errorf("expected sidedef 0 XOffset to be incremented (> 0), got %d", mapData.Sidedefs[0].XOffset)
	}
}

func TestLineSpecial088_PlatDownWaitUpStay(t *testing.T) {
	mapData := createTestMapData()
	// Sector 1 connects to Sector 0 (Floor 0) and Sector 2 (Floor 16)
	// Set sector 1 floor height to 100 initially
	mapData.Sectors[1].FloorHeight = 100

	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	if !repl.HasLineSpecial(88) {
		t.Fatal("expected line special 88 to exist")
	}

	// Trigger platform 88 on sec_id 1
	repl.TriggerLineSpecial(88, 0, 1, 0, 0)

	// Wait for platform to start lowering
	time.Sleep(100 * time.Millisecond)

	h := mapData.Sectors[1].FloorHeight
	if h >= 100 {
		t.Errorf("expected sector 1 floor height to decrease below 100, got %d", h)
	}
}

func TestE1M1SpecialsFromWAD(t *testing.T) {
	w, err := wad.Open("../../DoomShareware.wad")
	if err != nil {
		w, err = wad.Open("DoomShareware.wad")
		if err != nil {
			t.Skip("DoomShareware.wad not available in test environment")
		}
	}
	defer w.Close()

	mapData, err := w.LoadMap("E1M1")
	if err != nil {
		t.Fatalf("LoadMap(E1M1) failed: %v", err)
	}

	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	// Verify all specials used in E1M1 (1, 11, 36, 48, 88) exist and compile cleanly
	specialsInE1M1 := []int{1, 11, 36, 48, 88}
	for _, spec := range specialsInE1M1 {
		if !repl.HasLineSpecial(spec) {
			t.Errorf("expected line special %d to be available in embedded scripts", spec)
		}
		if _, err := repl.Cache().LoadLineSpecial(spec); err != nil {
			t.Errorf("failed to compile line special %d: %v", spec, err)
		}
	}
}

