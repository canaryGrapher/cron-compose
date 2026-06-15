// Package runs serves the read side of run history and logs.
package runs

import "time"

// Run mirrors the runs table for the API.
type Run struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	JobVersionID string     `json:"job_version_id"`
	ServerID     string     `json:"server_id"`
	Trigger      string     `json:"trigger"`
	Status       string     `json:"status"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	ExitCode     *int       `json:"exit_code,omitempty"`
	DurationMs   *int       `json:"duration_ms,omitempty"`
	Error        string     `json:"error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// LogLine is one chunk of captured output.
type LogLine struct {
	Stream string    `json:"stream"`
	Seq    int       `json:"seq"`
	Chunk  string    `json:"chunk"`
	Ts     time.Time `json:"ts"`
}
