package nat

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// NATInfo contains NAT traversal information
type NATInfo struct {
	PublicIP   string
	PublicPort int
	LocalIP    string
	LocalPort  int
	Type       string // "none", "full-cone", "restricted", "port-restricted", "symmetric"
}

// Traverser handles NAT traversal
type Traverser struct {
	localAddr  *net.TCPAddr
	publicAddr *net.TCPAddr
}

// NewTraverser creates a new NAT traverser
func NewTraverser() *Traverser {
	return &Traverser{}
}

// DiscoverPublicIP attempts to discover public IP using STUN-like techniques
func (t *Traverser) DiscoverPublicIP() (string, error) {
	// Try common STUN servers
	stunServers := []string{
		"stun.l.google.com:19302",
		"stun1.l.google.com:19302",
		"stun2.l.google.com:19302",
	}

	for _, server := range stunServers {
		ip, err := t.querySTUN(server)
		if err == nil {
			return ip, nil
		}
	}

	// Fallback: use external service
	return t.queryExternalIP()
}

// querySTUN queries a STUN server for public IP
func (t *Traverser) querySTUN(server string) (string, error) {
	conn, err := net.DialTimeout("udp", server, 5*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Simple STUN binding request
	request := []byte{
		0x00, 0x01, // Message type: Binding Request
		0x00, 0x00, // Message length
		0x21, 0x12, 0xA4, 0x42, // Magic cookie
	}

	// Add random transaction ID
	transactionID := make([]byte, 12)
	if _, err := rand.Read(transactionID); err != nil {
		return "", fmt.Errorf("failed to generate transaction ID: %w", err)
	}
	request = append(request, transactionID...)

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write(request)
	if err != nil {
		return "", err
	}

	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		return "", err
	}

	// Parse STUN response
	if n < 20 {
		return "", fmt.Errorf("invalid STUN response: too short")
	}

	// Check magic cookie (bytes 4-7)
	if response[4] != 0x21 || response[5] != 0x12 || response[6] != 0xA4 || response[7] != 0x42 {
		return "", fmt.Errorf("invalid STUN response: bad magic cookie")
	}

	// Parse message type (first 2 bytes)
	msgType := binary.BigEndian.Uint16(response[0:2])
	if msgType != 0x0101 { // Binding Response
		return "", fmt.Errorf("invalid STUN response: not a binding response")
	}

	// Parse message length
	msgLength := binary.BigEndian.Uint16(response[2:4])

	// Look for MAPPED-ADDRESS attribute (0x0001) or XOR-MAPPED-ADDRESS (0x0020)
	offset := 20 // Skip header
	for offset < int(msgLength)+20 {
		if offset+4 > len(response) {
			break
		}

		attrType := binary.BigEndian.Uint16(response[offset : offset+2])
		attrLength := binary.BigEndian.Uint16(response[offset+2 : offset+4])
		offset += 4

		if offset+int(attrLength) > len(response) {
			break
		}

		// MAPPED-ADDRESS (0x0001) or XOR-MAPPED-ADDRESS (0x0020)
		if attrType == 0x0001 || attrType == 0x0020 {
			if attrLength < 8 {
				break
			}

			// Skip family and port (first 4 bytes)
			port := binary.BigEndian.Uint16(response[offset+2 : offset+4])
			ipBytes := response[offset+4 : offset+4+4]

			// If XOR-MAPPED-ADDRESS, XOR with magic cookie
			if attrType == 0x0020 {
				port ^= 0x2112
				for i := range ipBytes {
					ipBytes[i] ^= []byte{0x21, 0x12, 0xA4, 0x42}[i]
				}
			}

			ip := net.IP(ipBytes)
			return fmt.Sprintf("%s:%d", ip.String(), port), nil
		}

		offset += int(attrLength)
		// Align to 4-byte boundary
		offset = (offset + 3) &^ 3
	}

	return "", fmt.Errorf("could not find mapped address in STUN response")
}

// queryExternalIP queries an external service for public IP
func (t *Traverser) queryExternalIP() (string, error) {
	services := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ifconfig.me/ip",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Try each service until one succeeds
	for _, serviceURL := range services {
		req, err := http.NewRequest("GET", serviceURL, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		if net.ParseIP(ip) != nil {
			return ip, nil
		}
	}

	return "", fmt.Errorf("could not determine public IP")
}

// AttemptUPnP attempts to open a port using UPnP
func (t *Traverser) AttemptUPnP(localPort, externalPort int, protocol string) error {
	// Discover UPnP IGD (Internet Gateway Device)
	gatewayURL, controlURL, err := t.discoverUPnPIGD()
	if err != nil {
		return fmt.Errorf("failed to discover UPnP gateway: %w", err)
	}

	// Add port mapping using SOAP
	return t.addPortMapping(gatewayURL, controlURL, localPort, externalPort, protocol)
}

// discoverUPnPIGD discovers UPnP Internet Gateway Device
func (t *Traverser) discoverUPnPIGD() (string, string, error) {
	// SSDP discovery
	ssdpAddr, err := net.ResolveUDPAddr("udp", "239.255.255.250:1900")
	if err != nil {
		return "", "", err
	}

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return "", "", err
	}
	defer conn.Close()

	// Send M-SEARCH request
	searchMsg := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n" +
		"MX: 3\r\n\r\n"

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, err = conn.WriteToUDP([]byte(searchMsg), ssdpAddr)
	if err != nil {
		return "", "", err
	}

	// Read response
	buffer := make([]byte, 4096)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return "", "", err
	}

	response := string(buffer[:n])
	location := ""
	controlURL := ""

	// Parse response headers
	lines := strings.Split(response, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "LOCATION:") {
			location = strings.TrimSpace(line[9:])
		}
	}

	if location == "" {
		return "", "", fmt.Errorf("no location header in UPnP response")
	}

	// Get device description
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(location)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	// Parse XML to find control URL (simplified - would use XML parser in production)
	bodyStr := string(body)
	if strings.Contains(bodyStr, "WANIPConnection") || strings.Contains(bodyStr, "WANPPPConnection") {
		// Extract control URL (simplified parsing)
		controlURL = location // Use location as base, would parse XML properly in production
	}

	if controlURL == "" {
		return "", "", fmt.Errorf("could not find control URL")
	}

	return location, controlURL, nil
}

// addPortMapping adds a port mapping via UPnP SOAP
func (t *Traverser) addPortMapping(gatewayURL, controlURL string, localPort, externalPort int, protocol string) error {
	// Get local IP
	localIP := "0.0.0.0" // Would get actual local IP
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					localIP = ipnet.IP.String()
					break
				}
			}
		}
	}

	// SOAP request for AddPortMapping
	soapAction := "urn:schemas-upnp-org:service:WANIPConnection:1#AddPortMapping"
	soapBody := fmt.Sprintf(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:AddPortMapping xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:1">
<NewRemoteHost></NewRemoteHost>
<NewExternalPort>%d</NewExternalPort>
<NewProtocol>%s</NewProtocol>
<NewInternalPort>%d</NewInternalPort>
<NewInternalClient>%s</NewInternalClient>
<NewEnabled>1</NewEnabled>
<NewPortMappingDescription>Ametie</NewPortMappingDescription>
<NewLeaseDuration>0</NewLeaseDuration>
</u:AddPortMapping>
</s:Body>
</s:Envelope>`, externalPort, protocol, localPort, localIP)

	req, err := http.NewRequest("POST", controlURL, bytes.NewBufferString(soapBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Set("SOAPAction", `"`+soapAction+`"`)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("UPnP request failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetNATType determines the NAT type using STUN
func (t *Traverser) GetNATType() (string, error) {
	// Basic NAT type detection using STUN
	// Full detection requires multiple queries from different ports
	// This is a simplified version that checks if we can get our public IP

	ip, err := t.DiscoverPublicIP()
	if err != nil {
		return "unknown", err
	}

	if ip != "" {
		// If we can discover public IP, NAT exists
		// To determine exact type (full-cone, restricted, etc.) would require:
		// 1. Multiple STUN queries from same local port to different servers
		// 2. Queries from different local ports
		// 3. Analysis of port mapping behavior
		return "detected", nil
	}

	return "none", nil
}
