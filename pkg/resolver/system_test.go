package resolver_test

import (
	"testing"

	"github.com/HT4w5/socks5/pkg/resolver"
)

func TestSystemResolver_SimpleResolve(t *testing.T) {
	var r *resolver.SystemResolver

	got, err := r.Resolve(t.Context(), "localhost")
	if err != nil {
		t.Errorf("resolve error: %v", err)
	}

	if !got.IsLoopback() {
		t.Errorf("bad resolve result, expected loopback, got %s", got)
	}
}

func TestSystemResolver_NXDOMAIN(t *testing.T) {
	var r *resolver.SystemResolver

	_, err := r.Resolve(t.Context(), "nosuchdomain")
	if err == nil {
		t.Errorf("expected error")
	}
}
