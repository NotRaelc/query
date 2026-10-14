package query

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/NotRaelc/mcutil/v4/options"
	"github.com/NotRaelc/mcutil/v4/status"
	"github.com/woozymasta/a2s/pkg/a2s"
)

// queryFunc is the signature every protocol implementation must satisfy.
type queryFunc func(ctx context.Context, host string, port uint16, opts Options) (Server, error)

// protocols is the dispatch table for non-auto query types. Adding a new
// protocol only requires registering it here.
var protocols = map[ServerType]queryFunc{
	ServerSource:           querySource,
	ServerMinecraftModern:  queryMinecraftModern,
	ServerMinecraftLegacy:  queryMinecraftLegacy,
	ServerMinecraftBedrock: queryMinecraftBedrock,
}

func portOrDefault(port uint16, def uint16) uint16 {
	if port != 0 {
		return port
	}
	return def
}

// graceOr returns min(d, remaining time until ctx deadline), clamped to
// non-negative values.
func graceOr(ctx context.Context, d time.Duration) time.Duration {
	dl, ok := ctx.Deadline()
	if !ok {
		return d
	}
	return max(0, min(d, time.Until(dl)))
}

func (h Hooks) attempt(host string, proto ServerType) {
	if h.OnAttempt != nil {
		h.OnAttempt(host, proto)
	}
}

func (h Hooks) result(host string, proto ServerType, err error, d time.Duration) {
	if h.OnResult != nil {
		h.OnResult(host, proto, err, d)
	}
}

// runProtocol wraps a protocol call with hooks, timing and error tagging.
// This is the single place where attempt/result callbacks are fired and
// where protocol errors get wrapped with ProtocolError.
func runProtocol(host string, proto ServerType, opts Options, fn func() (Server, error)) (Server, error) {
	opts.Hooks.attempt(host, proto)
	start := time.Now()
	s, err := fn()
	opts.Hooks.result(host, proto, err, time.Since(start))
	if err != nil {
		return Server{}, &ProtocolError{Proto: proto, Err: err}
	}
	return s, nil
}

func queryMinecraftModern(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	port = portOrDefault(port, DefaultMinecraftPort)
	return runProtocol(host, ServerMinecraftModern, opts, func() (Server, error) {
		resp, err := status.Modern(ctx, host, port, options.StatusModern{
			EnableSRV:         true,
			ReceiveLimitBytes: defaultReceiveLimit,
			ProtocolVersion:   -1,
			Ping:              true,
		})
		if err != nil {
			return Server{}, err
		}
		return serverFromModern(resp, opts), nil
	})
}

func queryMinecraftLegacy(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	port = portOrDefault(port, DefaultMinecraftPort)
	return runProtocol(host, ServerMinecraftLegacy, opts, func() (Server, error) {
		resp, err := status.Legacy(ctx, host, port, options.StatusLegacy{
			EnableSRV:         true,
			ReceiveLimitBytes: defaultReceiveLimit,
			ProtocolVersion:   -1,
		})
		if err != nil {
			return Server{}, err
		}
		return serverFromLegacy(resp, opts), nil
	})
}

func queryMinecraftBedrock(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	port = portOrDefault(port, DefaultBedrockPort)
	return runProtocol(host, ServerMinecraftBedrock, opts, func() (Server, error) {
		resp, err := status.Bedrock(ctx, host, port, options.StatusBedrock{
			ReceiveLimitBytes: defaultReceiveLimit,
			ClientGUID:        rand.Int63(),
		})
		if err != nil {
			return Server{}, err
		}
		return serverFromBedrock(resp, opts), nil
	})
}

// queryMinecraft runs modern and legacy in parallel. Modern has priority:
// on success we return immediately. If legacy wins first we give modern
// modernGrace to complete, else return legacy's result.
func queryMinecraft(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type mcResult struct {
		s        Server
		err      error
		isModern bool
	}
	ch := make(chan mcResult, 2)

	go func() {
		s, err := queryMinecraftModern(ctx, host, port, opts)
		ch <- mcResult{s: s, err: err, isModern: true}
	}()
	go func() {
		s, err := queryMinecraftLegacy(ctx, host, port, opts)
		ch <- mcResult{s: s, err: err, isModern: false}
	}()

	first := <-ch
	if first.err == nil {
		if first.isModern {
			return first.s, nil
		}
		timer := time.NewTimer(graceOr(ctx, modernGrace))
		defer timer.Stop()
		select {
		case r := <-ch:
			if r.err == nil {
				return r.s, nil
			}
			return first.s, nil
		case <-timer.C:
			return first.s, nil
		}
	}

	second := <-ch
	if second.err == nil {
		return second.s, nil
	}
	if first.isModern {
		return Server{}, first.err
	}
	return Server{}, second.err
}

// sourceTimeout picks the effective socket timeout for a2s based on ctx:
// the smaller of DefaultTimeout and the remaining ctx budget.
func sourceTimeout(ctx context.Context) time.Duration {
	if dl, ok := ctx.Deadline(); ok {
		if remain := time.Until(dl); remain > 0 && remain < DefaultTimeout {
			return remain
		}
	}
	return DefaultTimeout
}

func querySource(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	port = portOrDefault(port, DefaultSourcePort)
	return runProtocol(host, ServerSource, opts, func() (Server, error) {
		addr, err := opts.resolvePreferred(ctx, host)
		if err != nil {
			return Server{}, err
		}
		client, err := a2s.New(addr.String(), int(port), a2s.WithTimeout(sourceTimeout(ctx)))
		if err != nil {
			return Server{}, err
		}
		defer client.Close()

		stop := context.AfterFunc(ctx, func() { client.Close() })
		defer stop()

		info, err := client.GetInfo(ctx)
		if err != nil {
			return Server{}, fmt.Errorf("%w: %w", ErrProtocolFailed, err)
		}
		players, err := client.GetPlayers(ctx)
		if err != nil {
			return Server{}, fmt.Errorf("%w: %w", ErrProtocolFailed, err)
		}
		s := serverFromSource(info, players, opts)
		s.IP = addr.String()
		return s, nil
	})
}

// queryAuto runs every protocol in parallel and returns the first
// successful result. If all fail, picks the most informative error.
func queryAuto(ctx context.Context, host string, port uint16, opts Options) Response {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		s   Server
		err error
		mc  bool
		src bool
	}
	ch := make(chan result, 3)
	go func() { s, err := queryMinecraft(ctx, host, port, opts); ch <- result{s: s, err: err, mc: true} }()
	go func() { s, err := queryMinecraftBedrock(ctx, host, port, opts); ch <- result{s: s, err: err} }()
	go func() { s, err := querySource(ctx, host, port, opts); ch <- result{s: s, err: err, src: true} }()

	var mcErr, bedrockErr, srcErr error
	for range 3 {
		r := <-ch
		if r.err == nil {
			normalizeServer(&r.s, host, port, detectType(r.s.GameKind))
			return Response{Payload: r.s}
		}
		switch {
		case r.mc:
			mcErr = r.err
			if !IsDialError(r.err) {
				return Response{Error: r.err}
			}
		case r.src:
			srcErr = r.err
		default:
			bedrockErr = r.err
		}
	}

	chosen := mcErr
	if chosen == nil {
		chosen = bedrockErr
	}
	if chosen == nil {
		chosen = srcErr
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		chosen = fmt.Errorf("%w: %w", ErrTimeout, chosen)
	}
	return Response{Error: chosen}
}
