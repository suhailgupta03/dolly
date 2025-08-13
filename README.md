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
- **Real-time log streaming** from selected windows and panes

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
auto_color: true                     # Optional: Enable automatic pane coloring (default: true)

# Optional: Log streaming configuration
log_stream:
  enabled: true                      # Enable log streaming
  windows: ["*"]                     # Windows to stream from ("*" for all)
  panes: ["*"]                       # Panes to stream from ("*" for all)
  grep: ["ERROR", "WARNING"]         # Filter for specific keywords (optional)

windows:
  - name: "frontend"                 # Window name
    color: "green"                   # Optional: Window tab color
    panes:
      - command: "npm run dev"       # Command to execute
        split: "none"                # Split type: none/horizontal/vertical
        working_directory: "./web"   # Optional: Pane-specific directory
        pre_hooks:                   # Optional: Commands run before main command
          - "nvm use 18"
          - "export NODE_ENV=development"
```

### Window Tab Color Coding

Dolly supports automatic and manual color coding for window tabs to improve visual organization:

**Automatic Coloring (default):**
- When `auto_color: true` (default), windows without explicit colors are automatically assigned colors
- To disable auto-coloring: `auto_color: false`

**Manual Coloring:**
- Set explicit colors using the `color` field in window configuration
- Explicit colors override automatic assignment
- Supports multiple color formats:
  - **Basic colors**: `red`, `green`, `blue`, `yellow`, `cyan`, `magenta`, `white`, `black`
  - **Bright colors**: `brightred`, `brightgreen`, `brightblue`, `brightyellow`, `brightcyan`, `brightmagenta`, `brightwhite`


**Example Configuration:**
```yaml
auto_color: true  # Enable automatic coloring (default)
windows:
  - name: "development"
    # No color specified - gets "green" (first window in palette)
    panes:
      - id: "server"
        command: "npm run dev"
  - name: "testing"
    color: "brightred"  # Explicit color overrides auto-assignment
    panes:
      - id: "tests"
        command: "npm test"
  - name: "monitoring"
    # No color specified - gets "red" (third window in palette)
    panes:
      - id: "logs"
        command: "tail -f app.log"
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
| `auto_color` | Enable automatic window coloring | true |
| `windows[].name` | Window name | Required |
| `windows[].color` | Window tab color | auto-assigned if `auto_color: true` |
| `windows[].panes[].id` | Unique identifier for the pane | auto-generated |
| `windows[].panes[].command` | Command to execute | "" |
| `windows[].panes[].split` | Split type: `none`, `horizontal`, `vertical` | none |
| `windows[].panes[].split_from` | ID of pane to split from | previous pane |
| `windows[].panes[].working_directory` | Pane working directory | Inherits from session |
| `windows[].panes[].pre_hooks` | Commands to run before main command | [] |
| `log_stream.enabled` | Enable real-time log streaming | false |
| `log_stream.windows` | Windows to stream from (names or "*" for all) | [] |
| `log_stream.panes` | Panes to stream from (IDs or "*" for all) | [] |
| `log_stream.grep` | Keywords to filter log messages (case-insensitive) | [] |

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

## Log Streaming

Dolly supports real-time log streaming that captures output from selected windows and panes into a dedicated streaming window. When enabled, the streaming window becomes the first window (window 0) and continuously displays timestamped output from the specified sources.

### Basic Log Streaming

```yaml
session_name: "my-project"
log_stream:
  enabled: true
  windows: ["*"]    # Stream from all windows
  panes: ["*"]      # Stream from all panes
```

### Selective Streaming

**Stream from specific windows:**
```yaml
log_stream:
  enabled: true
  windows: ["frontend", "backend"]  # Only these windows
  panes: ["*"]                      # All panes within selected windows
```

**Stream from specific panes:**
```yaml
log_stream:
  enabled: true
  windows: ["*"]                    # All windows
  panes: ["server", "tests"]        # Only these pane IDs
```

**Combine window and pane filtering:**
```yaml
log_stream:
  enabled: true
  windows: ["development"]          # Only from development window
  panes: ["main-app", "logger"]     # Only these specific panes
```

**Filter by keywords (grep):**
```yaml
log_stream:
  enabled: true
  windows: ["*"]                    # All windows
  panes: ["*"]                      # All panes
  grep: ["ERROR", "WARNING", "FAIL"] # Only show lines containing these keywords
```

### How It Works

1. **Streaming Window**: When log streaming is enabled, Dolly creates a "logs" window as the first window (window 0)
2. **Real-time Capture**: The streaming window continuously monitors the specified panes using `tmux capture-pane`
3. **Timestamped Output**: Each log entry is prefixed with the pane ID and timestamp
4. **Incremental Updates**: Only new output since the last check is displayed
5. **Keyword Filtering**: Optional grep functionality to filter log messages by keywords (case-insensitive)
6. **Non-intrusive**: Original panes continue to work normally while being monitored

### Example Output

```
=== Log Streaming Started for Session: my-project ===
Streaming from 3 panes
==================================

[%123] 14:30:15:
[SERVER] Starting development server on port 3000
[SERVER] Webpack compilation completed

[%124] 14:30:16:
[TESTS] Running test suite...
[TESTS] ✓ All tests passed

[%125] 14:30:17:
[DB] Database connection established
```

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
