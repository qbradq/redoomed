package mode

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// GameBufferWidth is the logical width for the 320x200 game mode rendering.
	GameBufferWidth = 320
	// GameBufferHeight is the logical height for the 320x200 game mode rendering.
	GameBufferHeight = 200
)

// GameMode represents the 3D game rendering and gameplay application mode.
type GameMode struct {
	mapName string
	buffer  *ebiten.Image
	bgColor color.RGBA
}

// NewGameMode creates a new stubbed GameMode instance for the specified map.
func NewGameMode(mapName string) *GameMode {
	return &GameMode{
		mapName: mapName,
		buffer:  ebiten.NewImage(GameBufferWidth, GameBufferHeight),
		bgColor: color.RGBA{R: 10, G: 10, B: 15, A: 255},
	}
}

// MapName returns the active map name.
func (g *GameMode) MapName() string {
	return g.mapName
}

// SetMapName updates the active map name.
func (g *GameMode) SetMapName(name string) {
	g.mapName = name
}

// Update advances the gameplay simulation state.
func (g *GameMode) Update() error {
	return nil
}

// Draw renders the game mode buffer and composites it onto the target screen.
func (g *GameMode) Draw(screen *ebiten.Image) {
	// 1. Clear offscreen game buffer
	g.buffer.Fill(g.bgColor)

	// 2. Composite 320x200 game buffer onto the native 1280x800 screen buffer
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	scaleX := float64(sw) / float64(GameBufferWidth)
	scaleY := float64(sh) / float64(GameBufferHeight)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(g.buffer, op)
}
