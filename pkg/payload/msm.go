package payload

import (
	"bufio"
	"fmt"
	"io"
)

// VER
const (
	SocksVersion uint8 = 5
)

// METHOD
const (
	NoAuth       uint8 = 0
	UserPassAuth uint8 = 2
	NoAcceptable uint8 = 255
)

/*
   +----+----------+----------+
   |VER | NMETHODS | METHODS  |
   +----+----------+----------+
   | 1  |    1     | 1 to 255 |
   +----+----------+----------+
*/
// Method selection message sent by the client
type ClientMSM struct {
	Ver      uint8
	NMethods uint8
	Methods  [255]uint8 // Avoid heap allocation
}

func (m *ClientMSM) Read(br *bufio.Reader) error {
	var err error

	if m.Ver, err = br.ReadByte(); err != nil {
		return fmt.Errorf("failed to read ver: %w", err)
	}

	if m.NMethods, err = br.ReadByte(); err != nil {
		return fmt.Errorf("failed to read nmethods: %w", err)
	}

	if _, err := io.ReadFull(br, m.Methods[:m.NMethods]); err != nil {
		return fmt.Errorf("failed to read methods: %w", err)
	}

	return nil
}

func (m *ClientMSM) Write(bw *bufio.Writer) error {
	return nil
}

/*
   +----+--------+
   |VER | METHOD |
   +----+--------+
   | 1  |   1    |
   +----+--------+
*/
// Method selection message sent by the server
type ServerMSM struct {
	Ver    uint8
	Method uint8
}

func (m *ServerMSM) Read(br *bufio.Reader) error {
	// TODO
	return nil
}

func (m *ServerMSM) Write(bw *bufio.Writer) error {
	err := bw.WriteByte(m.Ver)
	if err != nil {
		return fmt.Errorf("failed to write ver: %w", err)
	}
	err = bw.WriteByte(m.Method)
	if err != nil {
		return fmt.Errorf("failed to write method: %w", err)
	}
	return nil
}
