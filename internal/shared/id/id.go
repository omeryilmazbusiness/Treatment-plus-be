// Package id is the shared ULID v2 identifier. Stored as UUID in PostgreSQL.
package id

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

// ID is a 128-bit ULID. Sortable, unique, UUID-compatible.
type ID ulid.ULID

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.Reader, 0)
)

// New generates a ULID for now (UTC).
func New() ID {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ID(ulid.MustNew(ulid.Timestamp(time.Now().UTC()), entropy))
}

// Parse decodes a canonical 26-char ULID.
func Parse(s string) (ID, error) {
	parsed, err := ulid.Parse(s)
	if err != nil {
		return ID{}, fmt.Errorf("parse id: %w", err)
	}
	return ID(parsed), nil
}

func (id ID) String() string { return ulid.ULID(id).String() }

func (id ID) IsZero() bool { return ulid.ULID(id).Compare(ulid.ULID{}) == 0 }

// Bytes returns the 16 raw bytes (PostgreSQL uuid).
func (id ID) Bytes() []byte {
	b := ulid.ULID(id)
	out := make([]byte, 16)
	copy(out, b[:])
	return out
}

// FromBytes reconstructs an ID from 16 bytes.
func FromBytes(b []byte) (ID, error) {
	if len(b) != 16 {
		return ID{}, fmt.Errorf("id: need 16 bytes, got %d", len(b))
	}
	var raw ulid.ULID
	copy(raw[:], b)
	return ID(raw), nil
}
