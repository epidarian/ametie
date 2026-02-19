package nat

import (
	"net"
	"strings"
	"testing"
)

func TestNewTraverser(t *testing.T) {
	traverser := NewTraverser()
	if traverser == nil {
		t.Fatal("NewTraverser returned nil")
	}
}

func TestQueryExternalIP(t *testing.T) {
	traverser := NewTraverser()

	// This will make actual HTTP requests, so it might fail in test environment
	ip, err := traverser.queryExternalIP()

	// If it succeeds, validate the IP
	if err == nil {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			t.Errorf("queryExternalIP returned invalid IP: %s", ip)
		}
	}
	// If it fails, that's okay - might be network issues in test environment
}

func TestGetNATType(t *testing.T) {
	traverser := NewTraverser()

	natType, err := traverser.GetNATType()
	if err != nil {
		t.Logf("GetNATType returned error (expected in some environments): %v", err)
		return
	}

	validTypes := []string{"detected", "none", "unknown"}
	valid := false
	for _, vt := range validTypes {
		if natType == vt {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("Invalid NAT type: %s", natType)
	}
}

func TestDiscoverPublicIP(t *testing.T) {
	traverser := NewTraverser()

	// This will try STUN first, then fallback to HTTP
	ip, err := traverser.DiscoverPublicIP()

	if err == nil {
		// STUN returns IP:port, HTTP services return just IP
		// Extract IP part if port is present
		ipStr := ip
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ipStr = ip[:idx]
		}

		// Validate IP
		parsedIP := net.ParseIP(ipStr)
		if parsedIP == nil {
			t.Errorf("DiscoverPublicIP returned invalid IP: %s (extracted from %s)", ipStr, ip)
		}
	} else {
		t.Logf("DiscoverPublicIP failed (expected in test environment): %v", err)
	}
}
