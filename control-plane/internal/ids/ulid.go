// Package ids generates ULIDs for use as primary keys across the system.
// ULIDs are time-sortable and safe to generate on the agent side.
package ids

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// New returns a fresh ULID string.
func New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
