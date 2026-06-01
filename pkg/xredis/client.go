package xredis

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func New(addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     envInt("REDIS_POOL_SIZE", 100),
		MinIdleConns: envInt("REDIS_MIN_IDLE_CONNS", 10),
		DialTimeout:  envDurationSeconds("REDIS_DIAL_TIMEOUT_SECONDS", 3*time.Second),
		ReadTimeout:  envDurationSeconds("REDIS_READ_TIMEOUT_SECONDS", 2*time.Second),
		WriteTimeout: envDurationSeconds("REDIS_WRITE_TIMEOUT_SECONDS", 2*time.Second),
		PoolTimeout:  envDurationSeconds("REDIS_POOL_TIMEOUT_SECONDS", 4*time.Second),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connect: %w", err)
	}

	AddMetricsHook(rdb)

	return rdb, nil
}

func envInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		log.Printf("WARN: invalid value for %s=%q, using default %d", key, s, defaultVal)
		return defaultVal
	}
	return v
}

func envDurationSeconds(key string, defaultVal time.Duration) time.Duration {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		log.Printf("WARN: invalid value for %s=%q, using default %s", key, s, defaultVal)
		return defaultVal
	}
	return time.Duration(v) * time.Second
}
