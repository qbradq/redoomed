package platform

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/audio"
	"github.com/qbradq/redoomed/pkg/data"
	"github.com/qbradq/redoomed/pkg/font"
	"github.com/qbradq/redoomed/pkg/gfx"
	"github.com/qbradq/redoomed/pkg/mode"
	"github.com/qbradq/redoomed/pkg/script"
	"github.com/qbradq/redoomed/pkg/wad"
)

const (
	// ScreenWidth is the native logical screen width for ReDoomEd.
	ScreenWidth = 1280
	// ScreenHeight is the native logical screen height for ReDoomEd.
	ScreenHeight = 800
)

// App implements ebiten.Game to drive the ReDoomEd engine platform lifecycle.
type App struct {
	wadFile       *wad.WAD
	hudFont       *wad.HUDFont
	consoleFont   *font.ConsoleFont
	musicMgr      *audio.MusicManager
	consoleMode   *mode.ConsoleMode
	gameMode      *mode.GameMode
	currentMode   mode.Mode
	previousMode  mode.Mode
	exitRequested bool
}

// NewApp creates and initializes a new ReDoomEd App instance.
func NewApp() *App {
	app := &App{}

	// Load the 8x8 fixed-width console font (unscii-8.ttf)
	cf, err := font.NewConsoleFont()
	if err != nil {
		log.Printf("Warning: failed to load console font: %v", err)
	} else {
		app.consoleFont = cf
	}

	// Locate and open default IWAD if available (retained for future gameplay / status bar rendering)
	searchPaths := []string{"freedoom2.wad", "../freedoom2.wad", "../../freedoom2.wad"}
	var wadPath string
	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			wadPath = p
			break
		}
	}

	if wadPath != "" {
		w, err := wad.Open(wadPath)
		if err != nil {
			log.Printf("Warning: failed to open WAD %q: %v", wadPath, err)
		} else {
			app.wadFile = w
			hf, err := wad.NewHUDFont(w)
			if err != nil {
				log.Printf("Warning: failed to load HUD font from WAD: %v", err)
			} else {
				app.hudFont = hf
			}
		}
	} else {
		log.Println("Warning: no IWAD found in standard search paths")
	}

	// Initialize background music manager with loaded WAD
	app.musicMgr = audio.NewMusicManager(app.wadFile)

	// Initialize with Quake-style console mode using the 8x8 unscii font
	app.consoleMode = mode.NewConsoleMode(app.consoleFont)
	app.consoleMode.SetOnClose(func() {
		app.ToggleConsole()
	})

	// Initialize GameMode with 7-layer stack
	app.gameMode = mode.NewGameMode("", app.wadFile, func() {
		app.ToggleConsole()
	})

	// Initialize Tengo REPL with exit, start_map, and music handlers
	repl := script.NewREPL(func() {
		app.exitRequested = true
	}, nil)
	repl.SetStartMapFunc(func(mapName string) {
		app.StartMap(mapName)
	})
	repl.SetPlayMusicFunc(func(track string) {
		if err := app.musicMgr.PlayMusic(track); err != nil {
			log.Printf("Error playing music track %s: %v", track, err)
		}
	})
	repl.SetStopMusicFunc(func() {
		app.musicMgr.Stop()
	})
	repl.SetSetMusicVolumeFunc(func(v float64) {
		app.musicMgr.SetVolume(v)
	})
	repl.SetGetMusicTrackFunc(func() string {
		return app.musicMgr.CurrentTrack()
	})

	app.consoleMode.SetREPL(repl)
	app.currentMode = app.consoleMode

	// Run the game logic main entry point script (autoexec.tengo)
	scriptName := "scripts/autoexec.tengo"
	autoexecScript, err := data.FS.ReadFile(scriptName)
	if err != nil {
		app.consoleMode.PrintColored(fmt.Sprintf("Failed to read %s: %v", scriptName, err), gfx.EGABrightRed)
		log.Printf("Warning: failed to read %s: %v", scriptName, err)
	} else {
		if _, err := repl.EvalScript("autoexec.tengo", string(autoexecScript)); err != nil {
			app.consoleMode.PrintColored(err.Error(), gfx.EGABrightRed)
			log.Printf("Error executing %s: %v", "autoexec.tengo", err)
		}
	}

	return app
}

// ToggleConsole toggles between ConsoleMode and GameMode (or the previous mode).
func (a *App) ToggleConsole() {
	if a.currentMode == a.consoleMode {
		if a.previousMode != nil {
			a.currentMode = a.previousMode
		} else if a.gameMode != nil {
			a.currentMode = a.gameMode
		}
	} else {
		a.previousMode = a.currentMode
		a.currentMode = a.consoleMode
	}
}

// StartMap enters the game mode for the given map name, starts map music, and hides the console.
func (a *App) StartMap(mapName string) {
	if a.gameMode == nil {
		a.gameMode = mode.NewGameMode(mapName, a.wadFile, func() {
			a.ToggleConsole()
		})
	} else {
		a.gameMode.SetMapName(mapName)
	}

	if a.musicMgr != nil && mapName != "" {
		if err := a.musicMgr.PlayMapMusic(mapName); err != nil {
			log.Printf("Warning: failed to play music for %s: %v", mapName, err)
		}
	}

	a.SetMode(a.gameMode)
}

// MusicManager returns the active MusicManager instance.
func (a *App) MusicManager() *audio.MusicManager {
	return a.musicMgr
}

// GameMode returns the active GameMode, or nil if not initialized.
func (a *App) GameMode() *mode.GameMode {
	return a.gameMode
}

// ConsoleMode returns the ConsoleMode instance.
func (a *App) ConsoleMode() *mode.ConsoleMode {
	return a.consoleMode
}

// SetMode switches the active top-level application mode.
func (a *App) SetMode(m mode.Mode) {
	if m == nil {
		panic("cannot set nil application mode")
	}
	a.currentMode = m
}

// CurrentMode returns the active application mode.
func (a *App) CurrentMode() mode.Mode {
	return a.currentMode
}

// ConsoleFont returns the loaded 8x8 console font, or nil if unavailable.
func (a *App) ConsoleFont() *font.ConsoleFont {
	return a.consoleFont
}

// HUDFont returns the loaded HUD message font, or nil if unavailable.
func (a *App) HUDFont() *wad.HUDFont {
	return a.hudFont
}

// WAD returns the active WAD container, or nil if none loaded.
func (a *App) WAD() *wad.WAD {
	return a.wadFile
}

// Update advances the game and active mode state.
func (a *App) Update() error {
	if a.exitRequested {
		return ebiten.Termination
	}
	if a.currentMode != nil {
		return a.currentMode.Update()
	}
	return nil
}

// Draw renders the active mode onto the screen.
func (a *App) Draw(screen *ebiten.Image) {
	if a.currentMode != nil {
		a.currentMode.Draw(screen)
		return
	}
	// Fallback clear if no mode is active
	_ = fmt.Sprint()
}

// Layout takes the outside window dimensions and returns the native logical screen dimensions.
func (a *App) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}
