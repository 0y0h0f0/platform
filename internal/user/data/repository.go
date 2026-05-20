package data

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"task-platform/pkg/xerr"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByAccount(ctx context.Context, account string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	BatchFindByIDs(ctx context.Context, ids []string) ([]*User, error)
	UpdatePasswordHash(ctx context.Context, userID, hash string) error
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") {
			return xerr.NewError(xerr.CodeAlreadyExists, "username or email already exists")
		}
		return xerr.NewError(xerr.CodeInternal, "create user failed")
	}
	return nil
}

func (r *userRepo) FindByAccount(ctx context.Context, account string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Raw(
		`SELECT * FROM user_svc.users WHERE username = ? AND deleted_at IS NULL
		 UNION ALL
		 SELECT * FROM user_svc.users WHERE email = ? AND username != ? AND deleted_at IS NULL
		 LIMIT 1`,
		account, account, account,
	).Scan(&user).Error
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "find user failed")
	}
	if user.ID == "" {
		return nil, xerr.NewError(xerr.CodeNotFound, "user not found")
	}
	return &user, nil
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NewError(xerr.CodeNotFound, "user not found")
	}
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "find user failed")
	}
	return &user, nil
}

func (r *userRepo) BatchFindByIDs(ctx context.Context, ids []string) ([]*User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []*User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, "batch find users failed")
	}
	return users, nil
}

func (r *userRepo) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	err := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("password_hash", hash).Error
	if err != nil {
		return xerr.NewError(xerr.CodeInternal, "update password hash failed")
	}
	return nil
}
