package wad

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestParseVertexes(t *testing.T) {
	buf := new(bytes.Buffer)
	// Write 2 vertices: (100, 200), (-50, -150)
	_ = binary.Write(buf, binary.LittleEndian, int16(100))
	_ = binary.Write(buf, binary.LittleEndian, int16(200))
	_ = binary.Write(buf, binary.LittleEndian, int16(-50))
	_ = binary.Write(buf, binary.LittleEndian, int16(-150))

	verts, err := ParseVertexes(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseVertexes failed: %v", err)
	}
	if len(verts) != 2 {
		t.Fatalf("expected 2 vertices, got %d", len(verts))
	}
	if verts[0].X != 100 || verts[0].Y != 200 {
		t.Errorf("vertex 0 mismatch: %+v", verts[0])
	}
	if verts[1].X != -50 || verts[1].Y != -150 {
		t.Errorf("vertex 1 mismatch: %+v", verts[1])
	}

	// Invalid byte length
	_, err = ParseVertexes([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error on invalid vertex data length")
	}
}

func TestParseLinedefs(t *testing.T) {
	buf := new(bytes.Buffer)
	ld := Linedef{
		V1:        0,
		V2:        1,
		Flags:     LinedefBlocking | LinedefTwoSided,
		Special:   0,
		Tag:       0,
		RightSide: 0,
		LeftSide:  1,
	}
	_ = binary.Write(buf, binary.LittleEndian, ld)

	linedefs, err := ParseLinedefs(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseLinedefs failed: %v", err)
	}
	if len(linedefs) != 1 {
		t.Fatalf("expected 1 linedef, got %d", len(linedefs))
	}
	if linedefs[0].V1 != 0 || linedefs[0].V2 != 1 || linedefs[0].Flags != (LinedefBlocking|LinedefTwoSided) {
		t.Errorf("linedef 0 mismatch: %+v", linedefs[0])
	}

	// Invalid byte length
	_, err = ParseLinedefs([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error on invalid linedef data length")
	}
}

func TestParseSidedefs(t *testing.T) {
	buf := new(bytes.Buffer)
	var raw rawSidedef
	raw.XOffset = 10
	raw.YOffset = 20
	copy(raw.UpperTexture[:], "STARTAN2")
	copy(raw.LowerTexture[:], "DOOR3")
	copy(raw.MiddleTexture[:], "-")
	raw.Sector = 5
	_ = binary.Write(buf, binary.LittleEndian, raw)

	sidedefs, err := ParseSidedefs(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseSidedefs failed: %v", err)
	}
	if len(sidedefs) != 1 {
		t.Fatalf("expected 1 sidedef, got %d", len(sidedefs))
	}
	s := sidedefs[0]
	if s.XOffset != 10 || s.YOffset != 20 || s.UpperTexture != "STARTAN2" || s.LowerTexture != "DOOR3" || s.MiddleTexture != "-" || s.Sector != 5 {
		t.Errorf("sidedef mismatch: %+v", s)
	}

	_, err = ParseSidedefs([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error on invalid sidedef data length")
	}
}

func TestParseSectors(t *testing.T) {
	buf := new(bytes.Buffer)
	var raw rawSector
	raw.FloorHeight = 0
	raw.CeilingHeight = 128
	copy(raw.FloorPic[:], "FLOOR4_8")
	copy(raw.CeilingPic[:], "CEIL1_1")
	raw.LightLevel = 160
	raw.Special = 0
	raw.Tag = 1
	_ = binary.Write(buf, binary.LittleEndian, raw)

	sectors, err := ParseSectors(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseSectors failed: %v", err)
	}
	if len(sectors) != 1 {
		t.Fatalf("expected 1 sector, got %d", len(sectors))
	}
	sec := sectors[0]
	if sec.FloorHeight != 0 || sec.CeilingHeight != 128 || sec.FloorPic != "FLOOR4_8" || sec.CeilingPic != "CEIL1_1" || sec.LightLevel != 160 || sec.Tag != 1 {
		t.Errorf("sector mismatch: %+v", sec)
	}

	_, err = ParseSectors([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error on invalid sector data length")
	}
}

func TestParseSegs(t *testing.T) {
	buf := new(bytes.Buffer)
	seg := Seg{
		V1:        1,
		V2:        2,
		Angle:     -16384,
		Linedef:   10,
		Direction: 0,
		Offset:    0,
	}
	_ = binary.Write(buf, binary.LittleEndian, seg)

	segs, err := ParseSegs(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseSegs failed: %v", err)
	}
	if len(segs) != 1 || segs[0].V1 != 1 || segs[0].V2 != 2 || segs[0].Linedef != 10 {
		t.Errorf("seg mismatch: %+v", segs)
	}

	_, err = ParseSegs([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error on invalid seg data length")
	}
}

func TestParseSubsectors(t *testing.T) {
	buf := new(bytes.Buffer)
	ss := Subsector{
		NumSegs:  4,
		FirstSeg: 12,
	}
	_ = binary.Write(buf, binary.LittleEndian, ss)

	subsectors, err := ParseSubsectors(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseSubsectors failed: %v", err)
	}
	if len(subsectors) != 1 || subsectors[0].NumSegs != 4 || subsectors[0].FirstSeg != 12 {
		t.Errorf("subsector mismatch: %+v", subsectors)
	}

	_, err = ParseSubsectors([]byte{1, 2})
	if err == nil {
		t.Error("expected error on invalid subsector data length")
	}
}

func TestParseNodes(t *testing.T) {
	buf := new(bytes.Buffer)
	node := Node{
		PartitionX:       100,
		PartitionY:       200,
		ChangeX:          0,
		ChangeY:          50,
		RightBoundingBox: [4]int16{250, 200, 100, 150},
		LeftBoundingBox:  [4]int16{250, 200, 50, 100},
		RightChild:       0x8001, // Subsector 1
		LeftChild:        0x8002, // Subsector 2
	}
	_ = binary.Write(buf, binary.LittleEndian, node)

	nodes, err := ParseNodes(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseNodes failed: %v", err)
	}
	if len(nodes) != 1 || nodes[0].PartitionX != 100 || nodes[0].RightChild != 0x8001 {
		t.Errorf("node mismatch: %+v", nodes)
	}

	_, err = ParseNodes([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error on invalid node data length")
	}
}

func TestParseThings(t *testing.T) {
	buf := new(bytes.Buffer)
	th := Thing{
		X:     128,
		Y:     -256,
		Angle: 90,
		Type:  ThingPlayer1Start,
		Flags: 7,
	}
	_ = binary.Write(buf, binary.LittleEndian, th)

	things, err := ParseThings(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseThings failed: %v", err)
	}
	if len(things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(things))
	}
	if things[0].X != 128 || things[0].Y != -256 || things[0].Angle != 90 || things[0].Type != ThingPlayer1Start {
		t.Errorf("thing 0 mismatch: %+v", things[0])
	}

	// Invalid byte length
	_, err = ParseThings([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error on invalid thing data length")
	}
}

func TestLoadMapFromWAD(t *testing.T) {
	wadPath := "../../freedoom2.wad"
	if _, err := os.Stat(wadPath); os.IsNotExist(err) {
		wadPath = "freedoom2.wad"
		if _, err := os.Stat(wadPath); os.IsNotExist(err) {
			t.Skip("freedoom2.wad not found in test path, skipping")
		}
	}

	w, err := Open(wadPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer w.Close()

	mapData, err := w.LoadMap("MAP01")
	if err != nil {
		t.Fatalf("LoadMap(MAP01) failed: %v", err)
	}

	if mapData.Name != "MAP01" {
		t.Errorf("expected map name 'MAP01', got %q", mapData.Name)
	}
	if len(mapData.Vertexes) == 0 {
		t.Error("expected non-empty vertexes for MAP01")
	}
	if len(mapData.Linedefs) == 0 {
		t.Error("expected non-empty linedefs for MAP01")
	}
	if len(mapData.Sidedefs) == 0 {
		t.Error("expected non-empty sidedefs for MAP01")
	}
	if len(mapData.Sectors) == 0 {
		t.Error("expected non-empty sectors for MAP01")
	}
	if len(mapData.Segs) == 0 {
		t.Error("expected non-empty segs for MAP01")
	}
	if len(mapData.Subsectors) == 0 {
		t.Error("expected non-empty subsectors for MAP01")
	}
	if len(mapData.Nodes) == 0 {
		t.Error("expected non-empty nodes for MAP01")
	}
	if len(mapData.Things) == 0 {
		t.Error("expected non-empty things for MAP01")
	}
	if mapData.Textures == nil {
		t.Error("expected non-nil TextureManager for MAP01")
	}

	p1, ok := mapData.Player1Start()
	if !ok {
		t.Fatal("expected Player 1 start in MAP01")
	}
	if p1.Type != ThingPlayer1Start {
		t.Errorf("expected p1 type 1, got %d", p1.Type)
	}

	// Verify finding player's sector via BSP
	sec, ok := mapData.SectorAt(float64(p1.X), float64(p1.Y))
	if !ok || sec == nil {
		t.Errorf("failed to find sector at player start (%d, %d)", p1.X, p1.Y)
	}

	minX, maxX, minY, maxY := mapData.Bounds()
	if minX >= maxX || minY >= maxY {
		t.Errorf("invalid map bounds: (%d, %d) to (%d, %d)", minX, minY, maxX, maxY)
	}
}





