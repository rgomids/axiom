# Clarifications — E2E Codex POC

## C1 — Delivery authority

The human request dated 2026-09-19 approves Specification, Plan, Tasks and
implementation for #31–#40, including issue branches, PRs, intermediate squash
merges, issue reconciliation and technical closure. It does not authorize merging
the final POC PR to `main` or claiming final human acceptance.

## C2 — Codex skill naming

Requested `axiom:<skill>` names were tested against the current supported
standalone skill contract. Official Codex documentation shows `$skill-name`, and
the bundled validator enforces `^[a-z0-9-]+$`; `:` is rejected. The accepted
bounded fallback is `axiom-<skill>`, invoked as `$axiom-<skill>`. This is a host
compatibility divergence, not a new Axiom domain name.

## C3 — User-global skill root

Use the official current user scope, `$HOME/.agents/skills`, with an explicit
override for isolated tests. Do not write portable Project intent or repository
paths into skill files.

## C4 — Workflow automation boundary

Lingo persists and validates the ordered workflow state, resolved repository,
bounded validation commands and Evidence. The Codex agent performs the actual
Specification/Plan/implementation/review work. Lingo does not attempt to embed or
replace an autonomous coding agent in this POC.

## C5 — GitHub transport

Use the installed authenticated `gh` CLI as the bounded POC transport behind a
GitHub adapter. This does not make Transport a domain concept or store its token.
Tests use a deterministic fake executable. Live dogfooding records provider
references separately.
