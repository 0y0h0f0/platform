package service

import (
	"context"
	"time"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/user/biz"
	"task-platform/internal/user/data"
	"task-platform/pkg/xerr"
	"task-platform/pkg/xjwt"
)

const jwtTTL = 2 * time.Hour

type UserService struct {
	userv1.UnimplementedUserServiceServer
	biz        *biz.UserBiz
	jwtManager *xjwt.Manager
}

func NewUserService(b *biz.UserBiz, jwtManager *xjwt.Manager) *UserService {
	return &UserService{biz: b, jwtManager: jwtManager}
}

func (s *UserService) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	user, err := s.biz.Register(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	token, _, err := s.jwtManager.Generate(user.ID, user.Username, jwtTTL)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "generate token failed")
	}
	return &userv1.RegisterResponse{
		AccessToken: token,
		User:        toProtoUser(user),
	}, nil
}

func (s *UserService) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	user, err := s.biz.Login(ctx, req.Account, req.Password)
	if err != nil {
		return nil, err
	}
	token, _, err := s.jwtManager.Generate(user.ID, user.Username, jwtTTL)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "generate token failed")
	}
	return &userv1.LoginResponse{
		AccessToken: token,
		User:        toProtoUser(user),
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := s.biz.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &userv1.GetUserResponse{
		User: toProtoUser(user),
	}, nil
}

func (s *UserService) BatchGetUsers(ctx context.Context, req *userv1.BatchGetUsersRequest) (*userv1.BatchGetUsersResponse, error) {
	users, err := s.biz.BatchGetUsers(ctx, req.UserIds)
	if err != nil {
		return nil, err
	}
	protoUsers := make([]*userv1.User, len(users))
	for i, u := range users {
		protoUsers[i] = toProtoUser(u)
	}
	return &userv1.BatchGetUsersResponse{
		Users: protoUsers,
	}, nil
}

func toProtoUser(u *data.User) *userv1.User {
	return &userv1.User{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Nickname:  u.Nickname,
		AvatarUrl: u.AvatarURL,
		Status:    u.Status,
	}
}
