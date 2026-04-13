package crashlog

import "time"

// Kind classifies the severity of a crash entry.
type Kind string

const (
	KindPanic         Kind = "panic"
	KindInternalError Kind = "internal_error"
)

// CrashEntry is one structured record in ~/.dolly/logs/crashes.jsonl.
// Submitted is derived at read time from submitted.json and is NEVER persisted
// to the JSONL file — the json:"-" tag ensures it is omitted from serialization.
type CrashEntry struct {
	ID           string    `json:"id"`                    // fmt.Sprintf("%x-%x", UnixNano, PID)
	Timestamp    time.Time `json:"timestamp"`
	Kind         Kind      `json:"kind"`
	Subcommand   string    `json:"subcommand"`
	Error        string    `json:"error"`
	DollyVersion string    `json:"version"`
	GOOS         string    `json:"os"`
	GOArch       string    `json:"arch"`
	StackTrace   string    `json:"stack_trace,omitempty"`
	Submitted    bool      `json:"-"` // NEVER serialized; populated at read time from sidecar
}
