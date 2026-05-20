package xredis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func New(addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connect: %w", err)
	}

	AddMetricsHook(rdb)

	return rdb, nil
}
