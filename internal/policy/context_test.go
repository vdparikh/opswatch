package policy

import (
	"testing"
	"time"

	"github.com/vdplabs/opswatch/internal/domain"
)

func TestContextPolicyAlertsOnProductionAccountMutation(t *testing.T) {
	state := domain.IncidentState{
		AWSAccounts: map[string]domain.AWSAccount{
			"123456789012": {
				ID:          "123456789012",
				Name:        "prod",
				Environment: "prod",
				Owner:       "platform",
			},
		},
	}
	event := domain.Event{
		Timestamp: time.Now(),
		Source:    domain.SourceScreen,
		Text:      "Update security group rule",
		Context: map[string]string{
			"account_id": "123456789012",
			"action":     "update",
		},
	}

	alerts := ContextPolicy{}.Evaluate(event, state)
	if len(alerts) != 1 {
		t.Fatalf("expected one alert, got %d", len(alerts))
	}
	if alerts[0].Severity != domain.SeverityCritical {
		t.Fatalf("expected critical severity, got %s", alerts[0].Severity)
	}
}

func TestContextPolicyIgnoresNonMutatingProductionAccountView(t *testing.T) {
	state := domain.IncidentState{
		AWSAccounts: map[string]domain.AWSAccount{
			"123456789012": {ID: "123456789012", Environment: "prod"},
		},
	}
	event := domain.Event{
		Timestamp: time.Now(),
		Source:    domain.SourceScreen,
		Text:      "Viewing EC2 instances",
		Context: map[string]string{
			"account_id": "123456789012",
		},
	}

	alerts := ContextPolicy{}.Evaluate(event, state)
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %d", len(alerts))
	}
}
