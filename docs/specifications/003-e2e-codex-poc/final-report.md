# Final Report — Axiom E2E Codex POC

## Final status

**ACCEPTED — 2026-09-20**

The human maintainer completed manual acceptance testing and accepted the bounded
E2E Codex POC. PR #52 is merged to `main`; #21, #30 and #40 are closed as
completed. Acceptance does not authorize MVP work or mark the remaining
Specification 002 Task DAG complete.

## Summary

The POC now provides one complete local path:

```text
install Axiom from source
-> lingo on user PATH
-> configure Codex and five user-global Axiom skills
-> configure Project with portable keys and local working-copy paths
-> resolve Project independently of caller CWD
-> create/select GitHub Issue Work Item
-> execute Specification / Clarification / Plan / Tasks
-> Implementation / Review / Evidence / Reconciliation
-> explicitly authorize Work Item completion
```

Direct Lingo and `$axiom-*` Codex skills call the same application behavior.
Skills contain no duplicate Project, provider or workflow rules.

## User journey

1. `scripts/install-axiom.sh` builds a revisioned Lingo binary into an explicit
   user PATH destination with a protected ownership receipt, reports dirty source
   metadata and gives safe PATH guidance without changing shell profiles.
2. `lingo runtime codex install` publishes five exact user-global skills and
   `status` verifies them.
3. `lingo project configure` writes logical repository keys to portable intent
   and absolute working-copy paths only to protected local state.
4. `project show/resolve` returns the Project and repository paths from any CWD.
5. Bounded `work-item` commands use the configured repository's GitHub `origin`
   and existing `gh` authentication; provider mutations require explicit
   `--authorize-external`.
6. Persistent ordered workflow gates record repository-relative artifacts and
   SHA-256 digests. Failure is inspectable and requires explicit resume.
7. Completion becomes available only after reconciliation and closes the linked
   Work Item only with explicit external authority.

## Issues delivered

| Issue | Delivery | Status |
|---|---|---|
| #31 | Specification 003, clarifications, Plan and Tasks | Closed / PR #41 |
| #32 | source-checkout installer and PATH binary | Closed / PR #42 |
| #33 | Codex Runtime and five global skills | Closed / PRs #43–#44 |
| #34 | CWD-independent Project/repository resolution | Closed / PR #45 |
| #35 | guided and repeatable Project configuration | Closed / PR #46 |
| #36 | bounded GitHub Work Item capabilities | Closed / PR #47 |
| #37 | persistent sequential workflow | Closed / PR #48 |
| #38 | equivalent human/JSON/skill entrypoints | Closed / PR #49 |
| #39 | deterministic and real Runtime dogfooding Evidence | Closed / PR #50 |
| #40 | final reconciliation and acceptance preparation | Accepted / closed |
| #19 | filesystem proof beyond controlled POC cases | Open; explicit gap |
| #20 | dependent persistence/recovery Evidence gate | Open; explicit gap |
| #21 | baseline dogfood/reconciliation | Accepted / closed |
| #30 | parent E2E outcome | Accepted / closed |

## Architecture

No new material ADR was required. Existing decisions remain intact:

- Axiom owns domain/contracts; Lingo is its local control plane (ADR-0003).
- Project remains distinct from Repository (ADR-0001).
- portable `axiom.yaml` contains no machine paths or secrets; bindings stay in
  protected local state (ADR-0004).
- Codex and GitHub are adapters at Runtime/Provider boundaries.
- the supported flow is `Codex Skill -> Lingo -> application/workflow`.

Implementation packages keep resolver, provider, workflow, presentation and
filesystem responsibilities separate. Sequential execution does not introduce
a multi-agent graph contract.

## #30 exit-criteria mapping

| Exit criterion | Reproducible Evidence |
|---|---|
| source install and PATH | `scripts/test-install-axiom.sh`; T32 Evidence; T39 dogfood |
| global Codex discovery | `scripts/test-codex-skills.sh`; T33 and T39 Runtime task Evidence |
| CWD-independent Project resolution | resolver/black-box tests; T34/T35/T39 Evidence |
| CLI and Codex Project configuration | shared `Configure` application method; T35/T38 tests and docs |
| GitHub Work Item create/select | Work Item unit/integration tests; T36; live #39 selection |
| complete bounded workflow | workflow/CLI black-box tests; T37; completed #39 workflow |
| thin skill delegation | embedded-skill contract tests; T33/T38 Evidence |
| equivalent authority/validation | CLI adapter tests; denied provider/completion checks in T36–T39 |
| deterministic tests/Evidence | macOS/Linux POC workflow; T31–T39 Evidence; final validation |
| limitations and foundation gates | this report; Specification 002 `evidence-poc.md`; #19/#20 comments |
| explicit human acceptance | accepted by maintainer on 2026-09-20; #21/#30/#40 closed |

Post-provider failure Results retain Project/repository identity, Issue URL and
confirmed state. Workflow completion propagates the same payload if either Work
Item linkage or workflow persistence fails after GitHub confirms `CLOSED`.

## Validation

The following passed locally on macOS 15 / Go 1.26.1 and in the POC verification
workflow on macOS 15 plus Ubuntu 24.04 for intermediate issue PRs:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
./scripts/dogfood-poc.sh
gitleaks git --no-banner --redact --log-opts='--all'
```

Staged sensitive-file checks passed before every issue commit. Final branch
validation and review passed before PR #52 merged to `main`; manual human
acceptance followed on 2026-09-20.

## Dogfooding Evidence

- [Structured T39 Evidence](evidence-t39.json)
- [Narrative/reproduction T39 Evidence](evidence-t39.md)
- Projectless Codex task `01a0bb43-9b1a-7513-92d3-b56d9dbee8bf`
- real Project `axiom-e2e-poc`, GitHub Work Item #39, and completed persisted
  workflow with artifact digests
- deterministic `scripts/dogfood-poc.sh` covering clean install through completion

The Runtime task began outside the Project, discovered the global skill, supplied
no path, and received the configured repository path from Lingo. Its isolated
sandbox required a separate UI approval for cross-workspace writes; this was
recorded rather than bypassed.

## Known limitations

- Only Codex is supported as Runtime; no Runtime-general compatibility claim.
- Standalone Codex rejects literal `axiom:<skill>` names. The validated reversible
  mapping is `$axiom-<skill>`.
- Only GitHub Issues via repository `origin` and existing `gh` auth are supported;
  Lingo is not a general GitHub client.
- Workflow is one sequential agent; no multi-agent topology, parallel DAG,
  retries/budgets or remote control plane.
- Installer is source-checkout/user-local. No Homebrew, apt, registry, signing,
  notarization, production auto-update or compatibility guarantee. Binary and
  receipt roots do not form a cross-filesystem transaction.
- Known Axiom skill upgrades are atomic per file after full preflight, not one
  transaction spanning all five skill directories.
- Local bindings/workflow state do not synchronize across machines.
- Automatic recovery, physical power-loss durability, the complete fault matrix
  and arbitrary hostile same-UID filesystem races remain unresolved in #19/#20.
- Optional Project documents, full rename and remaining Specification 002 Tasks
  are not authorized or implied by this POC.

## Human validation guide

Clean deterministic reproduction without live external mutation:

```bash
git clone https://github.com/rgomids/axiom.git
cd axiom
git checkout main
git pull --ff-only
./scripts/dogfood-poc.sh
```

User-global install and Runtime inspection:

```bash
./scripts/install-axiom.sh
export PATH="$HOME/.local/bin:$PATH"
cd "$HOME"
command -v lingo
lingo version
lingo runtime codex install
lingo runtime codex status
lingo help
```

Real Project/Work Item inspection using an existing open GitHub Issue:

```bash
lingo project configure \
  --slug acceptance-project \
  --name "Acceptance Project" \
  --repository main=/absolute/path/to/working-copy

cd "$HOME"
lingo --json project show --selector acceptance-project
lingo --json work-item select \
  --project acceptance-project --repository main --number ISSUE_NUMBER
lingo --json workflow start \
  --project acceptance-project --repository main --number ISSUE_NUMBER
```

Then invoke `$axiom-work-item-status` from Codex with only Project,
repository-key and Work Item selectors. Follow [Commands](../../commands.md#execute-the-bounded-workflow)
for gate advancement. Use `--authorize-external` only when deliberately creating,
commenting on or closing the validation Issue.

## Human acceptance

The maintainer accepted the bounded POC on 2026-09-20 after manual validation of
the merged flow:

```text
install -> Codex -> Project -> Work Item -> workflow
-> implementation -> review -> Evidence -> reconciliation -> completion
```

The acceptance explicitly retains #19/#20 as known persistence/recovery proof
gaps rather than silently treating them as complete.

The same validation identified four MVP follow-ups:

- [#54](https://github.com/rgomids/axiom/issues/54) — guided, explicit Project
  configuration interview instead of assuming the Runtime CWD.
- [#55](https://github.com/rgomids/axiom/issues/55) — Intent-driven Work Item
  creation with a structured draft before external mutation.
- [#56](https://github.com/rgomids/axiom/issues/56) — provider-visible workflow
  stage markers and bounded structured transition comments.
- [#57](https://github.com/rgomids/axiom/issues/57) — inline Runtime skill
  selectors/arguments to reduce unnecessary conversational round trips.

These are MVP direction, not retroactive POC blockers and not automatic
implementation authority.
