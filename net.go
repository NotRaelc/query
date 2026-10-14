package query

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// Resolver resolves a hostname to IP addresses ordered by preference:
// IPv4 first, then IPv6.
type Resolver func(ctx context.Context, host string) ([]netip.Addr, error)

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

// ResolveHost resolves host preferring IPv4. Already-IP inputs are
// returned as a single-element slice.
func ResolveHost(ctx context.Context, host string) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{addr}, nil
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("dns lookup failed for %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no IP addresses for %q", host)
	}
	v4 := make([]netip.Addr, 0, len(addrs))
	v6 := make([]netip.Addr, 0, len(addrs))
	for _, a := range addrs {
		if a.Is4() {
			v4 = append(v4, a)
		} else {
			v6 = append(v6, a)
		}
	}
	return append(v4, v6...), nil
}

// ResolvePreferred returns a single address preferring IPv4 over IPv6.
func ResolvePreferred(ctx context.Context, host string) (netip.Addr, error) {
	addrs, err := ResolveHost(ctx, host)
	if err != nil {
		return netip.Addr{}, err
	}
	return addrs[0], nil
}

func isBlockedAddr(a netip.Addr) bool {
	return a.IsLoopback() || a.IsUnspecified()
}

// IsBlockedHost reports whether host should be rejected using the
// default resolver.
func IsBlockedHost(ctx context.Context, host string) bool {
	return Options{}.isBlocked(ctx, host)
}

func (o Options) resolveHost(ctx context.Context, host string) ([]netip.Addr, error) {
	if o.Resolver != nil {
		return o.Resolver(ctx, host)
	}
	return ResolveHost(ctx, host)
}

func (o Options) resolvePreferred(ctx context.Context, host string) (netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr, nil
	}
	addrs, err := o.resolveHost(ctx, host)
	if err != nil {
		return netip.Addr{}, err
	}
	return addrs[0], nil
}

func (o Options) isBlocked(ctx context.Context, host string) bool {
	if o.AllowLocalhost {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return isBlockedAddr(addr)
	}
	addrs, err := o.resolveHost(ctx, host)
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if isBlockedAddr(a) {
			return true
		}
	}
	return false
}

// IsDialError reports whether err occurred during connection setup.
func IsDialError(err error) bool {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}
	switch opErr.Op {
	case "dial", "read", "write":
		return true
	}
	return false
}

func isIPLiteral(s string) bool {
	_, err := netip.ParseAddr(s)
	return err == nil
}
