package socks5

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/HT4w5/socks5/pkg/log"
	"github.com/HT4w5/socks5/pkg/payload"
)

type Server struct {
	logger log.Logger
}

// Creates a socks5 server
func New(opts ...func(*Server)) *Server {
	srv := &Server{
		// Use discard logger in default
		logger: &log.DiscardLogger{},
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

// Listen on network and addr
func (s *Server) ListenAndServe(ctx context.Context, network, addr string) error {
	cfg := net.ListenConfig{}

	lis, err := cfg.Listen(ctx, network, addr)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	// Closer
	go func() {
		<-ctx.Done()
		err := lis.Close()
		if err != nil {
			s.logger.Errorf("failed to close listener: %v", err)
		}
	}()

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				wg.Wait()
				return ctx.Err()
			default:
				s.logger.Errorf("failed to accept connection: %v", err)
			}
		}
		go s.handleConn(conn)
	}
}

// Handles a connection
func (s *Server) handleConn(conn net.Conn) error {
	defer func() {
		err := conn.Close()
		if err != nil {
			s.logger.Errorf("failed to close connection: %v", err)
		}
	}()

	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)

	// Check socks version for safety
	ver, err := br.ReadByte()
	if err != nil {
		s.logger.Errorf("failed to get version byte: %v", err)
		return err
	}

	if ver != payload.SocksVersion {
		err := fmt.Errorf("unsupported socks version: %v", ver)
		s.logger.Errorf("%v", err)
		return err
	}

	//
}
