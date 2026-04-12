package output

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	days := int64(d.Hours()) / 24
	hours := int64(d.Hours()) % 24
	minutes := int64(d.Minutes()) % 60
	seconds := int64(d.Seconds()) % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 && len(parts) < 2 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	if len(parts) == 0 {
		return "0s"
	}

	return strings.Join(parts, " ")
}

func FormatAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	duration := time.Since(t)

	if duration < 0 {
		return "future"
	}

	minutes := int64(duration.Minutes())
	hours := int64(duration.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30

	switch {
	case minutes < 1:
		return "now"
	case minutes < 60:
		return fmt.Sprintf("%dm ago", minutes)
	case hours < 24:
		return fmt.Sprintf("%dh ago", hours)
	case days < 7:
		return fmt.Sprintf("%dd ago", days)
	case weeks < 4:
		return fmt.Sprintf("%dw ago", weeks)
	case months < 12:
		return fmt.Sprintf("%dmo ago", months)
	default:
		years := months / 12
		return fmt.Sprintf("%dy ago", years)
	}
}

func FormatResourceQuantity(qty resource.Quantity) string {
	if qty.IsZero() {
		return "0"
	}

	milliValue := qty.MilliValue()

	switch {
	case milliValue < 1000:
		return fmt.Sprintf("%dm", milliValue)
	case milliValue < 1000000:
		return fmt.Sprintf("%.1f", float64(milliValue)/1000)
	default:
		return fmt.Sprintf("%.1f", float64(milliValue)/1000000)
	}
}

func FormatTimestamp(t time.Time, mode string) string {
	if t.IsZero() {
		return "-"
	}

	switch mode {
	case "relative":
		return FormatAge(t)
	case "iso8601":
		return t.UTC().Format(time.RFC3339)
	case "human":
		fallthrough
	default:
		now := time.Now()
		if t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day() {
			return t.Format("15:04:05")
		}
		if t.Year() == now.Year() {
			return t.Format("Jan 02 15:04")
		}
		return t.Format("2006-01-02 15:04")
	}
}

func FormatBytes(bytes int64) string {
	if bytes < 0 {
		return "-"
	}

	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	value := float64(bytes)

	for i, unit := range units {
		if value < 1024.0 {
			if i == 0 {
				return fmt.Sprintf("%d %s", int64(value), unit)
			}
			return fmt.Sprintf("%.1f %s", value, unit)
		}
		value /= 1024.0
	}

	return fmt.Sprintf("%.1f EiB", value)
}

func FormatCPU(cpu int64) string {
	if cpu == 0 {
		return "0"
	}

	if cpu%1000 == 0 {
		return fmt.Sprintf("%d", cpu/1000)
	}

	return fmt.Sprintf("%.1f", float64(cpu)/1000)
}

func FormatMemory(bytes int64) string {
	if bytes < 0 {
		return "-"
	}

	if bytes == 0 {
		return "0B"
	}

	units := []string{"B", "Ki", "Mi", "Gi", "Ti", "Pi"}
	value := float64(bytes)

	for i, unit := range units {
		if value < 1024.0 {
			if i == 0 {
				return fmt.Sprintf("%d%s", int64(value), unit)
			}
			return fmt.Sprintf("%.1f%s", value, unit)
		}
		value /= 1024.0
	}

	return fmt.Sprintf("%.1f%s", value, "Ei")
}

func FormatCount(count int) string {
	return fmt.Sprintf("%d", count)
}

func FormatPercent(current, total int) string {
	if total == 0 {
		return "0%"
	}

	percent := (current * 100) / total
	return fmt.Sprintf("%d%%", percent)
}

func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}

	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}

	return s[:maxLen-1] + "…"
}

func EllipsisString(s string, maxLen int) string {
	return TruncateString(s, maxLen)
}
