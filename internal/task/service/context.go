package service

import "context"

type contextKey string

const (
	ctxKeyUserID   contextKey = "user_id"
	ctxKeyUsername contextKey = "username"
)

func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

func GetUsername(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUsername).(string)
	return v
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, ctxKeyUsername, username)
}
