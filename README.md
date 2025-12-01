# Dolly - Tmux Session Manager

[![Release](https://github.com/suhailgupta03/dolly/workflows/Release/badge.svg)](https://github.com/suhailgupta03/dolly/actions)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A YAML-based tmux session manager that creates development environments from configuration files.

## Download

**Pre-built binaries available for all platforms:**
- 📦 **[Download from Releases](https://github.com/suhailgupta03/dolly/releases)** - Windows, macOS, Linux (recommended)
- 🔨 **Build from source** - See installation section below

## Features

- YAML configuration for tmux sessions
- Multi-shell support (bash, zsh, fish)
- Pre-hooks for setup commands
- Working directory management per session/window/pane
- Multiple windows and panes with custom layouts
- **Colored pane labels** for easy visual identification
- **Automatic base-index 1** for window numbering starting from 1

## Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Go to **[Releases](https://github.com/suhailgupta03/dolly/releases)**
2. Download the appropriate binary for your system:
   - **Windows**: `dolly_Windows_x86_64.zip`
   - **macOS Intel**: `dolly_Darwin_x86_64.tar.gz` 
   - **macOS Apple Silicon**: `dolly_Darwin_arm64.tar.gz`
   - **Linux x86_64**: `dolly_Linux_x86_64.tar.gz`
   - **Linux ARM64**: `dolly_Linux_arm64.tar.gz`
3. Extract and place in your PATH

### Option 2: Build from Source (POSIX systems)

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
auto_color: true                     # Optional: Enable automatic window coloring (default: true)
show_pane_labels: true               # Optional: Show pane labels (default: true)
default_label_color: "blue"          # Optional: Default color for pane labels (default: blue)

windows:
  - name: "frontend"                 # Window name
    color: "green"                   # Optional: Window tab color
    panes:
      - id: "dev-server"             # Optional: Pane identifier (used for labels)
        command: "npm run dev"       # Command to execute
        split: "none"                # Split type: none/horizontal/vertical
        working_directory: "./web"   # Optional: Pane-specific directory
        show_label: true             # Optional: Override global label setting
        label_color: "brightblue"    # Optional: Custom color for this pane label
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
auto_color: true          # Enable automatic window coloring (default)
show_pane_labels: true    # Enable pane labels (default)
default_label_color: "blue"  # Default label color
windows:
  - name: "development"
    # No color specified - gets "green" (first window in palette)
    panes:
      - id: "server"      # This ID becomes the pane label
        command: "npm run dev"
  - name: "testing"
    color: "brightred"    # Explicit color overrides auto-assignment
    panes:
      - id: "tests"       # Label: "tests" with blue background
        command: "npm test"
  - name: "monitoring"
    # No color specified - gets "red" (third window in palette)
    panes:
      - id: "logs"        # Label: "logs" with blue background
        command: "tail -f app.log"
```

### Pane Labels

Dolly supports colored pane labels to help identify panes visually. Labels appear at the top border of each pane with a colored background.

**Features:**
- Labels are extracted from pane IDs for easy identification
- Configurable background colors for better visual organization
- Can be enabled/disabled globally or per-pane
- Uses high-contrast white text on colored backgrounds

**Global Label Configuration:**
```yaml
show_pane_labels: true               # Enable pane labels (default: true)
default_label_color: "green"         # Default background color (default: blue)
```

**Per-Pane Label Configuration:**
```yaml
windows:
  - name: "development"
    panes:
      - id: "dev-server"             # This becomes the label text
        command: "npm start"
        show_label: true             # Override global setting (optional)
        label_color: "brightblue"    # Custom color for this pane (optional)
      - id: "tests"
        command: "npm test"
        label_color: "yellow"        # Different color for test pane
```

**Supported Colors:**
- **Basic**: `red`, `green`, `blue`, `yellow`, `cyan`, `magenta`, `white`, `black`
- **Bright**: `brightred`, `brightgreen`, `brightblue`, `brightyellow`, `brightcyan`, `brightmagenta`, `brightwhite`

**Visual Result:**
Each pane displays its ID in a colored label at the top:
```
[blue bg] dev-server [end]  [yellow bg] tests [end]
┌─────────────────┐        ┌─────────────────┐
│                 │        │                 │
│ Server output   │        │ Test results    │
│                 │        │                 │
└─────────────────┘        └─────────────────┘
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
| `show_pane_labels` | Show colored pane labels | true |
| `default_label_color` | Default background color for pane labels | blue |
| `windows[].name` | Window name | Required |
| `windows[].color` | Window tab color | auto-assigned if `auto_color: true` |
| `windows[].panes[].id` | Unique identifier for the pane (used for labels) | auto-generated |
| `windows[].panes[].command` | Command to execute | "" |
| `windows[].panes[].split` | Split type: `none`, `horizontal`, `vertical` | none |
| `windows[].panes[].split_from` | ID of pane to split from | previous pane |
| `windows[].panes[].working_directory` | Pane working directory | Inherits from session |
| `windows[].panes[].pre_hooks` | Commands to run before main command | [] |
| `windows[].panes[].show_label` | Override global pane label setting | Inherits from global |
| `windows[].panes[].label_color` | Background color for this pane's label | Inherits from default |

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

## Recommended Tmux Configuration

Dolly includes a recommended tmux configuration file (`.tmux.conf`) in the project root. You can use it by copying to your home directory:

```bash
cp .tmux.conf ~/.tmux.conf
```

Or source it in your existing tmux configuration:

```bash
echo "source-file /path/to/dolly/.tmux.conf" >> ~/.tmux.conf
```

See [`.tmux.conf`](.tmux.conf) for the full configuration including:
- Mouse support
- Window numbering starting from 1
- Custom theme with status bar
- Keybindings for opening new panes/windows in the current directory

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

### Cross-Platform Builds

Dolly uses [GoReleaser](https://goreleaser.com/) for automated cross-platform builds and releases:

**Automated Process:**
- **Build**: Triggers on every push/PR to validate builds
- **Release**: Creates GitHub releases with binaries on git tag push
- **Testing**: Separate workflow runs tests and integration tests
- **Artifacts**: Generates SHA256 checksums and release notes

**Manual Build with GoReleaser:**
```bash
# Install GoReleaser (macOS)
brew install goreleaser

# Build snapshot (all platforms)
goreleaser build --snapshot --clean

# Release (requires git tag)
goreleaser release --clean
```

**Creating Releases:**
- **Git tag**: `git tag v1.0.0 && git push origin v1.0.0`
- **Manual**: GitHub Actions → Release workflow → Enter version → Run

**Manual Cross-Platform Build:**
```bash
# Use GoReleaser config or manual build
make build  # Uses existing Makefile
```

## License

MIT License - see the [LICENSE](LICENSE) file for details.
