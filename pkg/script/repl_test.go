package script

import (
	"testing"
)

func TestREPLEvaluation(t *testing.T) {
	repl := NewREPL(nil, nil)

	// Test variable definition
	_, err := repl.Eval("a := 15")
	if err != nil {
		t.Fatalf("Eval(a := 15) failed: %v", err)
	}

	// Test arithmetic expression
	res, err := repl.Eval("a * 2")
	if err != nil {
		t.Fatalf("Eval(a * 2) failed: %v", err)
	}
	if res != "30" {
		t.Errorf("expected result '30', got %q", res)
	}

	// Test string concatenation
	res, err = repl.Eval(`"Hello, " + "World!"`)
	if err != nil {
		t.Fatalf("Eval string concatenation failed: %v", err)
	}
	if res != `"Hello, World!"` && res != "Hello, World!" {
		t.Errorf("expected Hello, World!, got %q", res)
	}

	// Test syntax error
	_, err = repl.Eval("invalid +++ syntax")
	if err == nil {
		t.Error("expected syntax error on invalid syntax")
	}
}

func TestGameExit(t *testing.T) {
	exitCalled := false
	repl := NewREPL(func() {
		exitCalled = true
	}, nil)

	_, err := repl.Eval(`import("game").exit()`)
	if err != nil {
		t.Fatalf("Eval(import(\"game\").exit()) failed: %v", err)
	}
	if !exitCalled {
		t.Error("expected game.exit() to call onExit callback")
	}
}

func TestAutoexecScript(t *testing.T) {
	var printed []string
	var startedMap string

	repl := NewREPL(nil, func(s string) {
		printed = append(printed, s)
	})
	repl.SetStartMapFunc(func(m string) {
		startedMap = m
	})

	scriptCode := `
fmt := import("fmt")
game := import("game")

fmt.println("Welcome to ReDoomEd!")
game.StartMap("MAP01")
`
	_, err := repl.EvalScript("autoexec.tengo", scriptCode)
	if err != nil {
		t.Fatalf("EvalScript failed: %v", err)
	}

	if startedMap != "MAP01" {
		t.Errorf("expected autoexec to start MAP01, got %s", startedMap)
	}

	if len(printed) == 0 {
		t.Error("expected printed output from autoexec")
	}
}

func TestGameMusicBindings(t *testing.T) {
	var playedMusic string
	var stopped bool
	var setVol float64
	currentTrack := "D_RUNNIN"

	repl := NewREPL(nil, nil)
	repl.SetPlayMusicFunc(func(m string) { playedMusic = m })
	repl.SetStopMusicFunc(func() { stopped = true })
	repl.SetSetMusicVolumeFunc(func(v float64) { setVol = v })
	repl.SetGetMusicTrackFunc(func() string { return currentTrack })

	_, err := repl.Eval(`
game := import("game")
game.PlayMusic("D_STALKS")
game.SetMusicVolume(0.5)
t := game.GetMusicTrack()
game.StopMusic()
`)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	if playedMusic != "D_STALKS" {
		t.Errorf("expected played music D_STALKS, got %q", playedMusic)
	}
	if setVol != 0.5 {
		t.Errorf("expected set volume 0.5, got %f", setVol)
	}
	if !stopped {
		t.Error("expected StopMusic to be called")
	}
}

func TestStartMap(t *testing.T) {
	var loadedMap string
	repl := NewREPL(nil, nil)
	repl.SetStartMapFunc(func(mapName string) {
		loadedMap = mapName
	})

	_, err := repl.Eval(`game := import("game"); game.StartMap("MAP01")`)
	if err != nil {
		t.Fatalf("Eval(game.StartMap(\"MAP01\")) failed: %v", err)
	}
	if loadedMap != "MAP01" {
		t.Errorf("expected loaded map 'MAP01', got %q", loadedMap)
	}

	// Test snake_case start_map
	_, err = repl.Eval(`game.start_map("E1M1")`)
	if err != nil {
		t.Fatalf("Eval(game.start_map(\"E1M1\")) failed: %v", err)
	}
	if loadedMap != "E1M1" {
		t.Errorf("expected loaded map 'E1M1', got %q", loadedMap)
	}
}

func TestGameImportExplicit(t *testing.T) {
	exitCalled := false
	repl := NewREPL(func() {
		exitCalled = true
	}, nil)

	_, err := repl.Eval(`g := import("game"); g.exit()`)
	if err != nil {
		t.Fatalf("Eval(g := import(\"game\"); g.exit()) failed: %v", err)
	}
	if !exitCalled {
		t.Error("expected g.exit() to call onExit callback")
	}
}

func TestFmtModuleRedirection(t *testing.T) {
	var captured []string
	repl := NewREPL(nil, func(s string) {
		captured = append(captured, s)
	})

	_, err := repl.Eval(`fmt := import("fmt")`)
	if err != nil {
		t.Fatalf("Eval(import(\"fmt\")) failed: %v", err)
	}

	// Test fmt.println
	_, err = repl.Eval(`fmt.println("Line from println:", 42)`)
	if err != nil {
		t.Fatalf("Eval(fmt.println) failed: %v", err)
	}

	// Test fmt.printf
	_, err = repl.Eval(`fmt.printf("Score: %04d", 7)`)
	if err != nil {
		t.Fatalf("Eval(fmt.printf) failed: %v", err)
	}

	// Test fmt.sprintf (should return string value, not write to console)
	res, err := repl.Eval(`fmt.sprintf("Formatted %s", "text")`)
	if err != nil {
		t.Fatalf("Eval(fmt.sprintf) failed: %v", err)
	}
	if res != `"Formatted text"` && res != "Formatted text" {
		t.Errorf("expected 'Formatted text', got %q", res)
	}

	if len(captured) != 2 {
		t.Fatalf("expected 2 captured output lines, got %d: %v", len(captured), captured)
	}
	if captured[0] != "Line from println: 42" {
		t.Errorf("expected 'Line from println: 42', got %q", captured[0])
	}
	if captured[1] != "Score: 0007" {
		t.Errorf("expected 'Score: 0007', got %q", captured[1])
	}
}
