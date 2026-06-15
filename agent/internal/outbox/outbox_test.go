package outbox

import (
	"testing"

	agentv1 "github.com/croncompose/croncompose/proto/agent/v1"
)

func TestEnqueueAndDrainInOrder(t *testing.T) {
	dir := t.TempDir()
	ob, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer ob.Close()

	for _, id := range []string{"run-1", "run-2", "run-3"} {
		if err := ob.Enqueue(&agentv1.AgentMessage{
			Body: &agentv1.AgentMessage_RunStarted{RunStarted: &agentv1.RunStarted{RunId: id}},
		}); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	if got := ob.Len(); got != 3 {
		t.Errorf("len after enqueue: got %d, want 3", got)
	}

	var seen []string
	if err := ob.Drain(func(m *agentv1.AgentMessage) error {
		seen = append(seen, m.GetRunStarted().GetRunId())
		return nil
	}); err != nil {
		t.Fatalf("drain: %v", err)
	}
	want := []string{"run-1", "run-2", "run-3"}
	if len(seen) != 3 {
		t.Fatalf("seen: %v", seen)
	}
	for i, s := range seen {
		if s != want[i] {
			t.Errorf("seen[%d]: got %q, want %q", i, s, want[i])
		}
	}
	if got := ob.Len(); got != 0 {
		t.Errorf("len after drain: got %d, want 0", got)
	}
}

func TestDrainStopsOnError(t *testing.T) {
	dir := t.TempDir()
	ob, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer ob.Close()

	for _, id := range []string{"a", "b", "c"} {
		_ = ob.Enqueue(&agentv1.AgentMessage{
			Body: &agentv1.AgentMessage_RunStarted{RunStarted: &agentv1.RunStarted{RunId: id}},
		})
	}

	calls := 0
	stop := errStop // package-internal sentinel; reuse to trigger early stop
	_ = stop
	err = ob.Drain(func(m *agentv1.AgentMessage) error {
		calls++
		return errFakeSend
	})
	if err != errFakeSend {
		t.Fatalf("drain err: got %v, want errFakeSend", err)
	}
	if calls != 1 {
		t.Errorf("calls: got %d, want 1 (drain should stop on first error)", calls)
	}
	if got := ob.Len(); got != 3 {
		t.Errorf("len after failed drain: got %d, want 3 (nothing deleted)", got)
	}
}

func TestPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	ob, err := Open(dir)
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	_ = ob.Enqueue(&agentv1.AgentMessage{
		Body: &agentv1.AgentMessage_RunStarted{RunStarted: &agentv1.RunStarted{RunId: "x"}},
	})
	_ = ob.Close()

	ob2, err := Open(dir)
	if err != nil {
		t.Fatalf("open 2: %v", err)
	}
	defer ob2.Close()
	if got := ob2.Len(); got != 1 {
		t.Errorf("len after reopen: got %d, want 1", got)
	}
	// Counter must continue from existing max so Enqueue order is preserved.
	_ = ob2.Enqueue(&agentv1.AgentMessage{
		Body: &agentv1.AgentMessage_RunStarted{RunStarted: &agentv1.RunStarted{RunId: "y"}},
	})
	var seen []string
	_ = ob2.Drain(func(m *agentv1.AgentMessage) error {
		seen = append(seen, m.GetRunStarted().GetRunId())
		return nil
	})
	if len(seen) != 2 || seen[0] != "x" || seen[1] != "y" {
		t.Errorf("ordering after reopen: %v", seen)
	}
}

// errFakeSend simulates a stream.Send failure for the drain test.
var errFakeSend = sentinel("fake send failure")

type sentinel string

func (s sentinel) Error() string { return string(s) }
