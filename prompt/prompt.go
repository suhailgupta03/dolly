package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Reader handles interactive prompts from stdin
type Reader struct {
	scanner *bufio.Scanner
}

// NewReader creates a new prompt reader
func NewReader() *Reader {
	return &Reader{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// GetSessionName prompts for session name if not provided
func (r *Reader) GetSessionName(defaultName string) (string, error) {
	if defaultName != "" {
		return defaultName, nil
	}

	fmt.Print("Enter session name: ")
	if r.scanner.Scan() {
		name := strings.TrimSpace(r.scanner.Text())
		if name == "" {
			return "", fmt.Errorf("session name cannot be empty")
		}
		return name, nil
	}

	if err := r.scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return "", fmt.Errorf("no input provided")
}

// ConfirmSaveConfig prompts user to confirm saving config to YAML
func (r *Reader) ConfirmSaveConfig() (bool, error) {
	fmt.Print("Save session configuration to YAML file? [y/N]: ")
	if r.scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(r.scanner.Text()))
		return answer == "y" || answer == "yes", nil
	}

	if err := r.scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	return false, nil
}

// GetConfigFilePath prompts for config file path
func (r *Reader) GetConfigFilePath(defaultPath string) (string, error) {
	fmt.Printf("Enter config file path [%s]: ", defaultPath)
	if r.scanner.Scan() {
		path := strings.TrimSpace(r.scanner.Text())
		if path == "" {
			return defaultPath, nil
		}
		return path, nil
	}

	if err := r.scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return defaultPath, nil
}
