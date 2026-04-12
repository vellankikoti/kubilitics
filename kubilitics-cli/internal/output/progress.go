package output

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	spinnerFrames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

type Spinner struct {
	message    string
	ticker     *time.Ticker
	stopChan   chan bool
	mu         sync.Mutex
	isRunning  bool
	lastFrame  int
	output     io.Writer
	useColor   bool
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:   message,
		stopChan:  make(chan bool, 1),
		output:    os.Stderr,
		useColor:  ShouldUseColor(false),
		isRunning: false,
	}
}

func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return
	}

	s.isRunning = true
	s.ticker = time.NewTicker(80 * time.Millisecond)

	go func() {
		frame := 0
		for {
			select {
			case <-s.stopChan:
				s.ticker.Stop()
				return
			case <-s.ticker.C:
				s.mu.Lock()
				frameChar := string(spinnerFrames[frame%len(spinnerFrames)])
				s.mu.Unlock()

				theme := GetTheme()
				spinnerText := theme.Primary.Render(frameChar)

				fmt.Fprintf(s.output, "\r%s %s", spinnerText, s.message)
				frame++
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.isRunning = false
	s.stopChan <- true

	fmt.Fprintf(s.output, "\r")
	fmt.Fprintf(s.output, "\033[K")
}

func (s *Spinner) StopWithSuccess(msg string) {
	s.Stop()

	theme := GetTheme()
	icon := theme.Success.Render("✓")
	fmt.Fprintf(s.output, "%s %s\n", icon, msg)
}

func (s *Spinner) StopWithError(msg string) {
	s.Stop()

	theme := GetTheme()
	icon := theme.Error.Render("✗")
	fmt.Fprintf(s.output, "%s %s\n", icon, msg)
}

type ProgressBar struct {
	width    int
	useColor bool
}

func NewProgressBar(width int) *ProgressBar {
	if width <= 0 {
		width = 30
	}

	return &ProgressBar{
		width:    width,
		useColor: ShouldUseColor(false),
	}
}

func (pb *ProgressBar) Render(current, total int) string {
	if total <= 0 {
		total = 100
	}

	if current < 0 {
		current = 0
	}

	if current > total {
		current = total
	}

	percent := (current * 100) / total

	filledWidth := (current * pb.width) / total
	emptyWidth := pb.width - filledWidth

	filled := ""
	empty := ""

	if pb.useColor {
		theme := GetTheme()
		filled = theme.Success.Render(fmt.Sprintf("%s", repeatChar("█", filledWidth)))
		empty = theme.Muted.Render(fmt.Sprintf("%s", repeatChar("░", emptyWidth)))
	} else {
		filled = repeatChar("█", filledWidth)
		empty = repeatChar("░", emptyWidth)
	}

	return fmt.Sprintf("[%s%s] %d%%", filled, empty, percent)
}

func (pb *ProgressBar) Update(current, total int) string {
	return pb.Render(current, total)
}

func repeatChar(char string, times int) string {
	result := ""
	for i := 0; i < times; i++ {
		result += char
	}
	return result
}

type ProgressTracker struct {
	current int
	total   int
	bar     *ProgressBar
	mu      sync.Mutex
}

func NewProgressTracker(total int) *ProgressTracker {
	if total <= 0 {
		total = 100
	}

	return &ProgressTracker{
		current: 0,
		total:   total,
		bar:     NewProgressBar(40),
	}
}

func (pt *ProgressTracker) Update(current int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if current >= 0 && current <= pt.total {
		pt.current = current
	}
}

func (pt *ProgressTracker) Increment() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.current < pt.total {
		pt.current++
	}
}

func (pt *ProgressTracker) Render() string {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	return pt.bar.Render(pt.current, pt.total)
}

func (pt *ProgressTracker) IsDone() bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	return pt.current >= pt.total
}
