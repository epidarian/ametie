package transport

import (
	"testing"
)

func TestClient_MakeRequest(t *testing.T) {
	client := NewClient("test-api-key", "https://example.com", "test-node-id")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey 'test-api-key', got '%s'", client.apiKey)
	}

	if client.serverURL != "https://example.com" {
		t.Errorf("Expected serverURL 'https://example.com', got '%s'", client.serverURL)
	}

	if client.nodeID != "test-node-id" {
		t.Errorf("Expected nodeID 'test-node-id', got '%s'", client.nodeID)
	}
}

func TestEncryptXOR(t *testing.T) {
	data := []byte("test data")
	apiKey := "test-key"
	timestamp := int64(1234567890)
	nonce := []byte("1234567890123456")

	encrypted := encryptXOR(data, apiKey, timestamp, nonce)

	if len(encrypted) != len(data) {
		t.Errorf("Encrypted length %d != original length %d", len(encrypted), len(data))
	}

	// XOR is symmetric, so encrypting again should get original
	decrypted := encryptXOR(encrypted, apiKey, timestamp, nonce)

	if string(decrypted) != string(data) {
		t.Errorf("Decryption failed: got '%s', expected '%s'", string(decrypted), string(data))
	}
}

func TestCreateSignature(t *testing.T) {
	apiKey := "test-key"
	body := []byte("test body")
	timestamp := int64(1234567890)
	nonce := []byte("1234567890123456")
	endpoint := "/test.php"

	sig1 := createSignature(apiKey, body, timestamp, nonce, endpoint)
	sig2 := createSignature(apiKey, body, timestamp, nonce, endpoint)

	if len(sig1) != 32 { // SHA256 = 32 bytes
		t.Errorf("Signature length %d != 32", len(sig1))
	}

	// Same inputs should produce same signature
	if string(sig1) != string(sig2) {
		t.Error("Same inputs produced different signatures")
	}

	// Different inputs should produce different signature
	sig3 := createSignature(apiKey, []byte("different"), timestamp, nonce, endpoint)
	if string(sig1) == string(sig3) {
		t.Error("Different inputs produced same signature")
	}
}

func TestGetRotatedHeaderNames(t *testing.T) {
	headers1 := getRotatedHeaderNames()
	headers2 := getRotatedHeaderNames()

	// Should have required keys
	required := []string{"request_id", "client_time", "node_id"}
	for _, key := range required {
		if _, ok := headers1[key]; !ok {
			t.Errorf("Missing header key: %s", key)
		}
	}

	// Headers should be consistent within same day
	if headers1["request_id"] != headers2["request_id"] {
		t.Error("Header names should be consistent within same day")
	}
}

func TestCreateChecksum(t *testing.T) {
	data := []byte("test data")
	apiKey := "test-key"
	timestamp := int64(1234567890)

	chk1 := createChecksum(data, apiKey, timestamp)
	chk2 := createChecksum(data, apiKey, timestamp)

	if chk1 == "" {
		t.Error("Checksum should not be empty")
	}

	// Same inputs should produce same checksum
	if chk1 != chk2 {
		t.Error("Same inputs produced different checksums")
	}

	// Different data should produce different checksum
	chk3 := createChecksum([]byte("different"), apiKey, timestamp)
	if chk1 == chk3 {
		t.Error("Different data produced same checksum")
	}
}
