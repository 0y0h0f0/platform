package middleware

import (
	"github.com/gin-gonic/gin"
)

func RateLimitByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func RateLimitByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
