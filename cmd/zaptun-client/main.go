package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/internal/client"
	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/harsh082ip/ZapTun/pkg/tunnel"
	"github.com/rs/zerolog"
)

type Config struct {
	Debug      bool
	Verbose    bool
	ConfigPath string
	// Add more flags here as needed
}

func parseArgs() (tunnelType string, localPort int, config Config) {
	var filteredArgs []string
	config = Config{}

	args := os.Args[1:] // Get all arguments except program name

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--debug" || arg == "-d":
			config.Debug = true
		case arg == "--verbose" || arg == "-v":
			config.Verbose = true
		case strings.HasPrefix(arg, "--config="):
			configPath := strings.TrimPrefix(arg, "--config=")
			if configPath == "" {
				fmt.Println("Error: --config flag requires a value, but got nothing")
				os.Exit(1)
			}
			config.ConfigPath = configPath
		case arg == "--config" || arg == "-c":
			// Handle --config <value> format
			if i+1 < len(args) {
				nextArg := args[i+1]
				// Check if the next argument is actually a flag, not a config value
				if strings.HasPrefix(nextArg, "-") {
					fmt.Printf("Error: --config flag requires a value, but got flag '%s'\n", nextArg)
					os.Exit(1)
				}
				config.ConfigPath = nextArg
				i++ // Skip the next argument since we consumed it
			} else {
				fmt.Println("Error: --config flag requires a value")
				os.Exit(1)
			}
		case strings.HasPrefix(arg, "-"):
			fmt.Printf("Unknown flag: %s\n", arg)
			os.Exit(1)
		default:
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) < 2 {
		fmt.Println("Usage: zaptun-client <http|tcp> <local_port> [flags]")
		fmt.Println("Flags:")
		fmt.Println("  --debug, -d          Enable debug logging")
		fmt.Println("  --verbose, -v        Enable verbose output")
		fmt.Println("  --config, -c <path>  Config file path")
		os.Exit(1)
	}

	tunnelType = filteredArgs[0]
	if tunnelType != "http" && tunnelType != "tcp" {
		fmt.Println("Invalid tunnel type. Use 'http' or 'tcp'.")
		os.Exit(1)
	}

	var err error
	localPort, err = strconv.Atoi(filteredArgs[1])
	if err != nil {
		fmt.Println("Invalid port number.")
		os.Exit(1)
	}

	return tunnelType, localPort, config
}

func main() {
	tunnelType, localPort, cfg := parseArgs()

	// Load client config
	configPath := cfg.ConfigPath
	if configPath == "" {
		configPath = "" // Use default
	}
	clientCfg, err := config.LoadClientConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load client config: %v", err)
	}

	// Setup logging based on flags
	var logWriter io.Writer = os.Stdout
	logLevel := zerolog.Disabled

	if cfg.Debug {
		logLevel = zerolog.DebugLevel
	} else if cfg.Verbose {
		logLevel = zerolog.InfoLevel
	}

	appLogger := logger.NewLogger(logWriter, logLevel, "tunnel-client")

	// Log configuration if verbose
	if cfg.Verbose {
		appLogger.LogInfoMessage().
			Str("tunnel_type", tunnelType).
			Int("local_port", localPort).
			Bool("debug", cfg.Debug).
			Msg("Configuration loaded")
	}

	controlMsg := &tunnel.ControlMessage{
		Type: tunnelType,
	}

	// Start the client with CLI-provided local port
	srv, _ := client.NewClient(clientCfg.ServerAddr, controlMsg, localPort, appLogger)
	appLogger.LogInfoMessage().Msgf("Starting Zaptun client for %s tunnel", tunnelType)

	if err := srv.Start(); err != nil {
		appLogger.LogFatalMessage().Err(err).Msg("Client failed to start")
	}
}
