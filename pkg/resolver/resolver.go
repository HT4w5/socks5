package resolver

import (
	"context"
	"net/netip"
	"time"
)

const (
	defaultResolverCacheMaxLen = 64
	defaultResolverCacheTTL    = 10 * time.Minute
)

type Resolver interface {
	Resolve(ctx context.Context, fqdn string) (netip.Addr, error)
}
