package dialer

import (
	"context"
	"net"

	"github.com/HT4w5/socks5/pkg/errors"
)

type Dialer func(ctx context.Context, network string, address string) (net.Conn, error)

var DefaultDialer = func(ctx context.Context, network string, address string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, network, address)
	return conn, errors.GetNetworkError(err)
}
