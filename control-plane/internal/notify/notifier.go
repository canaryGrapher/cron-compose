package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Notifier knows how to load enabled targets and POST a RunFailedEvent to each.
// Designed to be called from a goroutine; never blocks the caller's path.
type Notifier struct {
	store  *Store
	log    *slog.Logger
	client *http.Client
}

// NewNotifier wires a Notifier.
func NewNotifier(store *Store, log *slog.Logger) *Notifier {
	return &Notifier{
		store:  store,
		log:    log,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// FireRunFailed matches agentgw.FailedRunHook. Each enabled target is POSTed
// concurrently; failures are logged.
func (n *Notifier) FireRunFailed(serverID, jobID, runID, status string, exitCode, durationMs int32, errMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	targets, err := n.store.EnabledList(ctx)
	if err != nil {
		n.log.Warn("notify: list targets failed", "err", err)
		return
	}
	body, err := json.Marshal(RunFailedEvent{
		RunID:      runID,
		JobID:      jobID,
		ServerID:   serverID,
		Status:     status,
		ExitCode:   exitCode,
		DurationMs: durationMs,
		Error:      errMsg,
	})
	if err != nil {
		n.log.Warn("notify: marshal failed", "err", err)
		return
	}
	for _, t := range targets {
		go n.post(t, body)
	}
}

func (n *Notifier) post(t Target, body []byte) {
	req, err := http.NewRequest(http.MethodPost, t.URL, bytes.NewReader(body))
	if err != nil {
		n.log.Warn("notify: new request", "target", t.Name, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "croncompose-notifier/1")

	resp, err := n.client.Do(req)
	if err != nil {
		n.log.Warn("notify: post", "target", t.Name, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		n.log.Warn("notify: non-2xx", "target", t.Name, "status", resp.StatusCode)
	}
}
