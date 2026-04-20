package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vdplabs/opswatch/internal/domain"
)

const defaultEndpoint = "https://api.openai.com/v1/responses"

type ImageAnalyzer interface {
	AnalyzeImage(ctx context.Context, imagePath string, frame FrameContext) (domain.Event, error)
}

type OpenAIClient struct {
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

type FrameContext struct {
	Intent           string
	ExpectedAction   string
	ProtectedDomains []string
	Environment      string
	Actor            string
}

func NewOpenAIClientFromEnv(model string) (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required for image analysis")
	}
	if model == "" {
		model = "gpt-4.1-mini"
	}
	return &OpenAIClient{
		APIKey:   apiKey,
		Model:    model,
		Endpoint: defaultEndpoint,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func (c *OpenAIClient) AnalyzeImage(ctx context.Context, imagePath string, frame FrameContext) (domain.Event, error) {
	dataURL, err := dataURLForImage(imagePath)
	if err != nil {
		return domain.Event{}, err
	}

	prompt := buildPrompt(frame)
	reqBody := responsesRequest{
		Model: c.Model,
		Input: []responsesInput{{
			Role: "user",
			Content: []responsesContent{
				{Type: "input_text", Text: prompt},
				{Type: "input_image", ImageURL: dataURL, Detail: "high"},
			},
		}},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return domain.Event{}, err
	}

	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return domain.Event{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.Event{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.Event{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.Event{}, fmt.Errorf("openai vision request failed: %s: %s", resp.Status, string(body))
	}

	outputText, err := extractOutputText(body)
	if err != nil {
		return domain.Event{}, err
	}

	event, err := parseVisionEvent(outputText)
	if err != nil {
		return domain.Event{}, err
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Source == "" {
		event.Source = domain.SourceScreen
	}
	if event.Actor == "" {
		event.Actor = frame.Actor
	}
	if event.Context == nil {
		event.Context = make(map[string]string)
	}
	event.Context["image_path"] = imagePath
	if frame.Environment != "" && event.Context["environment"] == "" {
		event.Context["environment"] = frame.Environment
	}
	return event, nil
}

type responsesRequest struct {
	Model string           `json:"model"`
	Input []responsesInput `json:"input"`
}

type responsesInput struct {
	Role    string             `json:"role"`
	Content []responsesContent `json:"content"`
}

type responsesContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

func buildPrompt(frame FrameContext) string {
	var b strings.Builder
	b.WriteString("You are OpsWatch, a live incident change witness. Analyze this screenshot from an incident bridge.\n")
	b.WriteString("Return only one JSON object with fields: source, text, context.\n")
	b.WriteString("Use source=\"screen\". The text field should be a concise description of the operational action visible on screen.\n")
	b.WriteString("The context object should include any known action, resource_type, domain, environment, app, account, region, command, and risk_hint values. Use empty strings only when unknown.\n")
	b.WriteString("Focus on cloud consoles, DNS pages, terminals, Terraform, Kubernetes, CI/CD, feature flags, databases, load balancers, and destructive or high-blast-radius changes.\n")
	b.WriteString("If the screen is not operationally relevant, still summarize it with app and risk_hint=\"none\".\n")
	if frame.Intent != "" {
		b.WriteString("\nCurrent stated intent: ")
		b.WriteString(frame.Intent)
	}
	if frame.ExpectedAction != "" {
		b.WriteString("\nExpected runbook action: ")
		b.WriteString(frame.ExpectedAction)
	}
	if frame.Environment != "" {
		b.WriteString("\nKnown environment: ")
		b.WriteString(frame.Environment)
	}
	if len(frame.ProtectedDomains) > 0 {
		b.WriteString("\nProtected domains: ")
		b.WriteString(strings.Join(frame.ProtectedDomains, ", "))
	}
	return b.String()
}

func dataURLForImage(path string) (string, error) {
	encoded, mediaType, err := imageBase64(path)
	if err != nil {
		return "", err
	}
	return "data:" + mediaType + ";base64," + encoded, nil
}

func imageBase64(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	mediaType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mediaType == "" {
		mediaType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(mediaType, "image/") {
		return "", "", fmt.Errorf("%s does not look like an image file", path)
	}
	return base64.StdEncoding.EncodeToString(data), mediaType, nil
}

func extractOutputText(body []byte) (string, error) {
	var response struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Text string `json:"text"`
				Type string `json:"type"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.OutputText) != "" {
		return strings.TrimSpace(response.OutputText), nil
	}
	for _, item := range response.Output {
		for _, content := range item.Content {
			if strings.TrimSpace(content.Text) != "" {
				return strings.TrimSpace(content.Text), nil
			}
		}
	}
	return "", errors.New("openai response did not contain output text")
}

func parseVisionEvent(output string) (domain.Event, error) {
	cleaned := strings.TrimSpace(output)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var raw struct {
		Source  domain.Source     `json:"source"`
		Text    string            `json:"text"`
		Context map[string]string `json:"context"`
		TS      string            `json:"ts"`
		Actor   string            `json:"actor"`
	}
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return domain.Event{}, fmt.Errorf("parse vision event JSON: %w; output: %s", err, output)
	}

	var ts time.Time
	if raw.TS != "" {
		parsed, err := time.Parse(time.RFC3339, raw.TS)
		if err != nil {
			return domain.Event{}, err
		}
		ts = parsed
	}

	return domain.Event{
		Timestamp: ts,
		Source:    raw.Source,
		Actor:     raw.Actor,
		Text:      raw.Text,
		Context:   raw.Context,
	}, nil
}
