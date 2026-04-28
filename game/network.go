package game

import (
	"fmt"
	"net"
)

// GetLocalIP returns the first non-loopback IPv4 address of the host,
// excluding link-local addresses (169.254.x.x).
func GetLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			// Skip link-local addresses (169.254.x.x)
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}

			return ip.To4().String(), nil
		}
	}

	return "", fmt.Errorf("no non-loopback IPv4 address found")
}
