package redisstore

import "github.com/redis/go-redis/v9"

// fixedWindowScript atomically evaluates a rate-limit request.
//
// KEYS[1] = Window counter key
// KEYS[2] = Block key
//
// ARGV[1] = Request limit
// ARGV[2] = Window duration in milliseconds
// ARGV[3] = Block duration in milliseconds
//
// Return values:
// {allowed, remaining, blocked, retry_after_ms}
// [0] = Allowed (1 = yes, 0 = no)
// [1] = Remaining requests
// [2] = Blocked (1 = yes, 0 = no)
// [3] = RetryAfter in milliseconds
var fixedWindowScript = redis.NewScript(`
local windowKey = KEYS[1]
local blockKey = KEYS[2]

local limit = tonumber(ARGV[1])
local windowMs = tonumber(ARGV[2])
local blockMs = tonumber(ARGV[3])

-- Validate script arguments.
if not limit or not windowMs or not blockMs then
    return redis.error_reply("invalid rate-limit arguments")
end

if limit <= 0 or windowMs <= 0 or blockMs <= 0 then
    return redis.error_reply("rate-limit arguments must be positive")
end

-- Step 1: Check whether the IP is currently blocked.
local blockExists = redis.call("EXISTS", blockKey)

if blockExists == 1 then
    local retryAfter = redis.call("PTTL", blockKey)

    return {
        0,          -- Allowed
        0,          -- Remaining
        1,          -- Blocked
        retryAfter  -- RetryAfter in milliseconds
    }
end

-- Step 2: Increment the request counter.
local count = redis.call("INCR", windowKey)

-- Step 3: Set window expiration on the first request.
-- Also repair a counter that unexpectedly has no TTL.
local windowTTL = redis.call("PTTL", windowKey)

if count == 1 or windowTTL == -1 then
    redis.call("PEXPIRE", windowKey, windowMs)
end

-- Step 4: Check whether the request is within the limit.
if count <= limit then
    local remaining = limit - count

    return {
        1,          -- Allowed
        remaining,  -- Remaining
        0,          -- Not blocked
        0           -- No retry delay
    }
end

-- Step 5: Limit exceeded. Create a temporary block.
redis.call("SET", blockKey, "1", "PX", blockMs)

-- Step 6: Remove the old window counter.
-- After the block expires, the next request starts fresh.
redis.call("DEL", windowKey)

-- Step 7: Return the block decision.
return {
    0,          -- Not allowed
    0,          -- Remaining
    1,          -- Blocked
    blockMs     -- RetryAfter in milliseconds
}
`)

// The script is executed later through fixedWindowScript.Run().
// go-redis handles EVALSHA and falls back to EVAL when needed.
