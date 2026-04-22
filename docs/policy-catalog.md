# Policy Catalog

OpsWatch policies should start narrow, concrete, and high-signal. A good policy names the observed action, explains blast radius, and avoids generic warnings.

## Current Baseline Pack

DNS:

- intent mismatch: stated mitigation and observed DNS console action do not line up
- protected domain zone creation, enriched with owner, environment, and authoritative zone ID when supplied by context packs
- generic DNS zone creation when no intent is known

Context:

- mutating action in an AWS account marked `prod` or `production` by local context packs

Terminal:

- known destructive commands such as `kubectl delete`, `terraform apply`, `helm uninstall`, and DNS delete commands
- broad destructive selectors such as `--all`

High-risk operations:

- identity and access changes: IAM, roles, policies, permissions, keys, secrets, service accounts
- network edge changes: security groups, firewalls, route tables, load balancers, ingress, CDN, WAF
- infrastructure apply/deployment: Terraform, Kubernetes, Helm, CloudFormation, rollback, release
- database and data mutation: database delete/drop/truncate/restore/migration actions
- broad-scope operations: global, wildcard, all, force, bulk changes

## Policy Tiers

Tier 1 should work without incident context:

- destructive action observed
- protected resource mutation
- broad-scope operation
- high-blast-radius object creation or deletion

Tier 2 uses lightweight context:

- environment is prod
- domain/resource is protected
- account/region/cluster is production
- owner/service metadata is known from local context packs
- actor is using break-glass access

Tier 3 uses incident intent:

- compare spoken intent to observed action
- compare runbook expected action to observed action
- compare ticket/Slack mitigation plan to observed action

## Design Rules

- Prefer specific alerts over broad advice.
- Emit fewer alerts with stronger evidence.
- Explain why this action is risky.
- Include the observed action as evidence.
- Escalate severity with production/protected context.
- Keep raw screenshots ephemeral; store structured events.
