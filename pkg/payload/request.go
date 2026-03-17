package payload

import (
	"bufio"
	"fmt"
	"net/netip"
)

// ATYP
const (
	IPv4Addr   uint8 = 1
	DomainName uint8 = 3
	IPv6Addr   uint8 = 4
)

// RSV
const (
	requestRSV uint8 = 0
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
	Ver          uint8
	Cmd          uint8
	ATyp         uint8
	DstAddr      netip.Addr
	DstDomainLen uint8
	DstDomain    [255]byte // Avoid heap allocation
	DstPort      uint16
}

func (req *Request) Read(br *bufio.Reader) error {
	// Read Ver, Cmd and ATyp
	var err error
	if req.Ver, err = br.ReadByte(); err != nil {
		return fmt.Errorf("failed to read ver: %w", err)
	}
	if req.Cmd, err = br.ReadByte(); err != nil {
		return fmt.Errorf("failed to read cmd: %w", err)
	}
	if _, err = br.Discard(1); err != nil {
		return fmt.Errorf("failed to discard rsv: %w", err)
	}
	if req.ATyp, err = br.ReadByte(); err != nil {
		return fmt.Errorf("failed to read atyp: %w", err)
	}

	// Read DstAddr (or DstDomain)
	switch req.ATyp {
	case IPv4Addr:
		if req.DstAddr, err = readIPv4Addr(br); err != nil {
			return err
		}
	case IPv6Addr:
		if req.DstAddr, err = readIPv6Addr(br); err != nil {
			return err
		}
	case DomainName:
		if req.DstDomainLen, req.DstDomain, err = readDomainName(br); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported dst address type: %v", req.ATyp)
	}

	if req.DstPort, err = readPort(br); err != nil {
		return err
	}

	return nil
}

func (req *Request) Write(bw *bufio.Writer) error {
	// TODO
	return nil
}
