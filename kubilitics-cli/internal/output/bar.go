package output

import "fmt"

// ProgressBarString renders a text progress bar like: ██████████░░░░░░░░░░ 50%
func ProgressBarString(current, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}
	if width <= 0 {
		width = 20
	}

	pct := (current * 100) / total
	filled := (current * width) / total
	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}

	return fmt.Sprintf("%s %d%%", bar, pct)
}
