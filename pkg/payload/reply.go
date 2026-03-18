package payload

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

// RSV
const (
	replyRSV uint8 = 0
)

// REP
const (
	Succeeded               uint8 = 0
	ServerFailure           uint8 = 1
	NotAllowed              uint8 = 2
	NetworkUnreachable      uint8 = 3
	HostUnreachable         uint8 = 4
	ConnectionRefused       uint8 = 5
	TTLExpired              uint8 = 6
	CommandNotSupported     uint8 = 7
	AddressTypeNotSupported uint8 = 8
)

/*
   +----+-----+-------+------+----------+----------+
   |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
   +----+-----+-------+------+----------+----------+
   | 1  |  1  | X'00' |  1   | Variable |    2     |
   +----+-----+-------+------+----------+----------+
*/
// A socks reply
type Reply struct {
	Ver        uint8
	Rep        uint8
	ATyp       uint8
	BndAddr    netip.Addr
	BndFQDNLen uint8
	BndFQDN    [255]byte // Avoid heap allocation
	BndPort    uint16
}

func NewReply(opts ...func(*Reply)) Reply {
	r := Reply{
		Ver:     SocksVersion,
		BndAddr: netip.IPv6Unspecified(),
	}

	for _, opt := range opts {
		opt(&r)
	}

	return r
}

func ReplyWithRep(rep uint8) func(*Reply) {
	return func(r *Reply) {
		r.Rep = rep
	}
}

func ReplyWithATyp(atyp uint8) func(*Reply) {
	return func(r *Reply) {
		r.ATyp = atyp
	}
}

func ReplyWithBndAddr(atyp uint8) func(*Reply) {
	return func(r *Reply) {
		r.ATyp = atyp
	}
}

func (m *Reply) Read(r io.Reader) error {
	// TODO
	return nil
}

func (rep *Reply) Write(w io.Writer) error {
	var reply [261]byte // Max Reply length
	replyLen := 6       // Actual Reply length

	reply[0] = rep.Ver
	reply[1] = rep.Rep
	reply[2] = replyRSV
	reply[4] = rep.ATyp

	// Copy BndAddr

	switch rep.ATyp {
	case IPv4Addr:
		addr := rep.BndAddr.As4()
		copy(reply[5:], addr[:])
		replyLen += 4
	case FQDNAddr:
		reply[5] = rep.BndFQDNLen
		copy(reply[6:], rep.BndFQDN[:rep.BndFQDNLen])
		replyLen += int(rep.BndFQDNLen) + 1
	case IPv6Addr:
		addr := rep.BndAddr.As16()
		copy(reply[5:], addr[:])
		replyLen += 16
	default:
		return fmt.Errorf("unsupported bnd address type: %v", rep.ATyp)
	}

	binary.BigEndian.PutUint16(reply[replyLen-2:], rep.BndPort)

	_, err := w.Write(reply[:replyLen])
	if err != nil {
		return fmt.Errorf("failed to write reply: %w", err)
	}

	return nil
}
