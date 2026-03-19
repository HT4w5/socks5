package payload

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
	"strings"
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

// Get description of REP
func ReplyDesc(rep uint8) string {
	switch rep {
	case Succeeded:
		return "succeeded"
	case ServerFailure:
		return "general socks server failure"
	case NotAllowed:
		return "connection not allowed by ruleset"
	case NetworkUnreachable:
		return "network unreachable"
	case HostUnreachable:
		return "host unreachable"
	case ConnectionRefused:
		return "connection refused"
	case TTLExpired:
		return "ttl expired"
	case CommandNotSupported:
		return "command not supported"
	case AddressTypeNotSupported:
		return "address type not supported"
	default:
		return fmt.Sprintf("unknown rep (0x%02x)", rep)
	}
}

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

func (r *Reply) String() string {
	var b strings.Builder
	b.WriteString("{")

	// Ver
	fmt.Fprintf(&b, "Ver: %d, ", r.Ver)

	// Rep
	fmt.Fprintf(&b, "Status: %s, ", ReplyDesc(r.Rep))

	// BndAddr
	b.WriteString("Bnd: ")
	switch r.ATyp {
	case IPv4Addr, IPv6Addr:
		fmt.Fprintf(&b, "%s:%d", r.BndAddr.String(), r.BndPort)
	case FQDNAddr:
		domain := string(r.BndFQDN[:r.BndFQDNLen])
		fmt.Fprintf(&b, "%s:%d", domain, r.BndPort)
	default:
		fmt.Fprintf(&b, "INVALID_ATYP(%d)", r.ATyp)
	}

	b.WriteString("}")
	return b.String()
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

func ReplyWithIP(ip netip.Addr) func(*Reply) {
	return func(r *Reply) {
		if ip.Is4() {
			r.ATyp = IPv4Addr
		} else {
			r.ATyp = IPv6Addr
		}
		r.BndAddr = ip
	}
}

func ReplyWithFQDN(fqdn []byte) func(*Reply) {
	return func(r *Reply) {
		r.BndFQDNLen = uint8(len(fqdn))
		r.BndFQDN = [255]byte(fqdn)
	}
}

func ReplyWithPort(port uint16) func(*Reply) {
	return func(r *Reply) {
		r.BndPort = port
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
	reply[3] = rep.ATyp

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
