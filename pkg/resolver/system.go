package resolver

import (
	"context"
	"fmt"
	"net"
	"net/netip"
)

type SystemResolver struct{}

func (r *SystemResolver) Resolve(ctx context.Context, fqdn string) (netip.Addr, error) {
	addr, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", fqdn)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to resolve fqdn: %w", err)
	}
	// TODO: implement address family policy
	return addr[0], nil
}
