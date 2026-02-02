package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParseCommands parses a comma-separated command string into individual commands
func ParseCommands(execStr string) []string {
	if execStr == "" {
		return nil
	}

	parts := strings.Split(execStr, ",")

	commands := make([]string, 0, len(parts))
	for _, part := range parts {
		cmd := strings.TrimSpace(part)
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}

	return commands
}

// generatePaneID creates a pane ID from command using first and last characters
// For short commands, uses the whole sanitized command
// For longer commands, uses first 4 + ".." + last 4 characters
func generatePaneID(cmd string, usedIDs map[string]bool) string {
	// Sanitize: keep only alphanumeric, dash, underscore
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	sanitized := re.ReplaceAllString(cmd, "")

	if sanitized == "" {
		sanitized = "pane"
	}

	var baseID string
	const prefixLen = 4
	const suffixLen = 4
	minLenForSplit := prefixLen + suffixLen + 1 // need at least 9 chars to split meaningfully

	if len(sanitized) <= minLenForSplit {
		// Short command - use whole sanitized string
		baseID = sanitized
	} else {
		// Longer command - use first 4 + ".." + last 4
		baseID = sanitized[:prefixLen] + ".." + sanitized[len(sanitized)-suffixLen:]
	}

	// Ensure uniqueness
	finalID := baseID
	counter := 1
	for usedIDs[finalID] {
		counter++
		finalID = fmt.Sprintf("%s-%d", baseID, counter)
	}
	usedIDs[finalID] = true

	return finalID
}

// BuildConfigFromCommands creates a TmuxConfig from a list of commands
// Each command becomes a pane in a single window with vertical (side-by-side) splits
func BuildConfigFromCommands(sessionName string, commands []string, workingDir string) (*TmuxConfig, error) {
	if len(commands) == 0 {
		return nil, fmt.Errorf("no commands provided")
	}

	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	usedIDs := make(map[string]bool)
	paneIDs := make([]string, len(commands))

	// First pass: generate all pane IDs
	for i, cmd := range commands {
		paneIDs[i] = generatePaneID(cmd, usedIDs)
	}

	// Second pass: create panes with proper split_from references
	panes := make([]Pane, len(commands))
	for i, cmd := range commands {
		pane := Pane{
			ID:               paneIDs[i],
			Command:          cmd,
			WorkingDirectory: workingDir,
		}

		if i == 0 {
			pane.Split = "none"
		} else {
			pane.Split = "vertical"
			pane.SplitFrom = paneIDs[i-1]
		}

		panes[i] = pane
	}

	window := Window{
		Name:  "exec",
		Panes: panes,
	}

	cfg := &TmuxConfig{
		SessionName:      sessionName,
		WorkingDirectory: workingDir,
		Terminal:         "bash",
		Windows:          []Window{window},
	}

	return cfg, nil
}
