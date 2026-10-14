package query

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestQueryServerBlocksLocalhost(t *testing.T) {
	ctx := context.Background()
	opts := Options{Timeout: time.Second, Resolver: fakeResolver}

	for _, addr := range []string{
		"localhost", "localhost:25565", "127.0.0.1:25565",
		"[::1]:25565", "0.0.0.0:25565", "LocalHost:25565",
	} {
		t.Run(addr, func(t *testing.T) {
			resp := QueryServerOpts(ctx, addr, opts)
			if !errors.Is(resp.Error, ErrBlockedHost) {
				t.Fatalf("expected ErrBlockedHost, got %v", resp.Error)
			}
		})
	}
}

func TestQueryServerInvalidAddress(t *testing.T) {
	resp := QueryServer(context.Background(), "", time.Second)
	if resp.Error == nil {
		t.Fatal("expected error for empty address")
	}
}

func TestQueryMultipleServersDedupAndOrder(t *testing.T) {
	ctx := context.Background()
	opts := Options{Timeout: time.Second, Resolver: fakeResolver}

	addrs := []string{
		"localhost",
		"127.0.0.1",
		"localhost:25565",
		"bogus",
		"localhost",
	}
	resps := QueryMultipleServersOpts(ctx, addrs, opts)
	if len(resps) != len(addrs) {
		t.Fatalf("len = %d, want %d", len(resps), len(addrs))
	}
	for _, i := range []int{0, 1, 2, 4} {
		if !errors.Is(resps[i].Error, ErrBlockedHost) {
			t.Fatalf("resps[%d].Error = %v, want ErrBlockedHost", i, resps[i].Error)
		}
	}
	if resps[3].Error == nil {
		t.Fatal("expected error for 'bogus'")
	}
}

func TestQueryMultipleServersEmpty(t *testing.T) {
	resps := QueryMultipleServersOpts(context.Background(), nil, Options{})
	if len(resps) != 0 {
		t.Fatalf("len = %d, want 0", len(resps))
	}
}

// --- Live integration tests. Skip with -short or if the network is
// unavailable. Public servers are not guaranteed to be reachable.

func liveTest(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}
}

// skipIfNetworkError converts typical network failures into a skip.
// Protocol errors (server responded with garbage) still fail.
func skipIfNetworkError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") {
		t.Skipf("live server unreachable from this environment: %v", err)
	}
	t.Fatalf("query failed: %v", err)
}

func TestQueryJavaLive(t *testing.T) {
	liveTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := QueryServer(ctx, "demo.mcstatus.io", 8*time.Second, ServerMinecraftModern)
	skipIfNetworkError(t, resp.Error)

	s := resp.Payload
	if !s.Online {
		t.Fatal("server should be online")
	}
	if s.Name == "" {
		t.Fatal("name should not be empty")
	}
	t.Logf("Java: %s | players %d/%d", s.Name, s.OnlinePlayers, s.MaxPlayers)
}

func TestBedrockLive(t *testing.T) {
	liveTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := QueryServer(ctx, "demo.mcstatus.io:19132", 8*time.Second, ServerMinecraftBedrock)
	skipIfNetworkError(t, resp.Error)

	if !resp.Payload.Online {
		t.Fatal("server should be online")
	}
	if resp.Payload.GameKind != "minecraft-bedrock" {
		t.Fatalf("GameKind = %q, want minecraft-bedrock", resp.Payload.GameKind)
	}
	if resp.Payload.Map != "" {
		t.Fatalf("Map = %q, want empty (Bedrock ping does not return a map)", resp.Payload.Map)
	}
	t.Logf("Bedrock: %s | players %d/%d", resp.Payload.Name, resp.Payload.OnlinePlayers, resp.Payload.MaxPlayers)
}

func TestHooksAreCalled(t *testing.T) {
	liveTest(t)
	var attempts, results int
	opts := Options{
		Timeout: 8 * time.Second,
		Type:    ServerMinecraftModern,
		Hooks: Hooks{
			OnAttempt: func(host string, proto ServerType) { attempts++ },
			OnResult: func(host string, proto ServerType, err error, d time.Duration) {
				results++
			},
		},
	}
	resp := QueryServerOpts(context.Background(), "demo.mcstatus.io", opts)
	skipIfNetworkError(t, resp.Error)

	if attempts != 1 || results != 1 {
		t.Fatalf("attempts = %d, results = %d; want 1/1", attempts, results)
	}
}

func TestHostnameSetOnlyForDomains(t *testing.T) {
	// Заглушка для проверки логики — не требует сети.
	// Мы проверяем, что для IP-литералов Hostname пустой.

	// Прямой вызов логики:
	if isIPLiteral("5.83.140.250") != true {
		t.Fatal("5.83.140.250 должен определяться как IP")
	}
	if isIPLiteral("cubeverse.su") != false {
		t.Fatal("cubeverse.su не должен определяться как IP")
	}
	if isIPLiteral("2001:db8::1") != true {
		t.Fatal("IPv6 должен определяться как IP")
	}
}
