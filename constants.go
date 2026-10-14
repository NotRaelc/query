package query

import "time"

const (
	modernGrace          = 100 * time.Millisecond
	DefaultTimeout       = 5 * time.Second
	DefaultMaxConcurrent = 20
	DefaultMinecraftPort = 25565
	DefaultBedrockPort   = 19132
	DefaultSourcePort    = 27015

	// defaultReceiveLimit bounds how much mcutil reads from the socket.
	// Without it, mcutil's io.LimitReader(conn, 0) returns EOF immediately.
	defaultReceiveLimit int64 = 1 << 20 // 1 MiB
)
