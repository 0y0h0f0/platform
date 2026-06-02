package xratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenBucketScript performs refill and consume atomically in Redis so multiple
// gateway instances share one consistent bucket per key.
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil or last_refill == nil then
    tokens = capacity
    last_refill = now
end

local elapsed = now - last_refill
if elapsed < 0 then elapsed = 0 end

tokens = math.min(capacity, tokens + elapsed * rate)
last_refill = now

local allowed = 0
if tokens >= requested then
    allowed = 1
    tokens = tokens - requested
end

redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
redis.call('EXPIRE', key, math.ceil(capacity / rate) + 60)

return allowed
`)

// TokenBucket implements a Redis-backed token bucket rate limiter.
type TokenBucket struct {
	rdb *redis.Client
}

// New creates a token bucket helper around a Redis client.
func New(rdb *redis.Client) *TokenBucket {
	return &TokenBucket{rdb: rdb}
}

// Allow consumes one token from the bucket and reports whether the request may
// proceed. Rate is tokens per second and burst is bucket capacity.
func (tb *TokenBucket) Allow(ctx context.Context, key string, rate, burst int) (bool, error) {
	now := float64(time.Now().UnixNano()) / 1e9
	v, err := tokenBucketScript.Run(ctx, tb.rdb, []string{key}, burst, rate, now, 1).Int()
	if err != nil {
		return false, err
	}
	return v == 1, nil
}
