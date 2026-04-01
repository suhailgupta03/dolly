package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testRegistryPath overrides the registry path to use a temp dir
func setupTestRegistry(t *testing.T) (cleanup func()) {
	t.Helper()
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	return func() {
		os.Setenv("HOME", orig)
	}
}

func makeEntry(name string, typ SessionType, daysAgo int, alive bool) Entry {
	t := time.Now().AddDate(0, 0, -daysAgo)
	return Entry{
		Name:       name,
		Type:       typ,
		CreatedAt:  t,
		LastActive: t,
		WorkingDir: "/tmp",
		Windows:    2,
		Terminal:   "zsh",
	}
}

// ─── Load on empty dir ───────────────────────────────────────────────────────

func TestLoad_EmptyDir(t *testing.T) {
	defer setupTestRegistry(t)()
	reg, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error, got: %v", err)
	}
	if len(reg.Sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(reg.Sessions))
	}
}

// ─── Save → Load round-trip ──────────────────────────────────────────────────

func TestSaveLoad_RoundTrip(t *testing.T) {
	defer setupTestRegistry(t)()

	reg := &Registry{Sessions: []Entry{makeEntry("test-session", TypeYAML, 0, true)}}
	if err := Save(reg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(loaded.Sessions))
	}
	if loaded.Sessions[0].Name != "test-session" {
		t.Fatalf("expected name 'test-session', got %q", loaded.Sessions[0].Name)
	}
}

// ─── AddEntry: append ────────────────────────────────────────────────────────

func TestAddEntry_Append(t *testing.T) {
	defer setupTestRegistry(t)()

	e1 := makeEntry("session-a", TypeThrowaway, 0, true)
	e2 := makeEntry("session-b", TypeExec, 0, true)

	if err := AddEntry(e1); err != nil {
		t.Fatalf("AddEntry e1: %v", err)
	}
	if err := AddEntry(e2); err != nil {
		t.Fatalf("AddEntry e2: %v", err)
	}

	reg, _ := Load()
	if len(reg.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(reg.Sessions))
	}
}

// ─── AddEntry: upsert (same name replaces) ───────────────────────────────────

func TestAddEntry_Upsert(t *testing.T) {
	defer setupTestRegistry(t)()

	e := makeEntry("session-x", TypeYAML, 5, true)
	if err := AddEntry(e); err != nil {
		t.Fatalf("AddEntry first: %v", err)
	}

	// Re-add with updated WorkingDir
	e.WorkingDir = "/new/path"
	e.CreatedAt = time.Now()
	if err := AddEntry(e); err != nil {
		t.Fatalf("AddEntry upsert: %v", err)
	}

	reg, _ := Load()
	if len(reg.Sessions) != 1 {
		t.Fatalf("expected 1 session after upsert, got %d", len(reg.Sessions))
	}
	if reg.Sessions[0].WorkingDir != "/new/path" {
		t.Fatalf("expected updated WorkingDir, got %q", reg.Sessions[0].WorkingDir)
	}
}

// ─── RemoveEntry: existing ───────────────────────────────────────────────────

func TestRemoveEntry_Existing(t *testing.T) {
	defer setupTestRegistry(t)()

	AddEntry(makeEntry("to-remove", TypeThrowaway, 0, true))
	AddEntry(makeEntry("to-keep", TypeThrowaway, 0, true))

	if err := RemoveEntry("to-remove"); err != nil {
		t.Fatalf("RemoveEntry: %v", err)
	}

	reg, _ := Load()
	if len(reg.Sessions) != 1 {
		t.Fatalf("expected 1 session after remove, got %d", len(reg.Sessions))
	}
	if reg.Sessions[0].Name != "to-keep" {
		t.Fatalf("wrong session remained: %q", reg.Sessions[0].Name)
	}
}

// ─── RemoveEntry: not found returns nil ──────────────────────────────────────

func TestRemoveEntry_NotFound(t *testing.T) {
	defer setupTestRegistry(t)()

	if err := RemoveEntry("ghost"); err != nil {
		t.Fatalf("RemoveEntry on missing entry should return nil, got: %v", err)
	}
}

// ─── CleanupStale: removes only dead+old entries ─────────────────────────────

func TestCleanupStale(t *testing.T) {
	defer setupTestRegistry(t)()

	// dead and old (8 days ago) — should be removed
	deadOld := makeEntry("dead-old", TypeThrowaway, 8, false)
	// dead but recent (3 days ago) — should be kept (grace period)
	deadRecent := makeEntry("dead-recent", TypeThrowaway, 3, false)
	// old but alive — should be kept
	aliveOld := makeEntry("alive-old", TypeThrowaway, 8, true)

	AddEntry(deadOld)
	AddEntry(deadRecent)
	AddEntry(aliveOld)

	// CleanupStale with threshold=7 days
	// We fake isSessionAlive by testing with entries whose names won't exist in tmux.
	// "alive-old" is tmux-dead too, but LastActive is also 8 days old — however
	// the test environment has no tmux so ALL sessions are "dead".
	// Adjust: make aliveOld's LastActive only 2 days old so it survives threshold.
	aliveOld.LastActive = time.Now().AddDate(0, 0, -2)
	AddEntry(aliveOld) // upsert

	removed, err := CleanupStale(7, TypeThrowaway)
	if err != nil {
		t.Fatalf("CleanupStale: %v", err)
	}

	// dead-old (8 days) should be removed; dead-recent (3 days) should be kept;
	// alive-old (2 days LastActive) should be kept
	if len(removed) != 1 || removed[0] != "dead-old" {
		t.Fatalf("expected [dead-old] removed, got %v", removed)
	}

	reg, _ := Load()
	if len(reg.Sessions) != 2 {
		t.Fatalf("expected 2 sessions remaining, got %d", len(reg.Sessions))
	}
}

// ─── CleanupStale: type filter ignores other types ───────────────────────────

func TestCleanupStale_TypeFilter(t *testing.T) {
	defer setupTestRegistry(t)()

	AddEntry(makeEntry("yaml-old", TypeYAML, 10, false))
	AddEntry(makeEntry("throwaway-old", TypeThrowaway, 10, false))

	removed, err := CleanupStale(7, TypeThrowaway)
	if err != nil {
		t.Fatalf("CleanupStale: %v", err)
	}

	// Only throwaway-old should be removed; yaml-old is outside the type filter
	if len(removed) != 1 || removed[0] != "throwaway-old" {
		t.Fatalf("expected [throwaway-old] removed, got %v", removed)
	}

	reg, _ := Load()
	if len(reg.Sessions) != 1 || reg.Sessions[0].Name != "yaml-old" {
		t.Fatalf("yaml-old should remain, got %v", reg.Sessions)
	}
}

// ─── Registry never corrupted: verify JSON is always valid after ops ─────────

func TestRegistry_NeverCorrupted(t *testing.T) {
	defer setupTestRegistry(t)()

	ops := []func(){
		func() { AddEntry(makeEntry("s1", TypeYAML, 0, true)) },
		func() { AddEntry(makeEntry("s2", TypeThrowaway, 0, true)) },
		func() { AddEntry(makeEntry("s1", TypeYAML, 1, true)) }, // upsert
		func() { RemoveEntry("s2") },
		func() { AddEntry(makeEntry("s3", TypeExec, 0, true)) },
		func() { CleanupStale(7) },
	}

	for i, op := range ops {
		op()
		// After every operation, verify the file is valid JSON
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".dolly", "registry.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("step %d: could not read registry file: %v", i, err)
		}
		var reg Registry
		if err := json.Unmarshal(data, &reg); err != nil {
			t.Fatalf("step %d: registry is corrupted after op: %v\nContent: %s", i, err, data)
		}
	}
}

// ─── ListSessions: type filter works ─────────────────────────────────────────

func TestListSessions_TypeFilter(t *testing.T) {
	defer setupTestRegistry(t)()

	AddEntry(makeEntry("yaml-1", TypeYAML, 0, false))
	AddEntry(makeEntry("tw-1", TypeThrowaway, 0, false))
	AddEntry(makeEntry("exec-1", TypeExec, 0, false))

	all, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(all))
	}

	throwaway, err := ListSessions(TypeThrowaway)
	if err != nil {
		t.Fatalf("ListSessions throwaway: %v", err)
	}
	if len(throwaway) != 1 || throwaway[0].Name != "tw-1" {
		t.Fatalf("expected [tw-1], got %v", throwaway)
	}
}
