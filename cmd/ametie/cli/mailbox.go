package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"ametie/internal/config"
	"ametie/internal/network/transport"

	"github.com/spf13/cobra"
)

var mailboxCmd = &cobra.Command{
	Use:   "mailbox",
	Short: "Manage mailbox (command output, messages, and notifications)",
	Long:  "Unified mailbox system for command output, node-to-node messages, and general notifications",
}

var composeCmd = &cobra.Command{
	Use:   "compose [message]",
	Short: "Send message to mailbox or specific node",
	Long:  "Send a message: ametie compose \"message\" (to general mailbox) or ametie compose \"message\" --host hostname (to specific node)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		message := args[0]
		host, _ := cmd.Flags().GetString("host")

		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		var body map[string]interface{}
		if host == "" {
			// Send to general mailbox
			body = map[string]interface{}{
				"message_type": "notification",
				"content":      message,
			}
		} else {
			// Send to specific node
			body = map[string]interface{}{
				"message_type": "message",
				"to_node":      host,
				"message":      message,
			}
		}

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "POST",
			Endpoint: "/messages.php",
			Body:     body,
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			if host == "" {
				fmt.Println("Message sent to general mailbox")
			} else {
				fmt.Printf("Message sent to %s\n", host)
			}
		} else {
			fmt.Printf("Error: server returned status %d\n", resp.StatusCode)
		}
	},
}

var mailboxCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check mailbox for new messages/output/notifications",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		// Check mailbox entries
		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: "/messages.php?type=mailbox",
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

		entryCount := 0
		if entries, ok := result["entries"].([]interface{}); ok {
			entryCount = len(entries)
		}

		// Check messages
		resp2, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: "/messages.php?type=messages&unread=1",
		})

		messageCount := 0
		if err == nil {
			defer resp2.Body.Close()
			body2, _ := io.ReadAll(resp2.Body)
			var result2 map[string]interface{}
			if err := json.Unmarshal(body2, &result2); err == nil {
				if messages, ok := result2["messages"].([]interface{}); ok {
					messageCount = len(messages)
				}
			}
		}

		total := entryCount + messageCount
		if total == 0 {
			fmt.Println("No new messages in mailbox")
			return
		}

		fmt.Printf("Found %d new items in mailbox (%d entries, %d messages):\n", total, entryCount, messageCount)

		// Show mailbox entries
		if entries, ok := result["entries"].([]interface{}); ok && len(entries) > 0 {
			for i, entry := range entries {
				if i >= 5 {
					fmt.Printf("... and %d more entries\n", len(entries)-5)
					break
				}
				if entryMap, ok := entry.(map[string]interface{}); ok {
					id := entryMap["id"]
					msgType := entryMap["message_type"]
					created := entryMap["created_at"]
					content := entryMap["content"]
					contentStr := fmt.Sprintf("%v", content)
					if len(contentStr) > 50 {
						contentStr = contentStr[:50] + "..."
					}
					fmt.Printf("  [%v] %v - %v: %s\n", id, created, msgType, contentStr)
				}
			}
		}
	},
}

var mailboxReadCmd = &cobra.Command{
	Use:   "read [message-id]",
	Short: "Read specific message",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		messageID := args[0]
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: fmt.Sprintf("/messages.php?type=mailbox&command_id=%s", messageID),
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

		if entries, ok := result["entries"].([]interface{}); ok && len(entries) > 0 {
			entry := entries[0].(map[string]interface{})
			fmt.Printf("Message ID: %v\n", entry["id"])
			fmt.Printf("Type: %v\n", entry["message_type"])
			fmt.Printf("Created: %v\n", entry["created_at"])
			fmt.Printf("\nContent:\n%s\n", entry["content"])
		} else {
			fmt.Printf("Message %s not found\n", messageID)
		}
	},
}

var mailboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List mailbox entries (command output, messages, notifications)",
	Run: func(cmd *cobra.Command, args []string) {
		node, _ := cmd.Flags().GetString("node")
		unread, _ := cmd.Flags().GetBool("unread")
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		// Get both mailbox entries and messages
		endpoint := "/messages.php?type=mailbox"
		if node != "" {
			endpoint += "&node=" + node
		}

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: endpoint,
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

		fmt.Printf("Mailbox entries")
		if node != "" {
			fmt.Printf(" for node %s", node)
		}
		if unread {
			fmt.Print(" (unread only)")
		}
		fmt.Println(":")

		// Display mailbox entries
		if entries, ok := result["entries"].([]interface{}); ok {
			for _, entry := range entries {
				if entryMap, ok := entry.(map[string]interface{}); ok {
					id := entryMap["id"]
					cmdID := entryMap["command_id"]
					msgType := entryMap["message_type"]
					created := entryMap["created_at"]
					content := entryMap["content"]
					contentStr := fmt.Sprintf("%v", content)
					if len(contentStr) > 40 {
						contentStr = contentStr[:40] + "..."
					}
					fmt.Printf("  [%v] Type: %v | Command: %v | Created: %v\n", id, msgType, cmdID, created)
					fmt.Printf("      Content: %s\n", contentStr)
				}
			}
		}

		// Also get messages
		endpoint = "/messages.php?type=messages"
		if unread {
			endpoint += "&unread=1"
		}

		resp2, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: endpoint,
		})

		if err == nil {
			defer resp2.Body.Close()
			body2, _ := io.ReadAll(resp2.Body)
			var result2 map[string]interface{}
			if err := json.Unmarshal(body2, &result2); err == nil {
				if messages, ok := result2["messages"].([]interface{}); ok && len(messages) > 0 {
					fmt.Println("\nMessages:")
					for _, msg := range messages {
						if msgMap, ok := msg.(map[string]interface{}); ok {
							id := msgMap["id"]
							from := msgMap["from_hostname"]
							if from == nil {
								from = msgMap["from_name"]
							}
							isRead := msgMap["is_read"]
							created := msgMap["created_at"]
							message := msgMap["message"]
							msgStr := fmt.Sprintf("%v", message)
							if len(msgStr) > 40 {
								msgStr = msgStr[:40] + "..."
							}
							readStatus := "✓"
							if !isRead.(bool) {
								readStatus = "✗"
							}
							fmt.Printf("  [%s] ID: %v | From: %v | Created: %v\n", readStatus, id, from, created)
							fmt.Printf("      Message: %s\n", msgStr)
						}
					}
				}
			}
		}

		// Check if we have any entries at all
		entries, _ := result["entries"].([]interface{})
		if len(entries) == 0 {
			fmt.Println("  No mailbox entries found")
		}
	},
}

var mailboxClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear mailbox entries and messages",
	Run: func(cmd *cobra.Command, args []string) {
		node, _ := cmd.Flags().GetString("node")
		olderThan, _ := cmd.Flags().GetInt("older-than")
		read, _ := cmd.Flags().GetBool("read")
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		// Clear mailbox entries
		endpoint := "/messages.php?type=mailbox"
		if node != "" {
			endpoint += "&node=" + node
		}
		if olderThan > 0 {
			endpoint += "&older_than=" + strconv.Itoa(olderThan)
		}

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "DELETE",
			Endpoint: endpoint,
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		// Clear messages
		endpoint2 := "/messages.php?type=messages"
		if read {
			endpoint2 += "&read=1"
		}

		resp2, err2 := client.MakeRequest(transport.RequestOptions{
			Method:   "DELETE",
			Endpoint: endpoint2,
		})

		if err2 == nil {
			defer resp2.Body.Close()
		}

		if resp.StatusCode == 200 {
			fmt.Printf("Mailbox cleared")
			if node != "" {
				fmt.Printf(" for node %s", node)
			}
			if olderThan > 0 {
				fmt.Printf(" (older than %d days)", olderThan)
			}
			if read {
				fmt.Print(" (read messages only)")
			}
			fmt.Println()
		} else {
			fmt.Printf("Error: server returned status %d\n", resp.StatusCode)
		}
	},
}

var mailboxExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export mailbox data",
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")
		cfg, _ := config.NewConfigManager()
		conf := cfg.Get()
		nodeID := cfg.GetNodeIDHash()

		client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

		resp, err := client.MakeRequest(transport.RequestOptions{
			Method:   "GET",
			Endpoint: "/messages.php?type=mailbox",
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

		entries, ok := result["entries"].([]interface{})
		if !ok {
			fmt.Println("No entries to export")
			return
		}

		switch format {
		case "json":
			jsonData, _ := json.MarshalIndent(entries, "", "  ")
			fmt.Println(string(jsonData))
		case "csv":
			writer := csv.NewWriter(os.Stdout)
			writer.Write([]string{"ID", "Command ID", "Type", "Content", "Created"})
			for _, entry := range entries {
				if entryMap, ok := entry.(map[string]interface{}); ok {
					writer.Write([]string{
						fmt.Sprintf("%v", entryMap["id"]),
						fmt.Sprintf("%v", entryMap["command_id"]),
						fmt.Sprintf("%v", entryMap["message_type"]),
						fmt.Sprintf("%v", entryMap["content"]),
						fmt.Sprintf("%v", entryMap["created_at"]),
					})
				}
			}
			writer.Flush()
		default:
			fmt.Printf("Unsupported format: %s (use json or csv)\n", format)
		}
	},
}

func init() {
	mailboxCmd.AddCommand(mailboxCheckCmd)
	mailboxCmd.AddCommand(mailboxReadCmd)
	mailboxCmd.AddCommand(mailboxListCmd)
	mailboxCmd.AddCommand(mailboxClearCmd)
	mailboxCmd.AddCommand(mailboxExportCmd)
	mailboxListCmd.Flags().String("node", "", "Filter by node")
	mailboxListCmd.Flags().Bool("unread", false, "Show only unread messages")
	mailboxClearCmd.Flags().String("node", "", "Filter by node")
	mailboxClearCmd.Flags().Int("older-than", 0, "Clear entries older than N days")
	mailboxClearCmd.Flags().Bool("read", false, "Clear only read messages")
	mailboxExportCmd.Flags().String("format", "json", "Export format (json|csv)")
	composeCmd.Flags().String("host", "", "Target hostname (optional - omit to send to general mailbox)")
}
