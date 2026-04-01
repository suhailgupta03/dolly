package tmux

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectShell returns the user's current shell (zsh/fish/bash) from $SHELL,
// falling back to "bash" when the variable is unset or unrecognised.
func DetectShell() string {
	shell := filepath.Base(os.Getenv("SHELL"))
	switch shell {
	case "zsh", "fish", "bash":
		return shell
	default:
		return "bash"
	}
}

// GetShellCommand returns the appropriate shell command based on terminal type
func GetShellCommand(terminal string) string {
	switch strings.ToLower(terminal) {
	case "zsh":
		return "zsh -l"
	case "fish":
		return "fish -l"
	case "bash":
		return "bash -l"
	default:
		return terminal + " -l"
	}
}
