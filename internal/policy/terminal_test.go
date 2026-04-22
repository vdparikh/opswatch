package policy

import (
	"testing"
	"time"

	"github.com/vdplabs/opswatch/internal/domain"
)

func TestTerminalPolicyDetectsDestructiveProductionCommand(t *testing.T) {
	p := TerminalPolicy{}
	event := domain.Event{
		Timestamp: time.Date(2026, 4, 20, 20, 44, 0, 0, time.UTC),
		Source:    domain.SourceTerminal,
		Text:      "kubectl delete deployment api --namespace prod",
	}
	state := domain.IncidentState{KnownEnvironment: "prod"}

	alerts := p.Evaluate(event, state)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Severity != domain.SeverityCritical {
		t.Fatalf("expected critical severity, got %q", alerts[0].Severity)
	}
}
