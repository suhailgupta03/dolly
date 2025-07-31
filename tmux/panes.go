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

	// Execute pre-hooks and command for first pane
	if err := executePreHooks(sessionName, windowName, 0, panes[0].PreHooks, terminal); err != nil {
		return fmt.Errorf("failed to execute pre-hooks for first pane: %w", err)
	}

	if err := executeCommand(sessionName, windowName, 0, panes[0].Command); err != nil {
		return fmt.Errorf("failed to execute command for first pane: %w", err)
	}

	// Create additional panes
	for i, pane := range panes[1:] {
		paneIndex := i + 1

		// Split the pane
		var splitFlag string
		switch strings.ToLower(pane.Split) {
		case "horizontal", "h":
			splitFlag = "-h"
		case "vertical", "v":
			splitFlag = "-v"
		default:
			splitFlag = "-v" // default to vertical split
		}

		// Determine the shell command to use
		var shellCmd string
		switch strings.ToLower(terminal) {
		case "zsh":
			shellCmd = "zsh -l"
		case "fish":
			shellCmd = "fish -l"
		case "bash":
			shellCmd = "bash -l"
		default:
			shellCmd = terminal + " -l"
		}

		// Split the pane with per-pane working directory or fallback
		var cmd *exec.Cmd
		paneWorkingDir := getPaneWorkingDir(pane, workingDir)
		if paneWorkingDir != "" {
			cmd = exec.Command("tmux", "split-window", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), splitFlag, "-c", paneWorkingDir, shellCmd)
		} else {
			cmd = exec.Command("tmux", "split-window", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), splitFlag, shellCmd)
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to split pane: %w", err)
		}

		// Execute pre-hooks for the new pane
		if err := executePreHooks(sessionName, windowName, paneIndex, pane.PreHooks, terminal); err != nil {
			return fmt.Errorf("failed to execute pre-hooks for pane %d: %w", paneIndex, err)
		}

		// Execute command for the new pane
		if err := executeCommand(sessionName, windowName, paneIndex, pane.Command); err != nil {
			return fmt.Errorf("failed to execute command for pane %d: %w", paneIndex, err)
		}
	}

	return nil
}