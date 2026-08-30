package mode

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/qbradq/redoomed/pkg/gfx"
)

func TestConsoleModeBasics(t *testing.T) {
	cm := NewConsoleMode(nil)
	if cm == nil {
		t.Fatal("expected NewConsoleMode to return non-nil")
	}

	// Should start with empty history until scripts or user prints
	if len(cm.History()) != 0 {
		t.Fatalf("expected 0 initial history lines, got %d", len(cm.History()))
	}
}

func TestConsoleModeHistoryCap(t *testing.T) {
	cm := NewConsoleMode(nil)

	for i := 0; i < 350; i++ {
		cm.Print(fmt.Sprintf("Line %d", i))
	}

	if len(cm.History()) != MaxHistoryLines {
		t.Errorf("expected history length capped at %d, got %d", MaxHistoryLines, len(cm.History()))
	}

	lastLine := cm.History()[len(cm.History())-1]
	if lastLine.Color != gfx.EGABrightWhite {
		t.Errorf("expected default print color to be EGABrightWhite, got %+v", lastLine.Color)
	}
}

func TestConsoleModeColoredPrint(t *testing.T) {
	cm := NewConsoleMode(nil)
	customColor := color.RGBA{R: 0x55, G: 0xFF, B: 0x55, A: 0xFF}
	cm.PrintColored("Custom green line", customColor)

	lastLine := cm.History()[len(cm.History())-1]
	if lastLine.Text != "Custom green line" {
		t.Errorf("expected 'Custom green line', got %q", lastLine.Text)
	}
	if lastLine.Color != customColor {
		t.Errorf("expected color %+v, got %+v", customColor, lastLine.Color)
	}
}

func TestConsoleModeWordWrap(t *testing.T) {
	cm := NewConsoleMode(nil)
	lines := cm.wrapText("The quick brown fox jumps over the lazy dog", 120)
	if len(lines) <= 1 {
		t.Errorf("expected text to wrap into multiple lines, got %d lines: %v", len(lines), lines)
	}
}

func TestConsoleModeInputAndCursor(t *testing.T) {
	cm := NewConsoleMode(nil)

	// Simulate character typing
	cm.insertRune('h')
	cm.insertRune('e')
	cm.insertRune('l')
	cm.insertRune('p')

	if cm.InputText() != "help" {
		t.Errorf("expected input 'help', got %q", cm.InputText())
	}
	if cm.CursorPos() != 4 {
		t.Errorf("expected cursor pos 4, got %d", cm.CursorPos())
	}

	// Test cursor left and insert
	cm.cursorPos = 2 // cursor before 'l'
	cm.insertRune('x')
	if cm.InputText() != "hexlp" {
		t.Errorf("expected input 'hexlp', got %q", cm.InputText())
	}

	// Test backspace
	cm.inputRunes = append(cm.inputRunes[:cm.cursorPos-1], cm.inputRunes[cm.cursorPos:]...)
	cm.cursorPos--
	if cm.InputText() != "help" {
		t.Errorf("expected input 'help' after removing 'x', got %q", cm.InputText())
	}

	// Test delete at cursor (cursor is at 2, i.e., 'l')
	cm.inputRunes = append(cm.inputRunes[:cm.cursorPos], cm.inputRunes[cm.cursorPos+1:]...)
	if cm.InputText() != "hep" {
		t.Errorf("expected input 'hep' after delete, got %q", cm.InputText())
	}
}

func TestConsoleModeScrolling(t *testing.T) {
	cm := NewConsoleMode(nil)
	for i := 0; i < 50; i++ {
		cm.Print(fmt.Sprintf("Line %d", i))
	}

	cm.scrollUp(10)
	if cm.ScrollOffset() <= 0 {
		t.Errorf("expected scroll offset > 0 after scrolling up, got %d", cm.ScrollOffset())
	}

	cm.scrollDown(20)
	if cm.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset clamped to 0 after scrolling down, got %d", cm.ScrollOffset())
	}
}

func TestConsoleModeDraw(t *testing.T) {
	cm := NewConsoleMode(nil)
	screen := ebiten.NewImage(1280, 800)
	cm.Draw(screen)
}
