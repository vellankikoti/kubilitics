package output

import (
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

type ColorDepth int

const (
	NoColor ColorDepth = iota
	Color16
	Color256
	TrueColor
)

type Capabilities struct {
	Width          int
	Height         int
	ColorDepth     ColorDepth
	IsTTY          bool
	SupportsUnicode bool
	mu             sync.RWMutex
}

var (
	termCaps *Capabilities
	capsMutex sync.Mutex
)

func Init() *Capabilities {
	capsMutex.Lock()
	defer capsMutex.Unlock()

	if termCaps != nil {
		return termCaps
	}

	caps := &Capabilities{
		IsTTY: term.IsTerminal(int(os.Stdout.Fd())),
	}

	if caps.IsTTY {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width, height = 80, 24
		}
		caps.Width = width
		caps.Height = height
	} else {
		caps.Width = 80
		caps.Height = 24
	}

	caps.ColorDepth = detectColorDepth()
	caps.SupportsUnicode = detectUnicode()

	termCaps = caps

	if caps.IsTTY {
		startResizeListener(caps)
	}

	return caps
}

func GetTermCaps() *Capabilities {
	if termCaps == nil {
		return Init()
	}
	return termCaps
}

func detectColorDepth() ColorDepth {
	if os.Getenv("NO_COLOR") != "" {
		return NoColor
	}

	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return TrueColor
	}

	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") {
		return Color256
	}

	if strings.HasPrefix(term, "xterm") || strings.HasPrefix(term, "screen") {
		return Color256
	}

	return Color16
}

func detectUnicode() bool {
	lang := os.Getenv("LANG")
	if strings.Contains(lang, "UTF-8") || strings.Contains(lang, "utf-8") {
		return true
	}

	lc := os.Getenv("LC_ALL")
	if strings.Contains(lc, "UTF-8") || strings.Contains(lc, "utf-8") {
		return true
	}

	return term.IsTerminal(int(os.Stdout.Fd()))
}

func (c *Capabilities) SetSize(width, height int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Width = width
	c.Height = height
}

func (c *Capabilities) GetSize() (int, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Width, c.Height
}

func (c *Capabilities) GetColorDepth() ColorDepth {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ColorDepth
}

func (c *Capabilities) HasColors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ColorDepth != NoColor
}

func (c *Capabilities) IsTrueColor() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ColorDepth == TrueColor
}

func (c *Capabilities) Is256Color() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ColorDepth == Color256 || c.ColorDepth == TrueColor
}

func SupportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	return GetTermCaps().HasColors()
}

func ShouldUseColor(noColorFlag bool) bool {
	if noColorFlag {
		return false
	}

	return SupportsColor()
}

func TermWidth() int {
	width, _ := GetTermCaps().GetSize()
	return width
}

func TermHeight() int {
	_, height := GetTermCaps().GetSize()
	return height
}
