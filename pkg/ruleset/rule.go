package ruleset

import "github.com/HT4w5/socks5/pkg/payload"

type Rule interface {
	Match(request *payload.Request) bool
}

type And struct {
}
