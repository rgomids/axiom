# Periodic security and drift monitoring

Contract: [Issue #255](https://github.com/rgomids/axiom/issues/255), including
[the approved D1–D5 decisions](https://github.com/rgomids/axiom/issues/255#issuecomment-6089267928).
This is repository maintenance, outside Lingo's Execution/Provider lifecycle.
It adds no required check, release prerequisite, model call or remediation.

## Implementation contract and plan

Use one isolated schedule/dispatch workflow and standard-library Python tooling.
Daily analysis covers security and read-only Go module consistency/integrity.
Weekly analysis covers pinned no-credential CLI capabilities and existing Go
adapter, failure and compatibility tests. A separately permissioned reporting
job handles only monitoring-owned incidents. No product interface, dependency,
branch ruleset or release workflow changes are needed; no new ADR is warranted.

Security output is an allowlisted PASS/FAIL/category only. Raw advisory output
is never printed, saved or uploaded. Maintainers rerun locally and use
[the confidential reporting process](../../SECURITY.md) for details.

Acceptance requires offline success/failure/timeout/privacy/lifecycle tests,
repository and workflow validation, and real Actions runs in a controlled
sandbox proving create/update/recovery/recurrence. Until those runs are recorded
in the accompanying Evidence, operational acceptance remains pending.

## Running and operating

The daily layer runs at 05:17 UTC; the weekly layer runs Mondays at 06:37 UTC.
GitHub may delay/coalesce schedules. `workflow_dispatch` selects either layer;
the default scenario is `live`. Production always checks out `main`, never the
dispatch branch. A single non-cancelling concurrency group serializes both
analysis and incident reporting across layers. This is the incident writer's
lock: do not run the reporter concurrently outside this workflow.

```bash
python3 scripts/test-periodic-monitoring.py
python3 scripts/periodic-monitor.py run --layer daily --output /tmp/axiom-periodic
python3 scripts/periodic-monitor.py run --layer weekly --output /tmp/axiom-periodic
gh workflow run periodic-monitoring.yml --ref main -f layer=weekly -f scenario=live
```

Live analysis needs Go from `go.mod`, Git, Python 3.10+, Node/npm and public
network access. It uses a disposable HOME/configuration root and a credential
allowlist containing no credentials; stdin is closed. Weekly probes run only
version/help, never auth status/login, API requests, inference or provider
mutations. npm installs ignore lifecycle scripts; GitHub CLI extraction verifies
the release checksum, extracts just the expected regular binary and rejects
unsafe paths/types. No repository dependency file or installed user tool is
modified. npm optional binaries are used by the pinned packages.

Analysis has a 14-minute internal deadline, a 16-minute external command bound
and a 20-minute job timeout. Each command has a 30-second default timeout;
installation is 180 seconds, tidy/verify 120, security 240 and each Go suite
210 (with Go's own 180-second test bound). Expiring the shared budget creates
timeout evidence for remaining checks. POSIX process groups include child
processes and inherited pipes. Reporting has a 4-minute command and 5-minute
job bound; API requests time out after 20 seconds with at most five pages of
100 issues/comments. Inventory overflow refuses mutation instead of risking
duplicate incidents. There are no blind retries after ambiguous provider writes;
rerunning rediscovers existing state.

## Coverage and baseline policy

Daily uses `go mod tidy -diff` and `go mod verify` without rewriting files.
`govulncheck -json ./...` detects findings even if JSON mode returns exit code
zero. Empty/malformed output or invocation failure cannot pass. Security output
contains no package, advisory ID, CVE, description, trace or raw diagnostic.
The workflow summary directs maintainers to reproduce privately and follow
`SECURITY.md`; the job does not create a public vulnerability issue or retain
private output. There is no automatic private reporting API integration.

Weekly checks version and semantic root/subcommand capabilities for `gh`, Codex
and Claude. The baseline covers the repository's concrete `gh api`/issue adapter,
`codex exec --model`, `claude --print --model`, and auth-feature discovery
surfaces without executing authentication. Flags are whole tokens, so renamed
flags fail; help ordering, dates, spacing, ANSI colors and prose are irrelevant.
Snapshots contain only expected token names and boolean observations.

The weekly fixture simulations run existing suites in `runtimeadapter`,
`githubissues`, `executiongraph`, `graphapplication`, `compatibility` and
`runtimeprofile`. They exercise invocation contracts, provider responses,
failure handling and persisted-state compatibility without real model calls or
provider effects. Per-test evidence retains static top-level test declarations
and PASS/FAIL/SKIP; raw test output and dynamic subtest names are discarded.

Exact installation versions and semantic capabilities live in
[`periodic-baseline.json`](../../scripts/periodic-baseline.json). This monitors
those supported versions, not an automatically upgraded latest CLI. Updating a
baseline requires a reviewed PR, matching adapter/negative fixtures and new
operational evidence. A changed baseline cannot automatically recover an
incident from a different baseline; a maintainer must review and resolve it.
Go's version remains owned by `go.mod`. No automatic upgrades are performed.

## Evidence and failure categories

The versioned [`JSON schema`](../../scripts/periodic-evidence.schema.json) and
the reporter's strict semantic validator define the evidence boundary. Each
manifest identifies repository, workflow, run/attempt/URL, UTC timestamp,
`main` subject SHA, layer, baseline digest, overall outcome and check rows.
Rows contain component/contract, tool version, bounded duration, outcome,
category, stable fingerprint, before/after capabilities, reproduction guidance
and diagnostics references. Suites add safe per-test outcomes. Public fields
are allowlisted; unknown fields, unsafe versions/snapshots/reproduction text,
stale evidence and incorrect fingerprints are rejected before incident writes.

Stdout and stderr each retain at most 256 KiB in RAM and continue draining excess
to avoid deadlock; overflow is inconclusive. Raw content is never written to
disk. `manifest.json` is capped at 256 KiB; `summary.md` is built from fixed
columns. Only those two files are uploaded, with 14-day retention. Artifacts
are treated as public. Sensitive strings planted in raw outputs are checked
by the offline acceptance suite.

| Outcome | Categories | Meaning |
|---|---|---|
| PASS | `none` | Positive check |
| FAIL | `vulnerability`, `dependency_drift`, `contract_drift`, `unsupported_version`, `malformed_output`, `nonzero_exit` | Finding or definite contract/check failure; inspect sanitized comparison/tests |
| INCONCLUSIVE | `missing_binary`, `timeout`, `network`, `installation`, `runner`, `output_limit`, `artifact`, `permission` | Infrastructure/transport prevented proof; not a product regression |

Detected findings and inconclusive analysis both produce nonzero exits. Failed
analysis still uploads its sanitized evidence and reports incidents. Missing
artifact/upload/download/permission failures fail their job and emit a fixed
summary; they never count as recovery. PR/release nonblocking isolation means
these contexts are absent from required checks and existing workflow
dependencies, not that scheduled failure is disguised as success.

## Incident ownership and recovery

Only non-security findings are publicly tracked. Identity hashes check,
component, normalized category and contract, never run/time or raw text.
Infrastructure issues explicitly say INCONCLUSIVE; their identity differs from
proven drift. The writer uses existing `type:task`/`area:ci-cd` labels and never
adds lifecycle/status labels or changes human Work Item state.

Discovery includes closed bot-authored issues. Each issue has an exact owned
body with a versioned state marker, recurrence counter, last run/failure and
baseline digest. Repeated occurrences update that issue; the same run/attempt
is idempotent. Later recurrence reopens it. A bounded idempotent comment records
each occurrence/recovery with its revision and evidence URL, including repair
after an ambiguous write response. Unrelated issues and human-edited bodies
are excluded; duplicates require manual recovery instead of further creation.

Automatic resolution needs **two distinct newer workflow runs** positively
covering the same check scope, with complete successful layer coverage and
the original baseline digest. Same-run retries, partial/missing coverage,
transient failures, changed baselines, old results and another layer cannot
close an incident. Interrupted recovery resets its confirmation count.
Security is excluded at both creation and ownership boundaries.

For manual recovery, inspect evidence and rerun the applicable live layer.
Fix missing permissions/transports rather than treating them as regressions.
For intentionally retired/changed contracts, review the old incident and close
it manually; a body edit opts it out of automation. Do not modify the marker to
make an unrelated issue automation-owned. Confidential security findings use
the process above, never a public incident.

## Controlled operational validation

Sandbox fixtures are explicitly named `clean`, `drift`, `infrastructure` and
`security`, carry that scenario in their manifest, and are refused in
`rgomids/axiom`. Drift/infrastructure use the weekly layer; security uses daily.
The sandbox namespace is exactly `sandbox-255`. Dispatch live daily/weekly
checks as well as fixture runs; fixture PASS is not proof of live CLI/security
health. Sandbox checkouts may use the dispatched candidate branch.

Use a human-authorized sandbox. If its default branch has not yet registered the
new workflow, replay the exact candidate workflow on a test branch through an
existing registered workflow filename (for example `ci.yml`), documenting the
path alias; do not change the default branch, settings or production CI. Run
drift twice, clean twice, drift again, then clean twice. Preserve sanitized run
URLs, one issue's create/update/close/reopen history and final cleanup evidence.
Record outstanding operational blockers in
[Evidence #255](evidence-periodic-255.md); never claim completion from fixtures
or unexecuted Actions alone.
