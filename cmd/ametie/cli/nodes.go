package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"ametie/internal/config"
	"ametie/internal/network/transport"

	"github.com/spf13/cobra"
)

var listNodesCmd = &cobra.Command{
	Use:   "list nodes",
	Short: "List all registered nodes",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: "/list.php",
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if nodes, ok := result["nodes"].([]interface{}); ok {
			fmt.Println("Registered Nodes:")
			for _, node := range nodes {
				if nodeMap, ok := node.(map[string]interface{}); ok {
					hostname := nodeMap["hostname"]
					customName := nodeMap["custom_name"]
					status := nodeMap["status"]
					lastHeartbeat := nodeMap["last_heartbeat"]
					fmt.Printf("  %s (%s) - %s - Last seen: %v\n", customName, hostname, status, lastHeartbeat)
				}
			}
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show local node status",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		fmt.Printf("Node Name: %s\n", conf.NodeName)
		fmt.Printf("Hostname: %s\n", conf.Hostname)
		fmt.Printf("Server URL: %s\n", conf.ServerURL)
		fmt.Printf("Node ID: %s\n", cfg.GetNodeIDHash())
	},
}

var renameCmd = &cobra.Command{
	Use:   "rename [new-name]",
	Short: "Rename local node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.NewConfigManager()
		cfg.SetNodeName(args[0])
		cfg.Save()
		fmt.Printf("Node renamed to: %s\n", args[0])
	},
}
