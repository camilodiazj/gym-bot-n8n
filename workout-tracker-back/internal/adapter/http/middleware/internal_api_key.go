package middleware

import (
	"github.com/gin-gonic/gin"
)

// ValidateInternalAPIKey returns a middleware that validates the internal API key
// This is used for n8n -> Backend communication and other internal services
func ValidateInternalAPIKey(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check X-API-Key header first
		providedKey := c.GetHeader("X-API-Key")
		if providedKey == "" {
			// Fallback to Authorization header
			providedKey = c.GetHeader("Authorization")
			// Strip "Bearer " prefix if present
			if len(providedKey) > 7 && providedKey[:7] == "Bearer " {
				providedKey = providedKey[7:]
			}
		}

		if providedKey == "" {
			c.AbortWithStatusJSON(401, gin.H{
				"success": false,
				"error": gin.H{
					"code":    401,
					"message": "API key required",
				},
			})
			return
		}

		if providedKey != apiKey {
			c.AbortWithStatusJSON(403, gin.H{
				"success": false,
				"error": gin.H{
					"code":    403,
					"message": "invalid API key",
				},
			})
			return
		}

		c.Next()
	}
}
