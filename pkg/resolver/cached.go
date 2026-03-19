package resolver

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"
)

type CachedResolver struct {
	cache  map[string]record
	res    *net.Resolver
	ttl    time.Duration
	maxLen int
}

type record struct {
	addr      netip.Addr
	expiresAt time.Time
}

func NewCachedResolver(opts ...func(*CachedResolver)) *CachedResolver {
	r := &CachedResolver{
		maxLen: defaultResolverCacheMaxLen,
		ttl:    defaultResolverCacheTTL,
		cache:  make(map[string]record),
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.res == nil {
		r.res = net.DefaultResolver
	}

	return r
}

func WithMaxLen(maxLen int) func(*CachedResolver) {
	return func(c *CachedResolver) {
		c.maxLen = maxLen
	}
}

func WithTTL(ttl time.Duration) func(*CachedResolver) {
	return func(c *CachedResolver) {
		c.ttl = ttl
	}
}

func WithResolver(r *net.Resolver) func(*CachedResolver) {
	return func(c *CachedResolver) {
		c.res = r
	}
}

func (r *CachedResolver) Resolve(ctx context.Context, fqdn string) (netip.Addr, error) {
	if addr, ok := r.queryCache(fqdn); ok {
		return addr, nil
	}

	addr, err := r.res.LookupNetIP(ctx, "ip4", fqdn)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to resolve fqdn: %w", err)
	}

	// TODO: implement address family policy

	r.storeCache(fqdn, addr[0])

	return addr[0], nil
}

func (r *CachedResolver) storeCache(fqdn string, addr netip.Addr) {
	if len(r.cache) >= r.maxLen {
		clear(r.cache)
	}
	r.cache[fqdn] = record{
		addr:      addr,
		expiresAt: time.Now().Add(r.ttl),
	}
}

func (r *CachedResolver) queryCache(fqdn string) (netip.Addr, bool) {
	res, ok := r.cache[fqdn]
	if !ok {
		return netip.Addr{}, false
	}

	if res.expiresAt.Before(time.Now()) {
		return netip.Addr{}, false // Expired
	}

	return res.addr, true
}
