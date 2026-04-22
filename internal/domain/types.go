package domain

import "time"

type Source string

const (
	SourceSpeech   Source = "speech"
	SourceScreen   Source = "screen"
	SourceTerminal Source = "terminal"
	SourceAPI      Source = "api"
	SourceRunbook  Source = "runbook"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Event struct {
	Timestamp time.Time         `json:"ts"`
	Source    Source            `json:"source"`
	Actor     string            `json:"actor,omitempty"`
	Text      string            `json:"text,omitempty"`
	Context   map[string]string `json:"context,omitempty"`
}

type IncidentState struct {
	LatestIntent     string
	ExpectedAction   string
	KnownEnvironment string
	ProtectedDomains map[string]bool
	SeenEvents       []Event
}

type Alert struct {
	Timestamp   time.Time         `json:"ts"`
	Severity    Severity          `json:"severity"`
	Title       string            `json:"title"`
	Explanation string            `json:"explanation"`
	Evidence    []string          `json:"evidence"`
	Source      Source            `json:"source"`
	Confidence  float64           `json:"confidence"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type Policy interface {
	Evaluate(event Event, state IncidentState) []Alert
}
