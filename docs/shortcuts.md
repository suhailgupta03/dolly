<!-- AUTO-GENERATED — do not edit manually. Run `make shortcuts` to regenerate. -->

# Dolly Built-in Shortcuts

Every dolly-managed pane has these shortcuts available. They are sourced automatically at session creation.

---

## find

| Name | Description | Example |
|------|-------------|---------|
| `fd` | Find directories by name glob pattern | `fd "testdata"` |
| `ff` | Find files by name glob pattern | `ff "*.go"` |
| `fnew` | Find files newer than a given reference file | `fnew go.mod` |
| `fsize` | Find files larger than SIZE MB (default: 10 MB) | `fsize 50` |

## grep

| Name | Description | Example |
|------|-------------|---------|
| `search` | Recursively search for a pattern in all files under cwd | `search "TODO"` |
| `searchi` | Case-insensitive recursive search | `searchi "fixme"` |
| `searchl` | List only filenames containing a pattern | `searchl "import"` |
| `searchw` | Match whole words only (no partial matches) | `searchw "main"` |
| `searchx` | Recursive search excluding common dependency and VCS directories | `searchx "config"` |

## tmux

| Name | Description | Example |
|------|-------------|---------|
| `kp` | Kill (close) the current pane | `kp` |
| `nw` | Open a new window in the current session; shortcuts auto-sourced | `nw` |
| `rw` | Rename the current window | `rw backend` |
| `sp` | Split current pane — new pane opens below; shortcuts auto-sourced | `sp` |
| `vsp` | Split current pane — new pane opens to the right; shortcuts auto-sourced | `vsp` |
| `zoom` | Toggle current pane full-screen (run again to unzoom) | `zoom` |

