package server

import (
	"context"
	"io"
	"net"
	"net/netip"
	"strconv"

	"github.com/HT4w5/socks5/pkg/errors"
	"github.com/HT4w5/socks5/pkg/payload"
)

// Handles the CONNECT command
// Note that conn closure is handled by the caller
func (s *Server) handleConnect(ctx context.Context, conn net.Conn, request *payload.Request) {
	rep := payload.ServerFailure
	tgtConn, err := s.dialer(ctx, "tcp", net.JoinHostPort(request.DstAddr.String(), strconv.Itoa(int(request.DstPort))))
	if err != nil {
		s.logger.Warnf("dial error: %v", err)
		switch err {
		case errors.ErrNetUnreachable:
			rep = payload.NetworkUnreachable
		case errors.ErrHostUnreachable:
			rep = payload.HostUnreachable
		case errors.ErrConnRefused:
			rep = payload.ConnectionRefused
		case errors.ErrTimeout:
			rep = payload.TTLExpired
		}
		s.sendFailureReply(conn, rep)
		return
	}
	defer func() {
		err := tgtConn.Close()
		if err != nil {
			s.logger.Errorf("failed to close target connection: %v", err)
		}
	}()

	// Send success reply
	remoteAddr := conn.RemoteAddr().String()
	addrPort, err := netip.ParseAddrPort(remoteAddr)
	if err != nil {
		s.logger.Errorf("failed to parse remote address: %v", err)
		s.sendFailureReply(conn, rep)
		return
	}

	r := payload.NewReply(
		payload.ReplyWithRep(payload.Succeeded),
		payload.ReplyWithIP(addrPort.Addr()),
		payload.ReplyWithPort(addrPort.Port()),
	)

	s.logger.Debugf("sending reply: %s", r.String())

	if err := r.Write(conn); err != nil {
		s.logger.Errorf("failed to write reply: %v", err)
		return
	}

	// Start proxying
	errCh := make(chan error, 2)
	go copy(tgtConn, conn, errCh)
	go copy(conn, tgtConn, errCh)

	select {
	case <-ctx.Done():
		s.logger.Warnf("connection cancelled by context")
		return
	case err := <-errCh:
		if err != nil && err != io.EOF {
			s.logger.Errorf("copy error: %v", err)
		}
		return
	}
}
