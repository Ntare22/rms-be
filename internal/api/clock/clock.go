package clock

import "time"

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}

// RealClock returns the system clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
