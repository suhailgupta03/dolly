package crashlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"tmux-manager/internal/dolly"
)

// logDir returns ~/.dolly/logs, creating it if absent (0700 — private).
func logDir() (string, error) {
	base, err := dolly.DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "logs")
	return dir, os.MkdirAll(dir, 0700)
}

func crashLogPath() (string, error) {
	d, err := logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "crashes.jsonl"), nil
}

func submittedPath() (string, error) {
	d, err := logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "submitted.json"), nil
}

// newID returns a unique crash entry ID.
func newID() string {
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), os.Getpid())
}

// LogCrash appends entry as a JSON line to ~/.dolly/logs/crashes.jsonl.
// It is a pure writer — callers must populate all fields before calling.
func LogCrash(entry CrashEntry) error {
	path, err := crashLogPath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	line, err := json.Marshal(entry)
	if err != nil {
		f.Close()
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	f.Close()
	if werr != nil {
		return werr
	}
	_ = rotate(path, 100) // best-effort; failure is silent
	return nil
}

// rotate keeps the JSONL file under max lines via a streaming read + atomic rewrite.
// f.Close() is called inline before any return — do NOT move or defer it.
func rotate(path string, max int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB per line for stack traces
	var lines []string
	for scanner.Scan() {
		if l := strings.TrimSpace(scanner.Text()); l != "" {
			lines = append(lines, l)
		}
	}
	f.Close() // must stay here; do NOT defer or move after the early returns below
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(lines) <= max {
		return nil
	}

	dropped := lines[:len(lines)-max]
	lines = lines[len(lines)-max:]
	// Best-effort: prune submitted sidecar for dropped entries.
	// If this fails, orphaned IDs persist in submitted.json until ClearCrashes — harm is nil.
	_ = pruneSubmitted(dropped)

	content := strings.Join(lines, "\n") + "\n"
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "crashes-*.jsonl.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// pruneSubmitted removes IDs of dropped JSONL lines from submitted.json.
func pruneSubmitted(droppedLines []string) error {
	dropSet := make(map[string]bool, len(droppedLines))
	for _, l := range droppedLines {
		var e CrashEntry
		if json.Unmarshal([]byte(l), &e) == nil && e.ID != "" {
			dropSet[e.ID] = true
		}
	}
	if len(dropSet) == 0 {
		return nil
	}

	existing, err := readSubmittedIDs()
	if err != nil {
		return err
	}
	for id := range dropSet {
		delete(existing, id)
	}
	return writeSubmittedIDs(existing)
}

// ReadCrashes returns up to limit entries from the crash log, newest last.
// limit == 0 returns all entries. Also returns the total number of entries before limiting.
func ReadCrashes(limit int) (entries []CrashEntry, total int, err error) {
	path, err := crashLogPath()
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var all []CrashEntry
	for scanner.Scan() {
		if l := strings.TrimSpace(scanner.Text()); l != "" {
			var e CrashEntry
			if json.Unmarshal([]byte(l), &e) == nil {
				all = append(all, e)
			}
		}
	}
	if serr := scanner.Err(); serr != nil {
		return nil, 0, serr
	}

	total = len(all)

	// Merge submitted sidecar — best-effort; empty map on error is fine
	submittedIDs, _ := readSubmittedIDs()
	for i := range all {
		all[i].Submitted = submittedIDs[all[i].ID]
	}

	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, total, nil
}

// readSubmittedIDs reads ~/.dolly/logs/submitted.json → map[id]true.
// Returns an empty map when the file does not exist.
func readSubmittedIDs() (map[string]bool, error) {
	path, err := submittedPath()
	if err != nil {
		return map[string]bool{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return map[string]bool{}, err
	}
	var doc struct {
		IDs []string `json:"ids"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return map[string]bool{}, err
	}
	m := make(map[string]bool, len(doc.IDs))
	for _, id := range doc.IDs {
		m[id] = true
	}
	return m, nil
}

// writeSubmittedIDs atomically writes the submitted ID set to submitted.json.
func writeSubmittedIDs(ids map[string]bool) error {
	path, err := submittedPath()
	if err != nil {
		return err
	}
	list := make([]string, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	doc := struct {
		IDs []string `json:"ids"`
	}{IDs: list}
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "submitted-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// MarkSubmitted marks the given crash entry IDs as submitted in the sidecar.
// This is the only write path for submitted.json — crashes.jsonl is never touched.
func MarkSubmitted(ids []string) error {
	existing, err := readSubmittedIDs()
	if err != nil {
		existing = map[string]bool{}
	}
	for _, id := range ids {
		existing[id] = true
	}
	return writeSubmittedIDs(existing)
}

// ClearCrashes removes both crashes.jsonl and submitted.json.
func ClearCrashes() error {
	crashPath, _ := crashLogPath()
	subPath, _ := submittedPath()
	if err := os.Remove(crashPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(subPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Fatal logs an internal error and exits with code 1.
// The submit hint is only shown when the log write succeeded.
// Callers must set all fields — LogCrash is a pure writer.
func Fatal(subcommand, version string, err error) {
	entry := CrashEntry{
		ID:           newID(),
		Timestamp:    time.Now(),
		Kind:         KindInternalError,
		Subcommand:   subcommand,
		Error:        err.Error(),
		DollyVersion: version,
		GOOS:         runtime.GOOS,
		GOArch:       runtime.GOARCH,
	}
	if lerr := LogCrash(entry); lerr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write crash log: %v\n", lerr)
	} else {
		fmt.Fprintln(os.Stderr, "Crash logged. Submit a bug report:\n  dolly report submit")
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// Exit prints an error message and exits with code 1 without logging.
// Use for user errors (bad input, missing files) where no crash report is needed.
func Exit(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// HandlePanic is designed to be called via defer at the top of main().
// It recovers from unexpected panics, logs a crash entry with stack trace,
// prints the submit hint, and exits with code 2.
func HandlePanic(subcommand, version string) {
	r := recover()
	if r == nil {
		return
	}
	entry := CrashEntry{
		ID:           newID(),
		Timestamp:    time.Now(),
		Kind:         KindPanic,
		Subcommand:   subcommand,
		Error:        fmt.Sprintf("panic: %v", r),
		DollyVersion: version,
		GOOS:         runtime.GOOS,
		GOArch:       runtime.GOARCH,
		StackTrace:   string(debug.Stack()),
	}
	if lerr := LogCrash(entry); lerr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write crash log: %v\n", lerr)
	} else {
		fmt.Fprintln(os.Stderr, "Crash logged. Submit a bug report:\n  dolly report submit")
	}
	fmt.Fprintf(os.Stderr, "Error: unexpected panic: %v\n", r)
	os.Exit(2)
}
