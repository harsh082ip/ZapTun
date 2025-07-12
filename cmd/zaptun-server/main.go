package main

import (
	"io"
	"log"
	"os"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/internal/server"
	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.LoadServerConfig("")
	if err != nil {
		log.Fatalf("Failed to load server config: %v", err)
	}

	var logWriter io.Writer = os.Stdout
	if cfg.LogFile != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %v", cfg.LogFile, err)
		}
		logWriter = file
		defer file.Close()
	}

	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	appLogger := logger.NewLogger(logWriter, logLevel, "tunnel-server")

	// start the server
	srv := server.NewServer(cfg, appLogger)
	appLogger.LogInfoMessage().Msg("Starting Zaptun server...")
	if err := srv.Start(); err != nil {
		appLogger.LogFatalMessage().Err(err).Msg("Server failed to start")
	}
}
