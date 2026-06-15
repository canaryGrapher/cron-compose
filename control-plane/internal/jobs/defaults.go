package jobs

import (
	"encoding/json"
	"errors"
)

// applyDefaults fills zero-valued fields with the schema defaults so the handler
// stays simple. Mirrors migrations/0001_init.sql + 0004 defaults.
func applyDefaults(in *CreateInput) {
	if in.TargetKind == "" {
		in.TargetKind = "server"
	}
	if in.Interpreter == "" {
		in.Interpreter = "bash"
	}
	if in.Timezone == "" {
		in.Timezone = "UTC"
	}
	if in.Enabled == nil {
		t := true
		in.Enabled = &t
	}
	if in.TimeoutSeconds == 0 {
		in.TimeoutSeconds = 3600
	}
	if in.ConcurrencyPolicy == "" {
		in.ConcurrencyPolicy = "skip"
	}
	if in.CatchupPolicy == "" {
		in.CatchupPolicy = "once"
	}
	if in.Env == nil {
		in.Env = map[string]string{}
	}
	if in.TargetLabels == nil {
		in.TargetLabels = map[string]string{}
	}
}

// validateTarget enforces the target_kind invariants.
func validateTarget(kind, serverID string, labels map[string]string) error {
	switch kind {
	case "server":
		if serverID == "" {
			return errors.New("target_kind=server requires server_id")
		}
	case "labels":
		if len(labels) == 0 {
			return errors.New("target_kind=labels requires at least one label in target_labels")
		}
	default:
		return errors.New("target_kind must be 'server' or 'labels'")
	}
	return nil
}

// marshalLabels returns a JSON object (never null) for jsonb columns.
func marshalLabels(in map[string]string) []byte {
	if in == nil {
		in = map[string]string{}
	}
	b, _ := json.Marshal(in)
	return b
}
