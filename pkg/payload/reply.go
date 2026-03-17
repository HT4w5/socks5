package payload

import (
	"bufio"
	"fmt"
	"net/netip"
)

// RSV
const (
	replyRSV uint8 = 0
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
	Ver          uint8
	Rep          uint8
	ATyp         uint8
	BndAddr      netip.Addr
	BndDomainLen uint8
	BndDomain    [255]byte // Avoid heap allocation
	BndPort      uint16
}

func (m *Reply) Read(br *bufio.Reader) error {
	// TODO
	return nil
}

func (rep *Reply) Write(bw *bufio.Writer) error {
	var err error
	if err = bw.WriteByte(rep.Ver); err != nil {
		return fmt.Errorf("failed to write ver: %w", err)
	}
	if err = bw.WriteByte(rep.Rep); err != nil {
		return fmt.Errorf("failed to write rep: %w", err)
	}
	if err = bw.WriteByte(replyRSV); err != nil {
		return fmt.Errorf("failed to write rsv: %w", err)
	}
	if err = bw.WriteByte(rep.ATyp); err != nil {
		return fmt.Errorf("failed to write atyp: %w", err)
	}

	switch rep.ATyp {
	case IPv4Addr:
		err = writeIPv4Addr(bw, rep.BndAddr)
		if err != nil {
			return err
		}
	case DomainName:
		err = writeDomainName(bw, rep.BndDomainLen, rep.BndDomain[:])
		if err != nil {
			return err
		}
	case IPv6Addr:
		err = writeIPv6Addr(bw, rep.BndAddr)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported bnd address type: %v", rep.ATyp)
	}

	err = writePort(bw, rep.BndPort)
	if err != nil {
		return err
	}

	return nil
}
