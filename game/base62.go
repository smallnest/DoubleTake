package game

import (
	"fmt"
	"net"
	"strconv"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// EncodeRoomCode encodes an IP:port string into a base62 short string (room code).
func EncodeRoomCode(ipPort string) string {
	host, portStr, err := net.SplitHostPort(ipPort)
	if err != nil {
		return ""
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return ""
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}

	// Convert to uint64: 4 bytes IP + 2 bytes port
	var num uint64
	for _, b := range ip4 {
		num = num<<8 | uint64(b)
	}
	num = num<<16 | uint64(port)

	if num == 0 {
		return string(base62Chars[0])
	}

	var encoded []byte
	for num > 0 {
		encoded = append(encoded, base62Chars[num%62])
		num /= 62
	}

	// Reverse to get most-significant digit first
	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}

	return string(encoded)
}

// DecodeRoomCode decodes a base62 room code back to IP:port.
func DecodeRoomCode(code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("empty code")
	}

	var num uint64
	for _, c := range code {
		var idx int
		switch {
		case c >= '0' && c <= '9':
			idx = int(c - '0')
		case c >= 'A' && c <= 'Z':
			idx = 10 + int(c-'A')
		case c >= 'a' && c <= 'z':
			idx = 36 + int(c-'a')
		default:
			return "", fmt.Errorf("invalid character in code: %c", c)
		}
		num = num*62 + uint64(idx)
	}

	// Extract port (lower 16 bits) and IP (upper 32 bits)
	port := num & 0xFFFF
	num >>= 16

	ipBytes := make(net.IP, 4)
	for i := 3; i >= 0; i-- {
		ipBytes[i] = byte(num & 0xFF)
		num >>= 8
	}

	return fmt.Sprintf("%s:%d", ipBytes.String(), port), nil
}
