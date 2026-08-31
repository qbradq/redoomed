package platform

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/audio"
	"github.com/qbradq/redoomed/pkg/font"
	"github.com/qbradq/redoomed/pkg/gfx"
	"github.com/qbradq/redoomed/pkg/mode"
	"github.com/qbradq/redoomed/pkg/player"
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

// NewApp creates and initializes a new ReDoomEd App instance using default IWAD discovery.
func NewApp() *App {
	return NewAppWithIWAD("")
}

// NewAppWithIWAD creates and initializes a new ReDoomEd App instance with an optional IWAD path.
// If iwadPath is empty, standard search paths are searched for an available IWAD.
func NewAppWithIWAD(iwadPath string) *App {
	app := &App{}

	// Load the 8x8 fixed-width console font (unscii-8.ttf)
	cf, err := font.NewConsoleFont()
	if err != nil {
		log.Printf("Warning: failed to load console font: %v", err)
	} else {
		app.consoleFont = cf
	}

	// Locate and open IWAD
	var wadToOpen string
	if iwadPath != "" {
		if _, err := os.Stat(iwadPath); err == nil {
			wadToOpen = iwadPath
		} else {
			log.Printf("Warning: specified IWAD %q not found: %v", iwadPath, err)
		}
	} else {
		searchPaths := []string{
			"DoomShareware.wad", "doomshareware.wad", "DOOM1.WAD", "doom1.wad", "DOOM.WAD", "doom.wad",
			"freedoom2.wad", "freedoom1.wad", "DOOM2.WAD", "doom2.wad",
			"../DoomShareware.wad", "../../DoomShareware.wad",
			"../doomshareware.wad", "../../doomshareware.wad",
			"../freedoom2.wad", "../../freedoom2.wad",
		}
		for _, p := range searchPaths {
			if _, err := os.Stat(p); err == nil {
				wadToOpen = p
				break
			}
		}
	}

	if wadToOpen != "" {
		w, err := wad.Open(wadToOpen)
		if err != nil {
			log.Printf("Warning: failed to open WAD %q: %v", wadToOpen, err)
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
	app.gameMode.SetOnLog(func(msg string) {
		app.consoleMode.PrintColored(msg, gfx.EGABrightCyan)
	})

	// Initialize Tengo REPL with exit, start_map, and music handlers
	repl := script.NewREPL(func() {
		app.exitRequested = true
	}, func(msg string) {
		app.consoleMode.PrintColored(msg, gfx.EGABrightRed)
	})
	repl.SetMapDataProvider(func() *wad.MapData {
		if app.gameMode != nil {
			return app.gameMode.MapData()
		}
		return nil
	})
	repl.SetPlayerStatsProvider(func() *player.PlayerStats {
		if app.gameMode != nil {
			return app.gameMode.PlayerStats()
		}
		return nil
	})
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
	repl.SetSetNoClipFunc(func(enabled bool) {
		if app.gameMode != nil {
			app.gameMode.SetNoClip(enabled)
		}
	})

	app.gameMode.SetOnTriggerLineSpecial(func(special, lineID, secID, thingID, tag int) {
		repl.TriggerLineSpecial(special, lineID, secID, thingID, tag)
	})

	app.consoleMode.SetREPL(repl)
	app.currentMode = app.consoleMode

	// Run the game logic main entry point script (autoexec.tengo)
	scriptName := "scripts/autoexec.tengo"
	autoexecScript, err := repl.Cache().GetScript(scriptName)
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

// underlyingMode returns the mode that should be rendered beneath ConsoleMode.
func (a *App) underlyingMode() mode.Mode {
	if a.previousMode != nil && a.previousMode != a.consoleMode {
		return a.previousMode
	}
	if a.gameMode != nil {
		return a.gameMode
	}
	return nil
}

// ToggleConsole toggles between ConsoleMode and the underlying mode (GameMode, EditorMode, etc.).
func (a *App) ToggleConsole() {
	if a.currentMode == a.consoleMode {
		if underlying := a.underlyingMode(); underlying != nil {
			a.currentMode = underlying
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

// PreviousMode returns the previously active application mode, or nil if none.
func (a *App) PreviousMode() mode.Mode {
	return a.previousMode
}

// SetMode switches the active top-level application mode.
func (a *App) SetMode(m mode.Mode) {
	if m == nil {
		panic("cannot set nil application mode")
	}
	if a.currentMode != a.consoleMode && a.currentMode != m {
		a.previousMode = a.currentMode
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
// If ConsoleMode is active, the underlying mode (e.g. GameMode or EditorMode) is rendered beneath it.
func (a *App) Draw(screen *ebiten.Image) {
	if a.currentMode == a.consoleMode {
		if underlying := a.underlyingMode(); underlying != nil {
			underlying.Draw(screen)
		}
		a.consoleMode.Draw(screen)
		return
	}

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
