package registry

import "time"

const DefaultCleanupDays = 7

// SessionType identifies how a dolly session was created
type SessionType string

const (
	TypeThrowaway SessionType = "throwaway"
	TypeYAML      SessionType = "yaml"
	TypeExec      SessionType = "exec"
	TypeAttached  SessionType = "attached"
)

// Entry represents one registered dolly session
type Entry struct {
	Name       string      `json:"name"`
	Type       SessionType `json:"type"`
	CreatedAt  time.Time   `json:"created_at"`
	LastActive time.Time   `json:"last_active"`
	WorkingDir string      `json:"working_dir"`
	ConfigFile string      `json:"config_file,omitempty"` // absolute path to .yml (yaml mode only)
	Windows    int         `json:"windows"`
	Terminal   string      `json:"terminal"`
}

// Registry is the top-level JSON document stored at ~/.dolly/registry.json
type Registry struct {
	Sessions []Entry `json:"sessions"`
}

// SessionStatus is an Entry augmented with a live alive check
type SessionStatus struct {
	Entry
	Alive bool
}
