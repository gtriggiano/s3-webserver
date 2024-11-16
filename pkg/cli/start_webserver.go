package cli

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/gtriggiano/s3-webserver/pkg/utils"
	"github.com/spf13/cobra"
)

type StartFlagsValues struct {
	WebServerPort       int
	HealthServerPort    int
	MetricsServerPort   int
	ShutdownWaitSeconds int
	TrustProxy          bool
	ConfigFile          string
	TLSCertPath         string
	TLSKeyPath          string
}

func StartCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "start [FLAGS]",
		Short: "Run the S3 webserver",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := utils.NewLogger().With("app", "s3-webserver")
			defer func() {
				_ = logger.Sync()
			}()

			mainCtx, stop := signal.NotifyContext(
				context.Background(),
				syscall.SIGINT,
				syscall.SIGTERM,
			)
			defer stop()

			startFlags, err := GetStartFlagsValues(cmd)
			if err != nil {
				return err
			}

			config := StartConfig{
				WebServerPort:        startFlags.WebServerPort,
				HealthServerPort:     startFlags.HealthServerPort,
				MetricsServerPort:    startFlags.MetricsServerPort,
				ShutdownWaitSeconds:  startFlags.ShutdownWaitSeconds,
				WebserverTrustProxy:  startFlags.TrustProxy,
				S3ProxyConfigFile:    startFlags.ConfigFile,
				Context:              mainCtx,
				Logger:               logger,
				WebserverTLSCertPath: startFlags.TLSCertPath,
				WebserverTLSKeyPath:  startFlags.TLSKeyPath,
			}

			wg, errChan, sigs := start(config)

			select {
			case err := <-errChan:
				return err
			case sig := <-sigs:
				utils.ACTIVE_SERVICES.Store(0) // Start to fail readiness probes

				logger.Infow(fmt.Sprintf("Got interrupt signal, beginning shutdown %s", sig))
				wg.Wait()
				logger.Infow("All goroutines finished, shutting down!")
				return nil
			}

		},
	}

	cmd.PersistentFlags().Int("shutdown-wait-seconds", 5, "Time to wait between interrupt signal and services shutdown")
	cmd.PersistentFlags().Bool("trust-proxy", false, "Wether to trust reverse proxy headers")
	cmd.PersistentFlags().Int("health-port", 3000, "The port the health server binds to.")
	cmd.PersistentFlags().Int("metrics-port", 9090, "The port the metrics server binds to.")
	cmd.PersistentFlags().Int("server-port", 8080, "The port the authentication server binds to.")
	cmd.PersistentFlags().String("config", "", "Path to configuration file")
	cmd.PersistentFlags().String("tls-cert-path", "", "Path to the TLS server certificate.")
	cmd.PersistentFlags().String("tls-key-path", "", "Path to the TLS server key.")

	return &cmd
}

func GetStartFlagsValues(cmd *cobra.Command) (*StartFlagsValues, error) {

	healthPort, err := cmd.Flags().GetInt("health-port")
	if err != nil {
		return nil, err
	}
	metricsPort, err := cmd.Flags().GetInt("metrics-port")
	if err != nil {
		return nil, err
	}
	serverPort, err := cmd.Flags().GetInt("server-port")
	if err != nil {
		return nil, err
	}
	shutdownWaitSeconds, err := cmd.Flags().GetInt("shutdown-wait-seconds")
	if err != nil {
		return nil, err
	}
	trustProxy, err := cmd.Flags().GetBool("trust-proxy")
	if err != nil {
		return nil, err
	}
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, err
	}

	tlsCertPath, err := cmd.Flags().GetString("tls-cert-path")
	if err != nil {
		return nil, err
	}
	tlsKeyPath, err := cmd.Flags().GetString("tls-key-path")
	if err != nil {
		return nil, err
	}

	return &StartFlagsValues{
		WebServerPort:       serverPort,
		HealthServerPort:    healthPort,
		MetricsServerPort:   metricsPort,
		ShutdownWaitSeconds: shutdownWaitSeconds,
		TrustProxy:          trustProxy,
		ConfigFile:          configFile,
		TLSCertPath:         tlsCertPath,
		TLSKeyPath:          tlsKeyPath,
	}, nil
}
