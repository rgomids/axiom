# AXM-37 — Specify Evidence and handoff

## Authority and baseline

Snapshot: **2026-10-20**. Scope: `specify` only, Specification 010 rev.1 Proposed.
Human instruction authorizes these documentary artifacts, validation, commit,
push and one PR. No Plan, Tasks, production change, Runtime invocation, local
workflow mutation, acceptance, merge, release or product status transition.
This is sanitized original technical synthesis authorized for publication;
no raw private Provider response or chat history is imported.

Baseline local HEAD and remote main:
`0f1073ac6a3af615d48b651340d8fe895827080a`.
Initial `git status --short --branch`: clean detached HEAD. `git fetch origin main`
and `git diff --stat HEAD origin/main`: no baseline difference.

## Bounded investigation and delegation

Skills applied: `axiom-sdd`, `axiom-orchestrate`, `linear`.
Three bounded fronts, executed directly: documentation governance/inventory;
installation/release and identity; security/compatibility and Evidence criteria.
No subagents: runtime exposes model overrides, but this session has no proven
strict capability hierarchy below its Orchestrator. No profile/effort inheritance
or isolation claim is manufactured. Deterministic sources were integrated and
reviewed by the main session. No production code or Runtime skills were executed.

## Verified discovery

Read AXM-37 complete issue and comments through connected Linear tools before
investigating product sources. `get_issue(includeRelations=true)` returned `Todo`,
no `blockedBy`, blocks AXM-39/AXM-12, related AXM-30/AXM-33. `list_comments(limit=250)`
returned two comments, no next page: gate
`1a92315b-9206-4adb-8786-25a0f006bbff` and specify prompt
`8cbb1ea9-6585-4444-b1de-2f93a17d79be`. Neither demonstrates Execution.
No separate execution-attempt comment was identified. **Canonical Execution
remains unknown**; no local ledger was reconstructed, created or advanced.

Read-only remote/local searches:

- `git ls-remote --heads origin`: main and two unrelated branches; no AXM-37 branch.
- `git branch --all --list '*axm-37*'`: no existing local branch before creation.
- `gh pr list --repo rgomids/axiom --state all --search 'AXM-37'`:
  historical broad matches were unrelated on title/head/body inspection;
  no directly associated AXM-37 PR identified.
- `gh pr list --repo rgomids/axiom --state open --json number,title,headRefName,url --limit 100`:
  only unrelated PR #317 at inspection.
- `gh issue list --repo rgomids/axiom --state all --search '"AXM-37"' --json number,title,state,url --limit 100`:
  empty. No technical GitHub Issue invented to mirror Linear.
- Bounded search of `docs/specifications` and `.agents`: no pre-existing AXM-37
  Specification. Index ended at Specification 009.

Created branch `docs/axm-37-offline-operational-documentation` from verified main.
These observations support avoiding duplication; they do not prove absence of
an undiscovered canonical Execution. Recheck remote state before PR creation.

## Source findings and conflicts

Sources linked in [Specification baseline](spec.md#baseline-e-compatibilidade-preservada)
were read locally at the verified remote-main revision, including requested
governance, command reference, tracking, AGENTS, SDD and index. Constitution,
contribution/documentation/security policies, repository-security controls,
installer/archive construction, CLI help/command tree and domain skill excerpts
provided the bounded supporting context.

- `scripts/build-release-archives.sh` copies binary, LICENSE, installer,
  skills and metadata, then enumerates those hashes. No operational-doc package
  or manpage construction is present in that builder. This observation is not a
  clean-install experiment or proof about every published artifact.
- `docs/commands.md` archive prose says three skills; current builder enumerates
  `internal/codexruntime/skills/*`, which contains four SKILL.md files. Selection
  must follow verified effective inventory, not this stale prose. No correction
  outside specify scope; Plan must reconcile relevant release references.
- `docs/installation.md` and ADR-0021 Status note preserve Windows withdrawal;
  new proposal records conditional Windows behavior without restoring support.
- Current installation guide refuses downgrades and promises no automatic
  rollback. Proposed rollback criteria require a coherent, authorized recovery
  path without inventing one or silently weakening existing contracts.
- ADR-0005 bounds same-UID/concurrency/power-loss guarantees. New security and
  atomicity obligations retain its logical publication and uncertainty semantics.
- Documentation governance keeps editorial material in Notion and technical
  references in Docs-as-Code. Proposed operational package derives technical
  content only; AXM-30 does not authorize Notion/Wiki changes.

No accepted ADR edited or superseded. Q1–Q6 retain choices and risks for later
Clarification/Plan; no architecture or new capability is accepted here.

## Artifact scope and validation

Changed paths:

- `docs/specifications/010-installed-operational-documentation/spec.md`;
- `docs/specifications/010-installed-operational-documentation/evidence.md`;
- `docs/specifications/README.md`.

Checks actually run on the documentary changes:

| Check | Result and limit |
| --- | --- |
| `./scripts/validate-repository.sh .` | PASS; harness, automation registry, ADR governance, sensitive-file checks and repository diff validation. Native behavioral scenarios explicitly SKIP / UNVERIFIED |
| `git diff --check` | PASS; full proposed documentary diff reviewed by main session |
| Ephemeral Python local-link/requirement check | PASS: 22 local file links resolve; OD-01–13, AC-01–08, Q1–Q6 and Proposed header present. File existence/marker coverage only; not semantic acceptance or external-link validation |
| `gitleaks detect --source . --no-git --redact --no-banner` | PASS; no leaks found in worktree scan |

Staged-file check and exact commit/PR references are recorded in the delivery PR
after execution; they are not claimed in advance here.
No implementation tests, install experiments, generated manpages, native
Codex/Claude session, offline isolation trial or release validation ran in this
stage. AC-01–AC-08 remain **NOT_RUN**. Documentation checks cannot substitute for
those obligations. Existing installation/CLI/skills/ADRs/history remain unchanged.

## Recoverable handoff

- Objective: propose installation-bound offline operational documentation.
- State: specify artifact prepared; proposal requires independent human review.
- Completed units: bounded source investigation, behavior/invariants, inventory
  criteria, failure cases, Q1–Q6 options and future Evidence matrix.
- Remaining work: review exact Specification revision, resolve contract-changing
  Clarifications under a separate gate, authorize Plan independently. Plan must
  analyze packaging/discovery/publication mechanisms and native proof feasibility.
- Authority pending: Specification acceptance, every successor stage, product
  implementation, native Runtime effects, merge, release and AXM state changes.
- Canonical Execution: unknown; recover legitimate local truth before any gate
  mutation. Do not infer it from this branch, commit, PR, CI or Linear status.
- AXM-39/AXM-12 remain dependent on their actual readiness/acceptance gates;
  this proposal alone unblocks neither.
