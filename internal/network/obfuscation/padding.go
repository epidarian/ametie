package obfuscation

import (
	"crypto/rand"
	"math/big"
)

// AddRandomPadding adds random padding to request body
func AddRandomPadding(body []byte) []byte {
	// Add 0-64 bytes of random padding
	maxPadding := 64
	paddingSize, _ := rand.Int(rand.Reader, big.NewInt(int64(maxPadding)))
	
	padding := make([]byte, paddingSize.Int64())
	rand.Read(padding)
	
	return append(body, padding...)
}

// RemovePadding removes padding from response body
func RemovePadding(body []byte, originalLength int) []byte {
	if len(body) <= originalLength {
		return body
	}
	return body[:originalLength]
}

// VaryBodySize varies the body size to avoid fingerprinting
func VaryBodySize(body []byte) []byte {
	// Add small random variations
	variation, _ := rand.Int(rand.Reader, big.NewInt(10))
	
	if variation.Int64()%2 == 0 {
		// Sometimes add a bit more
		padding := make([]byte, variation.Int64())
		rand.Read(padding)
		return append(body, padding...)
	}
	
	return body
}

