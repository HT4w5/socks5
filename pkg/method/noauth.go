package method

import (
	"net"

	"github.com/HT4w5/socks5/pkg/payload"
)

type NoAuthHandler struct {
}

func (h *NoAuthHandler) GetCode() uint8 {
	return payload.NoAuth
}

func (h *NoAuthHandler) WrapConn(conn net.Conn) net.Conn {
	return conn
}

func (h *NoAuthHandler) TransformDatagram(data []byte) []byte {
	return data
}

func (h *NoAuthHandler) UntransformDatagram(data []byte) []byte {
	return data
}
