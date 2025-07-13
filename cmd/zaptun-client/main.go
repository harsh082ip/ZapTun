package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/internal/client"
	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/rs/zerolog"
)

func main() {
	// Define CLI flag
	localPort := flag.Int("lp", 8000, "Local port to tunnel (e.g., 8000)")
	flag.Parse()

	// Load client config
	cfg, err := config.LoadClientConfig("")
	if err != nil {
		log.Fatalf("Failed to load client config: %v", err)
	}

	var logWriter io.Writer = os.Stdout
	logLevel := zerolog.InfoLevel
	appLogger := logger.NewLogger(logWriter, logLevel, "tunnel-server")

	// Start the client with CLI-provided local port
	srv, _ := client.NewClient(cfg.ServerAddr, *localPort, appLogger)
	appLogger.LogInfoMessage().Msg("Starting Zaptun client...")
	if err := srv.Start(); err != nil {
		appLogger.LogFatalMessage().Err(err).Msg("Client failed to start")
	}
}
