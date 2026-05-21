package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminAuth(adminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Admin-Key")
		if key != adminKey || adminKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
