package main

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gtriggiano/s3-webserver/pkg/util"
)

func main() {
	config := util.LoadConfig()
	s3Handler := util.S3Handler(config)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.ForwardedByClientIP = config.App.TrustProxy

	if config.App.LogHTTPRequests {
		router.Use(util.HTTPLogMiddleware())
	}

	router.Use(gin.Recovery())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(cors.Default())

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "")
	})
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method == "GET" {
			s3Handler(c)
		}
	})

	router.Run(fmt.Sprintf(":%d", config.App.HTTPPort))

	/*
		We neet to wait for https://github.com/gin-gonic/gin/pull/2692/files landing
		in a release in order to use a graceful shutdown approach
	*/

	// server := &http.Server{
	// 	Addr:    fmt.Sprintf(":%d", config.App.HTTPPort),
	// 	Handler: router,
	// }

	// go func() {
	// 	if err := server.ListenAndServe(); errors.Is(err, http.ErrServerClosed) == false {
	// 		util.
	// 			LogWithHostname(log.Err(err)).
	// 			Msg(fmt.Sprintf("Could not start the server on %s", server.Addr))
	// 		os.Exit(1)
	// 	}
	// }()

	// quit := make(chan os.Signal)
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// signal := <-quit

	// util.
	// 	LogWithHostname(log.Info()).
	// 	Msg(fmt.Sprintf("Received %s signal. Shutting down server", signal.String()))

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// var finalLog *zerolog.Event

	// if err := server.Shutdown(ctx); err != nil {
	// 	finalLog = log.Err(err)
	// } else {
	// 	finalLog = log.Info()
	// }

	// util.
	// 	LogWithHostname(finalLog).
	// 	Dur("processLifetime", time.Since(processStartTime)).
	// 	Msg("Exit")
}
