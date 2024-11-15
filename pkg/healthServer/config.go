package healthServer

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type HealthChecker interface {
	CheckHealth() error
}

type HealthServerConfig struct {
	ErrChan             chan<- error
	Logger              *zap.SugaredLogger
	WG                  *sync.WaitGroup
	MainCtx             context.Context
	ListenPort          int
	ShutdownWaitSeconds int
	HealthChecker       HealthChecker
}
