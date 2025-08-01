# Dolly - Tmux Session Manager

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A YAML-based tmux session manager that creates development environments from configuration files.

## Features

- YAML configuration for tmux sessions
- Multi-shell support (bash, zsh, fish)
- Pre-hooks for setup commands
- Working directory management per session/window/pane
- Multiple windows and panes with custom layouts

## Installation (Tested on Mac)

```bash
git clone https://github.com/suhailgupta03/dolly.git
cd dolly
make install
```

Or build manually:
```bash
make build
sudo cp dolly /usr/local/bin/
```

## Usage

1. Copy the sample configuration:
   ```bash
   cp sample-config.yml my-project.yml
   ```

2. Edit your configuration:
   ```yaml
   session_name: "my-dev-setup"
   terminal: "zsh"
   working_directory: "/path/to/your/project"
   windows:
     - name: "development"
       panes:
         - command: "npm run dev"
           split: "none"
           pre_hooks:
             - "nvm use"
   ```

3. Launch your session:
   ```bash
   dolly my-project.yml
   ```

4. Terminate your session:
   ```bash
   dolly -terminate my-project.yml
   # or use the shortcut:
   dolly -t my-project.yml
   ```

## Configuration

```yaml
session_name: "my-session"           # Required: Session name
working_directory: "/path/to/project" # Optional: Default directory
terminal: "zsh"                      # Optional: Shell (bash/zsh/fish)

windows:
  - name: "frontend"                 # Window name
    panes:
      - command: "npm run dev"       # Command to execute
        split: "none"                # Split type: none/horizontal/vertical
        working_directory: "./web"   # Optional: Pane-specific directory
        pre_hooks:                   # Optional: Commands run before main command
          - "nvm use 18"
          - "export NODE_ENV=development"
```

### Split Types

Dolly supports different ways to split panes:

**Horizontal Split** (`horizontal` or `h`):
```
┌─────────────────┐
│     Pane 1      │
├─────────────────┤
│     Pane 2      │
├─────────────────┤
│     Pane 3      │
└─────────────────┘
```
Panes are stacked **top to bottom**

**Vertical Split** (`vertical` or `v`):
```
┌─────┬─────┬─────┐
│     │     │     │
│Pane1│Pane2│Pane3│
│     │     │     │
└─────┴─────┴─────┘
```
Panes are arranged **side by side**

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `session_name` | Name of the tmux session | Required |
| `working_directory` | Default working directory | Current directory |
| `terminal` | Shell to use (bash/zsh/fish) | bash |
| `windows[].name` | Window name | Required |
| `windows[].panes[].id` | Unique identifier for the pane | auto-generated |
| `windows[].panes[].command` | Command to execute | "" |
| `windows[].panes[].split` | Split type: `none`, `horizontal`, `vertical` | none |
| `windows[].panes[].split_from` | ID of pane to split from | previous pane |
| `windows[].panes[].working_directory` | Pane working directory | Inherits from session |
| `windows[].panes[].pre_hooks` | Commands to run before main command | [] |

**Split Behavior:**
- **First pane** in each window must use `split: "none"`
- **Subsequent panes** can specify exactly which pane to split from using `split_from`
- **Pane IDs** can be explicitly set or auto-generated (`pane1`, `pane2`, etc.)
- **Split directions**:
  - `horizontal`: Creates top/bottom arrangement (stacked)
  - `vertical`: Creates side-by-side arrangement

**Advanced Layout Control:**
```yaml
windows:
  - name: "complex-layout"
    panes:
      - id: "main"
        command: "echo 'Main pane'"
        split: "none"
      - id: "right"
        command: "echo 'Right side'"
        split: "vertical"
        split_from: "main"      # Split vertically from main
      - id: "bottom-left"
        command: "echo 'Bottom left'"
        split: "horizontal"
        split_from: "main"      # Split horizontally from main
      - id: "bottom-right"
        command: "echo 'Bottom right'"
        split: "horizontal"
        split_from: "right"     # Split horizontally from right
```

**Result layout:**
```
┌─────────┬─────────┐
│  main   │  right  │
├─────────┼─────────┤
│bottom-  │bottom-  │
│left     │right    │
└─────────┴─────────┘
```

**Backward Compatibility:**
- Configurations without `split_from` work as before (split from previous pane)
- Configurations without `id` get auto-generated IDs

## Command Line Options

```bash
dolly [options] <config.yml>

Options:
  -terminate, -t    Terminate the tmux session
  -help, -h         Show help information

Examples:
  dolly my-project.yml     # Create session
  dolly -t my-project.yml  # Terminate session
  dolly -h                 # Show help
```

## Make Commands

```bash
make help          # Show all available commands
make build         # Build the dolly binary
make install       # Install to system PATH
make test          # Run test suite
make run-sample    # Run with sample configuration
make clean         # Clean build artifacts
```

## Testing

```bash
make test           # Run all tests
./test_runner       # Run tests manually
```

## Development

### Prerequisites
- Go 1.21 or higher
- tmux installed on your system

### Setup
```bash
git clone https://github.com/suhailgupta03/dolly.git
cd dolly
make build
make test
```

## License

MIT License - see the [LICENSE](LICENSE) file for details.
