package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"tmux-manager/config"
	"tmux-manager/tmux"
)

func main() {
	fmt.Println("🧪 Starting Dolly test suite...")
	
	// Clean up any existing test files
	cleanupTestFiles()
	
	// Load test configuration
	testConfigPath := "test-config.yml"
	cfg, err := config.LoadConfig(testConfigPath)
	if err != nil {
		log.Fatalf("❌ Failed to load test config: %v", err)
	}
	
	fmt.Printf("✅ Loaded test config for session: %s\n", cfg.SessionName)
	
	// Create tmux session
	err = tmux.CreateTmuxSession(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create tmux session: %v", err)
	}
	
	fmt.Printf("✅ Created tmux session '%s' with terminal '%s'\n", cfg.SessionName, cfg.Terminal)
	
	// Wait for commands to execute
	fmt.Println("⏳ Waiting for commands to execute...")
	time.Sleep(3 * time.Second)
	
	// Verify session exists
	if !verifySessionExists(cfg.SessionName) {
		log.Fatal("❌ Tmux session does not exist")
	}
	fmt.Println("✅ Tmux session exists")
	
	// Verify windows exist
	if !verifyWindowsExist(cfg) {
		log.Fatal("❌ Not all windows exist")
	}
	fmt.Println("✅ All windows exist")
	
	// Wait a bit more for file operations
	time.Sleep(2 * time.Second)
	
	// Verify test files were created (indicating commands executed)
	if !verifyTestFiles() {
		log.Fatal("❌ Test files were not created - commands may not have executed")
	}
	fmt.Println("✅ All test files created - commands executed successfully")
	
	// Verify pre-hooks executed
	if !verifyPreHooks() {
		log.Fatal("❌ Pre-hook files were not created - pre-hooks may not have executed")
	}
	fmt.Println("✅ All pre-hook files created - pre-hooks executed successfully")
	
	// Clean up
	cleanupSession(cfg.SessionName)
	cleanupTestFiles()
	
	fmt.Println("🎉 All tests passed! Dolly is working correctly.")
}

func cleanupTestFiles() {
	testFiles := []string{
		"/tmp/dolly-test-pane1.txt",
		"/tmp/dolly-test-pane2.txt",
		"/tmp/dolly-test-w2p1.txt",
		"/tmp/dolly-test-prehook1.txt",
		"/tmp/dolly-test-prehook2.txt",
		"/tmp/dolly-test-pane2-hook.txt",
		"/tmp/dolly-test-w2-hook.txt",
	}
	
	for _, file := range testFiles {
		os.Remove(file)
	}
}

func verifySessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

func verifyWindowsExist(cfg *config.TmuxConfig) bool {
	for i, window := range cfg.Windows {
		windowIndex := i + 1
		cmd := exec.Command("tmux", "list-windows", "-t", cfg.SessionName, "-f", fmt.Sprintf("#{==:#{window_name},%s}", window.Name))
		output, err := cmd.Output()
		if err != nil || len(strings.TrimSpace(string(output))) == 0 {
			fmt.Printf("❌ Window '%s' (index %d) not found\n", window.Name, windowIndex)
			return false
		}
	}
	return true
}

func verifyTestFiles() bool {
	testFiles := []string{
		"/tmp/dolly-test-pane1.txt",
		"/tmp/dolly-test-pane2.txt",
		"/tmp/dolly-test-w2p1.txt",
	}
	
	for _, file := range testFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("❌ Test file %s does not exist\n", file)
			return false
		}
	}
	return true
}

func verifyPreHooks() bool {
	preHookFiles := []string{
		"/tmp/dolly-test-prehook1.txt",
		"/tmp/dolly-test-prehook2.txt",
		"/tmp/dolly-test-pane2-hook.txt",
		"/tmp/dolly-test-w2-hook.txt",
	}
	
	for _, file := range preHookFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("❌ Pre-hook file %s does not exist\n", file)
			return false
		}
	}
	return true
}

func cleanupSession(sessionName string) {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	cmd.Run()
	fmt.Printf("✅ Cleaned up tmux session '%s'\n", sessionName)
}