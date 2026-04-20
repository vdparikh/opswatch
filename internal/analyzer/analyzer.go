package analyzer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vdplabs/opswatch/internal/domain"
)

type Analyzer struct {
	policies []domain.Policy
}

func New(policies []domain.Policy) Analyzer {
	return Analyzer{policies: policies}
}

func (a Analyzer) AnalyzeEvents(ctx context.Context, events []domain.Event) ([]domain.Alert, error) {
	var alerts []domain.Alert
	state := domain.IncidentState{
		ProtectedDomains:       make(map[string]bool),
		ProtectedDomainDetails: make(map[string]domain.ProtectedDomain),
		AWSAccounts:            make(map[string]domain.AWSAccount),
		Services:               make(map[string]domain.Service),
	}

	for _, event := range events {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		updateState(&state, event)
		for _, p := range a.policies {
			alerts = append(alerts, p.Evaluate(event, state)...)
		}
		state.SeenEvents = append(state.SeenEvents, event)
	}

	return alerts, nil
}

func (a Analyzer) AnalyzeJSONL(ctx context.Context, r io.Reader) ([]domain.Alert, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var events []domain.Event

	line := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}

		var event domain.Event
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return a.AnalyzeEvents(ctx, events)
}

func updateState(state *domain.IncidentState, event domain.Event) {
	text := strings.ToLower(event.Text)

	if event.Source == domain.SourceRunbook {
		if intent := event.Context["intent"]; intent != "" {
			state.LatestIntent = intent
		}
		if action := event.Context["expected_action"]; action != "" {
			state.ExpectedAction = action
		}
	}

	if event.Source == domain.SourceSpeech {
		if strings.Contains(text, "add") || strings.Contains(text, "create") || strings.Contains(text, "delete") || strings.Contains(text, "apply") {
			state.LatestIntent = event.Text
		}
	}

	if env := event.Context["environment"]; env != "" {
		state.KnownEnvironment = env
	}

	if event.Source == domain.SourceAPI && event.Context["kind"] == "protected_domain" {
		if domainName := event.Context["domain"]; domainName != "" {
			key := strings.ToLower(domainName)
			state.ProtectedDomains[key] = true
			existing := state.ProtectedDomainDetails[key]
			state.ProtectedDomainDetails[key] = domain.ProtectedDomain{
				Name:                domainName,
				Environment:         firstNonEmpty(event.Context["environment"], existing.Environment),
				Owner:               firstNonEmpty(event.Context["owner"], existing.Owner),
				AuthoritativeZoneID: firstNonEmpty(event.Context["authoritative_zone_id"], existing.AuthoritativeZoneID),
				Risk:                firstNonEmpty(event.Context["risk"], existing.Risk),
			}
		}
	}

	if event.Source == domain.SourceAPI && event.Context["kind"] == "aws_account" {
		if accountID := event.Context["account_id"]; accountID != "" {
			state.AWSAccounts[accountID] = domain.AWSAccount{
				ID:          accountID,
				Name:        event.Context["account_name"],
				Environment: event.Context["environment"],
				Owner:       event.Context["owner"],
				Risk:        event.Context["risk"],
			}
		}
	}

	if event.Source == domain.SourceAPI && event.Context["kind"] == "service" {
		if service := event.Context["service"]; service != "" {
			state.Services[strings.ToLower(service)] = domain.Service{
				Name:        service,
				Environment: event.Context["environment"],
				Owner:       event.Context["owner"],
				Tier:        event.Context["tier"],
				Risk:        event.Context["risk"],
			}
		}
	}

	if accountID := event.Context["account_id"]; accountID != "" {
		if account, ok := state.AWSAccounts[accountID]; ok && account.Environment != "" {
			state.KnownEnvironment = account.Environment
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
