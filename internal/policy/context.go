package policy

import (
	"strings"

	"github.com/vdplabs/opswatch/internal/domain"
)

type ContextPolicy struct{}

func (ContextPolicy) Evaluate(event domain.Event, state domain.IncidentState) []domain.Alert {
	if event.Source != domain.SourceScreen && event.Source != domain.SourceTerminal {
		return nil
	}

	accountID := strings.TrimSpace(event.Context["account_id"])
	if accountID == "" {
		accountID = strings.TrimSpace(event.Context["account"])
	}
	account, hasAccount := state.AWSAccounts[accountID]
	if !hasAccount || !isProduction(account.Environment) {
		return nil
	}

	combined := strings.ToLower(event.Text + " " + event.Context["action"] + " " + event.Context["command"])
	if !containsAny(combined, "create", "delete", "update", "edit", "apply", "deploy", "destroy", "rotate", "disable", "detach", "attach") {
		return nil
	}

	evidence := []string{
		"account: " + firstNonEmpty(account.Name, account.ID),
		"environment: " + account.Environment,
		"observed: " + event.Text,
	}
	if account.Owner != "" {
		evidence = append(evidence, "owner: "+account.Owner)
	}

	return []domain.Alert{{
		Timestamp:   event.Timestamp,
		Severity:    domain.SeverityCritical,
		Title:       "Production account change observed",
		Explanation: "Observed a mutating action in an AWS account marked as production by local OpsWatch context.",
		Evidence:    evidence,
		Source:      event.Source,
		Confidence:  0.80,
		Labels: map[string]string{
			"category":    "prod_account_change",
			"account_id":  account.ID,
			"environment": account.Environment,
			"owner":       account.Owner,
		},
	}}
}

func isProduction(environment string) bool {
	return strings.EqualFold(environment, "prod") || strings.EqualFold(environment, "production")
}
