package middleware

import (
	"github.com/gin-gonic/gin"
)

// CodeResolver is a function that resolves a short code to a user ID
type CodeResolver func(code string) (string, error)

// ValidateAuth returns a middleware that validates authentication
// It supports two modes:
// 1. Short code auth via ?c= query parameter (production)
// 2. Fallback to ?user_id= for development
func ValidateAuth(codeResolver CodeResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Priority 1: Short code (?c=)
		if code := c.Query("c"); code != "" && codeResolver != nil {
			userID, err := codeResolver(code)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired code"})
				return
			}
			c.Set("user_id", userID)
			c.Next()
			return
		}

		// Priority 2: Development fallback (?user_id=)
		if userID := c.Query("user_id"); userID != "" {
			c.Set("user_id", userID)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
	}
}
