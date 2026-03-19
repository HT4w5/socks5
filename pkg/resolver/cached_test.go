package resolver_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/HT4w5/socks5/pkg/resolver"
)

func TestCachedResolver_SimpleResolve(t *testing.T) {
	r := resolver.NewCachedResolver(
		resolver.WithResolver(
			&net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					return net.Dial("udp", "8.8.8.8:53")
				},
			},
		),
		resolver.WithMaxLen(10),
		resolver.WithTTL(5*time.Second),
	)

	got, err := r.Resolve(t.Context(), "localhost")
	if err != nil {
		t.Errorf("resolve error: %v", err)
	}

	if !got.IsLoopback() {
		t.Errorf("bad resolve result, expected loopback, got %s", got)
	}
}
