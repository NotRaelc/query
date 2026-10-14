package query

import (
	"strconv"
	"strings"

	"github.com/NotRaelc/mcutil/v4/formatting"
	"github.com/NotRaelc/mcutil/v4/response"
	"github.com/woozymasta/a2s/pkg/a2s"
	"github.com/woozymasta/a2s/pkg/appid"
	"github.com/woozymasta/a2s/pkg/keywords"
)

func derefOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func motdFromFormatting(r formatting.Result) *MOTD {
	if r.Raw == "" && r.Clean == "" && r.HTML == "" {
		return nil
	}
	return &MOTD{
		Raw:   splitLines(r.Raw),
		Clean: splitLines(r.Clean),
		HTML:  splitLines(r.HTML),
	}
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

// newServer constructs a Server with the invariants that the public type
// relies on: non-nil slices, GameID=-1, Online=true.
func newServer(gameKind string, motd *MOTD, versions []string) Server {
	if versions == nil {
		versions = []string{}
	}
	return Server{
		GameKind: gameKind,
		Online:   true,
		GameID:   -1,
		MOTD:     motd,
		Players:  []Player{},
		Versions: versions,
	}
}

func serverFromModern(resp *response.StatusModern, opts Options) Server {
	s := newServer(
		"minecraft-java",
		motdFromFormatting(resp.MOTD),
		ProtocolVersions(resp.Version.Protocol, false),
	)
	s.Name = resp.Version.Name.Clean
	s.Protocol = resp.Version.Protocol
	if !opts.NoIcon && resp.Favicon != nil {
		s.Icon = *resp.Favicon
	}
	s.OnlinePlayers = derefOr(resp.Players.Online, 0)
	s.MaxPlayers = derefOr(resp.Players.Max, 0)
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
	var (
		name     string
		protocol int64
		versions []string
	)
	if resp.Version != nil {
		name = resp.Version.Name.Clean
		protocol = resp.Version.Protocol
		versions = ProtocolVersions(protocol, true)
	}
	s := newServer("minecraft-legacy", motdFromFormatting(resp.MOTD), versions)
	s.Name = name
	s.Protocol = protocol
	s.OnlinePlayers = resp.Players.Online
	s.MaxPlayers = resp.Players.Max
	return s
}

func serverFromBedrock(resp *response.StatusBedrock, opts Options) Server {
	versions := []string{}
	if resp.Version != nil {
		versions = []string{*resp.Version}
	}
	s := newServer("minecraft-bedrock", motdFromPtrFormatting(resp.MOTD), versions)
	if resp.Version != nil {
		s.Name = *resp.Version
	}
	s.Protocol = derefOr(resp.ProtocolVersion, 0)
	s.OnlinePlayers = derefOr(resp.OnlinePlayers, 0)
	s.MaxPlayers = derefOr(resp.MaxPlayers, 0)
	s.Gamemode = derefOr(resp.Gamemode, "")
	s.GamemodeID = derefOr(resp.GamemodeID, 0)
	s.ServerID = derefOr(resp.ServerID, "")
	s.Software = derefOr(resp.Edition, "")
	return s
}

func serverFromSource(info *a2s.Info, players []a2s.Player, opts Options) Server {
	s := newServer("source", motdFromString(info.Name), []string{info.Version})
	s.Name = info.Name
	s.Map = info.Map
	s.GameID = int64(info.EffectiveID())
	s.Protocol = int64(info.Protocol)
	s.OnlinePlayers = int64(info.Players)
	s.MaxPlayers = int64(info.MaxPlayers)
	s.Software = info.Game
	if info.SteamID != 0 {
		s.ServerID = strconv.FormatUint(info.SteamID, 10)
	}
	switch info.EffectiveID() {
	case appid.Arma3:
		arma := keywords.ParseArma3(info.Keywords)
		s.Keywords = arma
		if arma.GameType != "" {
			s.Gamemode = arma.GameType.String()
		}
		if arma.Island != "" {
			s.Map = arma.Island
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
