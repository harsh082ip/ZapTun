package cmd

import (
	"github.com/spf13/cobra"
)

var tcpCmd = &cobra.Command{
	Use:   "tcp [local_port]",
	Short: "Starts a TCP tunnel",
	Args:  cobra.ExactArgs(1),
	Run:   runTunnel("tcp"),
}

func init() {
	rootCmd.AddCommand(tcpCmd)
}
