package query

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// withDefaults fills zero fields with their defaults.
func (o Options) withDefaults() Options {
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.MaxConcurrent <= 0 {
		o.MaxConcurrent = DefaultMaxConcurrent
	}
	return o
}

// QueryServer is the main entry point. If no ServerType is provided,
// ServerAuto is used. localhost and loopback addresses are rejected.
func QueryServer(ctx context.Context, addrStr string, timeout time.Duration, st ...ServerType) Response {
	opts := Options{Timeout: timeout}
	if len(st) > 0 {
		opts.Type = st[0]
	}
	return QueryServerOpts(ctx, addrStr, opts)
}

// QueryServerOpts is the options-aware variant of QueryServer.
func QueryServerOpts(ctx context.Context, addrStr string, opts Options) Response {
	opts = opts.withDefaults()

	host, port, err := SplitHostPort(addrStr)
	if err != nil {
		return Response{Error: err}
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if opts.isBlocked(ctx, host) {
		return Response{Error: fmt.Errorf("%w: %q", ErrBlockedHost, host)}
	}

	var s Server
	switch opts.Type {
	case ServerSource:
		s, err = querySource(ctx, host, port, opts)
	case ServerMinecraftModern:
		s, err = queryMinecraftModern(ctx, host, port, opts)
	case ServerMinecraftLegacy:
		s, err = queryMinecraftLegacy(ctx, host, port, opts)
	case ServerMinecraftBedrock:
		s, err = queryMinecraftBedrock(ctx, host, port, opts)
	default:
		return queryAuto(ctx, host, port, opts)
	}
	if err == nil {
		if s.IP == "" {
			s.IP = host
		}
		s.Hostname = host
		s.Port = port
	}
	return Response{Error: err, Payload: s}
}

// QueryMultipleServers queries many addresses in parallel, preserving
// the input order. Duplicates are queried once; the same Response is
// returned for every duplicate slot.
func QueryMultipleServers(ctx context.Context, addrStrings []string, timeout time.Duration, st ...ServerType) []Response {
	opts := Options{Timeout: timeout}
	if len(st) > 0 {
		opts.Type = st[0]
	}
	return QueryMultipleServersOpts(ctx, addrStrings, opts)
}

// QueryMultipleServersOpts is the options-aware variant.
func QueryMultipleServersOpts(ctx context.Context, addrStrings []string, opts Options) []Response {
	opts = opts.withDefaults()

	out := make([]Response, len(addrStrings))

	// Group inputs by (host, port, kind). Invalid entries are recorded
	// immediately and skipped from scheduling.
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
