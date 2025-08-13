package tmux

import (
	"fmt"
	"os/exec"

	"tmux-manager/config"
)

// Default color palette for auto-coloring windows using user-friendly names
var defaultColorPalette = [15]string{
	"green",
	"blue", 
	"red",
	"yellow",
	"cyan",
	"magenta",
	"white",
	"black",
	"brightgreen",
	"brightblue",
	"brightred",
	"brightyellow",
	"brightcyan",
	"brightmagenta",
	"brightwhite",
}

// getAutoColor returns a color from the default palette based on window index
func getAutoColor(windowIndex int) string {
	return defaultColorPalette[windowIndex%len(defaultColorPalette)]
}

// shouldUseAutoColor checks if auto-coloring is enabled (default: true)
func shouldUseAutoColor(cfg *config.TmuxConfig) bool {
	if cfg.AutoColor == nil {
		return true // default is enabled
	}
	return *cfg.AutoColor
}

func setWindowColor(sessionName, windowName, color string) error {
	if color == "" {
		return nil
	}

	// Set the window tab background color in the status bar
	// This colors the "1:development", "2:monitoring" etc tabs at the bottom
	windowTarget := fmt.Sprintf("%s:%s", sessionName, windowName)
	
	// Set window status format with colored background
	cmd := exec.Command("tmux", "set-window-option", "-t", windowTarget, "window-status-style", fmt.Sprintf("bg=%s,fg=black", color))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set window tab color '%s' for window %s: %w", color, windowName, err)
	}

	// Also set the current window style to make it more visible when selected
	cmd = exec.Command("tmux", "set-window-option", "-t", windowTarget, "window-status-current-style", fmt.Sprintf("bg=bright%s,fg=black,bold", color))
	if err := cmd.Run(); err != nil {
		// If bright version fails, just use regular color
		exec.Command("tmux", "set-window-option", "-t", windowTarget, "window-status-current-style", fmt.Sprintf("bg=%s,fg=white,bold", color)).Run()
	}
	
	return nil
}

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

	// Determine the shell command to use
	shellCmd := GetShellCommand(cfg.Terminal)

	if firstPaneWorkingDir != "" {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", cfg.SessionName, "-n", firstWindow.Name, "-c", firstPaneWorkingDir, shellCmd)
	} else {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", cfg.SessionName, "-n", firstWindow.Name, shellCmd)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Setup panes for first window
	err := SetupWindowPanes(cfg.SessionName, firstWindow.Name, firstWindow.Panes, cfg.WorkingDirectory, cfg.Terminal)
	if err != nil {
		return fmt.Errorf("failed to setup panes for first window: %w", err)
	}

	// Apply color to first window
	globalWindowIndex := 0
	color := firstWindow.Color
	if color == "" && shouldUseAutoColor(cfg) {
		color = getAutoColor(globalWindowIndex)
	}
	globalWindowIndex++
	if err := setWindowColor(cfg.SessionName, firstWindow.Name, color); err != nil {
		return fmt.Errorf("failed to set color for first window: %w", err)
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
			cmd = exec.Command("tmux", "new-window", "-t", cfg.SessionName, "-n", window.Name, "-c", windowWorkingDir, shellCmd)
		} else {
			cmd = exec.Command("tmux", "new-window", "-t", cfg.SessionName, "-n", window.Name, shellCmd)
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create window '%s': %w", window.Name, err)
		}

		err = SetupWindowPanes(cfg.SessionName, window.Name, window.Panes, cfg.WorkingDirectory, cfg.Terminal)
		if err != nil {
			return fmt.Errorf("failed to setup panes for window '%s': %w", window.Name, err)
		}

		// Apply color to this window
		color := window.Color
		if color == "" && shouldUseAutoColor(cfg) {
			color = getAutoColor(globalWindowIndex)
		}
		globalWindowIndex++
		if err := setWindowColor(cfg.SessionName, window.Name, color); err != nil {
			return fmt.Errorf("failed to set color for window '%s': %w", window.Name, err)
		}

		// Select the first pane in the window
		cmd = exec.Command("tmux", "select-pane", "-t", fmt.Sprintf("%s:%d.0", cfg.SessionName, windowIndex+1))
		cmd.Run()
	}

	// Create streaming window if log streaming is enabled
	if cfg.LogStream.Enabled {
		err = CreateStreamingWindow(cfg)
		if err != nil {
			return fmt.Errorf("failed to create streaming window: %w", err)
		}

		// Start log streaming after all windows are created
		err = StartLogStreaming(cfg)
		if err != nil {
			return fmt.Errorf("failed to start log streaming: %w", err)
		}

		// Select the streaming window (now window 0)
		cmd = exec.Command("tmux", "select-window", "-t", fmt.Sprintf("%s:0", cfg.SessionName))
	} else {
		// Select first window (as before)
		cmd = exec.Command("tmux", "select-window", "-t", fmt.Sprintf("%s:1", cfg.SessionName))
	}

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to select first window: %w", err)
	}

	return nil
}

func TerminateTmuxSession(sessionName string) error {
	// Cleanup streaming files before terminating the session
	CleanupStreamingFiles(sessionName)

	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to terminate tmux session '%s': %w", sessionName, err)
	}
	return nil
}
