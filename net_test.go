package query

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		host    string
		port    uint16
		wantErr bool
	}{
		{"host with port", "example.com:25565", "example.com", 25565, false},
		{"host without port", "example.com", "example.com", 0, false},
		{"ipv4 with port", "127.0.0.1:19132", "127.0.0.1", 19132, false},
		{"ipv4 without port", "127.0.0.1", "127.0.0.1", 0, false},
		{"empty host with port", ":25565", "127.0.0.1", 25565, false},
		{"bracketed ipv6", "[2001:db8::1]:25565", "2001:db8::1", 25565, false},
		{"bracketed ipv6 no port", "[2001:db8::1]", "2001:db8::1", 0, false},
		{"bare ipv6 no port", "2001:db8::1", "2001:db8::1", 0, false},
		{"bare ipv6 with port", "[2001:db8::1]:25565", "2001:db8::1", 25565, false},

		{"empty string", "", "", 0, true},
		{"whitespace only", "   ", "", 0, true},
		{"tab only", "\t", "", 0, true},

		{"invalid port", "example.com:abc", "", 0, true},
		{"port overflow", "example.com:70000", "", 0, true},
		{"negative port", "example.com:-1", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := SplitHostPort(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SplitHostPort(%q) = (%q, %d, nil), want error", tt.in, host, port)
				}
				if host != "" || port != 0 {
					t.Fatalf("on error want zero values, got (%q, %d)", host, port)
				}
				return
			}
			if err != nil {
				t.Fatalf("SplitHostPort(%q) unexpected error: %v", tt.in, err)
			}
			if host != tt.host {
				t.Fatalf("host = %q, want %q", host, tt.host)
			}
			if port != tt.port {
				t.Fatalf("port = %d, want %d", port, tt.port)
			}
		})
	}
}

func TestResolveHostPrefersIPv4(t *testing.T) {
	addrs, err := ResolveHost(context.Background(), "127.0.0.1")
	if err != nil {
		t.Fatalf("ResolveHost: %v", err)
	}
	if len(addrs) != 1 || !addrs[0].Is4() {
		t.Fatalf("expected single IPv4, got %v", addrs)
	}
}

func TestResolvePreferredDirectIP(t *testing.T) {
	addr, err := ResolvePreferred(context.Background(), "2001:db8::1")
	if err != nil {
		t.Fatal(err)
	}
	if !addr.Is6() {
		t.Fatalf("expected IPv6, got %v", addr)
	}
}

func TestIsBlockedHost(t *testing.T) {
	ctx := context.Background()
	publicOpts := Options{Resolver: fakeResolver}
	loopbackOpts := Options{Resolver: loopbackResolver}

	tests := []struct {
		name    string
		host    string
		opts    Options
		blocked bool
	}{
		{"localhost literal", "localhost", publicOpts, true},
		{"localhost case", "LocalHost", publicOpts, true},
		{"loopback ipv4", "127.0.0.1", publicOpts, true},
		{"loopback ipv4 other", "127.1.2.3", publicOpts, true},
		{"loopback ipv6", "::1", publicOpts, true},
		{"unspecified ipv4", "0.0.0.0", publicOpts, true},
		{"unspecified ipv6", "::", publicOpts, true},
		{"public ip", "192.0.2.1", publicOpts, false},
		{"hostname resolving to public", "example.com", publicOpts, false},
		{"hostname resolving to loopback", "example.com", loopbackOpts, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.isBlocked(ctx, tt.host); got != tt.blocked {
				t.Fatalf("isBlocked(%q) = %v, want %v", tt.host, got, tt.blocked)
			}
		})
	}
}

func TestIsDialError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain", errors.New("plain"), false},
		{"dial op error", &net.OpError{Op: "dial"}, true},
		{"read op error", &net.OpError{Op: "read"}, true},
		{"write op error", &net.OpError{Op: "write"}, true},
		{"other op error", &net.OpError{Op: "close"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDialError(tt.err); got != tt.want {
				t.Fatalf("IsDialError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsBlockedHostRealResolver(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if !IsBlockedHost(ctx, "127.0.0.1") {
		t.Fatal("127.0.0.1 should be blocked")
	}
	if !IsBlockedHost(ctx, "localhost") {
		t.Fatal("localhost should be blocked")
	}
	if IsBlockedHost(ctx, "1.1.1.1") {
		t.Fatal("1.1.1.1 should not be blocked")
	}
}

func TestResolverTypeAcceptsFunction(t *testing.T) {
	var r Resolver = func(_ context.Context, _ string) ([]netip.Addr, error) {
		return nil, nil
	}
	_, _ = r(context.Background(), "x")
}
