package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/school-invoice/backend/internal/logger"
)

// AddLogger to add logger into gin context
func AddLogger(l *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ContextKeyLogger, l)
		c.Next()
	}
}

// Logger middleware logs request details
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// Log request
		log.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Printf("[GIN] Error: %s", e.Error())
			}
		}
	}
}
