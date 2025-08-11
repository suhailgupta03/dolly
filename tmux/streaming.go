package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"tmux-manager/config"
)

// StreamingWindow creates a window dedicated to streaming logs from other panes
func CreateStreamingWindow(cfg *config.TmuxConfig) error {
	if !cfg.LogStream.Enabled {
		return nil
	}

	// Create streaming window as the first window
	sessionName := cfg.SessionName
	streamingWindowName := "logs"

	// Determine the shell command to use
	shellCmd := GetShellCommand(cfg.Terminal)

	workingDir := cfg.WorkingDirectory
	var cmd *exec.Cmd
	if workingDir != "" {
		cmd = exec.Command("tmux", "new-window", "-t", sessionName, "-n", streamingWindowName, "-c", workingDir, shellCmd)
	} else {
		cmd = exec.Command("tmux", "new-window", "-t", sessionName, "-n", streamingWindowName, shellCmd)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create streaming window: %w", err)
	}

	// Move the streaming window to be the first window
	moveCmd := exec.Command("tmux", "move-window", "-s", fmt.Sprintf("%s:%s", sessionName, streamingWindowName), "-t", fmt.Sprintf("%s:0", sessionName))
	if err := moveCmd.Run(); err != nil {
		return fmt.Errorf("failed to move streaming window to first position: %w", err)
	}

	// Small delay to ensure window is ready
	time.Sleep(100 * time.Millisecond)

	return nil
}

// StartLogStreaming begins streaming logs from specified windows and panes
func StartLogStreaming(cfg *config.TmuxConfig) error {
	if !cfg.LogStream.Enabled {
		return nil
	}

	sessionName := cfg.SessionName
	streamingWindow := "logs"

	// Build list of target panes to stream from
	targetPanes, err := getTargetPanes(cfg)
	if err != nil {
		return fmt.Errorf("failed to get target panes: %w", err)
	}

	if len(targetPanes) == 0 {
		return fmt.Errorf("no panes found to stream from")
	}

	// Create a streaming command that will capture all target panes
	streamingCmd := buildStreamingCommand(sessionName, targetPanes, cfg.LogStream.Grep)

	// Send the streaming command to the logs window
	cmd := exec.Command("tmux", "send-keys", "-t", fmt.Sprintf("%s:%s", sessionName, streamingWindow), streamingCmd, "Enter")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start log streaming: %w", err)
	}

	return nil
}

// getTargetPanes returns a list of pane identifiers to stream from
func getTargetPanes(cfg *config.TmuxConfig) ([]string, error) {
	var targetPanes []string
	sessionName := cfg.SessionName

	// Check if we should stream from all windows/panes
	streamAllWindows := contains(cfg.LogStream.Windows, "*")
	streamAllPanes := contains(cfg.LogStream.Panes, "*")

	for _, window := range cfg.Windows {
		// Skip if we're not streaming all windows and this window isn't in the list
		if !streamAllWindows && !contains(cfg.LogStream.Windows, window.Name) {
			continue
		}

		for i, pane := range window.Panes {
			paneID := pane.ID
			if paneID == "" {
				paneID = fmt.Sprintf("pane%d", i+1)
			}

			// Skip if we're not streaming all panes and this pane isn't in the list
			if !streamAllPanes && !contains(cfg.LogStream.Panes, paneID) {
				continue
			}

			// Get the actual tmux pane ID
			tmuxPaneID, err := getTmuxPaneIDByWindow(sessionName, window.Name, i)
			if err != nil {
				// If we can't get the pane ID, log but continue
				fmt.Printf("Warning: Could not get tmux pane ID for %s:%s.%d: %v\n", sessionName, window.Name, i, err)
				continue
			}

			targetPanes = append(targetPanes, tmuxPaneID)
		}
	}

	return targetPanes, nil
}

// getTmuxPaneIDByWindow gets the tmux pane ID for a specific window and pane index
func getTmuxPaneIDByWindow(sessionName, windowName string, paneIndex int) (string, error) {
	cmd := exec.Command("tmux", "display-message", "-t", fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex), "-p", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get tmux pane ID: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// buildStreamingCommand creates a command that will continuously stream logs from multiple panes
func buildStreamingCommand(sessionName string, targetPanes []string, grepKeywords []string) string {
	if len(targetPanes) == 0 {
		return "echo 'No panes to stream from'"
	}

	// Get the current working directory to find the script
	// Use the script relative to where the binary is executed
	scriptPath := "./tmux/stream_monitor.sh"

	// Build the command arguments: script_path session_name [grep_keywords...] -- pane1 pane2 ...
	args := []string{scriptPath, sessionName}

	// Add grep keywords if any
	if len(grepKeywords) > 0 {
		args = append(args, "--grep")
		args = append(args, grepKeywords...)
		args = append(args, "--")
	}

	args = append(args, targetPanes...)

	// Create the command string
	return strings.Join(args, " ")
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
