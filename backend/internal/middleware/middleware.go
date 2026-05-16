package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// requestIDKey is the context key for the request ID.
type requestIDKey struct{}

// RequestID generates a unique request ID for every request and stores it in the context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := make([]byte, 8)
		_, _ = rand.Read(id)
		reqID := hex.EncodeToString(id)
		c.Set("requestID", reqID)
		c.Header("X-Request-ID", reqID)
		ctx := context.WithValue(c.Request.Context(), requestIDKey{}, reqID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// SecurityHeaders adds common security-related HTTP headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// Strict-Transport-Security should only be set over HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}
		c.Next()
	}
}

// StructuredLogger logs each request with method, path, status, duration, and request ID.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		reqID, _ := c.Get("requestID")
		log.Printf("[%s] %s %s %d %v",
			reqID,
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}

// RequestSizeLimit limits the request body size to the given number of bytes.
func RequestSizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// TimeoutMiddleware adds a context timeout to each request.
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
