package policy

import (
	"testing"
	"time"

	"github.com/vdplabs/opswatch/internal/domain"
)

func TestHighRiskPolicyDetectsSecurityGroupChange(t *testing.T) {
	p := HighRiskPolicy{}
	event := domain.Event{
		Timestamp: time.Date(2026, 4, 20, 20, 43, 0, 0, time.UTC),
		Source:    domain.SourceScreen,
		Text:      "Edit inbound rule for security group",
		Context: map[string]string{
			"action":        "edit",
			"resource_type": "security group",
		},
	}
	state := domain.IncidentState{KnownEnvironment: "prod"}

	alerts := p.Evaluate(event, state)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Title != "Network edge change observed" {
		t.Fatalf("unexpected alert %q", alerts[0].Title)
	}
	if alerts[0].Severity != domain.SeverityCritical {
		t.Fatalf("expected critical severity, got %q", alerts[0].Severity)
	}
}

func TestHighRiskPolicyDetectsBroadScopeCommand(t *testing.T) {
	p := HighRiskPolicy{}
	event := domain.Event{
		Timestamp: time.Date(2026, 4, 20, 20, 43, 0, 0, time.UTC),
		Source:    domain.SourceTerminal,
		Text:      "kubectl delete pods --all --namespace prod",
	}

	alerts := p.Evaluate(event, domain.IncidentState{})
	if len(alerts) == 0 {
		t.Fatal("expected broad-scope alert")
	}
}
