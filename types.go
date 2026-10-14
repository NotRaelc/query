package query

import (
	"encoding/json"
	"time"
)

// ServerType identifies the query protocol.
type ServerType int

const (
	ServerAuto ServerType = iota
	ServerSource
	ServerMinecraftModern
	ServerMinecraftLegacy
	ServerMinecraftBedrock
)

// String returns the short protocol name used in logs and cache keys.
func (t ServerType) String() string {
	switch t {
	case ServerSource:
		return "source"
	case ServerMinecraftModern:
		return "modern"
	case ServerMinecraftLegacy:
		return "legacy"
	case ServerMinecraftBedrock:
		return "bedrock"
	default:
		return "auto"
	}
}

// Options configures a single or batch query. The zero value is valid.
//
// Timeout is not part of Options. It is taken from ctx.Deadline(). If
// ctx has no deadline, DefaultTimeout is used.
type Options struct {
	// Type selects the protocol. Zero means ServerAuto.
	Type ServerType

	// NoIcon omits Server.Icon from the result.
	NoIcon bool

	// NoPlayers omits Server.Players from the result.
	NoPlayers bool

	// MaxConcurrent bounds parallel queries in a batch.
	// Zero uses DefaultMaxConcurrent.
	MaxConcurrent int

	// AllowLocalhost disables the loopback/unspecified check.
	// Intended for local testing only. Never enable on a public API.
	AllowLocalhost bool

	// Hooks receives protocol-level callbacks. All fields are optional.
	Hooks Hooks

	// Resolver overrides the default DNS resolver. Nil means the system one.
	Resolver Resolver
}

// Hooks receives protocol-level callbacks.
type Hooks struct {
	// OnAttempt is called before each protocol request.
	OnAttempt func(host string, proto ServerType)

	// OnResult is called after each protocol request, success or failure.
	OnResult func(host string, proto ServerType, err error, d time.Duration)
}

// Response wraps a query result.
//
// On success Error is nil and Payload contains the server data.
// On failure Error is non-nil and Payload is the zero value.
type Response struct {
	Error   error  `json:"-"`
	Payload Server `json:"payload"`
}

// MarshalJSON emits either {"payload": ...} on success or {"error": "..."}
// on failure. The Payload field is omitted when Error is set.
func (r Response) MarshalJSON() ([]byte, error) {
	if r.Error != nil {
		return json.Marshal(struct {
			Error string `json:"error"`
		}{Error: r.Error.Error()})
	}
	return json.Marshal(struct {
		Payload Server `json:"payload"`
	}{Payload: r.Payload})
}

// Server is the unified representation of a game server. Unpopulated
// fields use their zero value; numeric unknowns use -1. Slice fields are
// always non-nil in JSON, so consumers do not need nil checks.
type Server struct {
	// GameKind is one of "minecraft-java", "minecraft-legacy",
	// "minecraft-bedrock", "source".
	GameKind string `json:"game_kind,omitempty"`

	Online   bool     `json:"online"`
	Name     string   `json:"name"`
	Map      string   `json:"map"` // real map name; empty for Minecraft ping
	GameID   int64    `json:"game_id"`
	Protocol int64    `json:"protocol"`
	Versions []string `json:"versions"`

	IP       string `json:"ip"`
	Port     uint16 `json:"port"`
	Hostname string `json:"hostname,omitempty"` // set only for domain inputs

	Players       []Player `json:"players"`
	OnlinePlayers int64    `json:"online_players"`
	MaxPlayers    int64    `json:"max_players"`

	Icon        string     `json:"icon,omitempty"`
	Software    string     `json:"software,omitempty"`
	Gamemode    string     `json:"gamemode,omitempty"`
	GamemodeID  int64      `json:"gamemode_id,omitempty"`
	ServerID    string     `json:"serverid,omitempty"`
	EulaBlocked bool       `json:"eula_blocked,omitempty"`
	MOTD        *MOTD      `json:"motd,omitempty"`
	Plugins     []Plugin   `json:"plugins,omitempty"`
	Mods        []Mod      `json:"mods,omitempty"`
	Info        *MOTD      `json:"info,omitempty"`
	Keywords    any        `json:"keywords,omitempty"`
	Debug       *DebugInfo `json:"debug,omitempty"`
}

// MOTD holds the message of the day in three forms. Each slice element
// is one line.
type MOTD struct {
	Raw   []string `json:"raw,omitempty"`   // original with §-codes
	Clean []string `json:"clean,omitempty"` // plain text
	HTML  []string `json:"html,omitempty"`  // with <span style="color: ...">
}

type Plugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Mod struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type DebugInfo struct {
	Ping          bool  `json:"ping"`
	Query         bool  `json:"query"`
	Bedrock       bool  `json:"bedrock"`
	SRV           bool  `json:"srv"`
	QueryMismatch bool  `json:"querymismatch"`
	AnimatedMOTD  bool  `json:"animatedmotd"`
	CacheHit      bool  `json:"cachehit"`
	CacheTime     int64 `json:"cachetime"`
	CacheExpire   int64 `json:"cacheexpire"`
	APIVersion    int   `json:"apiversion"`
}

// Player is one player entry. Time is in seconds; -1 means unknown.
type Player struct {
	Name  string `json:"name"`
	ID    string `json:"id"` // UUID for Java, "-1" for Source
	Time  int64  `json:"time"`
	Score int32  `json:"score,omitempty"`
	Index uint8  `json:"index"`
}
