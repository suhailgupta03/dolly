package config

type Pane struct {
	Command          string   `yaml:"command"`
	Split            string   `yaml:"split"`
	WorkingDirectory string   `yaml:"working_directory,omitempty"`
	PreHooks         []string `yaml:"pre_hooks,omitempty"`
}

type Window struct {
	Name  string `yaml:"name"`
	Panes []Pane `yaml:"panes"`
}

type TmuxConfig struct {
	SessionName      string   `yaml:"session_name"`
	WorkingDirectory string   `yaml:"working_directory,omitempty"`
	Terminal         string   `yaml:"terminal,omitempty"`
	Windows          []Window `yaml:"windows"`
}