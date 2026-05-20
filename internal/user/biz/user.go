package biz

import (
	"context"
	"encoding/json"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"task-platform/internal/user/data"
	"task-platform/pkg/xerr"
)

const (
	userCacheTTL    = 5 * time.Minute
	userCachePrefix = "user:"
)

var (
	bcryptCost = func() int {
		s := os.Getenv("BCRYPT_COST")
		if s == "" {
			return 6
		}
		v, err := strconv.Atoi(s)
		if err != nil || v < 4 || v > 14 {
			return 6
		}
		return v
	}()

	bcryptMaxCost = func() int {
		if bcryptCost+2 > 14 {
			return 14
		}
		return bcryptCost + 2
	}()
)

var (
	usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	hasLetter  = regexp.MustCompile(`[a-zA-Z]`)
	hasDigit   = regexp.MustCompile(`[0-9]`)
)

type UserBiz struct {
	repo          data.UserRepository
	rdb           *redis.Client
	weakPasswords map[string]bool
}

func NewUserBiz(repo data.UserRepository, rdb *redis.Client, weakPasswords []string) *UserBiz {
	set := make(map[string]bool, len(weakPasswords))
	for _, pw := range weakPasswords {
		set[pw] = true
	}
	return &UserBiz{repo: repo, rdb: rdb, weakPasswords: set}
}

func HashPassword(plain string) (string, error) {
	return HashPasswordWithCost(plain, bcryptCost)
}

func HashPasswordWithCost(plain string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", xerr.NewError(xerr.CodeInternal, "hash password failed")
	}
	return string(bytes), nil
}

func VerifyPassword(hashed, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	if err != nil {
		return xerr.NewError(xerr.CodeUnauthenticated, "invalid password")
	}
	return nil
}

func NeedsRehash(hash string) bool {
	return needsRehash(hash)
}

func (b *UserBiz) Register(ctx context.Context, username, email, password string) (*data.User, error) {
	if !usernameRE.MatchString(username) {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "username: 3-32 chars, letters, digits, underscores")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "invalid email")
	}
	pwLen := utf8.RuneCountInString(password)
	if pwLen < 8 || pwLen > 64 {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "password must be 8-64 characters")
	}
	if !hasLetter.MatchString(password) || !hasDigit.MatchString(password) {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "password must contain at least one letter and one digit")
	}
	if b.weakPasswords[password] {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "password is too weak")
	}

	hashed, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &data.User{
		Username:     username,
		Email:        strings.ToLower(email),
		PasswordHash: hashed,
	}
	if err := b.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (b *UserBiz) Login(ctx context.Context, account, password string) (*data.User, error) {
	user, err := b.repo.FindByAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if err := VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, err
	}
	if user.Status != 0 {
		return nil, xerr.NewError(xerr.CodePermissionDenied, "account is disabled")
	}
	if needsRehash(user.PasswordHash) {
		if rehashed, err := HashPassword(password); err == nil {
			_ = b.repo.UpdatePasswordHash(ctx, user.ID, rehashed)
		}
	}
	return user, nil
}

func needsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false
	}
	return cost > bcryptMaxCost
}

func (b *UserBiz) GetUser(ctx context.Context, userID string) (*data.User, error) {
	if b.rdb != nil {
		if u := b.getCachedUser(ctx, userID); u != nil {
			return u, nil
		}
	}

	user, err := b.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if b.rdb != nil {
		b.cacheUser(ctx, user)
	}
	return user, nil
}

func (b *UserBiz) BatchGetUsers(ctx context.Context, ids []string) ([]*data.User, error) {
	if len(ids) > 100 {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "too many user ids, max 100")
	}
	if len(ids) == 0 {
		return nil, nil
	}

	if b.rdb == nil {
		return b.repo.BatchFindByIDs(ctx, ids)
	}

	var missed []string
	users := make([]*data.User, 0, len(ids))
	for _, id := range ids {
		if u := b.getCachedUser(ctx, id); u != nil {
			users = append(users, u)
		} else {
			missed = append(missed, id)
		}
	}

	if len(missed) == 0 {
		return users, nil
	}

	dbUsers, err := b.repo.BatchFindByIDs(ctx, missed)
	if err != nil {
		return nil, err
	}

	for _, u := range dbUsers {
		b.cacheUser(ctx, u)
	}

	return append(users, dbUsers...), nil
}

func (b *UserBiz) getCachedUser(ctx context.Context, userID string) *data.User {
	raw, err := b.rdb.Get(ctx, userCachePrefix+userID).Bytes()
	if err != nil {
		return nil
	}
	var u data.User
	if err := json.Unmarshal(raw, &u); err != nil {
		return nil
	}
	return &u
}

func (b *UserBiz) cacheUser(ctx context.Context, u *data.User) {
	raw, err := json.Marshal(u)
	if err != nil {
		return
	}
	b.rdb.Set(ctx, userCachePrefix+u.ID, raw, userCacheTTL)
}
