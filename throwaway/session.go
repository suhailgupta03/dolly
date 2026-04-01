package throwaway

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"tmux-manager/config"
	"tmux-manager/registry"
	"tmux-manager/tmux"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// GenerateName returns a unique throwaway session name based on current time.
// Format: tw-MMDD-HHMMSS (e.g. tw-0401-143022)
func GenerateName() string {
	return "tw-" + time.Now().Format("0102-150405")
}

// DetectShell returns the user's shell (zsh/fish/bash) from $SHELL, falling
// back to "bash" when the variable is unset or unrecognised.
func DetectShell() string {
	shell := filepath.Base(os.Getenv("SHELL"))
	switch shell {
	case "zsh", "fish", "bash":
		return shell
	default:
		return "bash"
	}
}

// BuildThrowawayConfig constructs a TmuxConfig for a throwaway session.
// Each window gets panesPerWindow side-by-side panes with no command set.
func BuildThrowawayConfig(name, workingDir string, numWindows, panesPerWindow int) (*config.TmuxConfig, error) {
	if numWindows < 1 {
		return nil, fmt.Errorf("windows must be >= 1, got %d", numWindows)
	}
	if panesPerWindow < 1 {
		return nil, fmt.Errorf("panes must be >= 1, got %d", panesPerWindow)
	}
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			workingDir = os.Getenv("HOME")
		}
	}

	windows := make([]config.Window, 0, numWindows)
	for w := 0; w < numWindows; w++ {
		panes := make([]config.Pane, 0, panesPerWindow)
		for p := 0; p < panesPerWindow; p++ {
			pane := config.Pane{
				ID:               fmt.Sprintf("p%d", p+1),
				Command:          "",
				WorkingDirectory: workingDir,
			}
			if p == 0 {
				pane.Split = "none"
			} else {
				pane.Split = "vertical"
				pane.SplitFrom = fmt.Sprintf("p%d", p)
			}
			panes = append(panes, pane)
		}
		windows = append(windows, config.Window{
			Name:  fmt.Sprintf("w%d", w+1),
			Panes: panes,
		})
	}

	return &config.TmuxConfig{
		SessionName:      name,
		WorkingDirectory: workingDir,
		Terminal:         DetectShell(),
		Windows:          windows,
	}, nil
}

// Create creates a throwaway tmux session and registers it in the dolly
// registry. Returns the resolved session name (auto-generated if empty).
func Create(name, workingDir string, numWindows, panesPerWindow int) (string, error) {
	if name == "" {
		name = GenerateName()
	} else if !validName.MatchString(name) {
		return "", fmt.Errorf("invalid session name %q: use only letters, digits, underscores, and hyphens", name)
	}

	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			workingDir = os.Getenv("HOME")
		}
	}

	cfg, err := BuildThrowawayConfig(name, workingDir, numWindows, panesPerWindow)
	if err != nil {
		return "", err
	}

	if err := tmux.CreateTmuxSession(cfg); err != nil {
		return "", fmt.Errorf("could not create tmux session: %w", err)
	}

	now := time.Now()
	if err := registry.AddEntry(registry.Entry{
		Name:       name,
		Type:       registry.TypeThrowaway,
		CreatedAt:  now,
		LastActive: now,
		WorkingDir: workingDir,
		Windows:    numWindows,
		Terminal:   cfg.Terminal,
	}); err != nil {
		// Registry failure is a warning, not fatal — session was already created
		fmt.Fprintf(os.Stderr, "Warning: could not register session in registry: %v\n", err)
	}

	return name, nil
}
