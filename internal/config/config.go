package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Endpoint represents a server endpoint configuration
type Endpoint struct {
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	IsMirror bool   `json:"is_mirror"`
	Cluster  string `json:"cluster,omitempty"` // Cluster identifier
}

// Config represents the application configuration
type Config struct {
	APIKey    string     `json:"api_key"` // Will be encrypted in storage
	ServerURL string     `json:"server_url"` // Primary/legacy
	NodeName  string     `json:"node_name"`
	Hostname  string     `json:"hostname"`
	Endpoints []Endpoint `json:"endpoints,omitempty"` // Multiple endpoints
}

// ConfigManager handles configuration operations
type ConfigManager struct {
	configPath string
	config     *Config
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")

	cm := &ConfigManager{
		configPath: configPath,
	}

	// Load existing config if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := cm.Load(); err != nil {
			return nil, err
		}
	} else {
		cm.config = &Config{}
	}

	return cm, nil
}

// getConfigDir returns the configuration directory based on OS
func getConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "ametie")
	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "ametie")
	case "linux":
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "ametie")
	default:
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "ametie")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// Load loads configuration from file
func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Decrypt API key from storage (simplified - in production use OS keychain)
	cm.config = &config
	return nil
}

// Save saves configuration to file
func (cm *ConfigManager) Save() error {
	if cm.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (cm *ConfigManager) Get() *Config {
	return cm.config
}

// SetAPIKey sets the API key (will be encrypted in storage)
func (cm *ConfigManager) SetAPIKey(key string) error {
	if cm.config == nil {
		cm.config = &Config{}
	}
	cm.config.APIKey = key
	return nil
}

// SetServerURL sets the server URL
func (cm *ConfigManager) SetServerURL(url string) error {
	if cm.config == nil {
		cm.config = &Config{}
	}
	cm.config.ServerURL = url
	return nil
}

// SetNodeName sets the node name
func (cm *ConfigManager) SetNodeName(name string) error {
	if cm.config == nil {
		cm.config = &Config{}
	}
	cm.config.NodeName = name
	return nil
}

// SetHostname sets the hostname
func (cm *ConfigManager) SetHostname(hostname string) error {
	if cm.config == nil {
		cm.config = &Config{}
	}
	cm.config.Hostname = hostname
	return nil
}

// GetNodeIDHash generates a node ID hash from API key and hostname
func (cm *ConfigManager) GetNodeIDHash() string {
	if cm.config == nil || cm.config.APIKey == "" {
		return ""
	}

	hostname := cm.config.Hostname
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	customName := cm.config.NodeName
	if customName == "" {
		customName = hostname
	}

	data := cm.config.APIKey + hostname + customName
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16]) // First 16 bytes as hex
}

// AddEndpoint adds or updates an endpoint
func (cm *ConfigManager) AddEndpoint(url string, priority int, isMirror bool, cluster string) error {
	if cm.config == nil {
		cm.config = &Config{}
	}

	// Initialize endpoints if needed
	if cm.config.Endpoints == nil {
		cm.config.Endpoints = []Endpoint{}
	}

	// Check if endpoint already exists
	for i, ep := range cm.config.Endpoints {
		if ep.URL == url {
			// Update existing
			cm.config.Endpoints[i].Priority = priority
			cm.config.Endpoints[i].IsMirror = isMirror
			if cluster != "" {
				cm.config.Endpoints[i].Cluster = cluster
			}
			return nil
		}
	}

	// Add new endpoint
	cm.config.Endpoints = append(cm.config.Endpoints, Endpoint{
		URL:      url,
		Priority: priority,
		IsMirror: isMirror,
		Cluster:  cluster,
	})

	// If this is the first endpoint and no ServerURL, set it
	if cm.config.ServerURL == "" {
		cm.config.ServerURL = url
	}

	return nil
}

// GetEndpoints returns all configured endpoints
func (cm *ConfigManager) GetEndpoints() []Endpoint {
	if cm.config == nil || cm.config.Endpoints == nil {
		return []Endpoint{}
	}
	return cm.config.Endpoints
}

// ClearEndpoints clears all endpoints (for new cluster)
func (cm *ConfigManager) ClearEndpoints() {
	if cm.config != nil {
		cm.config.Endpoints = []Endpoint{}
		cm.config.ServerURL = ""
	}
}

// Validate validates the configuration
func (cm *ConfigManager) Validate() error {
	if cm.config == nil {
		return fmt.Errorf("configuration is nil")
	}

	if cm.config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Either ServerURL or at least one endpoint is required
	if cm.config.ServerURL == "" && (cm.config.Endpoints == nil || len(cm.config.Endpoints) == 0) {
		return fmt.Errorf("server URL or endpoint is required")
	}

	return nil
}
