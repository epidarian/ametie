package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"ametie/internal/config"
	"ametie/internal/network/transport"
)

// Service represents the client service daemon
type Service struct {
	config      *config.ConfigManager
	transport   *transport.Client
	nodeID      string
	running     bool
	stopChan    chan struct{}
	commandChan chan Command
}

// Command represents a queued command
type Command struct {
	ID      int    `json:"id"`
	Command string `json:"command"`
}

// NewService creates a new client service
func NewService(cfg *config.ConfigManager) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	conf := cfg.Get()
	nodeID := cfg.GetNodeIDHash()

	client := transport.NewClient(conf.APIKey, conf.ServerURL, nodeID)

	return &Service{
		config:      cfg,
		transport:   client,
		nodeID:      nodeID,
		stopChan:    make(chan struct{}),
		commandChan: make(chan Command, 100),
	}, nil
}

// Start starts the service
func (s *Service) Start() error {
	if s.running {
		return fmt.Errorf("service is already running")
	}

	s.running = true
	log.Println("Starting Ametie client service...")

	// Start heartbeat loop
	go s.heartbeatLoop()

	// Start command execution loop
	go s.commandLoop()

	// Wait for stop signal
	<-s.stopChan
	s.running = false

	return nil
}

// Stop stops the service
func (s *Service) Stop() {
	if !s.running {
		return
	}

	log.Println("Stopping Ametie client service...")
	close(s.stopChan)
}

// heartbeatLoop performs periodic heartbeats
func (s *Service) heartbeatLoop() {
	baseInterval := 30 * time.Minute
	ticker := time.NewTicker(baseInterval)
	defer ticker.Stop()

	// Initial heartbeat
	s.sendHeartbeat()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.sendHeartbeat()
			// Add jitter for next interval
			jitter := time.Duration(time.Now().UnixNano()%600) * time.Second // 0-10 minutes
			ticker.Reset(baseInterval + jitter)
		}
	}
}

// sendHeartbeat sends a heartbeat to the server
func (s *Service) sendHeartbeat() {
	conf := s.config.Get()
	hostname := conf.Hostname
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	body := map[string]interface{}{
		"hostname":    hostname,
		"custom_name": conf.NodeName,
	}

	resp, err := s.transport.MakeRequest(transport.RequestOptions{
		Method:   "POST",
		Endpoint: "/sync.php",
		Body:     body,
	})

	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Heartbeat returned status %d", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode heartbeat response: %v", err)
		return
	}

	// Process commands
	if commands, ok := result["commands"].([]interface{}); ok {
		for _, cmd := range commands {
			if cmdMap, ok := cmd.(map[string]interface{}); ok {
				cmdID, _ := cmdMap["id"].(float64)
				cmdStr, _ := cmdMap["command"].(string)
				s.commandChan <- Command{
					ID:      int(cmdID),
					Command: cmdStr,
				}
			}
		}
	}

	// Process tunnel requests
	if tunnels, ok := result["tunnels"].([]interface{}); ok {
		for _, tunnel := range tunnels {
			if tunnelMap, ok := tunnel.(map[string]interface{}); ok {
				s.handleTunnelRequest(tunnelMap)
			}
		}
	}

	// Process messages
	if messages, ok := result["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				s.handleMessage(msgMap)
			}
		}
	}
}

// commandLoop processes queued commands
func (s *Service) commandLoop() {
	for {
		select {
		case <-s.stopChan:
			return
		case cmd := <-s.commandChan:
			go s.executeCommand(cmd)
		}
	}
}

// executeCommand executes a command and sends output to mailbox
func (s *Service) executeCommand(cmd Command) {
	log.Printf("Executing command %d: %s", cmd.ID, cmd.Command)

	var shell string
	var shellArg string

	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
		shellArg = "/c"
	} else {
		shell = "/bin/sh"
		shellArg = "-c"
	}

	execCmd := exec.Command(shell, shellArg, cmd.Command)

	// Capture stdout and stderr
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %v", err)
		return
	}

	if err := execCmd.Start(); err != nil {
		log.Printf("Failed to start command: %v", err)
		return
	}

	// Read output
	output, _ := io.ReadAll(stdout)
	errorOutput, _ := io.ReadAll(stderr)

	// Wait for command to complete
	err = execCmd.Wait()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	// Combine output
	combinedOutput := fmt.Sprintf("STDOUT:\n%s\n\nSTDERR:\n%s\n\nExit Code: %d", output, errorOutput, exitCode)

	// Send to mailbox
	s.sendToMailbox(cmd.ID, combinedOutput, err != nil)

	// Mark command as completed
	s.markCommandComplete(cmd.ID, err == nil)
}

// sendToMailbox sends command output to mailbox
func (s *Service) sendToMailbox(commandID int, content string, isError bool) {
	msgType := "command_output"
	if isError {
		msgType = "error"
	}

	body := map[string]interface{}{
		"message_type": msgType,
		"command_id":   commandID,
		"content":      content,
		"type":         msgType,
	}

	resp, err := s.transport.MakeRequest(transport.RequestOptions{
		Method:   "POST",
		Endpoint: "/messages.php",
		Body:     body,
	})

	if err != nil {
		log.Printf("Failed to send to mailbox: %v", err)
		return
	}
	defer resp.Body.Close()
}

// markCommandComplete marks a command as completed
func (s *Service) markCommandComplete(commandID int, success bool) {
	// This would typically update command status on server
	// For now, we'll just log it
	status := "completed"
	if !success {
		status = "failed"
	}
	log.Printf("Command %d marked as %s", commandID, status)
}

// handleTunnelRequest handles a tunnel request
func (s *Service) handleTunnelRequest(tunnel map[string]interface{}) {
	tunnelID, _ := tunnel["id"].(float64)
	localPort, _ := tunnel["local_port"].(float64)
	remotePort, _ := tunnel["remote_port"].(float64)
	
	log.Printf("Received tunnel request ID: %.0f, local: %.0f, remote: %.0f", tunnelID, localPort, remotePort)
	
	// Update tunnel status on server to 'active' when established
	// In a full implementation, this would:
	// 1. Establish SSH tunnel using internal/ssh package
	// 2. Update tunnel status on server
	// 3. Manage tunnel lifecycle
	
	// For now, acknowledge receipt
	body := map[string]interface{}{
		"tunnel_id": int(tunnelID),
		"status":    "acknowledged",
	}
	
	_, err := s.transport.MakeRequest(transport.RequestOptions{
		Method:   "POST",
		Endpoint: "/request.php",
		Body:     body,
	})
	
	if err != nil {
		log.Printf("Failed to acknowledge tunnel request: %v", err)
	}
}

// handleMessage handles a received message
func (s *Service) handleMessage(msg map[string]interface{}) {
	msgID, _ := msg["id"].(float64)
	fromNodeID, _ := msg["from_node_id"].(float64)
	message, _ := msg["message"].(string)
	
	log.Printf("Received message ID: %.0f from node: %.0f", msgID, fromNodeID)
	log.Printf("Message: %s", message)
	
	// Mark message as read on server
	body := map[string]interface{}{
		"message_id": int(msgID),
		"is_read":    true,
	}
	
	_, err := s.transport.MakeRequest(transport.RequestOptions{
		Method:   "POST",
		Endpoint: "/messages.php",
		Body:     body,
	})
	
	if err != nil {
		log.Printf("Failed to mark message as read: %v", err)
	}
	
	// Messages are stored on server and can be retrieved via CLI
	// The service just acknowledges receipt
}

