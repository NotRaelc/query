package query

import (
	"context"
	"testing"
	"time"
)

func TestGraceOr(t *testing.T) {
	if got := graceOr(context.Background(), time.Second); got != time.Second {
		t.Fatalf("no deadline: got %s, want %s", got, time.Second)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	got := graceOr(ctx, time.Second)
	if got <= 0 || got > 50*time.Millisecond {
		t.Fatalf("with deadline: got %s, want <= 50ms", got)
	}
}

func TestHooksZeroValueNoPanic(t *testing.T) {
	var h Hooks
	h.attempt("x", ServerSource)
	h.result("x", ServerSource, nil, 0)
}

func TestPortOrDefault(t *testing.T) {
	tests := []struct {
		in, def, want uint16
	}{
		{0, 25565, 25565},
		{8080, 25565, 8080},
		{0, 19132, 19132},
	}
	for _, tt := range tests {
		if got := portOrDefault(tt.in, tt.def); got != tt.want {
			t.Fatalf("portOrDefault(%d, %d) = %d, want %d", tt.in, tt.def, got, tt.want)
		}
	}
}

func TestDefaultPortFor(t *testing.T) {
	tests := []struct {
		in   ServerType
		want uint16
	}{
		{ServerMinecraftBedrock, DefaultBedrockPort},
		{ServerSource, DefaultSourcePort},
		{ServerMinecraftModern, DefaultMinecraftPort},
		{ServerMinecraftLegacy, DefaultMinecraftPort},
		{ServerAuto, DefaultMinecraftPort},
	}
	for _, tt := range tests {
		if got := defaultPortFor(tt.in); got != tt.want {
			t.Fatalf("defaultPortFor(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
