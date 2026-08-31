package main

import (
	"flag"
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
	var iwadPath string
	flag.StringVar(&iwadPath, "iwad", "", "Path to the IWAD file to open")
	flag.Parse()

	ebiten.SetWindowSize(defaultWindowWidth, defaultWindowHeight)
	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	app := platform.NewAppWithIWAD(iwadPath)
	if err := ebiten.RunGame(app); err != nil {
		log.Fatalf("Fatal error running ReDoomEd: %v", err)
	}
}
