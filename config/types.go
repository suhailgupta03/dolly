package config

type Pane struct {
	ID               string   `yaml:"id,omitempty"` // Unique identifier for this pane
	Command          string   `yaml:"command"`
	Split            string   `yaml:"split"`
	SplitFrom        string   `yaml:"split_from,omitempty"` // ID of pane to split from
	WorkingDirectory string   `yaml:"working_directory,omitempty"`
	PreHooks         []string `yaml:"pre_hooks,omitempty"`
	ShowLabel        *bool    `yaml:"show_label,omitempty"` // Show pane label (overrides global setting)
	LabelColor       string   `yaml:"label_color,omitempty"` // Color for pane label background
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
	SessionName       string          `yaml:"session_name"`
	WorkingDirectory  string          `yaml:"working_directory,omitempty"`
	Terminal          string          `yaml:"terminal,omitempty"`
	AutoColor         *bool           `yaml:"auto_color,omitempty"` // Enable automatic color assignment (default: true)
	ShowPaneLabels    *bool           `yaml:"show_pane_labels,omitempty"` // Show labels on panes (default: true)
	DefaultLabelColor string          `yaml:"default_label_color,omitempty"` // Default color for pane labels (default: blue)
	LogStream         LogStreamConfig `yaml:"log_stream,omitempty"`
	Windows           []Window        `yaml:"windows"`
}
