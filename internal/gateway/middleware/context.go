package middleware

import (
	"context"
	"time"
)

type ContextKey string

const (
	CtxKeyRequestID ContextKey = "request_id"
	CtxKeyUserID    ContextKey = "user_id"
	CtxKeyUsername  ContextKey = "username"
	CtxKeyJTI       ContextKey = "jti"
	CtxKeyTokenExp  ContextKey = "token_exp"
)

func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyUserID).(string)
	return v
}

func GetUsername(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyUsername).(string)
	return v
}

func GetJTI(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyJTI).(string)
	return v
}

func GetRequestID(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyRequestID).(string)
	return v
}

func GetTokenExpiry(ctx context.Context) time.Time {
	v, _ := ctx.Value(CtxKeyTokenExp).(time.Time)
	return v
}
