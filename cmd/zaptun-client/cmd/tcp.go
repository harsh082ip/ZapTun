package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var tcpCmd = &cobra.Command{
	Use:   "tcp [local_port]",
	Short: "Starts a TCP tunnel to a running local port",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		localPort, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println("Invalid port number.")
			os.Exit(1)
		}
		// Call the shared function
		startTunnel("tcp", localPort)
	},
}

func init() {
	rootCmd.AddCommand(tcpCmd)
}
