package output

import (
	"sort"
)

type Breakpoint int

const (
	XS Breakpoint = iota
	SM
	MD
	LG
	XL
)

func DetectBreakpoint(width int) Breakpoint {
	switch {
	case width < 80:
		return XS
	case width < 100:
		return SM
	case width < 120:
		return MD
	case width < 160:
		return LG
	default:
		return XL
	}
}

func BreakpointName(bp Breakpoint) string {
	switch bp {
	case XS:
		return "XS"
	case SM:
		return "SM"
	case MD:
		return "MD"
	case LG:
		return "LG"
	case XL:
		return "XL"
	default:
		return "Unknown"
	}
}

func AdaptColumns(columns []Column, width int) []Column {
	if len(columns) == 0 {
		return columns
	}

	if width <= 0 {
		width = TermWidth()
	}

	breakpoint := DetectBreakpoint(width)

	sorted := make([]Column, len(columns))
	copy(sorted, columns)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	var visible []Column
	totalWidth := 0
	minBorderWidth := 3

	for _, col := range sorted {
		colTotalWidth := col.MinWidth + minBorderWidth

		if breakpoint == XS && col.Priority > 3 {
			continue
		}
		if breakpoint == SM && col.Priority > 4 {
			continue
		}

		if totalWidth+colTotalWidth <= width {
			visible = append(visible, col)
			totalWidth += colTotalWidth
		} else if col.Priority == 1 {
			visible = append(visible, col)
			totalWidth += colTotalWidth
		}
	}

	if len(visible) == 0 && len(columns) > 0 {
		visible = []Column{columns[0]}
	}

	return visible
}

func GetRecommendedTableStyle(width int) TableStyle {
	switch DetectBreakpoint(width) {
	case XS, SM:
		return None
	case MD:
		return Minimal
	case LG, XL:
		return Rounded
	default:
		return Rounded
	}
}

func GetRecommendedTruncateWidth(width int) int {
	switch DetectBreakpoint(width) {
	case XS:
		return 15
	case SM:
		return 20
	case MD:
		return 30
	case LG:
		return 40
	case XL:
		return 50
	default:
		return 40
	}
}

func FitToWidth(text string, width int) string {
	if width <= 0 {
		return text
	}

	if len(text) <= width {
		return text
	}

	if width <= 3 {
		return "..."
	}

	return text[:width-1] + "…"
}

func IsSmallScreen() bool {
	width := TermWidth()
	return width < 100
}

func IsNarrowScreen() bool {
	width := TermWidth()
	return width < 80
}
