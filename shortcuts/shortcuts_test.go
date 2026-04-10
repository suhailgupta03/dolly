package shortcuts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMerge(t *testing.T) {
	defaults := map[string]string{"gs": "git status", "gl": "git log"}
	global := map[string]string{"gs": "git status -sb", "deploy": "./deploy.sh"}
	session := map[string]string{"gs": "git status -v", "test": "go test ./..."}

	merged := Merge(defaults, global, session)

	// session overrides global overrides defaults
	if merged["gs"] != "git status -v" {
		t.Errorf("expected session override for gs, got %q", merged["gs"])
	}
	if merged["gl"] != "git log" {
		t.Errorf("expected default for gl, got %q", merged["gl"])
	}
	if merged["deploy"] != "./deploy.sh" {
		t.Errorf("expected global for deploy, got %q", merged["deploy"])
	}
	if merged["test"] != "go test ./..." {
		t.Errorf("expected session for test, got %q", merged["test"])
	}
}

func TestMergeNilDefaults(t *testing.T) {
	global := map[string]string{"deploy": "./deploy.sh"}
	session := map[string]string{"test": "go test ./..."}

	merged := Merge(nil, global, session)

	if _, ok := merged["gs"]; ok {
		t.Error("expected no defaults when defaults is nil")
	}
	if merged["deploy"] != "./deploy.sh" {
		t.Errorf("expected global shortcut, got %q", merged["deploy"])
	}
	if merged["test"] != "go test ./..." {
		t.Errorf("expected session shortcut, got %q", merged["test"])
	}
}

func TestMergeAllNil(t *testing.T) {
	merged := Merge(nil, nil, nil)
	if len(merged) != 0 {
		t.Errorf("expected empty map, got %d entries", len(merged))
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		wantWarn bool
	}{
		{"search", false, false},
		{"my_func", false, false},
		{"_private", false, false},
		{"gs", false, false},
		{"cd", false, true},   // builtin warning
		{"ls", false, true},   // builtin warning
		{"my-func", true, false},  // hyphens not allowed
		{"123abc", true, false},   // starts with digit
		{"", true, false},         // empty
		{"a b", true, false},      // space
	}

	for _, tt := range tests {
		warn, err := ValidateName(tt.name)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateName(%q): expected error", tt.name)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateName(%q): unexpected error: %v", tt.name, err)
		}
		if tt.wantWarn && warn == "" {
			t.Errorf("ValidateName(%q): expected warning", tt.name)
		}
		if !tt.wantWarn && warn != "" {
			t.Errorf("ValidateName(%q): unexpected warning: %s", tt.name, warn)
		}
	}
}

func TestLoadGlobalMissing(t *testing.T) {
	// Override home dir to a temp dir so we don't touch the real ~/.dolly
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sc, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal with missing file: %v", err)
	}
	if len(sc) != 0 {
		t.Errorf("expected empty map, got %d entries", len(sc))
	}
}

func TestAddAndRemoveGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Add
	warn, err := AddGlobal("mygrep", `grep -rn "$1" .`)
	if err != nil {
		t.Fatalf("AddGlobal: %v", err)
	}
	if warn != "" {
		t.Errorf("unexpected warning: %s", warn)
	}

	// Verify it was saved
	sc, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal after add: %v", err)
	}
	if sc["mygrep"] != `grep -rn "$1" .` {
		t.Errorf("expected mygrep shortcut, got %q", sc["mygrep"])
	}

	// Remove
	if err := RemoveGlobal("mygrep"); err != nil {
		t.Fatalf("RemoveGlobal: %v", err)
	}

	sc, err = LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal after remove: %v", err)
	}
	if _, ok := sc["mygrep"]; ok {
		t.Error("expected mygrep to be removed")
	}
}

func TestRemoveGlobalNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	err := RemoveGlobal("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent shortcut")
	}
}

func TestWriteShellFileBashZsh(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sc := map[string]string{
		"gs": "git status",
		"ff": `find . -type f -name "$1"`,
	}

	path, err := WriteShellFile("test-session", "zsh", sc)
	if err != nil {
		t.Fatalf("WriteShellFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read generated file: %v", err)
	}
	content := string(data)

	// Check env vars
	if !strings.Contains(content, `export DOLLY_SESSION="test-session"`) {
		t.Error("missing DOLLY_SESSION export")
	}
	if !strings.Contains(content, `export DOLLY_SHORTCUTS_FILE=`) {
		t.Error("missing DOLLY_SHORTCUTS_FILE export")
	}

	// Check function syntax
	if !strings.Contains(content, "function gs() {") {
		t.Error("missing bash/zsh function syntax for gs")
	}
	if !strings.Contains(content, "function ff() {") {
		t.Error("missing bash/zsh function syntax for ff")
	}

	// Check file path pattern
	if !strings.HasSuffix(path, ".shortcuts_test-session.sh") {
		t.Errorf("unexpected file path: %s", path)
	}
}

func TestWriteShellFileFish(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sc := map[string]string{
		"gs": "git status",
	}

	path, err := WriteShellFile("fish-session", "fish", sc)
	if err != nil {
		t.Fatalf("WriteShellFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read generated file: %v", err)
	}
	content := string(data)

	// Check fish env vars
	if !strings.Contains(content, `set -gx DOLLY_SESSION "fish-session"`) {
		t.Error("missing fish DOLLY_SESSION export")
	}
	if !strings.Contains(content, "set -gx DOLLY_SHORTCUTS_FILE") {
		t.Error("missing fish DOLLY_SHORTCUTS_FILE export")
	}

	// Check fish function syntax
	if !strings.Contains(content, "function gs\n") {
		t.Error("missing fish function syntax for gs")
	}
	if !strings.Contains(content, "\nend\n") {
		t.Error("missing fish function end")
	}

	// Check file extension
	if !strings.HasSuffix(path, ".fish") {
		t.Errorf("expected .fish extension, got %s", path)
	}
}

func TestWriteShellFileEmpty(t *testing.T) {
	path, err := WriteShellFile("empty", "bash", map[string]string{})
	if err != nil {
		t.Fatalf("WriteShellFile with empty map: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path for empty shortcuts, got %s", path)
	}
}

func TestCleanupShellFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sc := map[string]string{"gs": "git status"}
	path, err := WriteShellFile("cleanup-test", "bash", sc)
	if err != nil {
		t.Fatalf("WriteShellFile: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist before cleanup: %v", err)
	}

	CleanupShellFile("cleanup-test")

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be removed after cleanup")
	}
}

func TestDefaultShortcutDefsComplete(t *testing.T) {
	for group, g := range DefaultShortcutGroups {
		for name, def := range g.Shortcuts {
			if def.Command == "" {
				t.Errorf("[%s/%s] Command is empty", group, name)
			}
			if def.Description == "" {
				t.Errorf("[%s/%s] Description is empty", group, name)
			}
			if def.Example == "" {
				t.Errorf("[%s/%s] Example is empty", group, name)
			}
		}
	}
}

func TestResetGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create a global file first
	_, _ = AddGlobal("test", "echo hi")

	path := filepath.Join(tmp, ".dolly", "shortcuts.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("global file should exist: %v", err)
	}

	if err := ResetGlobal(); err != nil {
		t.Fatalf("ResetGlobal: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("global file should be removed after reset")
	}
}
