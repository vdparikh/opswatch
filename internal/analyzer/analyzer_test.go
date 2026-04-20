package analyzer

import (
	"context"
	"strings"
	"testing"

	"github.com/vdplabs/opswatch/internal/policy"
)

func TestAnalyzeJSONLProducesDNSMismatchAlert(t *testing.T) {
	events := strings.NewReader(`
{"ts":"2026-04-20T20:42:00Z","source":"api","text":"domain registry loaded","context":{"kind":"protected_domain","domain":"example.com"}}
{"ts":"2026-04-20T20:42:10Z","source":"speech","actor":"incident-commander","text":"Add a CNAME record for api.example.com"}
{"ts":"2026-04-20T20:42:30Z","source":"screen","actor":"operator","text":"AWS Route53 Create hosted zone example.com","context":{"action":"create","resource_type":"hosted_zone","domain":"example.com","environment":"prod"}}
`)
	engine := New(policy.DefaultPolicies())

	alerts, err := engine.AnalyzeJSONL(context.Background(), events)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}
}

func TestAnalyzeJSONLPreservesProtectedDomainDetails(t *testing.T) {
	events := strings.NewReader(`
{"ts":"2026-04-20T20:42:00Z","source":"api","text":"context","context":{"kind":"protected_domain","domain":"example.com","owner":"platform","authoritative_zone_id":"Z123"}}
{"ts":"2026-04-20T20:42:01Z","source":"api","text":"legacy context","context":{"kind":"protected_domain","domain":"example.com"}}
{"ts":"2026-04-20T20:42:30Z","source":"screen","actor":"operator","text":"AWS Route53 Create hosted zone example.com","context":{"action":"create","resource_type":"hosted_zone","domain":"example.com","environment":"prod"}}
`)
	engine := New(policy.DefaultPolicies())

	alerts, err := engine.AnalyzeJSONL(context.Background(), events)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, alert := range alerts {
		if alert.Title != "Protected domain zone creation" {
			continue
		}
		found = true
		evidence := strings.Join(alert.Evidence, "\n")
		if !strings.Contains(evidence, "authoritative zone: Z123") || !strings.Contains(evidence, "owner: platform") {
			t.Fatalf("expected enriched evidence, got %#v", alert.Evidence)
		}
	}
	if !found {
		t.Fatal("missing protected domain alert")
	}
}
