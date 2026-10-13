package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrEmptyKey is returned when the rate-limit key is empty.
var ErrEmptyKey = errors.New("rate-limit key must not be empty")

// windowState stores rate-limiting information for one IP address.
type windowState struct {
	// windowStart is when the current window started.
	windowStart time.Time

	// window is the duration of the current window.
	window time.Duration

	// count is the number of requests evaluated in this window.
	count int64

	// blockedUntil is when the IP's block expires.
	// A zero value means the IP is not blocked.
	blockedUntil time.Time
}

// FixedWindowLimiter implements a fixed-window rate limiter.
//
// Each IP has its own window, starting when its first request arrives.
// The window resets after policy.Window.
//
// The implementation is safe for concurrent use.
type FixedWindowLimiter struct {
	mu sync.Mutex

	// states maps an actor key, such as a client IP, to its state.
	states map[string]windowState

	// clock provides the current time.
	clock Clock
}

// NewFixedWindowLimiter creates a new fixed-window rate limiter.
//
// If clock is nil, the system clock is used.
func NewFixedWindowLimiter(clock Clock) *FixedWindowLimiter {
	if clock == nil {
		clock = systemClock{}
	}

	return &FixedWindowLimiter{
		states: make(map[string]windowState),
		clock:  clock,
	}
}

// Allow determines whether a request is allowed.
//
// Behavior:
//   - Empty keys and invalid policies return errors.
//   - Actively blocked IPs are denied.
//   - Requests within the limit are allowed.
//   - The first request exceeding the limit triggers a block.
//   - An expired block is cleared before processing a new request.
//   - An expired window starts a fresh window when another request arrives.
//
// A rate-limit denial is returned as a Decision, not an error.
func (l *FixedWindowLimiter) Allow(
	ctx context.Context,
	key string,
	policy Policy,
) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}

	if key == "" {
		return Decision{}, ErrEmptyKey
	}

	if err := policy.Validate(); err != nil {
		return Decision{}, err
	}

	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check again in case the context was canceled while waiting
	// for the mutex.
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}

	state, exists := l.states[key]

	// Initialize state for a new IP.
	if !exists {
		state = windowState{
			windowStart: now,
			window:      policy.Window,
			count:       0,
		}
	}

	// Check for an active block before processing the window.
	if !state.blockedUntil.IsZero() {
		if now.Before(state.blockedUntil) {
			return Decision{
				Allowed:    false,
				Remaining:  0,
				RetryAfter: state.blockedUntil.Sub(now),
				Blocked:    true,
			}, nil
		}

		// The block has expired. Start a fresh window.
		state = windowState{
			windowStart: now,
			window:      policy.Window,
			count:       0,
		}
	}

	// Reset the counter if the window expired or its duration changed.
	if state.window != policy.Window ||
		!now.Before(state.windowStart.Add(state.window)) {
		state.windowStart = now
		state.window = policy.Window
		state.count = 0
	}

	// Allow the request if the limit has not been reached.
	if state.count < policy.Limit {
		state.count++

		l.states[key] = state

		return Decision{
			Allowed:   true,
			Remaining: policy.Limit - state.count,
			Blocked:   false,
		}, nil
	}

	// The limit has already been reached.
	// This request triggers the block.
	state.blockedUntil = now.Add(policy.BlockDuration)

	l.states[key] = state

	return Decision{
		Allowed:    false,
		Remaining:  0,
		RetryAfter: policy.BlockDuration,
		Blocked:    true,
	}, nil
}

// Cleanup removes expired entries from memory.
//
// Rules:
//   - IPs with active blocks are retained.
//   - IPs whose blocks have expired are deleted.
//   - IPs whose rate-limit windows have expired are deleted,
//     regardless of whether they used all their allowed requests.
//
// Cleanup is safe to call concurrently with Allow.
func (l *FixedWindowLimiter) Cleanup() {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, state := range l.states {
		// Keep IPs that are still blocked.
		if !state.blockedUntil.IsZero() {
			if now.Before(state.blockedUntil) {
				continue
			}

			// Block expired. Remove the state.
			// The next request will start a fresh window.
			delete(l.states, key)
			continue
		}

		// Delete state after its window expires.
		if state.window <= 0 ||
			!now.Before(state.windowStart.Add(state.window)) {
			delete(l.states, key)
		}
	}
}

// StartCleanup periodically removes expired entries.
//
// The cleanup goroutine stops when ctx is canceled.
// If interval is zero or negative, a one-minute interval is used.
func StartCleanup(
	ctx context.Context,
	limiter *FixedWindowLimiter,
	interval time.Duration,
) {
	if limiter == nil {
		return
	}

	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				limiter.Cleanup()
			}
		}
	}()
}

