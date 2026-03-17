package payload

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

const (
	ipv4AddrLen = 4
	ipv6AddrLen = 16
	portLen     = 2
)

// Read helpers

func readIPv4Addr(br *bufio.Reader) (netip.Addr, error) {
	var addr netip.Addr
	b, err := br.Peek(ipv4AddrLen)
	if err != nil {
		return addr, fmt.Errorf("failed to peek ipv4 address: %w", err)
	}
	_, err = br.Discard(ipv4AddrLen)
	if err != nil {
		return addr, fmt.Errorf("failed to discard buffer: %w", err)
	}
	addr = netip.AddrFrom4([ipv4AddrLen]byte(b))

	return addr, nil
}

func readIPv6Addr(br *bufio.Reader) (netip.Addr, error) {
	var addr netip.Addr
	b, err := br.Peek(ipv6AddrLen)
	if err != nil {
		return addr, fmt.Errorf("failed to peek ipv6 address: %w", err)
	}
	_, err = br.Discard(ipv6AddrLen)
	if err != nil {
		return addr, fmt.Errorf("failed to discard buffer: %w", err)
	}
	addr = netip.AddrFrom16([ipv6AddrLen]byte(b))

	return addr, nil
}

func readDomainName(br *bufio.Reader) (uint8, [255]byte, error) {
	var nameLen uint8
	var name [255]byte
	nameLen, err := br.ReadByte()
	if err != nil {
		return nameLen, name, fmt.Errorf("failed to read domain name length: %v", err)
	}

	if _, err := io.ReadFull(br, name[:nameLen]); err != nil {
		return nameLen, name, fmt.Errorf("failed to read domain name: %v", err)
	}

	return nameLen, name, nil
}

func readPort(br *bufio.Reader) (uint16, error) {
	var port uint16
	buf, err := br.Peek(portLen)
	if err != nil {
		return port, fmt.Errorf("failed to read port: %v", err)
	}
	_, err = br.Discard(ipv6AddrLen)
	if err != nil {
		return port, fmt.Errorf("failed to discard buffer: %w", err)
	}

	port = binary.BigEndian.Uint16(buf[:])

	return port, nil
}

// Write helpers

func writeIPv4Addr(bw *bufio.Writer, addr netip.Addr) error {
	addrBytes := addr.As4()
	if _, err := bw.Write(addrBytes[:]); err != nil {
		return fmt.Errorf("failed to write ipv4 address: %w", err)
	}
	return nil
}

func writeIPv6Addr(bw *bufio.Writer, addr netip.Addr) error {
	addrBytes := addr.As16()
	if _, err := bw.Write(addrBytes[:]); err != nil {
		return fmt.Errorf("failed to write ipv6 address: %w", err)
	}
	return nil
}

func writeDomainName(bw *bufio.Writer, nameLen uint8, nameBytes []byte) error {
	if err := bw.WriteByte(nameLen); err != nil {
		return fmt.Errorf("failed to write domain name length: %w", err)
	}
	if _, err := bw.Write(nameBytes[:nameLen]); err != nil {
		return fmt.Errorf("failed to write domain name: %w", err)
	}
	return nil
}

func writePort(bw *bufio.Writer, port uint16) error {
	var portBytes [2]byte
	binary.BigEndian.PutUint16(portBytes[:], port)
	if _, err := bw.Write(portBytes[:]); err != nil {
		return fmt.Errorf("failed to write port: %w", err)
	}
	return nil
}
