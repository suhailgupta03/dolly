package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"tmux-manager/config"
)

func getPaneWorkingDir(pane config.Pane, fallbackDir string) string {
	if pane.WorkingDirectory != "" {
		return pane.WorkingDirectory
	}
	return fallbackDir
}

func executePreHooks(sessionName, windowName string, paneIndex int, preHooks []string, terminal string) error {
	for _, hook := range preHooks {
		if hook == "" {
			continue
		}

		// Execute pre-hook command
		cmd := exec.Command("tmux", "send-keys", "-t", fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex), hook, "Enter")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute pre-hook '%s': %w", hook, err)
		}

		// Small delay to allow command to execute
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func executeCommand(sessionName, windowName string, paneIndex int, command string) error {
	if command == "" {
		return nil
	}

	// Send the command to the shell in the pane (shell should already be initialized with profile)
	cmd := exec.Command("tmux", "send-keys", "-t", fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex), command, "Enter")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send command to pane %d: %w", paneIndex, err)
	}
	return nil
}

func SetupWindowPanes(sessionName, windowName string, panes []config.Pane, workingDir, terminal string) error {
	if len(panes) == 0 {
		return nil
	}

	// Build pane ID to index map and validate configuration
	paneIDMap := make(map[string]int)
	paneIndexMap := make(map[int]string) // tmux pane index to pane ID

	// Assign IDs and validate
	for i, pane := range panes {
		// Auto-assign ID if not provided
		paneID := pane.ID
		if paneID == "" {
			paneID = fmt.Sprintf("pane%d", i+1)
		}

		// Check for duplicate IDs
		if _, exists := paneIDMap[paneID]; exists {
			return fmt.Errorf("duplicate pane ID: %s", paneID)
		}

		paneIDMap[paneID] = i
		paneIndexMap[i] = paneID
	}

	// Validate split_from references
	for i, pane := range panes {
		if pane.SplitFrom != "" {
			if _, exists := paneIDMap[pane.SplitFrom]; !exists {
				return fmt.Errorf("pane %d references unknown split_from ID: %s", i+1, pane.SplitFrom)
			}
		}
	}

	// Create panes in order, but handle split_from logic
	createdPanes := make(map[string]string) // pane ID to tmux pane ID (%xxx format)

	// Always create the first pane - get its tmux pane ID
	firstPane := panes[0]
	firstPaneID := firstPane.ID
	if firstPaneID == "" {
		firstPaneID = "pane1"
	}

	// Execute first pane commands
	if err := executePreHooks(sessionName, windowName, 0, firstPane.PreHooks, terminal); err != nil {
		return fmt.Errorf("failed to execute pre-hooks for first pane: %w", err)
	}
	if err := executeCommand(sessionName, windowName, 0, firstPane.Command); err != nil {
		return fmt.Errorf("failed to execute command for first pane: %w", err)
	}

	// Get the actual tmux pane ID for the first pane
	firstTmuxPaneID, err := getTmuxPaneID(sessionName, windowName, 0)
	if err != nil {
		return fmt.Errorf("failed to get tmux pane ID for first pane: %w", err)
	}
	createdPanes[firstPaneID] = firstTmuxPaneID

	// Create remaining panes
	for i, pane := range panes[1:] {
		configIndex := i + 1
		paneID := pane.ID
		if paneID == "" {
			paneID = fmt.Sprintf("pane%d", configIndex+1)
		}

		// Skip if split is "none" (only first pane should have this)
		if strings.ToLower(pane.Split) == "none" {
			fmt.Printf("Warning: Pane '%s' has split 'none' but is not the first pane. Skipping.\n", paneID)
			continue
		}

		// Determine which pane to split from
		var splitFromTmuxID string
		if pane.SplitFrom != "" {
			// Split from specified pane
			var exists bool
			splitFromTmuxID, exists = createdPanes[pane.SplitFrom]
			if !exists {
				return fmt.Errorf("cannot find created pane with ID '%s' to split from", pane.SplitFrom)
			}
		} else {
			// Split from the last created pane (backward compatibility)
			// For now, let's use the first pane as fallback
			splitFromTmuxID = firstTmuxPaneID
		}

		// Create the split using tmux pane ID
		newTmuxPaneID, err := createSplitPaneWithID(splitFromTmuxID, pane, workingDir, terminal)
		if err != nil {
			return fmt.Errorf("failed to create pane '%s': %w", paneID, err)
		}

		createdPanes[paneID] = newTmuxPaneID

		// Execute commands for the new pane - we need to find its index
		newPaneIndex, err := getTmuxPaneIndex(newTmuxPaneID)
		if err != nil {
			return fmt.Errorf("failed to get index for new pane '%s': %w", paneID, err)
		}

		if err := executePreHooks(sessionName, windowName, newPaneIndex, pane.PreHooks, terminal); err != nil {
			return fmt.Errorf("failed to execute pre-hooks for pane '%s': %w", paneID, err)
		}
		if err := executeCommand(sessionName, windowName, newPaneIndex, pane.Command); err != nil {
			return fmt.Errorf("failed to execute command for pane '%s': %w", paneID, err)
		}
	}

	return nil
}

func getTmuxPaneID(sessionName, windowName string, paneIndex int) (string, error) {
	cmd := exec.Command("tmux", "display-message", "-t", fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex), "-p", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get tmux pane ID: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func getTmuxPaneIndex(paneID string) (int, error) {
	cmd := exec.Command("tmux", "display-message", "-t", paneID, "-p", "#{pane_index}")
	output, err := cmd.Output()
	if err != nil {
		return -1, fmt.Errorf("failed to get tmux pane index: %w", err)
	}
	indexStr := strings.TrimSpace(string(output))
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		return -1, fmt.Errorf("failed to parse pane index '%s': %w", indexStr, err)
	}
	return index, nil
}

func createSplitPaneWithID(splitFromTmuxID string, pane config.Pane, workingDir, terminal string) (string, error) {
	// Determine split direction
	var splitFlag string
	switch strings.ToLower(pane.Split) {
	case "horizontal", "h":
		// User wants horizontal split (top/bottom) → use tmux -v
		splitFlag = "-v"
	case "vertical", "v":
		// User wants vertical split (side/side) → use tmux -h
		splitFlag = "-h"
	default:
		splitFlag = "-v" // default to horizontal split (top/bottom)
	}

	// Determine the shell command to use
	shellCmd := GetShellCommand(terminal)

	// Split from the specified pane using tmux pane ID
	paneWorkingDir := getPaneWorkingDir(pane, workingDir)
	var cmd *exec.Cmd

	if paneWorkingDir != "" {
		cmd = exec.Command("tmux", "split-window", "-t", splitFromTmuxID, splitFlag, "-c", paneWorkingDir, "-P", "-F", "#{pane_id}", shellCmd)
	} else {
		cmd = exec.Command("tmux", "split-window", "-t", splitFromTmuxID, splitFlag, "-P", "-F", "#{pane_id}", shellCmd)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to split from pane %s: %w", splitFromTmuxID, err)
	}

	newPaneID := strings.TrimSpace(string(output))
	return newPaneID, nil
}
