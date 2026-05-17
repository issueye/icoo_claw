package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func InternalToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}
		if c.GetHeader("X-Internal-Token") != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":  "unauthorized",
				"error": "invalid internal token",
			})
			return
		}
		c.Next()
	}
}
