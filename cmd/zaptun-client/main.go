package main

import (
	"io"
	"log"
	"os"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/internal/client"
	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.LoadClientConfig("")
	if err != nil {
		log.Fatalf("Failed to load client config: %v", err)
	}

	var logWriter io.Writer = os.Stdout

	logLevel := zerolog.InfoLevel

	appLogger := logger.NewLogger(logWriter, logLevel, "tunnel-server")

	// start the server
	srv, _ := client.NewClient(cfg.ServerAddr, 8080, appLogger)
	appLogger.LogInfoMessage().Msg("Starting Zaptun server...")
	if err := srv.Start(); err != nil {
		appLogger.LogFatalMessage().Err(err).Msg("Server failed to start")
	}
}
