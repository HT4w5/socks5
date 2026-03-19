package payload

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

// ATYP
const (
	IPv4Addr uint8 = 1
	FQDNAddr uint8 = 3
	IPv6Addr uint8 = 4
)

// RSV
const (
	requestRSV uint8 = 0
)

// CMD
const (
	Connect      uint8 = 1
	Bind         uint8 = 2
	UDPAssociate uint8 = 3
)

/*
   +----+-----+-------+------+----------+----------+
   |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
   +----+-----+-------+------+----------+----------+
   | 1  |  1  | X'00' |  1   | Variable |    2     |
   +----+-----+-------+------+----------+----------+
*/
// A socks request
type Request struct {
	Ver        uint8
	Cmd        uint8
	ATyp       uint8
	DstAddr    netip.Addr
	DstFQDNLen uint8
	DstFQDN    [255]byte // Avoid heap allocation
	DstPort    uint16
}

func (req *Request) Read(r io.Reader) error {
	var header [4]byte

	if _, err := io.ReadFull(r, header[:]); err != nil {
		return fmt.Errorf("failed to read request header: %w", err)
	}

	req.Ver = header[0]
	req.Cmd = header[1]
	req.ATyp = header[3]

	// Read DstAddr (or DstFQDN)
	switch req.ATyp {
	case IPv4Addr:
		var addr [4]byte
		if _, err := io.ReadFull(r, addr[:]); err != nil {
			return fmt.Errorf("failed to read ipv4 addr: %w", err)
		}
		req.DstAddr = netip.AddrFrom4(addr)
	case IPv6Addr:
		var addr [16]byte
		if _, err := io.ReadFull(r, addr[:]); err != nil {
			return fmt.Errorf("failed to read ipv6 addr: %w", err)
		}
		req.DstAddr = netip.AddrFrom16(addr)
	case FQDNAddr:
		var nameLen [1]uint8
		if _, err := io.ReadFull(r, nameLen[:]); err != nil {
			return fmt.Errorf("failed to read fqdn length: %w", err)
		}

		if _, err := io.ReadFull(r, req.DstFQDN[:nameLen[0]]); err != nil {
			return fmt.Errorf("failed to read fqdn: %w", err)
		}
	default:
		// Address type not supported
		// The caller is responsible for checking the address type
		// and sending a `X'08' Address type not supported` reply
		return nil
	}

	var port [2]byte
	if _, err := io.ReadFull(r, port[:]); err != nil {
		return fmt.Errorf("failed to read fqdn: %w", err)
	}
	req.DstPort = binary.BigEndian.Uint16(port[:])

	return nil
}

func (req *Request) Write(w io.Writer) error {
	// TODO
	return nil
}
