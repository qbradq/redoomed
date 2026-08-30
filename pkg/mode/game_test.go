package mode

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestGameMode(t *testing.T) {
	gm := NewGameMode("MAP01")
	if gm == nil {
		t.Fatal("expected NewGameMode to return non-nil")
	}

	if gm.MapName() != "MAP01" {
		t.Errorf("expected map name 'MAP01', got %q", gm.MapName())
	}

	gm.SetMapName("E1M1")
	if gm.MapName() != "E1M1" {
		t.Errorf("expected map name 'E1M1', got %q", gm.MapName())
	}

	if err := gm.Update(); err != nil {
		t.Errorf("Update returned error: %v", err)
	}

	screen := ebiten.NewImage(1280, 800)
	gm.Draw(screen)
}
