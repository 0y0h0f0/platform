package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"task-platform/internal/user/biz"
	"task-platform/internal/user/data"
	"task-platform/pkg/xpgsql"
)

func main() {
	count := 500
	if s := os.Getenv("SEED_COUNT"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			count = v
		}
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5433 user=postgres password=postgres dbname=task_platform sslmode=disable"
	}

	db, err := xpgsql.New(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
		os.Exit(1)
	}

	repo := data.NewUserRepository(db)
	b := biz.NewUserBiz(repo, nil, nil)

	concurrency := 50
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	errors := make(chan error, count)

	start := time.Now()
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			username := fmt.Sprintf("load_login_%d", idx)
			email := fmt.Sprintf("load_login_%d@test.local", idx)
			password := fmt.Sprintf("Pass%dword", idx%10000)

			if _, err := b.Register(context.Background(), username, email, password); err != nil {
				errors <- fmt.Errorf("register %s: %w", username, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start)
	errCount := 0
	for range errors {
		errCount++
	}
	fmt.Printf("Seeded %d users in %s (errors: %d)\n", count, elapsed, errCount)
}
