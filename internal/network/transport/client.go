package transport

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles HTTP requests with obfuscation
type Client struct {
	apiKey    string
	serverURL string
	nodeID    string
	client    *http.Client
}

// NewClient creates a new transport client
func NewClient(apiKey, serverURL, nodeID string) *Client {
	return &Client{
		apiKey:    apiKey,
		serverURL: serverURL,
		nodeID:    nodeID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RequestOptions contains options for making a request
type RequestOptions struct {
	Method   string
	Endpoint string
	Body     interface{}
	Headers  map[string]string
}

// MakeRequest makes an obfuscated HTTP request
func (c *Client) MakeRequest(opts RequestOptions) (*http.Response, error) {
	// Generate nonce
	nonce := generateNonce()
	timestamp := time.Now().Unix()

	// Serialize body
	var bodyBytes []byte
	var err error
	if opts.Body != nil {
		bodyBytes, err = json.Marshal(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
	}

	// Encrypt body
	encryptedBody := encryptXOR(bodyBytes, c.apiKey, timestamp, nonce)

	// Create signature
	signature := createSignature(c.apiKey, encryptedBody, timestamp, nonce, opts.Endpoint)

	// Create request body with obfuscated signature
	requestBody := map[string]interface{}{
		"sig":  base64.StdEncoding.EncodeToString(signature[16:]), // Last 16 bytes
		"chk":  createChecksum(encryptedBody, c.apiKey, timestamp),
		"data": base64.StdEncoding.EncodeToString(encryptedBody),
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Build URL
	url := c.serverURL + opts.Endpoint

	// Create HTTP request
	req, err := http.NewRequest(opts.Method, url, bytes.NewReader(requestBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set obfuscated headers (with rotation)
	headerNames := getRotatedHeaderNames()
	req.Header.Set(headerNames["request_id"], base64.StdEncoding.EncodeToString(nonce[:12]))
	req.Header.Set(headerNames["client_time"], fmt.Sprintf("%d", timestamp))
	req.Header.Set(headerNames["node_id"], c.nodeID)

	// Add custom headers
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	// Add browser-like headers for obfuscation
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// generateNonce generates a random nonce
func generateNonce() []byte {
	nonce := make([]byte, 16)
	// In production, use crypto/rand
	for i := range nonce {
		nonce[i] = byte(time.Now().UnixNano() % 256)
	}
	return nonce
}

// encryptXOR encrypts data using XOR with derived key
func encryptXOR(data []byte, apiKey string, timestamp int64, nonce []byte) []byte {
	keyData := fmt.Sprintf("%s%d%s", apiKey, timestamp, string(nonce))
	key := sha256.Sum256([]byte(keyData))
	keyBytes := key[:32]

	encrypted := make([]byte, len(data))
	for i := range data {
		encrypted[i] = data[i] ^ keyBytes[i%32]
	}

	return encrypted
}

// createSignature creates HMAC signature
func createSignature(apiKey string, body []byte, timestamp int64, nonce []byte, endpoint string) []byte {
	data := fmt.Sprintf("%s%d%s%s", string(body), timestamp, string(nonce), endpoint)
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

// createChecksum creates XOR checksum
func createChecksum(data []byte, apiKey string, timestamp int64) string {
	keyData := fmt.Sprintf("%s%d", apiKey, timestamp)
	key := sha256.Sum256([]byte(keyData))
	keyBytes := key[:8]

	checksum := make([]byte, 8)
	for i := 0; i < 8 && i < len(data); i++ {
		checksum[i] = data[i] ^ keyBytes[i]
	}

	return base64.StdEncoding.EncodeToString(checksum)
}

// getRotatedHeaderNames returns header names based on day of year
func getRotatedHeaderNames() map[string]string {
	dayOfYear := time.Now().YearDay()
	rotation := dayOfYear % 7

	variants := []map[string]string{
		{"request_id": "X-Request-Id", "client_time": "X-Client-Time", "node_id": "X-Node-Id"},
		{"request_id": "X-Req-Id", "client_time": "X-Time", "node_id": "X-Node"},
		{"request_id": "X-RID", "client_time": "X-CT", "node_id": "X-NID"},
		{"request_id": "X-Request-ID", "client_time": "X-ClientTime", "node_id": "X-NodeID"},
		{"request_id": "X-Rq-Id", "client_time": "X-C-Time", "node_id": "X-Node-Identifier"},
		{"request_id": "X-ReqID", "client_time": "X-Client-Timestamp", "node_id": "X-NodeId"},
		{"request_id": "X-RequestID", "client_time": "X-CTime", "node_id": "X-NID"},
	}

	return variants[rotation]
}

// DecodeResponse decodes and decrypts response body
func DecodeResponse(resp *http.Response, apiKey string, timestamp int64, nonce []byte) (map[string]interface{}, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If response contains encrypted data, decrypt it
	if encData, ok := data["data"].(string); ok {
		encrypted, err := base64.StdEncoding.DecodeString(encData)
		if err != nil {
			return data, nil // Return as-is if decryption fails
		}

		decrypted := encryptXOR(encrypted, apiKey, timestamp, nonce) // XOR is symmetric
		if err := json.Unmarshal(decrypted, &data); err != nil {
			return data, nil // Return original if parse fails
		}
	}

	return data, nil
}
