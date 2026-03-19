//go:build !windows

package errors

import (
	"errors"
	"net"
	"syscall"
)

func GetNetworkError(err error) error {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return ErrUnknown
	}

	var errno syscall.Errno
	if !errors.As(opErr.Err, &errno) {
		return ErrUnknown
	}

	switch errno {
	case syscall.ENETUNREACH:
		return ErrNetUnreachable
	case syscall.EHOSTUNREACH:
		return ErrHostUnreachable
	case syscall.ECONNREFUSED:
		return ErrConnRefused
	case syscall.ETIMEDOUT:
		return ErrTimeout
	}

	return ErrUnknown
}
