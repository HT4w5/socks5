package errors

import "errors"

var (
	ErrUnknown         = errors.New("unknown error")
	ErrNetUnreachable  = errors.New("network unreachable")
	ErrHostUnreachable = errors.New("host unreachable")
	ErrConnRefused     = errors.New("connection refused")
	ErrTimeout         = errors.New("timeout")
)
