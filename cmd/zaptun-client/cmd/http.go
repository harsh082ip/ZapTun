package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/internal/client"
	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/harsh082ip/ZapTun/pkg/tunnel"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var httpCmd = &cobra.Command{
	Use:   "http [local_port]",
	Short: "Starts an HTTP tunnel",
	Args:  cobra.ExactArgs(1),
	Run:   runTunnel("http"),
}

func init() {
	rootCmd.AddCommand(httpCmd)
}

// runTunnel is a helper function to avoid duplicating code for http and tcp commands
func runTunnel(tunnelType string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		localPort, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println("Invalid port number.")
			os.Exit(1)
		}

		clientCfg, err := config.LoadClientConfig()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		logLevel := zerolog.Disabled
		if debug {
			logLevel = zerolog.DebugLevel
		}

		var logWriter io.Writer = os.Stdout
		appLogger := logger.NewLogger(logWriter, logLevel, "zaptun-client")

		controlMsg := &tunnel.ControlMessage{Type: tunnelType}

		srv, _ := client.NewClient(controlMsg, clientCfg, localPort, appLogger)
		appLogger.LogInfoMessage().Msgf("Starting Zaptun client for %s tunnel", tunnelType)

		if err := srv.Start(logLevel); err != nil {
			appLogger.LogFatalMessage().Err(err).Msg("Client failed to start")
		}
	}

}
