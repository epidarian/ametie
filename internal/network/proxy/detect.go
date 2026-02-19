package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	Type     string // "http", "socks4", "socks5"
	Host     string
	Port     int
	Username string
	Password string
}

// DetectProxies detects available proxies from environment
func DetectProxies() []*ProxyConfig {
	var proxies []*ProxyConfig

	// Check HTTP_PROXY, HTTPS_PROXY
	if httpProxy := os.Getenv("HTTP_PROXY"); httpProxy != "" {
		if proxy, err := parseProxyURL(httpProxy, "http"); err == nil {
			proxies = append(proxies, proxy)
		}
	}

	if httpsProxy := os.Getenv("HTTPS_PROXY"); httpsProxy != "" {
		if proxy, err := parseProxyURL(httpsProxy, "http"); err == nil {
			proxies = append(proxies, proxy)
		}
	}

	// Check ALL_PROXY
	if allProxy := os.Getenv("ALL_PROXY"); allProxy != "" {
		if proxy, err := parseProxyURL(allProxy, "socks5"); err == nil {
			proxies = append(proxies, proxy)
		}
	}

	// Check NO_PROXY to exclude certain hosts
	noProxy := os.Getenv("NO_PROXY")

	return filterProxies(proxies, noProxy)
}

// parseProxyURL parses a proxy URL string
func parseProxyURL(proxyURL, defaultType string) (*ProxyConfig, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	config := &ProxyConfig{
		Type: defaultType,
		Host: u.Hostname(),
	}

	// Determine type from scheme
	if u.Scheme == "socks4" || u.Scheme == "socks4a" {
		config.Type = "socks4"
	} else if u.Scheme == "socks5" || u.Scheme == "socks5h" {
		config.Type = "socks5"
	} else if u.Scheme == "http" || u.Scheme == "https" {
		config.Type = "http"
	}

	// Parse port
	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &config.Port)
	} else {
		if config.Type == "http" {
			config.Port = 8080
		} else {
			config.Port = 1080
		}
	}

	// Parse credentials
	if u.User != nil {
		config.Username = u.User.Username()
		config.Password, _ = u.User.Password()
	}

	return config, nil
}

// filterProxies filters proxies based on NO_PROXY
func filterProxies(proxies []*ProxyConfig, noProxy string) []*ProxyConfig {
	if noProxy == "" {
		return proxies
	}

	excludedHosts := strings.Split(noProxy, ",")
	filtered := []*ProxyConfig{}

	for _, proxy := range proxies {
		shouldExclude := false
		for _, exclude := range excludedHosts {
			if strings.Contains(proxy.Host, strings.TrimSpace(exclude)) {
				shouldExclude = true
				break
			}
		}
		if !shouldExclude {
			filtered = append(filtered, proxy)
		}
	}

	return filtered
}

// GetHTTPTransport returns an HTTP transport configured for the proxy
func (p *ProxyConfig) GetHTTPTransport() (*http.Transport, error) {
	proxyURL := fmt.Sprintf("%s://%s:%d", p.Type, p.Host, p.Port)
	if p.Username != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		u.User = url.UserPassword(p.Username, p.Password)
		proxyURL = u.String()
	}

	proxyURLParsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURLParsed),
	}

	return transport, nil
}
