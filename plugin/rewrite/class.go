package rewrite

import (
	"context"
	"fmt"
	"strings"

	"github.com/coredns/coredns/request"

	"github.com/horahoradev/dns"
)

type classRule struct {
	fromClass  uint16
	toClass    uint16
	NextAction string
}

// newClassRule creates a class matching rule
func newClassRule(nextAction string, args ...string) (Rule, error) {
	var from, to uint16
	var ok bool
	if from, ok = dns.StringToClass[strings.ToUpper(args[0])]; !ok {
		return nil, fmt.Errorf("invalid class %q", strings.ToUpper(args[0]))
	}
	if to, ok = dns.StringToClass[strings.ToUpper(args[1])]; !ok {
		return nil, fmt.Errorf("invalid class %q", strings.ToUpper(args[1]))
	}
	return &classRule{from, to, nextAction}, nil
}

// Rewrite rewrites the current request.
func (rule *classRule) Rewrite(ctx context.Context, state request.Request) (ResponseRules, Result) {
	if rule.fromClass > 0 && rule.toClass > 0 {
		if state.Req.Question[0].Qclass == rule.fromClass {
			state.Req.Question[0].Qclass = rule.toClass
			return nil, RewriteDone
		}
	}
	return nil, RewriteIgnored
}

// Mode returns the processing mode.
func (rule *classRule) Mode() string { return rule.NextAction }
