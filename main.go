package main

import (
	"fmt"
	"log"
	"os"

	"tmux-manager/config"
	"tmux-manager/tmux"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <config.yml>")
	}

	configFile := os.Args[1]
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	err = tmux.CreateTmuxSession(cfg)
	if err != nil {
		log.Fatalf("Error creating tmux session: %v", err)
	}

	fmt.Printf("Tmux session '%s' created successfully with terminal '%s'!\n", cfg.SessionName, cfg.Terminal)
}