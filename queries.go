package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/status"
	"github.com/woozymasta/a2s/pkg/a2s"
)

func graceOr(ctx context.Context, d time.Duration) time.Duration {
	if dl, ok := ctx.Deadline(); ok {
		if remain := time.Until(dl); remain < d {
			return remain
		}
	}
	return d
}

func ctxTimeout(ctx context.Context, def time.Duration) time.Duration {
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 && d < def {
			return d
		}
	}
	return def
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

// ReceiveLimitBytes caps how much the underlying mcutil readers will
// accept from the network. 1 MiB covers the largest expected status
// and rules responses with headroom.
const defaultReceiveLimit int64 = 1 << 20

func queryMinecraftModern(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	opts.Hooks.attempt(host, ServerMinecraftModern)
	start := time.Now()
	resp, err := status.Modern(ctx, host, port, options.StatusModern{
		EnableSRV:         true,
		Timeout:           ctxTimeout(ctx, opts.Timeout),
		ProtocolVersion:   -1,
		ReceiveLimitBytes: defaultReceiveLimit,
		Ping:              true,
	})
	opts.Hooks.result(host, ServerMinecraftModern, err, time.Since(start))
	if err != nil {
		return Server{}, &ProtocolError{Proto: ServerMinecraftModern, Err: err}
	}
	return serverFromModern(resp, opts), nil
}

func queryMinecraftLegacy(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	opts.Hooks.attempt(host, ServerMinecraftLegacy)
	start := time.Now()
	resp, err := status.Legacy(ctx, host, port, options.StatusLegacy{
		EnableSRV:         true,
		Timeout:           ctxTimeout(ctx, opts.Timeout),
		ProtocolVersion:   -1,
		ReceiveLimitBytes: defaultReceiveLimit,
	})
	opts.Hooks.result(host, ServerMinecraftLegacy, err, time.Since(start))
	if err != nil {
		return Server{}, &ProtocolError{Proto: ServerMinecraftLegacy, Err: err}
	}
	return serverFromLegacy(resp, opts), nil
}

func queryMinecraftBedrock(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	opts.Hooks.attempt(host, ServerMinecraftBedrock)
	start := time.Now()
	resp, err := status.Bedrock(ctx, host, port, options.StatusBedrock{
		Timeout:           ctxTimeout(ctx, opts.Timeout),
		ReceiveLimitBytes: defaultReceiveLimit,
	})
	opts.Hooks.result(host, ServerMinecraftBedrock, err, time.Since(start))
	if err != nil {
		return Server{}, &ProtocolError{Proto: ServerMinecraftBedrock, Err: err}
	}
	return serverFromBedrock(resp, opts), nil
}

// queryMinecraft runs modern and legacy in parallel, giving modern priority.
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

func querySource(ctx context.Context, host string, port uint16, opts Options) (Server, error) {
	opts.Hooks.attempt(host, ServerSource)
	start := time.Now()

	addr, err := opts.resolvePreferred(ctx, host)
	if err != nil {
		opts.Hooks.result(host, ServerSource, err, time.Since(start))
		return Server{}, err
	}
	client, err := a2s.New(addr.String(), int(port), a2s.WithTimeout(ctxTimeout(ctx, opts.Timeout)))
	if err != nil {
		opts.Hooks.result(host, ServerSource, err, time.Since(start))
		return Server{}, err
	}
	defer client.Close()

	stop := context.AfterFunc(ctx, func() { client.Close() })
	defer stop()

	info, err := client.GetInfo(ctx)
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrProtocolFailed, err)
		opts.Hooks.result(host, ServerSource, err, time.Since(start))
		return Server{}, err
	}
	players, err := client.GetPlayers(ctx)
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrProtocolFailed, err)
		opts.Hooks.result(host, ServerSource, err, time.Since(start))
		return Server{}, err
	}
	opts.Hooks.result(host, ServerSource, nil, time.Since(start))
	s := serverFromSource(info, players, opts)
	s.IP = addr.String()
	return s, nil
}

// queryAuto probes Minecraft Java, Bedrock, and Source in parallel.
// The first successful response wins; on total failure, Minecraft's
// error is preferred, then Bedrock's, then Source's.
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
