package mode

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

// GameControlsLayer handles gameplay inputs like movement and weapon selection.
type GameControlsLayer struct {
	visible bool
}

// NewGameControlsLayer creates a new GameControlsLayer.
func NewGameControlsLayer() *GameControlsLayer {
	return &GameControlsLayer{
		visible: true,
	}
}

func (l *GameControlsLayer) Name() string { return "Game Controls" }

func (l *GameControlsLayer) Update() (bool, error) {
	return false, nil
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

// MiniMapLayer renders the vector line automap from above.
type MiniMapLayer struct {
	visible bool
}

// NewMiniMapLayer creates a new MiniMapLayer (hidden by default, occludes lower layers when visible).
func NewMiniMapLayer() *MiniMapLayer {
	return &MiniMapLayer{
		visible: false,
	}
}

func (l *MiniMapLayer) Name() string { return "Mini map" }

func (l *MiniMapLayer) Update() (bool, error) {
	if !l.visible {
		return false, nil
	}
	return false, nil
}

func (l *MiniMapLayer) Draw(screen *ebiten.Image) {}

func (l *MiniMapLayer) IsVisible() bool { return l.visible }

func (l *MiniMapLayer) SetVisible(v bool) { l.visible = v }

func (l *MiniMapLayer) PreventsLowerDrawing() bool { return true }

// LevelViewLayer renders the 2.5D software-rendered level view.
type LevelViewLayer struct {
	visible bool
}

// NewLevelViewLayer creates a new LevelViewLayer (hidden by default, occludes lower layers when visible).
func NewLevelViewLayer() *LevelViewLayer {
	return &LevelViewLayer{
		visible: false,
	}
}

func (l *LevelViewLayer) Name() string { return "Level view" }

func (l *LevelViewLayer) Update() (bool, error) {
	if !l.visible {
		return false, nil
	}
	return false, nil
}

func (l *LevelViewLayer) Draw(screen *ebiten.Image) {}

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
