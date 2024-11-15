package webServer

import (
	"github.com/gin-gonic/gin"
)

func s3Middleware(config *WebserverConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Method == "GET" {
			config.Proxy.Answer(ctx)
		}
	}
}
