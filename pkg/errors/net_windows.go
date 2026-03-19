//go:build windows

package errors

import (
	"errors"
	"net"
	"syscall"
)

const (
	WSAENETUNREACH  = syscall.Errno(10051)
	WSAEHOSTUNREACH = syscall.Errno(10065)
	WSAECONNREFUSED = syscall.Errno(10061)
	WSAETIMEDOUT    = syscall.Errno(10060)
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
	case WSAENETUNREACH:
		return ErrNetUnreachable
	case WSAEHOSTUNREACH:
		return ErrHostUnreachable
	case WSAECONNREFUSED:
		return ErrConnRefused
	case WSAETIMEDOUT:
		return ErrTimeout
	}

	return ErrUnknown
}
