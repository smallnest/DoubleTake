package game

import (
	"encoding/hex"
	"testing"
)

func TestGenerateRoomHash_Basic(t *testing.T) {
	hash, randomHex, err := GenerateRoomHash("192.168.1.100", "8080")
	if err != nil {
		t.Fatalf("GenerateRoomHash() returned error: %v", err)
	}

	// SHA256 produces 32 bytes = 64 hex characters
	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash))
	}
	if _, err := hex.DecodeString(hash); err != nil {
		t.Fatalf("hash is not valid hex: %v", err)
	}

	// 16 random bytes = 32 hex characters
	if len(randomHex) != 32 {
		t.Fatalf("randomHex length = %d, want 32", len(randomHex))
	}
	if _, err := hex.DecodeString(randomHex); err != nil {
		t.Fatalf("randomHex is not valid hex: %v", err)
	}
}

func TestGenerateRoomHash_UniquePerCall(t *testing.T) {
	// Each call should produce a different hash due to random component
	hashes := make(map[string]bool)
	const iterations = 100

	for i := 0; i < iterations; i++ {
		hash, _, err := GenerateRoomHash("192.168.1.100", "8080")
		if err != nil {
			t.Fatalf("iteration %d: GenerateRoomHash() returned error: %v", i, err)
		}
		if hashes[hash] {
			t.Fatalf("iteration %d: duplicate hash produced", i)
		}
		hashes[hash] = true
	}
}

func TestGenerateRoomHash_DifferentInputs(t *testing.T) {
	// Same IP/port but different calls should still produce different hashes
	// (because of the random component)
	hash1, _, err := GenerateRoomHash("10.0.0.1", "3000")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	hash2, _, err := GenerateRoomHash("10.0.0.1", "3000")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if hash1 == hash2 {
		t.Fatalf("expected different hashes for same input, got identical: %s", hash1)
	}
}

func TestGenerateRoomHash_VariousIPsAndPorts(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		port string
	}{
		{"localhost-like IP", "127.0.0.1", "8080"},
		{"LAN IP", "192.168.1.100", "9090"},
		{"public IP", "203.0.113.50", "443"},
		{"zero port", "10.0.0.1", "0"},
		{"max port", "10.0.0.1", "65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, randomHex, err := GenerateRoomHash(tt.ip, tt.port)
			if err != nil {
				t.Fatalf("GenerateRoomHash(%q, %q) error: %v", tt.ip, tt.port, err)
			}
			if len(hash) != 64 {
				t.Fatalf("hash length = %d, want 64", len(hash))
			}
			if len(randomHex) != 32 {
				t.Fatalf("randomHex length = %d, want 32", len(randomHex))
			}
		})
	}
}

func TestGenerateRoomHash_RandomHexUniqueness(t *testing.T) {
	randoms := make(map[string]bool)
	const iterations = 50

	for i := 0; i < iterations; i++ {
		_, randomHex, err := GenerateRoomHash("192.168.1.1", "8080")
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if randoms[randomHex] {
			t.Fatalf("iteration %d: duplicate randomHex produced", i)
		}
		randoms[randomHex] = true
	}
}
