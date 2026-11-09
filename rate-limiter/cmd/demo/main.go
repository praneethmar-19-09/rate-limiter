
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/praneethm/rate-limiter/internal/ratelimit"
	"github.com/praneethm/rate-limiter/internal/redisstore"
)

func main() {
	// Create a context with a timeout.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	// Step 1: Configure Redis connection.
	redisConfig := redisstore.Config{
		Addr:         "localhost:6379",
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// Step 2: Create Redis client.
	client, err := redisstore.NewClient(ctx, redisConfig)
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer client.Close()

	fmt.Println("Successfully connected to Redis!")

	// Step 3: Create Redis key builder.
	keyBuilder := redisstore.NewKeyBuilder("ratelimit-demo")

	// Use a documentation-only IP address for testing.
	ip := "192.0.2.10"

	// Step 4: Generate Redis keys.
	windowKey, err := keyBuilder.WindowKey(ip)
	if err != nil {
		log.Fatal(err)
	}

	blockKey, err := keyBuilder.BlockKey(ip)
	if err != nil {
		log.Fatal(err)
	}

	// Remove only this demo IP's previous state.
	if err := client.Del(ctx, windowKey, blockKey).Err(); err != nil {
		log.Fatal(err)
	}

	// Step 5: Configure the rate-limit policy.
	policy := ratelimit.Policy{
		Limit:         5,
		Window:        60 * time.Second,
		BlockDuration: 30 * time.Second,
	}

	// Step 6: Create Redis-backed limiter.
	limiter, err := redisstore.NewRedisFixedWindowLimiter(
		client,
		keyBuilder,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nRedis Fixed Window Rate Limiter")
	fmt.Println("--------------------------------")
	fmt.Println("IP:", ip)
	fmt.Println("Limit:", policy.Limit)
	fmt.Println("Window:", policy.Window)
	fmt.Println("Block duration:", policy.BlockDuration)

	// Step 7: Send 7 requests from the same IP.
	for i := 1; i <= 7; i++ {
		decision, err := limiter.Allow(ctx, ip, policy)
		if err != nil {
			log.Fatalf("Request %d failed: %v", i, err)
		}

		fmt.Printf(
			"\nRequest %d\nAllowed: %v\nRemaining: %d\nBlocked: %v\nRetryAfter: %v\n",
			i,
			decision.Allowed,
			decision.Remaining,
			decision.Blocked,
			decision.RetryAfter,
		)
	}

	// Step 8: Display Redis keys for inspection.
	fmt.Println("\nRedis keys:")
	fmt.Println("Window key:", windowKey)
	fmt.Println("Block key:", blockKey)

	windowTTL, err := client.PTTL(ctx, windowKey).Result()
	if err != nil {
		log.Fatal(err)
	}

	blockTTL, err := client.PTTL(ctx, blockKey).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Window TTL:", windowTTL)
	fmt.Println("Block TTL:", blockTTL)

	fmt.Println("\nDemo completed successfully!")
}
