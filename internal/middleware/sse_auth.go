package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SSEAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "SSE token missing",
			})
			c.Abort()
			return
		}

		// Existing AuthMiddleware যেন normal Authorization header পায়
		c.Request.Header.Set(
			"Authorization",
			"Bearer "+token,
		)

		c.Next()
	}
}