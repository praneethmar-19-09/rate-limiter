package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RedisAddr       string
	RLSAddr         string
	RateLimitDomain string

	RateLimitLimit         int64
	RateLimitWindow        time.Duration
	RateLimitBlockDuration time.Duration

	RedisPoolSize     int
	RedisMinIdleConns int
}

func Load() (Config, error) {
	cfg := Config{
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RLSAddr:           getEnv("RLS_ADDR", ":8081"),
		RateLimitDomain:   getEnv("RATE_LIMIT_DOMAIN", "backend-api"),
		RedisPoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 20),
		RedisMinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 0),
	}

	limit, err := strconv.ParseInt(
		getEnv("RATE_LIMIT_LIMIT", "5"), 10, 64,
	)
	if err != nil || limit <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_LIMIT must be a positive integer")
	}

	window, err := time.ParseDuration(
		getEnv("RATE_LIMIT_WINDOW", "1m"),
	)
	if err != nil || window <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_WINDOW must be a positive duration")
	}

	blockDuration, err := time.ParseDuration(
		getEnv("RATE_LIMIT_BLOCK_DURATION", "30s"),
	)
	if err != nil || blockDuration <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_BLOCK_DURATION must be a positive duration")
	}

	cfg.RateLimitLimit = limit
	cfg.RateLimitWindow = window
	cfg.RateLimitBlockDuration = blockDuration

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
