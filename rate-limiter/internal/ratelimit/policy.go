package ratelimit

import (
	"time"
	"errors"
)

var (
	ErrInvalidLimit = errors.New("rate limit must be greater than zero")
	ErrInvalidWindow = errors.New("rate-limit window must be greater than zero")
	ErrInvalidBlockDuration = errors.New("block duration must be greater than zero")
)

type Policy struct {

	Limit int64;					// max no of req allowed within the window
	Window time.Duration;			//duration of rate limiting window
	BlockDuration time.Duration;	// how long an actor should be blocked after exceeding the limit
}

func (p Policy) Validate() error {
	if p.Limit <= 0 {
		return ErrInvalidLimit
	}
	if p.Window <= 0 {
		return ErrInvalidWindow
	}
	if p.BlockDuration <= 0 {
		return ErrInvalidBlockDuration
	}
	return nil
}