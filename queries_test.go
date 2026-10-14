package query

import (
	"context"
	"testing"
	"time"
)

func TestCtxTimeout(t *testing.T) {
	if got := ctxTimeout(context.Background(), time.Second); got != time.Second {
		t.Fatalf("no deadline: got %s, want %s", got, time.Second)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	got := ctxTimeout(ctx, time.Second)
	if got <= 0 || got > 100*time.Millisecond {
		t.Fatalf("deadline sooner: got %s, want <= 100ms", got)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Hour)
	defer cancel2()
	if got := ctxTimeout(ctx2, time.Second); got != time.Second {
		t.Fatalf("deadline later: got %s, want %s", got, time.Second)
	}
}

func TestGraceOr(t *testing.T) {
	if got := graceOr(context.Background(), time.Second); got != time.Second {
		t.Fatalf("no deadline: got %s, want %s", got, time.Second)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	got := graceOr(ctx, time.Second)
	if got <= 0 || got > 50*time.Millisecond {
		t.Fatalf("with deadline: got %s, want <= 50ms", got)
	}
}

func TestHooksZeroValueNoPanic(t *testing.T) {
	var h Hooks
	h.attempt("x", ServerSource)
	h.result("x", ServerSource, nil, 0)
}
