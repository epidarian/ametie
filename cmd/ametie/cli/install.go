package cli

import (
	"fmt"
	"os"

	"ametie/internal/config"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and configure the Ametie service",
	Long:  "Interactive installation script that configures the service and installs it as a system service.",
	Run: func(cmd *cobra.Command, args []string) {
		apiKey, _ := cmd.Flags().GetString("api-key")
		serverURL, _ := cmd.Flags().GetString("server-url")
		nodeName, _ := cmd.Flags().GetString("node-name")

		cfg, err := config.NewConfigManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize config: %v\n", err)
			os.Exit(1)
		}

		// Prompt for missing values
		if apiKey == "" {
			fmt.Print("Enter API key: ")
			fmt.Scanln(&apiKey)
		}

		if serverURL == "" {
			fmt.Print("Enter server URL: ")
			fmt.Scanln(&serverURL)
		}

		if nodeName == "" {
			hostname, _ := os.Hostname()
			fmt.Printf("Enter node name (default: %s): ", hostname)
			fmt.Scanln(&nodeName)
			if nodeName == "" {
				nodeName = hostname
			}
		}

		// Set configuration
		cfg.SetAPIKey(apiKey)
		cfg.SetServerURL(serverURL)
		cfg.SetNodeName(nodeName)

		hostname, _ := os.Hostname()
		cfg.SetHostname(hostname)

		// Validate
		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
			os.Exit(1)
		}

		// Save
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Configuration saved successfully!")
		fmt.Println("Run 'ametie service install' to install as a system service.")
	},
}

func init() {
	installCmd.Flags().String("api-key", "", "API key")
	installCmd.Flags().String("server-url", "", "Server URL")
	installCmd.Flags().String("node-name", "", "Node name")
}
