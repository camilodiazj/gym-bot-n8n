// Package middleware contains HTTP middleware (auth, CORS, error handling).
package middleware

import (
	"github.com/gin-gonic/gin"
)

// CodeResolver is a function that resolves a short code to a user ID
type CodeResolver func(code string) (string, error)

// ValidateAuth returns a middleware that authenticates the incoming request.
//
// Priority 1 (production + development): short code via ?c=<code>, resolved
// through the supplied CodeResolver (typically backed by magic_links).
//
// Priority 2 (development ONLY, gated by allowDevUserIDAuth=true): accept
// ?user_id=<uuid> as authoritative. Closed-by-default because user_ids are
// not secrets — they leak through old WhatsApp URLs, logs, and debug
// responses. Wiring this on in production would let anyone impersonate any
// user by guessing or harvesting their user_id.
//
// Set the ALLOW_DEV_USER_ID_AUTH=true environment variable (read by the
// router at startup) to enable Priority 2 locally or in tests.
//
// When auth succeeds, "auth_method" is set on the gin context ("short_code"
// or "dev_user_id_fallback") so handlers / observability tooling can trace
// which path was taken.
func ValidateAuth(codeResolver CodeResolver, allowDevUserIDAuth bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Priority 1: short code (?c=)
		if code := c.Query("c"); code != "" && codeResolver != nil {
			userID, err := codeResolver(code)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired code"})
				return
			}
			c.Set("user_id", userID)
			c.Set("auth_method", "short_code")
			c.Next()
			return
		}

		// Priority 2: development fallback (?user_id=), gated by env flag.
		if allowDevUserIDAuth {
			if userID := c.Query("user_id"); userID != "" {
				c.Set("user_id", userID)
				c.Set("auth_method", "dev_user_id_fallback")
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
	}
}
