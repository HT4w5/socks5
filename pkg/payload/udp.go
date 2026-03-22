package payload

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"
)

// FRAG
const (
	NoFrag uint8 = 0
)

/*
   +----+------+------+----------+----------+----------+
   |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
   +----+------+------+----------+----------+----------+
   | 2  |  1   |  1   | Variable |    2     | Variable |
   +----+------+------+----------+----------+----------+
*/
// A socks UDP request header
type UDPRequest struct {
	Frag    uint8
	ATyp    uint8
	DstAddr netip.Addr
	DstFQDN []byte
	DstPort uint16
	Data    []byte
}

func (u *UDPRequest) String() string {
	var b strings.Builder
	b.WriteString("{")

	// Fragmentation
	if u.Frag == 0 {
		b.WriteString("Frag: NONE, ")
	} else {
		fmt.Fprintf(&b, "Frag: %d, ", u.Frag)
	}

	// DstAddr
	b.WriteString("Dst: ")
	switch u.ATyp {
	case IPv4Addr, IPv6Addr:
		fmt.Fprintf(&b, "%s:%d", u.DstAddr.String(), u.DstPort)
	case FQDNAddr:
		domain := string(u.DstFQDN)
		fmt.Fprintf(&b, "%s:%d", domain, u.DstPort)
	default:
		fmt.Fprintf(&b, "INVALID_ATYP(%d)", u.ATyp)
	}

	b.WriteString("}")
	return b.String()
}

// Parse extracts header from datagram
func (r *UDPRequest) Parse(datagram []byte) error {
	datagramLen := len(datagram)
	if datagramLen < 8 { // Minimum possible size
		return ErrTooShort
	}

	r.Frag = datagram[2]
	r.ATyp = datagram[3]

	switch r.ATyp {
	case IPv4Addr:
		if datagramLen < 10 { // Minimum possible size with IPv4
			return ErrTooShort
		}
		r.DstAddr = netip.AddrFrom4([4]byte(datagram[4:8]))
		r.DstPort = binary.BigEndian.Uint16(datagram[8:10])
		r.Data = datagram[10:]
	case IPv6Addr:
		if datagramLen < 22 { // Minimum possible size with IPv6
			return ErrTooShort
		}
		r.DstAddr = netip.AddrFrom16([16]byte(datagram[4:20]))
		r.DstPort = binary.BigEndian.Uint16(datagram[20:22])
		r.Data = datagram[22:]
	case FQDNAddr:
		len := int(datagram[4])
		if datagramLen < 7+len {
			return ErrTooShort
		}
		r.DstFQDN = datagram[5 : 5+len]
		r.DstPort = binary.BigEndian.Uint16(datagram[len+5 : len+7])
		r.Data = datagram[len+7 : len+9]
	}

	return nil
}

// Write writes to a byte slice and returns a new slice header
// for the caller to write the data to
//
// Caller is responsible for handling the lifecycle of the buffer slice
func (r *UDPRequest) Write(buffer []byte) error {
	headerLen := 6
	switch r.ATyp {
	case IPv4Addr:
		headerLen += 4
	case IPv6Addr:
		headerLen += 16
	case FQDNAddr:
		headerLen += 1 + len(r.DstFQDN)
	default:
		return ErrUnsupportedAddressType
	}

	if len(buffer) < headerLen {
		return ErrTooShort
	}

	// RSV
	buffer[0] = 0
	buffer[1] = 0

	buffer[2] = r.Frag
	buffer[3] = r.ATyp

	switch r.ATyp {
	case IPv4Addr:
		addrBytes := r.DstAddr.As4()
		copy(buffer[4:8], addrBytes[:])
	case IPv6Addr:
		addrBytes := r.DstAddr.As16()
		copy(buffer[4:20], addrBytes[:])
	case FQDNAddr:
		fqdnLen := len(r.DstFQDN)
		buffer[4] = uint8(fqdnLen)
		copy(buffer[5:5+fqdnLen], r.DstFQDN)
	}

	binary.BigEndian.PutUint16(buffer[headerLen-2:headerLen], r.DstPort)
	return nil
}
