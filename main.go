package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/platform"
)

const (
	defaultWindowWidth  = 1280
	defaultWindowHeight = 800
	windowTitle         = "ReDoomEd"
)

func main() {
	ebiten.SetWindowSize(defaultWindowWidth, defaultWindowHeight)
	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	app := platform.NewApp()
	if err := ebiten.RunGame(app); err != nil {
		log.Fatalf("Fatal error running ReDoomEd: %v", err)
	}
}
