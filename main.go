package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"tmux-manager/config"
	"tmux-manager/prompt"
	"tmux-manager/tmux"
)

func main() {
	var terminate = flag.Bool("terminate", false, "Terminate the tmux session")
	var terminateShort = flag.Bool("t", false, "Terminate the tmux session (shorthand)")
	var help = flag.Bool("help", false, "Show help information")
	var helpShort = flag.Bool("h", false, "Show help information (shorthand)")

	// New flags for -exec mode
	var execCmds = flag.String("exec", "", "Comma-separated commands to run in panes")
	var execCmdsShort = flag.String("e", "", "Comma-separated commands (shorthand)")
	var sessionName = flag.String("name", "", "Session name for -exec mode")
	var sessionNameShort = flag.String("n", "", "Session name (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.yml]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -terminate, -t           Terminate the tmux session\n")
		fmt.Fprintf(os.Stderr, "  -exec, -e \"cmd1,cmd2\"    Create session with commands in panes\n")
		fmt.Fprintf(os.Stderr, "  -name, -n                Session name (for -exec mode)\n")
		fmt.Fprintf(os.Stderr, "  -help, -h                Show help information\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s my-project.yml                           # Create session from YAML\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -t my-project.yml                        # Terminate session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -e \"npm run dev, npm test\" -n myproject  # Quick session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -h                                       # Show help\n", os.Args[0])
	}

	flag.Parse()

	if *help || *helpShort {
		flag.Usage()
		return
	}

	// Consolidate short and long flags
	execStr := *execCmds
	if execStr == "" {
		execStr = *execCmdsShort
	}

	name := *sessionName
	if name == "" {
		name = *sessionNameShort
	}

	// Determine mode: exec mode vs config file mode
	if execStr != "" {
		handleExecMode(execStr, name, *terminate || *terminateShort)
		return
	}

	// Config file mode (existing logic)
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
		err = tmux.TerminateTmuxSession(cfg.SessionName, cfg.RcFile)
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

func handleExecMode(execStr, sessionName string, terminate bool) {
	commands := config.ParseCommands(execStr)
	if len(commands) == 0 {
		log.Fatal("Error: no commands provided to -exec")
	}

	reader := prompt.NewReader()

	var err error
	if sessionName == "" {
		sessionName, err = reader.GetSessionName("")
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	if terminate {
		err = tmux.TerminateTmuxSession(sessionName, "")
		if err != nil {
			log.Fatalf("Error terminating tmux session: %v", err)
		}
		fmt.Printf("Tmux session '%s' terminated successfully!\n", sessionName)
		return
	}

	cfg, err := config.BuildConfigFromCommands(sessionName, commands, "")
	if err != nil {
		log.Fatalf("Error building config: %v", err)
	}

	err = tmux.CreateTmuxSession(cfg)
	if err != nil {
		log.Fatalf("Error creating tmux session: %v", err)
	}

	fmt.Printf("Tmux session '%s' created successfully!\n", cfg.SessionName)

	save, err := reader.ConfirmSaveConfig()
	if err != nil {
		log.Printf("Warning: %v", err)
		return
	}

	if save {
		defaultPath := fmt.Sprintf("%s.yml", sessionName)
		configPath, err := reader.GetConfigFilePath(defaultPath)
		if err != nil {
			log.Printf("Warning: %v", err)
			return
		}

		err = config.SaveConfig(cfg, configPath)
		if err != nil {
			log.Fatalf("Error saving config: %v", err)
		}

		fmt.Printf("Configuration saved to '%s'\n", configPath)
	}
}