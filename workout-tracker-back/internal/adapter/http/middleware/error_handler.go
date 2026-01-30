package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
	"github.com/gymbot/workout-tracker-back/pkg/response"
)

// ErrorHandler is a middleware that handles panics and recovers gracefully
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				c.JSON(http.StatusInternalServerError, response.Response{
					Success: false,
					Error: &response.ErrorInfo{
						Code:    http.StatusInternalServerError,
						Message: "internal server error",
					},
				})
				c.Abort()
			}
		}()
		c.Next()

		// Handle errors set in context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			appErr, ok := err.(*apperror.AppError)
			if !ok {
				appErr = apperror.NewInternalError("internal server error", err)
			}

			c.JSON(appErr.Code, response.Response{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    appErr.Code,
					Message: appErr.Message,
				},
			})
		}
	}
}
