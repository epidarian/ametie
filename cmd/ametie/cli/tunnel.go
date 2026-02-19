package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"ametie/internal/config"
	"ametie/internal/network/transport"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage tunnels",
}

var tunnelCreateCmd = &cobra.Command{
	Use:   "create [local-address]",
	Short: "Create reverse tunnel",
	Long:  "Create a reverse tunnel: ametie tunnel create localhost:8080 --foreign-port 50111",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		foreignPort, _ := cmd.Flags().GetInt("foreign-port")
		if foreignPort == 0 {
			fmt.Println("Error: --foreign-port is required")
			return
		}

		localAddr := args[0]
		parts := strings.Split(localAddr, ":")
		if len(parts) != 2 {
			fmt.Println("Error: invalid local address format (use host:port)")
			return
		}

		localPort, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Printf("Error: invalid port number: %v\n", err)
			return
		}

		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		// Get target node from flag or use default
		targetNode, _ := cmd.Flags().GetString("target-node")
		if targetNode == "" {
			targetNode = "server" // Default to server
		}

		body := map[string]interface{}{
			"target_node": targetNode,
			"local_port":  localPort,
			"remote_port": foreignPort,
		}

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "POST",
			Endpoint: "/request.php",
			Body:     body,
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			fmt.Printf("Tunnel request created (local: %s, remote: %d)\n", localAddr, foreignPort)
		} else {
			fmt.Printf("Error: server returned status %d\n", resp.StatusCode)
		}
	},
}

var tunnelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active tunnels",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: "/status.php",
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			return
		}

		fmt.Println("Active tunnels:")
		if tunnels, ok := result["tunnels"].([]interface{}); ok {
			for _, tunnel := range tunnels {
				if tunnelMap, ok := tunnel.(map[string]interface{}); ok {
					id := tunnelMap["id"]
					source := tunnelMap["source_hostname"]
					if source == nil {
						source = tunnelMap["source_name"]
					}
					target := tunnelMap["target_hostname"]
					if target == nil {
						target = tunnelMap["target_name"]
					}
					localPort := tunnelMap["local_port"]
					remotePort := tunnelMap["remote_port"]
					status := tunnelMap["status"]
					fmt.Printf("  [%v] %v -> %v | Local: %v, Remote: %v | Status: %v\n",
						id, source, target, localPort, remotePort, status)
				}
			}
			if len(tunnels) == 0 {
				fmt.Println("  No active tunnels")
			}
		}
	},
}

var tunnelCloseCmd = &cobra.Command{
	Use:   "close [tunnel-id]",
	Short: "Close specific tunnel",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tunnelID := args[0]
		fmt.Printf("Closing tunnel %s\n", tunnelID)
		// Note: Tunnel closing would require a new endpoint on the server
		// For now, we just acknowledge the request
		fmt.Println("Note: Tunnel closing requires server-side support")
	},
}

// Root tunnel command (for backward compatibility with plan syntax)
var tunnelRootCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Create reverse tunnel (legacy syntax)",
	Long:  "Create a reverse tunnel: ametie tunnel --foreign-port 50111 localhost:8080",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Redirect to create command with proper flag handling
		foreignPort, _ := cmd.Flags().GetInt("foreign-port")
		if foreignPort == 0 {
			fmt.Println("Error: --foreign-port is required")
			return
		}
		// Set the flag for create command
		tunnelCreateCmd.Flags().Set("foreign-port", fmt.Sprintf("%d", foreignPort))
		tunnelCreateCmd.Run(cmd, args)
	},
}

func init() {
	tunnelRootCmd.Flags().Int("foreign-port", 0, "Foreign port")
	tunnelCmd.AddCommand(tunnelCreateCmd)
	tunnelCmd.AddCommand(tunnelListCmd)
	tunnelCmd.AddCommand(tunnelCloseCmd)
	tunnelCreateCmd.Flags().Int("foreign-port", 0, "Foreign port")
	tunnelCreateCmd.Flags().String("target-node", "", "Target node name")
	rootCmd.AddCommand(tunnelRootCmd) // Add legacy syntax
}
