// Package clock abstracts time so use-cases stay deterministic in tests.
package clock

import "time"

// Clock is the time port. Injected; domain never calls time.Now directly.
type Clock interface {
	Now() time.Time
}

// System is UTC wall clock.
type System struct{}

func (System) Now() time.Time { return time.Now().UTC() }

// Fixed returns a constant instant (tests).
type Fixed struct{ T time.Time }

func (f Fixed) Now() time.Time { return f.T }
