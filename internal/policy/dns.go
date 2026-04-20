package policy

import (
	"strings"

	"github.com/vdplabs/opswatch/internal/domain"
)

type DNSPolicy struct{}

func (DNSPolicy) Evaluate(event domain.Event, state domain.IncidentState) []domain.Alert {
	if event.Source != domain.SourceScreen && event.Source != domain.SourceTerminal {
		return nil
	}

	text := strings.ToLower(event.Text)
	action := strings.ToLower(event.Context["action"])
	resourceType := strings.ToLower(event.Context["resource_type"])
	domainName := strings.ToLower(event.Context["domain"])

	creatingZone := strings.Contains(text, "create hosted zone") ||
		strings.Contains(text, "create primary zone") ||
		strings.Contains(text, "create dns zone") ||
		action == "create" && strings.Contains(resourceType, "zone")

	if !creatingZone {
		return nil
	}

	intent := strings.ToLower(state.LatestIntent + " " + state.ExpectedAction)
	intentMentionsRecord := strings.Contains(intent, "record") || strings.Contains(intent, "cname") || strings.Contains(intent, "a record") || strings.Contains(intent, "txt")

	var alerts []domain.Alert
	if intentMentionsRecord {
		alerts = append(alerts, domain.Alert{
			Timestamp:   event.Timestamp,
			Severity:    domain.SeverityCritical,
			Title:       "Possible DNS intent mismatch",
			Explanation: "Observed DNS zone creation while current intent appears to be adding or changing a DNS record.",
			Evidence: []string{
				"intent: " + firstNonEmpty(state.LatestIntent, state.ExpectedAction),
				"observed: " + event.Text,
			},
			Source:     event.Source,
			Confidence: 0.88,
			Labels: map[string]string{
				"category":      "intent_mismatch",
				"resource_type": "dns_zone",
				"domain":        domainName,
			},
		})
	}

	if domainName != "" && state.ProtectedDomains[domainName] {
		alerts = append(alerts, domain.Alert{
			Timestamp:   event.Timestamp,
			Severity:    domain.SeverityCritical,
			Title:       "Protected domain zone creation",
			Explanation: "Observed creation of a DNS zone for a protected domain. This can change authoritative DNS behavior and has high blast radius.",
			Evidence: []string{
				"protected domain: " + domainName,
				"observed: " + event.Text,
			},
			Source:     event.Source,
			Confidence: 0.92,
			Labels: map[string]string{
				"category": "blast_radius",
				"domain":   domainName,
			},
		})
	}

	if len(alerts) == 0 {
		alerts = append(alerts, domain.Alert{
			Timestamp:   event.Timestamp,
			Severity:    domain.SeverityWarning,
			Title:       "DNS zone creation observed",
			Explanation: "Observed creation of a DNS zone. This can be high blast radius and may deserve human verification during an incident.",
			Evidence: []string{
				"observed: " + event.Text,
			},
			Source:     event.Source,
			Confidence: 0.72,
			Labels: map[string]string{
				"category":      "dns_zone_creation",
				"resource_type": "dns_zone",
				"domain":        domainName,
			},
		})
	}

	return alerts
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown"
}
