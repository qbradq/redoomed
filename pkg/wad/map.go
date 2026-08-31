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

// Sidedef represents wall texture bindings from the SIDEDEFS lump.
type Sidedef struct {
	XOffset       int16
	YOffset       int16
	UpperTexture  string
	LowerTexture  string
	MiddleTexture string
	Sector        uint16
}

type rawSidedef struct {
	XOffset       int16
	YOffset       int16
	UpperTexture  [8]byte
	LowerTexture  [8]byte
	MiddleTexture [8]byte
	Sector        uint16
}

// Sector represents a 2D floor/ceiling region from the SECTORS lump.
type Sector struct {
	FloorHeight   int16
	CeilingHeight int16
	FloorPic      string
	CeilingPic    string
	LightLevel    int16
	Special       int16
	Tag           int16
}

type rawSector struct {
	FloorHeight   int16
	CeilingHeight int16
	FloorPic      [8]byte
	CeilingPic    [8]byte
	LightLevel    int16
	Special       int16
	Tag           int16
}

// Seg represents a line segment within a subsector from the SEGS lump.
type Seg struct {
	V1        uint16 // Start vertex index
	V2        uint16 // End vertex index
	Angle     int16  // Angle in BAM (Binary Angle Measurement)
	Linedef   uint16 // Linedef index
	Direction uint16 // 0: same as linedef (RightSide), 1: opposite (LeftSide)
	Offset    int16  // Distance along linedef to start of seg
}

// Subsector represents a convex sub-polygon in the BSP tree from the SSECTORS lump.
type Subsector struct {
	NumSegs  uint16
	FirstSeg uint16
}

// Node represents a partition node in the BSP tree from the NODES lump.
type Node struct {
	PartitionX       int16
	PartitionY       int16
	ChangeX          int16
	ChangeY          int16
	RightBoundingBox [4]int16 // Top, Bottom, Left, Right (maxY, minY, minX, maxX)
	LeftBoundingBox  [4]int16 // Top, Bottom, Left, Right (maxY, minY, minX, maxX)
	RightChild       uint16   // Subsector index if (RightChild & 0x8000 != 0); else Node index
	LeftChild        uint16   // Subsector index if (LeftChild & 0x8000 != 0); else Node index
}

// Thing represents an entity placement from the THINGS lump.
type Thing struct {
	X     int16
	Y     int16
	Angle int16 // Degrees: 0=East, 90=North, 180=West, 270=South
	Type  int16 // Entity type (e.g. 1 for Player 1 start)
	Flags int16 // Spawn / skill flags
}

// MapData contains all parsed geometry, BSP, entity lumps, and textures for a Doom map.
type MapData struct {
	Name       string
	Vertexes   []Vertex
	Linedefs   []Linedef
	Sidedefs   []Sidedef
	Sectors    []Sector
	Segs       []Seg
	Subsectors []Subsector
	Nodes      []Node
	Things     []Thing
	Textures   *TextureManager
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

// ParseSidedefs parses raw bytes from a SIDEDEFS lump (30 bytes per sidedef).
func ParseSidedefs(data []byte) ([]Sidedef, error) {
	if len(data)%30 != 0 {
		return nil, errors.New("invalid SIDEDEFS lump size: must be a multiple of 30 bytes")
	}

	num := len(data) / 30
	res := make([]Sidedef, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var raw rawSidedef
		if err := binary.Read(reader, binary.LittleEndian, &raw); err != nil {
			return nil, fmt.Errorf("failed to read sidedef %d: %w", i, err)
		}

		res[i] = Sidedef{
			XOffset:       raw.XOffset,
			YOffset:       raw.YOffset,
			UpperTexture:  strings.ToUpper(string(bytes.TrimRight(raw.UpperTexture[:], "\x00"))),
			LowerTexture:  strings.ToUpper(string(bytes.TrimRight(raw.LowerTexture[:], "\x00"))),
			MiddleTexture: strings.ToUpper(string(bytes.TrimRight(raw.MiddleTexture[:], "\x00"))),
			Sector:        raw.Sector,
		}
	}

	return res, nil
}

// ParseSectors parses raw bytes from a SECTORS lump (26 bytes per sector).
func ParseSectors(data []byte) ([]Sector, error) {
	if len(data)%26 != 0 {
		return nil, errors.New("invalid SECTORS lump size: must be a multiple of 26 bytes")
	}

	num := len(data) / 26
	res := make([]Sector, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var raw rawSector
		if err := binary.Read(reader, binary.LittleEndian, &raw); err != nil {
			return nil, fmt.Errorf("failed to read sector %d: %w", i, err)
		}

		res[i] = Sector{
			FloorHeight:   raw.FloorHeight,
			CeilingHeight: raw.CeilingHeight,
			FloorPic:      strings.ToUpper(string(bytes.TrimRight(raw.FloorPic[:], "\x00"))),
			CeilingPic:    strings.ToUpper(string(bytes.TrimRight(raw.CeilingPic[:], "\x00"))),
			LightLevel:    raw.LightLevel,
			Special:       raw.Special,
			Tag:           raw.Tag,
		}
	}

	return res, nil
}

// ParseSegs parses raw bytes from a SEGS lump (12 bytes per seg).
func ParseSegs(data []byte) ([]Seg, error) {
	if len(data)%12 != 0 {
		return nil, errors.New("invalid SEGS lump size: must be a multiple of 12 bytes")
	}

	num := len(data) / 12
	res := make([]Seg, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var s Seg
		if err := binary.Read(reader, binary.LittleEndian, &s); err != nil {
			return nil, fmt.Errorf("failed to read seg %d: %w", i, err)
		}
		res[i] = s
	}

	return res, nil
}

// ParseSubsectors parses raw bytes from a SSECTORS lump (4 bytes per subsector).
func ParseSubsectors(data []byte) ([]Subsector, error) {
	if len(data)%4 != 0 {
		return nil, errors.New("invalid SSECTORS lump size: must be a multiple of 4 bytes")
	}

	num := len(data) / 4
	res := make([]Subsector, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var ss Subsector
		if err := binary.Read(reader, binary.LittleEndian, &ss); err != nil {
			return nil, fmt.Errorf("failed to read subsector %d: %w", i, err)
		}
		res[i] = ss
	}

	return res, nil
}

// ParseNodes parses raw bytes from a NODES lump (28 bytes per node).
func ParseNodes(data []byte) ([]Node, error) {
	if len(data)%28 != 0 {
		return nil, errors.New("invalid NODES lump size: must be a multiple of 28 bytes")
	}

	num := len(data) / 28
	res := make([]Node, num)
	reader := bytes.NewReader(data)

	for i := 0; i < num; i++ {
		var n Node
		if err := binary.Read(reader, binary.LittleEndian, &n); err != nil {
			return nil, fmt.Errorf("failed to read node %d: %w", i, err)
		}
		res[i] = n
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

// GetMapLump finds and retrieves a map-specific lump (e.g. THINGS, LINEDEFS, VERTEXES, SIDEDEFS, SECTORS, SEGS, SSECTORS, NODES)
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

// LoadMap parses all geometry, BSP, sidedefs, sectors, entity lumps, and textures for the given map.
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

	sidedefData, err := w.GetMapLump(upperMap, "SIDEDEFS")
	if err != nil {
		return nil, fmt.Errorf("failed to load SIDEDEFS for %s: %w", upperMap, err)
	}
	sidedefs, err := ParseSidedefs(sidedefData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SIDEDEFS for %s: %w", upperMap, err)
	}

	sectorData, err := w.GetMapLump(upperMap, "SECTORS")
	if err != nil {
		return nil, fmt.Errorf("failed to load SECTORS for %s: %w", upperMap, err)
	}
	sectors, err := ParseSectors(sectorData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SECTORS for %s: %w", upperMap, err)
	}

	segData, err := w.GetMapLump(upperMap, "SEGS")
	if err != nil {
		return nil, fmt.Errorf("failed to load SEGS for %s: %w", upperMap, err)
	}
	segs, err := ParseSegs(segData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SEGS for %s: %w", upperMap, err)
	}

	ssectorData, err := w.GetMapLump(upperMap, "SSECTORS")
	if err != nil {
		return nil, fmt.Errorf("failed to load SSECTORS for %s: %w", upperMap, err)
	}
	subsectors, err := ParseSubsectors(ssectorData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSECTORS for %s: %w", upperMap, err)
	}

	nodeData, err := w.GetMapLump(upperMap, "NODES")
	if err != nil {
		return nil, fmt.Errorf("failed to load NODES for %s: %w", upperMap, err)
	}
	nodes, err := ParseNodes(nodeData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NODES for %s: %w", upperMap, err)
	}

	thingData, err := w.GetMapLump(upperMap, "THINGS")
	if err != nil {
		return nil, fmt.Errorf("failed to load THINGS for %s: %w", upperMap, err)
	}
	things, err := ParseThings(thingData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse THINGS for %s: %w", upperMap, err)
	}

	// Initialize texture manager and preload all textures & flats referenced by the map
	texMgr, err := NewTextureManager(w)
	if err == nil {
		mapData := &MapData{
			Name:       upperMap,
			Vertexes:   vertexes,
			Linedefs:   linedefs,
			Sidedefs:   sidedefs,
			Sectors:    sectors,
			Segs:       segs,
			Subsectors: subsectors,
			Nodes:      nodes,
			Things:     things,
			Textures:   texMgr,
		}
		texMgr.PreloadMap(mapData)
		return mapData, nil
	}

	return &MapData{
		Name:       upperMap,
		Vertexes:   vertexes,
		Linedefs:   linedefs,
		Sidedefs:   sidedefs,
		Sectors:    sectors,
		Segs:       segs,
		Subsectors: subsectors,
		Nodes:      nodes,
		Things:     things,
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

// FindSubsector finds the subsector index containing 2D point (x, y) by traversing the BSP tree.
func (m *MapData) FindSubsector(x, y float64) int {
	if len(m.Nodes) == 0 {
		return 0
	}

	nodeIdx := len(m.Nodes) - 1
	for {
		if nodeIdx < 0 || nodeIdx >= len(m.Nodes) {
			return 0
		}
		node := &m.Nodes[nodeIdx]

		// Doom BSP partition side test:
		// side = 0 (right/front) if (y - py) * dx < (x - px) * dy
		dx := x - float64(node.PartitionX)
		dy := y - float64(node.PartitionY)
		left := float64(node.ChangeY) * dx
		right := dy * float64(node.ChangeX)

		var child uint16
		if right < left {
			child = node.RightChild
		} else {
			child = node.LeftChild
		}

		if child&0x8000 != 0 {
			// Subsector leaf node
			return int(child & 0x7FFF)
		}
		nodeIdx = int(child)
	}
}

// SectorAt returns the sector containing the point (x, y) if found.
func (m *MapData) SectorAt(x, y float64) (*Sector, bool) {
	secIdx, ok := m.SectorIndexAt(x, y)
	if !ok {
		return nil, false
	}
	return &m.Sectors[secIdx], true
}

// SectorIndexAt returns the sector index containing the point (x, y) if found.
func (m *MapData) SectorIndexAt(x, y float64) (int, bool) {
	if len(m.Subsectors) == 0 || len(m.Segs) == 0 {
		return -1, false
	}
	minX, maxX, minY, maxY := m.Bounds()
	if x < float64(minX) || x > float64(maxX) || y < float64(minY) || y > float64(maxY) {
		return -1, false
	}

	ssIdx := m.FindSubsector(x, y)
	if ssIdx < 0 || ssIdx >= len(m.Subsectors) {
		return -1, false
	}
	ss := &m.Subsectors[ssIdx]
	if ss.NumSegs == 0 {
		return -1, false
	}

	firstSeg := int(ss.FirstSeg)
	if firstSeg < 0 || firstSeg >= len(m.Segs) {
		return -1, false
	}
	seg := &m.Segs[firstSeg]
	if int(seg.Linedef) >= len(m.Linedefs) {
		return -1, false
	}
	ld := &m.Linedefs[seg.Linedef]

	var sidedefIdx uint16
	if seg.Direction == 0 {
		sidedefIdx = ld.RightSide
	} else {
		sidedefIdx = ld.LeftSide
	}
	if sidedefIdx == 0xFFFF || int(sidedefIdx) >= len(m.Sidedefs) {
		return -1, false
	}
	secIdx := int(m.Sidedefs[sidedefIdx].Sector)
	if secIdx >= len(m.Sectors) {
		return -1, false
	}
	return secIdx, true
}
