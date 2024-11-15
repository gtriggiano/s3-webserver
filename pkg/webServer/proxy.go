package webServer

import "github.com/gin-gonic/gin"

type Proxy interface {
	Answer(*gin.Context)
}
