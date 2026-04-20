# OpsWatch Architecture

## Shape

```mermaid
flowchart LR
    Laptop["Laptop watcher"] --> Capture["Selected app/window capture"]
    Zoom["Zoom / Meet bot"] --> Vision["Frame sampler + OCR / vision"]
    Capture --> Vision
    Slack["Slack incident channel"] --> Intent["Intent extractor"]
    Runbook["Runbook / ticket"] --> Intent
    Cloud["Read-only cloud APIs"] --> State["State inventory"]
    Terminal["Terminal OCR"] --> Events["Normalized event stream"]
    Vision --> Events
    Intent --> Events
    State --> Events
    Events --> Analyzer["Analyzer"]
    Analyzer --> Policy["Policy engine"]
    Policy --> Alerts["Slack / Zoom alerts"]
    Policy --> Timeline["Post-incident timeline"]
```

## Event Stream

Adapters convert messy inputs into normalized observations:

- screen frame summaries
- command snippets
- speech transcript snippets
- runbook expectations
- read-only infrastructure state

The analyzer should not care whether a screen event came from Zoom, screenshots, browser automation, or a replay file.

## Current Vision Path

The current implementation has two entry points:

- `analyze-image`: analyze one screenshot/image
- `watch`: repeatedly capture the macOS full screen and analyze each frame

Both paths call OpenAI vision through the Responses API and ask for a normalized `screen` event. That keeps the rest of the system model-agnostic: policies only see operational events, not raw images.

The next capture milestone is selected-window watching:

- list visible windows and apps
- let the operator choose Zoom, browser, terminal, or another app
- capture only that window
- skip frames when nothing materially changed
- keep raw images ephemeral unless debug retention is explicitly enabled

## Policy Engine

Policies evaluate each event against rolling incident state. State includes:

- latest stated intent
- expected runbook action
- environment/account/region hints
- protected domains and resources
- recent observed actions

## First Policies

DNS policy:

- detect hosted zone creation
- compare against record-change intent
- flag protected domains

Terminal policy:

- detect destructive commands
- increase severity in production
- flag broad selectors

## Privacy Posture

OpsWatch should be designed so enterprise buyers can approve it:

- explicit bot participant
- incident-only activation
- ephemeral video processing
- no raw screen retention by default
- redact secrets from event summaries
- store structured timeline, not full recordings
