package output

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Primary       lipgloss.Style
	Secondary     lipgloss.Style
	Success       lipgloss.Style
	Warning       lipgloss.Style
	Error         lipgloss.Style
	Info          lipgloss.Style
	Muted         lipgloss.Style
	Highlight     lipgloss.Style
	Header        lipgloss.Style
	Border        lipgloss.Style
	StatusReady   lipgloss.Style
	StatusPending lipgloss.Style
	StatusError   lipgloss.Style
	StatusUnknown lipgloss.Style
}

var (
	currentTheme *Theme
	themeMutex   sync.RWMutex
)

func darkTheme() *Theme {
	if !GetTermCaps().HasColors() {
		return noColorTheme()
	}

	caps := GetTermCaps()

	switch caps.ColorDepth {
	case TrueColor:
		return &Theme{
			Primary:       lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")),
			Secondary:     lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")),
			Success:       lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")),
			Warning:       lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68")),
			Error:         lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")),
			Info:          lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")),
			Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")),
			Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("#BB9AF7")),
			Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true),
			Border:        lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")),
			StatusReady:   lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")),
			StatusPending: lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68")),
			StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")),
			StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")),
		}
	case Color256:
		return &Theme{
			Primary:       lipgloss.NewStyle().Foreground(lipgloss.Color("63")),
			Secondary:     lipgloss.NewStyle().Foreground(lipgloss.Color("107")),
			Success:       lipgloss.NewStyle().Foreground(lipgloss.Color("107")),
			Warning:       lipgloss.NewStyle().Foreground(lipgloss.Color("179")),
			Error:         lipgloss.NewStyle().Foreground(lipgloss.Color("204")),
			Info:          lipgloss.NewStyle().Foreground(lipgloss.Color("63")),
			Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("60")),
			Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("141")),
			Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true),
			Border:        lipgloss.NewStyle().Foreground(lipgloss.Color("60")),
			StatusReady:   lipgloss.NewStyle().Foreground(lipgloss.Color("107")),
			StatusPending: lipgloss.NewStyle().Foreground(lipgloss.Color("179")),
			StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("204")),
			StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color("60")),
		}
	default:
		return &Theme{
			Primary:       lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
			Secondary:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			Success:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			Warning:       lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			Error:         lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
			Info:          lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
			Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
			Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true),
			Border:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			StatusReady:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			StatusPending: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
			StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		}
	}
}

func lightTheme() *Theme {
	if !GetTermCaps().HasColors() {
		return noColorTheme()
	}

	caps := GetTermCaps()

	switch caps.ColorDepth {
	case TrueColor:
		return &Theme{
			Primary:       lipgloss.NewStyle().Foreground(lipgloss.Color("#0184BC")),
			Secondary:     lipgloss.NewStyle().Foreground(lipgloss.Color("#287222")),
			Success:       lipgloss.NewStyle().Foreground(lipgloss.Color("#287222")),
			Warning:       lipgloss.NewStyle().Foreground(lipgloss.Color("#A65E08")),
			Error:         lipgloss.NewStyle().Foreground(lipgloss.Color("#D20F39")),
			Info:          lipgloss.NewStyle().Foreground(lipgloss.Color("#0184BC")),
			Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA0B0")),
			Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("#7629BB")),
			Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("#0184BC")).Bold(true),
			Border:        lipgloss.NewStyle().Foreground(lipgloss.Color("#BCC7D0")),
			StatusReady:   lipgloss.NewStyle().Foreground(lipgloss.Color("#287222")),
			StatusPending: lipgloss.NewStyle().Foreground(lipgloss.Color("#A65E08")),
			StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("#D20F39")),
			StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA0B0")),
		}
	case Color256:
		return &Theme{
			Primary:       lipgloss.NewStyle().Foreground(lipgloss.Color("32")),
			Secondary:     lipgloss.NewStyle().Foreground(lipgloss.Color("28")),
			Success:       lipgloss.NewStyle().Foreground(lipgloss.Color("28")),
			Warning:       lipgloss.NewStyle().Foreground(lipgloss.Color("130")),
			Error:         lipgloss.NewStyle().Foreground(lipgloss.Color("161")),
			Info:          lipgloss.NewStyle().Foreground(lipgloss.Color("32")),
			Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("246")),
			Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("56")),
			Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("32")).Bold(true),
			Border:        lipgloss.NewStyle().Foreground(lipgloss.Color("251")),
			StatusReady:   lipgloss.NewStyle().Foreground(lipgloss.Color("28")),
			StatusPending: lipgloss.NewStyle().Foreground(lipgloss.Color("130")),
			StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("161")),
			StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color("246")),
		}
	default:
		return &Theme{
			Primary:       lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
			Secondary:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			Success:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			Warning:       lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			Error:         lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
			Info:          lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
			Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
			Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true),
			Border:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			StatusReady:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			StatusPending: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
			StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		}
	}
}

func noColorTheme() *Theme {
	return &Theme{
		Primary:       lipgloss.NewStyle(),
		Secondary:     lipgloss.NewStyle(),
		Success:       lipgloss.NewStyle(),
		Warning:       lipgloss.NewStyle(),
		Error:         lipgloss.NewStyle(),
		Info:          lipgloss.NewStyle(),
		Muted:         lipgloss.NewStyle(),
		Highlight:     lipgloss.NewStyle(),
		Header:        lipgloss.NewStyle().Bold(true),
		Border:        lipgloss.NewStyle(),
		StatusReady:   lipgloss.NewStyle(),
		StatusPending: lipgloss.NewStyle(),
		StatusError:   lipgloss.NewStyle(),
		StatusUnknown: lipgloss.NewStyle(),
	}
}

func detectTheme(theme string) *Theme {
	switch theme {
	case "dark":
		return darkTheme()
	case "light":
		return lightTheme()
	case "auto":
		if isDarkMode() {
			return darkTheme()
		}
		return lightTheme()
	default:
		return darkTheme()
	}
}

func isDarkMode() bool {
	colorfgbg := os.Getenv("COLORFGBG")
	if colorfgbg != "" {
		parts := strings.Split(colorfgbg, ";")
		if len(parts) > 0 {
			bgColor := parts[len(parts)-1]
			bgNum, err := strconv.Atoi(bgColor)
			if err == nil {
				if bgNum < 8 {
					return true
				}
				return false
			}
		}
	}

	return true
}

func InitTheme(themeName string) {
	themeMutex.Lock()
	defer themeMutex.Unlock()

	currentTheme = detectTheme(themeName)
}

func GetTheme() *Theme {
	themeMutex.RLock()
	defer themeMutex.RUnlock()

	if currentTheme == nil {
		return darkTheme()
	}

	return currentTheme
}

func StatusStyle(kind, status string) lipgloss.Style {
	kind = strings.ToLower(kind)
	status = strings.ToLower(status)

	theme := GetTheme()

	switch kind {
	case "pod":
		switch status {
		case "running", "succeeded":
			return theme.StatusReady
		case "pending", "terminating":
			return theme.StatusPending
		case "failed", "crashloopbackoff", "unknown":
			return theme.StatusError
		default:
			return theme.StatusUnknown
		}
	case "node":
		switch status {
		case "ready":
			return theme.StatusReady
		case "notready", "noschedule":
			return theme.StatusError
		default:
			return theme.StatusUnknown
		}
	case "deployment", "statefulset", "daemonset":
		switch status {
		case "ready":
			return theme.StatusReady
		case "pending":
			return theme.StatusPending
		case "failed":
			return theme.StatusError
		default:
			return theme.StatusUnknown
		}
	case "pvc":
		switch status {
		case "bound":
			return theme.StatusReady
		case "pending":
			return theme.StatusPending
		case "failed", "lost":
			return theme.StatusError
		default:
			return theme.StatusUnknown
		}
	default:
		return theme.Primary
	}
}
