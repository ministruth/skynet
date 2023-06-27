package log

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()

		entry := New().WithFields(logrus.Fields{
			"code":    statusCode,
			"latency": latency.Nanoseconds(),
			"ip":      clientIP,
			"method":  c.Request.Method,
			"path":    path,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := fmt.Sprintf("%s [%s] \"%s\" %d %s", clientIP, c.Request.Method, path, statusCode, latency.String())
			entry.Info(msg)
		}
	}
}
