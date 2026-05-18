package data_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"task-platform/internal/user/data"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("task_platform"),
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
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("pg connection string: %v", err)
	}

	db, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	rootDir := findRoot()
	migrationSQL, err := os.ReadFile(filepath.Join(rootDir, "migrations", "user_svc", "000001_init.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		t.Fatalf("run migration: %v", err)
	}

	return db
}

func findRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func TestCreate_Success(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	user := &data.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$hashed",
	}
	err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if user.ID == "" {
		t.Error("ID should be set after create")
	}
}

func TestCreate_Duplicate(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	u1 := &data.User{Username: "dup", Email: "dup1@example.com", PasswordHash: "x"}
	if err := repo.Create(context.Background(), u1); err != nil {
		t.Fatalf("create: %v", err)
	}

	u2 := &data.User{Username: "dup", Email: "dup2@example.com", PasswordHash: "x"}
	err := repo.Create(context.Background(), u2)
	if err == nil {
		t.Error("expected error for duplicate username")
	}
}

func TestFindByAccount_Username(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	_ = repo.Create(context.Background(), &data.User{
		Username: "findme", Email: "findme@example.com", PasswordHash: "x",
	})

	u, err := repo.FindByAccount(context.Background(), "findme")
	if err != nil {
		t.Fatalf("find by username: %v", err)
	}
	if u.Email != "findme@example.com" {
		t.Errorf("email = %s", u.Email)
	}
}

func TestFindByAccount_Email(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	_ = repo.Create(context.Background(), &data.User{
		Username: "emailuser", Email: "findbyemail@example.com", PasswordHash: "x",
	})

	u, err := repo.FindByAccount(context.Background(), "findbyemail@example.com")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if u.Username != "emailuser" {
		t.Errorf("username = %s", u.Username)
	}
}

func TestFindByAccount_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	_, err := repo.FindByAccount(context.Background(), "noone")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestFindByID_Success(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	u := &data.User{Username: "iduser", Email: "iduser@example.com", PasswordHash: "x"}
	_ = repo.Create(context.Background(), u)

	found, err := repo.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Username != "iduser" {
		t.Errorf("username = %s", found.Username)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	_, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestBatchFindByIDs_Success(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	u1 := &data.User{Username: "batch1", Email: "batch1@example.com", PasswordHash: "x"}
	u2 := &data.User{Username: "batch2", Email: "batch2@example.com", PasswordHash: "x"}
	_ = repo.Create(context.Background(), u1)
	_ = repo.Create(context.Background(), u2)

	users, err := repo.BatchFindByIDs(context.Background(), []string{u1.ID, u2.ID})
	if err != nil {
		t.Fatalf("batch find: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("len = %d, want 2", len(users))
	}
}

func TestBatchFindByIDs_Empty(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	users, err := repo.BatchFindByIDs(context.Background(), []string{})
	if err != nil {
		t.Fatalf("batch find empty: %v", err)
	}
	if users != nil {
		t.Error("expected nil for empty input")
	}
}

func TestBatchFindByIDs_PartialMiss(t *testing.T) {
	db := setupDB(t)
	repo := data.NewUserRepository(db)

	u := &data.User{Username: "partial", Email: "partial@example.com", PasswordHash: "x"}
	_ = repo.Create(context.Background(), u)

	users, err := repo.BatchFindByIDs(context.Background(), []string{u.ID, "00000000-0000-0000-0000-000000000000"})
	if err != nil {
		t.Fatalf("batch find partial: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("len = %d, want 1", len(users))
	}
}
