package output

import (
	"fmt"
	"os"
	"strings"
)

func FormatError(err error, context map[string]string, suggestion string) string {
	if err == nil {
		return ""
	}

	theme := GetTheme()
	var output strings.Builder

	output.WriteString("\n")
	output.WriteString(theme.Error.Render(fmt.Sprintf("Error: %v", err)))
	output.WriteString("\n")

	if len(context) > 0 {
		output.WriteString("\n")
		output.WriteString(theme.Primary.Render("Context:"))
		output.WriteString("\n")

		for key, value := range context {
			output.WriteString(fmt.Sprintf("  %s: %s\n", theme.Muted.Render(key), value))
		}
	}

	if suggestion != "" {
		output.WriteString("\n")
		output.WriteString(theme.Info.Render(fmt.Sprintf("Suggestion: %s", suggestion)))
		output.WriteString("\n")
	}

	output.WriteString("\n")

	return output.String()
}

func FormatWarning(message string, detail string) string {
	theme := GetTheme()
	var output strings.Builder

	output.WriteString("\n")
	output.WriteString(theme.Warning.Render(fmt.Sprintf("Warning: %s", message)))
	output.WriteString("\n")

	if detail != "" {
		output.WriteString(fmt.Sprintf("  %s\n", detail))
	}

	output.WriteString("\n")

	return output.String()
}

func PrintError(err error) {
	fmt.Fprint(os.Stderr, FormatError(err, nil, ""))
}

func PrintErrorWithContext(err error, context map[string]string) {
	fmt.Fprint(os.Stderr, FormatError(err, context, ""))
}

func PrintErrorWithSuggestion(err error, suggestion string) {
	fmt.Fprint(os.Stderr, FormatError(err, nil, suggestion))
}

func PrintWarning(message string) {
	fmt.Fprint(os.Stderr, FormatWarning(message, ""))
}

func PrintWarningDetail(message string, detail string) {
	fmt.Fprint(os.Stderr, FormatWarning(message, detail))
}

func ErrorIcon() string {
	return "✗"
}

func WarningIcon() string {
	return "⚠"
}

func SuccessIcon() string {
	return "✓"
}

func InfoIcon() string {
	return "ℹ"
}
