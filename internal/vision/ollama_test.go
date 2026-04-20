package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestOllamaAnalyzeImage(t *testing.T) {
	var sawImage bool
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var req ollamaGenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "test-vision" {
			t.Fatalf("unexpected model %q", req.Model)
		}
		if len(req.Images) != 1 {
			t.Fatalf("expected one image, got %d", len(req.Images))
		}
		if req.Options["num_predict"] == nil {
			t.Fatal("expected num_predict option")
		}
		if _, err := base64.StdEncoding.DecodeString(req.Images[0]); err != nil {
			t.Fatalf("image was not base64: %v", err)
		}
		sawImage = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewBufferString(`{"response":"{\"source\":\"screen\",\"text\":\"AWS Route53 Create hosted zone example.com\",\"context\":{\"action\":\"create\",\"resource_type\":\"hosted_zone\",\"domain\":\"example.com\"}}","done":true}`)),
			Header:     make(http.Header),
		}, nil
	})

	imagePath := filepath.Join(t.TempDir(), "frame.png")
	if err := os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0o600); err != nil {
		t.Fatal(err)
	}

	client := NewOllamaClient("test-vision", "http://ollama.test/api/generate", 0)
	client.HTTPClient = &http.Client{Transport: transport}
	event, err := client.AnalyzeImage(context.Background(), imagePath, FrameContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !sawImage {
		t.Fatal("server did not receive image")
	}
	if event.Context["resource_type"] != "hosted_zone" {
		t.Fatalf("unexpected resource_type %q", event.Context["resource_type"])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
