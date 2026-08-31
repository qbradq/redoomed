package mode

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/physics"
	"github.com/qbradq/redoomed/pkg/player"
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
	onItemPickup         func(item *wad.ItemEntity, msg string)

	playerStats *player.PlayerStats

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
	var mapData *wad.MapData

	playerStats := player.NewPlayerStats()
	hudAssets := NewHUDAssets(w)

	if w != nil {
		if img, err := w.GetPatchImage("TITLEPIC"); err == nil {
			titleImg = img
		} else if img, err := w.GetPatchImage("INTERPIC"); err == nil {
			titleImg = img
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
	hud := NewHUDLayerWithAssets(hudAssets, playerStats)
	intermission := NewIntermissionLayer(titleImg)

	if mapData != nil {
		levelView.SetMapData(mapData)
		miniMap.SetItems(levelView.Items())
		levelView.SetVisible(true)
		intermission.SetVisible(false)
	}

	g := &GameMode{
		mapName:           mapName,
		wadFile:           w,
		buffer:            ebiten.NewImage(GameBufferWidth, GameBufferHeight),
		bgColor:           color.RGBA{R: 0, G: 0, B: 0, A: 255},
		playerStats:       playerStats,
		commonLayer:       common,
		gameMenuLayer:     menu,
		hudLayer:          hud,
		miniMapLayer:      miniMap,
		levelViewLayer:    levelView,
		intermissionLayer: intermission,
	}

	controls := NewGameControlsLayer(func() {
		miniMap.SetVisible(!miniMap.IsVisible())
	})

	controls.SetOnMovePlayer(func(forward, strafe, turn float64) {
		levelView.MovePlayer(forward, strafe, turn)
		g.CheckItemPickups()
		cam := levelView.Camera()
		if cam != nil {
			miniMap.SetPlayer(cam.X, cam.Y, cam.Angle)
		}
	})

	controls.SetOnSelectWeaponSlot(func(slot int) {
		if playerStats != nil {
			playerStats.SelectSlot(slot)
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
	g.layers = layers
	g.gameControlsLayer = controls

	controls.SetOnUse(func() {
		g.Use()
	})

	levelView.SetOnCrossLinedef(func(lineIdx int, ld *wad.Linedef) {
		if ld != nil && ld.Special != 0 {
			md := g.MapData()
			msg := fmt.Sprintf("Cross: line %d (Special: %d, Tag: %d, Flags: 0x%04X)", lineIdx, ld.Special, ld.Tag, ld.Flags)
			log.Print(msg)
			if g.onLog != nil {
				g.onLog(msg)
			}
			if g.onTriggerLineSpecial != nil {
				secID := -1
				if ld.RightSide != 0xFFFF && md != nil && int(ld.RightSide) < len(md.Sidedefs) {
					secID = int(md.Sidedefs[ld.RightSide].Sector)
				}
				thingID := 0
				g.onTriggerLineSpecial(int(ld.Special), lineIdx, secID, thingID, int(ld.Tag))
			}
		}
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
	if g.playerStats != nil {
		g.playerStats.Reset()
	}
	if g.wadFile != nil && name != "" {
		if md, err := g.wadFile.LoadMap(name); err == nil {
			g.miniMapLayer.SetMapData(md)
			g.levelViewLayer.SetMapData(md)
			g.miniMapLayer.SetItems(g.levelViewLayer.Items())
			g.levelViewLayer.SetVisible(true)
			g.miniMapLayer.SetVisible(false)
			g.intermissionLayer.SetVisible(false)
			if g.gameControlsLayer != nil {
				g.gameControlsLayer.ResetMouse()
			}
			g.initMapSpecials(md)
		}
	}
}

// initMapSpecials launches any ambient map line specials (such as Special 48 scrolling textures).
func (g *GameMode) initMapSpecials(md *wad.MapData) {
	if md == nil || g.onTriggerLineSpecial == nil {
		return
	}
	for i := range md.Linedefs {
		ld := &md.Linedefs[i]
		if ld.Special == 48 {
			secID := -1
			if ld.RightSide != 0xFFFF && int(ld.RightSide) < len(md.Sidedefs) {
				secID = int(md.Sidedefs[ld.RightSide].Sector)
			}
			g.onTriggerLineSpecial(int(ld.Special), i, secID, 0, int(ld.Tag))
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

// PlayerStats returns the active player statistics instance.
func (g *GameMode) PlayerStats() *player.PlayerStats {
	return g.playerStats
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

// Items returns the current active map items tracked by the engine.
func (g *GameMode) Items() []*wad.ItemEntity {
	if g.levelViewLayer != nil {
		return g.levelViewLayer.Items()
	}
	if g.miniMapLayer != nil {
		return g.miniMapLayer.Items()
	}
	return nil
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
	if fn != nil && g.MapData() != nil {
		g.initMapSpecials(g.MapData())
	}
}

// SetOnItemPickup sets the callback invoked when the player collects an item.
func (g *GameMode) SetOnItemPickup(fn func(item *wad.ItemEntity, msg string)) {
	g.onItemPickup = fn
}

// SetNoClip toggles noclip mode for the player in the level view layer.
func (g *GameMode) SetNoClip(enabled bool) {
	if g.levelViewLayer != nil {
		g.levelViewLayer.SetNoClip(enabled)
	}
}

// NoClip reports whether noclip mode is currently enabled.
func (g *GameMode) NoClip() bool {
	if g.levelViewLayer != nil {
		return g.levelViewLayer.NoClip()
	}
	return false
}

// CheckItemPickups checks whether the player touches any uncollected items, applies their benefits, and notifies listeners.
func (g *GameMode) CheckItemPickups() []*wad.ItemEntity {
	if g.playerStats == nil || g.levelViewLayer == nil {
		return nil
	}
	actor := g.levelViewLayer.Player()
	if actor == nil {
		return nil
	}

	var pickedUp []*wad.ItemEntity
	items := g.levelViewLayer.Items()
	for _, item := range items {
		if item == nil || item.Collected {
			continue
		}
		if physics.CheckItemTouch(actor, item) {
			if ok, msg := g.playerStats.TryPickupItem(item.Def.Type); ok {
				item.Collected = true
				pickedUp = append(pickedUp, item)
				if msg != "" {
					log.Print(msg)
					if g.onLog != nil {
						g.onLog(msg)
					}
					if g.onItemPickup != nil {
						g.onItemPickup(item, msg)
					}
				}
			}
		}
	}
	return pickedUp
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
	g.CheckItemPickups()
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
