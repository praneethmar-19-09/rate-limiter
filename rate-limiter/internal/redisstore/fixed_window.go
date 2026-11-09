package redisstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/praneethm/rate-limiter/internal/ratelimit"
)

var (
	ErrNilRedisClient = errors.New("redis client must not be nil")
	ErrNilKeyBuilder  = errors.New("redis key builder must not be nil")
	ErrInvalidResult  = errors.New("invalid result returned by Redis script")
)

// RedisFixedWindowLimiter implements the rate-limiter interface
// using Redis as the shared state store.
type RedisFixedWindowLimiter struct {
	client *redis.Client
	keys   *KeyBuilder
}

// NewRedisFixedWindowLimiter creates a Redis-backed rate limiter.
func NewRedisFixedWindowLimiter(
	client *redis.Client,
	keys *KeyBuilder,
) (*RedisFixedWindowLimiter, error) {

	if client == nil {
		return nil, ErrNilRedisClient
	}

	if keys == nil {
		return nil, ErrNilKeyBuilder
	}

	return &RedisFixedWindowLimiter{
		client: client,
		keys:   keys,
	}, nil
}

// Allow evaluates a request using the atomic Redis Lua script.
func (l *RedisFixedWindowLimiter) Allow(
	ctx context.Context,
	ip string,
	policy ratelimit.Policy,
) (ratelimit.Decision, error) {

	if err := ctx.Err(); err != nil {
		return ratelimit.Decision{}, err
	}

	if err := policy.Validate(); err != nil {
		return ratelimit.Decision{}, err
	}

	// Generate Redis keys for this IP.
	windowKey, err := l.keys.WindowKey(ip)
	if err != nil {
		return ratelimit.Decision{}, err
	}

	blockKey, err := l.keys.BlockKey(ip)
	if err != nil {
		return ratelimit.Decision{}, err
	}

	// Convert durations into milliseconds.
	windowMs := durationToMilliseconds(policy.Window)
	blockMs := durationToMilliseconds(policy.BlockDuration)

	// Execute the Lua script atomically.
	result, err := fixedWindowScript.Run(
		ctx,
		l.client,
		[]string{windowKey, blockKey},
		policy.Limit,
		windowMs,
		blockMs,
	).Result()

	if err != nil {
		return ratelimit.Decision{}, fmt.Errorf(
			"failed to execute Redis rate-limit script: %w",
			err,
		)
	}

	// Convert the Lua response into our Decision structure.
	return parseDecision(result)
}

// durationToMilliseconds converts a positive Go duration to milliseconds.
// It rounds up so a positive sub-millisecond duration never becomes zero.
func durationToMilliseconds(d time.Duration) int64 {
	ms := int64(d / time.Millisecond)

	if d%time.Millisecond != 0 {
		ms++
	}

	return ms
}

// parseDecision converts the Lua script response into a Decision.
//
// Expected Lua response:
//
// [0] Allowed
// [1] Remaining
// [2] Blocked
// [3] RetryAfter in milliseconds
func parseDecision(result interface{}) (ratelimit.Decision, error) {

	values, ok := result.([]interface{})
	if !ok || len(values) != 4 {
		return ratelimit.Decision{}, ErrInvalidResult
	}

	allowed, err := redisInt(values[0])
	if err != nil {
		return ratelimit.Decision{}, err
	}

	remaining, err := redisInt(values[1])
	if err != nil {
		return ratelimit.Decision{}, err
	}

	blocked, err := redisInt(values[2])
	if err != nil {
		return ratelimit.Decision{}, err
	}

	retryAfterMs, err := redisInt(values[3])
	if err != nil {
		return ratelimit.Decision{}, err
	}

	if (allowed != 0 && allowed != 1) ||
		(blocked != 0 && blocked != 1) ||
		remaining < 0 ||
		retryAfterMs < 0 {
		return ratelimit.Decision{}, ErrInvalidResult
	}

	return ratelimit.Decision{
		Allowed:    allowed == 1,
		Remaining:  remaining,
		Blocked:    blocked == 1,
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
	}, nil
}

// redisInt converts Redis Lua integer results into int64.
func redisInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil

	case int:
		return int64(v), nil

	case string:
		var result int64
		_, err := fmt.Sscan(v, &result)
		return result, err

	case []byte:
		var result int64
		_, err := fmt.Sscan(string(v), &result)
		return result, err

	default:
		return 0, fmt.Errorf(
			"%w: unexpected integer type %T",
			ErrInvalidResult,
			value,
		)
	}
}

// Compile-time check that RedisFixedWindowLimiter implements Limiter.
var _ ratelimit.Limiter = (*RedisFixedWindowLimiter)(nil)
