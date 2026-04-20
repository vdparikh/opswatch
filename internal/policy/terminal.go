package policy

import (
	"strings"

	"github.com/vdplabs/opswatch/internal/domain"
)

type TerminalPolicy struct{}

func (TerminalPolicy) Evaluate(event domain.Event, state domain.IncidentState) []domain.Alert {
	if event.Source != domain.SourceTerminal {
		return nil
	}

	command := strings.ToLower(event.Text)
	if command == "" {
		return nil
	}

	if containsAny(command, "terraform apply", "kubectl delete", "helm uninstall", "aws route53 delete", "gcloud dns managed-zones delete") {
		return []domain.Alert{{
			Timestamp:   event.Timestamp,
			Severity:    severityForEnvironment(state.KnownEnvironment),
			Title:       "Destructive or high-risk command observed",
			Explanation: "Observed a command pattern that can mutate or delete production infrastructure during an incident.",
			Evidence:    []string{"command: " + event.Text},
			Source:      event.Source,
			Confidence:  0.82,
			Labels: map[string]string{
				"category":    "dangerous_command",
				"environment": state.KnownEnvironment,
			},
		}}
	}

	if strings.Contains(command, "--all") && containsAny(command, "delete", "remove", "destroy") {
		return []domain.Alert{{
			Timestamp:   event.Timestamp,
			Severity:    domain.SeverityCritical,
			Title:       "Broad destructive command observed",
			Explanation: "Observed a destructive command with a broad selector. This usually deserves an explicit pause and second human verification.",
			Evidence:    []string{"command: " + event.Text},
			Source:      event.Source,
			Confidence:  0.78,
			Labels: map[string]string{
				"category": "broad_destructive_action",
			},
		}}
	}

	return nil
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func severityForEnvironment(environment string) domain.Severity {
	if strings.EqualFold(environment, "prod") || strings.EqualFold(environment, "production") {
		return domain.SeverityCritical
	}
	return domain.SeverityWarning
}
