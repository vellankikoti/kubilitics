package kubectl

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// CommandType represents how kcli should handle a command
type CommandType int

const (
	TypePassthrough CommandType = iota // Direct kubectl exec
	TypeEnhanced                        // "with" modifier
	TypeNative                          // kcli-native command
)

// DefaultKubeconfigPath returns the default kubeconfig file path,
// checking KUBECONFIG env var first, then falling back to ~/.kube/config.
func DefaultKubeconfigPath() string {
	if p := os.Getenv("KUBECONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

// NativeCommands is the set of kcli-native commands that don't go through kubectl
var NativeCommands = map[string]bool{
	"ctx":        true,
	"ns":         true,
	"health":     true,
	"restarts":   true,
	"events":     true,
	"metrics":    true,
	"age":        true,
	"count":      true,
	"find":       true,
	"show":       true,
	"status":     true,
	"where":      true,
	"who":        true,
	"incident":   true,
	"logs":       true,
	"ui":         true,
	"config":     true,
	"diff":       true,
	"prompt":     true,
	"completion": true,
	"version":    true,
}

// ClassifyCommand determines how to handle the given args.
// Returns TypeNative if the first arg is a kcli-native command,
// TypeEnhanced if "with" appears in args, otherwise TypePassthrough.
func ClassifyCommand(args []string) CommandType {
	if len(args) == 0 {
		return TypePassthrough
	}

	firstArg := strings.ToLower(args[0])
	if NativeCommands[firstArg] {
		return TypeNative
	}

	for _, arg := range args {
		if strings.EqualFold(arg, "with") {
			return TypeEnhanced
		}
	}

	return TypePassthrough
}

// FindKubectl locates the kubectl binary.
// Checks: KCLI_KUBECTL env var, then searches PATH.
func FindKubectl(configPath string) (string, error) {
	if override := os.Getenv("KCLI_KUBECTL"); override != "" {
		if _, err := os.Stat(override); err == nil {
			return override, nil
		}
	}

	kubectlPath, err := exec.LookPath("kubectl")
	if err == nil {
		return kubectlPath, nil
	}

	commonPaths := []string{
		"/usr/local/bin/kubectl",
		"/usr/bin/kubectl",
		filepath.Join(os.Getenv("HOME"), ".local/bin/kubectl"),
		filepath.Join(os.Getenv("HOME"), "go/bin/kubectl"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("kubectl binary not found in PATH or common locations; set KCLI_KUBECTL environment variable")
}

// Execute runs kubectl with the given args, injecting context and namespace if not already present.
// It forwards stdin/stdout/stderr transparently, forwards signals to the subprocess,
// and returns the exit code.
func Execute(kubectlPath string, args []string, context, namespace string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmdArgs := make([]string, 0, len(args)+4)
	cmdArgs = append(cmdArgs, args...)

	hasContext := false
	hasNamespace := false
	for i, arg := range cmdArgs {
		if arg == "--context" && i+1 < len(cmdArgs) {
			hasContext = true
		}
		if (arg == "--namespace" || arg == "-n") && i+1 < len(cmdArgs) {
			hasNamespace = true
		}
	}

	if context != "" && !hasContext {
		cmdArgs = append(cmdArgs, "--context", context)
	}
	if namespace != "" && !hasNamespace {
		cmdArgs = append(cmdArgs, "--namespace", namespace)
	}

	cmd := exec.Command(kubectlPath, cmdArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = os.Environ()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start kubectl: %w", err)
	}

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	err := cmd.Wait()
	signal.Stop(sigChan)
	close(sigChan)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 1, fmt.Errorf("kubectl execution failed: %w", err)
		}
	}

	return exitCode, nil
}
