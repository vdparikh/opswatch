package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vdplabs/opswatch/internal/domain"
	"github.com/vdplabs/opswatch/internal/policy"
)

func TestRoute53ScreenshotGoldenEvents(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "r53_dns_events.jsonl")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	engine := New(policy.DefaultPolicies())
	alerts, err := engine.AnalyzeJSONL(context.Background(), file)
	if err != nil {
		t.Fatal(err)
	}

	requireAlert(t, alerts, "Possible DNS intent mismatch", domain.SeverityCritical)
	requireAlert(t, alerts, "Protected domain zone creation", domain.SeverityCritical)
}

func requireAlert(t *testing.T, alerts []domain.Alert, title string, severity domain.Severity) {
	t.Helper()
	for _, alert := range alerts {
		if alert.Title == title && alert.Severity == severity {
			return
		}
	}
	t.Fatalf("missing %s alert with %s severity; got %#v", title, severity, alerts)
}
