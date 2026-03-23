package ruleset

import "github.com/HT4w5/socks5/pkg/payload"

type RuleSetMode int

const (
	Whitelist RuleSetMode = iota // Rules match allowed requests
	Blacklist                    // Rules match disallowed requests
)

// RuleSet contains a set of rules that applies
// to a request for access control
type RuleSet struct {
	rules []Rule
	mode  RuleSetMode
}

func New(opts ...func(*RuleSet)) *RuleSet {
	rs := &RuleSet{
		mode: Blacklist,
	}
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func (rs *RuleSet) Allow(request *payload.Request) bool {
	switch rs.mode {
	case Whitelist:
		for _, r := range rs.rules {
			if r.Match(request) {
				return true
			}
		}
		return false
	case Blacklist:
		for _, r := range rs.rules {
			if r.Match(request) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func WithRule(rule Rule) func(*RuleSet) {
	return func(rs *RuleSet) {
		rs.rules = append(rs.rules, rule)
	}
}

func WithMode(mode RuleSetMode) func(*RuleSet) {
	return func(rs *RuleSet) {
		rs.mode = mode
	}
}
