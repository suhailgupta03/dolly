// gen-shortcuts generates docs/shortcuts.md from the built-in shortcut
// definitions in shortcuts.DefaultShortcutGroups. Run via:
//
//	make shortcuts
//	go generate ./shortcuts/...
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tmux-manager/shortcuts"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen-shortcuts: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll("docs", 0755); err != nil {
		return fmt.Errorf("could not create docs/: %w", err)
	}

	outPath := filepath.Join("docs", "shortcuts.md")
	var b strings.Builder

	b.WriteString("<!-- AUTO-GENERATED — do not edit manually. Run `make shortcuts` to regenerate. -->\n\n")
	b.WriteString("# Dolly Built-in Shortcuts\n\n")
	b.WriteString("Every dolly-managed pane has these shortcuts available. ")
	b.WriteString("They are sourced automatically at session creation.\n\n")
	b.WriteString("---\n\n")

	// Sort group names for deterministic output
	groups := make([]string, 0, len(shortcuts.DefaultShortcutGroups))
	for g := range shortcuts.DefaultShortcutGroups {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	for _, groupName := range groups {
		g := shortcuts.DefaultShortcutGroups[groupName]

		b.WriteString(fmt.Sprintf("## %s\n\n", groupName))
		b.WriteString("| Name | Description | Example |\n")
		b.WriteString("|------|-------------|---------|\n")

		// Sort shortcut names within each group for deterministic output
		names := make([]string, 0, len(g.Shortcuts))
		for name := range g.Shortcuts {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			def := g.Shortcuts[name]
			b.WriteString(fmt.Sprintf("| `%s` | %s | `%s` |\n",
				name, def.Description, def.Example))
		}
		b.WriteString("\n")
	}

	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("could not write %s: %w", outPath, err)
	}

	fmt.Printf("wrote %s (%d groups, %d shortcuts)\n",
		outPath, len(groups), countShortcuts())
	return nil
}

func countShortcuts() int {
	n := 0
	for _, g := range shortcuts.DefaultShortcutGroups {
		n += len(g.Shortcuts)
	}
	return n
}
