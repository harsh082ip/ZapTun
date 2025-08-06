package cmd

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve [directory]",
	Short: "Serves a local directory over an HTTP tunnel",
	Long: `Starts a local file server for the specified directory (or the current directory if none is provided) 
and exposes it to the internet through a Zaptun HTTP tunnel.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}
		if len(args) > 0 {
			dir = args[0]
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("Error: Directory '%s' does not exist.\n", dir)
			os.Exit(1)
		}

		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatalf("Failed to find a free port: %v", err)
		}
		localPort := listener.Addr().(*net.TCPAddr).Port

		fmt.Printf("Serving directory '%s' on local port %d...\n", dir, localPort)

		go func() {
			// Create a file server handler for the specified directory.
			fileServer := http.FileServer(http.Dir(dir))

			if err := http.Serve(listener, fileServer); err != nil {
				log.Fatalf("Local file server failed: %v", err)
			}
		}()

		// 4. Start an HTTP tunnel to the port our file server is running on.
		// This reuses the exact same logic as the 'zaptun-client http' command.
		startTunnel("http", localPort)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
