package shortcuts

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var validName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

var shellBuiltins = map[string]bool{
	"cd": true, "ls": true, "rm": true, "cp": true, "mv": true,
	"cat": true, "echo": true, "exit": true, "source": true, "export": true,
	"test": true, "kill": true, "wait": true, "read": true, "eval": true,
	"exec": true, "set": true, "unset": true, "type": true, "alias": true,
}

func dollyDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".dolly")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("could not create %s: %w", dir, err)
	}
	return dir, nil
}

func globalFilePath() (string, error) {
	dir, err := dollyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "shortcuts.yml"), nil
}

// shortcutsFile is the YAML structure for ~/.dolly/shortcuts.yml
type shortcutsFile struct {
	Shortcuts map[string]string `yaml:"shortcuts"`
}

// LoadGlobal reads ~/.dolly/shortcuts.yml and returns the user's global
// shortcuts. Returns an empty map (not an error) if the file does not exist.
func LoadGlobal() (map[string]string, error) {
	path, err := globalFilePath()
	if err != nil {
		return map[string]string{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return map[string]string{}, fmt.Errorf("could not read %s: %w", path, err)
	}

	var f shortcutsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return map[string]string{}, fmt.Errorf("could not parse %s: %w", path, err)
	}
	if f.Shortcuts == nil {
		return map[string]string{}, nil
	}
	return f.Shortcuts, nil
}

// SaveGlobal writes the given shortcuts to ~/.dolly/shortcuts.yml atomically.
func SaveGlobal(shortcuts map[string]string) error {
	path, err := globalFilePath()
	if err != nil {
		return err
	}

	f := shortcutsFile{Shortcuts: shortcuts}
	data, err := yaml.Marshal(&f)
	if err != nil {
		return fmt.Errorf("could not marshal shortcuts: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("could not write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("could not rename temp file: %w", err)
	}
	return nil
}

// Merge combines shortcut layers. Priority: session > global > defaults.
// Any layer may be nil.
func Merge(defaults, global, session map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range global {
		merged[k] = v
	}
	for k, v := range session {
		merged[k] = v
	}
	return merged
}

// ValidateName checks that a shortcut name is a valid shell identifier.
// Returns an error if invalid, or a warning string if the name shadows a
// shell builtin (empty string means no warning).
func ValidateName(name string) (warning string, err error) {
	if !validName.MatchString(name) {
		return "", fmt.Errorf("invalid shortcut name %q: must be a valid shell identifier (letters, digits, underscores; no hyphens)", name)
	}
	if shellBuiltins[name] {
		return fmt.Sprintf("warning: %q shadows a shell builtin", name), nil
	}
	return "", nil
}

// AddGlobal adds or updates a shortcut in ~/.dolly/shortcuts.yml.
func AddGlobal(name, command string) (warning string, err error) {
	warn, err := ValidateName(name)
	if err != nil {
		return "", err
	}

	shortcuts, err := LoadGlobal()
	if err != nil {
		return "", err
	}
	shortcuts[name] = command
	if err := SaveGlobal(shortcuts); err != nil {
		return "", err
	}
	return warn, nil
}

// RemoveGlobal removes a shortcut from ~/.dolly/shortcuts.yml.
func RemoveGlobal(name string) error {
	shortcuts, err := LoadGlobal()
	if err != nil {
		return err
	}
	if _, exists := shortcuts[name]; !exists {
		return fmt.Errorf("shortcut %q not found in global shortcuts", name)
	}
	delete(shortcuts, name)
	return SaveGlobal(shortcuts)
}

// WriteShellFile writes merged shortcuts as shell functions to a session-scoped
// file under ~/.dolly/. Returns the file path. The file includes DOLLY_SESSION
// and DOLLY_SHORTCUTS_FILE environment variables for introspection.
func WriteShellFile(sessionName, terminal string, shortcuts map[string]string) (string, error) {
	if len(shortcuts) == 0 {
		return "", nil
	}

	dir, err := dollyDir()
	if err != nil {
		return "", err
	}

	isFish := strings.ToLower(terminal) == "fish"
	ext := ".sh"
	if isFish {
		ext = ".fish"
	}
	path := filepath.Join(dir, fmt.Sprintf(".shortcuts_%s%s", sessionName, ext))

	var b strings.Builder

	// Environment variables for introspection
	if isFish {
		fmt.Fprintf(&b, "set -gx DOLLY_SESSION %q\n", sessionName)
		fmt.Fprintf(&b, "set -gx DOLLY_SHORTCUTS_FILE %q\n", path)
	} else {
		fmt.Fprintf(&b, "export DOLLY_SESSION=%q\n", sessionName)
		fmt.Fprintf(&b, "export DOLLY_SHORTCUTS_FILE=%q\n", path)
	}
	b.WriteString("\n")

	// Write functions in sorted order for deterministic output
	names := make([]string, 0, len(shortcuts))
	for name := range shortcuts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cmd := shortcuts[name]
		if isFish {
			fmt.Fprintf(&b, "function %s\n    %s\nend\n\n", name, cmd)
		} else {
			fmt.Fprintf(&b, "function %s() {\n    %s\n}\n\n", name, cmd)
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", fmt.Errorf("could not write shortcuts file: %w", err)
	}
	return path, nil
}

// CleanupShellFile removes the session-scoped shortcuts file(s) for a session.
func CleanupShellFile(sessionName string) {
	dir, err := dollyDir()
	if err != nil {
		return
	}
	for _, ext := range []string{".sh", ".fish"} {
		path := filepath.Join(dir, fmt.Sprintf(".shortcuts_%s%s", sessionName, ext))
		os.Remove(path)
	}
}

// GroupOf returns the root-command group name for a built-in default shortcut
// (e.g. "grep" for "search"), or "" if the name is not a built-in default.
func GroupOf(name string) string {
	for group, g := range DefaultShortcutGroups {
		if _, ok := g.Shortcuts[name]; ok {
			return group
		}
	}
	return ""
}

// ResetGlobal deletes ~/.dolly/shortcuts.yml entirely.
func ResetGlobal() error {
	path, err := globalFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove %s: %w", path, err)
	}
	return nil
}
