package config

type Pane struct {
	ID               string   `yaml:"id,omitempty"` // Unique identifier for this pane
	Command          string   `yaml:"command"`
	Split            string   `yaml:"split"`
	SplitFrom        string   `yaml:"split_from,omitempty"` // ID of pane to split from
	WorkingDirectory string   `yaml:"working_directory,omitempty"`
	PreHooks         []string `yaml:"pre_hooks,omitempty"`
}

type Window struct {
	Name  string `yaml:"name"`
	Color string `yaml:"color,omitempty"` // Background color for the window tab in status bar
	Panes []Pane `yaml:"panes"`
}

type LogStreamConfig struct {
	Enabled bool     `yaml:"enabled"`
	Windows []string `yaml:"windows,omitempty"` // Window names to stream from, "*" for all
	Panes   []string `yaml:"panes,omitempty"`   // Pane IDs to stream from, "*" for all
	Grep    []string `yaml:"grep,omitempty"`    // Keywords to filter log messages, empty for all
}

type TmuxConfig struct {
	SessionName      string          `yaml:"session_name"`
	WorkingDirectory string          `yaml:"working_directory,omitempty"`
	Terminal         string          `yaml:"terminal,omitempty"`
	AutoColor        *bool           `yaml:"auto_color,omitempty"` // Enable automatic color assignment (default: true)
	LogStream        LogStreamConfig `yaml:"log_stream,omitempty"`
	Windows          []Window        `yaml:"windows"`
}
