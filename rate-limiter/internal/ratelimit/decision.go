package ratelimit

import "time"

type Decision struct {
	Allowed bool;				// whether the request is allowed or not
	RetryAfter time.Duration;	// how long the client should wait before retrying
	Remaining int64;				// how many requests are remaining in the current window
	Blocked bool;				// whether the client is currently blocked
}
