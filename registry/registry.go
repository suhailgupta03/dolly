package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"tmux-manager/internal/dolly"
)

// registryPath returns the absolute path to ~/.dolly/registry.json,
// creating ~/.dolly/ if it does not exist.
func registryPath() (string, error) {
	dir, err := dolly.DataDir()
	if err != nil {
		return "", fmt.Errorf("could not determine dolly data directory: %w", err)
	}
	return filepath.Join(dir, "registry.json"), nil
}

// Load reads the registry from disk. Returns an empty Registry (not an error)
// when the file does not yet exist.
func Load() (*Registry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Registry{Sessions: []Entry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("registry is corrupted (%w); delete ~/.dolly/registry.json to reset", err)
	}
	if reg.Sessions == nil {
		reg.Sessions = []Entry{}
	}
	return &reg, nil
}

// Save writes the registry to disk atomically via a temp file + rename to
// prevent partial writes from corrupting the registry.
func Save(reg *Registry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not serialize registry: %w", err)
	}

	// Write to a temp file in the same directory then rename — atomic on POSIX
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "registry-*.json.tmp")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("could not write registry: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("could not close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("could not save registry: %w", err)
	}
	return nil
}

// AddEntry upserts an entry into the registry: if an entry with the same name
// already exists it is replaced; otherwise the entry is appended.
func AddEntry(entry Entry) error {
	reg, err := Load()
	if err != nil {
		return err
	}

	replaced := false
	for i, s := range reg.Sessions {
		if s.Name == entry.Name {
			reg.Sessions[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		reg.Sessions = append(reg.Sessions, entry)
	}

	return Save(reg)
}

// RemoveEntry removes the entry with the given name. Returns nil if the entry
// was not found (callers can decide whether to warn the user).
func RemoveEntry(name string) error {
	reg, err := Load()
	if err != nil {
		return err
	}

	filtered := reg.Sessions[:0]
	for _, s := range reg.Sessions {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}
	reg.Sessions = filtered

	return Save(reg)
}

// isSessionAlive probes tmux without leaking output to the user's terminal.
func isSessionAlive(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// ListSessions returns all sessions, optionally filtered by type.
// It probes each session for liveness, updates LastActive for alive ones,
// and performs a single Save at the end.
func ListSessions(typeFilter ...SessionType) ([]SessionStatus, error) {
	reg, err := Load()
	if err != nil {
		return nil, err
	}

	filter := toSet(typeFilter)
	statuses := make([]SessionStatus, 0)
	dirty := false

	for i, s := range reg.Sessions {
		if len(filter) > 0 && !filter[s.Type] {
			continue
		}
		alive := isSessionAlive(s.Name)
		if alive {
			reg.Sessions[i].LastActive = time.Now()
			dirty = true
		}
		statuses = append(statuses, SessionStatus{Entry: reg.Sessions[i], Alive: alive})
	}

	if dirty {
		if err := Save(reg); err != nil {
			return statuses, fmt.Errorf("warning: could not update registry: %w", err)
		}
	}

	return statuses, nil
}

// CleanupStale removes registry entries for sessions that are dead and whose
// LastActive timestamp is older than olderThanDays days. Optionally filtered
// by session type. Returns the names of removed entries.
func CleanupStale(olderThanDays int, typeFilter ...SessionType) ([]string, error) {
	reg, err := Load()
	if err != nil {
		return nil, err
	}

	threshold := time.Now().AddDate(0, 0, -olderThanDays)
	filter := toSet(typeFilter)
	var removed []string
	var kept []Entry

	for _, s := range reg.Sessions {
		// If type filter is set and this entry doesn't match, always keep it
		if len(filter) > 0 && !filter[s.Type] {
			kept = append(kept, s)
			continue
		}

		alive := isSessionAlive(s.Name)
		if !alive && s.LastActive.Before(threshold) {
			removed = append(removed, s.Name)
		} else {
			kept = append(kept, s)
		}
	}

	reg.Sessions = kept
	if reg.Sessions == nil {
		reg.Sessions = []Entry{}
	}

	if err := Save(reg); err != nil {
		return removed, fmt.Errorf("could not save registry after cleanup: %w", err)
	}

	return removed, nil
}

// toSet converts a slice of SessionType into a lookup map
func toSet(types []SessionType) map[SessionType]bool {
	if len(types) == 0 {
		return nil
	}
	m := make(map[SessionType]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}
