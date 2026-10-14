package query

import (
	"strings"

	"github.com/mcstatus-io/mcutil/v4/formatting"
	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/woozymasta/a2s/pkg/a2s"
	"github.com/woozymasta/a2s/pkg/appid"
	"github.com/woozymasta/a2s/pkg/keywords"
)

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func motdFromFormatting(r formatting.Result) *MOTD {
	raw := splitLines(r.Raw)
	clean := splitLines(r.Clean)
	html := splitLines(r.HTML)
	if raw == nil && clean == nil && html == nil {
		return nil
	}
	return &MOTD{Raw: raw, Clean: clean, HTML: html}
}

func motdFromPtrFormatting(r *formatting.Result) *MOTD {
	if r == nil {
		return nil
	}
	return motdFromFormatting(*r)
}

func motdFromString(s string) *MOTD {
	if s == "" {
		return nil
	}
	return &MOTD{
		Raw:   []string{s},
		Clean: []string{stripMinecraftColors(s)},
		HTML:  []string{htmlFromMinecraft(s)},
	}
}

func serverFromModern(resp *response.StatusModern, opts Options) Server {
	s := Server{
		GameKind: "minecraft-java",
		Online:   true,
		Name:     resp.Version.Name.Clean,
		// Map is intentionally empty: the Java ping protocol does not
		// report the loaded map name.
		GameID:   -1,
		Protocol: resp.Version.Protocol,
		Versions: ProtocolVersions(resp.Version.Protocol, false),
		MOTD:     motdFromFormatting(resp.MOTD),
		Players:  []Player{},
	}
	if !opts.NoIcon && resp.Favicon != nil {
		s.Icon = *resp.Favicon
	}
	if resp.Players.Online != nil {
		s.OnlinePlayers = *resp.Players.Online
	}
	if resp.Players.Max != nil {
		s.MaxPlayers = *resp.Players.Max
	}
	if resp.Mods != nil {
		s.Software = resp.Mods.Type
		for _, m := range resp.Mods.List {
			s.Mods = append(s.Mods, Mod{Name: m.ID, Version: m.Version})
		}
	}
	if !opts.NoPlayers {
		for i, p := range resp.Players.Sample {
			s.Players = append(s.Players, Player{
				Name:  p.Name.Clean,
				ID:    p.ID,
				Index: uint8(i),
				Time:  -1,
			})
		}
	}
	return s
}

func serverFromLegacy(resp *response.StatusLegacy, opts Options) Server {
	s := Server{
		GameKind: "minecraft-legacy",
		Online:   true,
		// Map is intentionally empty: the legacy ping protocol does not
		// report the loaded map name.
		GameID:   -1,
		MOTD:     motdFromFormatting(resp.MOTD),
		Players:  []Player{},
		Versions: []string{},
	}
	if resp.Version != nil {
		s.Name = resp.Version.Name.Clean
		s.Protocol = resp.Version.Protocol
		s.Versions = ProtocolVersions(resp.Version.Protocol, true)
		if s.Versions == nil {
			s.Versions = []string{}
		}
	}
	s.OnlinePlayers = resp.Players.Online
	s.MaxPlayers = resp.Players.Max
	return s
}

func serverFromBedrock(resp *response.StatusBedrock, opts Options) Server {
	s := Server{
		GameKind: "minecraft-bedrock",
		Online:   true,
		// Map is intentionally empty: the Bedrock ping protocol does not
		// report the loaded map name.
		GameID:   -1,
		MOTD:     motdFromPtrFormatting(resp.MOTD),
		Players:  []Player{},
		Versions: []string{},
	}
	if resp.Version != nil {
		s.Name = *resp.Version
		s.Versions = []string{*resp.Version}
	}
	if resp.ProtocolVersion != nil {
		s.Protocol = *resp.ProtocolVersion
	}
	if resp.OnlinePlayers != nil {
		s.OnlinePlayers = *resp.OnlinePlayers
	}
	if resp.MaxPlayers != nil {
		s.MaxPlayers = *resp.MaxPlayers
	}
	if resp.Gamemode != nil {
		s.Gamemode = *resp.Gamemode
	}
	if resp.GamemodeID != nil {
		s.GamemodeID = *resp.GamemodeID
	}
	if resp.ServerID != nil {
		s.ServerID = *resp.ServerID
	}
	if resp.Edition != nil {
		s.Software = *resp.Edition
	}
	return s
}

func serverFromSource(info *a2s.Info, players []a2s.Player, opts Options) Server {
	s := Server{
		GameKind:      "source",
		Online:        true,
		Name:          info.Name,
		Map:           info.Map, // real map name, e.g. "de_dust2"
		GameID:        int64(info.EffectiveID()),
		Protocol:      int64(info.Protocol),
		Versions:      []string{info.Version},
		OnlinePlayers: int64(info.Players),
		MaxPlayers:    int64(info.MaxPlayers),
		MOTD:          motdFromString(info.Name),
		Software:      info.Game,
		Players:       []Player{},
	}
	if info.SteamID != 0 {
		s.ServerID = formatUint(info.SteamID)
	}

	// Game-specific keyword parsing for known AppIDs.
	switch info.EffectiveID() {
	case appid.Arma3:
		arma := keywords.ParseArma3(info.Keywords)
		s.Keywords = arma
		if arma.GameType != "" {
			s.Gamemode = arma.GameType.String()
		}
		if arma.Island != "" {
			s.Map = arma.Island // Island overrides the generic map for Arma 3.
		}
	case appid.DayZ, appid.DayZExperimental:
		dayz := keywords.ParseDayZ(info.Keywords)
		s.Keywords = dayz
		if dayz.Shard != "" {
			s.ServerID = dayz.Shard
		}
		if dayz.GamePort != 0 {
			s.Port = dayz.GamePort
		}
	}

	if !opts.NoPlayers {
		for _, p := range players {
			s.Players = append(s.Players, Player{
				Name:  p.Name,
				Index: p.Index,
				Time:  int64(p.Duration.Seconds()),
				Score: p.Score,
				ID:    "-1",
			})
		}
	}
	return s
}

// formatUint formats an unsigned integer without importing strconv here.
func formatUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
