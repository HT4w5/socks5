package server

import (
	"io"
	"net"
	"net/netip"

	"github.com/HT4w5/socks5/pkg/payload"
)

func copy(dst io.Writer, src io.Reader, errCh chan error) {
	_, err := io.Copy(dst, src)
	errCh <- err
}

// Send socks reply with rep (failure only)
func (s *Server) sendFailureReply(conn net.Conn, rep uint8) {
	s.logger.Warnf("sending reply: %s", payload.ReplyReason(rep))
	r := payload.NewReply(
		payload.ReplyWithRep(rep),
		payload.ReplyWithIP(netip.IPv4Unspecified()),
	)
	if err := r.Write(conn); err != nil {
		s.logger.Errorf("failed to write reply: %v", err)
	}
}
