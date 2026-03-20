package server

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/HT4w5/socks5/pkg/dialer"
	"github.com/HT4w5/socks5/pkg/log"
	"github.com/HT4w5/socks5/pkg/method"
	"github.com/HT4w5/socks5/pkg/payload"
	"github.com/HT4w5/socks5/pkg/resolver"
)

const (
	shutdownTimeout = 30 * time.Second
)

type Server struct {
	endpoint netip.AddrPort
	res      resolver.Resolver
	dialer   func(ctx context.Context, network string, address string) (net.Conn, error)
	neg      *method.Negotiator
	logger   log.Logger
}

// Creates a socks5 server
func New(opts ...func(*Server)) *Server {
	srv := &Server{
		logger: &log.DiscardLogger{}, // Use discard logger by default
		dialer: dialer.DefaultDialer,
		res:    &resolver.SystemResolver{},
		neg:    method.New(),
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

func WithLogger(l log.Logger) func(*Server) {
	return func(s *Server) {
		s.logger = l
	}
}

func WithNegotiator(neg *method.Negotiator) func(*Server) {
	return func(s *Server) {
		s.neg = neg
	}
}

func WithDialer(d dialer.Dialer) func(*Server) {
	return func(s *Server) {
		s.dialer = d
	}
}

func WithResolver(r resolver.Resolver) func(*Server) {
	return func(s *Server) {
		s.res = r
	}
}

// Listen on network and addr
func (s *Server) ListenAndServe(ctx context.Context, addr netip.AddrPort) error {
	s.endpoint = addr
	addrString := addr.String()
	cfg := net.ListenConfig{}

	lis, err := cfg.Listen(ctx, "tcp", addrString)
	if err != nil {
		return err
	}
	s.logger.Infof("listening on %s", addrString)

	var wg sync.WaitGroup

	connCtx, cancelConn := context.WithCancel(context.Background())
	defer cancelConn()

	// Closer
	go func() {
		<-ctx.Done()
		if err := lis.Close(); err != nil {
			s.logger.Errorf("failed to close listener: %v", err)
		}
		time.AfterFunc(shutdownTimeout, cancelConn)
	}()

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				s.logger.Infof("server shutdown")
				s.logger.Infof("waiting for sessions to exit...")
				wg.Wait()
				return nil
			default:
				s.logger.Errorf("failed to accept connection: %v", err)
				continue
			}
		}
		s.logger.Infof("accepted connection from %s", conn.RemoteAddr().String())
		wg.Go(func() {
			s.handleConn(connCtx, conn)
		})
	}
}

// Handles a connection
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	s.logger.Debugf("handling connection")
	defer func() {
		err := conn.Close()
		if err != nil {
			s.logger.Errorf("failed to close connection: %v", err)
		}
	}()

	handler, err := s.neg.HandleNegotiation(conn)
	if err != nil {
		s.logger.Warnf("method negotiation failed: %v", err)
		return
	}

	// Wrap connection in method handler
	conn = handler.Wrap(conn)

	// From now on, a reply is guaranteed
	// `goto Failure` triggers non-success reply
	rep := payload.ServerFailure

	// Process request
	var request payload.Request
	if err := request.Read(conn); err != nil {
		s.logger.Errorf("failed to read request: %v", err)
		goto Failure
	}

	s.logger.Debugf("received request: %s", request.String())

	// Check version
	if request.Ver != payload.SocksVersion {
		s.logger.Warnf("unsupported socks version: %v", request.Ver)
		goto Failure
	}

	// Check command (for support, not allowance)
	switch request.Cmd {
	case payload.Connect:
	case payload.Bind: // TODO: implement Bind
		fallthrough
	case payload.UDPAssociate: // TODO: implement UDPAssociate
		fallthrough
	default:
		rep = payload.CommandNotSupported
		goto Failure
	}

	// Check address type (for support, not allowance)
	switch request.ATyp {
	case payload.IPv4Addr:
		fallthrough
	case payload.IPv6Addr:
		fallthrough
	case payload.FQDNAddr:
	default:
		rep = payload.AddressTypeNotSupported
		goto Failure
	}

	// TODO: implement ruleset

	// Resolve FQDN
	if request.ATyp == payload.FQDNAddr {
		if addr, err := s.res.Resolve(ctx, string(request.DstFQDN[:])); err != nil {
			s.logger.Warnf("failed to resolve dst fqdn: %v; aborting", err)
			rep = payload.HostUnreachable
			goto Failure
		} else {
			request.DstAddr = addr
		}
	}

	// Select command
	// Upsteam connection test is handled by specific command handler
	switch request.Cmd {
	case payload.Connect:
		s.handleConnect(ctx, conn, &request)
	case payload.Bind: // TODO: implement Bind
		fallthrough
	case payload.UDPAssociate: // TODO: implement UDPAssociate
		fallthrough
	default:
		// Should never reach this point if nothing is wrong
		rep = payload.CommandNotSupported
		goto Failure
	}

	return
	// Handle failure reply
Failure:
	s.sendFailureReply(conn, rep)
}
