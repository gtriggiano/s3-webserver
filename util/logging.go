package util

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ginRequestData struct {
	ClientIP   string
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
	UA         string
	Msg        string
}

func init() {
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.DurationFieldInteger = true
}

func JSONLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()

		c.Next()

		msg := c.Errors.String()
		if msg == "" {
			msg = "Request"
		}

		request := &ginRequestData{
			ClientIP:   c.ClientIP(),
			Method:     c.Request.Method,
			Path:       path,
			StatusCode: c.Writer.Status(),
			Duration:   time.Since(start),
			UA:         c.Request.UserAgent(),
			Msg:        msg,
		}

		go logRequest(request)
	}
}

func logRequest(request *ginRequestData) {
	var evt *zerolog.Event

	switch {
	case request.StatusCode >= 500:
		{
			evt = log.Error()
		}
	case request.StatusCode >= 400:
		{
			evt = log.Warn()
		}
	default:
		{
			evt = log.Info()
		}
	}

	evt.Str("ip", request.ClientIP).
		Str("method", request.Method).
		Str("path", request.Path).
		Int("status", request.StatusCode).
		Dur("responseTime", request.Duration).
		Str("userAgent", request.UA).
		Msg(request.Msg)
}
