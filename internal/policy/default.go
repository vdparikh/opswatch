package policy

import "github.com/vdplabs/opswatch/internal/domain"

func DefaultPolicies() []domain.Policy {
	return []domain.Policy{
		DNSPolicy{},
		TerminalPolicy{},
	}
}
