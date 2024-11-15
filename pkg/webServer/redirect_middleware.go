package webServer

import "github.com/gin-gonic/gin"

func redirectMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}
