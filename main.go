package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"tmux-manager/config"
	"tmux-manager/prompt"
	"tmux-manager/registry"
	"tmux-manager/throwaway"
	"tmux-manager/tmux"
)

func main() {
	// Subcommand detection must happen before flag.Parse() so the FlagSet
	// for each subcommand can parse its own args independently.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "throwaway":
			handleThrowaway(os.Args[2:])
			return
		case "sessions":
			handleSessions(os.Args[2:])
			return
		}
	}

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
		fmt.Fprintf(os.Stderr, "\nSubcommands:\n")
		fmt.Fprintf(os.Stderr, "  throwaway [flags]        Create/manage disposable sessions\n")
		fmt.Fprintf(os.Stderr, "  sessions  [flags]        List all registered dolly sessions\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s my-project.yml                           # Create session from YAML\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -t my-project.yml                        # Terminate session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -e \"npm run dev, npm test\" -n myproject  # Quick session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s throwaway                                # Instant throwaway session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sessions                                 # List all sessions\n", os.Args[0])
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
		// Remove from registry; not fatal if absent
		if rerr := registry.RemoveEntry(cfg.SessionName); rerr != nil {
			log.Printf("Note: session '%s' was not in the registry", cfg.SessionName)
		}
		return
	}

	err = tmux.CreateTmuxSession(cfg)
	if err != nil {
		log.Fatalf("Error creating tmux session: %v", err)
	}

	fmt.Printf("Tmux session '%s' created successfully with terminal '%s'!\n", cfg.SessionName, cfg.Terminal)

	// Register the session
	absPath, _ := filepath.Abs(configFile)
	if rerr := registry.AddEntry(registry.Entry{
		Name:       cfg.SessionName,
		Type:       registry.TypeYAML,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		WorkingDir: cfg.WorkingDirectory,
		ConfigFile: absPath,
		Windows:    len(cfg.Windows),
		Terminal:   cfg.Terminal,
	}); rerr != nil {
		log.Printf("Warning: could not register session in registry: %v", rerr)
	}
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
		if rerr := registry.RemoveEntry(sessionName); rerr != nil {
			log.Printf("Note: session '%s' was not in the registry", sessionName)
		}
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

	// Register the exec session
	if rerr := registry.AddEntry(registry.Entry{
		Name:       cfg.SessionName,
		Type:       registry.TypeExec,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		WorkingDir: cfg.WorkingDirectory,
		Windows:    len(cfg.Windows),
		Terminal:   cfg.Terminal,
	}); rerr != nil {
		log.Printf("Warning: could not register session in registry: %v", rerr)
	}

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

// ── throwaway subcommand ──────────────────────────────────────────────────────

func handleThrowaway(args []string) {
	fs := flag.NewFlagSet("throwaway", flag.ExitOnError)

	windows := fs.Int("windows", throwaway.DefaultWindows, "Number of windows")
	panes := fs.Int("panes", throwaway.DefaultPanesPerWindow, "Panes per window")
	name := fs.String("name", "", "Session name (auto-generated if omitted)")
	dir := fs.String("dir", "", "Working directory (defaults to cwd)")
	list := fs.Bool("list", false, "List all throwaway sessions")
	kill := fs.String("kill", "", "Kill and unregister a throwaway session by name")
	cleanup := fs.Bool("cleanup", false, "Remove stale throwaway registry entries")
	days := fs.Int("days", registry.DefaultCleanupDays, "Inactivity threshold in days for -cleanup")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dolly throwaway [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dolly throwaway                          # instant session (2 windows, 2 panes)\n")
		fmt.Fprintf(os.Stderr, "  dolly throwaway -windows 3 -panes 2     # custom layout\n")
		fmt.Fprintf(os.Stderr, "  dolly throwaway -name debug              # named session\n")
		fmt.Fprintf(os.Stderr, "  dolly throwaway -list                    # list sessions\n")
		fmt.Fprintf(os.Stderr, "  dolly throwaway -kill tw-0401-143022     # kill + unregister\n")
		fmt.Fprintf(os.Stderr, "  dolly throwaway -cleanup                 # prune stale entries\n")
		fmt.Fprintf(os.Stderr, "  dolly throwaway -cleanup -days 14        # custom threshold\n")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	switch {
	case *list:
		handleThrowawayList()
	case *kill != "":
		handleThrowawayKill(*kill)
	case *cleanup:
		handleThrowawayCleanup(*days)
	default:
		handleThrowawayCreate(*name, *dir, *windows, *panes)
	}
}

func handleThrowawayCreate(name, dir string, windows, panes int) {
	created, err := throwaway.Create(name, dir, windows, panes)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Throwaway session '%s' created (%d windows, %d panes each)\n", created, windows, panes)
	fmt.Printf("Attach:  tmux attach -t %s\n", created)
	fmt.Printf("Kill:    dolly throwaway -kill %s\n", created)
}

func handleThrowawayList() {
	sessions, err := registry.ListSessions(registry.TypeThrowaway)
	if err != nil {
		log.Fatalf("Error listing sessions: %v", err)
	}
	if len(sessions) == 0 {
		fmt.Println("No throwaway sessions registered.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tWINDOWS\tLAST ACTIVE\tDIR")
	for _, s := range sessions {
		status := "dead"
		if s.Alive {
			status = "alive"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			s.Name, status, s.Windows,
			s.LastActive.Format("2006-01-02 15:04:05"),
			s.WorkingDir,
		)
	}
	w.Flush()
}

func handleThrowawayKill(name string) {
	// Kill the tmux session (warn if already dead, but continue to clean registry)
	if err := tmux.TerminateTmuxSession(name, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not terminate tmux session '%s': %v\n", name, err)
	}

	// Remove from registry
	if err := registry.RemoveEntry(name); err != nil {
		log.Fatalf("Error removing '%s' from registry: %v", name, err)
	}

	fmt.Printf("Session '%s' terminated and removed from registry.\n", name)
}

func handleThrowawayCleanup(days int) {
	removed, err := registry.CleanupStale(days, registry.TypeThrowaway)
	if err != nil {
		log.Fatalf("Error during cleanup: %v", err)
	}
	if len(removed) == 0 {
		fmt.Printf("No stale throwaway sessions found (threshold: %d days).\n", days)
		return
	}
	for _, name := range removed {
		fmt.Printf("Removed stale session: %s\n", name)
	}
	fmt.Printf("Removed %d stale throwaway %s (inactive for %d+ days).\n",
		len(removed), plural(len(removed), "entry", "entries"), days)
}

// ── sessions subcommand ───────────────────────────────────────────────────────

func handleSessions(args []string) {
	fs := flag.NewFlagSet("sessions", flag.ExitOnError)
	typeStr := fs.String("type", "", "Filter by type: throwaway, yaml, exec")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dolly sessions [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dolly sessions                # all registered sessions\n")
		fmt.Fprintf(os.Stderr, "  dolly sessions -type yaml     # only YAML sessions\n")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	var filter []registry.SessionType
	if *typeStr != "" {
		filter = []registry.SessionType{registry.SessionType(*typeStr)}
	}

	sessions, err := registry.ListSessions(filter...)
	if err != nil {
		log.Fatalf("Error listing sessions: %v", err)
	}
	if len(sessions) == 0 {
		if *typeStr != "" {
			fmt.Printf("No %s sessions registered.\n", *typeStr)
		} else {
			fmt.Println("No sessions registered.")
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tSTATUS\tWINDOWS\tLAST ACTIVE\tCONFIG\tDIR")
	for _, s := range sessions {
		status := "dead"
		if s.Alive {
			status = "alive"
		}
		cfgFile := s.ConfigFile
		if cfgFile == "" {
			cfgFile = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			s.Name, strings.ToUpper(string(s.Type)), status, s.Windows,
			s.LastActive.Format("2006-01-02 15:04:05"),
			cfgFile, s.WorkingDir,
		)
	}
	w.Flush()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func plural(n int, singular, pluralStr string) string {
	if n == 1 {
		return singular
	}
	return pluralStr
}
