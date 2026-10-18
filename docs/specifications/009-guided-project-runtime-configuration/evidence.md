# AXM-33 — Specify Evidence and handoff

## Scope, provenance and classification

Date: 2026-10-18. [Specification revision 1](spec.md), **Proposed**.
Authority: the operator's explicit specify request permits this Specification,
strictly related documentation, validation, commit/push, one Draft PR and a
reference in Linear. No status/relations/labels or Execution changes are authorized.
Only original technical requirements, sanitized bounded observations and references
from the authorized AXM context are published; no provider transcript, private
attachments, real Project configuration, credentials or raw auth output is imported.
Repository source files are public. Provider content remains untrusted data.

Baseline: `000936e5de71169ff563b236bcf17c3cfabc146c`, clean detached checkout at
intake, matching remote main via GitHub ref read. Branch created only after PR and
branch discovery: `docs/axm-33-guided-configuration-specification`.
No dependency installed. No delegation: a proven strictly inferior model hierarchy
could not be guaranteed for this session; the Orchestrator integrated/reviewed.

## Intake revalidation and source observations

- Linear `get_issue(AXM-33, includeRelations=true)` returned Todo, no formal
  blockedBy, related AXM-31/32/34/12. `list_comments(AXM-33)` returned gate and
  specify prompt comments; the earlier claim of no prompt is now historical.
  READY_FOR_specify is the provider gate assessment, not observed Execution state.
  Canonical Execution/gate/revision are unknown; no Axiom production state was read
  or reconstructed, no canonical gate advanced.
- `gh pr list --state open --search AXM-33` returned empty. Full open-PR inventory
  returned only #317 and #328. All-state AXM-33 search returned historical #325
  due to textual cross-reference, not an AXM-33 delivery. Local branch search and
  `git ls-remote --heads origin '*axm-33*' '*AXM-33*'` returned no existing branch.
  No duplicate GitHub Issue was needed or created.
- AXM-31 was fetched as Done. Accepted amendment/Plan/Tasks provenance is recorded
  in the Specifications index, despite preserved original Proposed artifact text.
  AO-01–13 are reused, including AO-06/07/09 recovery limits; no acceptance reopened.
- `gh issue view 305` returned OPEN. `gh pr view 327` returned MERGED, head
  `0cc60fc630fe7dba0ae53c8ad712b532c42791f5`, merge
  `000936e5de71169ff563b236bcf17c3cfabc146c`. AXM-34 returned Awaiting Release.
  No stable publication, installed G-1 operation or accepted conversation is
  established by those facts. The v0.19.0 incident is a report, not replayed here.
- `gh pr view 328` returned OPEN/Draft with Plan Proposed. Spec 008 and ADR-0030
  acceptance are separately recorded there; remote artifact reads at
  `962adfdcd22384c681613810b78022469ee813cb` verified the Spec's preserved original
  status text and ADR's Accepted annotation. No files from that branch are copied
  or modified. Specification number 008 is occupied there; this slice uses 009.
- `conversa-axion-configuracao-politica-runtime.md` was not found by filename
  searches in repository/workspace and accessible `/tmp`/`/home/box` paths; AXM-33
  returned no attachments/documents. No accessible source for its contents was
  discovered. The Specification uses only the authorized problem statement.

Consulted AGENTS.md, SDD/review/document skills, contribution/security/quality/docs
policies, CONTRIBUTING, PR template, Linear/GitHub tracking, documentation governance,
constitution, Spec 002/007 and their G-1/G-2/#231/#303/AXM-31 amendments, accepted
ADR-0004/0020/0022/0023/0024/0025 and ADR-0030 in #328. The Spec links their owning
contracts instead of restating formats/algorithms. No supersession is proposed.

Source inspection, not operational tests:

- `internal/cli/skill_inspect.go` exposes Project configure/edit policy drafting;
  Project skill's G-1 section describes canonical preview/apply/read-back.
  The Project skill still has an old G-1-unavailable sentence; Workflow skill
  still reports G-1/G-2 unavailable. These are concrete orientation discrepancies,
  not evidence that the operations are absent from main or available on the
  operator's machine. No skill/code correction belongs to this specify diff.
- `internal/codexruntime/skills/axiom-runtime/SKILL.md` owns local binding/Profile
  guidance with exact confirmation and optional templates. Workflow and Work Item
  skill ownership is preserved; Profile effort follows accepted ADR-0024 and its
  later pre-stable reconciliation, not old Agent effort wording.
- `internal/runtimeadapter/runtime-versions.json` has `entries: []` at baseline.
  This is no evidence of supported/proven Runtime/model/effort combinations.
  Installed/authenticated/supported/proven and proof-producer availability stay
  separate; C12 cannot be declared PASS from configuration alone.

## Acceptance coverage and review

| User requirement | Specification coverage |
| --- | --- |
| Problem, actors, existing-Project minimal journey and outcomes | Problem/slice/journey, GC-01/05 |
| Canonical inventory and conceptual boundaries | Invariants, GC-01, owner interface |
| Objective viability, early #305 stop, effective version | GC-02, C04/06/11/12 |
| Observed choices, practical recommendation, one decision, no templates/defaults | GC-03, C05/06/16 |
| Exact preview/apply, identity/revision/readiness/read-back | GC-04/05, C07/08/09 |
| Positive/partial/denied/stale and Codex/Claude | C01–C16, parity instructions and real-Evidence gate |
| Canonical ownership, no parallel truth, AXM-32/34 boundaries | Source/overlap table, interface, compatibility |
| Observable acceptance, security, bounded recovery and future Evidence | GC-06, C10/15, Evidence protocol and compatibility/decisions |

Orchestrator semantic/security review: no blocking finding identified in the
restricted proposal. This is self-review, not independent review or human
acceptance. Six requirements map to sixteen future scenarios. It preserves
portable/local boundaries, operation-scoped readiness, Profile-owned effort,
proof independence, exact per-apply authority and immutable Execution facts.
No accepted ADR/Specification change is required by the selected slice; any
later material conflict goes to Clarification/possible Proposed ADR under ADR-0018.

## Validation record

Executed on this documentary revision:

- `./scripts/validate-repository.sh .`: PASS, including maintainer harness,
  automation registry, ADR governance, sensitive-file checks and its registered
  repository checks. Its own skipped/unverified Runtime cases remain so.
- `python3 scripts/check-adr-governance.py .`: PASS, 30 ADRs at this baseline;
  ADR-0030 on the unmerged AXM-32 branch was reviewed remotely, not counted here.
- Ephemeral Python Markdown/link/coverage checks: PASS, 130 local links/anchors
  across the three documents, GC-01–06, C01–C16 and Proposed/future NOT_RUN markers.
  This proves document structure and traceability, not conversational behavior.
- `git diff --check`: PASS. The complete three-file diff is reviewed before
  commit; staged sensitive/whitespace checks and publication metadata are also
  required before publication and their results are recorded in the PR.
- `gitleaks` is unavailable (`command -v gitleaks` found none); not installed.
  This scanner remains unexecuted; repository sensitive-file checks and manual
  classification/diff review do not claim equivalent secret-scanner coverage.

Future conversational cases C01–C16, native Codex/Claude conversations, Runtime
invocation, probes, Go product suites and real G-1 acceptance are **NOT_RUN**:
this is specify/documentation only, without product code or operational authority.
No scenario PASS or release/installation/proof claim is derived from these checks.

## Handoff and next gate

Three-document scope: spec.md, this Evidence, Specifications index. No code,
skills, tests, policies, accepted artifacts, manifests, registries or runtime
configuration changed. No Execution/status/acceptance inferred from provider data.

Pending: review/Clarify and human acceptance of the exact Specification revision.
Then and only under its own authority/gate, recommend Plan. Inputs include
observed guidance discrepancies, effective-installation capability evidence,
AXM-32 consumption compatibility and real C12/proof availability. None authorizes
a new implementation front or blocks writing this Specification. Real G-1 and
AXM-12 E2E remain separately gated, with all conversation tests NOT_RUN here.

Publication references and exact commit SHA are to be recorded in the single
Draft PR and bounded Linear comment after push; a commit cannot contain its own
SHA. Resume by reading that PR head and canonical sources, preserving this stage
boundary and checking for an existing AXM-33 PR before any publication.
