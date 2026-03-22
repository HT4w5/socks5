package method

import (
	"fmt"
	"net"

	"github.com/HT4w5/socks5/pkg/log"
	"github.com/HT4w5/socks5/pkg/payload"
)

// Negotiator handles socks method negotiation
// Returns a MethodHandler on success
type Negotiator struct {
	methods map[uint8]MethodHandler
	logger  log.Logger
}

func New(opts ...func(*Negotiator)) *Negotiator {
	neg := &Negotiator{
		methods: make(map[uint8]MethodHandler),
		logger:  &log.DiscardLogger{}, // Use discard logger by default
	}

	for _, opt := range opts {
		opt(neg)
	}

	return neg
}

func WithMethod(m MethodHandler) func(*Negotiator) {
	return func(n *Negotiator) {
		n.methods[m.GetCode()] = m
	}
}

func WithLogger(l log.Logger) func(*Negotiator) {
	return func(n *Negotiator) {
		n.logger = l
	}
}

// Handles client-server method negotiation; returns a MethodHandler
// for the negotiated method or error if no methods available
func (neg *Negotiator) HandleNegotiation(conn net.Conn) (MethodHandler, error) {
	// Read Client MSM
	var clientMSM payload.ClientMSM
	if err := clientMSM.Read(conn); err != nil {
		err = fmt.Errorf("failed to read client msm: %w", err)
		neg.logger.Errorf("%v", err)
		return nil, err
	}

	// Check socks version
	if clientMSM.Ver != payload.SocksVersion {
		return nil, fmt.Errorf("unsupported socks version: %v", clientMSM.Ver)
	}

	for _, m := range clientMSM.Methods[:clientMSM.NMethods] {
		if h, ok := neg.methods[m]; ok {
			// Method match; send msm
			serverMSM := payload.ServerMSM{
				Ver:    payload.SocksVersion,
				Method: m,
			}

			err := serverMSM.Write(conn)
			if err != nil {
				err := fmt.Errorf("failed to write server msm: %w", err)
				neg.logger.Errorf("%v", err)
				return nil, err
			}

			return h, nil
		}
	}

	// No matching methods; send NoAcceptable
	serverMSM := payload.ServerMSM{
		Ver:    payload.SocksVersion,
		Method: payload.NoAcceptable,
	}

	err := serverMSM.Write(conn)
	if err != nil {
		err := fmt.Errorf("failed to write server msm: %w", err)
		neg.logger.Errorf("%v", err)
		return nil, err
	}

	// Return error so caller tears down connection
	return nil, fmt.Errorf("no supported methods from client")
}

// A method handler handles method-dependent
// sub-negotiations
//
// Privides encapsulated net.Conn and method
// for transforming datagrams on departure
type MethodHandler interface {
	WrapConn(conn net.Conn) net.Conn
	TransformDatagram(data []byte) []byte
	UntransformDatagram(data []byte) []byte
	GetCode() uint8
}
