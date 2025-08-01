package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"tmux-manager/config"
	"tmux-manager/tmux"
)

func main() {
	var terminate = flag.Bool("terminate", false, "Terminate the tmux session")
	var terminateShort = flag.Bool("t", false, "Terminate the tmux session (shorthand)")
	var help = flag.Bool("help", false, "Show help information")
	var helpShort = flag.Bool("h", false, "Show help information (shorthand)")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <config.yml>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -terminate, -t    Terminate the tmux session\n")
		fmt.Fprintf(os.Stderr, "  -help, -h         Show help information\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s my-project.yml     # Create session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -t my-project.yml  # Terminate session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -h                 # Show help\n", os.Args[0])
	}
	
	flag.Parse()
	
	if *help || *helpShort {
		flag.Usage()
		return
	}
	
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	configFile := flag.Arg(0)
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if *terminate || *terminateShort {
		err = tmux.TerminateTmuxSession(cfg.SessionName)
		if err != nil {
			log.Fatalf("Error terminating tmux session: %v", err)
		}
		fmt.Printf("Tmux session '%s' terminated successfully!\n", cfg.SessionName)
		return
	}

	err = tmux.CreateTmuxSession(cfg)
	if err != nil {
		log.Fatalf("Error creating tmux session: %v", err)
	}

	fmt.Printf("Tmux session '%s' created successfully with terminal '%s'!\n", cfg.SessionName, cfg.Terminal)
}