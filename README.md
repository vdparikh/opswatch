# OpsWatch

OpsWatch is an incident change witness: it compares what operators intend to do during an incident with what is actually being changed on screen, in terminals, and through infrastructure APIs.

The first prototype is intentionally narrow. It reads a stream of observed incident events and emits precise alerts when a dangerous action does not match the stated intent or safety policy.

## Why

During incident bridges, screen share gives visibility but not verification. People can see a console or terminal, yet still miss the exact account, object type, region, command flag, or blast radius.

OpsWatch is built around the delta between:

- spoken or written intent
- observed operational action
- known infrastructure state
- incident policy

Example:

> Intent: add a DNS record
>
> Observed: create a new primary DNS zone
>
> Alert: possible intent mismatch with high DNS blast radius

## Current Prototype

This repo currently includes:

- a Go CLI: `opswatch analyze`
- JSON event ingestion for speech, screen, terminal, API, and runbook observations
- screenshot/image analysis through OpenAI vision
- a macOS fullscreen watcher prototype using `screencapture`
- DNS and terminal safety policies
- high-signal alert output
- a sample incident stream based on the DNS-zone-vs-record failure mode

## Try It

```bash
go test ./...
go run ./cmd/opswatch analyze --events examples/dns_incident.jsonl
```

Expected output includes a critical alert when a hosted zone creation is observed while the stated intent is to add a DNS record.

## Analyze A Screenshot

Set an OpenAI API key, then pass a screenshot into the same analyzer pipeline:

```bash
export OPENAI_API_KEY=...

go run ./cmd/opswatch analyze-image \
  --image /path/to/screenshot.png \
  --intent "Add a CNAME record for api.example.com" \
  --expected-action "add CNAME record in existing hosted zone" \
  --protected-domain example.com \
  --environment prod
```

The vision step converts the image into a normalized `screen` event, then the regular OpsWatch policies decide whether to alert.

## Start Watching

On macOS, the prototype can capture the full screen repeatedly and analyze each frame:

```bash
export OPENAI_API_KEY=...

go run ./cmd/opswatch watch \
  --interval 2s \
  --intent "Add a CNAME record for api.example.com" \
  --expected-action "add CNAME record in existing hosted zone" \
  --protected-domain example.com \
  --environment prod
```

This is the early laptop mode. The next adapter should target a selected app/window instead of the full screen, so OpsWatch can watch Zoom, a browser, or a terminal without sending unrelated desktop pixels.

## Event Model

OpsWatch consumes JSON Lines events. Each line is one observation:

```json
{"ts":"2026-04-20T20:42:10Z","source":"speech","actor":"incident-commander","text":"Add a CNAME record for api.example.com"}
```

Important event sources:

- `speech`: transcript snippets from Zoom or the bridge
- `screen`: OCR or vision summaries from shared screen frames
- `terminal`: commands and output extracted from terminals
- `api`: read-only infrastructure state
- `runbook`: expected action from runbook or ticket context

## Product Direction

The near-term wedge is DNS and terminal verification:

- Route53, Cloudflare, Azure DNS, and GCP DNS console flows
- `aws route53`, `gcloud dns`, `az network dns`, and common shell commands
- environment/account mismatch
- zone creation vs record creation
- protected domain mutations
- destructive command patterns

Later adapters can feed the same analyzer from Zoom, Slack, OCR, browser automation, read-only cloud APIs, and incident management systems.
