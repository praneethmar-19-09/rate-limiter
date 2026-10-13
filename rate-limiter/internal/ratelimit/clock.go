package ratelimit

import "time"

// Clock provides the current time.
type Clock interface {
    Now() time.Time
}

// systemClock uses the system clock.
type systemClock struct{}

// Now returns the current system time.
func (systemClock) Now() time.Time {
    return time.Now()
}