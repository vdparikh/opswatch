package contextpack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vdplabs/opswatch/internal/domain"
	"gopkg.in/yaml.v3"
)

type Pack struct {
	Incident         Incident          `json:"incident" yaml:"incident"`
	ProtectedDomains []ProtectedDomain `json:"protected_domains" yaml:"protected_domains"`
	AWSAccounts      []AWSAccount      `json:"aws_accounts" yaml:"aws_accounts"`
	Services         []Service         `json:"services" yaml:"services"`
	Runbooks         []Runbook         `json:"runbooks" yaml:"runbooks"`
}

type Incident struct {
	ID             string `json:"id" yaml:"id"`
	Title          string `json:"title" yaml:"title"`
	Intent         string `json:"intent" yaml:"intent"`
	ExpectedAction string `json:"expected_action" yaml:"expected_action"`
	Environment    string `json:"environment" yaml:"environment"`
	Service        string `json:"service" yaml:"service"`
}

type ProtectedDomain struct {
	Name                string `json:"name" yaml:"name"`
	Environment         string `json:"environment" yaml:"environment"`
	Owner               string `json:"owner" yaml:"owner"`
	AuthoritativeZoneID string `json:"authoritative_zone_id" yaml:"authoritative_zone_id"`
	Risk                string `json:"risk" yaml:"risk"`
}

type AWSAccount struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Environment string `json:"environment" yaml:"environment"`
	Owner       string `json:"owner" yaml:"owner"`
	Risk        string `json:"risk" yaml:"risk"`
}

type Service struct {
	Name        string `json:"name" yaml:"name"`
	Environment string `json:"environment" yaml:"environment"`
	Owner       string `json:"owner" yaml:"owner"`
	Tier        string `json:"tier" yaml:"tier"`
	Risk        string `json:"risk" yaml:"risk"`
}

type Runbook struct {
	ID             string   `json:"id" yaml:"id"`
	Title          string   `json:"title" yaml:"title"`
	Service        string   `json:"service" yaml:"service"`
	Environment    string   `json:"environment" yaml:"environment"`
	ExpectedAction string   `json:"expected_action" yaml:"expected_action"`
	AllowedActions []string `json:"allowed_actions" yaml:"allowed_actions"`
}

func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".opswatch", "context")
}

func LoadDir(ctx context.Context, dir string) ([]domain.Event, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return LoadFile(ctx, dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if supported(path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	var events []domain.Event
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		fileEvents, err := LoadFile(ctx, path)
		if err != nil {
			return nil, err
		}
		events = append(events, fileEvents...)
	}
	return events, nil
}

func LoadFile(ctx context.Context, path string) ([]domain.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pack Pack
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		err = json.Unmarshal(data, &pack)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &pack)
	default:
		return nil, fmt.Errorf("unsupported context pack extension %q", filepath.Ext(path))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return pack.Events(filepath.Base(path)), nil
}

func (p Pack) Events(source string) []domain.Event {
	now := time.Now().UTC()
	var events []domain.Event

	if p.Incident.Intent != "" || p.Incident.ExpectedAction != "" || p.Incident.Environment != "" || p.Incident.Service != "" {
		context := map[string]string{
			"kind": "incident_context",
		}
		put(context, "incident_id", p.Incident.ID)
		put(context, "title", p.Incident.Title)
		put(context, "intent", p.Incident.Intent)
		put(context, "expected_action", p.Incident.ExpectedAction)
		put(context, "environment", p.Incident.Environment)
		put(context, "service", p.Incident.Service)
		put(context, "context_pack", source)
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceRunbook,
			Actor:     "context-pack",
			Text:      firstNonEmpty(p.Incident.Title, p.Incident.Intent, "Loaded incident context"),
			Context:   context,
		})
	}

	for _, item := range p.ProtectedDomains {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		context := map[string]string{
			"kind": "protected_domain",
		}
		put(context, "domain", strings.ToLower(strings.TrimSpace(item.Name)))
		put(context, "environment", item.Environment)
		put(context, "owner", item.Owner)
		put(context, "authoritative_zone_id", item.AuthoritativeZoneID)
		put(context, "risk", item.Risk)
		put(context, "context_pack", source)
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceAPI,
			Actor:     "context-pack",
			Text:      "Loaded protected domain " + item.Name,
			Context:   context,
		})
	}

	for _, item := range p.AWSAccounts {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		context := map[string]string{
			"kind": "aws_account",
		}
		put(context, "account_id", item.ID)
		put(context, "account_name", item.Name)
		put(context, "environment", item.Environment)
		put(context, "owner", item.Owner)
		put(context, "risk", item.Risk)
		put(context, "context_pack", source)
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceAPI,
			Actor:     "context-pack",
			Text:      "Loaded AWS account " + firstNonEmpty(item.Name, item.ID),
			Context:   context,
		})
	}

	for _, item := range p.Services {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		context := map[string]string{
			"kind": "service",
		}
		put(context, "service", item.Name)
		put(context, "environment", item.Environment)
		put(context, "owner", item.Owner)
		put(context, "tier", item.Tier)
		put(context, "risk", item.Risk)
		put(context, "context_pack", source)
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceAPI,
			Actor:     "context-pack",
			Text:      "Loaded service " + item.Name,
			Context:   context,
		})
	}

	for _, item := range p.Runbooks {
		if item.ExpectedAction == "" && len(item.AllowedActions) == 0 {
			continue
		}
		context := map[string]string{
			"kind": "runbook",
		}
		put(context, "runbook_id", item.ID)
		put(context, "title", item.Title)
		put(context, "service", item.Service)
		put(context, "environment", item.Environment)
		put(context, "expected_action", item.ExpectedAction)
		put(context, "allowed_actions", strings.Join(item.AllowedActions, ","))
		put(context, "context_pack", source)
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceRunbook,
			Actor:     "context-pack",
			Text:      firstNonEmpty(item.Title, item.ExpectedAction, "Loaded runbook context"),
			Context:   context,
		})
	}

	return events
}

func supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func put(values map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		values[key] = value
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
