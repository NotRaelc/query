package query

import "errors"

// ProtocolError wraps an error with the protocol that produced it.
type ProtocolError struct {
	Proto ServerType
	Err   error
}

func (e *ProtocolError) Error() string { return e.Proto.String() + ": " + e.Err.Error() }
func (e *ProtocolError) Unwrap() error { return e.Err }

var (
	ErrTimeout        = errors.New("query timeout exceeded")
	ErrProtocolFailed = errors.New("protocol query failed")
	ErrBlockedHost    = errors.New("query: host is blocked")
)
