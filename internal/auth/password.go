package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
)

var (
	sessionPassword string
	sessionMutex    sync.RWMutex
)

// GetPassword retrieves the password from environment or session cache
func GetPassword() (string, error) {
	// Check environment variable first
	if pwd := os.Getenv("AMETIE_PASSWORD"); pwd != "" {
		return pwd, nil
	}

	// Check session cache
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	if sessionPassword != "" {
		return sessionPassword, nil
	}

	return "", fmt.Errorf("password not set")
}

// SetPassword sets the password in session cache
func SetPassword(password string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	sessionPassword = password
}

// ClearPassword clears the session password
func ClearPassword() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	sessionPassword = ""
}

// HashPassword creates a SHA256 hash of the password
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) bool {
	return HashPassword(password) == hash
}

// GenerateNonce generates a random nonce
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return nonce, nil
}

// EncodeNonce encodes nonce to base64
func EncodeNonce(nonce []byte) string {
	return base64.StdEncoding.EncodeToString(nonce)
}

// DecodeNonce decodes base64 nonce
func DecodeNonce(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
