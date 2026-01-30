package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles CORS with configurable origins
func CORS(allowedOrigins []string) gin.HandlerFunc {
	// Build allowed origins map for O(1) lookup
	originsMap := make(map[string]bool)
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
		originsMap[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Determine which origin to allow
		var allowOrigin string
		if allowAll {
			allowOrigin = "*"
		} else if originsMap[origin] {
			allowOrigin = origin
		}

		if allowOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
