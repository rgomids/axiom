# AXM-32 — Specify Evidence

## Original specify investigation — historical baseline

Documentation-only investigation on 2026-10-18, baseline
`14cb721ac7b8ce09a7f20f2bc2175adf61b7934b` (v0.19.0).
[Specification](spec.md) and
[ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md)
remain Proposed. Evidence here proves source inspection/document integrity only.
No code, tests, adapters, installed skills, operational state or authority changed.

Initial checkout was clean and detached at the baseline; `git ls-remote origin
refs/heads/main` matched. Both open PRs were inspected by title, body and branch
with `gh pr list --repo rgomids/axiom --state open --limit 100 --json
number,title,headRefName,body`: #317 and #327 were unrelated to AXM-32.
Search results containing #327 were not assumed to identify an AXM-32 execution.
`git branch -a` and `git ls-remote --heads origin '*axm-32*' '*AXM-32*'`
showed no AXM-32 branch. Created `docs/axm-32-output-separation-specification`.
Recheck before opening a single draft PR; no empty PR/commit is authorized.

`gh repo view rgomids/axiom --json visibility` confirmed PUBLIC. Only public
repository sources and sanitized user-supplied scope inform these artifacts;
no private Linear/Notion contents or raw personal test data were imported.
The supplied Linear/gate facts were not independently revalidated. No canonical
Execution was observable, reconstructed or advanced; other-host concurrency
cannot be proven absent from a provider PR search.

## Confirmed source observations

- `gh issue view 232 --repo rgomids/axiom --json title,state,url,body` confirms
  closed/accepted history, human decision dated 2026-10-09 and stable v0.13.0.
  `gh pr view 291 --repo rgomids/axiom --json state,mergeCommit,mergedAt`
  confirms merge `2d7a7cdf24feb4312321e44755657d4e0b11b246`,
  2026-10-09T22:51:14Z. Historical Evidence is unchanged.
- `gh pr view 325 --repo rgomids/axiom --json state,mergedAt,mergeCommit,title,body,url`
  confirms MERGED, 2026-10-18T01:57:12Z,
  `5b6bd0b354ec38794be33fd8847333bf5132049f`.
  The integrated files, not the PR narrative alone, confirm exact relay and
  separate Auxiliary explanation. Existing Specification/Plan acceptance records
  remain preserved; AXM-31's synthetic evidence is not native-host proof.
- `gh issue view 305 --repo rgomids/axiom --json title,state,url,body` confirms
  the separate open Project Runtime/Profile policy-editing gap. This proposal
  neither implements nor masks it and does not change dependencies.
- Source inspection of `internal/cli/{cli,completion,presentation,readiness}.go`
  confirms leading output flags, default human mode, exact Markdown over canonical
  JSON, fallback dumps, separate readiness payload and existing exit mapping.
- `cmd/lingo/project_lifecycle.go` returns successful structural validation when
  `report.Structure` is valid and includes the readiness report.
  `cmd/lingo/bootstrap_readiness_test.go` contains an existing synthetic case with
  successful validation, effective partial, Work Item ready and Execution blocked
  by `runtime_policy_unavailable`. Read as source, **not executed here**. This does
  not prove the reported personal Project had those exact facts.
- Four `internal/codexruntime/skills/*/SKILL.md` instruction sources and their
  routing/content/history tests were inspected. Shared Runtime integration tests
  cover both Codex and Claude; no separate `internal/clauderuntime` package exists
  in this checkout. Future native-host behavior remains unverified.
- Consulted constitution, contribution/security/architecture/quality/documentation
  and automation policies, contribution/PR templates, commands/tracking reference,
  Specification 004/#232 Evidence, Specification 007 and AXM-31 amendment/Plan,
  ADR-0003/0006/0018/0022/0025. Exact conflict mapping is in the Specification.

## Original documentary checks — historical baseline

No future behavior acceptance is implied by these checks. Link validation is
scoped to new artifacts and added
index links; public issue/PR references are verified with the read-only GitHub
queries above. No external website crawling or speculative capability lookup is
required. One-off link checking runs outside the tracked tree.

| Check executed | Result and limit |
|---|---|
| `./scripts/validate-repository.sh .` | PASS: repository bootstrap, maintainer harness structure, Claude adapters, automation inventory, label policy, ADR structure and sensitive-file checks; native Runtime behavioral scenarios explicitly skipped by the validator |
| `python3 scripts/check-adr-governance.py .` | PASS: 31 ADRs, Proposed status and index reconciled; structure does not establish semantic acceptance |
| Scoped one-off Python link check | PASS: 44 local links including heading anchors and added index entries; target existence/heading normalization checked without a new versioned script |
| `git diff --check`, `git diff --cached --check`, `./scripts/check-sensitive-files.sh --staged .` | PASS: whitespace and staged sensitive-file scan; staged paths/content reviewed before commit |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN, no installation attempted; repository sensitive-file scanner remains applicable |

Review found only five intended Markdown paths: the Specification, this Evidence,
ADR proposal and their two indexes. No accepted specification/ADR/history,
commands reference, executable, tests, adapters, automation registry, dependencies
or installed skill bytes are modified. Any later commit/PR reference is carried
by Git/PR metadata; it is not fabricated inside its own commit.

## Review and limitations

Self-review checks source hierarchy, historical acceptance, absence of an accepted
ADR conflict, status/readiness separation, resolved D1–D3 choices, single-PR scope,
semantic matrix, secret/injection controls and public-data classification.
This is not independent review or human acceptance. D1–D3 are resolved choices;
R1 producer/consumer coverage remains unverified before Plan. Exact revised
Specification/ADR acceptance remains pending; no schema gap is claimed resolved.

**NOT_RUN:** AC1–AC9 implementation, terminal/golden/channel tests, JSON consumer
regressions, security canary tests, installed-skill upgrade/integration journeys,
native Codex/Claude conversations and AXM-12 E2E. They require future approved
contracts/implementation and, for real Runtime effects, separate authority.
No additional Go suite is needed to validate a specification-only change.

## D1–D3 specify refinement — 2026-10-18

The [GitHub human decision](https://github.com/rgomids/axiom/pull/328#issuecomment-6245936652)
(created 2026-10-18T05:06:23Z) and matching
[Linear human decision](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-43e5ffd1)
(created 2026-10-18T05:06:51Z) were read directly, together with AXM-32 and
[review #5561505065](https://github.com/rgomids/axiom/pull/328#pullrequestreview-5561505065).
The user explicitly authorized publication of these D1–D3 choices in this public
repository. Only their bounded decision content and links are incorporated;
unrelated Linear content and raw conversations are not imported. Original
investigation/validation observations above remain dated historical evidence.

Before edits, `gh pr view 328` confirmed OPEN/Draft, branch
`docs/axm-32-output-separation-specification`, head
`e59480d432403104fccdd7ba61f8f9b46752d3c2`.
`git fetch origin docs/axm-32-output-separation-specification` and `git rev-parse`
confirmed local HEAD and the fetched remote branch matched that reviewed head;
the checkout was clean, with no intervening remote edits to reconcile.
`gh pr list --state open` confirmed #328 as the sole open AXM-32 PR.
`gh repo view --json visibility` reconfirmed PUBLIC. The resulting revision is
identified by Git history and PR head metadata, avoiding a self-referential SHA.

Linear currently reports In Progress. This provider operational status neither
proves work start/completion nor projects a canonical Execution stage. No local
Execution was reconstructed or advanced; no provider lifecycle metadata changed.

Refinement reconciles native human CLI defaults in TTY/pipes and explicit JSON
(D1), coordinated replacement of default Markdown/literal relay/Auxiliary within
AXM-32 across binary and all four installed Codex/Claude skills (D2), and current
typed wire-contract preservation without speculative schema evolution (D3).
The fact-by-fact R1 producer/consumer inventory is still outstanding before Plan;
proven material gaps require explicit proposal review/acceptance with affected
consumers/tests before code changes. No JSON coverage or implementation is claimed.

Only the Specification, Proposed ADR, this Evidence and their two stale index
summaries are revised. PR #328's stale decision/lifecycle description is reconciled
in place. Accepted documents, historical Evidence files, executable code, tests,
scripts and installed skills are untouched. The ADR remains Proposed; no formal
supersession or contract acceptance is inferred. All AC1–AC9 remain NOT_RUN.

No subagents were spawned: the edits are coupled and no model strictly inferior
to the Orchestrator was verifiable. Deterministic validation and self-review are
performed by the Orchestrator; no independent review is claimed.

### Refinement validation

Results below concern documentary integrity only; they cannot establish behavioral
acceptance, native-host fidelity, final human approval or merge readiness.

| Check executed for refinement | Result and limit |
|---|---|
| `./scripts/validate-repository.sh .` | PASS; includes harness/adapter structure, automation registry, ADR/label/security checker regressions and worktree scan. Native Runtime behavior explicitly skipped/UNVERIFIED |
| `python3 scripts/check-adr-governance.py .` | PASS, 31 ADRs; structural status/index reconciliation only |
| Scoped one-off Python link/anchor check in `/tmp` | PASS, 43 local links/anchors across the three artifacts and revised index references; no tracked script added |
| `git diff --check`, `git diff --cached --check` | PASS; staged paths/content reviewed for only five authorized Markdown files |
| `./scripts/check-sensitive-files.sh --staged .` | PASS; no secrets/private unrelated content identified in documentary self-review |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN, no dependency installed |

Self-review found no blocking documentary finding: resolved choices, flags/pipes,
no extra formatting LLM call, authority/status, migration/rollback, historical
acceptance and scope remain coherent. These checks do not claim JSON R1 coverage,
AC1–AC9 execution, native smoke, implementation completion or independent review.
Explicit human re-review/acceptance of the resulting Specification/ADR revision
is still required; the same PR remains Draft.
