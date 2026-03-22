package server

import (
	"context"
	"io"
	"net"
	"net/netip"

	"github.com/HT4w5/socks5/pkg/payload"
)

func (s *Server) streamCopy(dst io.Writer, src io.Reader, errCh chan error) {
	// Check manually for zero-copy to avoid unecessarily allocating a buffer

	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		_, err := wt.WriteTo(dst)
		errCh <- err
		return
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rf, ok := dst.(io.ReaderFrom); ok {
		_, err := rf.ReadFrom(src)
		errCh <- err
		return
	}

	// Use CopyBuffer with buffer from pool
	buf := s.tcpBytePool.Get()
	defer s.tcpBytePool.Put(buf)

	_, err := io.CopyBuffer(dst, src, buf)
	errCh <- err
}

// Send socks reply with rep (failure only)
func (s *Server) sendFailureReply(conn net.Conn, rep uint8) {
	s.logger.Warnf("sending reply: %s", payload.ReplyDesc(rep))
	r := payload.NewReply(
		payload.ReplyWithRep(rep),
		payload.ReplyWithIP(netip.IPv4Unspecified()),
	)
	s.logger.Debugf("sending reply: %s", r.String())
	if err := r.Write(conn); err != nil {
		s.logger.Errorf("failed to write reply: %v", err)
	}
}

// Drain connection and cancel context on closure
func contextFromConn(parent context.Context, conn net.Conn) context.Context {
	ctx, cancel := context.WithCancel(parent)

	go func() {
		defer cancel()
		io.Copy(io.Discard, conn)
	}()

	return ctx
}
