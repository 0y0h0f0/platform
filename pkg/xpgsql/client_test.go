package xpgsql

import (
	"context"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestNew_NoDB(t *testing.T) {
	db, err := New("host=127.0.0.1 port=19999 user=test dbname=test connect_timeout=1")
	if err == nil {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		t.Log("gorm did not error on invalid DSN (lazy connection)")
		return
	}
	// Error path: gorm ping failed, New returned nil and error
	if db != nil {
		t.Error("db should be nil on error")
	}
}

func TestNew(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	db, err := New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Fatal("db is nil")
	}
}

func TestNew_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if db == nil {
		t.Fatal("db is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 100 {
		t.Errorf("MaxOpenConns = %d, want 100", stats.MaxOpenConnections)
	}
}
