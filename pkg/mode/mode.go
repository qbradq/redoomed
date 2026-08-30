package mode

import "github.com/hajimehoshi/ebiten/v2"

// Mode represents a top-level application mode (e.g. Console, Game, Menu).
type Mode interface {
	// Update advances the mode state.
	Update() error
	// Draw renders the mode onto the destination screen image.
	Draw(screen *ebiten.Image)
}
