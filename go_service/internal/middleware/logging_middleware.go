package middleware

import (
	"time"

	"go_service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Tạo request ID để trace
		requestID := uuid.New().String()
		c.Set("requestID", requestID)

		// Thời gian bắt đầu request
		startTime := time.Now()

		// Xử lý request
		c.Next()

		// Thời gian xử lý
		duration := time.Since(startTime)

		// Log thông tin response
		logEvent := logger.Log.Info()
		if c.Writer.Status() >= 400 {
			logEvent = logger.Log.Error()
		}

		logEvent.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", duration).
			Msg("Incoming request")
	}
}
