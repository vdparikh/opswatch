# OpsWatch Product Spec

## One-Liner

OpsWatch is a live incident change witness that verifies whether observed operational changes match the stated intent, runbook, and safety policy.

## Problem

Incident bridges rely on screen sharing and human attention. That creates visibility, but not reliable verification. Operators are often stressed, moving fast, and working in unfamiliar consoles. Reviewers may see the screen without catching semantic mistakes like wrong account, wrong region, wrong object type, or destructive flags.

The motivating failure mode:

- intended action: create or update a DNS record
- actual action: create a primary DNS zone with zero records
- result: broad DNS outage

## Wedge

Start with high-precision alerts for DNS and terminal changes during declared incidents.

OpsWatch should detect:

- zone creation when the intent is record creation
- protected domain mutation
- production account or environment mismatch
- destructive terminal commands
- broad selectors like `--all`

## User Experience

OpsWatch joins the incident bridge as an explicit participant, watches shared operational context, and posts short alerts to the incident channel.

Good alert:

> Possible DNS intent mismatch: observed creation of hosted zone `example.com`, but current intent is to add a CNAME record in an existing zone. Blast radius: protected production DNS.

Bad alert:

> This might be risky. Please review.

## MVP

The first implementation should support:

- Zoom or meeting frame ingestion through a pluggable adapter
- OCR/vision summaries normalized into `screen` events
- speech transcript snippets normalized into `speech` events
- read-only DNS inventory normalized into `api` events
- policy-driven alerting
- Slack or text output
- post-incident timeline export

The current repo starts at the analyzer boundary, with JSONL events standing in for future adapters.

## Non-Goals

- do not record full meeting video by default
- do not try to understand every possible UI
- do not block all changes in v1
- do not emit generic AI safety warnings

## Principles

- high precision over high coverage
- alerts must name the exact observed action
- policy should explain why the action is risky
- raw video should be ephemeral by default
- stored artifacts should prefer structured events and alert summaries

