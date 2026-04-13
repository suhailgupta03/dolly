package crashlog

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"
)

const errorTruncLen = 55

// truncate trims s to at most n runes, appending "..." if truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-3]) + "..."
}

// FormatReport returns a human-readable table of crash entries.
// total is the total number of entries in the log (before any limit was applied).
func FormatReport(entries []CrashEntry, total int) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tSTATUS\tTIMESTAMP\tKIND\tSUBCOMMAND\tERROR")
	for i, e := range entries {
		status := ""
		if e.Submitted {
			status = "[reported]"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			i+1,
			status,
			e.Timestamp.Format("2006-01-02 15:04:05"),
			string(e.Kind),
			e.Subcommand,
			truncate(e.Error, errorTruncLen),
		)
	}
	w.Flush()

	// Footer
	b.WriteString("\n")
	if total > len(entries) {
		fmt.Fprintf(&b, "Showing %d of %d entries — use -last N to see more (-last 0 for all).\n", len(entries), total)
	}

	allSubmitted := true
	for _, e := range entries {
		if !e.Submitted {
			allSubmitted = false
			break
		}
	}
	if allSubmitted {
		b.WriteString("All entries have been reported.\n")
	} else {
		b.WriteString("Run `dolly report submit` to open a GitHub issue.\n")
	}

	return b.String()
}

// FormatIssueBody returns a Markdown string suitable for a GitHub issue body.
func FormatIssueBody(entries []CrashEntry, version string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Crash Report\n\n")
	if len(entries) > 0 {
		fmt.Fprintf(&b, "**Dolly version:** %s  \n", version)
		fmt.Fprintf(&b, "**OS:** %s/%s  \n\n", entries[0].GOOS, entries[0].GOArch)
	}
	fmt.Fprintf(&b, "### Entries\n\n")
	for i, e := range entries {
		fmt.Fprintf(&b, "**#%d** `%s` at %s  \n", i+1, e.Subcommand, e.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&b, "Kind: `%s`  \n", string(e.Kind))
		fmt.Fprintf(&b, "Error: `%s`  \n", e.Error)
		if e.StackTrace != "" {
			fmt.Fprintf(&b, "<details><summary>Stack trace</summary>\n\n```\n%s\n```\n</details>\n", e.StackTrace)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "### Steps to reproduce\n\n_Please describe what you were doing when this occurred._\n")
	return b.String()
}

// GitHubIssueURL returns a pre-filled GitHub Issues URL.
// The title is taken from the most recent (last) entry.
func GitHubIssueURL(entries []CrashEntry, version string) string {
	const base = "https://github.com/suhailgupta03/dolly/issues/new"
	title := fmt.Sprintf("[crash] %s", version)
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		title = fmt.Sprintf("[%s] %s: %s", last.Subcommand, version, truncate(last.Error, 60))
	}
	body := FormatIssueBody(entries, version)
	return base + "?title=" + url.QueryEscape(title) + "&body=" + url.QueryEscape(body)
}

// OpenBrowser attempts to open rawURL in the system browser.
// Falls back to printing the URL when the browser tool is unavailable.
// This function never returns an error — it is always best-effort.
func OpenBrowser(rawURL string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", rawURL)
	default:
		fmt.Printf("Open in your browser:\n%s\n", rawURL)
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser. Open manually:\n%s\n", rawURL)
	}
}
