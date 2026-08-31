package script

import (
	"testing"

	"github.com/d5/tengo/v2"

	"github.com/qbradq/redoomed/pkg/player"
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

func TestTengoExport(t *testing.T) {
	scriptSrc := `
fn := func(line_id, sec_id, thing_id, tag) {
	return line_id + sec_id + thing_id + tag
}
export fn
`
	modMap := tengo.NewModuleMap()
	modMap.AddSourceModule("line_001", []byte(scriptSrc))

	callerSrc := `
line := import("line_001")
res := line(line_id, sec_id, thing_id, tag)
`
	s := tengo.NewScript([]byte(callerSrc))
	s.SetImports(modMap)
	s.Add("line_id", 0)
	s.Add("sec_id", 0)
	s.Add("thing_id", 0)
	s.Add("tag", 0)

	compiled, err := s.Compile()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Call 1
	c1 := compiled.Clone()
	_ = c1.Set("line_id", 10)
	_ = c1.Set("sec_id", 20)
	_ = c1.Set("thing_id", 1)
	_ = c1.Set("tag", 5)
	if err := c1.Run(); err != nil {
		t.Fatalf("c1 Run failed: %v", err)
	}
	if c1.Get("res").Int() != 36 {
		t.Fatalf("expected 36, got %v", c1.Get("res"))
	}

	// Call 2 (reusing compiled with different args)
	c2 := compiled.Clone()
	_ = c2.Set("line_id", 100)
	_ = c2.Set("sec_id", 200)
	_ = c2.Set("thing_id", 2)
	_ = c2.Set("tag", 10)
	if err := c2.Run(); err != nil {
		t.Fatalf("c2 Run failed: %v", err)
	}
	if c2.Get("res").Int() != 312 {
		t.Fatalf("expected 312, got %v", c2.Get("res"))
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

func TestPlayerModuleScripting(t *testing.T) {
	ps := player.NewPlayerStats()
	repl := NewREPL(nil, nil)
	repl.SetPlayerStatsProvider(func() *player.PlayerStats {
		return ps
	})

	_, err := repl.Eval(`p := import("player")`)
	if err != nil {
		t.Fatalf("Eval(import(\"player\")) failed: %v", err)
	}

	// Test health & armor queries
	res, err := repl.Eval(`p.get_health()`)
	if err != nil {
		t.Fatalf("Eval(p.get_health()) failed: %v", err)
	}
	if res != "100" {
		t.Errorf("expected health 100, got %s", res)
	}

	// Test damage via script
	_, err = repl.Eval(`p.damage(25)`)
	if err != nil {
		t.Fatalf("Eval(p.damage(25)) failed: %v", err)
	}
	if ps.Health != 75 {
		t.Errorf("expected health 75 after damage, got %d", ps.Health)
	}

	// Test give health
	_, err = repl.Eval(`p.give_health(15, 100)`)
	if err != nil {
		t.Fatalf("Eval(p.give_health(15, 100)) failed: %v", err)
	}
	if ps.Health != 90 {
		t.Errorf("expected health 90 after give_health, got %d", ps.Health)
	}

	// Test give weapon and select weapon
	_, err = repl.Eval(`p.give_weapon("shotgun")`)
	if err != nil {
		t.Fatalf("Eval(p.give_weapon) failed: %v", err)
	}
	if !ps.HasWeapon(player.WeaponShotgun) {
		t.Error("expected player to own shotgun")
	}

	res, err = repl.Eval(`p.has_weapon("shotgun")`)
	if err != nil || res != "true" {
		t.Errorf("expected has_weapon to return true, got %s (err: %v)", res, err)
	}

	// Test give ammo
	_, err = repl.Eval(`p.give_ammo("shells", 20)`)
	if err != nil {
		t.Fatalf("Eval(p.give_ammo) failed: %v", err)
	}
	if ps.Ammo[player.AmmoShells] != 28 { // 8 on pickup + 20
		t.Errorf("expected 28 shells, got %d", ps.Ammo[player.AmmoShells])
	}

	// Test give key
	_, err = repl.Eval(`p.give_key("blue_card")`)
	if err != nil {
		t.Fatalf("Eval(p.give_key) failed: %v", err)
	}
	if !ps.HasKey(player.KeyBlueCard) {
		t.Error("expected player to possess blue card")
	}

	// Test god mode
	_, err = repl.Eval(`p.god_mode(true)`)
	if err != nil {
		t.Fatalf("Eval(p.god_mode(true)) failed: %v", err)
	}
	if !ps.GodMode {
		t.Error("expected god mode to be enabled")
	}

	// Test give all (IDKFA)
	_, err = repl.Eval(`p.give_all()`)
	if err != nil {
		t.Fatalf("Eval(p.give_all()) failed: %v", err)
	}
	if ps.Health != 200 || ps.Armor != 200 {
		t.Errorf("expected 200 health/armor from give_all, got %d/%d", ps.Health, ps.Armor)
	}
}

func TestGameNoClip(t *testing.T) {
	repl := NewREPL(nil, nil)

	noClipState := false
	repl.SetSetNoClipFunc(func(enabled bool) {
		noClipState = enabled
	})

	// Test game.no_clip(true)
	_, err := repl.Eval(`game := import("game"); game.no_clip(true)`)
	if err != nil {
		t.Fatalf("Eval(game.no_clip(true)) failed: %v", err)
	}
	if !noClipState {
		t.Error("expected noclip to be enabled (true)")
	}

	// Test game.no_clip(false)
	_, err = repl.Eval(`game.no_clip(false)`)
	if err != nil {
		t.Fatalf("Eval(game.no_clip(false)) failed: %v", err)
	}
	if noClipState {
		t.Error("expected noclip to be disabled (false)")
	}

	// Test default game.no_clip() enables noclip
	_, err = repl.Eval(`game.no_clip()`)
	if err != nil {
		t.Fatalf("Eval(game.no_clip()) failed: %v", err)
	}
	if !noClipState {
		t.Error("expected noclip to be enabled (default)")
	}
}


