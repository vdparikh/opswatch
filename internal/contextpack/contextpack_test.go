package contextpack

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vdplabs/opswatch/internal/domain"
)

func TestLoadDirProducesContextEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "team.yaml")
	if err := os.WriteFile(path, []byte(`
incident:
  id: inc-123
  intent: Add a CNAME record for api.example.com
  expected_action: add CNAME record in existing hosted zone
  environment: prod
protected_domains:
  - name: example.com
    environment: prod
    owner: platform
    authoritative_zone_id: Z123
aws_accounts:
  - id: "123456789012"
    name: prod
    environment: prod
services:
  - name: api
    owner: platform
    environment: prod
`), 0o600); err != nil {
		t.Fatal(err)
	}

	events, err := LoadDir(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	requireEvent(t, events, domain.SourceRunbook, "incident_context")
	requireEvent(t, events, domain.SourceAPI, "protected_domain")
	requireEvent(t, events, domain.SourceAPI, "aws_account")
	requireEvent(t, events, domain.SourceAPI, "service")
}

func TestLoadDirIgnoresMissingDirectory(t *testing.T) {
	events, err := LoadDir(context.Background(), filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
}

func requireEvent(t *testing.T, events []domain.Event, source domain.Source, kind string) {
	t.Helper()
	for _, event := range events {
		if event.Source == source && event.Context["kind"] == kind {
			return
		}
	}
	t.Fatalf("missing %s event %q in %#v", source, kind, events)
}
