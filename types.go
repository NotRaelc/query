package query

import (
	"encoding/json"
	"errors"
	"time"
)

// Response wraps a query result. Error is nil on success.
type Response struct {
	Error   error  `json:"-"`
	Payload Server `json:"payload"`
}

func (r Response) MarshalJSON() ([]byte, error) {
	errStr := ""
	if r.Error != nil {
		errStr = r.Error.Error()
	}
	return json.Marshal(struct {
		Error   string `json:"error,omitempty"`
		Payload Server `json:"payload"`
	}{Error: errStr, Payload: r.Payload})
}

// Server is the unified representation of a game server.
// Unpopulated fields use their zero value; numeric unknowns use -1.
type Server struct {
	Online   bool     `json:"online"`
	Name     string   `json:"name"`
	Map      string   `json:"map"`
	GameID   int64    `json:"game_id"`
	Protocol int64    `json:"protocol"`
	Versions []string `json:"versions"`

	IP       string `json:"ip"`
	Port     uint16 `json:"port"`
	Hostname string `json:"hostname,omitempty"`

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

type MOTD struct {
	Raw   []string `json:"raw,omitempty"`
	Clean []string `json:"clean,omitempty"`
	HTML  []string `json:"html,omitempty"`
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
	ID    string `json:"id"`
	Time  int64  `json:"time"`
	Score int32  `json:"score,omitempty"`
	Index uint8  `json:"index"`
}

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

// Options configures a single or batch query. Zero values are replaced
// with defaults by withDefaults.
type Options struct {
	Timeout       time.Duration
	Type          ServerType
	NoIcon        bool
	NoPlayers     bool
	MaxConcurrent int
	Hooks         Hooks
	Resolver      Resolver
}

// Hooks receives protocol-level callbacks. All fields are optional.
type Hooks struct {
	// OnAttempt is called before each protocol request.
	OnAttempt func(host string, proto ServerType)
	// OnResult is called after each protocol request, success or failure.
	OnResult func(host string, proto ServerType, err error, d time.Duration)
}

// ProtocolError wraps an error with the protocol that produced it.
// Use errors.As to extract the ServerType:
//
//	var pe *query.ProtocolError
//	if errors.As(err, &pe) { ... pe.Proto ... }
type ProtocolError struct {
	Proto ServerType
	Err   error
}

func (e *ProtocolError) Error() string { return e.Proto.String() + ": " + e.Err.Error() }
func (e *ProtocolError) Unwrap() error { return e.Err }

const (
	modernGrace          = 100 * time.Millisecond
	DefaultTimeout       = 5 * time.Second
	DefaultMaxConcurrent = 20
	DefaultMinecraftPort = 25565
	DefaultBedrockPort   = 19132
)

var (
	ErrTimeout        = errors.New("query timeout exceeded")
	ErrProtocolFailed = errors.New("protocol query failed")
	// ErrBlockedHost is returned when the target is localhost,
	// loopback, or an unspecified address.
	ErrBlockedHost = errors.New("query: host is blocked")
)
