package query

import (
	"context"
	"net/netip"
	"testing"
)

func FuzzSplitHostPort(f *testing.F) {
	seeds := []string{
		"127.0.0.1:25565",
		"example.com",
		"[::1]:25565",
		":25565",
		"",
		"example.com:0",
		"[2001:db8::1]",
		"   ",
		"example.com:",
		"example.com:abc",
		"[::1]",
		"2001:db8::1:25565",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		host, port, err := SplitHostPort(s)
		if err != nil {
			if host != "" || port != 0 {
				t.Fatalf("SplitHostPort(%q) = (%q, %d, %v), want zero values on error",
					s, host, port, err)
			}
			return
		}
		// On success host must be non-empty. Port may be 0 when the
		// input had no port — callers apply their own default.
		if host == "" {
			t.Fatalf("SplitHostPort(%q) returned empty host without error", s)
		}
	})
}

func FuzzOptionsIsBlocked(f *testing.F) {
	seeds := []string{
		"localhost", "127.0.0.1", "::1", "0.0.0.0",
		"example.com", "192.0.2.1", "", "[::1]", "127.0.0.1%lo0",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, host string) {
		opts := Options{
			Resolver: func(_ context.Context, _ string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
			},
		}
		_ = opts.isBlocked(context.Background(), host)
	})
}

func FuzzStripMinecraftColors(f *testing.F) {
	f.Add("§aHello")
	f.Add("no codes")
	f.Add("§")
	f.Add("§§§")
	f.Fuzz(func(t *testing.T, s string) {
		_ = stripMinecraftColors(s)
	})
}

func FuzzHTMLFromMinecraft(f *testing.F) {
	f.Add("§aHello <script>")
	f.Add("plain")
	f.Fuzz(func(t *testing.T, s string) {
		_ = htmlFromMinecraft(s)
	})
}
