package cli

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gtriggiano/s3-webserver/pkg/healthServer"
	"github.com/gtriggiano/s3-webserver/pkg/metricsServer"
	"github.com/gtriggiano/s3-webserver/pkg/webServer"
	"go.uber.org/zap"
)

type Proxy interface {
	healthServer.HealthChecker
	metricsServer.MetricsProvider
	webServer.Proxy
}

type StartConfig struct {
	WebServerPort       int
	HealthServerPort    int
	MetricsServerPort   int
	ShutdownWaitSeconds int
	TrustProxy          bool
	Proxy               Proxy
	Context             context.Context
	Logger              *zap.SugaredLogger
	TLSCertPath         string
	TLSKeyPath          string
}

func start(config StartConfig) (*sync.WaitGroup, <-chan error, <-chan os.Signal) {
	var wg sync.WaitGroup
	errChan := make(chan error)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	wg.Add(1)
	go webServer.Start(&webServer.WebserverConfig{
		Proxy:               config.Proxy,
		ErrChan:             errChan,
		Logger:              config.Logger.With("component", "web-server"),
		WG:                  &wg,
		MainCtx:             config.Context,
		ListenPort:          config.WebServerPort,
		ShutdownWaitSeconds: config.ShutdownWaitSeconds,
		TrustProxy:          config.TrustProxy,
		TLSCertPath:         config.TLSCertPath,
		TLSKeyPath:          config.TLSKeyPath,
	})

	wg.Add(1)
	go healthServer.Start(&healthServer.HealthServerConfig{
		ErrChan:       errChan,
		ListenPort:    config.HealthServerPort,
		Logger:        config.Logger.With("component", "health-server"),
		MainCtx:       config.Context,
		WG:            &wg,
		HealthChecker: config.Proxy,
	})

	wg.Add(1)
	go metricsServer.Start(&metricsServer.MetricsServerConfig{
		ErrChan:         errChan,
		ListenPort:      config.MetricsServerPort,
		Logger:          config.Logger.With("component", "metrics-server"),
		MainCtx:         config.Context,
		MetricsProvider: config.Proxy,
		WG:              &wg,
	})

	return &wg, errChan, sigs
}
