package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// completionMarkerFile is the sentinel that indicates completion has been installed.
const completionMarkerFile = ".completion-installed"

// EnsureShellCompletion checks if shell completion is installed and installs it
// automatically on first run. Called once from main before command execution.
// It is intentionally silent on errors — completion is a convenience, not critical.
func EnsureShellCompletion() {
	// Skip in CI, non-interactive, or completion commands themselves
	if os.Getenv("KCLI_CI") == "true" || os.Getenv("CI") == "true" {
		return
	}

	kcliDir := kcliConfigDir()
	if kcliDir == "" {
		return
	}

	markerPath := filepath.Join(kcliDir, completionMarkerFile)
	if _, err := os.Stat(markerPath); err == nil {
		return // already installed
	}

	shell := detectUserShell()
	if shell == "" {
		return
	}

	// Create completion directory
	completionDir := filepath.Join(kcliDir, "completion")
	if err := os.MkdirAll(completionDir, 0o755); err != nil {
		return
	}

	// Generate and write the completion script
	var completionFile string
	var rcFile string
	var sourceLine string

	switch shell {
	case "zsh":
		completionFile = filepath.Join(completionDir, "_kcli")
		rcFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
		sourceLine = fmt.Sprintf("\n# kcli shell completion (auto-installed)\nfpath=(%s $fpath)\nautoload -Uz compinit && compinit\n", completionDir)
		// Generate zsh completion
		root := NewRootCommand()
		f, err := os.Create(completionFile)
		if err != nil {
			return
		}
		root.GenZshCompletion(f)
		f.Close()

	case "bash":
		completionFile = filepath.Join(completionDir, "kcli.bash")
		rcFile = filepath.Join(os.Getenv("HOME"), ".bashrc")
		sourceLine = fmt.Sprintf("\n# kcli shell completion (auto-installed)\nsource %s\n", completionFile)
		root := NewRootCommand()
		f, err := os.Create(completionFile)
		if err != nil {
			return
		}
		root.GenBashCompletionV2(f, true)
		f.Close()

	case "fish":
		completionFile = filepath.Join(os.Getenv("HOME"), ".config", "fish", "completions", "kcli.fish")
		if err := os.MkdirAll(filepath.Dir(completionFile), 0o755); err != nil {
			return
		}
		// Fish auto-loads from ~/.config/fish/completions/ — no rc edit needed
		root := NewRootCommand()
		f, err := os.Create(completionFile)
		if err != nil {
			return
		}
		root.GenFishCompletion(f, true)
		f.Close()
		rcFile = "" // fish doesn't need rc modification

	default:
		return
	}

	// Add source line to shell rc file (if not already present)
	if rcFile != "" {
		addSourceToRC(rcFile, sourceLine, "kcli shell completion")
	}

	// Write marker file so we don't do this again
	os.WriteFile(markerPath, []byte(shell), 0o644)

	// Print one-time message
	fmt.Fprintf(os.Stderr, "\033[36m✓ kcli: shell completion installed for %s\033[0m\n", shell)
	fmt.Fprintf(os.Stderr, "\033[90m  Restart your terminal or run: source <(kcli completion %s)\033[0m\n\n", shell)
}

func kcliConfigDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	dir := filepath.Join(home, ".kcli")
	os.MkdirAll(dir, 0o755)
	return dir
}

func detectUserShell() string {
	// Check SHELL env var
	shell := os.Getenv("SHELL")
	if shell != "" {
		base := filepath.Base(shell)
		switch base {
		case "zsh":
			return "zsh"
		case "bash":
			return "bash"
		case "fish":
			return "fish"
		}
	}
	// Fallback: check parent process name on macOS/Linux
	if ppid := os.Getppid(); ppid > 0 {
		if cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", ppid)); err == nil {
			cmd := strings.ToLower(string(cmdline))
			if strings.Contains(cmd, "zsh") {
				return "zsh"
			}
			if strings.Contains(cmd, "bash") {
				return "bash"
			}
			if strings.Contains(cmd, "fish") {
				return "fish"
			}
		}
	}
	return ""
}

func addSourceToRC(rcFile, sourceLine, marker string) {
	// Check if already present
	content, err := os.ReadFile(rcFile)
	if err == nil && strings.Contains(string(content), marker) {
		return // already sourced
	}

	// Append to rc file
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(sourceLine)
}
