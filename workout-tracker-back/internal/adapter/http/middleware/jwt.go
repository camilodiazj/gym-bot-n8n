package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID string `json:"sub"`
	Date   string `json:"date"`
	jwt.RegisteredClaims
}

// ValidateJWT returns a middleware that validates JWT tokens
// It supports two modes:
// 1. Token-based auth via ?token= query parameter (production)
// 2. Fallback to ?user_id= for backwards compatibility (development)
func ValidateJWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			// Fallback: check user_id for backwards compatibility
			if userID := c.Query("user_id"); userID != "" {
				c.Set("user_id", userID)
				c.Next()
				return
			}
			c.AbortWithStatusJSON(401, gin.H{"error": "token required"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token claims"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
