package proxy

import (
	"os"
	"testing"
)

func TestDetectProxies(t *testing.T) {
	// Save original env
	origHTTP := os.Getenv("HTTP_PROXY")
	origHTTPS := os.Getenv("HTTPS_PROXY")
	origAll := os.Getenv("ALL_PROXY")
	defer func() {
		if origHTTP != "" {
			os.Setenv("HTTP_PROXY", origHTTP)
		} else {
			os.Unsetenv("HTTP_PROXY")
		}
		if origHTTPS != "" {
			os.Setenv("HTTPS_PROXY", origHTTPS)
		} else {
			os.Unsetenv("HTTPS_PROXY")
		}
		if origAll != "" {
			os.Setenv("ALL_PROXY", origAll)
		} else {
			os.Unsetenv("ALL_PROXY")
		}
	}()

	// Test with HTTP_PROXY
	os.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	proxies := DetectProxies()
	if len(proxies) == 0 {
		t.Error("Expected to detect HTTP proxy")
	}
	if len(proxies) > 0 && proxies[0].Host != "proxy.example.com" {
		t.Errorf("Expected host 'proxy.example.com', got '%s'", proxies[0].Host)
	}
	if len(proxies) > 0 && proxies[0].Port != 8080 {
		t.Errorf("Expected port 8080, got %d", proxies[0].Port)
	}
}

func TestParseProxyURL(t *testing.T) {
	tests := []struct {
		url          string
		defaultType  string
		expectedHost string
		expectedPort int
		expectedType string
	}{
		{"http://proxy.com:8080", "http", "proxy.com", 8080, "http"},
		{"https://proxy.com:8443", "http", "proxy.com", 8443, "http"},
		{"socks5://proxy.com:1080", "socks5", "proxy.com", 1080, "socks5"},
		{"socks4://proxy.com:1080", "socks5", "proxy.com", 1080, "socks4"},
		{"http://user:pass@proxy.com:8080", "http", "proxy.com", 8080, "http"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			config, err := parseProxyURL(tt.url, tt.defaultType)
			if err != nil {
				t.Fatalf("parseProxyURL failed: %v", err)
			}
			if config.Host != tt.expectedHost {
				t.Errorf("Host: got '%s', expected '%s'", config.Host, tt.expectedHost)
			}
			if config.Port != tt.expectedPort {
				t.Errorf("Port: got %d, expected %d", config.Port, tt.expectedPort)
			}
			if config.Type != tt.expectedType {
				t.Errorf("Type: got '%s', expected '%s'", config.Type, tt.expectedType)
			}
		})
	}
}

func TestFilterProxies(t *testing.T) {
	proxies := []*ProxyConfig{
		{Host: "proxy1.com", Port: 8080},
		{Host: "proxy2.example.com", Port: 8080},
		{Host: "proxy3.com", Port: 8080},
	}

	// Filter out proxy2
	filtered := filterProxies(proxies, "example.com")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 proxies after filtering, got %d", len(filtered))
	}

	// Check that proxy2 is excluded
	for _, p := range filtered {
		if p.Host == "proxy2.example.com" {
			t.Error("proxy2 should have been filtered out")
		}
	}
}

func TestProxyConfig_GetHTTPTransport(t *testing.T) {
	config := &ProxyConfig{
		Type: "http",
		Host: "proxy.com",
		Port: 8080,
	}

	transport, err := config.GetHTTPTransport()
	if err != nil {
		t.Fatalf("GetHTTPTransport failed: %v", err)
	}
	if transport == nil {
		t.Fatal("Transport should not be nil")
	}
}
