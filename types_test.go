package query

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestOptionsWithDefaults(t *testing.T) {
	tests := []struct {
		name    string
		in      Options
		wantMax int
	}{
		{"zero", Options{}, DefaultMaxConcurrent},
		{"explicit", Options{MaxConcurrent: 5}, 5},
		{"negative", Options{MaxConcurrent: -1}, DefaultMaxConcurrent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.withDefaults()
			if got.MaxConcurrent != tt.wantMax {
				t.Fatalf("MaxConcurrent = %d, want %d", got.MaxConcurrent, tt.wantMax)
			}
		})
	}
}

func TestServerTypeString(t *testing.T) {
	tests := []struct {
		in   ServerType
		want string
	}{
		{ServerAuto, "auto"},
		{ServerSource, "source"},
		{ServerMinecraftModern, "modern"},
		{ServerMinecraftLegacy, "legacy"},
		{ServerMinecraftBedrock, "bedrock"},
		{ServerType(99), "auto"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Fatalf("ServerType(%d).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestProtocolErrorUnwrap(t *testing.T) {
	inner := errors.New("dial failed")
	pe := &ProtocolError{Proto: ServerSource, Err: inner}

	if pe.Error() != "source: dial failed" {
		t.Fatalf("Error() = %q, want %q", pe.Error(), "source: dial failed")
	}
	if !errors.Is(pe, inner) {
		t.Fatal("errors.Is should unwrap to inner error")
	}
	var got *ProtocolError
	if !errors.As(pe, &got) {
		t.Fatal("errors.As should extract *ProtocolError")
	}
	if got.Proto != ServerSource {
		t.Fatalf("Proto = %v, want %v", got.Proto, ServerSource)
	}
}

func TestResponseMarshalJSONSuccess(t *testing.T) {
	resp := Response{
		Payload: Server{Name: "test", OnlinePlayers: 5, MaxPlayers: 20},
	}
	data, err := resp.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(data), `"error"`) {
		t.Fatalf("success response should not contain an error field: %s", data)
	}
	if !contains(string(data), `"name":"test"`) {
		t.Fatalf("expected name field: %s", data)
	}
}

func TestResponseMarshalJSONError(t *testing.T) {
	resp := Response{Error: errors.New("boom")}
	data, err := resp.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), `"error":"boom"`) {
		t.Fatalf("unexpected JSON: %s", data)
	}
	if contains(string(data), `"payload"`) {
		t.Fatalf("error response should not contain payload: %s", data)
	}
}

func TestPlayerTimeIsSeconds(t *testing.T) {
	p := Player{Name: "x", Time: 60}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), `"time":60`) {
		t.Fatalf("expected time in seconds, got: %s", data)
	}
}
