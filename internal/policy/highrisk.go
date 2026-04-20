package policy

import (
	"strings"

	"github.com/vdplabs/opswatch/internal/domain"
)

type HighRiskPolicy struct{}

func (HighRiskPolicy) Evaluate(event domain.Event, state domain.IncidentState) []domain.Alert {
	if event.Source != domain.SourceScreen && event.Source != domain.SourceTerminal {
		return nil
	}

	text := strings.ToLower(event.Text)
	action := strings.ToLower(event.Context["action"] + " " + event.Context["command"])
	resource := strings.ToLower(event.Context["resource_type"] + " " + event.Context["resource"] + " " + event.Context["app"])
	combined := strings.TrimSpace(text + " " + action + " " + resource)
	if combined == "" {
		return nil
	}

	var alerts []domain.Alert
	for _, rule := range highRiskRules {
		if !matchesAny(combined, rule.match) {
			continue
		}
		if len(rule.actionHints) > 0 && !matchesAny(combined, rule.actionHints) {
			continue
		}
		alerts = append(alerts, domain.Alert{
			Timestamp:   event.Timestamp,
			Severity:    rule.severity(state.KnownEnvironment),
			Title:       rule.title,
			Explanation: rule.explanation,
			Evidence:    []string{"observed: " + event.Text},
			Source:      event.Source,
			Confidence:  rule.confidence,
			Labels: map[string]string{
				"category": rule.category,
			},
		})
	}
	return alerts
}

type highRiskRule struct {
	title       string
	explanation string
	category    string
	match       []string
	actionHints []string
	confidence  float64
	severity    func(string) domain.Severity
}

var highRiskRules = []highRiskRule{
	{
		title:       "Identity or access change observed",
		explanation: "Observed an IAM, permission, role, policy, key, or secret change. Access changes can have broad blast radius during incidents.",
		category:    "identity_access",
		match:       []string{"iam", "role", "policy", "permission", "access key", "secret", "kms", "service account"},
		actionHints: []string{"create", "delete", "detach", "attach", "update", "edit", "rotate", "disable", "grant", "revoke"},
		confidence:  0.74,
		severity:    severityForEnvironment,
	},
	{
		title:       "Network edge change observed",
		explanation: "Observed a change to routing, firewall, security group, load balancer, ingress, CDN, or WAF configuration.",
		category:    "network_edge",
		match:       []string{"security group", "firewall", "route table", "load balancer", "target group", "ingress", "cdn", "cloudfront", "waf", "listener"},
		actionHints: []string{"create", "delete", "update", "edit", "remove", "allow", "deny", "open", "route"},
		confidence:  0.72,
		severity:    severityForEnvironment,
	},
	{
		title:       "Infrastructure apply or deployment observed",
		explanation: "Observed an infrastructure apply, deployment, rollback, or release action. Verify environment, scope, and recent plan/output.",
		category:    "deployment_infra",
		match:       []string{"terraform", "apply", "plan", "deployment", "deploy", "rollback", "release", "helm", "kubernetes", "kubectl", "cloudformation"},
		actionHints: []string{"apply", "deploy", "delete", "destroy", "rollback", "upgrade", "release"},
		confidence:  0.70,
		severity:    severityForEnvironment,
	},
	{
		title:       "Database or data mutation observed",
		explanation: "Observed a database, table, backup, replication, migration, truncate, or delete action. Data changes are difficult to recover under pressure.",
		category:    "data_mutation",
		match:       []string{"database", "db", "table", "migration", "backup", "replica", "rds", "dynamodb", "truncate", "drop table"},
		actionHints: []string{"delete", "drop", "truncate", "restore", "migrate", "update", "disable"},
		confidence:  0.73,
		severity:    severityForEnvironment,
	},
	{
		title:       "Broad-scope change observed",
		explanation: "Observed wording or flags that suggest a broad or bulk operation. This usually deserves an explicit pause and second verification.",
		category:    "broad_scope",
		match:       []string{"all", "wildcard", "*", "global", "entire", "bulk", "force", "--all", "--force"},
		actionHints: []string{"delete", "remove", "replace", "update", "apply", "disable", "allow"},
		confidence:  0.68,
		severity: func(string) domain.Severity {
			return domain.SeverityCritical
		},
	},
}

func matchesAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
