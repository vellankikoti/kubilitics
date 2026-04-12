package output

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Prompter struct {
	mu       sync.Mutex
	useColor bool
	isTTY    bool
}

var prompter *Prompter
var promptMutex sync.Mutex

func getPrompter() *Prompter {
	promptMutex.Lock()
	defer promptMutex.Unlock()

	if prompter == nil {
		prompter = &Prompter{
			useColor: ShouldUseColor(false),
			isTTY:    GetTermCaps().IsTTY,
		}
	}

	return prompter
}

func isConfirmAnswered() (bool, bool) {
	if os.Getenv("KCLI_CONFIRM") == "false" {
		return true, false
	}

	return false, false
}

func ConfirmYesNo(message string, details map[string]string) (bool, error) {
	p := getPrompter()
	p.mu.Lock()
	defer p.mu.Unlock()

	answered, result := isConfirmAnswered()
	if answered {
		return result, nil
	}

	if !p.isTTY {
		return false, fmt.Errorf("not a TTY; cannot prompt for confirmation (use --yes flag to skip)")
	}

	theme := GetTheme()

	fmt.Fprintf(os.Stderr, "\n%s\n", theme.Warning.Render(message))

	for key, value := range details {
		fmt.Fprintf(os.Stderr, "  %s: %s\n", theme.Primary.Render(key), value)
	}

	fmt.Fprintf(os.Stderr, "\n%s ", theme.Primary.Render("Proceed? [y/N]"))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes", nil
}

func ConfirmCritical(message string, details map[string]string, typeName string) (bool, error) {
	p := getPrompter()
	p.mu.Lock()
	defer p.mu.Unlock()

	answered, result := isConfirmAnswered()
	if answered {
		return result, nil
	}

	if !p.isTTY {
		return false, fmt.Errorf("not a TTY; cannot prompt for confirmation (use --yes flag to skip)")
	}

	theme := GetTheme()

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%s\n", theme.Error.Render("⚠  CRITICAL OPERATION"))
	fmt.Fprintf(os.Stderr, "%s\n", theme.Error.Render(message))

	for key, value := range details {
		fmt.Fprintf(os.Stderr, "  %s: %s\n", theme.Primary.Render(key), value)
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", theme.Muted.Render(fmt.Sprintf("Type '%s' to confirm:", typeName)))
	fmt.Fprintf(os.Stderr, "%s ", theme.Primary.Render("> "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(input)

	fmt.Fprintf(os.Stderr, "\n")

	return input == typeName, nil
}

func ShowConfirmation(message string, details map[string]string) {
	theme := GetTheme()

	width := TermWidth()
	if width > 80 {
		width = 80
	}

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%s\n", theme.Primary.Render(message))

	maxKeyLen := 0
	for key := range details {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	for key, value := range details {
		padding := strings.Repeat(" ", maxKeyLen-len(key))
		fmt.Fprintf(os.Stderr, "  %s%s: %s\n", key, padding, value)
	}

	fmt.Fprintf(os.Stderr, "\n")
}

func SelectFromList(message string, options []string) (int, error) {
	p := getPrompter()
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isTTY {
		if len(options) > 0 {
			return 0, nil
		}
		return -1, fmt.Errorf("not a TTY; cannot prompt for selection")
	}

	theme := GetTheme()

	fmt.Fprintf(os.Stderr, "\n%s\n", theme.Primary.Render(message))

	for i, option := range options {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, option)
	}

	fmt.Fprintf(os.Stderr, "%s ", theme.Primary.Render("Select [1]:"))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return -1, err
	}

	input = strings.TrimSpace(input)

	if input == "" {
		return 0, nil
	}

	var selection int
	_, err = fmt.Sscanf(input, "%d", &selection)
	if err != nil {
		return -1, fmt.Errorf("invalid selection")
	}

	if selection < 1 || selection > len(options) {
		return -1, fmt.Errorf("selection out of range")
	}

	return selection - 1, nil
}

func PromptString(prompt string, defaultValue string) (string, error) {
	p := getPrompter()
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isTTY {
		return defaultValue, nil
	}

	theme := GetTheme()

	if defaultValue != "" {
		fmt.Fprintf(os.Stderr, "%s [%s] ", theme.Primary.Render(prompt), defaultValue)
	} else {
		fmt.Fprintf(os.Stderr, "%s ", theme.Primary.Render(prompt))
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue, nil
	}

	return input, nil
}
