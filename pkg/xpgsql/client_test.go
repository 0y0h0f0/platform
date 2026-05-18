package xpgsql

import (
	"os"
	"testing"
)

func TestNew_NoDB(t *testing.T) {
	db, err := New("host=127.0.0.1 port=19999 user=test dbname=test connect_timeout=1")
	if err == nil {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		return
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
