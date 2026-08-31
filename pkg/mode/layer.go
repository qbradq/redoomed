package mode

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/qbradq/redoomed/pkg/gfx"
	"github.com/qbradq/redoomed/pkg/physics"
	"github.com/qbradq/redoomed/pkg/render"
	"github.com/qbradq/redoomed/pkg/wad"
)

// Layer represents a composable UI or gameplay layer within a Mode stack.
type Layer interface {
	// Name returns the descriptive name of the layer.
	Name() string
	// Update advances the layer state and processes input.
	// Returns true if the input was consumed and should not propagate to lower layers.
	Update() (consumed bool, err error)
	// Draw renders the layer content onto the game buffer.
	Draw(screen *ebiten.Image)
	// IsVisible reports whether the layer is active and should be drawn/updated.
	IsVisible() bool
	// SetVisible updates the visibility state of the layer.
	SetVisible(visible bool)
	// PreventsLowerDrawing reports whether this layer occludes/blocks lower layers from drawing when visible.
	PreventsLowerDrawing() bool
}

// CommonLayer handles engine-level common inputs (e.g. console toggle) and renders nothing.
type CommonLayer struct {
	visible         bool
	onToggleConsole func()
}

// NewCommonLayer creates a new CommonLayer.
func NewCommonLayer(onToggleConsole func()) *CommonLayer {
	return &CommonLayer{
		visible:         true,
		onToggleConsole: onToggleConsole,
	}
}

func (l *CommonLayer) Name() string { return "Common" }

func (l *CommonLayer) Update() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyGraveAccent) {
		if l.onToggleConsole != nil {
			l.onToggleConsole()
		}
		return true, nil
	}
	return false, nil
}

func (l *CommonLayer) Draw(screen *ebiten.Image) {}

func (l *CommonLayer) IsVisible() bool { return l.visible }

func (l *CommonLayer) SetVisible(v bool) { l.visible = v }

func (l *CommonLayer) PreventsLowerDrawing() bool { return false }

// SetOnToggleConsole sets the console toggle callback.
func (l *CommonLayer) SetOnToggleConsole(fn func()) {
	l.onToggleConsole = fn
}

// GameMenuLayer represents the main menu and options menu layer.
type GameMenuLayer struct {
	visible bool
}

// NewGameMenuLayer creates a new GameMenuLayer (hidden by default).
func NewGameMenuLayer() *GameMenuLayer {
	return &GameMenuLayer{
		visible: false,
	}
}

func (l *GameMenuLayer) Name() string { return "Game Menu" }

func (l *GameMenuLayer) Update() (bool, error) {
	if !l.visible {
		return false, nil
	}
	return false, nil
}

func (l *GameMenuLayer) Draw(screen *ebiten.Image) {}

func (l *GameMenuLayer) IsVisible() bool { return l.visible }

func (l *GameMenuLayer) SetVisible(v bool) { l.visible = v }

func (l *GameMenuLayer) PreventsLowerDrawing() bool { return false }

// GameControlsLayer handles gameplay inputs like movement, use actions, and automap toggle.
type GameControlsLayer struct {
	visible         bool
	onToggleMiniMap func()
	onMovePlayer    func(forward, strafe, turn float64)
	onUse           func()
}

// NewGameControlsLayer creates a new GameControlsLayer.
func NewGameControlsLayer(onToggleMiniMap func()) *GameControlsLayer {
	return &GameControlsLayer{
		visible:         true,
		onToggleMiniMap: onToggleMiniMap,
	}
}

func (l *GameControlsLayer) Name() string { return "Game Controls" }

// SetOnToggleMiniMap updates the mini map toggle callback.
func (l *GameControlsLayer) SetOnToggleMiniMap(fn func()) {
	l.onToggleMiniMap = fn
}

// SetOnMovePlayer updates the player movement callback.
func (l *GameControlsLayer) SetOnMovePlayer(fn func(forward, strafe, turn float64)) {
	l.onMovePlayer = fn
}

// SetOnUse updates the player use action callback.
func (l *GameControlsLayer) SetOnUse(fn func()) {
	l.onUse = fn
}

func (l *GameControlsLayer) Update() (bool, error) {
	if !l.visible {
		return false, nil
	}

	consumed := false

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if l.onToggleMiniMap != nil {
			l.onToggleMiniMap()
		}
		return true, nil
	}

	// Use action (E or Spacebar)
	if inpututil.IsKeyJustPressed(ebiten.KeyE) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if l.onUse != nil {
			l.onUse()
		}
		consumed = true
	}

	// Player movement inputs (WASD / Arrows)
	moveSpeed := 3.0
	turnSpeed := 2.5

	if ebiten.IsKeyPressed(ebiten.KeyShift) ||
		ebiten.IsKeyPressed(ebiten.KeyShiftLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		moveSpeed = 6.0
		turnSpeed = 4.0
	}

	var forward, strafe, turn float64

	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		forward += moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		forward -= moveSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyComma) {
		strafe -= moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyPeriod) {
		strafe += moveSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		turn += turnSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		turn -= turnSpeed
	}

	if forward != 0 || strafe != 0 || turn != 0 {
		if l.onMovePlayer != nil {
			l.onMovePlayer(forward, strafe, turn)
		}
		consumed = true
	}

	return consumed, nil
}

func (l *GameControlsLayer) Draw(screen *ebiten.Image) {}

func (l *GameControlsLayer) IsVisible() bool { return l.visible }

func (l *GameControlsLayer) SetVisible(v bool) { l.visible = v }

func (l *GameControlsLayer) PreventsLowerDrawing() bool { return false }

// HUDLayer renders the game's heads-up display (status bar).
type HUDLayer struct {
	visible    bool
	stbarImage *ebiten.Image
}

// NewHUDLayer creates a new HUDLayer with the provided status bar texture.
func NewHUDLayer(stbar *ebiten.Image) *HUDLayer {
	return &HUDLayer{
		visible:    true,
		stbarImage: stbar,
	}
}

func (l *HUDLayer) Name() string { return "HUD" }

func (l *HUDLayer) Update() (bool, error) {
	return false, nil
}

// Draw renders the status bar at the bottom of the 320x200 logical game buffer.
func (l *HUDLayer) Draw(screen *ebiten.Image) {
	if !l.visible || l.stbarImage == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	stbarHeight := l.stbarImage.Bounds().Dy()
	y := GameBufferHeight - stbarHeight
	op.GeoM.Translate(0, float64(y))
	screen.DrawImage(l.stbarImage, op)
}

func (l *HUDLayer) IsVisible() bool { return l.visible }

func (l *HUDLayer) SetVisible(v bool) { l.visible = v }

func (l *HUDLayer) PreventsLowerDrawing() bool { return false }

// SetSTBARImage updates the status bar image texture.
func (l *HUDLayer) SetSTBARImage(img *ebiten.Image) {
	l.stbarImage = img
}

// MiniMapLayer renders the vector line automap from above with player position arrow.
type MiniMapLayer struct {
	visible     bool
	mapData     *wad.MapData
	playerX     float64
	playerY     float64
	playerAngle float64
	hasPlayer   bool
	zoomLevel   float64
}

const (
	minMiniMapZoom = 0.25
	maxMiniMapZoom = 20.0
)

// NewMiniMapLayer creates a new MiniMapLayer (hidden by default, occludes lower layers when visible).
func NewMiniMapLayer(mapData *wad.MapData) *MiniMapLayer {
	layer := &MiniMapLayer{
		visible:   false,
		zoomLevel: 1.0,
	}
	if mapData != nil {
		layer.SetMapData(mapData)
	}
	return layer
}

func (l *MiniMapLayer) Name() string { return "Mini map" }

// ZoomIn multiplies the zoom factor by the given amount.
func (l *MiniMapLayer) ZoomIn(factor float64) {
	if factor <= 0 {
		factor = 1.2
	}
	if l.zoomLevel <= 0 {
		l.zoomLevel = 1.0
	}
	l.zoomLevel *= factor
	if l.zoomLevel > maxMiniMapZoom {
		l.zoomLevel = maxMiniMapZoom
	}
}

// ZoomOut divides the zoom factor by the given amount.
func (l *MiniMapLayer) ZoomOut(factor float64) {
	if factor <= 0 {
		factor = 1.2
	}
	if l.zoomLevel <= 0 {
		l.zoomLevel = 1.0
	}
	l.zoomLevel /= factor
	if l.zoomLevel < minMiniMapZoom {
		l.zoomLevel = minMiniMapZoom
	}
}

// SetZoom sets the explicit zoom level, clamped within bounds.
func (l *MiniMapLayer) SetZoom(z float64) {
	if z < minMiniMapZoom {
		z = minMiniMapZoom
	}
	if z > maxMiniMapZoom {
		z = maxMiniMapZoom
	}
	l.zoomLevel = z
}

// Zoom returns the current zoom level.
func (l *MiniMapLayer) Zoom() float64 {
	if l.zoomLevel <= 0 {
		return 1.0
	}
	return l.zoomLevel
}

func (l *MiniMapLayer) Update() (bool, error) {
	if !l.visible {
		return false, nil
	}

	consumed := false

	// 1. Scroll wheel zooming
	_, wheelY := ebiten.Wheel()
	if wheelY > 0 {
		l.ZoomIn(1.15)
		consumed = true
	} else if wheelY < 0 {
		l.ZoomOut(1.15)
		consumed = true
	}

	// 2. Ctrl+ and Ctrl- zooming
	ctrlPressed := ebiten.IsKeyPressed(ebiten.KeyControl) ||
		ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyControlRight)

	if ctrlPressed {
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
			l.ZoomIn(1.25)
			consumed = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
			l.ZoomOut(1.25)
			consumed = true
		}
	}

	return consumed, nil
}

// SetMapData assigns new map data and initializes the player start position.
func (l *MiniMapLayer) SetMapData(m *wad.MapData) {
	l.mapData = m
	if m != nil {
		if p1, ok := m.Player1Start(); ok {
			l.playerX = float64(p1.X)
			l.playerY = float64(p1.Y)
			l.playerAngle = float64(p1.Angle)
			l.hasPlayer = true
		} else {
			l.hasPlayer = false
		}
	} else {
		l.hasPlayer = false
	}
}

// MapData returns the active map data.
func (l *MiniMapLayer) MapData() *wad.MapData {
	return l.mapData
}

// SetPlayer updates the player's position and viewing angle on the mini map.
func (l *MiniMapLayer) SetPlayer(x, y, angle float64) {
	l.playerX = x
	l.playerY = y
	l.playerAngle = angle
	l.hasPlayer = true
}

// HasPlayer reports whether the player position is set.
func (l *MiniMapLayer) HasPlayer() bool {
	return l.hasPlayer
}

// PlayerPosition returns the player's coordinates and angle on the mini map.
func (l *MiniMapLayer) PlayerPosition() (x, y, angle float64) {
	return l.playerX, l.playerY, l.playerAngle
}

// Draw renders the vector line map and the player directional arrow onto the 320x200 buffer.
func (l *MiniMapLayer) Draw(screen *ebiten.Image) {
	if !l.visible {
		return
	}

	// 1. Fill background with Black
	screen.Fill(gfx.EGABlack)

	if l.mapData == nil || len(l.mapData.Vertexes) == 0 || len(l.mapData.Linedefs) == 0 {
		return
	}

	// 2. Compute map bounds and scaling to fit the 320x168 viewport (leaving space for status bar)
	minX, maxX, minY, maxY := l.mapData.Bounds()
	mapW := float64(maxX - minX)
	mapH := float64(maxY - minY)
	if mapW <= 0 {
		mapW = 1
	}
	if mapH <= 0 {
		mapH = 1
	}

	// Available viewport dimensions
	vpW := 320.0 - 20.0
	vpH := 168.0 - 20.0

	scaleX := vpW / mapW
	scaleY := vpH / mapH
	baseScale := scaleX
	if scaleY < baseScale {
		baseScale = scaleY
	}

	zoom := l.zoomLevel
	if zoom <= 0 {
		zoom = 1.0
	}
	scale := baseScale * zoom

	midX := float64(minX+maxX) / 2.0
	midY := float64(minY+maxY) / 2.0

	vpCenterX := 160.0
	vpCenterY := 84.0

	// Focus position transitions smoothly from map center (at zoom 1.0) towards player position when zoomed in
	focusX := midX
	focusY := midY
	if l.hasPlayer && zoom > 1.0 {
		t := (zoom - 1.0) / 2.0
		if t > 1.0 {
			t = 1.0
		}
		focusX = midX + (l.playerX-midX)*t
		focusY = midY + (l.playerY-midY)*t
	}

	worldToScreen := func(wx, wy float64) (float32, float32) {
		sx := float32(vpCenterX + (wx-focusX)*scale)
		sy := float32(vpCenterY - (wy-focusY)*scale) // Invert Y for screen space
		return sx, sy
	}

	// 3. Draw linedefs respecting flags
	for _, ld := range l.mapData.Linedefs {
		if int(ld.V1) >= len(l.mapData.Vertexes) || int(ld.V2) >= len(l.mapData.Vertexes) {
			continue
		}

		// Don't draw flag (ML_DONTDRAW): skip completely
		if ld.Flags&wad.LinedefDontDraw != 0 {
			continue
		}

		// Determine line color based on flags:
		// - Secret linedefs (0x0020) and 1-sided walls: Red
		// - Two-sided linedefs (0x0004): Brown / Dark Gray
		clr := gfx.EGARed
		if ld.Flags&wad.LinedefSecret != 0 {
			clr = gfx.EGARed
		} else if ld.Flags&wad.LinedefTwoSided != 0 {
			clr = gfx.EGABrown
		}

		v1 := l.mapData.Vertexes[ld.V1]
		v2 := l.mapData.Vertexes[ld.V2]

		x1, y1 := worldToScreen(float64(v1.X), float64(v1.Y))
		x2, y2 := worldToScreen(float64(v2.X), float64(v2.Y))

		vector.StrokeLine(screen, x1, y1, x2, y2, 1.0, clr, false)
	}

	// 4. Draw player green arrow
	if l.hasPlayer {
		px, py := worldToScreen(l.playerX, l.playerY)

		// Doom angle: 0=East, 90=North, 180=West, 270=South
		rad := l.playerAngle * math.Pi / 180.0
		dirX := math.Cos(rad)
		dirY := -math.Sin(rad) // Invert Y

		perpX := -dirY
		perpY := dirX

		arrowLen := 6.0
		arrowHalfW := 3.5

		tipX := px + float32(dirX*arrowLen)
		tipY := py + float32(dirY*arrowLen)

		leftX := px - float32(dirX*(arrowLen*0.5)) + float32(perpX*arrowHalfW)
		leftY := py - float32(dirY*(arrowLen*0.5)) + float32(perpY*arrowHalfW)

		rightX := px - float32(dirX*(arrowLen*0.5)) - float32(perpX*arrowHalfW)
		rightY := py - float32(dirY*(arrowLen*0.5)) - float32(perpX*arrowHalfW)

		backX := px - float32(dirX*(arrowLen*0.2))
		backY := py - float32(dirY*(arrowLen*0.2))

		green := gfx.EGABrightGreen
		vector.StrokeLine(screen, leftX, leftY, tipX, tipY, 1.2, green, false)
		vector.StrokeLine(screen, rightX, rightY, tipX, tipY, 1.2, green, false)
		vector.StrokeLine(screen, leftX, leftY, backX, backY, 1.2, green, false)
		vector.StrokeLine(screen, rightX, rightY, backX, backY, 1.2, green, false)
	}
}

func (l *MiniMapLayer) IsVisible() bool { return l.visible }

func (l *MiniMapLayer) SetVisible(v bool) { l.visible = v }

func (l *MiniMapLayer) PreventsLowerDrawing() bool { return true }

// LevelViewLayer renders the 2.5D software-rendered level view.
type LevelViewLayer struct {
	visible     bool
	mapData     *wad.MapData
	renderer    *render.Renderer
	cam         render.Camera
	playerActor *physics.Actor
	hasPlayer   bool
}

// NewLevelViewLayer creates a new LevelViewLayer (hidden by default, occludes lower layers when visible).
func NewLevelViewLayer() *LevelViewLayer {
	return &LevelViewLayer{
		visible:     false,
		renderer:    render.NewRenderer(GameBufferWidth, 168, GameBufferHeight),
		playerActor: physics.NewPlayerActor(0, 0, render.DefaultPlayerEyeHeight, 0),
		cam: render.Camera{
			EyeHeight: render.DefaultPlayerEyeHeight,
		},
	}
}

func (l *LevelViewLayer) Name() string { return "Level view" }

// SetMapData updates the active map data and initializes the camera from Player 1 start.
func (l *LevelViewLayer) SetMapData(m *wad.MapData) {
	l.mapData = m
	if m != nil {
		if p1, ok := m.Player1Start(); ok {
			l.playerActor = physics.NewPlayerActor(float64(p1.X), float64(p1.Y), 0, float64(p1.Angle))
			l.hasPlayer = true
			if sec, ok := m.SectorAt(l.playerActor.X, l.playerActor.Y); ok && sec != nil {
				l.playerActor.FloorZ = float64(sec.FloorHeight)
				l.playerActor.CeilingZ = float64(sec.CeilingHeight)
			}
			l.playerActor.Z = l.playerActor.FloorZ + l.playerActor.EyeHeight
			l.cam.X = l.playerActor.X
			l.cam.Y = l.playerActor.Y
			l.cam.Z = l.playerActor.Z
			l.cam.Angle = l.playerActor.Angle
		} else {
			l.hasPlayer = false
		}
	} else {
		l.hasPlayer = false
	}
}

// MapData returns the active map data.
func (l *LevelViewLayer) MapData() *wad.MapData {
	return l.mapData
}

// Camera returns a pointer to the level view camera.
func (l *LevelViewLayer) Camera() *render.Camera {
	return &l.cam
}

// Player returns the underlying physics actor for the player.
func (l *LevelViewLayer) Player() *physics.Actor {
	return l.playerActor
}

// SetCamera updates the camera position and angle.
func (l *LevelViewLayer) SetCamera(x, y, z, angle float64) {
	l.cam.X = x
	l.cam.Y = y
	l.cam.Z = z
	l.cam.Angle = angle
	l.hasPlayer = true
	if l.playerActor == nil {
		l.playerActor = physics.NewPlayerActor(x, y, z, angle)
	} else {
		l.playerActor.X = x
		l.playerActor.Y = y
		l.playerActor.Z = z
		l.playerActor.Angle = angle
		l.playerActor.FloorZ = z - l.playerActor.EyeHeight
	}
}

// MovePlayer moves and rotates the player using physics collision detection and sliding.
func (l *LevelViewLayer) MovePlayer(forward, strafe, turn float64) {
	if !l.hasPlayer {
		return
	}
	if l.playerActor == nil {
		l.playerActor = physics.NewPlayerActor(l.cam.X, l.cam.Y, l.cam.Z, l.cam.Angle)
	}

	physics.Move(l.mapData, l.playerActor, forward, strafe, turn)

	l.cam.X = l.playerActor.X
	l.cam.Y = l.playerActor.Y
	l.cam.Z = l.playerActor.Z
	l.cam.Angle = l.playerActor.Angle
}

func (l *LevelViewLayer) Update() (bool, error) {
	if !l.visible {
		return false, nil
	}
	return false, nil
}

// Draw renders the 2.5D view onto the 320x200 screen buffer.
func (l *LevelViewLayer) Draw(screen *ebiten.Image) {
	if !l.visible || l.mapData == nil {
		return
	}
	l.renderer.Render(screen, l.mapData, &l.cam)
}

func (l *LevelViewLayer) IsVisible() bool { return l.visible }

func (l *LevelViewLayer) SetVisible(v bool) { l.visible = v }

func (l *LevelViewLayer) PreventsLowerDrawing() bool { return true }

// IntermissionLayer renders full-screen background graphics (title screen / intermission screen).
type IntermissionLayer struct {
	visible    bool
	titleImage *ebiten.Image
}

// NewIntermissionLayer creates a new IntermissionLayer with the provided full screen image.
func NewIntermissionLayer(titleImage *ebiten.Image) *IntermissionLayer {
	return &IntermissionLayer{
		visible:    true,
		titleImage: titleImage,
	}
}

func (l *IntermissionLayer) Name() string { return "Intermission Screen" }

func (l *IntermissionLayer) Update() (bool, error) {
	return false, nil
}

// Draw renders the title/intermission graphic to cover the 320x200 logical game buffer.
func (l *IntermissionLayer) Draw(screen *ebiten.Image) {
	if !l.visible || l.titleImage == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(l.titleImage, op)
}

func (l *IntermissionLayer) IsVisible() bool { return l.visible }

func (l *IntermissionLayer) SetVisible(v bool) { l.visible = v }

func (l *IntermissionLayer) PreventsLowerDrawing() bool { return false }

// SetTitleImage updates the title image texture.
func (l *IntermissionLayer) SetTitleImage(img *ebiten.Image) {
	l.titleImage = img
}
