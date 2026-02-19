package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"ametie/internal/config"
	"ametie/internal/network/transport"

	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect ssh [pc_name]",
	Short: "Establish SSH connection to node",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if args[0] != "ssh" {
			fmt.Println("Error: only 'ssh' connections are supported")
			return
		}

		pcName := args[1]
		fmt.Printf("Connecting to %s via SSH...\n", pcName)

		// Get node connection info from server
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		// First, get tunnel info for this node
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

		// Find tunnel for this node
		if tunnels, ok := result["tunnels"].([]interface{}); ok {
			for _, tunnel := range tunnels {
				if tunnelMap, ok := tunnel.(map[string]interface{}); ok {
					target := tunnelMap["target_hostname"]
					if target == nil {
						target = tunnelMap["target_name"]
					}
					if fmt.Sprintf("%v", target) == pcName {
						remotePort := tunnelMap["remote_port"]
						fmt.Printf("Found tunnel: Use 'ssh -p %v localhost' to connect\n", remotePort)
						return
					}
				}
			}
		}

		fmt.Printf("No active tunnel found for node %s\n", pcName)
		fmt.Println("Create a tunnel first using: ametie tunnel create localhost:22 --foreign-port <port>")
	},
}

var sendCommandCmd = &cobra.Command{
	Use:   "send-command [pc_name] [command]",
	Short: "Queue command for execution",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		pcName := args[0]
		command := strings.Join(args[1:], " ")

		fileFlag, _ := cmd.Flags().GetString("file")
		if fileFlag != "" {
			data, err := os.ReadFile(fileFlag)
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				return
			}
			command = string(data)
		}

		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		body := map[string]interface{}{
			"target_node": pcName,
			"command":     command,
		}

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "POST",
			Endpoint: "/submit.php",
			Body:     body,
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		fmt.Printf("Command queued for %s\n", pcName)
	},
}

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "Manage commands",
}

var commandsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending commands",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: "/fetch.php",
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

		fmt.Println("Pending commands:")
		if commands, ok := result["commands"].([]interface{}); ok {
			for _, cmd := range commands {
				if cmdMap, ok := cmd.(map[string]interface{}); ok {
					id := cmdMap["id"]
					command := cmdMap["command"]
					created := cmdMap["created_at"]
					fmt.Printf("  [%v] %v (created: %v)\n", id, command, created)
				}
			}
			if len(commands) == 0 {
				fmt.Println("  No pending commands")
			}
		}
	},
}

var commandsCancelCmd = &cobra.Command{
	Use:   "cancel [command-id]",
	Short: "Cancel pending command",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commandID := args[0]
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		// Parse command ID as integer
		cmdID, err := strconv.Atoi(commandID)
		if err != nil {
			fmt.Printf("Error: invalid command ID: %v\n", err)
			return
		}

		body := map[string]interface{}{
			"command_id": cmdID,
		}

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "POST",
			Endpoint: "/cancel.php",
			Body:     body,
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			return
		}

		if resp.StatusCode == 200 {
			if msg, ok := result["message"].(string); ok {
				fmt.Printf("%s\n", msg)
			} else {
				fmt.Printf("Command %s cancelled\n", commandID)
			}
		} else {
			if errMsg, ok := result["error"].(string); ok {
				fmt.Printf("Error: %s\n", errMsg)
			} else {
				fmt.Printf("Error: server returned status %d\n", resp.StatusCode)
			}
		}
	},
}

func init() {
	commandsCmd.AddCommand(commandsListCmd)
	commandsCmd.AddCommand(commandsCancelCmd)
	sendCommandCmd.Flags().String("file", "", "Read command from file")
}
