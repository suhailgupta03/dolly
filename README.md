# Dolly — Tmux Session Manager

[![Release](https://github.com/suhailgupta03/dolly/workflows/Release/badge.svg)](https://github.com/suhailgupta03/dolly/actions)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

YAML-based tmux session manager. Define your dev environment once, spin it up anywhere.

## Install

**Pre-built binary** (recommended): [Releases](https://github.com/suhailgupta03/dolly/releases)

**From source:**
```bash
git clone https://github.com/suhailgupta03/dolly.git
cd dolly && make install
```

---

## Usage

### YAML mode — repeatable environments

```bash
dolly my-project.yml       # create session
dolly -t my-project.yml    # terminate session
```

Minimal config:
```yaml
session_name: "my-session"
terminal: "zsh"
working_directory: "/path/to/project"
windows:
  - name: "dev"
    panes:
      - command: "npm run dev"
        split: "none"
      - command: "npm test"
        split: "vertical"
```

### Throwaway mode — disposable sessions

No config needed. Opens empty shells in the current directory.

```bash
dolly throwaway                          # 2 windows, 2 panes each, auto-named
dolly throwaway -windows 3 -panes 2     # custom layout
dolly throwaway -name my-debug          # named session
dolly throwaway -dir /path/to/project   # custom directory
```

Output:
```
Throwaway session 'tw-0401-143022' created (2 windows, 2 panes each)
Attach:  tmux attach -t tw-0401-143022
Kill:    dolly throwaway -kill tw-0401-143022
```

Manage throwaway sessions:
```bash
dolly throwaway -list                    # list sessions with status
dolly throwaway -kill SESSION            # kill + remove from registry
dolly throwaway -cleanup                 # remove dead sessions (default: 7 days)
dolly throwaway -cleanup -days 14        # custom threshold
```

### Exec mode — quick one-off sessions

```bash
dolly -e "npm run dev, npm test" -n my-session
```

Runs each comma-separated command in its own pane. Prompts to save as YAML.

### Session registry

All dolly sessions are tracked in `~/.dolly/registry.json`. View them:

```bash
dolly sessions                           # all sessions (yaml, exec, throwaway)
dolly sessions -type yaml                # filter by type
```

Registry is updated on every create, terminate, and cleanup. Sessions show as `alive` or `dead` based on live tmux status.

---

## YAML Configuration Reference

```yaml
session_name: "my-session"           # required
working_directory: "/path/to/project" # default for all panes
terminal: "zsh"                      # bash | zsh | fish (default: bash)
auto_color: true                     # auto-assign window tab colors
show_pane_labels: true               # show pane ID as label in border
default_label_color: "blue"          # label background color

windows:
  - name: "frontend"
    color: "green"                   # window tab color (overrides auto)
    panes:
      - id: "dev-server"             # becomes the pane label
        command: "npm run dev"
        split: "none"                # first pane is always none
        working_directory: "./web"   # overrides session default
        label_color: "brightblue"    # overrides default_label_color
        pre_hooks:
          - "nvm use 18"
      - id: "tests"
        command: "npm test"
        split: "vertical"            # side-by-side
        split_from: "dev-server"     # split from specific pane (optional)
```

**Split types:** `none` (first pane), `vertical` (side by side), `horizontal` (stacked)

**Colors:** `red`, `green`, `blue`, `yellow`, `cyan`, `magenta`, `white`, `black` — prefix with `bright` for bright variants

**Advanced layouts** use `split_from` to split from any named pane, not just the previous one:
```yaml
panes:
  - id: "main"   split: "none"
  - id: "right"  split: "vertical"    split_from: "main"
  - id: "bl"     split: "horizontal"  split_from: "main"
  - id: "br"     split: "horizontal"  split_from: "right"
```
Result: `main | right` on top, `bl | br` on bottom.

---

## Tmux config

Copy the included `.tmux.conf` for mouse support, 1-based window numbering, and sensible keybindings:
```bash
cp .tmux.conf ~/.tmux.conf
```

---

## Build

```bash
make build     # build binary
make install   # install to /usr/local/bin
make test      # run tests
```

## License

MIT — see [LICENSE](LICENSE)
