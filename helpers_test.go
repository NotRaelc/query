package query

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/mcstatus-io/mcutil/v4/formatting"
	"github.com/mcstatus-io/mcutil/v4/response"
)

// fakeResolver returns a single TEST-NET-1 address (RFC 5737), so
// network is never touched and isBlocked is driven only by the literal
// host string.
func fakeResolver(_ context.Context, _ string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
}

// loopbackResolver always resolves to 127.0.0.1, for testing the
// DNS-based block path.
func loopbackResolver(_ context.Context, _ string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
}

// buildModernFixture builds a StatusModern response with the given
// player counts and sample size. Sample names are deterministic.
func buildModernFixture(online, max int64, sampleCount int) *response.StatusModern {
	sample := make([]response.SamplePlayer, sampleCount)
	for i := range sample {
		sample[i] = response.SamplePlayer{
			ID:   fmt.Sprintf("uuid-%d", i),
			Name: formatting.Result{Clean: fmt.Sprintf("Player%d", i)},
		}
	}
	return &response.StatusModern{
		Version: response.Version{
			Name:     formatting.Result{Clean: "1.20.4"},
			Protocol: 765,
		},
		Players: response.Players{
			Online: &online,
			Max:    &max,
			Sample: sample,
		},
		MOTD: formatting.Result{Raw: "§aHi", Clean: "Hi", HTML: "Hi"},
	}
}

// contains is a tiny substring helper to avoid importing strings.
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
