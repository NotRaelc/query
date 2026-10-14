package query

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"strings"
)

func isBlockedAddr(a netip.Addr) bool {
	return a.IsLoopback() || a.IsUnspecified()
}

// IsBlockedHost reports whether host should be rejected using the
// default resolver.
func IsBlockedHost(ctx context.Context, host string) bool {
	return Options{}.isBlocked(ctx, host)
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
	return slices.ContainsFunc(addrs, isBlockedAddr)
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
