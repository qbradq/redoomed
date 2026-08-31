package mode

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/qbradq/redoomed/pkg/data"
	"github.com/qbradq/redoomed/pkg/font"
	"github.com/qbradq/redoomed/pkg/gfx"
	"github.com/qbradq/redoomed/pkg/wad"
)

var editorIconsImage *ebiten.Image

// loadEditorIcons loads the 128x128 icons atlas from pkg/data/gfx/editor_icons.png.
func loadEditorIcons() (*ebiten.Image, error) {
	if editorIconsImage != nil {
		return editorIconsImage, nil
	}

	f, err := data.FS.Open("gfx/editor_icons.png")
	if err != nil {
		return nil, fmt.Errorf("failed to open editor_icons.png: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode editor_icons.png: %w", err)
	}

	editorIconsImage = ebiten.NewImageFromImage(img)
	return editorIconsImage, nil
}

const (
	// EditorBufferWidth is the logical width of the editor mode (640).
	EditorBufferWidth = 640
	// EditorBufferHeight is the logical height of the editor mode (400).
	EditorBufferHeight = 400

	// EditingWindowWidth is the width of the editing grid area (400).
	EditingWindowWidth = 400
	// EditingWindowHeight is the height of the editing grid area (392).
	EditingWindowHeight = 392

	// UserPanelX is the horizontal starting coordinate of the user panel (400).
	UserPanelX = 400
	// UserPanelY is the vertical starting coordinate of the user panel (0).
	UserPanelY = 0
	// UserPanelWidth is the width of the user panel (240).
	UserPanelWidth = 240
	// UserPanelHeight is the height of the user panel (392).
	UserPanelHeight = 392

	// StatusBarX is the horizontal starting coordinate of the status bar (0).
	StatusBarX = 0
	// StatusBarY is the vertical starting coordinate of the status bar (392).
	StatusBarY = 392
	// StatusBarWidth is the width of the status bar (640).
	StatusBarWidth = 640
	// StatusBarHeight is the height of the status bar (8).
	StatusBarHeight = 8

	// NumIconButtons is the number of 16x16 icon buttons in the top of the user panel (15).
	NumIconButtons = 15
	// IconButtonSize is the dimension of each square icon button in pixels.
	IconButtonSize = 16

	// MinGridSize is the minimum allowed grid size (1 unit).
	MinGridSize = 1
	// MaxGridSize is the maximum allowed grid size (256 units).
	MaxGridSize = 256
	// DefaultGridSize is the initial grid size (64 units).
	DefaultGridSize = 64

	// MinZoomLevel is the minimum zoom level index (0 = 64 map units per pixel).
	MinZoomLevel = 0
	// MaxZoomLevel is the maximum zoom level index (11 = 32 pixels per map unit).
	MaxZoomLevel = 11
	// DefaultZoomLevel is the initial zoom level index (3 = 8 map units per pixel).
	DefaultZoomLevel = 3
)

// ZoomScales maps zoom levels 0..11 to pixels per map unit.
// Level 0 is 1/64 (64 units per pixel), Level 6 is 1.0 (1 unit per pixel), Level 11 is 32.0 (32 pixels per unit).
var ZoomScales = [12]float64{
	1.0 / 64.0, // Level 0: 64 map units per pixel
	1.0 / 32.0, // Level 1: 32 map units per pixel
	1.0 / 16.0, // Level 2: 16 map units per pixel
	1.0 / 8.0,  // Level 3: 8 map units per pixel
	1.0 / 4.0,  // Level 4: 4 map units per pixel
	1.0 / 2.0,  // Level 5: 2 map units per pixel
	1.0,        // Level 6: 1 map unit per pixel
	2.0,        // Level 7: 2 pixels per map unit
	4.0,        // Level 8: 4 pixels per map unit
	8.0,        // Level 9: 8 pixels per map unit
	16.0,       // Level 10: 16 pixels per map unit
	32.0,       // Level 11: 32 pixels per map unit
}

// EditMode represents the active editing sub-mode.
type EditMode int

const (
	// EditModeVertex is the vertex editing mode (on by default).
	EditModeVertex EditMode = iota
	// EditModeLine is the linedef editing mode.
	EditModeLine
	// EditModeSector is the sector editing mode.
	EditModeSector
	// EditModeThing is the thing/entity editing mode.
	EditModeThing
)

// IconButton represents a single 16x16 icon button slot in the user panel.
type IconButton struct {
	Index    int
	Icon     int
	Visible  bool
	Enabled  bool
	IsToggle bool
	Active   bool
	Hovered  bool
	Tooltip  string
	OnClick  func(*EditorMode)
}

// EditorMode represents the 640x400 map editor mode composited to the screen.
type EditorMode struct {
	buffer          *ebiten.Image
	wadFile         *wad.WAD
	mapData         *wad.MapData
	mapDataProvider func() *wad.MapData
	lastMapData     *wad.MapData
	hasCenteredCam  bool
	font            *font.ConsoleFont
	iconsImage      *ebiten.Image

	gridSize  int
	zoomLevel int
	editMode  EditMode

	// Selection sets per mode
	selectedVertexes map[int]bool
	selectedLines    map[int]bool
	selectedSectors  map[int]bool
	selectedThings   map[int]bool

	// Hovered target indices (-1 if none)
	hoveredVertex  int
	hoveredLinedef int
	hoveredSector  int
	hoveredThing   int

	// Camera world position (map units) centered in the editing window
	camX float64
	camY float64

	// Dragging state for panning
	isPanning  bool
	panStartX  int
	panStartY  int
	origCamX   float64
	origCamY   float64

	// Buttons
	buttons [NumIconButtons]IconButton

	// Mode switching callbacks
	onToggle        func()
	onToggleConsole func()
}

// NewEditorMode creates and initializes a new EditorMode instance.
func NewEditorMode(f *font.ConsoleFont, w *wad.WAD) *EditorMode {
	icons, _ := loadEditorIcons()

	ed := &EditorMode{
		buffer:           ebiten.NewImage(EditorBufferWidth, EditorBufferHeight),
		wadFile:          w,
		font:             f,
		iconsImage:       icons,
		gridSize:         DefaultGridSize,
		zoomLevel:        DefaultZoomLevel,
		editMode:         EditModeVertex,
		selectedVertexes: make(map[int]bool),
		selectedLines:    make(map[int]bool),
		selectedSectors:  make(map[int]bool),
		selectedThings:   make(map[int]bool),
		hoveredVertex:    -1,
		hoveredLinedef:   -1,
		hoveredSector:    -1,
		hoveredThing:     -1,
		camX:             0,
		camY:             0,
	}

	ed.initButtons()
	return ed
}

// initButtons populates the 15 user panel button slots.
func (e *EditorMode) initButtons() {
	// Button 0: New map button. Icon 0. Disabled.
	e.buttons[0] = IconButton{
		Index:   0,
		Icon:    0,
		Visible: true,
		Enabled: false,
		Tooltip: "New Map",
	}

	// Button 1: Open map button. Icon 1. Disabled.
	e.buttons[1] = IconButton{
		Index:   1,
		Icon:    1,
		Visible: true,
		Enabled: false,
		Tooltip: "Open Map",
	}

	// Button 2: Save map button. Icon 2. Disabled.
	e.buttons[2] = IconButton{
		Index:   2,
		Icon:    2,
		Visible: true,
		Enabled: false,
		Tooltip: "Save Map",
	}

	// The next 4 buttons are a group of toggle buttons (Radio group: exactly one active at all times)
	// Button 3: Vertex mode button. Icon 3. Enabled, on by default.
	e.buttons[3] = IconButton{
		Index:    3,
		Icon:     3,
		Visible:  true,
		Enabled:  true,
		IsToggle: true,
		Active:   true,
		Tooltip:  "Vertex Mode",
	}

	// Button 4: Line mode button. Icon 4. Enabled, off by default.
	e.buttons[4] = IconButton{
		Index:    4,
		Icon:     4,
		Visible:  true,
		Enabled:  true,
		IsToggle: true,
		Active:   false,
		Tooltip:  "Line Mode",
	}

	// Button 5: Sector mode button. Icon 5. Enabled, off by default.
	e.buttons[5] = IconButton{
		Index:    5,
		Icon:     5,
		Visible:  true,
		Enabled:  true,
		IsToggle: true,
		Active:   false,
		Tooltip:  "Sector Mode",
	}

	// Button 6: Thing mode button. Icon 6. Enabled, off by default.
	e.buttons[6] = IconButton{
		Index:    6,
		Icon:     6,
		Visible:  true,
		Enabled:  true,
		IsToggle: true,
		Active:   false,
		Tooltip:  "Thing Mode",
	}

	// Button 7: Zoom In button. Icon 7. Enabled. Zooms in.
	e.buttons[7] = IconButton{
		Index:   7,
		Icon:    7,
		Visible: true,
		Enabled: true,
		Tooltip: "Zoom In",
		OnClick: func(ed *EditorMode) {
			ed.IncreaseZoom()
		},
	}

	// Button 8: Zoom Out button. Icon 8. Enabled. Zooms out.
	e.buttons[8] = IconButton{
		Index:   8,
		Icon:    8,
		Visible: true,
		Enabled: true,
		Tooltip: "Zoom Out",
		OnClick: func(ed *EditorMode) {
			ed.DecreaseZoom()
		},
	}

	// Button 9: Increase grid size. Icon 9. Enabled. Increases grid size.
	e.buttons[9] = IconButton{
		Index:   9,
		Icon:    9,
		Visible: true,
		Enabled: true,
		Tooltip: "Increase Grid Size",
		OnClick: func(ed *EditorMode) {
			ed.IncreaseGridSize()
		},
	}

	// Button 10: Decrease grid size. Icon 10. Enabled. Decreases grid size.
	e.buttons[10] = IconButton{
		Index:   10,
		Icon:    10,
		Visible: true,
		Enabled: true,
		Tooltip: "Decrease Grid Size",
		OnClick: func(ed *EditorMode) {
			ed.DecreaseGridSize()
		},
	}

	// Unused button slots (11..14) are not drawn
	for i := 11; i < NumIconButtons; i++ {
		e.buttons[i] = IconButton{
			Index:   i,
			Icon:    -1,
			Visible: false,
		}
	}
}

// EditMode returns the currently active editing mode.
func (e *EditorMode) EditMode() EditMode {
	return e.editMode
}

// SetEditMode updates the active editing mode and synchronizes the toggle buttons.
func (e *EditorMode) SetEditMode(m EditMode) {
	e.editMode = m
	for i := 3; i <= 6; i++ {
		e.buttons[i].Active = (i-3 == int(m))
	}
}

// Button returns the icon button state at the specified slot index (0..14).
func (e *EditorMode) Button(index int) IconButton {
	if index < 0 || index >= NumIconButtons {
		return IconButton{}
	}
	return e.buttons[index]
}

// Buttons returns all 15 icon button slot definitions.
func (e *EditorMode) Buttons() [NumIconButtons]IconButton {
	return e.buttons
}

// HoveredVertex returns the index of the vertex currently hovered in vertex mode (-1 if none).
func (e *EditorMode) HoveredVertex() int {
	return e.hoveredVertex
}

// HoveredLinedef returns the index of the linedef currently hovered in line mode (-1 if none).
func (e *EditorMode) HoveredLinedef() int {
	return e.hoveredLinedef
}

// HoveredSector returns the index of the sector currently hovered in sector mode (-1 if none).
func (e *EditorMode) HoveredSector() int {
	return e.hoveredSector
}

// HoveredThing returns the index of the thing currently hovered in thing mode (-1 if none).
func (e *EditorMode) HoveredThing() int {
	return e.hoveredThing
}

// SelectedVertexes returns a slice of currently selected vertex indices.
func (e *EditorMode) SelectedVertexes() []int {
	var res []int
	for i := range e.selectedVertexes {
		res = append(res, i)
	}
	return res
}

// SelectedLines returns a slice of currently selected linedef indices.
func (e *EditorMode) SelectedLines() []int {
	var res []int
	for i := range e.selectedLines {
		res = append(res, i)
	}
	return res
}

// SelectedSectors returns a slice of currently selected sector indices.
func (e *EditorMode) SelectedSectors() []int {
	var res []int
	for i := range e.selectedSectors {
		res = append(res, i)
	}
	return res
}

// SelectedThings returns a slice of currently selected thing indices.
func (e *EditorMode) SelectedThings() []int {
	var res []int
	for i := range e.selectedThings {
		res = append(res, i)
	}
	return res
}

// IsVertexSelected reports whether the given vertex index is selected.
func (e *EditorMode) IsVertexSelected(i int) bool {
	return e.selectedVertexes[i]
}

// IsLineSelected reports whether the given linedef index is selected.
func (e *EditorMode) IsLineSelected(i int) bool {
	return e.selectedLines[i]
}

// IsSectorSelected reports whether the given sector index is selected.
func (e *EditorMode) IsSectorSelected(i int) bool {
	return e.selectedSectors[i]
}

// IsThingSelected reports whether the given thing index is selected.
func (e *EditorMode) IsThingSelected(i int) bool {
	return e.selectedThings[i]
}

// SelectVertex selects or toggles a vertex in the selection set.
func (e *EditorMode) SelectVertex(i int, multi bool) {
	if multi {
		if e.selectedVertexes[i] {
			delete(e.selectedVertexes, i)
		} else {
			e.selectedVertexes[i] = true
		}
	} else {
		e.selectedVertexes = map[int]bool{i: true}
	}
}

// SelectLine selects or toggles a linedef in the selection set.
func (e *EditorMode) SelectLine(i int, multi bool) {
	if multi {
		if e.selectedLines[i] {
			delete(e.selectedLines, i)
		} else {
			e.selectedLines[i] = true
		}
	} else {
		e.selectedLines = map[int]bool{i: true}
	}
}

// SelectSector selects or toggles a sector in the selection set.
func (e *EditorMode) SelectSector(i int, multi bool) {
	if multi {
		if e.selectedSectors[i] {
			delete(e.selectedSectors, i)
		} else {
			e.selectedSectors[i] = true
		}
	} else {
		e.selectedSectors = map[int]bool{i: true}
	}
}

// SelectThing selects or toggles a thing in the selection set.
func (e *EditorMode) SelectThing(i int, multi bool) {
	if multi {
		if e.selectedThings[i] {
			delete(e.selectedThings, i)
		} else {
			e.selectedThings[i] = true
		}
	} else {
		e.selectedThings = map[int]bool{i: true}
	}
}

// ClearSelection clears the selection set for the current editing mode.
func (e *EditorMode) ClearSelection() {
	switch e.editMode {
	case EditModeVertex:
		e.selectedVertexes = make(map[int]bool)
	case EditModeLine:
		e.selectedLines = make(map[int]bool)
	case EditModeSector:
		e.selectedSectors = make(map[int]bool)
	case EditModeThing:
		e.selectedThings = make(map[int]bool)
	}
}

// ActiveSelection returns a slice of currently selected element indices for the active mode.
func (e *EditorMode) ActiveSelection() []int {
	switch e.editMode {
	case EditModeVertex:
		return e.SelectedVertexes()
	case EditModeLine:
		return e.SelectedLines()
	case EditModeSector:
		return e.SelectedSectors()
	case EditModeThing:
		return e.SelectedThings()
	default:
		return nil
	}
}

// SelectionCount returns the number of selected items in the active editing mode.
func (e *EditorMode) SelectionCount() int {
	return len(e.ActiveSelection())
}

// LookupThingName returns a human-readable name for a given Doom Thing type.
func LookupThingName(thingType int16) string {
	if def, ok := wad.LookupItemDef(thingType); ok && def.Name != "" {
		return def.Name
	}
	switch thingType {
	case 1:
		return "Player 1 Start"
	case 2:
		return "Player 2 Start"
	case 3:
		return "Player 3 Start"
	case 4:
		return "Player 4 Start"
	case 11:
		return "Deathmatch Start"
	case 14:
		return "Teleport Destination"
	case 3004:
		return "Zombieman"
	case 9:
		return "Shotgun Guy"
	case 65:
		return "Chaingunner"
	case 3001:
		return "Imp"
	case 3002:
		return "Demon"
	case 58:
		return "Spectre"
	case 3006:
		return "Lost Soul"
	case 3005:
		return "Cacodemon"
	case 3003:
		return "Baron of Hell"
	case 69:
		return "Hell Knight"
	case 16:
		return "Cyberdemon"
	case 7:
		return "Spider Mastermind"
	case 64:
		return "Arch-Vile"
	case 66:
		return "Revenant"
	case 67:
		return "Mancubus"
	case 68:
		return "Arachnotron"
	case 84:
		return "Wolfenstein SS"
	case 88:
		return "Boss Brain"
	case 2035:
		return "Explosive Barrel"
	case 70:
		return "Burning Barrel"
	case 34:
		return "Candle"
	case 35:
		return "Candelabra"
	case 44:
		return "Tall Blue Torch"
	case 45:
		return "Tall Green Torch"
	case 46:
		return "Tall Red Torch"
	case 55:
		return "Short Blue Torch"
	case 56:
		return "Short Green Torch"
	case 57:
		return "Short Red Torch"
	case 47:
		return "Stump"
	case 43:
		return "Burnt Tree"
	case 54:
		return "Tall Tree"
	case 48:
		return "Tech Column"
	case 30:
		return "Tall Green Pillar"
	case 32:
		return "Tall Red Pillar"
	case 31:
		return "Short Green Pillar"
	case 33:
		return "Short Red Pillar"
	case 85:
		return "Tall Tech Lamp"
	case 86:
		return "Short Tech Lamp"
	default:
		return fmt.Sprintf("Thing %d", thingType)
	}
}

// GetThingRadius returns the collision/bounding radius of a Doom thing in map units.
func GetThingRadius(thingType int16) float64 {
	if def, ok := wad.LookupItemDef(thingType); ok && def.Radius > 0 {
		return def.Radius
	}
	switch thingType {
	case 1, 2, 3, 4, 11: // Player starts / deathmatch start
		return 16.0
	case 3004, 9, 65, 3001, 3006: // Zombieman, Shotgun guy, Chaingunner, Imp, Lost Soul
		return 20.0
	case 3002, 58: // Demon, Specter
		return 30.0
	case 3005, 3003, 69: // Cacodemon, Baron of Hell, Hell Knight
		return 31.0
	case 16, 7: // Cyberdemon, Spider Mastermind
		return 40.0
	default:
		return 16.0
	}
}

// distPointToSegment computes the minimum Euclidean distance from point (px, py) to segment (x1, y1)-(x2, y2).
func distPointToSegment(px, py, x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	lenSq := dx*dx + dy*dy
	if lenSq == 0 {
		return math.Hypot(px-x1, py-y1)
	}
	t := ((px-x1)*dx + (py-y1)*dy) / lenSq
	if t < 0 {
		return math.Hypot(px-x1, py-y1)
	}
	if t > 1 {
		return math.Hypot(px-x2, py-y2)
	}
	projX := x1 + t*dx
	projY := y1 + t*dy
	return math.Hypot(px-projX, py-projY)
}

// updateHoverTargets updates which map entity is hovered based on cursor position and active edit mode.
func (e *EditorMode) updateHoverTargets(mx, my int) {
	e.hoveredVertex = -1
	e.hoveredLinedef = -1
	e.hoveredSector = -1
	e.hoveredThing = -1

	// Only track hover when cursor is inside the 400x392 editing window
	if mx < 0 || mx >= EditingWindowWidth || my < 0 || my >= EditingWindowHeight {
		return
	}

	md := e.MapData()
	if md == nil {
		return
	}

	curX := float64(mx)
	curY := float64(my)
	const maxScreenDist = 5.0

	switch e.editMode {
	case EditModeVertex:
		minDist := maxScreenDist
		for i, v := range md.Vertexes {
			vx, vy := e.WorldToScreen(float64(v.X), float64(v.Y))
			dist := math.Hypot(curX-vx, curY-vy)
			if dist <= maxScreenDist && dist < minDist {
				minDist = dist
				e.hoveredVertex = i
			}
		}

	case EditModeLine:
		minDist := maxScreenDist
		for i, ld := range md.Linedefs {
			if int(ld.V1) < len(md.Vertexes) && int(ld.V2) < len(md.Vertexes) {
				v1 := md.Vertexes[ld.V1]
				v2 := md.Vertexes[ld.V2]
				x1, y1 := e.WorldToScreen(float64(v1.X), float64(v1.Y))
				x2, y2 := e.WorldToScreen(float64(v2.X), float64(v2.Y))
				dist := distPointToSegment(curX, curY, x1, y1, x2, y2)
				if dist <= maxScreenDist && dist < minDist {
					minDist = dist
					e.hoveredLinedef = i
				}
			}
		}

	case EditModeSector:
		wx, wy := e.ScreenToWorld(curX, curY)
		if secIdx, ok := md.SectorIndexAt(wx, wy); ok {
			e.hoveredSector = secIdx
		}

	case EditModeThing:
		minDist := math.MaxFloat64
		for i, t := range md.Things {
			tx, ty := e.WorldToScreen(float64(t.X), float64(t.Y))
			radius := GetThingRadius(t.Type)
			screenRadius := radius * e.ZoomScale()
			if screenRadius < 2.0 {
				screenRadius = 2.0
			}
			dist := math.Hypot(curX-tx, curY-ty)
			if dist <= screenRadius+maxScreenDist && dist < minDist {
				minDist = dist
				e.hoveredThing = i
			}
		}
	}
}

// SetWAD updates the WAD container reference for the editor.
func (e *EditorMode) SetWAD(w *wad.WAD) {
	e.wadFile = w
}

// WAD returns the active WAD container.
func (e *EditorMode) WAD() *wad.WAD {
	return e.wadFile
}

// SetMapData sets the active map geometry data to display and centers the camera on it.
func (e *EditorMode) SetMapData(m *wad.MapData) {
	e.mapData = m
	e.lastMapData = m
	if m != nil {
		e.hasCenteredCam = true
		e.CenterOnMap()
	}
}

// SetMapDataProvider sets a dynamic provider function for map data.
func (e *EditorMode) SetMapDataProvider(fn func() *wad.MapData) {
	e.mapDataProvider = fn
}

// MapData returns the active map geometry data.
func (e *EditorMode) MapData() *wad.MapData {
	if e.mapData != nil {
		return e.mapData
	}
	if e.mapDataProvider != nil {
		return e.mapDataProvider()
	}
	return nil
}

// CenterOnMap centers the camera on the map's player 1 start position or bounding box center.
func (e *EditorMode) CenterOnMap() {
	md := e.MapData()
	if md == nil {
		return
	}
	if p1, ok := md.Player1Start(); ok {
		e.camX = float64(p1.X)
		e.camY = float64(p1.Y)
		return
	}
	minX, maxX, minY, maxY := md.Bounds()
	e.camX = float64(minX+maxX) / 2.0
	e.camY = float64(minY+maxY) / 2.0
}

// SetFont updates the console font reference for the editor.
func (e *EditorMode) SetFont(f *font.ConsoleFont) {
	e.font = f
}

// SetOnToggle sets the callback invoked when F12 is pressed to toggle out of the editor.
func (e *EditorMode) SetOnToggle(fn func()) {
	e.onToggle = fn
}

// SetOnToggleConsole sets the callback invoked when GraveAccent is pressed to toggle console.
func (e *EditorMode) SetOnToggleConsole(fn func()) {
	e.onToggleConsole = fn
}

// GridSize returns the current grid size in map units (1..256).
func (e *EditorMode) GridSize() int {
	return e.gridSize
}

// SetGridSize sets the grid size, clamping it to the nearest valid power-of-2 between 1 and 256.
func (e *EditorMode) SetGridSize(size int) {
	if size <= MinGridSize {
		e.gridSize = MinGridSize
		return
	}
	if size >= MaxGridSize {
		e.gridSize = MaxGridSize
		return
	}
	// Round to nearest power of 2
	p := 1
	for p < size && p < MaxGridSize {
		p *= 2
	}
	if p > MaxGridSize {
		p = MaxGridSize
	}
	e.gridSize = p
}

// IncreaseGridSize doubles the grid size up to MaxGridSize (256).
func (e *EditorMode) IncreaseGridSize() {
	if e.gridSize < MaxGridSize {
		e.gridSize *= 2
		if e.gridSize > MaxGridSize {
			e.gridSize = MaxGridSize
		}
	}
}

// DecreaseGridSize halves the grid size down to MinGridSize (1).
func (e *EditorMode) DecreaseGridSize() {
	if e.gridSize > MinGridSize {
		e.gridSize /= 2
		if e.gridSize < MinGridSize {
			e.gridSize = MinGridSize
		}
	}
}

// ZoomLevel returns the current zoom level index (0..11).
func (e *EditorMode) ZoomLevel() int {
	return e.zoomLevel
}

// ZoomScale returns the pixels per map unit scale corresponding to the current zoom level.
func (e *EditorMode) ZoomScale() float64 {
	if e.zoomLevel < 0 {
		return ZoomScales[0]
	}
	if e.zoomLevel >= len(ZoomScales) {
		return ZoomScales[len(ZoomScales)-1]
	}
	return ZoomScales[e.zoomLevel]
}

// SetZoomLevel sets the zoom level index, clamping it between MinZoomLevel (0) and MaxZoomLevel (11).
func (e *EditorMode) SetZoomLevel(z int) {
	if z <= MinZoomLevel {
		e.zoomLevel = MinZoomLevel
		return
	}
	if z >= MaxZoomLevel {
		e.zoomLevel = MaxZoomLevel
		return
	}
	e.zoomLevel = z
}

// IncreaseZoom steps up the zoom level index up to MaxZoomLevel (11).
func (e *EditorMode) IncreaseZoom() {
	if e.zoomLevel < MaxZoomLevel {
		e.zoomLevel++
	}
}

// DecreaseZoom steps down the zoom level index down to MinZoomLevel (0).
func (e *EditorMode) DecreaseZoom() {
	if e.zoomLevel > MinZoomLevel {
		e.zoomLevel--
	}
}

// CamX returns the current camera X position in world units.
func (e *EditorMode) CamX() float64 {
	return e.camX
}

// CamY returns the current camera Y position in world units.
func (e *EditorMode) CamY() float64 {
	return e.camY
}

// SetCamPosition sets the camera center position in world units.
func (e *EditorMode) SetCamPosition(x, y float64) {
	e.camX = x
	e.camY = y
}

// WorldToScreen converts world map coordinates to screen pixel coordinates in the editing window.
func (e *EditorMode) WorldToScreen(wx, wy float64) (float64, float64) {
	centerX := float64(EditingWindowWidth) / 2.0
	centerY := float64(EditingWindowHeight) / 2.0
	scale := e.ZoomScale()
	sx := centerX + (wx-e.camX)*scale
	sy := centerY - (wy-e.camY)*scale // Doom +Y is up, screen +Y is down
	return sx, sy
}

// ScreenToWorld converts editing window pixel coordinates to world map coordinates.
func (e *EditorMode) ScreenToWorld(sx, sy float64) (float64, float64) {
	centerX := float64(EditingWindowWidth) / 2.0
	centerY := float64(EditingWindowHeight) / 2.0
	scale := e.ZoomScale()
	wx := e.camX + (sx-centerX)/scale
	wy := e.camY - (sy-centerY)/scale
	return wx, wy
}

// Update handles input and advances editor state.
func (e *EditorMode) Update() error {
	// Mode switching with F12
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		if e.onToggle != nil {
			e.onToggle()
		}
		return nil
	}

	// Console toggle with Grave Accent (~)
	if inpututil.IsKeyJustPressed(ebiten.KeyGraveAccent) {
		if e.onToggleConsole != nil {
			e.onToggleConsole()
		}
		return nil
	}

	// Escape clears selection
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		e.ClearSelection()
	}

	// Grid size adjustment with [ and ]
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		e.DecreaseGridSize()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		e.IncreaseGridSize()
	}

	// Map cursor coordinates from 1280x800 screen space to 640x400 editor space
	rawMx, rawMy := ebiten.CursorPosition()
	mx := rawMx * EditorBufferWidth / 1280
	my := rawMy * EditorBufferHeight / 800

	// Zoom level adjustment with mouse wheel
	_, wheelY := ebiten.Wheel()
	if mx < EditingWindowWidth && my < EditingWindowHeight {
		if wheelY > 0 {
			e.IncreaseZoom()
		} else if wheelY < 0 {
			e.DecreaseZoom()
		}
	}

	// Zoom level adjustment with = / + and -
	ctrlPressed := ebiten.IsKeyPressed(ebiten.KeyControl) ||
		ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyControlRight)

	if ctrlPressed || mx < EditingWindowWidth {
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
			e.IncreaseZoom()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
			e.DecreaseZoom()
		}
	}

	// Panning with middle click or right click drag in editing window
	isPanButton := ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) ||
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)

	if isPanButton && mx < EditingWindowWidth && my < EditingWindowHeight {
		if !e.isPanning {
			e.isPanning = true
			e.panStartX = mx
			e.panStartY = my
			e.origCamX = e.camX
			e.origCamY = e.camY
		} else {
			dx := float64(mx - e.panStartX)
			dy := float64(my - e.panStartY)
			scale := e.ZoomScale()
			e.camX = e.origCamX - dx/scale
			e.camY = e.origCamY + dy/scale
		}
	} else {
		e.isPanning = false
	}

	// Panning with arrow keys
	panStep := 16.0 / e.ZoomScale()
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		e.camX -= panStep
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		e.camX += panStep
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		e.camY += panStep
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		e.camY -= panStep
	}

	// Mode switching with 1, 2, 3, 4 number keys
	if inpututil.IsKeyJustPressed(ebiten.Key1) || inpututil.IsKeyJustPressed(ebiten.KeyDigit1) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad1) {
		e.SetEditMode(EditModeVertex)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) || inpututil.IsKeyJustPressed(ebiten.KeyDigit2) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad2) {
		e.SetEditMode(EditModeLine)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) || inpututil.IsKeyJustPressed(ebiten.KeyDigit3) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad3) {
		e.SetEditMode(EditModeSector)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) || inpututil.IsKeyJustPressed(ebiten.KeyDigit4) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad4) {
		e.SetEditMode(EditModeThing)
	}

	// Update hover targets based on cursor location
	e.updateHoverTargets(mx, my)

	// Center camera on map with C or Home key
	if inpututil.IsKeyJustPressed(ebiten.KeyC) || inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		e.CenterOnMap()
	}

	// Left-click handling: button toolbar and map element selection
	leftClick := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	// Selection interaction in editing grid window
	if leftClick && mx >= 0 && mx < EditingWindowWidth && my >= 0 && my < EditingWindowHeight {
		switch e.editMode {
		case EditModeVertex:
			if e.hoveredVertex >= 0 {
				e.SelectVertex(e.hoveredVertex, ctrlPressed)
			} else {
				e.ClearSelection()
			}
		case EditModeLine:
			if e.hoveredLinedef >= 0 {
				e.SelectLine(e.hoveredLinedef, ctrlPressed)
			} else {
				e.ClearSelection()
			}
		case EditModeSector:
			if e.hoveredSector >= 0 {
				e.SelectSector(e.hoveredSector, ctrlPressed)
			} else {
				e.ClearSelection()
			}
		case EditModeThing:
			if e.hoveredThing >= 0 {
				e.SelectThing(e.hoveredThing, ctrlPressed)
			} else {
				e.ClearSelection()
			}
		}
	}

	// Update button hover states and handle click events
	for i := 0; i < NumIconButtons; i++ {
		if !e.buttons[i].Visible {
			e.buttons[i].Hovered = false
			continue
		}

		bx := UserPanelX + i*IconButtonSize
		by := UserPanelY
		isHovered := (mx >= bx && mx < bx+IconButtonSize && my >= by && my < by+IconButtonSize)
		e.buttons[i].Hovered = isHovered

		if isHovered && leftClick && e.buttons[i].Enabled {
			if e.buttons[i].IsToggle {
				// Radio-group behavior: exactly one toggle button active at all times
				e.SetEditMode(EditMode(i - 3))
			}
			if e.buttons[i].OnClick != nil {
				e.buttons[i].OnClick(e)
			}
		}
	}

	return nil
}

// Draw renders the entire 640x400 editor mode interface and composites onto the screen.
func (e *EditorMode) Draw(screen *ebiten.Image) {
	if e.buffer == nil {
		e.buffer = ebiten.NewImage(EditorBufferWidth, EditorBufferHeight)
	}

	// 1. Draw 400x392 Editing Window
	e.drawEditingWindow(e.buffer)

	// 2. Draw 240x392 User Panel
	e.drawUserPanel(e.buffer)

	// 3. Draw 640x8 Status Bar
	e.drawStatusBar(e.buffer)

	// 4. Composite 640x400 editor buffer onto destination screen
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	scaleX := float64(sw) / float64(EditorBufferWidth)
	scaleY := float64(sh) / float64(EditorBufferHeight)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(e.buffer, op)
}

// drawEditingWindow renders the editing background, darker grid lines, map geometry, things, and vertices.
func (e *EditorMode) drawEditingWindow(dst *ebiten.Image) {
	// Fill editing area background with black
	vector.DrawFilledRect(dst, 0, 0, EditingWindowWidth, EditingWindowHeight, gfx.EGABlack, false)

	// Compute visible world bounds
	minWx, maxWy := e.ScreenToWorld(0, 0)
	maxWx, minWy := e.ScreenToWorld(EditingWindowWidth, EditingWindowHeight)

	gridStep := float64(e.gridSize)
	pixelSpacing := gridStep * e.ZoomScale()

	// 1. Draw darker grid lines
	if pixelSpacing >= 1.0 {
		gridColor := color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}

		// Vertical grid lines
		startK := int(math.Floor(minWx / gridStep))
		endK := int(math.Ceil(maxWx / gridStep))

		for k := startK; k <= endK; k++ {
			wx := float64(k) * gridStep
			sx, _ := e.WorldToScreen(wx, 0)
			if sx >= 0 && sx < EditingWindowWidth {
				vector.StrokeLine(dst, float32(sx), 0, float32(sx), float32(EditingWindowHeight), 1.0, gridColor, false)
			}
		}

		// Horizontal grid lines
		startM := int(math.Floor(minWy / gridStep))
		endM := int(math.Ceil(maxWy / gridStep))

		for m := startM; m <= endM; m++ {
			wy := float64(m) * gridStep
			_, sy := e.WorldToScreen(0, wy)
			if sy >= 0 && sy < EditingWindowHeight {
				vector.StrokeLine(dst, 0, float32(sy), float32(EditingWindowWidth), float32(sy), 1.0, gridColor, false)
			}
		}
	}

	// 2. Draw map geometry (linedefs, things, vertices)
	md := e.MapData()
	if md != nil {
		if !e.hasCenteredCam || e.lastMapData != md {
			e.lastMapData = md
			e.hasCenteredCam = true
			e.CenterOnMap()
		}

		// 2a. Draw linedefs (1px-wide lines, selected in yellow, hovered in green)
		for i, ld := range md.Linedefs {
			if int(ld.V1) < len(md.Vertexes) && int(ld.V2) < len(md.Vertexes) {
				v1 := md.Vertexes[ld.V1]
				v2 := md.Vertexes[ld.V2]
				x1, y1 := e.WorldToScreen(float64(v1.X), float64(v1.Y))
				x2, y2 := e.WorldToScreen(float64(v2.X), float64(v2.Y))

				lineColor := gfx.EGABrightWhite
				if e.editMode == EditModeLine {
					if e.selectedLines[i] {
						lineColor = gfx.EGABrightYellow
					} else if e.hoveredLinedef == i {
						lineColor = gfx.EGABrightGreen
					}
				} else if e.editMode == EditModeSector {
					isHoveredSectorLine := false
					isSelectedSectorLine := false
					if ld.RightSide != 0xFFFF && int(ld.RightSide) < len(md.Sidedefs) {
						sec := int(md.Sidedefs[ld.RightSide].Sector)
						if e.selectedSectors[sec] {
							isSelectedSectorLine = true
						}
						if sec == e.hoveredSector {
							isHoveredSectorLine = true
						}
					}
					if ld.LeftSide != 0xFFFF && int(ld.LeftSide) < len(md.Sidedefs) {
						sec := int(md.Sidedefs[ld.LeftSide].Sector)
						if e.selectedSectors[sec] {
							isSelectedSectorLine = true
						}
						if sec == e.hoveredSector {
							isHoveredSectorLine = true
						}
					}
					if isSelectedSectorLine {
						lineColor = gfx.EGABrightYellow
					} else if isHoveredSectorLine {
						lineColor = gfx.EGABrightGreen
					}
				}

				vector.StrokeLine(dst, float32(x1), float32(y1), float32(x2), float32(y2), 1.0, lineColor, true)
			}
		}

		// 2b. Draw Things as squares with crosses through them (selected in yellow, hovered in green, normal cyan)
		for i, t := range md.Things {
			tx, ty := e.WorldToScreen(float64(t.X), float64(t.Y))
			radius := GetThingRadius(t.Type)
			screenRadius := radius * e.ZoomScale()
			boxSize := screenRadius * 2.0
			if boxSize < 4.0 {
				boxSize = 4.0
			}
			halfBox := boxSize / 2.0

			// Screen bounding box check
			if tx+halfBox < 0 || tx-halfBox > float64(EditingWindowWidth) ||
				ty+halfBox < 0 || ty-halfBox > float64(EditingWindowHeight) {
				continue
			}

			thingColor := gfx.EGABrightCyan
			if e.editMode == EditModeThing {
				if e.selectedThings[i] {
					thingColor = gfx.EGABrightYellow
				} else if e.hoveredThing == i {
					thingColor = gfx.EGABrightGreen
				}
			}

			left := float32(tx - halfBox)
			top := float32(ty - halfBox)
			w := float32(boxSize)
			h := float32(boxSize)

			// Draw square outline
			vector.StrokeRect(dst, left, top, w, h, 1.0, thingColor, true)

			// Draw diagonal cross through the square
			vector.StrokeLine(dst, left, top, left+w, top+h, 1.0, thingColor, true)
			vector.StrokeLine(dst, left+w, top, left, top+h, 1.0, thingColor, true)
		}

		// 2c. Draw vertices as chunky squares (4x4, selected in yellow, hovered in green, normal white)
		const vertexSize = 4.0
		const halfVertex = vertexSize / 2.0
		for i, v := range md.Vertexes {
			vx, vy := e.WorldToScreen(float64(v.X), float64(v.Y))
			if vx >= -10 && vx <= float64(EditingWindowWidth)+10 &&
				vy >= -10 && vy <= float64(EditingWindowHeight)+10 {
				vColor := gfx.EGABrightWhite
				if e.editMode == EditModeVertex {
					if e.selectedVertexes[i] {
						vColor = gfx.EGABrightYellow
					} else if e.hoveredVertex == i {
						vColor = gfx.EGABrightGreen
					}
				}
				vector.DrawFilledRect(dst, float32(vx)-halfVertex, float32(vy)-halfVertex, vertexSize, vertexSize, vColor, false)
			}
		}
	}
}

// PropertyPair represents a name-value inspector property.
type PropertyPair struct {
	Name  string
	Value string
}

// InspectorProperties calculates the list of inspector properties for the given element indices.
func (e *EditorMode) InspectorProperties(indices []int) []PropertyPair {
	md := e.MapData()
	if md == nil || len(indices) == 0 {
		return nil
	}

	getPropsForIndex := func(idx int) []PropertyPair {
		switch e.editMode {
		case EditModeVertex:
			if idx < 0 || idx >= len(md.Vertexes) {
				return nil
			}
			v := md.Vertexes[idx]
			return []PropertyPair{
				{Name: "Index", Value: fmt.Sprintf("%d", idx)},
				{Name: "X", Value: fmt.Sprintf("%d", v.X)},
				{Name: "Y", Value: fmt.Sprintf("%d", v.Y)},
			}
		case EditModeLine:
			if idx < 0 || idx >= len(md.Linedefs) {
				return nil
			}
			ld := md.Linedefs[idx]
			rSide := "None"
			if ld.RightSide != 0xFFFF {
				rSide = fmt.Sprintf("%d", ld.RightSide)
			}
			lSide := "None"
			if ld.LeftSide != 0xFFFF {
				lSide = fmt.Sprintf("%d", ld.LeftSide)
			}
			return []PropertyPair{
				{Name: "Index", Value: fmt.Sprintf("%d", idx)},
				{Name: "V1", Value: fmt.Sprintf("%d", ld.V1)},
				{Name: "V2", Value: fmt.Sprintf("%d", ld.V2)},
				{Name: "Flags", Value: fmt.Sprintf("%d", ld.Flags)},
				{Name: "Special", Value: fmt.Sprintf("%d", ld.Special)},
				{Name: "Tag", Value: fmt.Sprintf("%d", ld.Tag)},
				{Name: "Right Sidedef", Value: rSide},
				{Name: "Left Sidedef", Value: lSide},
			}
		case EditModeSector:
			if idx < 0 || idx >= len(md.Sectors) {
				return nil
			}
			sec := md.Sectors[idx]
			return []PropertyPair{
				{Name: "Index", Value: fmt.Sprintf("%d", idx)},
				{Name: "Floor Height", Value: fmt.Sprintf("%d", sec.FloorHeight)},
				{Name: "Ceiling Height", Value: fmt.Sprintf("%d", sec.CeilingHeight)},
				{Name: "Floor Pic", Value: sec.FloorPic},
				{Name: "Ceiling Pic", Value: sec.CeilingPic},
				{Name: "Light Level", Value: fmt.Sprintf("%d", sec.LightLevel)},
				{Name: "Special", Value: fmt.Sprintf("%d", sec.Special)},
				{Name: "Tag", Value: fmt.Sprintf("%d", sec.Tag)},
			}
		case EditModeThing:
			if idx < 0 || idx >= len(md.Things) {
				return nil
			}
			th := md.Things[idx]
			return []PropertyPair{
				{Name: "Index", Value: fmt.Sprintf("%d", idx)},
				{Name: "Type", Value: fmt.Sprintf("%d", th.Type)},
				{Name: "Name", Value: LookupThingName(th.Type)},
				{Name: "X", Value: fmt.Sprintf("%d", th.X)},
				{Name: "Y", Value: fmt.Sprintf("%d", th.Y)},
				{Name: "Angle", Value: fmt.Sprintf("%d", th.Angle)},
				{Name: "Flags", Value: fmt.Sprintf("%d", th.Flags)},
			}
		default:
			return nil
		}
	}

	base := getPropsForIndex(indices[0])
	if len(base) == 0 {
		return nil
	}

	if len(indices) == 1 {
		return base
	}

	// Multi-selection comparison
	result := make([]PropertyPair, len(base))
	copy(result, base)

	for _, idx := range indices[1:] {
		other := getPropsForIndex(idx)
		if len(other) != len(result) {
			continue
		}
		for i := range result {
			if result[i].Value != "[many]" && result[i].Value != other[i].Value {
				result[i].Value = "[many]"
			}
		}
	}

	return result
}

// drawUserPanel renders the 240x392 dark gray panel with 16x16 icon buttons at the top and inspector properties below.
func (e *EditorMode) drawUserPanel(dst *ebiten.Image) {
	// Fill user panel background with dark gray
	vector.DrawFilledRect(dst, UserPanelX, UserPanelY, UserPanelWidth, UserPanelHeight, gfx.EGADarkGray, false)

	if e.iconsImage == nil {
		e.iconsImage, _ = loadEditorIcons()
	}

	// Draw array of 16x16 icon buttons
	btnBorderColor := color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xFF}
	btnHoverColor := color.RGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xFF}
	btnNormalColor := color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}

	for i := 0; i < NumIconButtons; i++ {
		btn := e.buttons[i]
		if !btn.Visible || btn.Icon < 0 {
			continue
		}

		bx := float32(UserPanelX + i*IconButtonSize)
		by := float32(UserPanelY)

		btnBg := btnNormalColor
		if btn.Hovered && btn.Enabled {
			btnBg = btnHoverColor
		}

		// Button background
		vector.DrawFilledRect(dst, bx+1, by+1, IconButtonSize-2, IconButtonSize-2, btnBg, false)

		// Button border
		vector.StrokeRect(dst, bx, by, IconButtonSize, IconButtonSize, 1.0, btnBorderColor, false)

		// Draw icon with color tinting
		if e.iconsImage != nil {
			srcX := (btn.Icon % 8) * 16
			srcY := (btn.Icon / 8) * 16
			sub := e.iconsImage.SubImage(image.Rect(srcX, srcY, srcX+16, srcY+16)).(*ebiten.Image)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(bx), float64(by))

			if !btn.Enabled {
				// Draw tinted dark gray when disabled (85/255)
				op.ColorScale.Scale(85.0/255.0, 85.0/255.0, 85.0/255.0, 1.0)
			} else if btn.IsToggle && btn.Active {
				// When a toggle button is on, draw tinted bright / lime green
				op.ColorScale.Scale(85.0/255.0, 1.0, 85.0/255.0, 1.0)
			} else {
				// Draw tinted white when enabled
				op.ColorScale.Scale(1.0, 1.0, 1.0, 1.0)
			}

			dst.DrawImage(sub, op)
		}
	}

	// Draw property inspector below buttons
	if e.font != nil {
		var targetIndices []int
		activeSel := e.ActiveSelection()
		if len(activeSel) > 0 {
			targetIndices = activeSel
		} else {
			var hovered int
			switch e.editMode {
			case EditModeVertex:
				hovered = e.hoveredVertex
			case EditModeLine:
				hovered = e.hoveredLinedef
			case EditModeSector:
				hovered = e.hoveredSector
			case EditModeThing:
				hovered = e.hoveredThing
			}
			if hovered >= 0 {
				targetIndices = []int{hovered}
			}
		}

		if len(targetIndices) > 0 {
			props := e.InspectorProperties(targetIndices)
			leftX := UserPanelX + 8
			rightX := UserPanelX + UserPanelWidth - 8
			y := 28
			const lineHeight = 12

			for _, p := range props {
				// Align names of things to the left, use black
				e.font.DrawText(dst, p.Name, leftX, y, gfx.EGABlack)

				// Align values to the right, use black
				valWidth, _ := e.font.MeasureText(p.Value)
				valX := rightX - valWidth
				if valX < leftX {
					valX = leftX
				}
				e.font.DrawText(dst, p.Value, valX, y, gfx.EGABlack)

				y += lineHeight
			}
		}
	}
}

// drawStatusBar renders the 640x8 dark blue status bar with yellow IWAD/PWAD and right-justified Grid/Zoom text.
func (e *EditorMode) drawStatusBar(dst *ebiten.Image) {
	// Fill status bar background with dark blue
	vector.DrawFilledRect(dst, StatusBarX, StatusBarY, StatusBarWidth, StatusBarHeight, gfx.EGADarkBlue, false)

	if e.font == nil {
		return
	}

	// 1. Far-left: IWAD=[filename]  PWAD=NONE
	iwadName := "NONE"
	if e.wadFile != nil && e.wadFile.Filename() != "" {
		iwadName = e.wadFile.Filename()
	}
	leftText := fmt.Sprintf("IWAD=%s  PWAD=NONE", iwadName)
	e.font.DrawText(dst, leftText, StatusBarX, StatusBarY, gfx.EGABrightYellow)

	// 2. Right-justified: Grid and Zoom settings in fixed-width fields
	rightText := fmt.Sprintf("GRID: %3d  ZOOM: %2d", e.gridSize, e.zoomLevel)
	rightTextWidth, _ := e.font.MeasureText(rightText)
	rightX := StatusBarWidth - rightTextWidth
	if rightX > 0 {
		e.font.DrawText(dst, rightText, rightX, StatusBarY, gfx.EGABrightYellow)
	}
}
