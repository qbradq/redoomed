package mode

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/qbradq/redoomed/pkg/font"
	"github.com/qbradq/redoomed/pkg/gfx"
	"github.com/qbradq/redoomed/pkg/script"
)

const (
	// ConsoleBufferWidth is the logical width for console mode rendering.
	ConsoleBufferWidth = 640
	// ConsoleBufferHeight is the logical height for console mode rendering.
	ConsoleBufferHeight = 400

	// MaxHistoryLines is the maximum number of wrapped output lines stored in history.
	MaxHistoryLines = 300

	// MaxInputHistory is the maximum number of previous command lines stored.
	MaxInputHistory = 20

	// Margin and layout constants
	consoleMarginLeft   = 8
	consoleMarginRight  = 8
	consoleMarginTop    = 8
	consoleMarginBottom = 4
	consolePromptPrefix = "] "
)

// ConsoleLine represents a single line of text with its associated display color.
type ConsoleLine struct {
	Text  string
	Color color.RGBA
}

// ConsoleMode represents the Quake-style console application mode.
// It renders to a 640x400 internal buffer and composites onto the native 1280x800 buffer.
type ConsoleMode struct {
	buffer  *ebiten.Image
	font    *font.ConsoleFont
	repl    *script.REPL
	bgColor color.RGBA

	// Output history
	history      []ConsoleLine
	scrollOffset int

	// Interactive input
	inputRunes        []rune
	cursorPos         int
	cmdHistory        []string
	cmdHistoryIdx     int
	savedCurrentInput string
	cursorTick        int
}

// SetREPL assigns a Tengo REPL instance to the console and wires output to console.Print.
func (c *ConsoleMode) SetREPL(r *script.REPL) {
	c.repl = r
	if r != nil {
		r.SetPrintFunc(func(s string) {
			c.Print(s)
		})
	}
}

// REPL returns the active Tengo REPL instance.
func (c *ConsoleMode) REPL() *script.REPL {
	return c.repl
}

// NewConsoleMode creates a new ConsoleMode instance with a black background and the 8x8 fixed-width font.
func NewConsoleMode(f *font.ConsoleFont) *ConsoleMode {
	return &ConsoleMode{
		buffer:        ebiten.NewImage(ConsoleBufferWidth, ConsoleBufferHeight),
		font:          f,
		bgColor:       color.RGBA{R: 0, G: 0, B: 0, A: 255}, // Black background
		cmdHistoryIdx: -1,
	}
}

// Print appends a string to the console output with the default bright white color.
func (c *ConsoleMode) Print(text string) {
	c.PrintColored(text, gfx.EGABrightWhite)
}

// PrintColored appends a string to the console output in the specified color, word-wrapping lines as necessary.
func (c *ConsoleMode) PrintColored(text string, clr color.RGBA) {
	maxTextWidth := ConsoleBufferWidth - consoleMarginLeft - consoleMarginRight
	wrappedLines := c.wrapText(text, maxTextWidth)

	for _, line := range wrappedLines {
		c.history = append(c.history, ConsoleLine{
			Text:  line,
			Color: clr,
		})
	}

	if len(c.history) > MaxHistoryLines {
		c.history = c.history[len(c.history)-MaxHistoryLines:]
	}

	// If currently at bottom, keep scrolled to bottom
	if c.scrollOffset > 0 {
		maxScroll := c.maxScrollOffset()
		if c.scrollOffset > maxScroll {
			c.scrollOffset = maxScroll
		}
	}
}

// Clear removes all history lines from the console.
func (c *ConsoleMode) Clear() {
	c.history = nil
	c.scrollOffset = 0
}

// History returns the current output history lines.
func (c *ConsoleMode) History() []ConsoleLine {
	return c.history
}

// InputText returns the current input string.
func (c *ConsoleMode) InputText() string {
	return string(c.inputRunes)
}

// CursorPos returns the current cursor index.
func (c *ConsoleMode) CursorPos() int {
	return c.cursorPos
}

// ScrollOffset returns the current scroll offset.
func (c *ConsoleMode) ScrollOffset() int {
	return c.scrollOffset
}

// wrapText breaks text into lines that fit within maxWidth pixels.
func (c *ConsoleMode) wrapText(text string, maxWidth int) []string {
	var result []string
	paragraphs := strings.Split(text, "\n")

	for _, para := range paragraphs {
		if para == "" {
			result = append(result, "")
			continue
		}

		words := strings.Fields(para)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}

		currentLine := ""
		for _, word := range words {
			testLine := word
			if currentLine != "" {
				testLine = currentLine + " " + word
			}

			w := c.measureStringWidth(testLine)
			if w <= maxWidth {
				currentLine = testLine
			} else {
				if currentLine != "" {
					result = append(result, currentLine)
					currentLine = ""
				}

				// If word itself is wider than maxWidth, split by characters
				wordWidth := c.measureStringWidth(word)
				if wordWidth > maxWidth {
					var partial strings.Builder
					for _, r := range word {
						nextStr := partial.String() + string(r)
						if c.measureStringWidth(nextStr) > maxWidth {
							result = append(result, partial.String())
							partial.Reset()
						}
						partial.WriteRune(r)
					}
					currentLine = partial.String()
				} else {
					currentLine = word
				}
			}
		}

		if currentLine != "" {
			result = append(result, currentLine)
		}
	}

	return result
}

// measureStringWidth returns the pixel width of the string using the 8x8 console font.
func (c *ConsoleMode) measureStringWidth(s string) int {
	if c.font != nil {
		w, _ := c.font.MeasureText(s)
		return w
	}
	return len(s) * 8
}

// lineHeight returns the font line height (8px).
func (c *ConsoleMode) lineHeight() int {
	if c.font != nil {
		return c.font.LineHeight()
	}
	return 8
}

// maxVisibleLines returns how many lines of output fit in the output area.
func (c *ConsoleMode) maxVisibleLines() int {
	lh := c.lineHeight()
	inputY := ConsoleBufferHeight - lh - consoleMarginBottom
	outputHeight := inputY - consoleMarginTop - 2
	if outputHeight <= 0 || lh <= 0 {
		return 1
	}
	return outputHeight / lh
}

// maxScrollOffset returns the maximum allowed scroll offset.
func (c *ConsoleMode) maxScrollOffset() int {
	vis := c.maxVisibleLines()
	if len(c.history) <= vis {
		return 0
	}
	return len(c.history) - vis
}

func (c *ConsoleMode) scrollUp(lines int) {
	c.scrollOffset += lines
	maxScroll := c.maxScrollOffset()
	if c.scrollOffset > maxScroll {
		c.scrollOffset = maxScroll
	}
}

func (c *ConsoleMode) scrollDown(lines int) {
	c.scrollOffset -= lines
	if c.scrollOffset < 0 {
		c.scrollOffset = 0
	}
}

// isKeyRepeating checks if a key was just pressed or is repeating.
func isKeyRepeating(key ebiten.Key) bool {
	d := inpututil.KeyPressDuration(key)
	return d == 1 || (d >= 20 && (d-20)%4 == 0)
}

// Update handles keyboard input for navigation, editing, scrolling, and submission.
func (c *ConsoleMode) Update() error {
	c.cursorTick++

	// 1. Scrolling keys (Page Up / Page Down)
	scrollStep := c.maxVisibleLines() / 2
	if scrollStep < 5 {
		scrollStep = 5
	}

	if isKeyRepeating(ebiten.KeyPageUp) {
		c.scrollUp(scrollStep)
	}
	if isKeyRepeating(ebiten.KeyPageDown) {
		c.scrollDown(scrollStep)
	}

	// 2. Left / Right cursor movement
	if isKeyRepeating(ebiten.KeyLeft) {
		if c.cursorPos > 0 {
			c.cursorPos--
			c.cursorTick = 0
		}
	}
	if isKeyRepeating(ebiten.KeyRight) {
		if c.cursorPos < len(c.inputRunes) {
			c.cursorPos++
			c.cursorTick = 0
		}
	}
	if isKeyRepeating(ebiten.KeyHome) {
		c.cursorPos = 0
		c.cursorTick = 0
	}
	if isKeyRepeating(ebiten.KeyEnd) {
		c.cursorPos = len(c.inputRunes)
		c.cursorTick = 0
	}

	// 3. Backspace and Delete
	if isKeyRepeating(ebiten.KeyBackspace) {
		if c.cursorPos > 0 {
			c.inputRunes = append(c.inputRunes[:c.cursorPos-1], c.inputRunes[c.cursorPos:]...)
			c.cursorPos--
			c.cursorTick = 0
		}
	}
	if isKeyRepeating(ebiten.KeyDelete) {
		if c.cursorPos < len(c.inputRunes) {
			c.inputRunes = append(c.inputRunes[:c.cursorPos], c.inputRunes[c.cursorPos+1:]...)
			c.cursorTick = 0
		}
	}

	// 4. Input history navigation (Up / Down arrows)
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		if len(c.cmdHistory) > 0 {
			if c.cmdHistoryIdx == -1 {
				c.savedCurrentInput = string(c.inputRunes)
				c.cmdHistoryIdx = len(c.cmdHistory) - 1
			} else if c.cmdHistoryIdx > 0 {
				c.cmdHistoryIdx--
			}
			if c.cmdHistoryIdx >= 0 && c.cmdHistoryIdx < len(c.cmdHistory) {
				c.inputRunes = []rune(c.cmdHistory[c.cmdHistoryIdx])
				c.cursorPos = len(c.inputRunes)
				c.cursorTick = 0
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		if c.cmdHistoryIdx != -1 {
			c.cmdHistoryIdx++
			if c.cmdHistoryIdx >= len(c.cmdHistory) {
				c.cmdHistoryIdx = -1
				c.inputRunes = []rune(c.savedCurrentInput)
				c.cursorPos = len(c.inputRunes)
			} else {
				c.inputRunes = []rune(c.cmdHistory[c.cmdHistoryIdx])
				c.cursorPos = len(c.inputRunes)
			}
			c.cursorTick = 0
		}
	}

	// 5. Enter submission
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		cmdText := string(c.inputRunes)
		if strings.TrimSpace(cmdText) != "" {
			c.cmdHistory = append(c.cmdHistory, cmdText)
			if len(c.cmdHistory) > MaxInputHistory {
				c.cmdHistory = c.cmdHistory[len(c.cmdHistory)-MaxInputHistory:]
			}
		}

		// Echo the user's input to the output buffer in EGA bright yellow
		c.PrintColored(consolePromptPrefix+cmdText, gfx.EGABrightYellow)

		// Execute in Tengo REPL if available
		if c.repl != nil && strings.TrimSpace(cmdText) != "" {
			res, err := c.repl.Eval(cmdText)
			if err != nil {
				c.PrintColored(err.Error(), gfx.EGABrightRed)
			} else if res != "" {
				c.PrintColored(res, gfx.EGABrightWhite)
			}
		}

		// Reset input line state and scroll position
		c.inputRunes = nil
		c.cursorPos = 0
		c.cmdHistoryIdx = -1
		c.savedCurrentInput = ""
		c.scrollOffset = 0
		c.cursorTick = 0
	}

	// 6. Character input (ASCII except ` and ~)
	inputChars := ebiten.AppendInputChars(nil)
	for _, r := range inputChars {
		if r == '`' || r == '~' {
			continue
		}
		if r >= 32 && r <= 126 {
			c.insertRune(r)
			c.cursorTick = 0
		}
	}

	return nil
}

func (c *ConsoleMode) insertRune(r rune) {
	if c.cursorPos == len(c.inputRunes) {
		c.inputRunes = append(c.inputRunes, r)
		c.cursorPos++
		return
	}
	c.inputRunes = append(c.inputRunes[:c.cursorPos], append([]rune{r}, c.inputRunes[c.cursorPos:]...)...)
	c.cursorPos++
}

// Draw renders the console content onto the 640x400 buffer and composites to the screen.
func (c *ConsoleMode) Draw(screen *ebiten.Image) {
	// 1. Fill background with Black
	c.buffer.Fill(c.bgColor)

	lh := c.lineHeight()
	inputY := ConsoleBufferHeight - lh - consoleMarginBottom
	outputBottomY := inputY - 2

	// 2. Render output history aligned to the bottom with each line's color
	if len(c.history) > 0 && c.font != nil {
		endIdx := len(c.history) - 1 - c.scrollOffset
		drawY := outputBottomY - lh

		for i := endIdx; i >= 0 && drawY >= consoleMarginTop; i-- {
			line := c.history[i]
			c.font.DrawText(c.buffer, line.Text, consoleMarginLeft, drawY, line.Color)
			drawY -= lh
		}
	}

	// 3. Render bottom input prompt line in EGA bright yellow
	if c.font != nil {
		// Draw prompt prefix in EGA bright yellow
		c.font.DrawText(c.buffer, consolePromptPrefix, consoleMarginLeft, inputY, gfx.EGABrightYellow)
		prefixWidth, _ := c.font.MeasureText(consolePromptPrefix)

		// Draw typed input text in EGA bright yellow
		inputStr := string(c.inputRunes)
		textX := consoleMarginLeft + prefixWidth
		c.font.DrawText(c.buffer, inputStr, textX, inputY, gfx.EGABrightYellow)

		// Draw blinking cursor (every 30 frames) in EGA bright yellow
		if (c.cursorTick/30)%2 == 0 {
			cursorOffset, _ := c.font.MeasureText(string(c.inputRunes[:c.cursorPos]))
			cursorX := textX + cursorOffset
			c.font.DrawText(c.buffer, "_", cursorX, inputY, gfx.EGABrightYellow)
		}
	}

	// 4. Composite 640x400 console buffer onto the screen
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	scaleX := float64(sw) / float64(ConsoleBufferWidth)
	scaleY := float64(sh) / float64(ConsoleBufferHeight)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(c.buffer, op)
}
