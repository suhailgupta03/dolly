package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"tmux-manager/config"
	"tmux-manager/prompt"
	"tmux-manager/registry"
	"tmux-manager/shortcuts"
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
		case "attach":
			handleAttach(os.Args[2:])
			return
		case "sync":
			handleSync(os.Args[2:])
			return
		case "shortcuts":
			handleShortcuts(os.Args[2:])
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
		fmt.Fprintf(os.Stderr, "  attach    [SESSION|-all|-list]   Adopt existing tmux sessions\n")
		fmt.Fprintf(os.Stderr, "  sync      [flags]                Sync registry with live tmux sessions\n")
		fmt.Fprintf(os.Stderr, "  shortcuts [add|remove|reset|sync] Manage pane command shortcuts\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s my-project.yml                           # Create session from YAML\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -t my-project.yml                        # Terminate session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -t my-session                            # Terminate by name (no YAML needed)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -e \"npm run dev, npm test\" -n myproject  # Quick session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s throwaway                                # Instant throwaway session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sessions                                 # List all sessions\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s attach -list                             # Discover unmanaged sessions\n", os.Args[0])
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

	arg := flag.Arg(0)

	// If -t is set and the argument is not an existing file, treat it as a
	// bare session name so that attached/throwaway/exec sessions can be
	// terminated without a YAML config file.
	if *terminate || *terminateShort {
		if _, err := os.Stat(arg); os.IsNotExist(err) {
			handleTerminateByName(arg)
			return
		}
	}

	cfg, err := config.LoadConfig(arg)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if *terminate || *terminateShort {
		err = tmux.TerminateTmuxSession(cfg.SessionName, cfg.RcFile)
		if err != nil {
			log.Fatalf("Error terminating tmux session: %v", err)
		}
		fmt.Printf("Tmux session '%s' terminated successfully!\n", cfg.SessionName)
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

	absPath, _ := filepath.Abs(arg)
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

// handleTerminateByName terminates a tmux session by bare name (no YAML needed).
// Used when -t is given a name that is not an existing file path.
func handleTerminateByName(name string) {
	if err := tmux.TerminateTmuxSession(name, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not terminate tmux session '%s': %v\n", name, err)
	} else {
		fmt.Printf("Tmux session '%s' terminated successfully!\n", name)
	}
	if rerr := registry.RemoveEntry(name); rerr != nil {
		log.Printf("Note: session '%s' was not in the registry", name)
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
	if err := tmux.TerminateTmuxSession(name, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not terminate tmux session '%s': %v\n", name, err)
	}
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

// ── attach subcommand ─────────────────────────────────────────────────────────

func handleAttach(args []string) {
	fs := flag.NewFlagSet("attach", flag.ExitOnError)

	all := fs.Bool("all", false, "Attach all unmanaged running tmux sessions")
	list := fs.Bool("list", false, "List unmanaged tmux sessions (not yet in dolly registry)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dolly attach [SESSION | -all | -list]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dolly attach work         # adopt session named 'work'\n")
		fmt.Fprintf(os.Stderr, "  dolly attach -all         # adopt all unmanaged sessions\n")
		fmt.Fprintf(os.Stderr, "  dolly attach -list        # discover unmanaged sessions\n")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	switch {
	case *list:
		handleAttachList()
	case *all:
		handleAttachAll()
	case fs.NArg() >= 1:
		handleAttachDirect(fs.Arg(0))
	default:
		fs.Usage()
		os.Exit(1)
	}
}

// handleAttachOne registers a single tmux session in the dolly registry.
// Returns (alreadyRegistered, error). The caller decides how to surface the
// alreadyRegistered flag to the user.
func handleAttachOne(name string) (alreadyRegistered bool, err error) {
	// Verify the session is actually running
	if !tmux.IsSessionAlive(name) {
		return false, fmt.Errorf("no tmux session named %q is currently running", name)
	}

	// Check for existing registry entry
	reg, err := registry.Load()
	if err != nil {
		return false, fmt.Errorf("could not load registry: %w", err)
	}
	for _, s := range reg.Sessions {
		if s.Name == name {
			alreadyRegistered = true
			break
		}
	}

	// Query tmux for current session metadata
	windows, workingDir, detailErr := tmux.GetSessionDetails(name)
	if detailErr != nil {
		log.Printf("Warning: could not read session details for '%s': %v", name, detailErr)
	}

	now := time.Now()
	if aerr := registry.AddEntry(registry.Entry{
		Name:       name,
		Type:       registry.TypeAttached,
		CreatedAt:  now,
		LastActive: now,
		WorkingDir: workingDir,
		Windows:    windows,
		Terminal:   tmux.DetectShell(),
	}); aerr != nil {
		return alreadyRegistered, fmt.Errorf("could not update registry: %w", aerr)
	}

	return alreadyRegistered, nil
}

func handleAttachDirect(name string) {
	// Load registry first so we can show the previous type if already registered
	reg, _ := registry.Load()
	var prevType registry.SessionType
	for _, s := range reg.Sessions {
		if s.Name == name {
			prevType = s.Type
			break
		}
	}

	already, err := handleAttachOne(name)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	if already {
		fmt.Printf("Warning: session '%s' was already registered (type: %s). Updating with current info.\n",
			name, strings.ToUpper(string(prevType)))
	}

	// Read back the saved entry to report accurate details
	reg2, _ := registry.Load()
	windows, workingDir := 0, ""
	for _, s := range reg2.Sessions {
		if s.Name == name {
			windows = s.Windows
			workingDir = s.WorkingDir
			break
		}
	}
	fmt.Printf("Session '%s' attached to dolly (%d windows, %s)\n", name, windows, workingDir)
}

func handleAttachAll() {
	sessions, err := tmux.ListSessions()
	if err != nil {
		log.Fatalf("Error listing tmux sessions: %v", err)
	}
	if len(sessions) == 0 {
		fmt.Println("No tmux sessions are currently running.")
		return
	}

	attached, skipped := 0, 0
	for _, name := range sessions {
		already, err := handleAttachOne(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error attaching '%s': %v\n", name, err)
			continue
		}
		if already {
			skipped++
			fmt.Printf("  skipped '%s' (already managed)\n", name)
		} else {
			attached++
			fmt.Printf("  attached '%s'\n", name)
		}
	}

	switch {
	case attached == 0 && skipped > 0:
		fmt.Println("All running tmux sessions are already managed by dolly.")
	case attached > 0 && skipped > 0:
		fmt.Printf("Attached %d %s. Skipped %d already-managed %s.\n",
			attached, plural(attached, "session", "sessions"),
			skipped, plural(skipped, "session", "sessions"))
	default:
		fmt.Printf("Attached %d %s.\n", attached, plural(attached, "session", "sessions"))
	}
}

func handleAttachList() {
	sessions, err := tmux.ListSessions()
	if err != nil {
		log.Fatalf("Error listing tmux sessions: %v", err)
	}
	if len(sessions) == 0 {
		fmt.Println("No tmux sessions running.")
		return
	}

	reg, err := registry.Load()
	if err != nil {
		log.Fatalf("Error loading registry: %v", err)
	}
	managed := make(map[string]bool, len(reg.Sessions))
	for _, s := range reg.Sessions {
		managed[s.Name] = true
	}

	var unmanaged []string
	for _, name := range sessions {
		if !managed[name] {
			unmanaged = append(unmanaged, name)
		}
	}

	if len(unmanaged) == 0 {
		fmt.Println("All running tmux sessions are already managed by dolly.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Println("Unmanaged tmux sessions (not in dolly registry):")
	fmt.Fprintln(w, "NAME\tWINDOWS\tDIR")
	for _, name := range unmanaged {
		wins, dir, detailErr := tmux.GetSessionDetails(name)
		if detailErr != nil {
			wins, dir = 0, "-"
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", name, wins, dir)
	}
	w.Flush()
	fmt.Println()
	fmt.Println(`Run "dolly attach -all" to attach all, or "dolly attach NAME" for one.`)
}

// ── sessions subcommand ───────────────────────────────────────────────────────

func handleSessions(args []string) {
	fs := flag.NewFlagSet("sessions", flag.ExitOnError)
	typeStr := fs.String("type", "", "Filter by type: throwaway, yaml, exec, attached")
	format := fs.String("format", "table", "Output format: table | json")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dolly sessions [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dolly sessions                    # all registered sessions\n")
		fmt.Fprintf(os.Stderr, "  dolly sessions -type yaml         # only YAML sessions\n")
		fmt.Fprintf(os.Stderr, "  dolly sessions -type attached     # only attached sessions\n")
		fmt.Fprintf(os.Stderr, "  dolly sessions -format json       # output as JSON\n")
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

	switch strings.ToLower(*format) {
	case "json":
		printSessionsJSON(sessions)
	default:
		printSessionsTable(sessions, *typeStr)
	}
}

func printSessionsTable(sessions []registry.SessionStatus, typeFilter string) {
	if len(sessions) == 0 {
		if typeFilter != "" {
			fmt.Printf("No %s sessions registered.\n", typeFilter)
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

func printSessionsJSON(sessions []registry.SessionStatus) {
	// Build a plain serialisable slice so Alive is included in the output.
	type jsonEntry struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Alive      bool   `json:"alive"`
		Windows    int    `json:"windows"`
		WorkingDir string `json:"working_dir"`
		ConfigFile string `json:"config_file,omitempty"`
		Terminal   string `json:"terminal"`
		CreatedAt  string `json:"created_at"`
		LastActive string `json:"last_active"`
	}

	out := make([]jsonEntry, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, jsonEntry{
			Name:       s.Name,
			Type:       string(s.Type),
			Alive:      s.Alive,
			Windows:    s.Windows,
			WorkingDir: s.WorkingDir,
			ConfigFile: s.ConfigFile,
			Terminal:   s.Terminal,
			CreatedAt:  s.CreatedAt.Format(time.RFC3339),
			LastActive: s.LastActive.Format(time.RFC3339),
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
	}
}

// ── sync subcommand ───────────────────────────────────────────────────────────

func handleSync(args []string) {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	adopt  := fs.Bool("adopt",   false, "Also adopt running sessions not in registry")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	format := fs.String("format", "table", "Output format: table | json")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dolly sync [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dolly sync               # prune dead registry entries\n")
		fmt.Fprintf(os.Stderr, "  dolly sync -adopt        # prune dead + adopt unregistered sessions\n")
		fmt.Fprintf(os.Stderr, "  dolly sync -dry-run      # preview changes\n")
		fmt.Fprintf(os.Stderr, "  dolly sync -format json  # JSON output\n")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// 1. Load registry once
	reg, err := registry.Load()
	if err != nil {
		log.Fatalf("Error loading registry: %v", err)
	}

	// 2. Get all live tmux sessions
	liveSessions, err := tmux.ListSessions()
	if err != nil {
		log.Fatalf("Error listing tmux sessions: %v", err)
	}
	liveSet := make(map[string]bool, len(liveSessions))
	for _, s := range liveSessions {
		liveSet[s] = true
	}

	// 3. Compute which registry entries are dead
	managedSet := make(map[string]bool, len(reg.Sessions))
	var kept []registry.Entry
	var removed []string
	for _, entry := range reg.Sessions {
		managedSet[entry.Name] = true
		if liveSet[entry.Name] {
			kept = append(kept, entry)
		} else {
			removed = append(removed, entry.Name)
		}
	}

	// 4. Compute which live sessions are unmanaged (only if -adopt)
	var adopted []string
	var newEntries []registry.Entry
	if *adopt {
		for _, name := range liveSessions {
			if !managedSet[name] {
				adopted = append(adopted, name)
				windows, workingDir, _ := tmux.GetSessionDetails(name)
				now := time.Now()
				newEntries = append(newEntries, registry.Entry{
					Name:       name,
					Type:       registry.TypeAttached,
					CreatedAt:  now,
					LastActive: now,
					WorkingDir: workingDir,
					Windows:    windows,
					Terminal:   tmux.DetectShell(),
				})
			}
		}
	}

	// 5. Single write: build final slice and save once (no-op in dry-run)
	if !*dryRun && (len(removed) > 0 || len(newEntries) > 0) {
		final := append(kept, newEntries...)
		reg.Sessions = final
		if serr := registry.Save(reg); serr != nil {
			log.Fatalf("Error saving registry: %v", serr)
		}
	}

	// 6. Output
	if strings.ToLower(*format) == "json" {
		printSyncJSON(removed, adopted, *dryRun)
	} else {
		printSyncTable(removed, adopted, *dryRun)
	}
}

func printSyncTable(removed, adopted []string, dryRun bool) {
	if len(removed) == 0 && len(adopted) == 0 {
		fmt.Println("Registry is already in sync with live tmux sessions.")
		return
	}
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	for _, name := range removed {
		fmt.Printf("%sWould remove: %s (dead)\n", prefix, name)
	}
	for _, name := range adopted {
		fmt.Printf("%sWould adopt: %s (unmanaged)\n", prefix, name)
	}
	if dryRun {
		fmt.Println("No changes made.")
	} else {
		fmt.Printf("Sync complete. Removed %d, adopted %d.\n", len(removed), len(adopted))
	}
}

func printSyncJSON(removed, adopted []string, dryRun bool) {
	if removed == nil {
		removed = []string{}
	}
	if adopted == nil {
		adopted = []string{}
	}
	out := struct {
		DryRun  bool     `json:"dry_run"`
		Removed []string `json:"removed"`
		Adopted []string `json:"adopted"`
	}{dryRun, removed, adopted}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

// ── shortcuts subcommand ─────────────────────────────────────────────────────

func handleShortcuts(args []string) {
	if len(args) == 0 {
		handleShortcutsList()
		return
	}

	switch args[0] {
	case "add":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: dolly shortcuts add NAME \"COMMAND\"\n")
			os.Exit(1)
		}
		handleShortcutsAdd(args[1], args[2])
	case "remove":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: dolly shortcuts remove NAME\n")
			os.Exit(1)
		}
		handleShortcutsRemove(args[1])
	case "reset":
		handleShortcutsReset()
	case "sync":
		handleShortcutsSync()
	default:
		fmt.Fprintf(os.Stderr, "Unknown shortcuts action: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage: dolly shortcuts [add|remove|reset|sync]\n")
		os.Exit(1)
	}
}

func handleShortcutsList() {
	global, err := shortcuts.LoadGlobal()
	if err != nil {
		log.Fatalf("Error loading global shortcuts: %v", err)
	}

	// Build combined list with source and group tracking
	type entry struct {
		group, name, source, command string
	}
	var entries []entry

	// Defaults first
	for name, cmd := range shortcuts.DefaultShortcuts {
		source := "default"
		if gcmd, ok := global[name]; ok {
			cmd = gcmd
			source = "global"
		}
		entries = append(entries, entry{shortcuts.GroupOf(name), name, source, cmd})
	}
	// Global-only (not overriding a default)
	for name, cmd := range global {
		if _, isDefault := shortcuts.DefaultShortcuts[name]; !isDefault {
			entries = append(entries, entry{"", name, "global", cmd})
		}
	}

	if len(entries) == 0 {
		fmt.Println("No shortcuts configured.")
		return
	}

	// Sort by group then name; ungrouped (global-only) entries sort last
	sort.Slice(entries, func(i, j int) bool {
		gi, gj := entries[i].group, entries[j].group
		if gi != gj {
			// empty group sorts after named groups
			if gi == "" {
				return false
			}
			if gj == "" {
				return true
			}
			return gi < gj
		}
		return entries[i].name < entries[j].name
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "GROUP\tNAME\tSOURCE\tCOMMAND")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.group, e.name, e.source, e.command)
	}
	w.Flush()
}

func handleShortcutsAdd(name, command string) {
	warn, err := shortcuts.AddGlobal(name, command)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	if warn != "" {
		fmt.Fprintf(os.Stderr, "%s\n", warn)
	}
	fmt.Printf("Shortcut '%s' added to global shortcuts.\n", name)
}

func handleShortcutsRemove(name string) {
	if err := shortcuts.RemoveGlobal(name); err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Shortcut '%s' removed from global shortcuts.\n", name)
}

func handleShortcutsReset() {
	if err := shortcuts.ResetGlobal(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Println("Global shortcuts reset. Built-in defaults will still apply.")
}

func handleShortcutsSync() {
	reg, err := registry.Load()
	if err != nil {
		log.Fatalf("Error loading registry: %v", err)
	}

	global, err := shortcuts.LoadGlobal()
	if err != nil {
		log.Fatalf("Error loading global shortcuts: %v", err)
	}

	// Merge defaults + globals — same result for every session.
	merged := shortcuts.Merge(shortcuts.DefaultShortcuts, global, nil)

	synced := 0
	for _, s := range reg.Sessions {
		if !tmux.IsSessionAlive(s.Name) {
			continue
		}
		path, err := shortcuts.WriteShellFile(s.Name, s.Terminal, merged)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error syncing '%s': %v\n", s.Name, err)
			continue
		}
		fmt.Printf("  synced  %s  →  %s\n", s.Name, path)
		synced++
	}

	if synced == 0 {
		fmt.Println("No live sessions to sync.")
		return
	}
	fmt.Printf("\n%d %s updated. Run this in each pane to apply:\n    source $DOLLY_SHORTCUTS_FILE\n",
		synced, plural(synced, "session", "sessions"))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func plural(n int, singular, pluralStr string) string {
	if n == 1 {
		return singular
	}
	return pluralStr
}
