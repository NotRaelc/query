package query

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// SplitHostPort parses "host:port". When the port is missing, 0 is
// returned. Callers decide which default port to apply per protocol.
// An empty host becomes 127.0.0.1. An empty or whitespace-only input
// returns an error. On error, host and port are both zero values.
func SplitHostPort(s string) (string, uint16, error) {
	if strings.TrimSpace(s) == "" {
		return "", 0, fmt.Errorf("invalid address %q: empty host", s)
	}

	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		var addrErr *net.AddrError
		if !errors.As(err, &addrErr) {
			return "", 0, fmt.Errorf("invalid address %q: %w", s, err)
		}
		switch addrErr.Err {
		case "missing port in address", "too many colons in address":
			host = strings.TrimSpace(s)
			portStr = ""
		default:
			return "", 0, fmt.Errorf("invalid address %q: %w", s, err)
		}
	}

	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if portStr == "" {
		return host, 0, nil
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port %q: %w", portStr, err)
	}
	return host, uint16(port), nil
}

func isIPLiteral(s string) bool {
	_, err := netip.ParseAddr(s)
	return err == nil
}
