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

	// Autoexec script runs game.StartMap("E1M1"), so active mode is GameMode
	gameMode, ok := app.CurrentMode().(*mode.GameMode)
	if !ok || gameMode == nil {
		t.Fatalf("expected current mode to be *mode.GameMode after autoexec, got %T", app.CurrentMode())
	}
	if gameMode.MapName() != "E1M1" {
		t.Errorf("expected map name 'E1M1', got %q", gameMode.MapName())
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
	if err := app.Update(); !errors.Is(err, ebiten.Termination) {
		t.Errorf("expected ebiten.Termination after exit, got %v", err)
	}
}

func TestAppMusicManager(t *testing.T) {
	app := NewApp()
	if app.MusicManager() == nil {
		t.Fatal("expected non-nil MusicManager from App")
	}

	// Should have default volume 0.7
	if app.MusicManager().Volume() != 0.7 {
		t.Errorf("expected default volume 0.7, got %f", app.MusicManager().Volume())
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

func TestNewAppWithIWAD(t *testing.T) {
	// Test with explicit IWAD path
	app := NewAppWithIWAD("../../freedoom2.wad")
	if app == nil {
		t.Fatal("expected NewAppWithIWAD to return non-nil app")
	}
	if app.WAD() == nil {
		t.Log("Note: freedoom2.wad was not found at relative path in this environment")
	}

	// Test with invalid path (should gracefully handle missing file and not panic)
	appInvalid := NewAppWithIWAD("nonexistent.wad")
	if appInvalid == nil {
		t.Fatal("expected NewAppWithIWAD to handle missing file gracefully")
	}
}

type mockMode struct {
	updateCount int
	drawCount   int
}

func (m *mockMode) Update() error {
	m.updateCount++
	return nil
}

func (m *mockMode) Draw(screen *ebiten.Image) {
	m.drawCount++
}

func TestAppConsoleUnderlyingModeRendering(t *testing.T) {
	app := NewApp()

	// Initial mode after autoexec is GameMode
	gameMode, ok := app.CurrentMode().(*mode.GameMode)
	if !ok || gameMode == nil {
		t.Fatalf("expected initial CurrentMode to be GameMode, got %T", app.CurrentMode())
	}

	// Toggle to Console
	app.ToggleConsole()
	if app.CurrentMode() != app.ConsoleMode() {
		t.Fatalf("expected CurrentMode to be ConsoleMode, got %T", app.CurrentMode())
	}

	// Draw should composite gameMode under consoleMode without panicking
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	app.Draw(screen)

	// In ConsoleMode, Update should only update console and not cause error
	if err := app.Update(); err != nil {
		t.Errorf("app.Update() error: %v", err)
	}

	// Toggle back to GameMode
	app.ToggleConsole()
	if app.CurrentMode() != gameMode {
		t.Fatalf("expected mode after toggle back to be GameMode, got %T", app.CurrentMode())
	}
}

func TestAppConsoleOverCustomMode(t *testing.T) {
	app := NewApp()

	mock := &mockMode{}
	app.SetMode(mock)

	if app.CurrentMode() != mock {
		t.Fatalf("expected CurrentMode to be mockMode, got %T", app.CurrentMode())
	}

	// Toggle console on top of mock mode (simulating future editor mode)
	app.ToggleConsole()
	if app.CurrentMode() != app.ConsoleMode() {
		t.Fatalf("expected CurrentMode to be ConsoleMode, got %T", app.CurrentMode())
	}

	// When Draw is called, mock mode Draw should be called once as the underlying layer
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	app.Draw(screen)
	if mock.drawCount != 1 {
		t.Errorf("expected mockMode.drawCount to be 1, got %d", mock.drawCount)
	}

	// When Update is called, mock mode Update should NOT be called (simulation frozen)
	if err := app.Update(); err != nil {
		t.Errorf("app.Update() error: %v", err)
	}
	if mock.updateCount != 0 {
		t.Errorf("expected mockMode.updateCount to be 0 while console is open, got %d", mock.updateCount)
	}

	// Toggle console off should restore mock mode
	app.ToggleConsole()
	if app.CurrentMode() != mock {
		t.Fatalf("expected CurrentMode restored to mockMode, got %T", app.CurrentMode())
	}

	// Now Update should advance mock mode
	if err := app.Update(); err != nil {
		t.Errorf("app.Update() error: %v", err)
	}
	if mock.updateCount != 1 {
		t.Errorf("expected mockMode.updateCount to be 1 after console closed, got %d", mock.updateCount)
	}
}

func TestAppCursorMode(t *testing.T) {
	app := NewApp()

	// Initial mode after autoexec is GameMode -> cursor should be captured
	if _, ok := app.CurrentMode().(*mode.GameMode); !ok {
		t.Fatalf("expected GameMode after autoexec, got %T", app.CurrentMode())
	}
	if ebiten.CursorMode() != ebiten.CursorModeCaptured {
		t.Errorf("expected CursorModeCaptured in GameMode, got %v", ebiten.CursorMode())
	}

	// Toggle console -> cursor should be visible
	app.ToggleConsole()
	if ebiten.CursorMode() != ebiten.CursorModeVisible {
		t.Errorf("expected CursorModeVisible in ConsoleMode, got %v", ebiten.CursorMode())
	}

	// Toggle console back to GameMode -> cursor should be captured again
	app.ToggleConsole()
	if ebiten.CursorMode() != ebiten.CursorModeCaptured {
		t.Errorf("expected CursorModeCaptured back in GameMode, got %v", ebiten.CursorMode())
	}

	// Set to custom/editor mode -> cursor should be visible
	editorMock := &mockMode{}
	app.SetMode(editorMock)
	if ebiten.CursorMode() != ebiten.CursorModeVisible {
		t.Errorf("expected CursorModeVisible in Editor/Custom mode, got %v", ebiten.CursorMode())
	}

	// Switch back to GameMode -> cursor should be captured
	app.SetMode(app.GameMode())
	if ebiten.CursorMode() != ebiten.CursorModeCaptured {
		t.Errorf("expected CursorModeCaptured when returning to GameMode, got %v", ebiten.CursorMode())
	}
}

func TestAppToggleEditor(t *testing.T) {
	app := NewApp()

	if app.EditorMode() == nil {
		t.Fatal("expected app.EditorMode() to be non-nil")
	}

	// Initially in GameMode after autoexec
	if _, ok := app.CurrentMode().(*mode.GameMode); !ok {
		t.Fatalf("expected initial mode to be GameMode, got %T", app.CurrentMode())
	}

	// Toggle to EditorMode
	app.ToggleEditor()
	if _, ok := app.CurrentMode().(*mode.EditorMode); !ok {
		t.Fatalf("expected CurrentMode to be EditorMode after ToggleEditor(), got %T", app.CurrentMode())
	}
	if ebiten.CursorMode() != ebiten.CursorModeVisible {
		t.Errorf("expected cursor to be visible in EditorMode, got %v", ebiten.CursorMode())
	}

	// Toggle back to GameMode
	app.ToggleEditor()
	if _, ok := app.CurrentMode().(*mode.GameMode); !ok {
		t.Fatalf("expected CurrentMode to return to GameMode after second ToggleEditor(), got %T", app.CurrentMode())
	}

	// Toggle console, then toggle editor
	app.ToggleConsole()
	if _, ok := app.CurrentMode().(*mode.ConsoleMode); !ok {
		t.Fatalf("expected ConsoleMode, got %T", app.CurrentMode())
	}

	app.ToggleEditor()
	if _, ok := app.CurrentMode().(*mode.EditorMode); !ok {
		t.Fatalf("expected EditorMode after ToggleEditor from console, got %T", app.CurrentMode())
	}

	// Draw and update while in EditorMode
	screen := ebiten.NewImage(1280, 800)
	app.Draw(screen)
	if err := app.Update(); err != nil {
		t.Errorf("app.Update() in EditorMode returned error: %v", err)
	}
}


