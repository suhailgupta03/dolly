package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// expandTilde expands ~ in a file path to the user's home directory
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	// Replace ~ with home directory
	return filepath.Join(homeDir, path[1:]), nil
}

// findNextAvailableAlias finds the next available alias name if there's a conflict
// Returns baseName-dolly, baseName-dolly-1, baseName-dolly-2, etc.
func findNextAvailableAlias(existingContent, baseName string) string {
	// Check if baseName-dolly is available
	suffix := baseName + "-dolly"
	aliasPattern := regexp.MustCompile(`^\s*alias\s+` + regexp.QuoteMeta(suffix) + `=`)

	lines := strings.Split(existingContent, "\n")
	found := false
	for _, line := range lines {
		if aliasPattern.MatchString(line) {
			found = true
			break
		}
	}

	if !found {
		return suffix
	}

	// Try baseName-dolly-1, baseName-dolly-2, etc.
	for i := 1; i < 100; i++ {
		suffix = fmt.Sprintf("%s-dolly-%d", baseName, i)
		aliasPattern = regexp.MustCompile(`^\s*alias\s+` + regexp.QuoteMeta(suffix) + `=`)

		found = false
		for _, line := range lines {
			if aliasPattern.MatchString(line) {
				found = true
				break
			}
		}

		if !found {
			return suffix
		}
	}

	// Fallback (should never reach here)
	return baseName + "-dolly-99"
}

// AddShellAlias adds a shell alias to the RC file for attaching to the tmux session
// Returns the actual alias name created (might be different if there was a conflict)
func AddShellAlias(rcFilePath, sessionName string) (string, error) {
	// Expand tilde in path
	expandedPath, err := expandTilde(rcFilePath)
	if err != nil {
		return "", err
	}

	// Check if RC file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("RC file not found: %s. Please create it first", expandedPath)
	}

	// Read current RC file contents
	content, err := os.ReadFile(expandedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read RC file: %w", err)
	}

	existingContent := string(content)
	aliasName := sessionName

	// Check if alias already exists (not managed by dolly)
	aliasPattern := regexp.MustCompile(`^\s*alias\s+` + regexp.QuoteMeta(sessionName) + `=`)
	dollyManagedPattern := regexp.MustCompile(`^\s*alias\s+.*=.*#\s*dolly-managed:\s*` + regexp.QuoteMeta(sessionName))

	lines := strings.Split(existingContent, "\n")
	hasConflict := false
	isDollyManaged := false

	for _, line := range lines {
		if dollyManagedPattern.MatchString(line) {
			isDollyManaged = true
			break
		}
		if aliasPattern.MatchString(line) {
			hasConflict = true
		}
	}

	// If alias exists and is not dolly-managed, create with suffix
	if hasConflict && !isDollyManaged {
		aliasName = findNextAvailableAlias(existingContent, sessionName)
	}

	// If already dolly-managed, remove old entry first
	if isDollyManaged {
		if err := RemoveShellAlias(rcFilePath, sessionName); err != nil {
			return "", fmt.Errorf("failed to remove old alias: %w", err)
		}
		// Re-read the file after removal
		content, err = os.ReadFile(expandedPath)
		if err != nil {
			return "", fmt.Errorf("failed to read RC file after removal: %w", err)
		}
		existingContent = string(content)
	}

	// Create the alias line
	aliasLine := fmt.Sprintf("alias %s='tmux attach -t %s' # dolly-managed: %s", aliasName, sessionName, sessionName)

	// Append to RC file
	newContent := existingContent
	if !strings.HasSuffix(newContent, "\n") && len(newContent) > 0 {
		newContent += "\n"
	}
	newContent += aliasLine + "\n"

	// Write back to RC file
	if err := os.WriteFile(expandedPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write to RC file: %w", err)
	}

	return aliasName, nil
}

// RemoveShellAlias removes the shell alias from the RC file
func RemoveShellAlias(rcFilePath, sessionName string) error {
	// Expand tilde in path
	expandedPath, err := expandTilde(rcFilePath)
	if err != nil {
		return err
	}

	// Check if RC file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		// Nothing to remove, return nil
		return nil
	}

	// Read current RC file contents
	content, err := os.ReadFile(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to read RC file: %w", err)
	}

	// Find and remove lines with dolly-managed marker for this session
	dollyManagedPattern := regexp.MustCompile(`^\s*alias\s+.*=.*#\s*dolly-managed:\s*` + regexp.QuoteMeta(sessionName) + `\s*$`)

	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		if !dollyManagedPattern.MatchString(line) {
			newLines = append(newLines, line)
		}
	}

	// Write back cleaned contents
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(expandedPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write to RC file: %w", err)
	}

	return nil
}
