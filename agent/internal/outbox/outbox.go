// Package outbox is the agent's persistent queue for AgentMessages.
//
// Every message we want to send upstream is first written to a bbolt-backed queue.
// A drain goroutine pulls the queue in order, sends each message, and deletes the key
// on success. Disconnect or restart leaves messages on disk; next connect drains them
// in order. This is what makes the "jobs keep firing while the control plane is down"
// promise actually hold for status + log events.
package outbox

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/croncompose/croncompose/proto/agent/v1"
)

var bucket = []byte("outbox")

// Outbox persists pending AgentMessages.
type Outbox struct {
	db   *bbolt.DB
	mu   sync.Mutex // guards Enqueue's monotonic counter
	next uint64
}

// Open creates or opens the outbox database under dataDir/outbox.db.
func Open(dataDir string) (*Outbox, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	db, err := bbolt.Open(filepath.Join(dataDir, "outbox.db"), 0o600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	o := &Outbox{db: db}
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	}); err != nil {
		db.Close()
		return nil, err
	}
	// Seed monotonic counter from the highest existing key.
	if err := db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket(bucket).Cursor()
		if k, _ := c.Last(); k != nil {
			o.next = binary.BigEndian.Uint64(k) + 1
		}
		return nil
	}); err != nil {
		db.Close()
		return nil, err
	}
	return o, nil
}

// Close shuts down the underlying database.
func (o *Outbox) Close() error { return o.db.Close() }

// Enqueue serializes msg and appends it to the queue. Safe for concurrent callers.
func (o *Outbox) Enqueue(msg *agentv1.AgentMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	key := o.next
	o.next++
	return o.db.Update(func(tx *bbolt.Tx) error {
		k := make([]byte, 8)
		binary.BigEndian.PutUint64(k, key)
		return tx.Bucket(bucket).Put(k, data)
	})
}

// Drain iterates the queue in order. fn returns nil on success (the key is then
// deleted) or an error (drain stops, the key stays). Returns nil when the queue is
// empty.
func (o *Outbox) Drain(fn func(msg *agentv1.AgentMessage) error) error {
	for {
		var key []byte
		var msg agentv1.AgentMessage
		err := o.db.View(func(tx *bbolt.Tx) error {
			k, v := tx.Bucket(bucket).Cursor().First()
			if k == nil {
				return errStop
			}
			key = append([]byte(nil), k...)
			return proto.Unmarshal(v, &msg)
		})
		if errors.Is(err, errStop) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := fn(&msg); err != nil {
			return err
		}
		if err := o.db.Update(func(tx *bbolt.Tx) error {
			return tx.Bucket(bucket).Delete(key)
		}); err != nil {
			return err
		}
	}
}

// Len returns the number of pending messages. Useful for metrics.
func (o *Outbox) Len() int {
	n := 0
	_ = o.db.View(func(tx *bbolt.Tx) error {
		n = tx.Bucket(bucket).Stats().KeyN
		return nil
	})
	return n
}

var errStop = errors.New("outbox: empty")
