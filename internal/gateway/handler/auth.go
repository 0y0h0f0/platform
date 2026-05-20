package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

type AuthHandler struct {
	userClient userv1.UserServiceClient
	rdb        *redis.Client
}

func NewAuthHandler(userClient userv1.UserServiceClient, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{userClient: userClient, rdb: rdb}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req userv1.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := xerr.NewError(xerr.CodeInvalidArgument, "invalid request body")
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(resp, middleware.GetRequestID(c.Request.Context())))
		return
	}

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.userClient.Register(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req userv1.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := xerr.NewError(xerr.CodeInvalidArgument, "invalid request body")
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(resp, middleware.GetRequestID(c.Request.Context())))
		return
	}

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.userClient.Login(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	jti := middleware.GetJTI(c.Request.Context())
	exp := middleware.GetTokenExpiry(c.Request.Context())

	ttl := time.Until(exp)
	if ttl > 0 {
		blacklistKey := "blacklist:" + jti
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := h.rdb.Set(ctx, blacklistKey, "1", ttl).Err(); err != nil {
			resp := xerr.NewError(xerr.CodeInternal, "logout failed")
			c.JSON(http.StatusInternalServerError, xerr.ToHTTPResponse(resp, middleware.GetRequestID(c.Request.Context())))
			return
		}
	}

	c.JSON(http.StatusOK, xerr.ToHTTPResponse(nil, middleware.GetRequestID(c.Request.Context())))
}
