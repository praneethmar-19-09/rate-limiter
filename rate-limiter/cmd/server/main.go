package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/praneethm/rate-limiter/internal/config"
	"github.com/praneethm/rate-limiter/internal/envoyrls"
	"github.com/praneethm/rate-limiter/internal/ratelimit"
	"github.com/praneethm/rate-limiter/internal/redisstore"
)

func main() {
	// 1. Load application configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Create a context that listens for termination signals.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// 3. Configure and connect to Redis.
	redisCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

	redisClient, err := redisstore.NewClient(
		redisCtx,
		redisstore.Config{
			Addr:         cfg.RedisAddr,
			PoolSize:     cfg.RedisPoolSize,
			MinIdleConns: cfg.RedisMinIdleConns,
		},
	)
	cancel()
	if err != nil {
		log.Fatalf("Redis initialization failed: %v", err)
	}
	defer redisClient.Close()

	log.Println("Connected to Redis:", cfg.RedisAddr)

	// Periodically log Redis pool statistics.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := redisClient.PoolStats()
				log.Printf(
					"Redis pool: total=%d idle=%d stale=%d hits=%d misses=%d timeouts=%d",
					stats.TotalConns,
					stats.IdleConns,
					stats.StaleConns,
					stats.Hits,
					stats.Misses,
					stats.Timeouts,
				)
			}
		}
	}()

	// 4. Create the Redis key builder.
	keyBuilder := redisstore.NewKeyBuilder("ratelimit")

	// 5. Create the Redis-backed limiter.
	limiter, err := redisstore.NewRedisFixedWindowLimiter(
		redisClient,
		keyBuilder,
	)
	if err != nil {
		log.Fatalf("Failed to create rate limiter: %v", err)
	}

	// 6. Configure the rate-limit policy.
	policy := ratelimit.Policy{
		Limit:         cfg.RateLimitLimit,
		Window:        cfg.RateLimitWindow,
		BlockDuration: cfg.RateLimitBlockDuration,
	}

	// 7. Create the Envoy RLS handler.
	handler, err := envoyrls.NewHandler(
		limiter,
		cfg.RateLimitDomain,
		policy,
	)
	if err != nil {
		log.Fatalf("Failed to create Envoy RLS handler: %v", err)
	}

	// 8. Create the gRPC server.
	server, err := envoyrls.NewServer(cfg.RLSAddr, handler)
	if err != nil {
		log.Fatalf("Failed to create gRPC server: %v", err)
	}

	log.Println("Envoy Rate Limit Service listening on", cfg.RLSAddr)
	log.Println("Rate-limit domain:", cfg.RateLimitDomain)

	log.Printf(
		"Policy: limit=%d, window=%s, block=%s",
		policy.Limit,
		policy.Window,
		policy.BlockDuration,
	)

	// 9. Serve gRPC requests.
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Serve()
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}

	case <-ctx.Done():
		log.Println("Shutdown signal received")
		server.GracefulStop()
		log.Println("gRPC server stopped")
	}
}
