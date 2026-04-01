package throwaway

import (
	"fmt"
	"regexp"
	"testing"
)

var namePattern = regexp.MustCompile(`^tw-\d{4}-\d{6}$`)

func TestGenerateName_Format(t *testing.T) {
	name := GenerateName()
	if !namePattern.MatchString(name) {
		t.Fatalf("GenerateName() = %q, want format tw-MMDD-HHMMSS", name)
	}
}

func TestGenerateName_Unique(t *testing.T) {
	// Generate multiple names quickly — within the same second they may collide,
	// which is expected. We just verify the format is consistent.
	for i := 0; i < 5; i++ {
		name := GenerateName()
		if !namePattern.MatchString(name) {
			t.Fatalf("name %d %q does not match pattern", i, name)
		}
	}
}

func TestDetectShell_Fallback(t *testing.T) {
	// With no $SHELL set, should fall back to "bash"
	t.Setenv("SHELL", "")
	if got := DetectShell(); got != "bash" {
		t.Fatalf("DetectShell() = %q, want 'bash'", got)
	}
}

func TestDetectShell_Zsh(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	if got := DetectShell(); got != "zsh" {
		t.Fatalf("DetectShell() = %q, want 'zsh'", got)
	}
}

func TestDetectShell_Fish(t *testing.T) {
	t.Setenv("SHELL", "/usr/local/bin/fish")
	if got := DetectShell(); got != "fish" {
		t.Fatalf("DetectShell() = %q, want 'fish'", got)
	}
}

func TestBuildThrowawayConfig_Defaults(t *testing.T) {
	cfg, err := BuildThrowawayConfig("tw-test", "/tmp", DefaultWindows, DefaultPanesPerWindow)
	if err != nil {
		t.Fatalf("BuildThrowawayConfig: %v", err)
	}

	if cfg.SessionName != "tw-test" {
		t.Errorf("SessionName = %q, want 'tw-test'", cfg.SessionName)
	}
	if len(cfg.Windows) != DefaultWindows {
		t.Errorf("Windows = %d, want %d", len(cfg.Windows), DefaultWindows)
	}
	for wi, w := range cfg.Windows {
		if len(w.Panes) != DefaultPanesPerWindow {
			t.Errorf("window %d: Panes = %d, want %d", wi, len(w.Panes), DefaultPanesPerWindow)
		}
		// First pane must be "none"
		if w.Panes[0].Split != "none" {
			t.Errorf("window %d pane 0: Split = %q, want 'none'", wi, w.Panes[0].Split)
		}
		// Subsequent panes must be "vertical"
		for pi := 1; pi < len(w.Panes); pi++ {
			if w.Panes[pi].Split != "vertical" {
				t.Errorf("window %d pane %d: Split = %q, want 'vertical'", wi, pi, w.Panes[pi].Split)
			}
			expectedSplitFrom := fmt.Sprintf("p%d", pi)
			if w.Panes[pi].SplitFrom != expectedSplitFrom {
				t.Errorf("window %d pane %d: SplitFrom = %q, want %q", wi, pi, w.Panes[pi].SplitFrom, expectedSplitFrom)
			}
		}
	}
}

func TestBuildThrowawayConfig_WindowNames(t *testing.T) {
	cfg, err := BuildThrowawayConfig("tw-x", "/tmp", 3, 1)
	if err != nil {
		t.Fatalf("BuildThrowawayConfig: %v", err)
	}
	for i, w := range cfg.Windows {
		expected := fmt.Sprintf("w%d", i+1)
		if w.Name != expected {
			t.Errorf("window %d name = %q, want %q", i, w.Name, expected)
		}
	}
}

func TestBuildThrowawayConfig_InvalidWindows(t *testing.T) {
	_, err := BuildThrowawayConfig("tw-x", "/tmp", 0, 2)
	if err == nil {
		t.Fatal("expected error for windows=0, got nil")
	}
}

func TestBuildThrowawayConfig_InvalidPanes(t *testing.T) {
	_, err := BuildThrowawayConfig("tw-x", "/tmp", 2, 0)
	if err == nil {
		t.Fatal("expected error for panes=0, got nil")
	}
}

func TestBuildThrowawayConfig_WorkingDir(t *testing.T) {
	cfg, err := BuildThrowawayConfig("tw-x", "/my/project", 1, 1)
	if err != nil {
		t.Fatalf("BuildThrowawayConfig: %v", err)
	}
	if cfg.WorkingDirectory != "/my/project" {
		t.Errorf("WorkingDirectory = %q, want '/my/project'", cfg.WorkingDirectory)
	}
	if cfg.Windows[0].Panes[0].WorkingDirectory != "/my/project" {
		t.Errorf("Pane WorkingDirectory = %q, want '/my/project'", cfg.Windows[0].Panes[0].WorkingDirectory)
	}
}
