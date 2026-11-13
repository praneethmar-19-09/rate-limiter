
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/praneethm/rate-limiter/internal/envoyrls"
	"github.com/praneethm/rate-limiter/internal/ratelimit"
	"github.com/praneethm/rate-limiter/internal/redisstore"
)

func main() {
	// 1. Create a context that listens for termination signals.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// 2. Configure Redis.
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	redisClient, err := redisstore.NewClient(redisCtx, redisstore.Config{
		Addr: redisAddr,
	})
	if err != nil {
		log.Fatalf("Redis initialization failed: %v", err)
	}
	defer redisClient.Close()

	log.Println("Connected to Redis:", redisAddr)

	// 3. Create the Redis key builder.
	keyBuilder := redisstore.NewKeyBuilder("ratelimit")

	// 4. Create the Redis-backed limiter.
	limiter, err := redisstore.NewRedisFixedWindowLimiter(
		redisClient,
		keyBuilder,
	)
	if err != nil {
		log.Fatalf("Failed to create rate limiter: %v", err)
	}

	// 5. Configure the rate-limit policy.
	policy := ratelimit.Policy{
		Limit:         5,
		Window:        time.Minute,
		BlockDuration: 30 * time.Second,
	}

	// 6. Create the Envoy RLS handler.
	domain := os.Getenv("RATE_LIMIT_DOMAIN")
	if domain == "" {
		domain = "backend-api"
	}

	handler, err := envoyrls.NewHandler(
		limiter,
		domain,
		policy,
	)
	if err != nil {
		log.Fatalf("Failed to create Envoy RLS handler: %v", err)
	}

	// 7. Start the gRPC server.
	serverAddr := os.Getenv("RLS_ADDR")
	if serverAddr == "" {
		serverAddr = ":8081"
	}

	server, err := envoyrls.NewServer(serverAddr, handler)
	if err != nil {
		log.Fatalf("Failed to create gRPC server: %v", err)
	}

	log.Println("Envoy Rate Limit Service listening on", serverAddr)
	log.Println("Rate-limit domain:", domain)
	log.Printf(
		"Policy: limit=%d, window=%s, block=%s",
		policy.Limit,
		policy.Window,
		policy.BlockDuration,
	)

	// 8. Serve gRPC requests.
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
