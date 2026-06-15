package agentgw

import (
	"testing"
	"time"

	agentv1 "github.com/croncompose/croncompose/proto/agent/v1"
)

func TestBrokerFansOut(t *testing.T) {
	b := NewLogBroker()
	a := b.Subscribe("run-1")
	c := b.Subscribe("run-1")
	defer b.Unsubscribe("run-1", a)
	defer b.Unsubscribe("run-1", c)

	b.Publish("run-1", &agentv1.LogChunk{RunId: "run-1", Stream: "stdout", Seq: 0, Data: []byte("hi")})

	for _, ch := range []chan LogEvent{a, c} {
		select {
		case ev := <-ch:
			if ev.Chunk == nil || string(ev.Chunk.GetData()) != "hi" {
				t.Errorf("unexpected event: %+v", ev)
			}
		case <-time.After(time.Second):
			t.Fatal("subscriber did not receive event")
		}
	}
}

func TestBrokerIgnoresUnrelatedRuns(t *testing.T) {
	b := NewLogBroker()
	a := b.Subscribe("run-1")
	defer b.Unsubscribe("run-1", a)
	b.Publish("run-2", &agentv1.LogChunk{RunId: "run-2"})
	select {
	case ev := <-a:
		t.Fatalf("should not have received cross-run event: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestBrokerUnsubscribeClosesChannel(t *testing.T) {
	b := NewLogBroker()
	a := b.Subscribe("run-1")
	b.Unsubscribe("run-1", a)
	if _, ok := <-a; ok {
		t.Fatal("expected closed channel after Unsubscribe")
	}
}
