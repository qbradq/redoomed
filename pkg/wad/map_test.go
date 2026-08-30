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
	if len(mapData.Things) == 0 {
		t.Error("expected non-empty things for MAP01")
	}

	p1, ok := mapData.Player1Start()
	if !ok {
		t.Fatal("expected Player 1 start in MAP01")
	}
	if p1.Type != ThingPlayer1Start {
		t.Errorf("expected p1 type 1, got %d", p1.Type)
	}

	minX, maxX, minY, maxY := mapData.Bounds()
	if minX >= maxX || minY >= maxY {
		t.Errorf("invalid map bounds: (%d, %d) to (%d, %d)", minX, minY, maxX, maxY)
	}
}
