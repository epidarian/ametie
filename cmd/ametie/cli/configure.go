package cli

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"

	"ametie/internal/auth"
	"ametie/internal/config"

	"github.com/keybase/go-keychain"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Update service configuration",
	Long:  "Update the service configuration with optional overrides.",
	Run: func(cmd *cobra.Command, args []string) {
		secureEndEnable, _ := cmd.Flags().GetBool("secure-end-enable")

		if secureEndEnable {
			// Store password in OS keychain
			fmt.Println("Secure endpoint mode enabled - password will be cached persistently")

			// Get password from session or prompt
			password, err := auth.GetPassword()
			if err != nil {
				fmt.Print("Enter password to store: ")
				var pwdInput string
				fmt.Scanln(&pwdInput)
				password = pwdInput
			}

			// Store in OS keychain
			switch runtime.GOOS {
			case "darwin":
				// macOS Keychain
				item := keychain.NewItem()
				item.SetSecClass(keychain.SecClassGenericPassword)
				item.SetService("ametie")
				item.SetAccount("cli_password")
				item.SetData([]byte(password))
				item.SetAccessible(keychain.AccessibleWhenUnlocked)
				err := keychain.AddItem(item)
				if err == keychain.ErrorDuplicateItem {
					// Update existing item
					keychain.DeleteItem(item)
					keychain.AddItem(item)
				}
				if err != nil {
					fmt.Printf("Error storing in keychain: %v\n", err)
				} else {
					fmt.Println("Password stored in macOS Keychain")
				}
			case "windows":
				// Windows Credential Manager
				fmt.Println("Windows credential storage requires additional implementation")
				fmt.Println("For now, use AMETIE_PASSWORD environment variable")
			case "linux":
				// Linux secret service
				fmt.Println("Linux secret service storage requires additional implementation")
				fmt.Println("For now, use AMETIE_PASSWORD environment variable")
			default:
				fmt.Println("Keychain storage not supported on this platform")
			}
		}

		fmt.Println("Configuration updated")
	},
}

var configureEndpointCmd = &cobra.Command{
	Use:   "endpoint <url>",
	Short: "Configure server endpoint",
	Long:  "Add or update a server endpoint with optional priority and mirror settings",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		newCluster, _ := cmd.Flags().GetBool("new-cluster")
		priority, _ := cmd.Flags().GetInt("priority")
		isMirror, _ := cmd.Flags().GetBool("mirror")

		cfg, err := config.NewConfigManager()
		if err != nil {
			fmt.Printf("Error: Failed to load configuration: %v\n", err)
			return
		}

		// If new-cluster flag is set, show warning and confirm
		if newCluster {
			fmt.Println("WARNING: You may be connecting to a different cluster.")
			fmt.Println("This will delete all current server data and endpoints.")
			fmt.Print("Continue? (y/n): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				fmt.Println("Cancelled.")
				return
			}

			// Clear all endpoints and server URL
			cfg.ClearEndpoints()
			fmt.Println("Cleared all existing endpoints and server data.")
		}

		// Check if this might be a different cluster
		existingEndpoints := cfg.GetEndpoints()
		if len(existingEndpoints) > 0 && !newCluster {
			// Try to detect if this is a different cluster by checking URL domain
			existingDomain := extractDomain(cfg.Get().ServerURL)
			newDomain := extractDomain(url)

			if existingDomain != "" && newDomain != "" && existingDomain != newDomain {
				fmt.Println("WARNING: You may be connecting to a different cluster.")
				fmt.Printf("Current endpoint domain: %s\n", existingDomain)
				fmt.Printf("New endpoint domain: %s\n", newDomain)
				fmt.Print("Continue? (y/n): ")

				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response != "y" && response != "yes" {
					fmt.Println("Cancelled.")
					return
				}
			}
		}

		// Default priority if not specified
		if priority == 0 {
			if isMirror {
				priority = 10 // Mirrors have lower priority
			} else {
				priority = 1 // Primary endpoints have high priority
			}
		}

		// Extract cluster identifier from URL domain
		cluster := extractDomain(url)

		// Add endpoint
		err = cfg.AddEndpoint(url, priority, isMirror, cluster)
		if err != nil {
			fmt.Printf("Error: Failed to add endpoint: %v\n", err)
			return
		}

		// If this is the first endpoint or highest priority non-mirror, set as primary
		existingEndpoints = cfg.GetEndpoints()
		if !isMirror {
			shouldSetPrimary := cfg.Get().ServerURL == ""
			if !shouldSetPrimary {
				// Check if this has higher priority than current primary
				currentPrimary := cfg.Get().ServerURL
				for _, ep := range existingEndpoints {
					if ep.URL == currentPrimary && !ep.IsMirror {
						if priority < ep.Priority {
							shouldSetPrimary = true
						}
						break
					}
				}
			}
			if shouldSetPrimary {
				cfg.SetServerURL(url)
			}
		}

		// Save configuration
		err = cfg.Save()
		if err != nil {
			fmt.Printf("Error: Failed to save configuration: %v\n", err)
			return
		}

		endpointType := "mirror"
		if !isMirror {
			endpointType = "primary"
		}
		fmt.Printf("Endpoint configured: %s (priority: %d, type: %s)\n", url, priority, endpointType)
	},
}

// extractDomain extracts the domain from a URL
func extractDomain(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// Fallback to simple string parsing
		if strings.HasPrefix(urlStr, "http://") {
			urlStr = urlStr[7:]
		} else if strings.HasPrefix(urlStr, "https://") {
			urlStr = urlStr[8:]
		}

		// Remove path and port
		if idx := strings.Index(urlStr, "/"); idx != -1 {
			urlStr = urlStr[:idx]
		}
		if idx := strings.Index(urlStr, ":"); idx != -1 {
			urlStr = urlStr[:idx]
		}
		return urlStr
	}

	host := parsedURL.Hostname()
	if host == "" {
		return parsedURL.Host
	}
	return host
}

func init() {
	configureCmd.Flags().Bool("secure-end-enable", false, "Enable secure endpoint mode")
	configureCmd.Flags().String("optional_overrides", "", "Optional configuration overrides")

	configureCmd.AddCommand(configureEndpointCmd)
	configureEndpointCmd.Flags().Bool("new-cluster", false, "Delete all current server data (new cluster)")
	configureEndpointCmd.Flags().Int("priority", 0, "Endpoint priority (lower = higher priority, default: 1 for primary, 10 for mirror)")
	configureEndpointCmd.Flags().Bool("mirror", false, "Mark endpoint as mirror (backup)")
}
