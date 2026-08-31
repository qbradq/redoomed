package mode

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/qbradq/redoomed/pkg/font"
	"github.com/qbradq/redoomed/pkg/gfx"
	"github.com/qbradq/redoomed/pkg/wad"
)

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

// IconButton represents a single 16x16 icon button slot in the user panel.
type IconButton struct {
	Index   int
	Hovered bool
	Active  bool
	Tooltip string
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

	gridSize  int
	zoomLevel int

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
	ed := &EditorMode{
		buffer:    ebiten.NewImage(EditorBufferWidth, EditorBufferHeight),
		wadFile:   w,
		font:      f,
		gridSize:  DefaultGridSize,
		zoomLevel: DefaultZoomLevel,
		camX:      0,
		camY:      0,
	}

	for i := 0; i < NumIconButtons; i++ {
		ed.buttons[i] = IconButton{
			Index:   i,
			Tooltip: fmt.Sprintf("Tool %d", i+1),
		}
	}

	return ed
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

	// Center camera on map with C or Home key
	if inpututil.IsKeyJustPressed(ebiten.KeyC) || inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		e.CenterOnMap()
	}

	// Update button hover states
	for i := 0; i < NumIconButtons; i++ {
		bx := UserPanelX + i*IconButtonSize
		by := UserPanelY
		e.buttons[i].Hovered = (mx >= bx && mx < bx+IconButtonSize && my >= by && my < by+IconButtonSize)
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

// drawEditingWindow renders the editing background, darker grid lines, and map geometry (linedefs and vertices).
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

	// 2. Draw map geometry (linedefs and vertices) in white
	md := e.MapData()
	if md != nil {
		if !e.hasCenteredCam || e.lastMapData != md {
			e.lastMapData = md
			e.hasCenteredCam = true
			e.CenterOnMap()
		}

		white := gfx.EGABrightWhite

		// Draw 1px-wide linedef lines
		for _, ld := range md.Linedefs {
			if int(ld.V1) < len(md.Vertexes) && int(ld.V2) < len(md.Vertexes) {
				v1 := md.Vertexes[ld.V1]
				v2 := md.Vertexes[ld.V2]
				x1, y1 := e.WorldToScreen(float64(v1.X), float64(v1.Y))
				x2, y2 := e.WorldToScreen(float64(v2.X), float64(v2.Y))
				vector.StrokeLine(dst, float32(x1), float32(y1), float32(x2), float32(y2), 1.0, white, true)
			}
		}

		// Draw vertices as chunky squares (4x4)
		const vertexSize = 4.0
		const halfVertex = vertexSize / 2.0
		for _, v := range md.Vertexes {
			vx, vy := e.WorldToScreen(float64(v.X), float64(v.Y))
			if vx >= -10 && vx <= float64(EditingWindowWidth)+10 &&
				vy >= -10 && vy <= float64(EditingWindowHeight)+10 {
				vector.DrawFilledRect(dst, float32(vx)-halfVertex, float32(vy)-halfVertex, vertexSize, vertexSize, white, false)
			}
		}
	}
}

// drawUserPanel renders the 240x392 dark gray panel with 15x1 16x16 icon buttons at the top.
func (e *EditorMode) drawUserPanel(dst *ebiten.Image) {
	// Fill user panel background with dark gray
	vector.DrawFilledRect(dst, UserPanelX, UserPanelY, UserPanelWidth, UserPanelHeight, gfx.EGADarkGray, false)

	// Draw 15x1 array of 16x16 icon buttons
	btnBorderColor := color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xFF}
	btnHoverColor := color.RGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xFF}
	btnNormalColor := color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}

	for i := 0; i < NumIconButtons; i++ {
		bx := float32(UserPanelX + i*IconButtonSize)
		by := float32(UserPanelY)

		btnBg := btnNormalColor
		if e.buttons[i].Hovered {
			btnBg = btnHoverColor
		}

		// Button background
		vector.DrawFilledRect(dst, bx+1, by+1, IconButtonSize-2, IconButtonSize-2, btnBg, false)

		// Button border
		vector.StrokeRect(dst, bx, by, IconButtonSize, IconButtonSize, 1.0, btnBorderColor, false)
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
