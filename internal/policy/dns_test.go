package policy

import (
	"testing"
	"time"

	"github.com/vdplabs/opswatch/internal/domain"
)

func TestDNSPolicyDetectsZoneCreationWhenIntentIsRecordChange(t *testing.T) {
	p := DNSPolicy{}
	event := domain.Event{
		Timestamp: time.Date(2026, 4, 20, 20, 43, 0, 0, time.UTC),
		Source:    domain.SourceScreen,
		Text:      "AWS Route53 Create hosted zone example.com",
		Context: map[string]string{
			"action":        "create",
			"resource_type": "hosted_zone",
			"domain":        "example.com",
		},
	}
	state := domain.IncidentState{
		LatestIntent:     "Add a CNAME record for api.example.com",
		ProtectedDomains: map[string]bool{"example.com": true},
	}

	alerts := p.Evaluate(event, state)
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}
	if alerts[0].Severity != domain.SeverityCritical {
		t.Fatalf("expected critical severity, got %q", alerts[0].Severity)
	}
}
