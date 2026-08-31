package script

import (
	"testing"

	"github.com/qbradq/redoomed/pkg/wad"
)

func createTestMapData() *wad.MapData {
	return &wad.MapData{
		Name: "TESTMAP",
		Vertexes: []wad.Vertex{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
			{X: 100, Y: 100},
			{X: 0, Y: 100},
		},
		Sectors: []wad.Sector{
			{FloorHeight: 0, CeilingHeight: 128, FloorPic: "FLOOR0_1", CeilingPic: "CEIL1_1", LightLevel: 160, Tag: 1},
			{FloorHeight: 0, CeilingHeight: 0, FloorPic: "DOOR1", CeilingPic: "DOOR1", LightLevel: 140, Tag: 0}, // Door sector
			{FloorHeight: 16, CeilingHeight: 192, FloorPic: "FLOOR0_2", CeilingPic: "CEIL1_2", LightLevel: 180, Tag: 2},
		},
		Sidedefs: []wad.Sidedef{
			{Sector: 0}, // 0
			{Sector: 1}, // 1
			{Sector: 1}, // 2
			{Sector: 2}, // 3
		},
		Linedefs: []wad.Linedef{
			{V1: 0, V2: 1, Flags: wad.LinedefTwoSided, Special: 1, Tag: 0, RightSide: 0, LeftSide: 1},  // Line between Sec 0 and Sec 1
			{V1: 1, V2: 2, Flags: wad.LinedefTwoSided, Special: 0, Tag: 0, RightSide: 2, LeftSide: 3},  // Line between Sec 1 and Sec 2
			{V1: 2, V2: 3, Flags: 0, Special: 0, Tag: 2, RightSide: 3, LeftSide: 0xFFFF},                // 1-sided line
		},
		Things: []wad.Thing{
			{X: 50, Y: 50, Angle: 90, Type: wad.ThingPlayer1Start, Flags: 7},
			{X: 80, Y: 80, Angle: 180, Type: wad.ThingPlayer2Start, Flags: 7},
		},
	}
}

func TestMapModuleLines(t *testing.T) {
	mapData := createTestMapData()
	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	code := `
m := import("map")
nl := m.num_lines()
l0 := m.get_line(0)
l1 := m.get_line(1)
m.set_line_special(1, 11)
m.set_line_flags(0, 0x0001)
m.set_line_tag(0, 5)
tagged_lines := m.find_lines_by_tag(2)
`
	_, err := repl.Eval(code)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	res, err := repl.Eval("nl")
	if err != nil || res != "3" {
		t.Errorf("expected 3 lines, got %q (err: %v)", res, err)
	}

	res, err = repl.Eval("l0.special")
	if err != nil || res != "1" {
		t.Errorf("expected line 0 special 1, got %q (err: %v)", res, err)
	}

	res, err = repl.Eval("l0.front_sector")
	if err != nil || res != "0" {
		t.Errorf("expected line 0 front_sector 0, got %q", res)
	}

	res, err = repl.Eval("l0.back_sector")
	if err != nil || res != "1" {
		t.Errorf("expected line 0 back_sector 1, got %q", res)
	}

	// Verify mutations in mapData
	if mapData.Linedefs[1].Special != 11 {
		t.Errorf("expected line 1 special to be updated to 11, got %d", mapData.Linedefs[1].Special)
	}
	if mapData.Linedefs[0].Flags != 0x0001 {
		t.Errorf("expected line 0 flags to be updated to 1, got %d", mapData.Linedefs[0].Flags)
	}
	if mapData.Linedefs[0].Tag != 5 {
		t.Errorf("expected line 0 tag to be updated to 5, got %d", mapData.Linedefs[0].Tag)
	}

	res, err = repl.Eval("len(tagged_lines)")
	if err != nil || res != "1" {
		t.Errorf("expected 1 tagged line for tag 2, got %q", res)
	}
}

func TestMapModuleSectors(t *testing.T) {
	mapData := createTestMapData()
	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	code := `
m := import("map")
ns := m.num_sectors()
s0 := m.get_sector(0)
m.set_sector_ceiling_height(1, 120)
m.set_sector_floor_height(1, 10)
m.set_sector_light_level(0, 200)
m.set_sector_special(2, 9)
m.set_sector_tag(1, 7)
m.set_sector_floor_pic(0, "NUKAGE1")
m.set_sector_ceiling_pic(0, "F_SKY1")
tagged_secs := m.find_sectors_by_tag(7)
`
	_, err := repl.Eval(code)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	res, err := repl.Eval("ns")
	if err != nil || res != "3" {
		t.Errorf("expected 3 sectors, got %q", res)
	}

	res, err = repl.Eval("s0.floor_height")
	if err != nil || res != "0" {
		t.Errorf("expected s0 floor_height 0, got %q", res)
	}

	// Verify mutations in mapData
	if mapData.Sectors[1].CeilingHeight != 120 {
		t.Errorf("expected sector 1 ceiling height 120, got %d", mapData.Sectors[1].CeilingHeight)
	}
	if mapData.Sectors[1].FloorHeight != 10 {
		t.Errorf("expected sector 1 floor height 10, got %d", mapData.Sectors[1].FloorHeight)
	}
	if mapData.Sectors[0].LightLevel != 200 {
		t.Errorf("expected sector 0 light level 200, got %d", mapData.Sectors[0].LightLevel)
	}
	if mapData.Sectors[2].Special != 9 {
		t.Errorf("expected sector 2 special 9, got %d", mapData.Sectors[2].Special)
	}
	if mapData.Sectors[1].Tag != 7 {
		t.Errorf("expected sector 1 tag 7, got %d", mapData.Sectors[1].Tag)
	}
	if mapData.Sectors[0].FloorPic != "NUKAGE1" {
		t.Errorf("expected sector 0 floor pic NUKAGE1, got %q", mapData.Sectors[0].FloorPic)
	}
	if mapData.Sectors[0].CeilingPic != "F_SKY1" {
		t.Errorf("expected sector 0 ceiling pic F_SKY1, got %q", mapData.Sectors[0].CeilingPic)
	}

	res, err = repl.Eval("len(tagged_secs)")
	if err != nil || res != "1" {
		t.Errorf("expected 1 tagged sector for tag 7, got %q", res)
	}
}

func TestMapModuleAdjacentCalculations(t *testing.T) {
	mapData := createTestMapData()
	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	// Sector 1 (door) is adjacent to Sector 0 (ceiling 128, floor 0) and Sector 2 (ceiling 192, floor 16)
	code := `
m := import("map")
adj := m.get_adjacent_sectors(1)
lowest_ceil := m.get_lowest_adjacent_ceiling(1)
highest_ceil := m.get_highest_adjacent_ceiling(1)
lowest_floor := m.get_lowest_adjacent_floor(1)
highest_floor := m.get_highest_adjacent_floor(1)
`
	_, err := repl.Eval(code)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	res, err := repl.Eval("len(adj)")
	if err != nil || res != "2" {
		t.Errorf("expected 2 adjacent sectors for sector 1, got %q", res)
	}

	res, err = repl.Eval("lowest_ceil")
	if err != nil || res != "128" {
		t.Errorf("expected lowest adjacent ceiling 128, got %q", res)
	}

	res, err = repl.Eval("highest_ceil")
	if err != nil || res != "192" {
		t.Errorf("expected highest adjacent ceiling 192, got %q", res)
	}

	res, err = repl.Eval("lowest_floor")
	if err != nil || res != "0" {
		t.Errorf("expected lowest adjacent floor 0, got %q", res)
	}

	res, err = repl.Eval("highest_floor")
	if err != nil || res != "16" {
		t.Errorf("expected highest adjacent floor 16, got %q", res)
	}
}

func TestMapModuleThingsAndInfo(t *testing.T) {
	mapData := createTestMapData()
	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	code := `
m := import("map")
map_name := m.name()
nth := m.num_things()
nv := m.num_vertexes()
p1 := m.get_player(1)
th0 := m.get_thing(0)
v0 := m.get_vertex(0)
`
	_, err := repl.Eval(code)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	res, err := repl.Eval("map_name")
	if err != nil || (res != `"TESTMAP"` && res != "TESTMAP") {
		t.Errorf("expected map name 'TESTMAP', got %q", res)
	}

	res, err = repl.Eval("nth")
	if err != nil || res != "2" {
		t.Errorf("expected 2 things, got %q", res)
	}

	res, err = repl.Eval("nv")
	if err != nil || res != "4" {
		t.Errorf("expected 4 vertexes, got %q", res)
	}

	res, err = repl.Eval("p1.x")
	if err != nil || res != "50" {
		t.Errorf("expected p1.x 50, got %q", res)
	}

	res, err = repl.Eval("v0.x")
	if err != nil || res != "0" {
		t.Errorf("expected v0.x 0, got %q", res)
	}
}

func TestMapModuleSidedefs(t *testing.T) {
	mapData := createTestMapData()
	repl := NewREPL(nil, nil)
	repl.SetMapDataProvider(func() *wad.MapData {
		return mapData
	})

	code := `
m := import("map")
ns := m.num_sidedefs()
s0 := m.get_sidedef(0)
m.set_sidedef_x_offset(0, 15)
m.set_sidedef_y_offset(0, 25)
m.set_sidedef_upper_texture(0, "STARTAN2")
m.set_sidedef_lower_texture(0, "DOOR3")
m.set_sidedef_middle_texture(0, "WALL1")
s0_after := m.get_sidedef(0)
`
	_, err := repl.Eval(code)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	res, err := repl.Eval("ns")
	if err != nil || res != "4" {
		t.Errorf("expected 4 sidedefs, got %q", res)
	}

	res, err = repl.Eval("s0_after.x_offset")
	if err != nil || res != "15" {
		t.Errorf("expected x_offset 15, got %q", res)
	}

	res, err = repl.Eval("s0_after.y_offset")
	if err != nil || res != "25" {
		t.Errorf("expected y_offset 25, got %q", res)
	}

	res, err = repl.Eval("s0_after.upper_texture")
	if err != nil || (res != `"STARTAN2"` && res != "STARTAN2") {
		t.Errorf("expected upper_texture 'STARTAN2', got %q", res)
	}

	if mapData.Sidedefs[0].XOffset != 15 || mapData.Sidedefs[0].YOffset != 25 {
		t.Errorf("expected Go mapData sidedef 0 updated, got %+v", mapData.Sidedefs[0])
	}
}

