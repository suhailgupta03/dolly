package tmux

import "strings"

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
