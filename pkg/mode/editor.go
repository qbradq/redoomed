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
	// EditorBufferWidth is the logical width of the editor mode (1280).
	EditorBufferWidth = 1280
	// EditorBufferHeight is the logical height of the editor mode (800).
	EditorBufferHeight = 800

	// EditingWindowWidth is the width of the editing grid area (960).
	EditingWindowWidth = 960
	// EditingWindowHeight is the height of the editing grid area (792).
	EditingWindowHeight = 792

	// UserPanelX is the horizontal starting coordinate of the user panel (960).
	UserPanelX = 960
	// UserPanelY is the vertical starting coordinate of the user panel (0).
	UserPanelY = 0
	// UserPanelWidth is the width of the user panel (320).
	UserPanelWidth = 320
	// UserPanelHeight is the height of the user panel (792).
	UserPanelHeight = 792

	// StatusBarX is the horizontal starting coordinate of the status bar (0).
	StatusBarX = 0
	// StatusBarY is the vertical starting coordinate of the status bar (792).
	StatusBarY = 792
	// StatusBarWidth is the width of the status bar (1280).
	StatusBarWidth = 1280
	// StatusBarHeight is the height of the status bar (8).
	StatusBarHeight = 8

	// NumIconButtons is the number of 16x16 icon buttons in the top of the user panel.
	NumIconButtons = 20
	// IconButtonSize is the dimension of each square icon button in pixels.
	IconButtonSize = 16

	// MinGridSize is the minimum allowed grid size (1 unit).
	MinGridSize = 1
	// MaxGridSize is the maximum allowed grid size (256 units).
	MaxGridSize = 256
	// DefaultGridSize is the initial grid size (16 units).
	DefaultGridSize = 16

	// MinZoomLevel is the minimum allowed zoom level (0.25 pixels per unit).
	MinZoomLevel = 0.25
	// MaxZoomLevel is the maximum allowed zoom level (32 pixels per unit).
	MaxZoomLevel = 32.0
	// DefaultZoomLevel is the initial zoom level (1.0 pixel per unit).
	DefaultZoomLevel = 1.0
)

// IconButton represents a single 16x16 icon button slot in the user panel.
type IconButton struct {
	Index   int
	Hovered bool
	Active  bool
	Tooltip string
}

// EditorMode represents the 1280x800 map editor mode.
type EditorMode struct {
	wadFile *wad.WAD
	font    *font.ConsoleFont

	gridSize  int
	zoomLevel float64

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

// ZoomLevel returns the current zoom level (pixels per map unit).
func (e *EditorMode) ZoomLevel() float64 {
	return e.zoomLevel
}

// SetZoomLevel sets the zoom level, clamping it between MinZoomLevel (0.25) and MaxZoomLevel (32).
func (e *EditorMode) SetZoomLevel(z float64) {
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

// IncreaseZoom doubles the zoom level up to MaxZoomLevel (32).
func (e *EditorMode) IncreaseZoom() {
	if e.zoomLevel < MaxZoomLevel {
		e.zoomLevel *= 2.0
		if e.zoomLevel > MaxZoomLevel {
			e.zoomLevel = MaxZoomLevel
		}
	}
}

// DecreaseZoom halves the zoom level down to MinZoomLevel (0.25).
func (e *EditorMode) DecreaseZoom() {
	if e.zoomLevel > MinZoomLevel {
		e.zoomLevel /= 2.0
		if e.zoomLevel < MinZoomLevel {
			e.zoomLevel = MinZoomLevel
		}
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
	sx := centerX + (wx-e.camX)*e.zoomLevel
	sy := centerY - (wy-e.camY)*e.zoomLevel // Doom +Y is up, screen +Y is down
	return sx, sy
}

// ScreenToWorld converts editing window pixel coordinates to world map coordinates.
func (e *EditorMode) ScreenToWorld(sx, sy float64) (float64, float64) {
	centerX := float64(EditingWindowWidth) / 2.0
	centerY := float64(EditingWindowHeight) / 2.0
	wx := e.camX + (sx-centerX)/e.zoomLevel
	wy := e.camY - (sy-centerY)/e.zoomLevel
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

	// Zoom level adjustment with mouse wheel
	mx, my := ebiten.CursorPosition()
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
			e.camX = e.origCamX - dx/e.zoomLevel
			e.camY = e.origCamY + dy/e.zoomLevel
		}
	} else {
		e.isPanning = false
	}

	// Panning with arrow keys
	panStep := 16.0 / e.zoomLevel
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

	// Update button hover states
	for i := 0; i < NumIconButtons; i++ {
		bx := UserPanelX + i*IconButtonSize
		by := UserPanelY
		e.buttons[i].Hovered = (mx >= bx && mx < bx+IconButtonSize && my >= by && my < by+IconButtonSize)
	}

	return nil
}

// Draw renders the entire 1280x800 editor mode interface.
func (e *EditorMode) Draw(screen *ebiten.Image) {
	// 1. Draw 960x792 Editing Window
	e.drawEditingWindow(screen)

	// 2. Draw 320x792 User Panel
	e.drawUserPanel(screen)

	// 3. Draw 1280x8 Status Bar
	e.drawStatusBar(screen)
}

// drawEditingWindow renders the editing background and grid lines.
func (e *EditorMode) drawEditingWindow(screen *ebiten.Image) {
	// Fill editing area background with black
	vector.DrawFilledRect(screen, 0, 0, EditingWindowWidth, EditingWindowHeight, gfx.EGABlack, false)

	// Compute visible world bounds
	minWx, maxWy := e.ScreenToWorld(0, 0)
	maxWx, minWy := e.ScreenToWorld(EditingWindowWidth, EditingWindowHeight)

	gridStep := float64(e.gridSize)
	pixelSpacing := gridStep * e.zoomLevel

	// Only draw grid lines if spacing is at least 1 pixel
	if pixelSpacing >= 1.0 {
		gridColor := gfx.EGALightGray

		// Vertical grid lines
		startK := int(math.Floor(minWx / gridStep))
		endK := int(math.Ceil(maxWx / gridStep))

		for k := startK; k <= endK; k++ {
			wx := float64(k) * gridStep
			sx, _ := e.WorldToScreen(wx, 0)
			if sx >= 0 && sx < EditingWindowWidth {
				vector.StrokeLine(screen, float32(sx), 0, float32(sx), float32(EditingWindowHeight), 1.0, gridColor, false)
			}
		}

		// Horizontal grid lines
		startM := int(math.Floor(minWy / gridStep))
		endM := int(math.Ceil(maxWy / gridStep))

		for m := startM; m <= endM; m++ {
			wy := float64(m) * gridStep
			_, sy := e.WorldToScreen(0, wy)
			if sy >= 0 && sy < EditingWindowHeight {
				vector.StrokeLine(screen, 0, float32(sy), float32(EditingWindowWidth), float32(sy), 1.0, gridColor, false)
			}
		}
	}
}

// drawUserPanel renders the 320x792 dark gray panel with 20x1 16x16 icon buttons at the top.
func (e *EditorMode) drawUserPanel(screen *ebiten.Image) {
	// Fill user panel background with dark gray
	vector.DrawFilledRect(screen, UserPanelX, UserPanelY, UserPanelWidth, UserPanelHeight, gfx.EGADarkGray, false)

	// Draw 20x1 array of 16x16 icon buttons
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
		vector.DrawFilledRect(screen, bx+1, by+1, IconButtonSize-2, IconButtonSize-2, btnBg, false)

		// Button border
		vector.StrokeRect(screen, bx, by, IconButtonSize, IconButtonSize, 1.0, btnBorderColor, false)
	}
}

// drawStatusBar renders the 1280x8 dark blue status bar with yellow IWAD/PWAD and right-justified Grid/Zoom text.
func (e *EditorMode) drawStatusBar(screen *ebiten.Image) {
	// Fill status bar background with dark blue
	vector.DrawFilledRect(screen, StatusBarX, StatusBarY, StatusBarWidth, StatusBarHeight, gfx.EGADarkBlue, false)

	if e.font == nil {
		return
	}

	// 1. Far-left: IWAD=[filename]  PWAD=NONE
	iwadName := "NONE"
	if e.wadFile != nil && e.wadFile.Filename() != "" {
		iwadName = e.wadFile.Filename()
	}
	leftText := fmt.Sprintf("IWAD=%s  PWAD=NONE", iwadName)
	e.font.DrawText(screen, leftText, StatusBarX, StatusBarY, gfx.EGABrightYellow)

	// 2. Right-justified: Grid and Zoom settings in fixed-width fields
	rightText := fmt.Sprintf("GRID: %3d  ZOOM: %5.2fx", e.gridSize, e.zoomLevel)
	rightTextWidth, _ := e.font.MeasureText(rightText)
	rightX := StatusBarWidth - rightTextWidth
	if rightX > 0 {
		e.font.DrawText(screen, rightText, rightX, StatusBarY, gfx.EGABrightYellow)
	}
}
