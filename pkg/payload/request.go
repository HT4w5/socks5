package payload

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

// ATYP
const (
	IPv4Addr   uint8 = 1
	DomainName uint8 = 3
	IPv6Addr   uint8 = 4
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
	Ver       uint8
	Cmd       uint8
	ATyp      uint8
	DstAddr   netip.Addr
	DstDomain [255]byte // Avoid heap allocation
	DstPort   uint16
}

func (req *Request) Read(r io.Reader) error {
	// Read Ver, Cmd and ATyp
	header := make([]uint8, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("failed to cread header: %w", err)
	}

	req.Ver = header[0]
	req.Cmd = header[1]
	// Skip Rsv
	req.ATyp = header[3]

	// Read DstAddr (or DstDomain)
	switch req.ATyp {
	case IPv4Addr:
		var addr [4]byte
		if _, err := io.ReadFull(r, addr[:]); err != nil {
			return fmt.Errorf("failed to read ipv4 dst address: %v", err)
		}
		req.DstAddr = netip.AddrFrom4(addr)
	case IPv6Addr:
		var addr [16]byte
		if _, err := io.ReadFull(r, addr[:]); err != nil {
			return fmt.Errorf("failed to read ipv6 dst address: %v", err)
		}
		req.DstAddr = netip.AddrFrom16(addr)
	case DomainName:
		var nameLen [1]uint8
		if _, err := io.ReadFull(r, nameLen[:]); err != nil {
			return fmt.Errorf("failed to read dst domain name length: %v", err)
		}

		if _, err := io.ReadFull(r, req.DstDomain[:nameLen[0]]); err != nil {
			return fmt.Errorf("failed to read dst domain name: %v", err)
		}
	}

	// Read DstPort
	var port [2]byte
	if _, err := io.ReadFull(r, port[:]); err != nil {
		return err
	}
	req.DstPort = binary.BigEndian.Uint16(port[:])

	return nil
}

func (req *Request) Write(r io.Writer) error {
	// TODO
	return nil
}
