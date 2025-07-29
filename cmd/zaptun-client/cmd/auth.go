package cmd

import (
	"fmt"
	"os"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth [token]",
	Short: "Saves your authtoken to the config file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := args[0]
		if err := config.WriteAuthToken(token); err != nil {
			fmt.Printf("Error saving auth token: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Auth token successfully saved.")
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
