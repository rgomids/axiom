# AXM-39 — Implementation Evidence, attempts 01–04

Current checkpoint: **attempt-04 PARTIAL; T07 BLOCKED_HOST_BASELINE;
native Codex/Claude NOT_RUN; T08 PARTIAL**. Manual host update observed, but
installed skills precede this PR and docs inspection refuses `source_unqualified`.
Both Axiom authentication preflights also refuse environment overrides.
See [attempt-04 read-back](#attempt-04--manual-host-update-preflight).

## Historical attempt-01

2026-10-23. Baseline: [Specification r1](spec.md), [Plan r1](plan.md),
[Tasks r1](tasks.md), unchanged [C01–C16/native protocol](examples.md).
[Safe implementation authority](https://linear.app/rgomids/issue/AXM-39#comment-a8ab2503),
[documentary review](https://github.com/rgomids/axiom/pull/343#pullrequestreview-5628290008).
Same [Draft PR #343](https://github.com/rgomids/axiom/pull/343), branch
`docs/axm-39-conversational-presentation`; input local/remote/PR HEAD
`f69d950867141aaaf9f0f6ddfb66726f259bd1b8`, initially clean. Final commit, actual
CI run/status and publication read-back belong to the same-attempt Implement
completion event on [AXM-39](https://linear.app/rgomids/issue/AXM-39) and PR checks.
This avoids claiming a self-referential final commit SHA inside its own bytes.

**Outcome: PARTIAL. T07 BLOCKED_AUTHORIZATION; both native conversations NOT_RUN.**
Safe local implementation and synthetic validation do not complete AC5 or product
acceptance. DESIGN=NOT_APPLICABLE; text/Unicode/fallback requirements still apply.
No new PR/AXM/ADR, Kanban/relation changes, merge, release, Done or human acceptance.

## E01 — Sources, ownership and invariants

- Authenticated current AXM-39 card/events, exact human approval and both PR
  reviews read. All 27 comments returned with `hasNextPage: false`; two Implement
  prompt/dispatch records are delivery observations, not execution Evidence.
  No existing Implement completion event was present at entry.
- AXM-36/37/38 current dependency cards read: Done; AXM-12 In Progress.
  Source includes operational admission, installed documentation inspection and
  rendered Workflow/text oracles. Done is not installed/loaded/native Evidence.
- Full 012 Tasks/Spec/Plan and examples read; ADR-0030/0032, relevant ADR-0024
  effort ownership and RA-1–6 boundaries inspected. Canonical read-only owners:
  `internal/cli/cli.go`, `internal/projectapp/{list,readiness,workflow_authoring}.go`,
  shared integration, bootstrap, prior history and upgrade/restoration tests.
- Worktree inventory, process executable/CWD inventory and open-file checks found
  no observed writer on scoped source files. Existing host processes are not
  proof of a second Task execution. Rechecked unchanged HEAD and original skill
  bytes immediately before shared writes; no other local changes appeared.
  This is bounded observation, not a global cross-process lock or ownership lease.
- Direct sequential execution; lower subagent capability hierarchy unproven.
  T03 Project edits preceded T04; T05 captured only final skill bytes. No subagents.
- Shared field-note tool was advertised but returned tool-not-found. Canonical
  issue/events/source remained available; no fabricated notes or callback.
- Only approved public intent/references imported. Fixtures contain synthetic
  IDs, model labels, observations and text; no private provider exports, global
  Runtime-root inventory, auth output, sessions, credentials or raw transcript.

## E02–E04 — Regression, guidance and case ledger

Regression-first `go test ./internal/codexruntime -run
TestOperatorConversationalPresentation012Guidance -count=1` failed against
unchanged guidance on the absent 012 rules. After guidance changes: PASS.
Initial oracle scaffolding also exposed an absent synthetic provenance value and
incorrect literal control escaping; fixtures were corrected, with no producer or
existing checker relaxation. The semantic oracles were developed independently of those instruction checks;
positive candidates need not fail against old guidance because no model is run.

[CLI oracle](../../../internal/cli/output_separation_test.go) extends only test
code, reusing `semanticObservation`, `semanticClaims` and `checkCandidate`.
Existing completion/effects/reference checks remain unchanged. Ordinary labels
have no forced all-ID/reference obligation. Project lists, readiness and diagnostics
use existing typed DTOs; context flags/obligations are test-only, not wire fields.
[Stage oracle](../../../internal/codexruntime/workflow_presentation_test.go) uses
existing typed Workflow fixtures and `inspectPresentation` with independent
hand-authored summary/detail specimens. [Producer check](../../../internal/cli/workflow_presentation_fixture_test.go)
compares the new synthetic Project list payload to actual JSON emission and checks
that ProjectView supplies no name; the four existing Workflow envelope equality
checks are reused. No production mapper, schema or formatter introduced.

[012 specimens](../../../internal/cli/testdata/presentation/012/) are offline
candidate answers, never generated/observed host conversations. Cases with absent
identity, duplicate/conflicting records, partial inventory, missing gates or
inaccessible/truncated observations are explicitly adversarial sources, not
advertised valid producer output. Scenario obligations describe the supplied
observation, not a new product diagnostic producer. All per-case omission mutants
and forbidden-claim mutants are rejected; changed typed Project/readiness/Stage/
diagnostic observations invalidate old answers. Exact full ID, command and digest
omissions/replacements fail. Existing AXM-38 input deltas/order/edge/full fallback
oracles retain authority. The checker recognizes bounded fact phrases, not arbitrary
natural-language entailment. Entry-labelled replay is the same offline oracle,
not real natural/explicit host routing or proof of fluency.

| Case / positive specimen | Required facts / negative mutants | Synthetic result |
| --- | --- | --- |
| C01 | Órbita (orbita), no routine UUID; omit label/invent name | PASS |
| C02-name, C02-slug | One supplied field; invented second field; same-text name/slug edge | PASS |
| C03 | No-name fallback and full ID/reason; missing/wrong identity | PASS |
| C04, C04-name | Both long-prefix-colliding IDs; exact target; no first match; duplicate/conflicting same-ID edges | PASS |
| C05 | Requested ID, copyable selector, full digest and preview revision; changed/omitted bytes | PASS |
| C06 | Partial inventory, absent ID, control escaping and inert hostile label; generated identity/injected command/bidi edge | PASS |
| C07 intake/implementation summary + detail | Purpose/I/O/owner/Role, gates, Profile certainty, criteria; advanced validator/config/instructions, requiredness, limits, effects and failure policy; missing vs empty gates | PASS |
| C08 intake/implementation risk | Unavailable validator and authority/timeout/attempt constraint initially visible; omitted risk/invented execution | PASS |
| C09 | Allowed codex/claude, declared policy-default/codex-test; unknown selection/resolution/model/effort; invented host preference/selection | PASS |
| C10, C10-selection | Resolved Plan/Profile and separate login-only observation; exact choice without resolution; no attempt/billing/acceptance inference | PASS |
| C11 | Success, valid structure, execution blocked, exact reason/scope, confirmed no-dispatch, next owner/authority; lost blocker | PASS |
| C12 | Prior attempt, both blockers, timeout, invalid result despite exit 0, confirmed/unknown effects; reconcile before retry; false no-dispatch/automatic retry | PASS |
| C13 | Stale/conflicting revision and inaccessible/truncated detail; missing uncertainty/fresh read, fabricated current resolution | PASS |
| C14 | Unknown version, otherwise admissible source/authority; probe/registry/certification or success invention | PASS |
| C15 | Full reference/selection, ordered Stages, independent roots and exact join/gates; text fallback; omitted/reversed edge/fake rendering | PASS |
| C16 | Unicode/full ID, blocker/gate and plain text; entry-independent offline replay; omitted material fact | PASS |

T03 adds Project name/slug matrix, exact-ID metadata correlation/fallback,
collision/partial inventory and inert identity guidance. T04 adds progressive
Stage/Agent disclosure and source-attributed Runtime/readiness rules. The existing
AXM-38 graph/reference/fallback obligations are retained; advanced nonmaterial
fields are recoverable on request, risks promoted immediately. Work Item reviewed
unchanged: Plan/Execution separation, Role not selecting model, owner handoff,
blocked execution and confirmed/uncertain effects already covered. Four-skill
review does not imply all four files require edits.

## E05 — Complete history, both roots and restoration

Original full bytes preserved in
[pre-conversational-presentation](../../../internal/codexruntime/testdata/pre-conversational-presentation/),
following the existing pre-rendered Workflow fixture precedent. This supporting
fixture directory is necessary to exercise exact replaced-revision ownership and
restoration; it adds no installer behavior. All older fixtures/receipts unchanged.

| Skill | Replaced SHA-256 | Current SHA-256 / review |
| --- | --- | --- |
| axiom-project | a249ea87133eeefd53f7e6bccf8b5d083c2bf87802f8141ec85183d52d9a892d | 775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d; P1–P2/R3 |
| axiom-runtime | 90a725ed0ac96aa3378b3c9860f5357c8ef63b090662a19d362c24c8cc0aad0b | 165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d; R1–R3 |
| axiom-work-item | ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3 | same bytes; reviewed unchanged |
| axiom-workflow | 8b47aa65611d28395eab3695ce4ca90773e541800f72d4189838565268a7f235 | efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76; S1–S2/R1–R3 |

Replaced manifest `508f3f1b1a31a403d5deba227cffef981bc67061812394420948290a78c03216`;
current manifest `d039a843c6289c3f91af73fab96e9e62c3cc1efafe6f08f07e163c925f5cc4b7`.
Complete original set appended to `sharedSkillHistory`; current pin added to
`publishedSharedRevisions`. SkillSetVersion/BinaryCompatibility remain **3**:
only compatible guidance changed; no wire/consumer incompatibility demonstrated.
Frozen Runtime-only history remains untouched. Earlier AXM-38 test now locates
its exact historical digest instead of assuming the newest history entry.

`TestOperatorConversational012UpgradeConflictInterruptionAndRestoration` covers
both temporary Codex/Claude roots: previous full set → exact embedded bytes,
ready/current receipt, repeat unchanged, modified/foreign content refusal and
byte preservation, interruption partial/not ready then convergence, foreign file
preservation and compatible previous pair restoration with Project/Workflow/
Execution/Profile sentinel bytes intact. Initial test found that a Claude backup
receipt generated for the backup path was not the original receipt; corrected
test backup to preserve original destination-bound receipt bytes before restore.
No production receipt repair, host installation or global-root access performed.
Existing coordinated upgrade/pair-restoration and published-history tests also run.

Rollback: scoped normal revert/review of 012 changes; if published, preserve
appended ownership history. Restore an actually compatible binary/skill pair only
through a separately authorized supported mechanism. Preserve canonical state,
foreign/modified files and historical Evidence; this local test is not a product
rollback command or native-host rollback observation.

## E06 — Actual validation and engineering/security review

- PASS focused CLI: `go test ./internal/cli -run 'Test(OutputSeparation|WorkflowShowSynthetic|Presentation)' -count=1`.
- PASS focused package: `go test ./internal/codexruntime -run 'Test(RenderedWorkflow|OperatorConversational|WorkflowSkill)' -count=1`.
- PASS package gate: `go test ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./cmd/lingo`; codexruntime 16.083s, runtimebootstrap 0.870s, CLI 0.817s, cmd/lingo 402.381s. A subsequent Stage-reference test strengthening passed the focused package check again.
- PASS `./scripts/validate-repository.sh .`: structure/bootstrap, automation, ADR
  governance, sensitive-file checks. Native maintainer scenarios explicitly
  SKIP/UNVERIFIED; not AC5 conversations. No optional global skill validator invoked.
- PASS `git diff --check`; 195 local Markdown link paths resolve; four original
  skill fixtures match input commit bytes; Spec/examples/Plan/earlier Evidence
  unchanged. PASS `./scripts/check-sensitive-files.sh --staged .`, scoped redacted
  Gitleaks for new specimens/012 documents and `gitleaks protect --staged --source . --redact --no-banner`.
  Final exact-head CI links/status go in completion receipt; pending/unrun CI is
  not PASS. No separate native-acceptance runner used as AC5 evidence.

Review: only three embedded guidance files, append-only complete manifest history,
current manifest pin, test code/new synthetic fixtures, 012 Task/Evidence and 012
index entry. No domain/CLI producer/JSON/text/flag/Registry/Execution/admission,
bootstrap or preview digest semantics changed. Accepted ADR-0030/0032 and effort
ownership preserved. No blocking finding identified in scoped self-review;
independent review/human acceptance remain pending. No arbitrary renderer,
subprocess, network fetch or hostile label execution. Failed intermediate tests
above were corrected and revalidated, not relabelled native successes.

## E07 — Prepared native protocol; BLOCKED_AUTHORIZATION

| Record | Authorization / host / installed / loaded | Observed conversations | Result |
| --- | --- | --- | --- |
| E07-Codex | NOT_PROVIDED / NOT_OBSERVED / NOT_OBSERVED / UNKNOWN | none | NOT_RUN |
| E07-Claude | NOT_PROVIDED / NOT_OBSERVED / NOT_OBSERVED / UNKNOWN | none | NOT_RUN |

Preparation complete; [existing protocol](examples.md#two-native-conversations--future-protocol-not_run)
and sanitized 012 specimens ready. Operator checklist for **each** host:

- [ ] Record specific authority reference: target host/surface/build, permitted
  installed reads and conversation/inference/cost envelope. Installation/update,
  credential access and external effects each require their own explicit grant.
- [ ] Outside maintainer checkout, record actual binary provenance and supported
  `axiom --json runtime codex status` or `axiom --json runtime claude status` for
  the authorized target; `axiom --json docs inspect`, returned verified paths and
  `axiom --json skill inspect <domain-skill>`. Do not invoke bootstrap for inspection.
- [ ] Record compatibility and full skill manifest/content identity. Mixed,
  modified, foreign, partial or absent pair holds the dependent conversation;
  no receipt edits or automatic installation. On-disk presence does not prove
  loaded revision. Record actual loaded revision only when observed.
- [ ] Natural prompt: “Liste Projects; mostre e valide este Project. Informe seu
  ID completo.” Resolve target safely. Replay C01–C06 only as labelled synthetic
  input if actual catalog lacks those cases; never manufacture installed state.
- [ ] “Explique intake e implementation e seus Agents. Agora mostre validators,
  limites, tentativas, timeout, concorrência, efeitos e política de falha.” Use
  one exact Workflow revision; check summaries, advanced detail and C07/C08/C15.
- [ ] “Codex ou Claude? O que está permitido, declarado, selecionado, resolvido
  e observado? Está pronto?” Check C09–C14; replay synthetic blockers explicitly.
  No dispatch simply to obtain an example. Observe graph or honest text fallback.
- [ ] Record date/build, authorization, installed/loaded revisions, canonical
  inputs/source revision, case/AC, sanitized visible excerpts, PASS/FAIL/PARTIAL/
  NOT_RUN, gaps and next action. Retain failures; no raw transcript/auth/path export.

Nothing above was executed on a real installed host. No sessions, installation,
authentication inspection, inference, expenditure or operational external effects.
AXM-12 still owns E2E/inference journey acceptance; these conversations do not
replace it. No operational authority discovered in approval/events supplies T07.

## E08 — AC/Task reconciliation and handoff

| AC | Cases | Tasks / Evidence | Actual status |
| --- | --- | --- | --- |
| AC1 | C01–C06, C16 | T01–T03,T05–T08; E01–E03,E05–E08 | synthetic PASS; native NOT_RUN; PARTIAL |
| AC2 | C07,C08,C15,C16 | T02,T04–T08; E02,E04–E08 | synthetic PASS; native NOT_RUN; PARTIAL |
| AC3 | C09,C10,C13,C14 | T02,T04,T06–T08; E02,E04,E06–E08 | synthetic PASS; native NOT_RUN; PARTIAL |
| AC4 | C11–C13 | T02–T04,T06–T08; E02–E04,E06–E08 | synthetic PASS; native NOT_RUN; PARTIAL |
| AC5 | C01–C16 | T01–T08; E01–E08 | regressions PASS; Codex/Claude NOT_RUN; PARTIAL |
| AC6 | C01–C16 scope/negative review | T01–T06,T08; T07 authority boundary; E01,E05,E06,E08 | scoped diff/authority review PASS; no product acceptance |

Task checkpoints: T01–T06 local PASS; T06 local checks and exact-head CI are
separate (see completion receipt); T07 **BLOCKED_AUTHORIZATION**, preparation PASS;
T08 safe crosswalk/receipt handoff prepared, **PARTIAL** because native gap remains.
No claim all Tasks complete. Final remote publication/CI and event read-back are
added to same-AXM receipt after their actual execution, not presumed here.

## technical_decisions[]

All TECHNICAL_LOCAL; executor-ai; scope_impact none; reversible within baseline.

| ID | Choice / alternative | Rationale / GitHub source Evidence |
| --- | --- | --- |
| IMPL-TD-01 | Direct serialized execution, no subagent | Shared Project/manifest writers and unproven lower capability hierarchy; [Tasks ownership](tasks.md#ownership-order-and-shared-evidence-rules) |
| IMPL-TD-02 | Test-only typed claims plus separately authored candidates; no product formatter | Existing [semantic oracle](../../../internal/cli/output_separation_test.go) and [Workflow oracle](../../../internal/codexruntime/workflow_presentation_test.go); no new producer/schema |
| IMPL-TD-03 | Exact-ID list metadata or slug/ID fallback; Work Item unchanged | [ProjectView/list DTO](../../../internal/cli/cli.go), [Project skill](../../../internal/codexruntime/skills/axiom-project/SKILL.md), existing [Work Item handoff](../../../internal/codexruntime/skills/axiom-work-item/SKILL.md) |
| IMPL-TD-04 | Compatibility 3, complete replaced set and original-byte fixtures | Compatible guidance only; [manifest](../../../internal/codexruntime/manifest.go), [both-root tests](../../../internal/codexruntime/rendered_workflow_upgrade_test.go); avoids reflexive bump or lost ownership |
| IMPL-TD-05 | Preserve root-bound receipt in temporary pair backup; exact-digest historical lookup | [Restoration precedent](../../../internal/codexruntime/output_separation_test.go) and [updated tests](../../../internal/codexruntime/rendered_workflow_upgrade_test.go); no installer redesign/receipt repair |

## deferred_findings[], remaining work and next gate

**DF-01 — DEFERRED_NOT_IMPLEMENTED:** predecessor summaries in
[Specifications index](../README.md) remain historically stale; [original Evidence](evidence-specify.md).
Show prominently at pre-merge. No predecessor correction or new requirement.
No additional optional finding promoted to scope.

Remaining work: specific T07 operational authorization, compatible installed/loaded
pair read-back and both native conversations with sanitized Evidence; reconcile
AC1–AC5/T07/T08; independent code review and real final-head CI verification.
No proven authorized continuation callback available. Handoff checkpoint:
**AWAITING_EXECUTION_REPORT**, next_gate **review_partial_implementation** with
**T07_AUTHORIZATION_REQUIRED** gate on native work only. Continue safe review;
no ceremonial human continue request, fabricated callback or merge/readiness claim.


## Attempt-02 — Native preflight and reconciliation

2026-10-23. Resume only T07/T08; T01–T06 preserved. Input local, remote and
Draft PR #343 HEAD verified as `4d2f5e96d586316381d0947abfcd1b4057a45614`;
clean tree, same branch. Linear returned 31 comments, no next page: one attempt-02
prompt and its dispatch, no attempt-02 completion or second observed execution.
Process executable inventory and scoped open-file check found no observed writer
on these two documents; this is bounded observation, not an ownership lease.
[Previous receipt](https://linear.app/rgomids/issue/AXM-39#comment-1e2513b3)
remains historical. Shared field-note API returned tool-not-found; issue/event
and repository sources remained available.

### E07 — Actual installed-host read-back, conversations NOT_RUN

[Specific T07 authorization](https://linear.app/rgomids/issue/AXM-39#comment-b3a7bcea)
was read back: Rafael's authorization communicated by delegated AI Maestro,
declared 2026-10-23 10:15 BRT, without independent human authentication. Current
user instruction authorizes these native conversations using existing legitimate
access; it does not authorize replacing this host's binary/skills or repairing
receipts. E07's installation/update gate and T07's no-host-path-edits limit remain.

Host: actual executor, Darwin arm64; commands executed outside maintainer checkout.
Existing standalone Codex CLI reports `0.163.0`; Claude Code reports `2.1.308`.
Existing local Axiom executable SHA-256:
`d33992f315c3d72d999cce200b46eb62187096d13e6ef76c2f8adf192fbe6699`.
Its supported status/skill inspection provenance reports Axiom `0.20.0`, revision
`dd2ab51f609e`, sourceState `clean`. `axiom --version` is unsupported; provenance
above comes from actual JSON, not inferred from that failed command.

| Read | Actual result | Interpretation |
| --- | --- | --- |
| `axiom --json runtime codex status` | success; skill-set 3, compatibility 3, four equivalent skills, receipt current | compatible with installed old binary only |
| `axiom --json runtime claude status` | success; same four digests and compatibility, receipt current | compatible with installed old binary only |
| `axiom --json docs inspect` | error / invalid_command / operation unknown; next: review supported inputs | required installed documentation read-back unavailable |
| `axiom --json skill inspect` for Project, Workflow, Runtime and Work Item | success, same 0.20.0 provenance | argument registry inspected; does not prove loaded skill revision |
| Targeted four-skill file hashes in each actual Runtime root | both equal status digests below | installed bytes observed, no broader host inventory imported |

Complete content identity (SHA-256); approved source is input PR HEAD, not an
installed or loaded revision:

| Skill | Actual installed, both roots | Approved source |
| --- | --- | --- |
| axiom-project | `95bbbbb7190759a7c12f262f826f048db50d000fd5480c1764753e8a5345bdfe` | `775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d` |
| axiom-workflow | `5148e83308b8927aeb98cc027a6d8cbd0c2c5539b3a335f39cb6b72f2781b36e` | `efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76` |
| axiom-runtime | `596876d7ad5d36a2feac834475dd7487ef2d72e2aa3bbd88eb0ac9be53ec184a` | `165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d` |
| axiom-work-item | `99e4063d95ec80dccea30efe4c3d8abce8f8302c9d373cee34e124fc8f339b2e` | `ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3` |

| Record | Authorized / installed / loaded / authenticated | Conversation | Result |
| --- | --- | --- | --- |
| E07-Codex attempt-02 | conversation authorized / older pair observed / UNKNOWN / NOT_CHECKED | no prompt submitted, no response | NOT_RUN; BLOCKED_HOST_BASELINE |
| E07-Claude attempt-02 | conversation authorized / older pair observed / UNKNOWN / NOT_CHECKED | no prompt submitted, no response | NOT_RUN; BLOCKED_HOST_BASELINE |

No native conversation started: old compatible installation lacks approved skill
content and required docs surface. No canonical Project/Workflow input selected,
no model preference assumed, no claims about native identity, summaries/details,
blockers, Runtime resolution or graph rendering tested. Prompts remain exactly
those prepared in historical E07; there are no executed prompts or visible answers
to sanitize. Maintainer context is not evidence of a loaded product session.
No auth inspection, credential reads, bootstrap, install/update, receipt repair,
inference, dispatch or rollback executed. No defect in approved guidance proved.

### E08 — Attempt-02 reconciliation and next gate

Historical E01–E06 and C01–C16 synthetic results remain attributed to attempt-01;
behavioral suites were not rerun for this documentation-only checkpoint.
Historical crosswalk above retains AC/Case/Task links unchanged. Current results:

| AC / Tasks | Synthetic / local | Native attempt-02 | Aggregate |
| --- | --- | --- | --- |
| AC1 / C01–C06,C16 / T01–T03,T05–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC2 / C07,C08,C15,C16 / T02,T04–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC3 / C09,C10,C13,C14 / T02,T04,T06–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC4 / C11–C13 / T02–T04,T06–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC5 / C01–C16 / T01–T08 | historical PASS | both NOT_RUN | PARTIAL |
| AC6 / C01–C16 scope / T01–T06,T08 | historical PASS; current diff documentation only | no product effect | PASS within reviewed scope |

T07 authorization gap resolved for conversations; installed-baseline gap remains.
T08 current crosswalk and partial handoff reconciled; full completion still PARTIAL.
Input-head [CI](https://github.com/rgomids/axiom/actions/runs/39957542043) and
[Delivery metadata](https://github.com/rgomids/axiom/actions/runs/39957632827)
were read through GitHub API: completed/success, exact input head. These reads do
not certify a later documentation commit; final-head checks and publication
read-back belong to attempt-02 completion event. Local documentary/security
validation results likewise belong to that event after actual execution.

Technical decision **IMPL2-TD-01** (TECHNICAL_LOCAL; executor-ai; scope_impact none):
question: proceed against old pair, replace host installation, or hold conversations?
Chosen: hold both dependent conversations and reconcile documentary Evidence.
Alternatives: old-pair conversation cannot validate approved content; automatic
replacement exceeds E07's explicit update gate. Evidence: actual read-back and
hash comparison above, [T07](tasks.md#axm39-t07--two-native-conversations-external-authorization-boundary),
[protocol](examples.md#two-native-conversations--future-protocol-not_run).
Reversible: rerun preflight on a legitimately available baseline-compatible host.
No material architecture/product/design decision introduced.

**DF-01 DEFERRED_NOT_IMPLEMENTED** remains unchanged: stale predecessor index
summaries, historical source above; no optional work implemented.
Remaining work: obtain an authorized baseline-compatible installed host/pair and
supported docs read-back (specific update permission if host replacement is needed),
observe loaded revisions and legitimate auth availability, execute both real E07
conversations, reconcile AC1–AC5/T07/T08, independent review.
Next gate: **resume_implement_after_host_baseline**; safe independent review may
proceed. No configured continuation callback verified: **AWAITING_EXECUTION_REPORT**.
No merge, release, acceptance, Done or readiness claim.

## Attempt-03 — Authorized host upgrade refused

2026-10-23. Same Draft PR #343 and branch; clean input local/remote/PR HEAD
`03b54d3cb400d7058aedde20235f5091e54e3dee`. Spec 012 r1, Plan r1, historical
Tasks r1, examples and E01–E06 preserved. Full Linear comment inventory returned
`hasNextPage: false`: one attempt-03 prompt/accepted dispatch, no later competing
dispatch or completion at entry. Scoped process executable observation found no
second Claude/installer; existing Codex processes do not establish another Task
execution. This bounded observation is not a global ownership lease.

[Host/skills authority](https://linear.app/rgomids/issue/AXM-39#comment-5e2b99c4)
confirmed: Maestro's delegated declaration of Rafael's 10:35 BRT authorization,
without independent human authentication. Grants existing-host Axiom/four-skill
update and T07/T08, not state migration/repair, credentials, extraordinary expense,
merge, release or acceptance. Shared field-note call again returned tool-not-found;
canonical source/events remained available. Direct sequential execution: no proven
lower-capability hierarchy or useful independent writer unit.

### E07 — Candidate, backup, refusal and actual host read-back

Actual host: Darwin arm64, Codex CLI 0.163.0, Claude Code 2.1.308. Before update,
Axiom reported 0.20.0/dd2ab51f609e/clean; `docs inspect` returned invalid_command.
Both Runtime statuses reported compatibility 3/current receipt, but all eight
installed skill files retained the older digests in attempt-02's table.

Official source installer was inspected but **not run**: it requires
`axiom.receipt`; existing installation has release `installation.receipt`.
No receipt conversion or ownership bypass attempted. Official generation-2
archive route supports documentation read-back and existing owned upgrades:

```sh
./scripts/build-release-archives.sh --version 0.21.1 --output <private-archive-dir> --format 2 --development --only macos-arm64
./scripts/install-release.sh --helper <independently-built-helper> --helper-sha256 <actual-helper-sha256> --verify-only --archive <archive> --checksums <SHA256SUMS> --bin-dir <existing-bin-root> --receipt-dir <existing-receipt-root>
# Same verified arguments, without --verify-only, attempted the owned upgrade.
```

Commands above are sanitized records, not literal commands with usable paths.
Version 0.21.1 comes from the input commit's `.release-please-manifest.json`;
`--development` records **release=false**, not a published release or new version.
Candidate metadata: full source commit `03b54d3cb400d7058aedde20235f5091e54e3dee`,
revision `03b54d3cb400`, clean, generation 2, darwin/arm64. Archive SHA-256
`a94604c94b1658fb377d1d125566ae5b005b9fe10c25061d3cc9573ed3fea9b1`;
candidate binary `174911796d7ecbe7686369b99d8c1393009a48b390a065b1054f1bf2404f3d34`.
All four archive skill bytes were compared directly with approved source; each
matches the approved SHA-256 in attempt-02's table. Candidate is **not installed**.

A first verify-only invocation lacked the required independently pinned helper
and refused before installation. Corrected using a helper built from the same
approved checkout with the official `cmd/axiom-archive-verify` identity convention;
its SHA-256 supplied explicitly. Actual verify-only then exited 0:
`archive_verify_success: format=2 platform=macos architecture=arm64`.

Before the effectful call, 12 targeted regular non-symlink files were copied with
metadata preserved to a private, retained host backup: binary, installation
receipt, two skill receipts and eight SKILL.md files. SHA-256 and permission modes
recorded locally. No credentials, unrelated installations or broad host inventory
were backed up or imported. HEAD/worktree rechecked unchanged immediately before
installer execution.

**Owned upgrade exited 1**:

```text
install_error: owned upgrade refused: Upgrade blocked before any effect: state_unsafe
install_next: Existing Axiom state is unrecognized, modified, ambiguous, or unsafe; nothing was changed, so preserve it for operator review
```

This is the installer's compatibility-state refusal, not evidence of a defective
012 presentation rule. Exact offending state object/cause **UNKNOWN**; no state
contents, credentials or repair were inspected. Source mapping was reviewed in
`cmd/lingo/install_posix.go` and `internal/install/upgrade.go`: preview refuses
before applying managed effects. No isolated-state override, receipt fabrication,
manual binary replacement or migration used to bypass the refusal.

A following status batch also invoked `runtime codex install` and `runtime claude
install` on the unchanged old binary. Both returned `already_configured` with old
hashes/current receipts, **not** a baseline update. These dependent calls should
have been gated on installer success; subsequent file read-back confirmed no
managed-content change. No candidate skill installer was invoked.

| Actual post-attempt check | Result |
| --- | --- |
| Binary SHA-256 before/after | `d33992f315c3d72d999cce200b46eb62187096d13e6ef76c2f8adf192fbe6699`, unchanged |
| Binary/version | 0.20.0/dd2ab51f609e/clean, unchanged |
| All 12 backup targets | Exact original SHA-256 and permission modes retained |
| Four skills, both roots | Older attempt-02 digests retained; approved content not installed |
| Installed documentation | `docs inspect` still invalid_command; no `.axiom-docs` directory created |
| Rollback | NOT_NEEDED_NO_MANAGED_CHANGE; backup retained, restore not executed |
| Codex authentication availability | `codex login status`: logged in using ChatGPT; no secret inspected |
| Claude authentication availability | `claude auth status --json`: loggedIn=true, oauth_token, firstParty; identity fields not emitted or retained |

Authentication status is an availability observation, not successful inference.
No existing session was restarted. Installed != loaded != authenticated != observed.

| Native record | Installed / loaded / authentication | Prompt / answer | Result |
| --- | --- | --- | --- |
| E07-Codex attempt-03 | Older pair / UNKNOWN / availability reported | None submitted / none produced | NOT_RUN; BLOCKED_HOST_BASELINE after state_unsafe |
| E07-Claude attempt-03 | Older pair / UNKNOWN / availability reported | None submitted / none produced | NOT_RUN; BLOCKED_HOST_BASELINE after state_unsafe |

No live Project/Workflow selected, no conversational UUID, Stage, Runtime state,
blocker, success+blocked or graph/text rendering case executed. Prepared E07
prompts remain historical; no synthetic transcript substitutes for a native run.

### E08 — Attempt-03 reconciliation and remaining work

| AC / cases / Tasks | Historical synthetic | Native attempt-03 | Current aggregate |
| --- | --- | --- | --- |
| AC1 / C01–C06,C16 / T01–T03,T05–T08 | PASS, attempt-01 | NOT_RUN | PARTIAL |
| AC2 / C07,C08,C15,C16 / T02,T04–T08 | PASS, attempt-01 | NOT_RUN | PARTIAL |
| AC3 / C09,C10,C13,C14 / T02,T04,T06–T08 | PASS, attempt-01 | NOT_RUN | PARTIAL |
| AC4 / C11–C13 / T02–T04,T06–T08 | PASS, attempt-01 | NOT_RUN | PARTIAL |
| AC5 / C01–C16 / T01–T08 | PASS, attempt-01 | Both NOT_RUN | PARTIAL |
| AC6 / C01–C16 scope / T01–T06,T08 | Historical PASS; current scoped documentary review | No native conversation | PASS_SCOPED_REVIEW |

T01–T06/E01–E06 remain historical PASS, not rerun. T07 **BLOCKED_HOST_BASELINE**
with actual update refusal **BLOCKED_STATE_UNSAFE**. T08 **PARTIAL**, current
AC/case/Evidence join reconciled; full completion awaits both native records.
Input-head CI and both metadata runs re-read completed/success (39960629686,
39960629774, 39960719716); these do not certify this later documentary commit.
Actual local validation and final-head CI/SHA belong to attempt-03's completion
receipt, published after execution/read-back, never inferred from prior runs.

Current local checks: PASS `./scripts/validate-repository.sh .` (native Runtime
behavioral scenarios SKIP/UNVERIFIED), 43 local Markdown paths, `git diff --check`,
scoped complete diff review, `./scripts/check-sensitive-files.sh --staged .` and
`gitleaks git --staged --redact --no-banner` (no leaks). No Go behavior changed;
archive build compiled the exact candidate, historical behavior suites not repeated.
This self-review is not independent review or native acceptance.

Technical decisions (TECHNICAL_LOCAL, executor-ai, scope_impact:none):

- **IMPL3-TD-01:** choose official generation-2 development archive at approved
  commit over source installer or latest stable; rationale: existing release
  ownership and installed-doc binding require the supported archive route, while
  latest stable cannot establish approved 012 skill content. Version comes from
  source manifest; archive build publishes nothing. Evidence: command reference
  installed-doc/source-install sections, archive metadata and verified hashes above.
- **IMPL3-TD-02:** stop dependent native runs after state_unsafe; retain original
  pair and backup instead of forcing replacement or redirecting state roots.
  Rationale: refusal precedes managed effects; repairing unidentified protected
  state or testing old skills cannot establish approved native behavior. Reversal:
  retry official preflight after separately scoped state reconciliation.
- **IMPL3-TD-03:** execute directly; no advantageous independent unit with proven
  strictly lower capability. Deterministic tools own archive/hash/read-back checks.

**DF-01 DEFERRED_NOT_IMPLEMENTED:** stale predecessor summaries in Specifications
index remain outside scope; show prominently at pre-merge. Only current 012 index
checkpoint reconciled; no predecessor history changed. No optional finding promoted
to a requirement; state_unsafe is a required-action blocker, not DF-01.

Remaining work: scoped diagnosis/reconciliation of the existing installer's
compatibility-state refusal, without widening this update grant; retry official
host update; verify installed docs, both skill sets and loaded session identity;
run both real E07 conversations; reconcile AC1–AC5/T07/T08 and independent review.
No material architecture/product/design decision proposed. No verified continuation
callback: **AWAITING_EXECUTION_REPORT**. Next gate:
**resume_implement_after_host_state_reconciliation**. No merge, release, Done or
human acceptance. Partial completion event/read-back stays on AXM-39.


## Attempt-04 — Manual host update preflight

2026-10-23, Darwin arm64. Local HEAD, exact remote branch and open Draft PR #343
all read back as `5c985516c006b96858ba033b8154f26bbdc0a0e2` before edits;
working tree clean. Same branch `docs/axm-39-conversational-presentation`.
All 38 Linear comments returned, `hasNextPage: false`; current card, accepted
baseline, receipts, attempt-04 prompt/dispatch and native/host authorities read.
No attempt-04 completion existed. [Attempt-04 prompt](https://linear.app/rgomids/issue/AXM-39#comment-f8154bf0)
attributes manual update to Maestro's declaration of Rafael's 10:55 BRT action;
this preflight verifies actual observable state, not the actor or update method.
[10:15 native authority](https://linear.app/rgomids/issue/AXM-39#comment-b3a7bcea)
and [host authority](https://linear.app/rgomids/issue/AXM-39#comment-5e2b99c4)
retain delegated provenance and exclusions. No installation retry, protected-state
inspection/migration/repair or receipt mutation. Shared field-note API returned
tool-not-found; card, comments and repository remained accessible.

### E07 — Attempt-04 installed, loaded and authentication observations

Commands ran on the actual Mac. Paths below use sanitized root aliases;
no credentials, account identity or raw host inventory retained. Results describe
this process environment, not every possible native app environment.

| Actual command/read | Result | Scope |
| --- | --- | --- |
| `uname -sm` | Darwin arm64 | actual executor |
| `axiom --json runtime codex status` | exit 0; 0.21.1 / 921507c27c5b / clean; compatibility 3, receipt current, four equivalent skills | compatible with this installed binary only |
| `axiom --json runtime claude status` | exit 0; same provenance, compatibility and digests | no loaded-session proof |
| `shasum -a 256 <installed-binary>` | `7eb8efaa69e386aa781531694f6ec93a5624d26f97d06531f224cb1e1a9023b9` | byte identity, not build attestation |
| `axiom --json docs inspect` | exit 1; failure, state unknown, reason source_unqualified; no paths/descriptor/binding, loaded unknown for both roots | command exists; verified documentation unavailable |
| `axiom --json skill inspect <skill>` for all four skills | exit 0; Skill arguments inspected; same 0.21.1 provenance | argument registry, not loaded content |
| `codex --version`; `claude --version` | 0.163.0; Claude Code 2.1.308 | executable observations |
| `codex login status` | exit 0; logged in using ChatGPT | direct login availability only |
| `claude auth status --json` projected to loggedIn/authMethod/apiProvider | exit 0; true / oauth_token / firstParty | no identity fields retained |
| `axiom --json runtime codex auth` | exit 1; incompatible / environment_override; override name env:OPENAI_API_KEY | subscription route not established in current environment |
| `axiom --json runtime claude auth` | exit 1; incompatible / environment_override; override name env:CLAUDE_CODE_OAUTH_TOKEN | subscription route not established in current environment |

No override values read, emitted or changed. Direct login does not override the
Axiom preflight refusal; no successful inference, billing route or usability claimed.
No login, refresh, environment cleanup or credential/configuration mutation.

Actual SHA-256 reads of all eight installed SKILL.md files, compared with approved
input HEAD and `git show 921507c27c5b:internal/codexruntime/skills/<skill>/SKILL.md`:

| Skill | Installed, both roots; also exact 921507c27c5b source | Approved PR baseline |
| --- | --- | --- |
| axiom-project | a249ea87133eeefd53f7e6bccf8b5d083c2bf87802f8141ec85183d52d9a892d | 775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d |
| axiom-workflow | 8b47aa65611d28395eab3695ce4ca90773e541800f72d4189838565268a7f235 | efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76 |
| axiom-runtime | 90a725ed0ac96aa3378b3c9860f5357c8ef63b090662a19d362c24c8cc0aad0b | 165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d |
| axiom-work-item | ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3 | same bytes |

Full installed/source hashes match the complete historical replaced set recorded
in E05, not the new 012 set. `git show --no-patch 921507c27c5b` identifies main's
release 0.21.1 commit, full SHA `921507c27c5b1ec5468508a6d48ce0e080e90a42`.
Source diff against PR confirms missing Project identity section, progressive
Stage guidance and Runtime/readiness distinctions in three installed skills.
This is an established baseline mismatch, not a version-string objection.

Diagnosis: [Resolve](../../../internal/operationaldocs/resolve.go) returns
source_unqualified before installation resolution when the compiled descriptor
is invalid or platform-unqualified; the reported diagnostic states this build
carries no documentation descriptor. Exact build/install method is UNKNOWN;
no conclusion about protected state safety follows. The previous attempt's
state_unsafe remains historical, neither reproduced nor proved repaired.
[Official commands](../../commands.md#inspect-installed-operational-documentation)
require verified descriptor-bound documentation and distinguish on-disk/loaded.
The [embedded revision owner](../../../internal/codexruntime/manifest.go) and
[installer](../../../internal/codexruntime/runtime.go) install their binary's set:
rerunning this binary's skill installer cannot supply approved PR bytes.
Manual copies, receipt edits, temporary-state bypass or replacement-binary installs
are not used. Current instruction explicitly forbids repeating installation;
there is no demonstrated safe authorized procedure resolving both gaps here.
No host file changed, so no new backup/rollback required or claimed.

| Native record | Installed / loaded / authenticated / observed | Prompt and response | Verdict |
| --- | --- | --- | --- |
| E07-Codex attempt-04 | old 012-predecessor set / UNKNOWN / direct login available, Axiom route incompatible / no inference observed | none submitted; none produced | NOT_RUN; BLOCKED_HOST_BASELINE; auth override independently blocks current route |
| E07-Claude attempt-04 | same predecessor set / UNKNOWN / direct login available, Axiom route incompatible / no inference observed | none submitted; none produced | NOT_RUN; BLOCKED_HOST_BASELINE; auth override independently blocks current route |

No native session started or reloaded, no Project/Workflow selected, no approved
prompt exercised. Identity/full operational UUID, both Stage summaries/details,
allowed/declared/resolved/observed Runtime, blocker/success+blocked and graph/text
fallback remain NOT_RUN in both native records. No synthetic transcript substitutes
for these observations. Current maintainer skill catalog cannot prove a native
product session loaded baseline bytes.

### E08 — Attempt-04 reconciliation and remaining work

| AC / cases / Tasks | E01–E06 synthetic/local | E07 attempt-04 | Aggregate |
| --- | --- | --- | --- |
| AC1 / C01–C06,C16 / T01–T03,T05–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC2 / C07,C08,C15,C16 / T02,T04–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC3 / C09,C10,C13,C14 / T02,T04,T06–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC4 / C11–C13 / T02–T04,T06–T08 | historical PASS | NOT_RUN | PARTIAL |
| AC5 / C01–C16 / T01–T08 | historical PASS | both NOT_RUN | PARTIAL |
| AC6 / C01–C16 scope / T01–T06,T08 | historical PASS; current documentary diff reviewed | no product effects | PASS_SCOPED_REVIEW |

T01–T06 and E01–E06 retain historical PASS within original scope; native coverage
for every C01–C16 remains NOT_RUN. T07 BLOCKED_HOST_BASELINE. E08/T08 PARTIAL:
join complete for observed partial state, full completion requires both E07 records.
No implementation defect established; only current checkpoint docs changed.
Input-head CI 39964035694 and metadata 39964035696/39964105079 read back
completed/success. Later final-head checks belong to completion receipt, not
these prior runs. Pending CI is IN_PROGRESS, never PASS by inheritance.
Current checks: PASS both focused regression commands from E06 with `-count=1`
(CLI 0.384s; codexruntime 0.807s), 47 local Markdown paths, unchanged historical
attempts 01–03 and scoped diff review. No production code changed; full package
suite not repeated. Repository validator and staged security gates are recorded
in the same-attempt completion receipt after actual execution.

technical_decisions[] (TECHNICAL_LOCAL; executor-ai; scope_impact:none):

- IMPL4-TD-01: stop dependent conversations on proven content/docs mismatch;
  prefer truthful partial evidence over old-skill inference or unapproved install.
  Rationale: both root hashes equal release source, missing approved guidance;
  docs refuses. Reversible when a baseline-compatible host is legitimately available.
- IMPL4-TD-02: record direct login and Axiom auth refusal separately; preserve
  current environment rather than changing credential routing to force success.
  Auth values never inspected; no claim all host surfaces share this environment.
- IMPL4-TD-03: execute directly, serialize documentary edits; no proven strictly
  lower-capability agent hierarchy or independent useful delegation unit.

**DF-01 — DEFERRED_NOT_IMPLEMENTED:** historical predecessor Specifications index
summaries remain outside baseline; preserve original evidence and highlight at
pre-merge. No optional finding implemented or promoted to a requirement.

Remaining work: obtain a legitimately baseline-compatible binary/four-skill pair
with verified installed docs through a separately scoped safe operator path;
recheck actual native child authentication without reading secrets, prove loaded
revisions, run both E07 conversations and reconcile AC1–AC5/T07/T08; independent
review and final-head CI. No protected-state repair or installation retry authorized
by this partial result. No material architecture/product/design decision required.
Next gate: resume_implement_after_verified_host_baseline_and_auth.
No configured continuation into another executor verified: AWAITING_EXECUTION_REPORT.
Completion callback is published/read back on AXM-39; publication alone does not
start review or establish implementation completion, acceptance, merge, release or Done.
