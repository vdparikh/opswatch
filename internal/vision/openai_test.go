package vision

import (
	"testing"

	"github.com/vdplabs/opswatch/internal/domain"
)

func TestExtractOutputTextAndParseVisionEvent(t *testing.T) {
	body := []byte(`{
		"output": [{
			"content": [{
				"type": "output_text",
				"text": "{\"source\":\"screen\",\"text\":\"AWS Route53 Create hosted zone example.com\",\"context\":{\"action\":\"create\",\"resource_type\":\"hosted_zone\",\"domain\":\"example.com\",\"environment\":\"prod\"}}"
			}]
		}]
	}`)

	text, err := extractOutputText(body)
	if err != nil {
		t.Fatal(err)
	}

	event, err := parseVisionEvent(text)
	if err != nil {
		t.Fatal(err)
	}
	if event.Source != domain.SourceScreen {
		t.Fatalf("expected screen source, got %q", event.Source)
	}
	if event.Context["resource_type"] != "hosted_zone" {
		t.Fatalf("expected hosted_zone resource, got %q", event.Context["resource_type"])
	}
}
