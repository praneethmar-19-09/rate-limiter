
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/praneethm/rate-limiter/internal/redisstore"
)

func main() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	cfg := redisstore.Config{
		Addr:         "localhost:6379",
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	client, err := redisstore.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("Successfully connected to Redis!")

	// Write a test key.
	err = client.Set(ctx, "rate-limiter:test", "connected", time.Minute).Err()
	if err != nil {
		log.Fatal(err)
	}

	// Read the test key.
	value, err := client.Get(ctx, "rate-limiter:test").Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Redis test value:", value)

	// Create Redis key builder.
	keyBuilder := redisstore.NewKeyBuilder("ratelimit")

	ip := "192.168.1.10"

	// Generate the window key.
	windowKey, err := keyBuilder.WindowKey(ip)
	if err != nil {
		log.Fatal(err)
	}

	// Generate the block key.
	blockKey, err := keyBuilder.BlockKey(ip)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Window key:", windowKey)
	fmt.Println("Block key:", blockKey)

	// Store a sample request count.
	err = client.Set(ctx, windowKey, 75, time.Minute).Err()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sample request count stored in Redis")

	// Verify the value.
	count, err := client.Get(ctx, windowKey).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Request count:", count)

	fmt.Println("Block key:", blockKey)
}


