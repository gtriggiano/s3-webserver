package metricsServer

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type MetricsProvider interface {
	RegisterMetrics(prometheus.Registerer)
}

type MetricsServerConfig struct {
	ErrChan             chan<- error
	ListenPort          int
	ShutdownWaitSeconds int
	Logger              *zap.SugaredLogger
	MainCtx             context.Context
	MetricsProvider     MetricsProvider
	WG                  *sync.WaitGroup
}
