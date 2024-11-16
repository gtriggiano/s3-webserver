package metricsServer

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type MetricsServerConfig struct {
	ErrChan             chan<- error
	ListenPort          int
	ShutdownWaitSeconds int
	Logger              *zap.SugaredLogger
	MainCtx             context.Context
	Registry            *prometheus.Registry
	WG                  *sync.WaitGroup
}
