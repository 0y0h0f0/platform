package biz_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"task-platform/internal/user/biz"
	"task-platform/internal/user/data"
)

type redisRepo struct {
	users map[string]*data.User
}

func (m *redisRepo) Create(ctx context.Context, user *data.User) error {
	user.ID = "test-uuid-" + user.Username
	m.users[user.ID] = user
	return nil
}

func (m *redisRepo) FindByAccount(ctx context.Context, account string) (*data.User, error) {
	for _, u := range m.users {
		if u.Username == account || u.Email == account {
			return u, nil
		}
	}
	return nil, nil
}

func (m *redisRepo) FindByID(ctx context.Context, id string) (*data.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *redisRepo) BatchFindByIDs(ctx context.Context, ids []string) ([]*data.User, error) {
	var out []*data.User
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

func (m *redisRepo) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	if u, ok := m.users[userID]; ok {
		u.PasswordHash = hash
	}
	return nil
}

func setupRedis(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start redis: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")

	rdb := redis.NewClient(&redis.Options{
		Addr: host + ":" + port.Port(),
	})
	return rdb
}

func TestBatchGetUsers_WithRedisCache(t *testing.T) {
	rdb := setupRedis(t)
	repo := &redisRepo{
		users: map[string]*data.User{
			"user-a": {ID: "user-a", Username: "usera", Email: "usera@test.com"},
			"user-b": {ID: "user-b", Username: "userb", Email: "userb@test.com"},
		},
	}
	bu := biz.NewUserBiz(repo, rdb, nil)
	ctx := context.Background()

	// First call: cache miss, fetch from repo
	users, err := bu.BatchGetUsers(ctx, []string{"user-a", "user-b"})
	if err != nil {
		t.Fatalf("first BatchGetUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len = %d, want 2", len(users))
	}

	// Second call: should hit Redis cache (repo is unchanged but items were cached)
	users2, err := bu.BatchGetUsers(ctx, []string{"user-a", "user-b"})
	if err != nil {
		t.Fatalf("second BatchGetUsers: %v", err)
	}
	if len(users2) != 2 {
		t.Fatalf("len = %d, want 2", len(users2))
	}
}

func TestGetUser_WithRedisCache(t *testing.T) {
	rdb := setupRedis(t)
	repo := &redisRepo{
		users: map[string]*data.User{
			"user-x": {ID: "user-x", Username: "userx", Email: "userx@test.com"},
		},
	}
	bu := biz.NewUserBiz(repo, rdb, nil)
	ctx := context.Background()

	// First call: cache miss
	user, err := bu.GetUser(ctx, "user-x")
	if err != nil {
		t.Fatalf("first GetUser: %v", err)
	}
	if user.Username != "userx" {
		t.Errorf("username = %s", user.Username)
	}

	// Second call: should hit cache
	user2, err := bu.GetUser(ctx, "user-x")
	if err != nil {
		t.Fatalf("second GetUser: %v", err)
	}
	if user2.Username != "userx" {
		t.Errorf("username = %s", user2.Username)
	}
}

func TestBatchGetUsers_PartialCacheHit(t *testing.T) {
	rdb := setupRedis(t)
	repo := &redisRepo{
		users: map[string]*data.User{
			"user-m": {ID: "user-m", Username: "userm", Email: "userm@test.com"},
			"user-n": {ID: "user-n", Username: "usern", Email: "usern@test.com"},
		},
	}
	bu := biz.NewUserBiz(repo, rdb, nil)
	ctx := context.Background()

	// Prime the cache for user-m only
		_, _ = bu.BatchGetUsers(ctx, []string{"user-m"})

	// Now request both - user-m should hit cache, user-n hits DB
	users, err := bu.BatchGetUsers(ctx, []string{"user-m", "user-n"})
	if err != nil {
		t.Fatalf("BatchGetUsers: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("len = %d, want 2", len(users))
	}
}
