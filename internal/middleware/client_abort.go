package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SilenceClientAbortErrors removes expected network abort errors from Gin context
// so they are not logged as application errors by Gin logger middleware.
func SilenceClientAbortErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		filtered := c.Errors[:0]
		for _, item := range c.Errors {
			if item == nil || item.Err == nil {
				filtered = append(filtered, item)
				continue
			}
			if isClientAbortError(item.Err.Error()) {
				continue
			}
			filtered = append(filtered, item)
		}
		c.Errors = filtered
	}
}

func isClientAbortError(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}

	signatures := []string{
		"wsasend: an established connection was aborted",
		"broken pipe",
		"connection reset by peer",
		"use of closed network connection",
		"client disconnected",
	}
	for _, sig := range signatures {
		if strings.Contains(msg, sig) {
			return true
		}
	}
	return false
}

