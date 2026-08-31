package mode

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/physics"
	"github.com/qbradq/redoomed/pkg/wad"
)

const (
	// GameBufferWidth is the logical width for the 320x200 game mode rendering.
	GameBufferWidth = 320
	// GameBufferHeight is the logical height for the 320x200 game mode rendering.
	GameBufferHeight = 200
)

// GameMode represents the composite game application mode composed of a UI/rendering layer stack.
type GameMode struct {
	mapName string
	wadFile *wad.WAD
	buffer  *ebiten.Image
	bgColor color.RGBA
	onLog   func(string)
	onTriggerLineSpecial func(special, lineID, secID, thingID, tag int)

	// Ordered top to bottom
	layers []Layer

	// Direct layer handles
	commonLayer       *CommonLayer
	gameMenuLayer     *GameMenuLayer
	gameControlsLayer *GameControlsLayer
	hudLayer          *HUDLayer
	miniMapLayer      *MiniMapLayer
	levelViewLayer    *LevelViewLayer
	intermissionLayer *IntermissionLayer
}

// NewGameMode creates a new GameMode instance with the 7-layer stack.
func NewGameMode(mapName string, w *wad.WAD, onToggleConsole func()) *GameMode {
	var titleImg *ebiten.Image
	var stbarImg *ebiten.Image
	var mapData *wad.MapData

	if w != nil {
		if img, err := w.GetPatchImage("TITLEPIC"); err == nil {
			titleImg = img
		} else if img, err := w.GetPatchImage("INTERPIC"); err == nil {
			titleImg = img
		}

		if img, err := w.GetPatchImage("STBAR"); err == nil {
			stbarImg = img
		}

		if mapName != "" {
			if md, err := w.LoadMap(mapName); err == nil {
				mapData = md
			}
		}
	}

	common := NewCommonLayer(onToggleConsole)
	menu := NewGameMenuLayer()
	miniMap := NewMiniMapLayer(mapData)
	levelView := NewLevelViewLayer()
	hud := NewHUDLayer(stbarImg)
	intermission := NewIntermissionLayer(titleImg)

	if mapData != nil {
		levelView.SetMapData(mapData)
		levelView.SetVisible(true)
		intermission.SetVisible(false)
	}

	controls := NewGameControlsLayer(func() {
		miniMap.SetVisible(!miniMap.IsVisible())
	})

	controls.SetOnMovePlayer(func(forward, strafe, turn float64) {
		levelView.MovePlayer(forward, strafe, turn)
		cam := levelView.Camera()
		if cam != nil {
			miniMap.SetPlayer(cam.X, cam.Y, cam.Angle)
		}
	})

	// Layer stack ordered from top to bottom
	layers := []Layer{
		common,
		menu,
		controls,
		hud,
		miniMap,
		levelView,
		intermission,
	}

	g := &GameMode{
		mapName:           mapName,
		wadFile:           w,
		buffer:            ebiten.NewImage(GameBufferWidth, GameBufferHeight),
		bgColor:           color.RGBA{R: 0, G: 0, B: 0, A: 255},
		layers:            layers,
		commonLayer:       common,
		gameMenuLayer:     menu,
		gameControlsLayer: controls,
		hudLayer:          hud,
		miniMapLayer:      miniMap,
		levelViewLayer:    levelView,
		intermissionLayer: intermission,
	}

	controls.SetOnUse(func() {
		g.Use()
	})

	return g
}

// MapName returns the active map name.
func (g *GameMode) MapName() string {
	return g.mapName
}

// SetMapName updates the active map name and reloads map geometry, BSP, and textures.
func (g *GameMode) SetMapName(name string) {
	g.mapName = name
	if g.wadFile != nil && name != "" {
		if md, err := g.wadFile.LoadMap(name); err == nil {
			g.miniMapLayer.SetMapData(md)
			g.levelViewLayer.SetMapData(md)
			g.levelViewLayer.SetVisible(true)
			g.miniMapLayer.SetVisible(false)
			g.intermissionLayer.SetVisible(false)
		}
	}
}

// MapData returns the active map data from the level view layer (or mini map layer), or nil if none loaded.
func (g *GameMode) MapData() *wad.MapData {
	if g.levelViewLayer != nil && g.levelViewLayer.MapData() != nil {
		return g.levelViewLayer.MapData()
	}
	if g.miniMapLayer != nil {
		return g.miniMapLayer.MapData()
	}
	return nil
}

// Layers returns the layer stack from top to bottom.
func (g *GameMode) Layers() []Layer {
	return g.layers
}

// CommonLayer returns the CommonLayer instance.
func (g *GameMode) CommonLayer() *CommonLayer {
	return g.commonLayer
}

// GameMenuLayer returns the GameMenuLayer instance.
func (g *GameMode) GameMenuLayer() *GameMenuLayer {
	return g.gameMenuLayer
}

// GameControlsLayer returns the GameControlsLayer instance.
func (g *GameMode) GameControlsLayer() *GameControlsLayer {
	return g.gameControlsLayer
}

// HUDLayer returns the HUDLayer instance.
func (g *GameMode) HUDLayer() *HUDLayer {
	return g.hudLayer
}

// MiniMapLayer returns the MiniMapLayer instance.
func (g *GameMode) MiniMapLayer() *MiniMapLayer {
	return g.miniMapLayer
}

// LevelViewLayer returns the LevelViewLayer instance.
func (g *GameMode) LevelViewLayer() *LevelViewLayer {
	return g.levelViewLayer
}

// IntermissionLayer returns the IntermissionLayer instance.
func (g *GameMode) IntermissionLayer() *IntermissionLayer {
	return g.intermissionLayer
}

// SetOnToggleConsole updates the console toggle handler in the common layer.
func (g *GameMode) SetOnToggleConsole(fn func()) {
	if g.commonLayer != nil {
		g.commonLayer.SetOnToggleConsole(fn)
	}
}

// SetOnLog sets the callback for logging game events (e.g. debug messages).
func (g *GameMode) SetOnLog(fn func(string)) {
	g.onLog = fn
}

// SetOnTriggerLineSpecial sets the callback when a line special is triggered.
func (g *GameMode) SetOnTriggerLineSpecial(fn func(special, lineID, secID, thingID, tag int)) {
	g.onTriggerLineSpecial = fn
}

// Use triggers a line interaction check in front of the player, prints a debug message,
// and invokes any associated line special function.
func (g *GameMode) Use() (int, *wad.Linedef, bool) {
	mapData := g.MapData()
	if mapData == nil || g.levelViewLayer == nil {
		return -1, nil, false
	}
	actor := g.levelViewLayer.Player()
	if actor == nil {
		return -1, nil, false
	}

	lineIdx, ld, dist, hit := physics.UseLine(mapData, actor, physics.DefaultUseRange)
	if hit && ld != nil {
		msg := fmt.Sprintf("Use: line %d (Special: %d, Tag: %d, Flags: 0x%04X, Dist: %.1f)", lineIdx, ld.Special, ld.Tag, ld.Flags, dist)
		log.Print(msg)
		if g.onLog != nil {
			g.onLog(msg)
		}
		if ld.Special != 0 && g.onTriggerLineSpecial != nil {
			secID := -1
			if ld.RightSide != 0xFFFF && int(ld.RightSide) < len(mapData.Sidedefs) {
				secID = int(mapData.Sidedefs[ld.RightSide].Sector)
			}
			thingID := 0
			g.onTriggerLineSpecial(int(ld.Special), lineIdx, secID, thingID, int(ld.Tag))
		}
		return lineIdx, ld, true
	}

	msg := "Use: no line in range"
	log.Print(msg)
	if g.onLog != nil {
		g.onLog(msg)
	}
	return -1, nil, false
}

// Update advances the gameplay simulation state and propagates input top-to-bottom.
func (g *GameMode) Update() error {
	consumed := false
	for _, layer := range g.layers {
		if !layer.IsVisible() {
			continue
		}
		if consumed {
			continue
		}
		c, err := layer.Update()
		if err != nil {
			return err
		}
		if c {
			consumed = true
		}
	}
	return nil
}

// Draw renders the layer stack buffer and composites it onto the target screen.
func (g *GameMode) Draw(screen *ebiten.Image) {
	// 1. Clear offscreen game buffer
	g.buffer.Fill(g.bgColor)

	// 2. Determine lowest layer index to draw based on occlusion.
	// If a visible layer prevents lower drawing, all layers below it are skipped.
	lowestDrawnIdx := len(g.layers) - 1
	for i := 0; i < len(g.layers); i++ {
		l := g.layers[i]
		if l.IsVisible() && l.PreventsLowerDrawing() {
			lowestDrawnIdx = i
			break
		}
	}

	// 3. Render from bottom (lowestDrawnIdx) up to top (0)
	for i := lowestDrawnIdx; i >= 0; i-- {
		l := g.layers[i]
		if l.IsVisible() {
			l.Draw(g.buffer)
		}
	}

	// 4. Composite 320x200 game buffer onto the native 1280x800 screen buffer
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	scaleX := float64(sw) / float64(GameBufferWidth)
	scaleY := float64(sh) / float64(GameBufferHeight)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(g.buffer, op)
}
