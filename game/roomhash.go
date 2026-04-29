package game

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateRoomHash creates a SHA256 hash based on IP + port + a cryptographically
// secure random number. The hash serves as a room connection password that players
// can use to connect to the judge's server without manually entering the address.
//
// Parameters:
//   - ip: the actual reachable IP address (not 0.0.0.0 or 127.0.0.1)
//   - port: the port string (e.g. "8080")
//
// Returns:
//   - hash: the hex-encoded SHA256 hash (room password)
//   - randomHex: the hex-encoded random bytes used (for verification/debugging)
//   - err: error if random generation fails
func GenerateRoomHash(ip string, port string) (hash string, randomHex string, err error) {
	// Generate 16 cryptographically secure random bytes
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomHex = hex.EncodeToString(randomBytes)

	// Compute SHA256(ip + port + randomHex)
	input := ip + port + randomHex
	sum := sha256.Sum256([]byte(input))
	hash = hex.EncodeToString(sum[:])

	return hash, randomHex, nil
}
