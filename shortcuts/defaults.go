package shortcuts

// ShortcutGroup holds shortcuts that share the same root CLI tool.
type ShortcutGroup struct {
	// Shortcuts maps shortcut name → shell command template.
	Shortcuts map[string]string
}

// DefaultShortcutGroups organises built-in shortcuts by their root command.
// Each group key is the root CLI tool (grep, find, …). Adding a new shortcut
// here automatically makes it available in every dolly-managed pane.
var DefaultShortcutGroups = map[string]ShortcutGroup{
	"grep": {
		Shortcuts: map[string]string{
			"search":  `grep -rn "$1" .`,                                                                                          // recursively search for a pattern in all files
			"searchi": `grep -rni "$1" .`,                                                                                         // case-insensitive recursive search
			"searchl": `grep -rln "$1" .`,                                                                                         // list only filenames that contain a pattern
			"searchw": `grep -rn -w "$1" .`,                                                                                       // match whole words only (no partial matches)
			"searchx": `grep -rn --exclude-dir=".git" --exclude-dir="node_modules" --exclude-dir="vendor" "$1" .`,                 // search excluding common dependency/vcs dirs
		},
	},
	"find": {
		Shortcuts: map[string]string{
			"ff":    `find . -type f -name "$1"`,        // find files by name glob pattern
			"fd":    `find . -type d -name "$1"`,        // find directories by name glob pattern
			"fnew":  `find . -newer "$1" -type f`,       // find files newer than a reference file
			"fsize": `find . -type f -size +${1:-10}M`,  // find files larger than SIZE (default: 10 MB)
		},
	},
}

// DefaultShortcuts is the flattened view of DefaultShortcutGroups used by the
// rest of the codebase (Merge, WriteShellFile, etc.) — no callers need to change.
var DefaultShortcuts = flattenGroups(DefaultShortcutGroups)

func flattenGroups(groups map[string]ShortcutGroup) map[string]string {
	flat := make(map[string]string)
	for _, g := range groups {
		for name, cmd := range g.Shortcuts {
			flat[name] = cmd
		}
	}
	return flat
}
