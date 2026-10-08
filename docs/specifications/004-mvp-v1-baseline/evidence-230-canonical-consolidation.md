# Issue #230 — Canonical consolidation Evidence

Date: 2026-10-08. Baseline: refreshed `origin/main` and local `main` at
`c7260797aad7865419f555eea94a2eef7ae78285` (v0.10.0). Initial tracked worktree
clean. At the local validation checkpoint, no commits, push, merge, release, Issue
mutation or human acceptance had occurred. #229 is closed historically; #230 remains open
with published implementation and human acceptance pending.

## Change and architectural reconciliation

[Approved contract / plan / tasks](issue-230-canonical-consolidation.md) records
the human decision superseding #229's six compatibility entrypoints. Existing
ADRs remain sufficient; no new accepted architecture or ADR status is inferred.
Specification 004, Plan, Tasks and matrix §10 link this amendment. Historical
#229/#230 Evidence is annotated with a current reconciliation link; original
observations, decisions and release fixtures are preserved.

| Surface | Historical | Current distributed/installed target |
|---|---|---|
| pre-#229, e.g. v0.6.0 | six operation-specific skills | two canonical skills |
| #229 through v0.10.0 | eight skills, including canonical pair | two canonical skills |
| fresh / repeated install | zero / current pair | `axiom-project`, `axiom-work-item` |

Runtime stays a thin semantic/argument/completion surface. Natural-language
intent can select clear supported lifecycle operations without English CLI
spelling. Ambiguity never supplies authority. Parser FlagSets own arguments;
executable operation metadata owns routing. Tests mechanically check catalog,
SKILL.md routing tables, modes/effects/authority, current lifecycle matrix,
parser arguments and concrete application dispatch. All underlying supported
CLI operations remain; legacy names fail inspection/discovery.

D2 assessment: `cmd/lingo/project_lifecycle.go` composes presentation.
`installation.SelectForInspection`, `projectapp.InspectProjectState`,
operational View and application readiness Evaluate own identity/state/admission
and readiness decisions. CLI's `local.RepositoryProbe` displays availability
for declared local bindings; next text and completion status render the report.
No leaked domain decision justified a move; no cosmetic refactor performed.

## Changed-file overview

- `internal/codexruntime`: canonical inventory, internal ownership inventory,
  known historical digests, anchored retirement, receipt/status/upgrade/recovery,
  two active skills, historical fixtures and migration/security tests.
- `internal/install`: closed two-skill archive, reviewed `skill_retire` effects,
  exact expected digests and interruption/resume tests.
- `internal/cli`: canonical-only help/inspection, shared catalog parity and
  authority/dispatch tests; `cmd/lingo` black-box and workflow list tests.
- `internal/local`, `internal/compatibility`: operational corpus, historical
  readers, writer coverage, POC preservation with historical-only ownership API.
- `internal/runtimebootstrap`: exact two-skill parity and idempotence checks.
- Installer/archive/facade/dogfood/native/upgrade-journey scripts: current
  inventory expectations; `.github/workflows/release-artifacts.yml`: explicit
  frozen compatibility gate before artifact build.
- README English/Portuguese, installation/command docs, Specification 004,
  Plan/Tasks/matrix and reconciliation/acceptance Evidence.

## Migration, ownership and recovery

Current format and compatibility versions remain unchanged: no codec or
wire-format change requires a bump; receipt/manifest digests distinguish exact
sets. Published revisions remain recognized internally. Historical v0.6.0 six
and v0.10.0 eight skill fixture bytes were compared with their exact tag paths;
[provenance](../../../internal/codexruntime/testdata/published-skills/PROVENANCE.md)
records their purpose.

Installer preflight classifies every current and retired candidate, rejects
foreign/malformed receipts, unknown/modified/linked content and unexpected
files. Retirement uses locked, anchored private root/child handles, known
content digests and object-identity rechecks. It removes only `SKILL.md` and its
verified directory; no recursive skill-root deletion occurs. Known bytes prove
content ownership; an empty directory requires historical receipt attestation
for that specific name before being treated as interrupted cleanup.

Owned binary upgrade previews include individual digest-bound `skill_retire`
effects. Confirmed effects and recovery markers remain truthful after each
interruption. Canonical publication/receipt follows retirement; the historical
receipt remains available until the whole skill set converges. Rerun rechecks
remaining content and refuses a user edit made between effects. Runtime install
and first-run use the same Codex/Claude implementation and ownership rules.

## D1 — Persisted-state compatibility

Confirmed original finding: exact v0.10.0 tag writes format-1 operational.json,
while its stable corpus lacked that representation. The tag was not rewritten.
Its exact source was exported to a temporary isolated checkout; its existing
writer test generated a retrospective snapshot. Sixteen files and MANIFEST
entries were appended; previous snapshots/manifest entries were unchanged.
[Corpus provenance](../../../internal/compatibility/testdata/stable-v1/PROVENANCE)
records revision, generation command, isolation and retrospective limitation.
This is reproducible historical writer proof, not release-time Evidence.

Current reader loads the frozen archived Project and disabled `work-items`
Integration. Operational kind now belongs to required v1 corpus; the corpus
guard also covers additive kinds. Release preparation clears the generator
variable and runs local/compatibility tests before building. A contract test
pins that enforced ordering. Future writers must be inventoried, exercised and
frozen; adding an additive kind no longer exempts it from corpus coverage.

## Executed local validation

Environment: macOS 27.0.1 arm64/APFS; Go 1.26.1. Tests use private temporary
resource roots; published Runtime/Provider behavior is represented by stubs
unless stated otherwise. No real user installation or Provider was mutated.

| Command / check | Actual outcome |
|---|---|
| `gofmt -l cmd/lingo internal/cli internal/codexruntime internal/compatibility internal/install internal/local internal/runtimebootstrap` | PASS, no files listed |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go mod verify` | PASS, all modules verified |
| `go test ./... -count=1` | Final PASS; cmd/lingo 87.764s. Earlier run compiled stale eight-skill assertions and failed; corrected and rerun |
| `go test -race ./...` | PASS; cmd/lingo 288.863s |
| `AXIOM_FREEZE_STATE_CORPUS= go test ./internal/local ./internal/compatibility -count=1` | PASS (39.090s / 4.561s), includes new release gate contract after broad run |
| `go test -race ./internal/compatibility -count=1` | PASS (5.075s), including new gate test |
| `./scripts/test-codex-skills.sh` | PASS Go contracts; optional Python quick-validator dependency unavailable, explicitly WARN |
| `./scripts/test-release-archives.sh` | PASS current-source two-skill archive, installed binary, bootstrap, repeat install, receipt/conflicts/recovery |
| `./scripts/test-install-posix-facade.sh` | PASS, failures=0 |
| `./scripts/test-install-bootstrap.sh` | PASS, failures=0; offline transport/Provider fixtures |
| `./scripts/dogfood-poc.sh` | PASS, globalSkillCount=2, workflow completed; fake Runtime/GitHub; repeated with isolated HOME/Codex/Claude roots |
| `./scripts/validate-repository.sh .` | BLOCKED by preexisting ignored `.claude/worktrees/agent-ae82234ded7063ddc/.claude`; preserved unchanged |
| `./scripts/validate-repository.sh <isolated-current-source-copy>` | PASS; complete Git-index/unignored-file copy with maintainer symlinks, excludes ignored external worktrees |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `gitleaks dir . --redact --no-banner --exit-code 1 --log-level error` | PASS, exit 0 |
| `git diff --check` and scoped diff review | PASS |
| `./scripts/test-s7-native.sh` | BLOCKED, exit 78: historical exact native row requires macOS 27.0; host 27.0.1. Policy not changed here |

Targeted package and fault tests passed, including full `internal/codexruntime`,
`internal/install`, `internal/local`, `internal/compatibility`, `internal/cli`,
`cmd/lingo`, runtime bootstrap and historical exact-tag writer freeze.
Workflow list tests cover human/JSON parity, empty/3/300 Executions, deterministic
ordering/repeated output, missing Project, damaged state and zero write effects.

## Published-binary skill migration observations

Downloaded only public native archives and SHA256SUMS. Verified checksum,
bounded archive paths/no links before extraction. Each published binary ran its
own first-run with isolated HOME, Projects, State, Codex, Claude and fake Runtime
executables. Candidate came from current-source development archive built with
`--version 9999.0.0-acceptance.1 --development`; this is not a clean release.

| Published baseline | Native archive SHA256 | Observed migration |
|---|---|---|
| v0.6.0 | `ddd54dc0b43c211500bec56f188145a44a472a0623519982916a0fab3898c047` | 6→2 on Codex and Claude; foreign sentinel preserved; rerun byte-identical |
| v0.10.0 | `b9f2277e09d4728cacf393e5ba9e8ed37d5d81c2d9da53a8c1b090702ac21e22` | 8→2 on Codex and Claude; foreign sentinel preserved; rerun byte-identical |

Installed canonical bytes equal candidate archive bytes. Both Runtime status
commands succeed; canonical inspection succeeds; legacy inspection exits 1.
Candidate manifest has exactly two rows:

```text
skill.axiom-project=64054953abe8bcdac1226b6a1a711879e4cfe7d83379c82f18cd10a445635041
skill.axiom-work-item=09028588aec1447ec51842ffd7c9ae2b41830b2dd093e087e18c35f61f499a9e
```

This proves real published skill/receipt bytes migrate through the installed
candidate binary. It is not the full clean-provenance binary upgrade journey.
Automated install tests cover the binary effect sequence with controlled clean
candidate provenance and interruption fixtures. The full
`test-upgrade-journeys.sh` and clean-source `test-release-pipeline.sh` remain for
the approved committed candidate; running the latter now would test committed
v0.10.0 instead of these uncommitted changes.

## Acceptance ledger

PASS means local technical Evidence, never human acceptance.

| AC | Evidence / status |
|---|---|
| AC-01 | PASS: two embedded entries, exact archive/embedded inventory test |
| AC-02 | PASS: fresh installed binary/archive first-run, Codex/Claude temporary roots |
| AC-03 | PASS: six/eight retirement tests and checksum-verified published-binary observations |
| AC-04 | PASS: modified/foreign/extra/linked/receipt conflict tests; unattested empty directory refusal |
| AC-05 | PASS: full lifecycle regressions, catalog→parser/application dispatch; supported CLI unchanged |
| AC-06 | Automated routing contract PASS; actual Codex/Claude natural-language selection PENDING |
| AC-07 | PASS: authoritative FlagSets, required/conditional argument parity, mode/authority/payload tests |
| AC-08 | Historical six/eight/POC/partial paths PASS in tests; full clean archive binary-upgrade journey PENDING approved candidate |
| AC-09 | PASS local reconciliation links in #229/#230 Evidence, spec/plan/tasks/matrix/docs; remote Issues unchanged |
| AC-10 | PASS retrospective exact-tag corpus + semantic reader + enforced future release gate; original publication-time omission remains historical |
| AC-11 | D1–D4 addressed locally; D5 manual reproducible protocol prepared, native/Provider observations PENDING |
| AC-12 | PASS authority limits observed; no real Provider/user-installation mutation, publication or closure |
| AC-13 | Automated suites PASS; unavailable/blocked/pending native observations explicitly recorded |
| AC-14 | PASS exact two-skill embedded/manifest/installed/discovery surfaces |

## Review, risks and next action

Three bounded delegated units used inherited Orchestrator capability/effort:
installer security/migration, routing/catalog, historical compatibility/list
output. Parallel files were disjoint, except coordinated canonical skill tests
and ownership API. Orchestrator reran broad tests and artifact observations.
Independent cleanup review found empty-directory preview ownership too loose;
fixed with name-specific historical attestation and regression test. No remaining
blocking local security/routing finding identified. Native semantic behavior is
unverified; deterministic text/dispatch tests cannot claim that acceptance.

Ready for local human diff review. A remote PR has not been created. After
explicit commit/push/PR authorization, validate a clean candidate and execute
checksum-verified full archive upgrade journeys; then request exact isolated
Runtime/Provider acceptance authority. [macOS commands and 18 intent cases](acceptance-230-canonical.md)
provide the reproducible protocol. Human acceptance and Issue closure stay
separate.

## Bounded handoff

```yaml
objective: two canonical Runtime skills with preserved MVP lifecycle
state: local_implementation_validated_native_acceptance_pending
baseline: c7260797aad7865419f555eea94a2eef7ae78285
commits_created: none
changed_files: overview above; git diff plus untracked reconciliation/tests/fixtures
completed_units: [safe_retirement, canonical_catalog, lifecycle_regressions, v0100_corpus, release_gate, documentation]
remaining_units:
  - clean_candidate_release_pipeline_and_full_binary_upgrade_journeys
  - real_codex_and_claude_semantic_selection
  - explicitly_authorized_disposable_github_provider_acceptance
  - human_diff_and_acceptance_review
authority_granted: local_implementation_tests_public_read_only_evidence
authority_pending: [commit, push, PR, merge, release, real_installation, provider_mutation, issue_closure, human_acceptance]
validation_status: local_pass_with_explicit_unavailable_blocked_pending_rows_above
next_action: human_diff_review_then_explicit_clean_candidate_and_native_acceptance_authority
```

## PR preparation authority — 2026-10-08

After reviewing the local delivery summary, the maintainer requested "crie o
PR". This authorizes branch/commit/push/PR creation for this change. Earlier
checkpoint/handoff authority statements above are historical. Merge, release,
real Runtime/Provider effects, Issue closure and human acceptance remain outside
this authorization. Clean committed-candidate checks are reported separately in
the PR; native acceptance remains pending.
