package query

import "testing"

func TestProtocolVersionsModern(t *testing.T) {
	got := ProtocolVersions(47, false)
	if len(got) == 0 {
		t.Fatalf("expected at least one version for protocol 47, got %v", got)
	}
	if ProtocolVersions(9999, false) != nil {
		t.Fatal("expected nil for unknown protocol")
	}
}

func TestProtocolVersionsLegacy(t *testing.T) {
	got := ProtocolVersions(78, true)
	if len(got) == 0 {
		t.Fatalf("expected at least one legacy version for 78, got %v", got)
	}
	if ProtocolVersions(9999, true) != nil {
		t.Fatal("expected nil for unknown legacy protocol")
	}
}
