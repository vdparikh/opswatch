package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vdplabs/opswatch/internal/analyzer"
	"github.com/vdplabs/opswatch/internal/capture"
	"github.com/vdplabs/opswatch/internal/domain"
	"github.com/vdplabs/opswatch/internal/policy"
	"github.com/vdplabs/opswatch/internal/report"
	"github.com/vdplabs/opswatch/internal/vision"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "opswatch: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage()
	}

	switch args[0] {
	case "analyze":
		return runAnalyze(ctx, args[1:])
	case "analyze-image":
		return runAnalyzeImage(ctx, args[1:])
	case "watch":
		return runWatch(ctx, args[1:])
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() error {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  opswatch analyze --events <events.jsonl>")
	fmt.Fprintln(os.Stderr, "  opswatch analyze-image --image <screenshot.png> [--intent <text>] [--expected-action <text>]")
	fmt.Fprintln(os.Stderr, "  opswatch watch [--interval 2s] [--intent <text>] [--expected-action <text>]")
	return nil
}

func runAnalyze(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	eventsPath := fs.String("events", "", "path to JSONL incident events")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *eventsPath == "" {
		return fmt.Errorf("--events is required")
	}

	file, err := os.Open(*eventsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	engine := analyzer.New(policy.DefaultPolicies())
	alerts, err := engine.AnalyzeJSONL(ctx, file)
	if err != nil {
		return err
	}

	switch *format {
	case "text":
		return report.WriteText(os.Stdout, alerts)
	case "json":
		return report.WriteJSON(os.Stdout, alerts)
	default:
		return fmt.Errorf("unsupported format %q", *format)
	}
}

func runAnalyzeImage(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("analyze-image", flag.ContinueOnError)
	imagePath := fs.String("image", "", "path to screenshot/image")
	intent := fs.String("intent", "", "current stated operator intent")
	expectedAction := fs.String("expected-action", "", "expected runbook action")
	environment := fs.String("environment", "", "known environment, such as prod")
	protectedDomains := fs.String("protected-domain", "", "comma-separated protected domains")
	model := fs.String("model", "", "OpenAI vision-capable model")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *imagePath == "" {
		return fmt.Errorf("--image is required")
	}

	events, err := imageEvents(ctx, *imagePath, vision.FrameContext{
		Intent:           *intent,
		ExpectedAction:   *expectedAction,
		Environment:      *environment,
		ProtectedDomains: splitCSV(*protectedDomains),
		Actor:            "local-operator",
	}, *model)
	if err != nil {
		return err
	}

	engine := analyzer.New(policy.DefaultPolicies())
	alerts, err := engine.AnalyzeEvents(ctx, events)
	if err != nil {
		return err
	}
	return writeAlerts(*format, alerts)
}

func runWatch(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	interval := fs.Duration("interval", 2*time.Second, "capture interval")
	intent := fs.String("intent", "", "current stated operator intent")
	expectedAction := fs.String("expected-action", "", "expected runbook action")
	environment := fs.String("environment", "", "known environment, such as prod")
	protectedDomains := fs.String("protected-domain", "", "comma-separated protected domains")
	model := fs.String("model", "", "OpenAI vision-capable model")
	captureDir := fs.String("capture-dir", filepath.Join(os.TempDir(), "opswatch-frames"), "directory for temporary captures")
	once := fs.Bool("once", false, "capture and analyze one frame")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *interval <= 0 {
		return fmt.Errorf("--interval must be greater than zero")
	}
	if err := os.MkdirAll(*captureDir, 0o755); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	frame := vision.FrameContext{
		Intent:           *intent,
		ExpectedAction:   *expectedAction,
		Environment:      *environment,
		ProtectedDomains: splitCSV(*protectedDomains),
		Actor:            "local-operator",
	}
	capturer := capture.MacOSCapture{}

	for {
		imagePath := filepath.Join(*captureDir, fmt.Sprintf("frame-%d.png", time.Now().UnixNano()))
		if err := capturer.Fullscreen(ctx, imagePath); err != nil {
			return err
		}

		events, err := imageEvents(ctx, imagePath, frame, *model)
		if err != nil {
			return err
		}
		engine := analyzer.New(policy.DefaultPolicies())
		alerts, err := engine.AnalyzeEvents(ctx, events)
		if err != nil {
			return err
		}
		if len(alerts) > 0 {
			if err := report.WriteText(os.Stdout, alerts); err != nil {
				return err
			}
		}

		if *once {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(*interval):
		}
	}
}

func imageEvents(ctx context.Context, imagePath string, frame vision.FrameContext, model string) ([]domain.Event, error) {
	events := make([]domain.Event, 0, 3+len(frame.ProtectedDomains))
	now := time.Now().UTC()
	for _, domainName := range frame.ProtectedDomains {
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceAPI,
			Text:      "Loaded protected domain policy",
			Context: map[string]string{
				"kind":   "protected_domain",
				"domain": domainName,
			},
		})
	}
	if frame.ExpectedAction != "" {
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceRunbook,
			Text:      "Expected action",
			Context: map[string]string{
				"expected_action": frame.ExpectedAction,
			},
		})
	}
	if frame.Intent != "" {
		events = append(events, domain.Event{
			Timestamp: now,
			Source:    domain.SourceSpeech,
			Actor:     "operator",
			Text:      frame.Intent,
		})
	}

	client, err := vision.NewOpenAIClientFromEnv(model)
	if err != nil {
		return nil, err
	}
	screenEvent, err := client.AnalyzeImage(ctx, imagePath, frame)
	if err != nil {
		return nil, err
	}
	events = append(events, screenEvent)
	return events, nil
}

func writeAlerts(format string, alerts []domain.Alert) error {
	switch format {
	case "text":
		return report.WriteText(os.Stdout, alerts)
	case "json":
		return report.WriteJSON(os.Stdout, alerts)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func splitCSV(value string) []string {
	var values []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, strings.ToLower(part))
		}
	}
	return values
}
