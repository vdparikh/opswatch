package main

import (
	"testing"
	"time"

	"github.com/vdplabs/opswatch/internal/domain"
)

func TestFilterAlertCooldown(t *testing.T) {
	alert := domain.Alert{
		Severity: domain.SeverityCritical,
		Title:    "Possible DNS intent mismatch",
		Evidence: []string{"observed: Create hosted zone"},
	}
	lastAlertAt := make(map[string]time.Time)
	now := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)

	first := filterAlertCooldown([]domain.Alert{alert}, lastAlertAt, time.Minute, now)
	if len(first) != 1 {
		t.Fatalf("expected first alert through, got %d", len(first))
	}

	second := filterAlertCooldown([]domain.Alert{alert}, lastAlertAt, time.Minute, now.Add(30*time.Second))
	if len(second) != 0 {
		t.Fatalf("expected duplicate alert suppressed, got %d", len(second))
	}

	third := filterAlertCooldown([]domain.Alert{alert}, lastAlertAt, time.Minute, now.Add(2*time.Minute))
	if len(third) != 1 {
		t.Fatalf("expected alert after cooldown, got %d", len(third))
	}
}

func TestParseCaptureRect(t *testing.T) {
	rect, ok, err := parseCaptureRect("600,0,1440,1000")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected rect to be present")
	}
	if rect.X != 600 || rect.Y != 0 || rect.Width != 1440 || rect.Height != 1000 {
		t.Fatalf("unexpected rect: %#v", rect)
	}
}

func TestParseCaptureRectEmpty(t *testing.T) {
	_, ok, err := parseCaptureRect("")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no rect")
	}
}
