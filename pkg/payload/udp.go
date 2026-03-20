package payload

import (
	"encoding/binary"
	"net/netip"
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
func (r *UDPRequest) Write(buffer []byte) ([]byte, error) {
	headerLen := 6
	switch r.ATyp {
	case IPv4Addr:
		headerLen += 4
	case IPv6Addr:
		headerLen += 16
	case FQDNAddr:
		headerLen += 1 + len(r.DstFQDN)
	default:
		return nil, ErrUnsupportedAddressType
	}

	if len(buffer) < headerLen {
		return nil, ErrTooShort
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
	return buffer[headerLen:], nil
}
