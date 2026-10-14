package query

import (
	"testing"

	"github.com/mcstatus-io/mcutil/v4/formatting"
	"github.com/mcstatus-io/mcutil/v4/response"
)

func TestMotdFromFormatting(t *testing.T) {
	r := formatting.Result{
		Raw:   "§aHello\n§bWorld",
		Clean: "Hello\nWorld",
		HTML:  `<span style="color: #55FF55">Hello</span>\n<span style="color: #55FFFF">World</span>`,
	}
	got := motdFromFormatting(r)
	if got == nil {
		t.Fatal("nil MOTD for non-empty input")
	}
	if len(got.Raw) != 2 || got.Raw[0] != "§aHello" {
		t.Fatalf("Raw = %v", got.Raw)
	}
	if len(got.Clean) != 2 || got.Clean[0] != "Hello" {
		t.Fatalf("Clean = %v", got.Clean)
	}
}

func TestMotdFromFormattingEmpty(t *testing.T) {
	if motdFromFormatting(formatting.Result{}) != nil {
		t.Fatal("expected nil for empty formatting")
	}
}

func TestMotdFromPtrFormatting(t *testing.T) {
	if motdFromPtrFormatting(nil) != nil {
		t.Fatal("expected nil for nil ptr")
	}
	r := &formatting.Result{Raw: "x", Clean: "x", HTML: "x"}
	if motdFromPtrFormatting(r) == nil {
		t.Fatal("expected non-nil for non-nil ptr")
	}
}

func TestMotdFromString(t *testing.T) {
	got := motdFromString("§aHello")
	if got == nil {
		t.Fatal("nil MOTD")
	}
	if len(got.Raw) != 1 || got.Raw[0] != "§aHello" {
		t.Fatalf("Raw = %v", got.Raw)
	}
	if len(got.Clean) != 1 || got.Clean[0] != "Hello" {
		t.Fatalf("Clean = %v", got.Clean)
	}
	if motdFromString("") != nil {
		t.Fatal("empty string should return nil")
	}
}

func TestServerFromModernMinimal(t *testing.T) {
	online := int64(5)
	max := int64(20)
	favicon := "data:image/png;base64,xxx"
	resp := &response.StatusModern{
		Version: response.Version{
			Name:     formatting.Result{Clean: "1.20.4"},
			Protocol: 765,
		},
		Players: response.Players{
			Online: &online,
			Max:    &max,
			Sample: []response.SamplePlayer{
				{ID: "uuid-1", Name: formatting.Result{Clean: "Alice"}},
				{ID: "uuid-2", Name: formatting.Result{Clean: "Bob"}},
			},
		},
		MOTD:    formatting.Result{Raw: "§aHi", Clean: "Hi", HTML: "Hi"},
		Favicon: &favicon,
	}
	s := serverFromModern(resp, Options{})
	if !s.Online || s.Name != "1.20.4" || s.Map != "Minecraft" {
		t.Fatalf("basic fields = %+v", s)
	}
	if s.OnlinePlayers != 5 || s.MaxPlayers != 20 {
		t.Fatalf("player counts = %d/%d", s.OnlinePlayers, s.MaxPlayers)
	}
	if len(s.Players) != 2 || s.Players[0].Name != "Alice" {
		t.Fatalf("players = %+v", s.Players)
	}
	if s.Players[0].Time != -1 {
		t.Fatalf("modern player time should be -1, got %d", s.Players[0].Time)
	}
	if s.Icon != favicon {
		t.Fatal("icon not preserved")
	}
}

func TestServerFromModernNoIcon(t *testing.T) {
	favicon := "data:image/png;base64,xxx"
	resp := &response.StatusModern{
		Version: response.Version{Name: formatting.Result{Clean: "x"}},
		Favicon: &favicon,
	}
	s := serverFromModern(resp, Options{NoIcon: true})
	if s.Icon != "" {
		t.Fatalf("icon should be stripped with NoIcon, got %q", s.Icon)
	}
}

func TestServerFromModernNoPlayers(t *testing.T) {
	resp := &response.StatusModern{
		Version: response.Version{Name: formatting.Result{Clean: "x"}},
		Players: response.Players{
			Sample: []response.SamplePlayer{
				{ID: "u", Name: formatting.Result{Clean: "Alice"}},
			},
		},
	}
	s := serverFromModern(resp, Options{NoPlayers: true})
	if len(s.Players) != 0 {
		t.Fatalf("players should be empty with NoPlayers, got %v", s.Players)
	}
}

func TestServerFromModernMods(t *testing.T) {
	resp := &response.StatusModern{
		Version: response.Version{Name: formatting.Result{Clean: "x"}},
		Mods: &response.ModInfo{
			Type: "Forge",
			List: []response.Mod{
				{ID: "jei", Version: "15.0"},
			},
		},
	}
	s := serverFromModern(resp, Options{})
	if s.Software != "Forge" {
		t.Fatalf("Software = %q, want Forge", s.Software)
	}
	if len(s.Mods) != 1 || s.Mods[0].Name != "jei" {
		t.Fatalf("Mods = %+v", s.Mods)
	}
}

func TestServerFromLegacy(t *testing.T) {
	resp := &response.StatusLegacy{
		Version: &response.Version{
			Name:     formatting.Result{Clean: "1.6.4"},
			Protocol: 78,
		},
		Players: response.LegacyPlayers{Online: 3, Max: 10},
		MOTD:    formatting.Result{Clean: "hello"},
	}
	s := serverFromLegacy(resp, Options{})
	if s.Name != "1.6.4" || s.OnlinePlayers != 3 || s.MaxPlayers != 10 {
		t.Fatalf("legacy server = %+v", s)
	}
	if s.Map != "MinecraftOld" {
		t.Fatalf("Map = %q, want MinecraftOld", s.Map)
	}
}

func TestServerFromLegacyNilVersion(t *testing.T) {
	resp := &response.StatusLegacy{
		Players: response.LegacyPlayers{Online: 1, Max: 2},
	}
	s := serverFromLegacy(resp, Options{})
	if s.Name != "" {
		t.Fatalf("Name should be empty for nil version, got %q", s.Name)
	}
}

func TestServerFromBedrock(t *testing.T) {
	version := "1.21.0"
	protocol := int64(649)
	gamemode := "Survival"
	gamemodeID := int64(1)
	serverID := "abc"
	edition := "MCPE"
	online := int64(4)
	max := int64(10)
	resp := &response.StatusBedrock{
		Version:         &version,
		ProtocolVersion: &protocol,
		Gamemode:        &gamemode,
		GamemodeID:      &gamemodeID,
		ServerID:        &serverID,
		Edition:         &edition,
		OnlinePlayers:   &online,
		MaxPlayers:      &max,
	}
	s := serverFromBedrock(resp, Options{})
	if s.Name != version || s.Protocol != protocol {
		t.Fatalf("bedrock fields = %+v", s)
	}
	if s.Gamemode != gamemode || s.GamemodeID != gamemodeID {
		t.Fatalf("gamemode = %q/%d", s.Gamemode, s.GamemodeID)
	}
	if s.ServerID != serverID || s.Software != edition {
		t.Fatalf("serverid/software = %q/%q", s.ServerID, s.Software)
	}
	if s.OnlinePlayers != online || s.MaxPlayers != max {
		t.Fatalf("counts = %d/%d", s.OnlinePlayers, s.MaxPlayers)
	}
	if s.Map != "MinecraftBedrock" {
		t.Fatalf("Map = %q", s.Map)
	}
}

func TestServerFromBedrockAllNil(t *testing.T) {
	s := serverFromBedrock(&response.StatusBedrock{}, Options{})
	if !s.Online || s.Name != "" || s.Protocol != 0 {
		t.Fatalf("all-nil bedrock = %+v", s)
	}
}
