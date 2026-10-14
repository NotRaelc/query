package query

import (
	"context"
	"fmt"
	"net"
	"net/netip"
)

// Resolver resolves a hostname to IP addresses ordered by preference:
// IPv4 first, then IPv6.
type Resolver func(ctx context.Context, host string) ([]netip.Addr, error)

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
