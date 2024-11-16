package cli

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gtriggiano/s3-webserver/pkg/healthServer"
	"github.com/gtriggiano/s3-webserver/pkg/metricsServer"
	"github.com/gtriggiano/s3-webserver/pkg/s3Proxy"
	"github.com/gtriggiano/s3-webserver/pkg/webServer"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Proxy interface {
	healthServer.HealthChecker
	webServer.Proxy
}

type StartConfig struct {
	WebServerPort        int
	WebserverTrustProxy  bool
	WebserverTLSCertPath string
	WebserverTLSKeyPath  string
	HealthServerPort     int
	MetricsServerPort    int
	ShutdownWaitSeconds  int
	S3ProxyConfigFile    string
	Context              context.Context
	Logger               *zap.SugaredLogger
}

func start(config StartConfig) (*sync.WaitGroup, <-chan error, <-chan os.Signal) {
	var wg sync.WaitGroup
	errChan := make(chan error, 10)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	registry := prometheus.NewRegistry()

	proxy, err := s3Proxy.NewS3Proxy(config.S3ProxyConfigFile, config.Logger.With("component", "s3-proxy"), registry)
	if err != nil {
		errChan <- err
		return &wg, errChan, sigs
	}

	wg.Add(1)
	go webServer.Start(&webServer.WebserverConfig{
		ErrChan:    errChan,
		ListenPort: config.WebServerPort,
		Logger:     config.Logger.With("component", "web-server"),
		MainCtx:    config.Context,
		WG:         &wg,

		Proxy:               proxy,
		Registry:            registry,
		ShutdownWaitSeconds: config.ShutdownWaitSeconds,
		TrustProxy:          config.WebserverTrustProxy,
		TLSCertPath:         config.WebserverTLSCertPath,
		TLSKeyPath:          config.WebserverTLSKeyPath,
	})

	wg.Add(1)
	go healthServer.Start(&healthServer.HealthServerConfig{
		ErrChan:    errChan,
		ListenPort: config.HealthServerPort,
		Logger:     config.Logger.With("component", "health-server"),
		MainCtx:    config.Context,
		WG:         &wg,

		HealthChecker: proxy,
	})

	wg.Add(1)
	go metricsServer.Start(&metricsServer.MetricsServerConfig{
		ErrChan:    errChan,
		ListenPort: config.MetricsServerPort,
		Logger:     config.Logger.With("component", "metrics-server"),
		MainCtx:    config.Context,
		WG:         &wg,

		Registry: registry,
	})

	return &wg, errChan, sigs
}
