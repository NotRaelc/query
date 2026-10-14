package query

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
)

func (o Options) withDefaults() Options {
	if o.MaxConcurrent <= 0 {
		o.MaxConcurrent = DefaultMaxConcurrent
	}
	return o
}

// QueryServer queries one game server. The timeout is taken from ctx.
// If ctx has no deadline, DefaultTimeout is applied.
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	resp := query.QueryServer(ctx, "play.example.com")
//	resp = query.QueryServer(ctx, "play.example.com", query.ServerSource)
func QueryServer(ctx context.Context, addrStr string, st ...ServerType) Response {
	opts := Options{}
	if len(st) > 0 {
		opts.Type = st[0]
	}
	return QueryServerOpts(ctx, addrStr, opts)
}

// QueryServerOpts is the options-aware variant of QueryServer.
func QueryServerOpts(ctx context.Context, addrStr string, opts Options) Response {
	ctx, cancel := ensureDeadline(ctx)
	defer cancel()

	opts = opts.withDefaults()

	host, port, err := SplitHostPort(addrStr)
	if err != nil {
		return Response{Error: err}
	}

	if opts.isBlocked(ctx, host) {
		return Response{Error: fmt.Errorf("%w: %q", ErrBlockedHost, host)}
	}

	if opts.Type == ServerAuto {
		return queryAuto(ctx, host, port, opts)
	}

	fn, ok := protocols[opts.Type]
	if !ok {
		return Response{Error: fmt.Errorf("query: unknown server type: %d", opts.Type)}
	}

	s, err := fn(ctx, host, port, opts)
	if err != nil {
		return Response{Error: err}
	}
	normalizeServer(&s, host, port, opts.Type)
	return Response{Payload: s}
}

// normalizeServer fills IP / Hostname / Port on a Server if the protocol
// layer did not. The proto argument is used to pick the default port.
func normalizeServer(s *Server, host string, port uint16, proto ServerType) {
	if s.IP == "" {
		s.IP = host
	}
	if !isIPLiteral(host) {
		s.Hostname = host
	}
	if s.Port == 0 {
		s.Port = portOrDefault(port, defaultPortFor(proto))
	}
}

// detectType maps a GameKind string back to a ServerType. Used by the
// auto path, where the winning protocol is not known in advance.
func detectType(gameKind string) ServerType {
	switch gameKind {
	case "minecraft-java":
		return ServerMinecraftModern
	case "minecraft-legacy":
		return ServerMinecraftLegacy
	case "minecraft-bedrock":
		return ServerMinecraftBedrock
	case "source":
		return ServerSource
	default:
		return ServerAuto
	}
}

// QueryMultipleServers queries many game servers in parallel, preserving
// the input order. Duplicates are queried once; the same Response is
// returned for every duplicate slot.
func QueryMultipleServers(ctx context.Context, addrStrings []string, st ...ServerType) []Response {
	opts := Options{}
	if len(st) > 0 {
		opts.Type = st[0]
	}
	return QueryMultipleServersOpts(ctx, addrStrings, opts)
}

// QueryMultipleServersOpts is the options-aware variant.
func QueryMultipleServersOpts(ctx context.Context, addrStrings []string, opts Options) []Response {
	ctx, cancel := ensureDeadline(ctx)
	defer cancel()

	opts = opts.withDefaults()

	out := make([]Response, len(addrStrings))

	type key struct {
		host string
		port uint16
		kind ServerType
	}
	type group struct {
		k    key
		idxs []int
	}
	groups := make(map[key]*group)
	order := make([]key, 0, len(addrStrings))

	for i, a := range addrStrings {
		host, port, err := SplitHostPort(a)
		if err != nil {
			out[i] = Response{Error: err}
			continue
		}
		k := key{host, port, opts.Type}
		g, ok := groups[k]
		if !ok {
			g = &group{k: k}
			groups[k] = g
			order = append(order, k)
		}
		g.idxs = append(g.idxs, i)
	}

	results := make(map[key]Response, len(order))
	var mu sync.Mutex
	sem := make(chan struct{}, opts.MaxConcurrent)
	var wg sync.WaitGroup

	for _, k := range order {
		k := k
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			addr := net.JoinHostPort(k.host, strconv.Itoa(int(k.port)))
			// Port 0 means "not specified" — the JoinHostPort round-trip
			// would lose that, so reconstruct the bare host.
			if k.port == 0 {
				addr = k.host
			}
			perCall := opts
			perCall.Type = k.kind
			resp := QueryServerOpts(ctx, addr, perCall)

			mu.Lock()
			results[k] = resp
			mu.Unlock()
		}()
	}
	wg.Wait()

	for _, k := range order {
		r := results[k]
		for _, i := range groups[k].idxs {
			out[i] = r
		}
	}
	return out
}

// ensureDeadline returns a context with a deadline. If ctx already has
// one, it is returned unchanged. Otherwise, DefaultTimeout is applied.
func ensureDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, DefaultTimeout)
}

func defaultPortFor(t ServerType) uint16 {
	switch t {
	case ServerMinecraftBedrock:
		return DefaultBedrockPort
	case ServerSource:
		return DefaultSourcePort
	default:
		return DefaultMinecraftPort
	}
}
