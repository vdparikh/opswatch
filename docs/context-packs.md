# Context Packs

OpsWatch uses local context packs to understand company-specific infrastructure during an incident. Vision describes what appears on screen; context packs tell OpsWatch which domains, accounts, services, and runbook actions are in scope.

By default, OpsWatch reads YAML or JSON files from:

```text
~/.opswatch/context
```

You can override this path with `--context-dir` or `OPSWATCH_CONTEXT_DIR`. The menu bar app exposes the same value as `Context directory` in `Settings...`.

## Quickstart

Create a starter pack:

```bash
opswatch context init
opswatch context inspect
```

Then run with context:

```bash
go run ./cmd/opswatch analyze-image \
  --vision-provider ollama \
  --model llama3.2-vision \
  --image examples/r53_dns.png \
  --context-dir ~/.opswatch/context
```

The context pack can provide intent and expected action, so those CLI flags become optional when the pack contains active incident context.

## Schema

```yaml
incident:
  id: inc-demo
  title: Demo DNS incident
  intent: Add a CNAME record for api.example.com
  expected_action: add CNAME record in existing hosted zone
  environment: prod
  service: api

protected_domains:
  - name: example.com
    environment: prod
    owner: platform
    authoritative_zone_id: Z123456789
    risk: critical

aws_accounts:
  - id: "123456789012"
    name: prod
    environment: prod
    owner: platform
    risk: critical

services:
  - name: api
    environment: prod
    owner: application-platform
    tier: tier-0
    risk: critical

runbooks:
  - id: dns-add-cname
    title: Add API CNAME
    service: api
    environment: prod
    expected_action: add CNAME record in existing hosted zone
    allowed_actions:
      - route53.change_record
```

## How Context Is Used

OpsWatch converts context pack entries into normal incident events before analyzing a screenshot or event file.

- `incident` fills current intent, expected action, service, and environment when CLI flags are not provided.
- `protected_domains` enrich DNS policies with owner, environment, risk, and authoritative zone ID.
- `aws_accounts` lets policies recognize production account mutations when vision extracts an `account_id`.
- `services` records ownership and criticality for future service-aware policies.
- `runbooks` provides expected action hints and allowed action names for policy evolution.

Context packs are local files. They are not uploaded by OpsWatch.
