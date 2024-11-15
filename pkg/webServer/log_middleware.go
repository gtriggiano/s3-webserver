package webServer

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func logMiddleware(config *WebserverConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		ctx.Next()

		var statusCode = ctx.Writer.Status()

		var level = zap.InfoLevel
		switch {
		case statusCode >= 500:
			{
				level = zap.ErrorLevel
			}
		case statusCode >= 400:
			{
				level = zap.WarnLevel
			}
		default:
			{
				level = zap.InfoLevel
			}
		}

		log := config.Logger.With(
			"ip", ctx.ClientIP(),
			"method", ctx.Request.Method,
			"host", ctx.Request.Host,
			"uri", ctx.Request.RequestURI,
			"status", ctx.Writer.Status(),
			"ms", time.Since(start).Milliseconds(),
			"ua", ctx.Request.UserAgent(),
		)

		msg := ctx.Errors.String()

		if msg == "" {
			log.Log(level)
		} else {
			log.Log(level, msg)
		}

	}
}
