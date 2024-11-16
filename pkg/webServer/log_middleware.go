package webServer

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func logMiddleware(config *WebserverConfig) gin.HandlerFunc {
	webserverRequestsCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_webserver_requests",
			Help: "Total requests handled by s3-webserver",
		},
		[]string{"host", "status", "method"},
	)
	webserverRequestTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "s3_webserver_request_duration_seconds",
	}, []string{"host"})

	config.Registry.MustRegister(webserverRequestsCounter)
	config.Registry.MustRegister(webserverRequestTime)

	return func(ctx *gin.Context) {
		start := time.Now()

		ctx.Next()

		statusCode := ctx.Writer.Status()
		statusCodeString := http.StatusText(statusCode)
		requestDuration := time.Since(start)

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

		msg := ctx.Errors.String()
		if msg == "" {
			msg = statusCodeString
		}

		config.Logger.With(
			"ip", ctx.ClientIP(),
			"method", ctx.Request.Method,
			"host", ctx.Request.Host,
			"uri", ctx.Request.RequestURI,
			"status", statusCode,
			"ms", requestDuration.Milliseconds(),
			"ua", ctx.Request.UserAgent(),
		).Log(level, msg)

		webserverRequestsCounter.With(prometheus.Labels{
			"host":   ctx.Request.Host,
			"status": strconv.Itoa(statusCode),
			"method": ctx.Request.Method,
		}).Inc()

		webserverRequestTime.With(prometheus.Labels{
			"host": ctx.Request.Host,
		}).Observe(requestDuration.Seconds())
	}
}
