package shortcuts

//go:generate go run ../cmd/gen-shortcuts/main.go

// ShortcutDef holds everything dolly knows about one built-in shortcut.
type ShortcutDef struct {
	Command     string // shell command template ($1, $2 for positional args)
	Description string // one-line human description shown in docs
	Example     string // ready-to-run invocation shown in docs (no $ prefix)
}

// ShortcutGroup holds shortcuts that share the same root CLI tool.
type ShortcutGroup struct {
	Shortcuts map[string]ShortcutDef
}

// DefaultShortcutGroups organises built-in shortcuts by their root command.
// Each group key is the root CLI tool (grep, find, tmux, …). Adding a new
// shortcut here automatically makes it available in every dolly-managed pane
// and in the generated docs/shortcuts.md.
var DefaultShortcutGroups = map[string]ShortcutGroup{
	"grep": {Shortcuts: map[string]ShortcutDef{
		"search": {
			Command:     `grep -rn "$1" .`,
			Description: "Recursively search for a pattern in all files under cwd",
			Example:     `search "TODO"`,
		},
		"searchi": {
			Command:     `grep -rni "$1" .`,
			Description: "Case-insensitive recursive search",
			Example:     `searchi "fixme"`,
		},
		"searchl": {
			Command:     `grep -rln "$1" .`,
			Description: "List only filenames containing a pattern",
			Example:     `searchl "import"`,
		},
		"searchw": {
			Command:     `grep -rn -w "$1" .`,
			Description: "Match whole words only (no partial matches)",
			Example:     `searchw "main"`,
		},
		"searchx": {
			Command:     `grep -rn --exclude-dir=".git" --exclude-dir="node_modules" --exclude-dir="vendor" "$1" .`,
			Description: "Recursive search excluding common dependency and VCS directories",
			Example:     `searchx "config"`,
		},
	}},
	"find": {Shortcuts: map[string]ShortcutDef{
		"ff": {
			Command:     `find . -type f -name "$1"`,
			Description: "Find files by name glob pattern",
			Example:     `ff "*.go"`,
		},
		"fd": {
			Command:     `find . -type d -name "$1"`,
			Description: "Find directories by name glob pattern",
			Example:     `fd "testdata"`,
		},
		"fnew": {
			Command:     `find . -newer "$1" -type f`,
			Description: "Find files newer than a given reference file",
			Example:     `fnew go.mod`,
		},
		"fsize": {
			Command:     `find . -type f -size +${1:-10}M`,
			Description: "Find files larger than SIZE MB (default: 10 MB)",
			Example:     `fsize 50`,
		},
	}},
	"tmux": {Shortcuts: map[string]ShortcutDef{
		"vsp": {
			Command:     `pane=$(tmux split-window -h -P -F "#{pane_id}") && tmux send-keys -t "$pane" "source $DOLLY_SHORTCUTS_FILE" Enter`,
			Description: "Split current pane — new pane opens to the right; shortcuts auto-sourced",
			Example:     `vsp`,
		},
		"sp": {
			Command:     `pane=$(tmux split-window -v -P -F "#{pane_id}") && tmux send-keys -t "$pane" "source $DOLLY_SHORTCUTS_FILE" Enter`,
			Description: "Split current pane — new pane opens below; shortcuts auto-sourced",
			Example:     `sp`,
		},
		"zoom": {
			Command:     `tmux resize-pane -Z`,
			Description: "Toggle current pane full-screen (run again to unzoom)",
			Example:     `zoom`,
		},
		"nw": {
			Command:     `pane=$(tmux new-window -P -F "#{pane_id}") && tmux send-keys -t "$pane" "source $DOLLY_SHORTCUTS_FILE" Enter`,
			Description: "Open a new window in the current session; shortcuts auto-sourced",
			Example:     `nw`,
		},
		"kp": {
			Command:     `tmux kill-pane`,
			Description: "Kill (close) the current pane",
			Example:     `kp`,
		},
		"rw": {
			Command:     `tmux rename-window "$1"`,
			Description: "Rename the current window",
			Example:     `rw backend`,
		},
	}},
}

// DefaultShortcuts is the flattened command-only map used by Merge and
// WriteShellFile. No callers outside this package need to change.
var DefaultShortcuts = flattenGroups(DefaultShortcutGroups)

func flattenGroups(groups map[string]ShortcutGroup) map[string]string {
	flat := make(map[string]string)
	for _, g := range groups {
		for name, def := range g.Shortcuts {
			flat[name] = def.Command
		}
	}
	return flat
}
