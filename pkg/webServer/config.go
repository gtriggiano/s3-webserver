package webServer

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type WebserverConfig struct {
	Proxy               Proxy
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
