package healthServer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gtriggiano/s3-webserver/pkg/utils"
)

func Start(config *HealthServerConfig) {
	defer config.WG.Done()

	gin.SetMode(gin.ReleaseMode)
	serverAddress := fmt.Sprintf("0.0.0.0:%d", config.ListenPort)
	router := gin.New()

	router.GET("/health", func(c *gin.Context) {
		err := config.HealthChecker.CheckHealth()

		if err == nil {
			c.String(http.StatusOK, "OK")
		} else {
			config.Logger.With("error", err).Error("S3 Healthcheck failed")
			c.String(http.StatusServiceUnavailable, "Service Unavailable")
		}
	})

	router.GET("/ready", func(c *gin.Context) {
		if utils.ACTIVE_SERVICES.Load() < 3 {
			c.String(http.StatusServiceUnavailable, "Service Unavailable")
			return
		}
		c.String(http.StatusOK, "OK")
	})

	server := &http.Server{
		Addr:    serverAddress,
		Handler: router,
	}

	go func() {
		time.AfterFunc(1*time.Second, func() {
			utils.ACTIVE_SERVICES.Add(1)
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.ErrChan <- utils.ExitError{Code: utils.EX_FAIL, Err: err}
		}
	}()

	config.Logger.Infof("Server starting at: %s", serverAddress)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	config.Logger.Infof("Got %v signal. HTTP Server will shut down in %v seconds", sig, config.ShutdownWaitSeconds)
	time.Sleep(time.Duration(config.ShutdownWaitSeconds) * time.Second)
	config.Logger.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(config.MainCtx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		config.Logger.Errorf("Server forced to shutdown: %v", err)
	}
	config.Logger.Info("Server has stopped!")
}
