package mode

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

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
	controls := NewGameControlsLayer(func() {
		miniMap.SetVisible(!miniMap.IsVisible())
	})
	hud := NewHUDLayer(stbarImg)
	levelView := NewLevelViewLayer()
	intermission := NewIntermissionLayer(titleImg)

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

	return &GameMode{
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
}

// MapName returns the active map name.
func (g *GameMode) MapName() string {
	return g.mapName
}

// SetMapName updates the active map name and reloads map geometry and entities.
func (g *GameMode) SetMapName(name string) {
	g.mapName = name
	if g.wadFile != nil && name != "" {
		if md, err := g.wadFile.LoadMap(name); err == nil {
			g.miniMapLayer.SetMapData(md)
		}
	}
}

// MapData returns the active map data from the mini map layer, or nil if none loaded.
func (g *GameMode) MapData() *wad.MapData {
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
