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
	// Resolve FQDN
	if request.ATyp == payload.FQDNAddr {
		if addr, err := s.res.Resolve(ctx, string(request.DstFQDN[:])); err != nil {
			s.logger.Warnf("failed to resolve dst fqdn: %v; aborting", err)
			s.sendFailureReply(conn, payload.HostUnreachable)
		} else {
			request.DstAddr = addr
		}
	}

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
			s.logger.Warnf("failed to close target connection: %v", err)
		}
	}()

	// Send success reply
	addrPort, err := netip.ParseAddrPort(conn.LocalAddr().String())
	if err != nil {
		s.logger.Errorf("failed to parse local endpoint address: %v", err)
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
	go s.streamCopy(tgtConn, conn, errCh)
	go s.streamCopy(conn, tgtConn, errCh)

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
