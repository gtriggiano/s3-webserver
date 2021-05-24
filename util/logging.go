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

		data := &ginRequestData{
			ClientIP:   c.ClientIP(),
			Method:     c.Request.Method,
			Path:       path,
			StatusCode: c.Writer.Status(),
			Duration:   time.Since(start),
			UA:         c.Request.UserAgent(),
			Msg:        msg,
		}

		go emitLog(data)
	}
}

func emitLog(data *ginRequestData) {
	var evt *zerolog.Event

	switch {
	case data.StatusCode >= 500:
		{
			evt = log.Error()
		}
	case data.StatusCode >= 400:
		{
			evt = log.Warn()
		}
	default:
		{
			evt = log.Info()
		}
	}

	evt.Str("ip", data.ClientIP).
		Str("method", data.Method).
		Str("path", data.Path).
		Int("status", data.StatusCode).
		Dur("responseTime", data.Duration).
		Str("userAgent", data.UA).
		Msg(data.Msg)
}
