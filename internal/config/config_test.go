package config

import (
	"os"
	"testing"
)

func TestNewConfigManager(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set HOME to temp dir
	os.Setenv("HOME", tmpDir)

	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}
	if cm == nil {
		t.Fatal("NewConfigManager returned nil")
	}
}

func TestConfigManager_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)

	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	// Test SetAPIKey
	err = cm.SetAPIKey("test-key")
	if err != nil {
		t.Fatalf("SetAPIKey failed: %v", err)
	}

	// Test SetServerURL
	err = cm.SetServerURL("https://example.com")
	if err != nil {
		t.Fatalf("SetServerURL failed: %v", err)
	}

	// Test SetNodeName
	err = cm.SetNodeName("test-node")
	if err != nil {
		t.Fatalf("SetNodeName failed: %v", err)
	}

	// Test Get
	config := cm.Get()
	if config == nil {
		t.Fatal("Get returned nil")
	}
	if config.APIKey != "test-key" {
		t.Errorf("Expected APIKey 'test-key', got '%s'", config.APIKey)
	}
	if config.ServerURL != "https://example.com" {
		t.Errorf("Expected ServerURL 'https://example.com', got '%s'", config.ServerURL)
	}
	if config.NodeName != "test-node" {
		t.Errorf("Expected NodeName 'test-node', got '%s'", config.NodeName)
	}
}

func TestConfigManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)

	// Create and save config
	cm1, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	cm1.SetAPIKey("test-key")
	cm1.SetServerURL("https://example.com")
	cm1.SetNodeName("test-node")

	err = cm1.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load config
	cm2, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	err = cm2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	config := cm2.Get()
	if config.APIKey != "test-key" {
		t.Errorf("Loaded APIKey '%s' != saved 'test-key'", config.APIKey)
	}
	if config.ServerURL != "https://example.com" {
		t.Errorf("Loaded ServerURL '%s' != saved 'https://example.com'", config.ServerURL)
	}
	if config.NodeName != "test-node" {
		t.Errorf("Loaded NodeName '%s' != saved 'test-node'", config.NodeName)
	}
}

func TestConfigManager_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)

	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	// Empty config should fail validation
	err = cm.Validate()
	if err == nil {
		t.Error("Empty config should fail validation")
	}

	// Set required fields
	cm.SetAPIKey("test-key")
	cm.SetServerURL("https://example.com")

	// Should pass validation now
	err = cm.Validate()
	if err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}
}

func TestConfigManager_GetNodeIDHash(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)

	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	cm.SetAPIKey("test-key")
	cm.SetHostname("test-host")

	hash := cm.GetNodeIDHash()
	if hash == "" {
		t.Error("GetNodeIDHash returned empty string")
	}
	if len(hash) != 32 { // SHA256 first 16 bytes as hex = 32 chars
		t.Errorf("Expected hash length 32, got %d", len(hash))
	}

	// Same inputs should produce same hash
	hash2 := cm.GetNodeIDHash()
	if hash != hash2 {
		t.Error("Same inputs produced different hashes")
	}
}
