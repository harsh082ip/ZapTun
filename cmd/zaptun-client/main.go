package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/internal/client"
	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/harsh082ip/ZapTun/pkg/tunnel"
	"github.com/rs/zerolog"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: zaptun-client <http|tcp> <local_port>")
		os.Exit(1)
	}

	tunnelType := os.Args[1]
	if tunnelType != "http" && tunnelType != "tcp" {
		fmt.Println("Invalid tunnel type. Use 'http' or 'tcp'.")
		os.Exit(1)
	}
	// Define CLI flag
	localPort, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Invalid port number.")
		os.Exit(1)
	}

	// Load client config
	cfg, err := config.LoadClientConfig("")
	if err != nil {
		log.Fatalf("Failed to load client config: %v", err)
	}

	var logWriter io.Writer = os.Stdout
	logLevel := zerolog.InfoLevel
	appLogger := logger.NewLogger(logWriter, logLevel, "tunnel-server")

	controlMsg := &tunnel.ControlMessage{
		Type: tunnelType,
	}

	// Start the client with CLI-provided local port
	srv, _ := client.NewClient(cfg.ServerAddr, controlMsg, localPort, appLogger)
	appLogger.LogInfoMessage().Msgf("Starting Zaptun client for %s tunnel", tunnelType)
	if err := srv.Start(); err != nil {
		appLogger.LogFatalMessage().Err(err).Msg("Client failed to start")
	}
}
