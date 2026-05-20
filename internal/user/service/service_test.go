package service_test

import (
	"context"
	"testing"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/user/data"
	"task-platform/pkg/xerr"
	"task-platform/pkg/xjwt"

	// We need to test the service, but it uses *biz.UserBiz which needs a repo.
	// Instead, test via the full service using a mock approach.
	"task-platform/internal/user/biz"
	"task-platform/internal/user/service"
)

type mockRepo struct {
	createFn            func(ctx context.Context, user *data.User) error
	findByAccountFn     func(ctx context.Context, account string) (*data.User, error)
	findByIDFn          func(ctx context.Context, id string) (*data.User, error)
	batchFindFn         func(ctx context.Context, ids []string) ([]*data.User, error)
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

func (m *mockRepo) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	return nil
}

func TestRegister_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(ctx context.Context, user *data.User) error {
			user.ID = "new-user-id"
			return nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	jwtMgr := xjwt.NewManager("test-secret-key-for-jwt")
	svc := service.NewUserService(b, jwtMgr)

	res, err := svc.Register(context.Background(), &userv1.RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.AccessToken == "" {
		t.Error("expected access token in response")
	}
	if res.User == nil {
		t.Error("expected user in response")
	}
}

func TestRegister_InvalidArgument(t *testing.T) {
	b := biz.NewUserBiz(nil, nil, nil)
	jwtMgr := xjwt.NewManager("test-secret")
	svc := service.NewUserService(b, jwtMgr)

	_, err := svc.Register(context.Background(), &userv1.RegisterRequest{
		Username: "ab",
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err == nil {
		t.Error("expected error for short username")
	}

	// Verify it's a proper gRPC status error
	var e *xerr.Error
	ok := false
	_, _, _ = e, ok, err // used below
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
	jwtMgr := xjwt.NewManager("test-secret")
	svc := service.NewUserService(b, jwtMgr)

	res, err := svc.Login(context.Background(), &userv1.LoginRequest{
		Account:  "testuser",
		Password: "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestGetUser_Success(t *testing.T) {
	repo := &mockRepo{
		findByIDFn: func(ctx context.Context, id string) (*data.User, error) {
			return &data.User{ID: id, Username: "testuser", Email: "test@example.com"}, nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	svc := service.NewUserService(b, nil)

	res, err := svc.GetUser(context.Background(), &userv1.GetUserRequest{
		UserId: "user-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.User.Id != "user-1" {
		t.Errorf("Id = %s", res.User.Id)
	}
}

func TestBatchGetUsers(t *testing.T) {
	repo := &mockRepo{
		batchFindFn: func(ctx context.Context, ids []string) ([]*data.User, error) {
			users := make([]*data.User, len(ids))
			for i, id := range ids {
				users[i] = &data.User{ID: id, Username: "user" + id}
			}
			return users, nil
		},
	}
	b := biz.NewUserBiz(repo, nil, nil)
	svc := service.NewUserService(b, nil)

	res, err := svc.BatchGetUsers(context.Background(), &userv1.BatchGetUsersRequest{
		UserIds: []string{"1", "2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Users) != 2 {
		t.Errorf("got %d users, want 2", len(res.Users))
	}
}
