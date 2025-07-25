package tmux

import (
	"fmt"
	"os/exec"

	"tmux-manager/config"
)

func CreateTmuxSession(cfg *config.TmuxConfig) error {
	// Kill existing session if it exists
	exec.Command("tmux", "kill-session", "-t", cfg.SessionName).Run()

	// Create new session with first window
	if len(cfg.Windows) == 0 {
		return fmt.Errorf("no windows defined in config")
	}

	firstWindow := cfg.Windows[0]

	// Create session with working directory (use first pane's dir if specified, otherwise session's dir)
	var cmd *exec.Cmd
	firstPaneWorkingDir := cfg.WorkingDirectory
	if len(firstWindow.Panes) > 0 && firstWindow.Panes[0].WorkingDirectory != "" {
		firstPaneWorkingDir = firstWindow.Panes[0].WorkingDirectory
	}

	if firstPaneWorkingDir != "" {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", cfg.SessionName, "-n", firstWindow.Name, "-c", firstPaneWorkingDir)
	} else {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", cfg.SessionName, "-n", firstWindow.Name)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Setup panes for first window
	err := SetupWindowPanes(cfg.SessionName, firstWindow.Name, firstWindow.Panes, cfg.WorkingDirectory, cfg.Terminal)
	if err != nil {
		return fmt.Errorf("failed to setup panes for first window: %w", err)
	}

	// Create additional windows
	for i, window := range cfg.Windows[1:] {
		windowIndex := i + 1

		// Create new window with working directory (use first pane's dir if specified, otherwise session's dir)
		windowWorkingDir := cfg.WorkingDirectory
		if len(window.Panes) > 0 && window.Panes[0].WorkingDirectory != "" {
			windowWorkingDir = window.Panes[0].WorkingDirectory
		}

		if windowWorkingDir != "" {
			cmd = exec.Command("tmux", "new-window", "-t", cfg.SessionName, "-n", window.Name, "-c", windowWorkingDir)
		} else {
			cmd = exec.Command("tmux", "new-window", "-t", cfg.SessionName, "-n", window.Name)
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create window '%s': %w", window.Name, err)
		}

		err = SetupWindowPanes(cfg.SessionName, window.Name, window.Panes, cfg.WorkingDirectory, cfg.Terminal)
		if err != nil {
			return fmt.Errorf("failed to setup panes for window '%s': %w", window.Name, err)
		}

		// Select the first pane in the window
		cmd = exec.Command("tmux", "select-pane", "-t", fmt.Sprintf("%s:%d.0", cfg.SessionName, windowIndex+1))
		cmd.Run()
	}

	// Select first window
	cmd = exec.Command("tmux", "select-window", "-t", fmt.Sprintf("%s:1", cfg.SessionName))
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to select first window: %w", err)
	}

	return nil
}