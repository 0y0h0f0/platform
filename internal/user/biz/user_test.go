package biz_test

import (
	"context"
	"testing"

	"task-platform/internal/user/biz"
	"task-platform/internal/user/data"
)

type mockRepo struct {
	createFn        func(ctx context.Context, user *data.User) error
	findByAccountFn func(ctx context.Context, account string) (*data.User, error)
	findByIDFn      func(ctx context.Context, id string) (*data.User, error)
	batchFindFn     func(ctx context.Context, ids []string) ([]*data.User, error)
}

func (m *mockRepo) Create(ctx context.Context, user *data.User) error {
	return m.createFn(ctx, user)
}

func (m *mockRepo) FindByAccount(ctx context.Context, account string) (*data.User, error) {
	return m.findByAccountFn(ctx, account)
}

func (m *mockRepo) FindByID(ctx context.Context, id string) (*data.User, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockRepo) BatchFindByIDs(ctx context.Context, ids []string) ([]*data.User, error) {
	return m.batchFindFn(ctx, ids)
}

func TestHashPassword(t *testing.T) {
	hash, err := biz.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Error("hash is empty")
	}
	if hash == "secret123" {
		t.Error("password was not hashed")
	}
}

func TestVerifyPassword_Success(t *testing.T) {
	hash, _ := biz.HashPassword("secret123")
	if err := biz.VerifyPassword(hash, "secret123"); err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, _ := biz.HashPassword("secret123")
	if err := biz.VerifyPassword(hash, "wrongpassword"); err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestRegister_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(ctx context.Context, user *data.User) error {
			user.ID = "test-uuid"
			return nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	user, err := b.Register(context.Background(), "testuser", "test@example.com", "secret123")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "test-uuid" {
		t.Errorf("ID = %s, want test-uuid", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email = %s", user.Email)
	}
}

func TestRegister_InvalidUsername(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	_, err := b.Register(context.Background(), "ab", "test@example.com", "secret123")
	if err == nil {
		t.Error("expected error for short username")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	_, err := b.Register(context.Background(), "testuser", "not-an-email", "secret123")
	if err == nil {
		t.Error("expected error for invalid email")
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, []string{"password123"})
	_, err := b.Register(context.Background(), "testuser", "test@example.com", "password123")
	if err == nil {
		t.Error("expected error for weak password")
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	_, err := b.Register(context.Background(), "testuser", "test@example.com", "short")
	if err == nil {
		t.Error("expected error for short password")
	}
}

func TestRegister_PasswordNoLetter(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	_, err := b.Register(context.Background(), "testuser", "test@example.com", "12345678")
	if err == nil {
		t.Error("expected error for password without a letter")
	}
}

func TestRegister_PasswordNoDigit(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	_, err := b.Register(context.Background(), "testuser", "test@example.com", "abcdefgh")
	if err == nil {
		t.Error("expected error for password without a digit")
	}
}

func TestRegister_PasswordTooLong(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	long := make([]byte, 65)
	for i := range long {
		long[i] = 'a'
	}
	long[0] = '1' // ensure has digit
	_, err := b.Register(context.Background(), "testuser", "test@example.com", string(long))
	if err == nil {
		t.Error("expected error for too-long password")
	}
}

func TestLogin_Success(t *testing.T) {
	hash, _ := biz.HashPassword("secret123")
	repo := &mockRepo{
		findByAccountFn: func(ctx context.Context, account string) (*data.User, error) {
			return &data.User{
				ID:           "user-1",
				Username:     "testuser",
				PasswordHash: hash,
				Status:       0,
			}, nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	user, err := b.Login(context.Background(), "testuser", "secret123")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "user-1" {
		t.Errorf("ID = %s", user.ID)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := biz.HashPassword("secret123")
	repo := &mockRepo{
		findByAccountFn: func(ctx context.Context, account string) (*data.User, error) {
			return &data.User{ID: "user-1", PasswordHash: hash, Status: 0}, nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	_, err := b.Login(context.Background(), "testuser", "wrongpassword")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestLogin_DisabledUser(t *testing.T) {
	hash, _ := biz.HashPassword("secret123")
	repo := &mockRepo{
		findByAccountFn: func(ctx context.Context, account string) (*data.User, error) {
			return &data.User{ID: "user-1", PasswordHash: hash, Status: 1}, nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	_, err := b.Login(context.Background(), "testuser", "secret123")
	if err == nil {
		t.Error("expected error for disabled user")
	}
}

func TestGetUser_Success(t *testing.T) {
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, id string) (*data.User, error) {
			return &data.User{ID: id, Username: "testuser"}, nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	user, err := b.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %s", user.Username)
	}
}

func TestBatchGetUsers_TooManyIDs(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = "id"
	}
	_, err := b.BatchGetUsers(context.Background(), ids)
	if err == nil {
		t.Error("expected error for too many IDs")
	}
}
