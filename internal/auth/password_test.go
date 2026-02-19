package auth

import (
	"os"
	"testing"
)

func TestGetPassword_EnvironmentVariable(t *testing.T) {
	// Save original
	origPwd := os.Getenv("AMETIE_PASSWORD")
	defer func() {
		if origPwd != "" {
			os.Setenv("AMETIE_PASSWORD", origPwd)
		} else {
			os.Unsetenv("AMETIE_PASSWORD")
		}
		ClearPassword()
	}()

	// Test environment variable
	os.Setenv("AMETIE_PASSWORD", "test-password")
	ClearPassword() // Clear session cache

	pwd, err := GetPassword()
	if err != nil {
		t.Fatalf("GetPassword failed: %v", err)
	}
	if pwd != "test-password" {
		t.Errorf("Expected 'test-password', got '%s'", pwd)
	}
}

func TestSetAndGetPassword_SessionCache(t *testing.T) {
	ClearPassword()
	os.Unsetenv("AMETIE_PASSWORD")

	// Should fail initially
	_, err := GetPassword()
	if err == nil {
		t.Error("GetPassword should fail when no password is set")
	}

	// Set password
	SetPassword("session-password")

	// Should succeed now
	pwd, err := GetPassword()
	if err != nil {
		t.Fatalf("GetPassword failed: %v", err)
	}
	if pwd != "session-password" {
		t.Errorf("Expected 'session-password', got '%s'", pwd)
	}
}

func TestHashPassword(t *testing.T) {
	pwd := "test-password"
	hash1 := HashPassword(pwd)
	hash2 := HashPassword(pwd)

	if hash1 == "" {
		t.Error("HashPassword returned empty string")
	}
	if len(hash1) != 64 { // SHA256 hex = 64 chars
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
	if hash1 != hash2 {
		t.Error("Same password produced different hashes")
	}
}

func TestVerifyPassword(t *testing.T) {
	pwd := "test-password"
	hash := HashPassword(pwd)

	if !VerifyPassword(pwd, hash) {
		t.Error("VerifyPassword failed for correct password")
	}
	if VerifyPassword("wrong-password", hash) {
		t.Error("VerifyPassword succeeded for wrong password")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}
	if len(nonce) != 16 {
		t.Errorf("Expected nonce length 16, got %d", len(nonce))
	}

	// Generate another nonce - should be different
	nonce2, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}
	if string(nonce) == string(nonce2) {
		t.Error("Two nonces should be different")
	}
}

func TestEncodeDecodeNonce(t *testing.T) {
	original := []byte("1234567890123456")
	encoded := EncodeNonce(original)

	if encoded == "" {
		t.Error("EncodeNonce returned empty string")
	}

	decoded, err := DecodeNonce(encoded)
	if err != nil {
		t.Fatalf("DecodeNonce failed: %v", err)
	}

	if string(decoded) != string(original) {
		t.Errorf("Decoded nonce '%s' != original '%s'", string(decoded), string(original))
	}
}
