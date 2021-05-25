package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gtriggiano/s3-webserver/pkg/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	processStartTime := time.Now()
	config := util.LoadConfig()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.ForwardedByClientIP = config.App.TrustProxy

	if config.App.LogHTTPRequests {
		router.Use(util.JSONLogMiddleware())
	}

	router.Use(gin.Recovery())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(cors.Default())

	router.GET("/*path", util.MainHandler(config))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.App.HTTPPort),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); errors.Is(err, http.ErrServerClosed) == false {
			util.
				LogWithHostname(log.Err(err)).
				Msg(fmt.Sprintf("Could not start the server on %s", server.Addr))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	signal := <-quit

	util.
		LogWithHostname(log.Info()).
		Msg(fmt.Sprintf("Received %s signal. Shutting down server", signal.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var finalLog *zerolog.Event

	if err := server.Shutdown(ctx); err != nil {
		finalLog = log.Err(err)
	} else {
		finalLog = log.Info()
	}

	util.
		LogWithHostname(finalLog).
		Dur("processLifetime", time.Since(processStartTime)).
		Msg("Exit")
}
