package webServer

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Proxy interface {
	Answer(*gin.Context)
}

type WebserverConfig struct {
	Proxy               Proxy
	Registry            *prometheus.Registry
	ErrChan             chan<- error
	Logger              *zap.SugaredLogger
	WG                  *sync.WaitGroup
	MainCtx             context.Context
	ListenPort          int
	ShutdownWaitSeconds int
	TrustProxy          bool
	TLSCaPath           string
	TLSCertPath         string
	TLSKeyPath          string
}
