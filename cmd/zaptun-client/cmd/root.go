package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	debug      bool
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "zaptun-client",
	Short: "Zaptun exposes your local ports to the internet.",
	Long:  `Zaptun creates a secure tunnel from a public URL to a service running on your local machine.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file")
}
