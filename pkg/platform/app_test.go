package platform

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/gfx"
	"github.com/qbradq/redoomed/pkg/mode"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("expected NewApp() to return non-nil App instance")
	}

	w, h := app.Layout(1280, 800)
	if w != ScreenWidth || h != ScreenHeight {
		t.Errorf("Layout(1280, 800) = (%d, %d); want (%d, %d)", w, h, ScreenWidth, ScreenHeight)
	}

	if app.CurrentMode() == nil {
		t.Error("expected initial application mode to be set")
	}

	if app.ConsoleFont() == nil {
		t.Error("expected console font (unscii-8) to be loaded")
	}

	// Autoexec script runs game.StartMap("MAP01"), so active mode is GameMode
	gameMode, ok := app.CurrentMode().(*mode.GameMode)
	if !ok || gameMode == nil {
		t.Fatalf("expected current mode to be *mode.GameMode after autoexec, got %T", app.CurrentMode())
	}
	if gameMode.MapName() != "MAP01" {
		t.Errorf("expected map name 'MAP01', got %q", gameMode.MapName())
	}

	console := app.ConsoleMode()
	if console == nil {
		t.Fatal("expected app.ConsoleMode() to be non-nil")
	}

	// Verify autoexec.tengo output lines were printed to console
	if len(console.History()) < 4 {
		t.Fatalf("expected at least 4 history lines from autoexec.tengo, got %d", len(console.History()))
	}
	if console.History()[0].Text != "Welcome to ReDoomEd!" {
		t.Errorf("expected first line 'Welcome to ReDoomEd!', got %q", console.History()[0].Text)
	}

	// If autoexec.tengo produced an error, verify it was recorded in bright red
	lastLine := console.History()[len(console.History())-1]
	if len(console.History()) > 4 {
		if lastLine.Color != gfx.EGABrightRed {
			t.Errorf("expected error line to be EGABrightRed, got %+v", lastLine.Color)
		}
	}

	if err := app.Update(); err != nil {
		t.Errorf("Update() returned error: %v", err)
	}

	screen := ebiten.NewImage(1280, 800)
	app.Draw(screen)
}

func TestAppToggleConsole(t *testing.T) {
	app := NewApp()

	// Initial mode is GameMode after autoexec
	if _, ok := app.CurrentMode().(*mode.GameMode); !ok {
		t.Fatalf("expected initial mode to be GameMode, got %T", app.CurrentMode())
	}

	// Toggle to Console
	app.ToggleConsole()
	if _, ok := app.CurrentMode().(*mode.ConsoleMode); !ok {
		t.Fatalf("expected mode after ToggleConsole to be ConsoleMode, got %T", app.CurrentMode())
	}

	// Toggle back to GameMode
	app.ToggleConsole()
	if _, ok := app.CurrentMode().(*mode.GameMode); !ok {
		t.Fatalf("expected mode after second ToggleConsole to be GameMode, got %T", app.CurrentMode())
	}
}

func TestAppExitCommand(t *testing.T) {
	app := NewApp()

	console := app.ConsoleMode()
	if console == nil {
		t.Fatal("expected ConsoleMode to be non-nil")
	}

	repl := console.REPL()
	if repl == nil {
		t.Fatal("expected console REPL to be initialized")
	}

	// Execute game.exit() through REPL
	_, err := repl.Eval(`import("game").exit()`)
	if err != nil {
		t.Fatalf("Eval(import(\"game\").exit()) failed: %v", err)
	}

	// Update() should now return ebiten.Termination
	err = app.Update()
	if !errors.Is(err, ebiten.Termination) {
		t.Errorf("expected Update() to return ebiten.Termination, got: %v", err)
	}
}

func TestAppStartMapCommand(t *testing.T) {
	app := NewApp()

	console := app.ConsoleMode()
	if console == nil {
		t.Fatal("expected ConsoleMode to be non-nil")
	}

	repl := console.REPL()
	if repl == nil {
		t.Fatal("expected console REPL to be initialized")
	}

	// Execute game.StartMap("E1M1") through REPL
	_, err := repl.Eval(`game.StartMap("E1M1")`)
	if err != nil {
		t.Fatalf("Eval(game.StartMap) failed: %v", err)
	}

	// Active mode should now be GameMode
	gameMode, ok := app.CurrentMode().(*mode.GameMode)
	if !ok || gameMode == nil {
		t.Fatalf("expected CurrentMode to be *mode.GameMode after StartMap, got %T", app.CurrentMode())
	}
	if gameMode.MapName() != "E1M1" {
		t.Errorf("expected map name 'E1M1', got %q", gameMode.MapName())
	}

	if err := app.Update(); err != nil {
		t.Errorf("Update() returned error: %v", err)
	}
}
