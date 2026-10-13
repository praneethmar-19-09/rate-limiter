package ratelimit

import "context"

type Limiter interface {
	Allow(
		ctx context.Context, 
		key string,
		policy Policy) (Decision, error)
}