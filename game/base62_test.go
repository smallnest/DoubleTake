package game

import (
	"strings"
	"testing"
)

func TestRoomCodeRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		ipPort string
	}{
		{"standard", "192.168.1.100:8127"},
		{"localhost", "127.0.0.1:8080"},
		{"broadcast", "255.255.255.255:65535"},
		{"zeros", "0.0.0.0:0"},
		{"min port", "10.0.0.1:1"},
		{"low IP", "1.2.3.4:1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeRoomCode(tt.ipPort)
			if encoded == "" {
				t.Fatalf("EncodeRoomCode(%q) returned empty string", tt.ipPort)
			}

			decoded, err := DecodeRoomCode(encoded)
			if err != nil {
				t.Fatalf("DecodeRoomCode(%q) returned error: %v", encoded, err)
			}

			if decoded != tt.ipPort {
				t.Errorf("roundtrip failed: got %q, want %q", decoded, tt.ipPort)
			}
		})
	}
}

func TestEncodeRoomCode_ProducesBase62(t *testing.T) {
	encoded := EncodeRoomCode("192.168.1.100:8127")
	if encoded == "" {
		t.Fatal("EncodeRoomCode returned empty string")
	}
	for _, c := range encoded {
		if !strings.ContainsRune(base62Chars, c) {
			t.Errorf("encoded string contains non-base62 character: %c", c)
		}
	}
}

func TestEncodeRoomCode_InvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		ipPort string
	}{
		{"empty", ""},
		{"no port", "192.168.1.100"},
		{"invalid IP", "not-an-ip:8080"},
		{"port out of range", "1.2.3.4:99999"},
		{"non-numeric port", "1.2.3.4:abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeRoomCode(tt.ipPort)
			if result != "" {
				t.Errorf("EncodeRoomCode(%q) = %q, want empty string", tt.ipPort, result)
			}
		})
	}
}

func TestDecodeRoomCode_InvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"empty", "", true},
		{"invalid char", "a@b", true},
		{"special char", "a b", true},
		{"newline", "a\nb", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeRoomCode(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeRoomCode(%q) error = %v, wantErr %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestEncodeRoomCode_ShortOutput(t *testing.T) {
	encoded := EncodeRoomCode("192.168.1.100:8127")
	// IPv4:port is 6 bytes = 48 bits, base62 log2(62) ~ 5.95, so ~9 chars max
	if len(encoded) == 0 || len(encoded) > 12 {
		t.Errorf("encoded length %d seems unreasonable for %q", len(encoded), "192.168.1.100:8127")
	}
}
