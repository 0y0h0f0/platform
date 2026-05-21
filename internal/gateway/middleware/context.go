package middleware

import (
	"context"
	"log"
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
	v, ok := ctx.Value(CtxKeyUserID).(string)
	if !ok {
		return ""
	}
	return v
}

func GetUsername(ctx context.Context) string {
	v, ok := ctx.Value(CtxKeyUsername).(string)
	if !ok {
		return ""
	}
	return v
}

func GetJTI(ctx context.Context) string {
	v, ok := ctx.Value(CtxKeyJTI).(string)
	if !ok {
		return ""
	}
	return v
}

func GetRequestID(ctx context.Context) string {
	v, ok := ctx.Value(CtxKeyRequestID).(string)
	if !ok {
		return ""
	}
	return v
}

func GetTokenExpiry(ctx context.Context) time.Time {
	v, ok := ctx.Value(CtxKeyTokenExp).(time.Time)
	if !ok {
		log.Printf("WARN: unexpected type for context key %s", CtxKeyTokenExp)
		return time.Time{}
	}
	return v
}
