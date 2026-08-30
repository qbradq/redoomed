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

	console, ok := app.CurrentMode().(*mode.ConsoleMode)
	if !ok || console == nil {
		t.Fatal("expected current mode to be *mode.ConsoleMode")
	}

	// Verify autoexec.tengo output lines were printed to console
	if len(console.History()) < 4 {
		t.Fatalf("expected at least 4 history lines from autoexec.tengo, got %d", len(console.History()))
	}
	if console.History()[0].Text != "Welcome to ReDoomEd!" {
		t.Errorf("expected first line 'Welcome to ReDoomEd!', got %q", console.History()[0].Text)
	}

	// If autoexec.tengo produced an error (e.g. invalid method call), verify it was recorded in bright red
	lastLine := console.History()[len(console.History())-1]
	if len(console.History()) > 4 {
		if lastLine.Color != gfx.EGABrightRed {
			t.Errorf("expected error line to be EGABrightRed, got %+v", lastLine.Color)
		}
	}

	if err := app.Update(); err != nil {
		t.Errorf("Update() returned error: %v", err)
	}
}

func TestAppExitCommand(t *testing.T) {
	app := NewApp()

	console, ok := app.CurrentMode().(*mode.ConsoleMode)
	if !ok || console == nil {
		t.Fatal("expected CurrentMode to be *mode.ConsoleMode")
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

	console, ok := app.CurrentMode().(*mode.ConsoleMode)
	if !ok || console == nil {
		t.Fatal("expected CurrentMode to be *mode.ConsoleMode initially")
	}

	repl := console.REPL()
	if repl == nil {
		t.Fatal("expected console REPL to be initialized")
	}

	// Execute game.StartMap("E1M1") through REPL (game was already imported in autoexec.tengo)
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
