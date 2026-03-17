package payload

import (
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

func (m *ClientMSM) Read(r io.Reader) error {
	var header [2]uint8
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	m.Ver = header[0]
	m.NMethods = header[1]

	if _, err := io.ReadFull(r, m.Methods[:m.NMethods]); err != nil {
		return fmt.Errorf("failed to read methods: %w", err)
	}

	return nil
}

func (m *ClientMSM) Write(w io.Writer) error {
	// TODO
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

func (m *ServerMSM) Read(r io.Reader) error {
	// TODO
	return nil
}

func (m *ServerMSM) Write(w io.Writer) error {
	buf := [2]byte{m.Ver, m.Method} // Avoid heap allocation
	_, err := w.Write(buf[:])
	return err
}
