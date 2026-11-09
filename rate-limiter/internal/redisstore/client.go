
package redisstore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config contains Redis connection configuration.
type Config struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// NewClient creates and initializes a Redis client.
func NewClient(ctx context.Context, cfg Config) (*redis.Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis address must not be empty")
	}

	// Apply default timeouts.
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}

	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 3 * time.Second
	}

	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 3 * time.Second
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Verify that Redis is reachable.
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}
