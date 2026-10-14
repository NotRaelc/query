package query

import (
	"reflect"
	"strings"

	"github.com/NotRaelc/mcutil/v4/formatting"
	"github.com/NotRaelc/mcutil/v4/response"
	"github.com/woozymasta/a2s/pkg/a2s"
	"github.com/woozymasta/a2s/pkg/appid"
	"github.com/woozymasta/a2s/pkg/keywords"
)

// textField extracts a named string field from a value, following
// pointers. Returns "" when the field is absent or is not a string.
// Used to bridge differences between mcutil versions where Name may be
// either a string or a formatting.Result.
func textField(v any, name string) string {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	f := rv.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

// cleanText returns the clean (no §-codes) representation of a
// name-like value.
func cleanText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if r, ok := v.(formatting.Result); ok {
		return r.Clean
	}
	return textField(v, "Clean")
}

// rawText returns the raw (§-coded) representation.
func rawText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if r, ok := v.(formatting.Result); ok {
		return r.Raw
	}
	return textField(v, "Raw")
}

// htmlText returns the HTML representation.
func htmlText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if r, ok := v.(formatting.Result); ok {
		return r.HTML
	}
	return textField(v, "HTML")
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// motdFromFormatting converts mcutil's Formatting result to a MOTD.
func motdFromFormatting(r formatting.Result) *MOTD {
	raw := splitLines(rawText(r))
	clean := splitLines(cleanText(r))
	html := splitLines(htmlText(r))
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

// motdFromString converts a plain string (Source name) to a MOTD.
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
		Name:     cleanText(resp.Version.Name),
		GameID:   -1,
		Protocol: resp.Version.Protocol,
		Versions: ProtocolVersions(resp.Version.Protocol, false),
		MOTD:     motdFromFormatting(resp.MOTD),
		Players:  []Player{},
	}
	if s.Versions == nil {
		s.Versions = []string{}
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
				Name:  cleanText(p.Name),
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
		GameID:   -1,
		MOTD:     motdFromFormatting(resp.MOTD),
		Players:  []Player{},
		Versions: []string{},
	}
	if resp.Version != nil {
		s.Name = cleanText(resp.Version.Name)
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
		Map:           info.Map,
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
