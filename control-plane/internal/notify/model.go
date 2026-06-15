// Package notify fires webhook notifications when a run finishes with a non-success
// status. Designed to be best-effort: failures are logged, never blocking.
package notify

import "time"

// Target is a webhook destination.
type Target struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	URL       string    `json:"url"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateInput is the body of POST /notification-targets.
type CreateInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// RunFailedEvent is the JSON body POSTed to each enabled target.
type RunFailedEvent struct {
	RunID      string `json:"run_id"`
	JobID      string `json:"job_id"`
	ServerID   string `json:"server_id"`
	Status     string `json:"status"`
	ExitCode   int32  `json:"exit_code"`
	DurationMs int32  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}
