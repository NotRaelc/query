package query

import (
	"context"
	"testing"
)

func BenchmarkSplitHostPort(b *testing.B) {
	inputs := []string{
		"example.com:25565",
		"127.0.0.1",
		"[2001:db8::1]:25565",
		"localhost",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = SplitHostPort(inputs[i%len(inputs)])
	}
}

func BenchmarkStripMinecraftColors(b *testing.B) {
	s := "§aHello §lWorld§r! §cRed §7Gray"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stripMinecraftColors(s)
	}
}

func BenchmarkHTMLFromMinecraft(b *testing.B) {
	s := "§aHello §lWorld§r! §cRed §7Gray"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = htmlFromMinecraft(s)
	}
}

func BenchmarkIsBlockedHost(b *testing.B) {
	opts := Options{Resolver: fakeResolver}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = opts.isBlocked(ctx, "example.com")
	}
}

func BenchmarkServerFromModern(b *testing.B) {
	resp := buildModernFixture(5, 20, 12)
	opts := Options{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = serverFromModern(resp, opts)
	}
}

func BenchmarkResponseMarshalJSON(b *testing.B) {
	resp := Response{
		Payload: Server{Name: "test", OnlinePlayers: 5, MaxPlayers: 20},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resp.MarshalJSON()
	}
}
