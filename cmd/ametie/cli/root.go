package cli

import (
	"fmt"
	"os"

	"ametie/internal/auth"
	"ametie/internal/config"

	"github.com/spf13/cobra"
)

var (
	password string
	cfg      *config.ConfigManager
)

var rootCmd = &cobra.Command{
	Use:   "ametie",
	Short: "Ametie C2 Reverse Tunnel Service",
	Long:  "Ametie is a cross-platform service for command and control with reverse tunneling capabilities.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration
		var err error
		cfg, err = config.NewConfigManager()
		if err != nil {
			// Allow install command to run without config
			if cmd.Name() != "install" {
				fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Get password (skip for install command)
		if cmd.Name() == "install" {
			return
		}

		pwd, err := auth.GetPassword()
		if err != nil {
			// Prompt for password
			fmt.Print("Enter password: ")
			var pwdInput string
			fmt.Scanln(&pwdInput)
			password = pwdInput
			auth.SetPassword(password)
		} else {
			password = pwd
		}
	},
}

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(listNodesCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(tunnelRootCmd) // Legacy syntax
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(sendCommandCmd)
	rootCmd.AddCommand(commandsCmd)
	rootCmd.AddCommand(mailboxCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(composeCmd) // Root-level compose command
}
