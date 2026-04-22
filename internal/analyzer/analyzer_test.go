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
