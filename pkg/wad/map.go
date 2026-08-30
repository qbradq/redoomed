package wad

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Linedef flag constants from Doom engine specification.
const (
	LinedefBlocking      uint16 = 0x0001 // Blocks players and monsters
	LinedefBlockMonsters uint16 = 0x0002 // Blocks monsters only
	LinedefTwoSided      uint16 = 0x0004 // 2-sided line (passable or window/step)
	LinedefDontPegTop    uint16 = 0x0008 // Upper texture unpegged
	LinedefDontPegBottom uint16 = 0x0010 // Lower texture unpegged
	LinedefSecret        uint16 = 0x0020 // Secret (drawn as 1-sided on automap)
	LinedefSoundBlock    uint16 = 0x0040 // Blocks sound propagation
	LinedefDontDraw      uint16 = 0x0080 // Never drawn on automap
	LinedefMapped        uint16 = 0x0100 // Already revealed / mapped
)

// Thing type constants.
const (
	ThingPlayer1Start int16 = 1
	ThingPlayer2Start int16 = 2
	ThingPlayer3Start int16 = 3
	ThingPlayer4Start int16 = 4
	ThingDeathmatch   int16 = 11
)

// Vertex represents a 2D map coordinate from the VERTEXES lump.
type Vertex struct {
	X int16
	Y int16
}

// Linedef represents a line definition from the LINEDEFS lump.
type Linedef struct {
	V1        uint16 // Start vertex index
	V2        uint16 // End vertex index
	Flags     uint16 // Flags (blocking, two-sided, secret, dontdraw, etc.)
	Special   int16  // Line special type / trigger action
	Tag       int16  // Sector tag
	RightSide uint16 // Right / front sidedef index (0xFFFF if none)
	LeftSide  uint16 // Left / back sidedef index (0xFFFF if none)
}

// Thing represents an entity placement from the THINGS lump.
type Thing struct {
	X     int16
	Y     int16
	Angle int16 // Degrees: 0=East, 90=North, 180=West, 270=South
	Type  int16 // Entity type (e.g. 1 for Player 1 start)
	Flags int16 // Spawn / skill flags
}

// MapData contains all parsed geometry and entity lumps for a Doom map.
type MapData struct {
	Name     string
	Vertexes []Vertex
	Linedefs []Linedef
	Things   []Thing
}

// ParseVertexes parses raw bytes from a VERTEXES lump (4 bytes per vertex: int16 X, int16 Y).
func ParseVertexes(data []byte) ([]Vertex, error) {
	if len(data)%4 != 0 {
		return nil, errors.New("invalid VERTEXES lump size: must be a multiple of 4 bytes")
	}

	num := len(data) / 4
	res := make([]Vertex, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var v Vertex
		if err := binary.Read(reader, binary.LittleEndian, &v); err != nil {
			return nil, fmt.Errorf("failed to read vertex %d: %w", i, err)
		}
		res[i] = v
	}

	return res, nil
}

// ParseLinedefs parses raw bytes from a LINEDEFS lump (14 bytes per linedef).
func ParseLinedefs(data []byte) ([]Linedef, error) {
	if len(data)%14 != 0 {
		return nil, errors.New("invalid LINEDEFS lump size: must be a multiple of 14 bytes")
	}

	num := len(data) / 14
	res := make([]Linedef, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var l Linedef
		if err := binary.Read(reader, binary.LittleEndian, &l); err != nil {
			return nil, fmt.Errorf("failed to read linedef %d: %w", i, err)
		}
		res[i] = l
	}

	return res, nil
}

// ParseThings parses raw bytes from a THINGS lump (10 bytes per thing).
func ParseThings(data []byte) ([]Thing, error) {
	if len(data)%10 != 0 {
		return nil, errors.New("invalid THINGS lump size: must be a multiple of 10 bytes")
	}

	num := len(data) / 10
	res := make([]Thing, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var t Thing
		if err := binary.Read(reader, binary.LittleEndian, &t); err != nil {
			return nil, fmt.Errorf("failed to read thing %d: %w", i, err)
		}
		res[i] = t
	}

	return res, nil
}

// GetMapLump finds and retrieves a map-specific lump (e.g. THINGS, LINEDEFS, VERTEXES)
// associated with the given map marker (e.g. "MAP01" or "E1M1").
func (w *WAD) GetMapLump(mapName, lumpName string) ([]byte, error) {
	mapIdx := w.GetLumpIndex(mapName)
	if mapIdx < 0 {
		return nil, fmt.Errorf("%w: map %s", ErrLumpNotFound, mapName)
	}

	targetName := strings.ToUpper(lumpName)
	// Map lumps immediately follow the map header entry
	maxScan := mapIdx + 15
	if maxScan > len(w.lumps) {
		maxScan = len(w.lumps)
	}

	for i := mapIdx + 1; i < maxScan; i++ {
		if w.lumps[i].Name == targetName {
			return w.GetLumpByIndex(i)
		}
		// If we encounter another map header, stop scanning
		if isMapHeader(w.lumps[i].Name) {
			break
		}
	}

	return nil, fmt.Errorf("%w: lump %s in map %s", ErrLumpNotFound, lumpName, mapName)
}

// isMapHeader checks if a lump name matches standard Doom map name patterns (ExMy or MAPxy).
func isMapHeader(name string) bool {
	upper := strings.ToUpper(name)
	if len(upper) == 4 && upper[0] == 'E' && upper[2] == 'M' &&
		upper[1] >= '0' && upper[1] <= '9' && upper[3] >= '0' && upper[3] <= '9' {
		return true
	}
	if len(upper) == 5 && strings.HasPrefix(upper, "MAP") &&
		upper[3] >= '0' && upper[3] <= '9' && upper[4] >= '0' && upper[4] <= '9' {
		return true
	}
	return false
}

// LoadMap parses all primary geometry and entity lumps (THINGS, LINEDEFS, VERTEXES) for the given map.
func (w *WAD) LoadMap(mapName string) (*MapData, error) {
	upperMap := strings.ToUpper(mapName)

	vertexData, err := w.GetMapLump(upperMap, "VERTEXES")
	if err != nil {
		return nil, fmt.Errorf("failed to load VERTEXES for %s: %w", upperMap, err)
	}
	vertexes, err := ParseVertexes(vertexData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VERTEXES for %s: %w", upperMap, err)
	}

	linedefData, err := w.GetMapLump(upperMap, "LINEDEFS")
	if err != nil {
		return nil, fmt.Errorf("failed to load LINEDEFS for %s: %w", upperMap, err)
	}
	linedefs, err := ParseLinedefs(linedefData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LINEDEFS for %s: %w", upperMap, err)
	}

	thingData, err := w.GetMapLump(upperMap, "THINGS")
	if err != nil {
		return nil, fmt.Errorf("failed to load THINGS for %s: %w", upperMap, err)
	}
	things, err := ParseThings(thingData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse THINGS for %s: %w", upperMap, err)
	}

	return &MapData{
		Name:     upperMap,
		Vertexes: vertexes,
		Linedefs: linedefs,
		Things:   things,
	}, nil
}

// Bounds calculates the 2D bounding box (minX, maxX, minY, maxY) of all vertexes in the map.
func (m *MapData) Bounds() (minX, maxX, minY, maxY int16) {
	if len(m.Vertexes) == 0 {
		return 0, 0, 0, 0
	}

	minX = m.Vertexes[0].X
	maxX = m.Vertexes[0].X
	minY = m.Vertexes[0].Y
	maxY = m.Vertexes[0].Y

	for _, v := range m.Vertexes[1:] {
		if v.X < minX {
			minX = v.X
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}

	return minX, maxX, minY, maxY
}

// Player1Start finds the Player 1 start thing (Type == 1).
func (m *MapData) Player1Start() (Thing, bool) {
	for _, t := range m.Things {
		if t.Type == ThingPlayer1Start {
			return t, true
		}
	}
	return Thing{}, false
}
