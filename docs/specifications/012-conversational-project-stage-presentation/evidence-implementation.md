# AXM-39 — Implementation Evidence, attempts 01–06

Current checkpoint: **attempt-06 PARTIAL; T07 BLOCKED_HOST_BASELINE;
native Codex/Claude NOT_RUN; T08 PARTIAL**. Host v0.21.2 docs verified and
subscription authentication observed in isolated child environments. Three skills
in each Runtime root still differ from approved PR #343; loaded content UNKNOWN.
No supported compatible PR installation established. See
[attempt-06 read-back](#attempt-06--stable-release-read-back-and-pr-baseline-mismatch).

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


## Attempt-05 — Transactional host preflight and official upgrade refusal

2026-10-23. Local HEAD, remote branch and Draft PR #343 all matched
`4914b28d38ab83497db37df37222eb7b1047cf6d`; clean working tree before work.
Linear inventory returned 42 comments, hasNextPage=false; attempt-05 was latest
prompt/dispatch, with no completion or later conflicting attempt. Current card,
Spec/Plan/Tasks r1, prior receipts and scoped
[host authority](https://linear.app/rgomids/issue/AXM-39#comment-1d523e01)
checked. Authority retains Maestro's reported provenance, without independent
human authentication. Prior implementation prompts authorize commit/push to this
same branch/PR; no merge/release/acceptance/Done. Attempts 01–04 below their
original headings and T01–T06 PASS scope remain historical and unchanged.

### E07 — Actual baseline, backup and supported packaging

Actual host Darwin arm64; Axiom 0.21.1 / 921507c27c5b / clean,
Codex CLI 0.163.0 and Claude Code 2.1.308. Both Runtime status commands report
compatibility 3, current receipts and equivalent files against the installed
release. All eight installed skill digests still equal attempt-04's table;
Project, Workflow and Runtime differ from approved PR source, Work Item matches.
Both loaded revisions remain UNKNOWN. All four `skill inspect` operations
are metadata inspection, not proof of loaded bytes.

Official `compatibility inspect` now reports valid_v1 / v1_readable_state,
zero findings, digest
`beb1caaaefbcd9ea1b82d9940a042e66bf20aedcbf679aff76a8fd75cfdf2758`.
`recovery inspect` reports no interrupted local publication or recovery plans.
These observations do not explain or certify repair of historical state_unsafe.
No protected compatibility state, Project, Workflow, Profile or credential read
or edited manually. Only supported sanitized diagnostics used for state.

Before candidate evaluation, 12 managed files were copied into a retained
private backup: binary, release installation receipt, both skill receipts and
eight SKILL.md files. Original SHA-256, UID/GID and permission modes captured;
backup objects and manifest are 0600 inside 0700 directories. Independent
restore rehearsal into that private backup namespace verified every hash,
mode and UID/GID. Backup manifest SHA-256:
`3862461b917c6b6556cecd7ca9b2b002981a1f9a63bd86bd775a58515a6e0805`.
No backup contains vendor authentication/configuration. Actual private path is
retained locally, omitted here. Documented [pair restoration](../../commands.md#cli-output-and-help)
requires the complete prior managed set at original paths with matching binary;
no restore to active roots was needed or executed. Unrelated skill content
was neither copied into evidence nor changed.

Official packaging, with version taken unchanged from source release manifest:

```sh
./scripts/build-release-archives.sh --version 0.21.1 --format 2 --development --only macos-arm64 --output <private-artifact-root>
./scripts/install-release.sh --helper <verified-helper> --helper-sha256 <actual-helper-sha256> --verify-only --archive <archive> --checksums <SHA256SUMS> --bin-dir <existing-bin-root> --receipt-dir <existing-receipt-root>
<candidate> --json upgrade --archive <archive> --checksums <SHA256SUMS> --bin-dir <existing-bin-root> --receipt-dir <existing-receipt-root>
```

Build and strict archive verification PASS. Archive metadata: format 2,
release=false, sourceState=clean, revision=4914b28d38ab,
sourceCommit=4914b28d38ab83497db37df37222eb7b1047cf6d, macos/darwin/arm64.
All four archived skills byte-match approved input HEAD and the hashes in
attempt-04's approved column. No new release version or fabricated descriptor.

| Artifact | SHA-256 |
| --- | --- |
| Candidate archive | 7ff863ede6ccbf859435b08867f479b444bfcaac51532c982bd141da6d68cebc |
| Candidate binary | 1ab9b235841f277560429dc514fe2e06db8b98341d371085b340c022f6c38a6b |
| Active binary, before and after | 7eb8efaa69e386aa781531694f6ec93a5624d26f97d06531f224cb1e1a9023b9 |

Read-only owned-upgrade preview exits 1:
**Upgrade blocked before any effect: divergent_equivalent_version**;
next: **Downgrade and divergent same-version replacement are not supported**.
Existing release receipt identifies the same 0.21.1 version and exact active
binary hash. [Version guard](../../../internal/install/upgrade.go) rejects
same-version divergent bytes before state transition or publication. No apply,
installer mutation, manual receipt edit, version inflation or alternate active
state/prefix used to evade this refusal. Source installer is unsuitable: its
separate axiom.receipt ownership and descriptor-free build do not replace this
release-owned installation through the supported route. The three divergent
skills therefore remain untouched in both roots, as does the matching fourth.

Independent official package validation ran
`scripts/test-docs-offline.sh --archive <archive> --checksums <SHA256SUMS>`:
PASS macos-arm64; disposable-prefix docs verified/compatible, 26 corpus files,
closed links, integrity inventory 29 files/413507 bytes. Descriptor produced by
`axiom-docs-build` through generation-2 packaging, not written by hand.
Pinned generation-1 installer rejection, extra/swapped content, stale metadata,
tampered docs and modified binary negative checks PASS. Script's aggregate
qualification remains PARTIAL: network/source denial and native Runtime
observation NOT_RUN. This independent test does not replace active-host update:
active `docs inspect` still refuses source_unqualified, returns no paths.

### E07 — Official authentication and native-session boundary

Parent diagnostics reproduce environment_override naming only OPENAI_API_KEY
and CLAUDE_CODE_OAUTH_TOKEN. No values inspected or retained. Axiom's official
diagnostic explicitly directs removal of overrides from the child environment
before repeating auth. [OpenAI authentication](https://learn.chatgpt.com/docs/auth)
separates cached ChatGPT login from API-key login;
[Claude precedence](https://code.claude.com/docs/en/authentication#authentication-precedence)
puts the environment token above saved subscription login. Credential stores
and configuration roots were left intact. Used only process-scoped omission:

```sh
env -u OPENAI_API_KEY -u CLAUDE_CODE_OAUTH_TOKEN axiom --json runtime codex auth
env -u OPENAI_API_KEY -u CLAUDE_CODE_OAUTH_TOKEN axiom --json runtime claude auth
```

Both exit 0, subscription_observed, evidenceKind local_observation:
Codex method chatgpt / version 0.163.0; Claude method claude.ai / version 2.1.308.
Both report usability unproven_until_real_dispatch and before_each_dispatch
revalidation. Direct `codex login status` confirms ChatGPT;
`claude auth status --json`, projected to nonidentity fields only, confirms
loggedIn=true / claude.ai / firstParty. Parent variables still present by name;
no persistent shell/profile/configuration or login/logout/refresh operation.
This verifies the official isolated auth diagnostic route, not a dispatch.

| Native record | Installed / loaded / auth / observed conversation | Result |
| --- | --- | --- |
| E07-Codex attempt-05 | predecessor skills / UNKNOWN / subscription_observed in isolated child / none | NOT_RUN; BLOCKED_HOST_BASELINE |
| E07-Claude attempt-05 | predecessor skills / UNKNOWN / subscription_observed in isolated child / none | NOT_RUN; BLOCKED_HOST_BASELINE |

No fresh conversation started, no prompt submitted, no model response,
inference or cost incurred. No Project/Stage/Runtime case or graph/text behavior
observed natively; no synthetic output relabelled as real conversation.
All 12 active backup targets re-read byte-exact with original ownership/modes;
rollback NOT_NEEDED_NO_MANAGED_CHANGE. Backup retained. Installed != loaded;
auth observed != usable dispatch != conversation PASS.

### E08 — Attempt-05 reconciliation and remaining work

| AC / cases | Historical local scope | Native attempt-05 | Aggregate |
| --- | --- | --- | --- |
| AC1 / C01–C06,C16 | T01–T06 PASS preserved | NOT_RUN | PARTIAL |
| AC2 / C07,C08,C15,C16 | T01–T06 PASS preserved | NOT_RUN | PARTIAL |
| AC3 / C09,C10,C13,C14 | T01–T06 PASS preserved | NOT_RUN | PARTIAL |
| AC4 / C11–C13 | T01–T06 PASS preserved | NOT_RUN | PARTIAL |
| AC5 / C01–C16 | synthetic regressions PASS | both conversations NOT_RUN | PARTIAL |
| AC6 / C01–C16 scope | scoped documentary diff review | no product effects | PASS_SCOPED_REVIEW |

T07 BLOCKED_HOST_BASELINE; T08 PARTIAL. Historical T01–T06/E01–E06 retain their
original PASS, with focused regression commands rerun successfully using
`-count=1` (CLI 0.406s, codexruntime 0.990s). Full package suite not repeated:
no production code changed. Input-head CI PR Checks 39966547119 and metadata
39966547128 completed/success. Final-head CI and repository/security gates
are recorded in attempt-05's completion callback after actual execution/read-back;
input-head results do not establish later-head CI.

Current local checks PASS: `./scripts/validate-repository.sh .`, 203 local
Markdown path references, historical-content preservation assertions, complete
scoped diff review, `git diff --check`, staged sensitive-file check and redacted
Gitleaks (no leaks). Repository validator's Runtime behavioral scenarios remain
SKIP/UNVERIFIED. These checks do not establish native acceptance or independent Review.

technical_decisions[]: TECHNICAL_LOCAL, executor-ai, scope_impact:none.

- IMPL5-TD-01: use supported generation-2 development packaging with manifest's
  actual 0.21.1 version, verify-only then read-only upgrade preview. Alternative
  manual descriptor/source-installer replacement would not preserve official
  ownership/docs qualification. Stop on observed same-version refusal; do not
  invent a release identity to evade the guard.
- IMPL5-TD-02: use official child-environment guidance with only two reported
  overrides omitted temporarily; retain legitimate saved logins and configuration.
  Prefer this reversible diagnostic over persistent credential routing changes.
  Both official auth reports PASS availability; usability remains unproven.
- IMPL5-TD-03: retain/rehearse private backups before evaluation and stop host
  effects after official refusal. Disposable-prefix package checks supply
  independent source/descriptor evidence only; they do not qualify the active
  installation or bypass its version guard.
- IMPL5-TD-04: execute directly, no verified strictly lower-capability hierarchy;
  deterministic hashes/tests provide bounded independent checks.

**DF-01 DEFERRED_NOT_IMPLEMENTED:** stale predecessor Specifications summaries
remain outside scope, unchanged; highlight at pre-merge. No additional optional
requirement, ADR or production change introduced.

Remaining work: obtain an official supported owned-upgrade path for the approved
binary/docs/skill set, currently refused for divergent equivalent version; then
verify both installed roots and fresh loaded revisions, recheck exact child auth,
execute both real conversations, reconcile T07/T08 and independent Review.
No manual compatibility-state or receipt repair authorized. No new contract or
version policy adopted. Next gate:
**resume_implement_after_supported_owned_host_upgrade**.
No demonstrated continuation callback; **AWAITING_EXECUTION_REPORT**.
No merge, release, acceptance, closure or Done. Same-AXM completion event records
actual final HEAD, checks and read-back, without claiming Review was dispatched.


## Attempt-06 — Stable release read-back and PR baseline mismatch

2026-10-24, actual Mac Darwin arm64. Input local HEAD, exact remote branch and
open Draft PR #343 matched `facb807fa11473c7cfafdcf487349a391d8f1c6d`;
working tree clean, branch `docs/axm-39-conversational-presentation`.
Current Linear issue and all 45 comments read, `hasNextPage=false`.
[Attempt-06 dispatch](https://linear.app/rgomids/issue/AXM-39#comment-2552fe39)
was latest; no competing continuation/completion observed. Dispatch acceptance
is not execution. Read back [T07 authority](https://linear.app/rgomids/issue/AXM-39#comment-b3a7bcea),
[scoped host authority](https://linear.app/rgomids/issue/AXM-39#comment-1d523e01)
and [attempt-05 receipt](https://linear.app/rgomids/issue/AXM-39#comment-271875ce).
Delegated Maestro provenance remains reported, not independent human authentication;
current continuation authorizes bounded T07/T08 and same-branch commit/push.
Spec/Plan/Tasks r1, six AC, C01–C16, attempts 01–05 and historical T01–T06 retained.

### E07 — Supported preflight, release versus PR versus installed versus loaded

| Actual read-only command/read | Observed result | Limit |
| --- | --- | --- |
| `axiom --json version` | 0.21.2 / c52f1ff78c91 / clean | stable build, not PR #343 |
| `axiom --json compatibility inspect` | valid_v1 / v1_readable_state; four current skills | current against installed binary only |
| `axiom --json docs inspect` | verified / ok; compatible installation; both Runtime file manifests match, receipts recognized, loaded unknown | installed paths returned and qualified; no native load proof |
| `axiom --json runtime codex status` | success; skillSetVersion 3, binaryCompatibility 3; four equivalent skills, current receipt | Codex installed release baseline |
| `axiom --json runtime claude status` | success; same compatibility and four equivalent skills, current receipt | Claude installed release baseline |
| `axiom --json skill inspect <skill>` for all four domain skills | success; Project 13, Work Item 33, Workflow 14, Runtime 18 command entries | argument registry inspection, not skill loading |
| `codex --version`; `claude --version` | 0.163.0; Claude Code 2.1.310 | executable observations |
| isolated `axiom --json runtime codex auth` | subscription_observed / chatgpt / local_observation | usability unproven until actual dispatch |
| isolated `axiom --json runtime claude auth` | subscription_observed / claude.ai / local_observation | usability unproven until actual dispatch |

Auth commands used `env -u OPENAI_API_KEY -u CLAUDE_CODE_OAUTH_TOKEN` only in
child processes. No credential reads, persistent configuration changes,
login/logout, inference or API key use. No bootstrap Run or installation invoked.
GitHub release [v0.21.2](https://github.com/rgomids/axiom/releases/tag/v0.21.2)
read back: source `c52f1ff78c915f00f7b9c4fea1e399bf1ee87b5e`, published
2026-10-24T19:21:20Z. [PR #345](https://github.com/rgomids/axiom/pull/345)
merged as `08af192b6313ae2a56d8a0817af52109a958f943`.
[Issue #344](https://github.com/rgomids/axiom/issues/344) and
[DF-1 #346](https://github.com/rgomids/axiom/issues/346) remain OPEN; no status changed.
The latter is a separate existing deferral, not AXM-39 DF-01 or a new Task.

Binary SHA-256: `21a0d6884b97d5ca7b43634024d622532c51da3e3e3dfa8f75195d5e67c02057`.
Verified docs inventory SHA-256:
`e28d1a05129499eca41e894dc54ac88baba3f407999574c6ffa8b8a3fc24660d`;
29 files / 413507 bytes. Expected and observed installed skill manifest:
`1f59038c225cc47709eab44fad41a6257e00894003920e9c7a49aad2582b8cc5`.
Returned docs paths resolve under the qualified installation generation;
private absolute host paths omitted. Runtime roots checked against documented
Codex `$HOME/.agents/skills` and Claude `$HOME/.claude/skills` defaults, including
absence of an effective alternate root. Every installed SKILL.md read and hashed;
both roots have identical per-skill bytes below. PR column hashes actual input-HEAD
`internal/codexruntime/skills/<skill>/SKILL.md` bytes, not stable-release guidance.

| Skill | Installed Codex AND Claude SHA-256 / bytes | Approved PR SHA-256 / bytes | Comparison |
| --- | --- | --- | --- |
| axiom-project | a249ea87133eeefd53f7e6bccf8b5d083c2bf87802f8141ec85183d52d9a892d / 31717 | 775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d / 34213 | MISMATCH |
| axiom-work-item | ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3 / 36577 | ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3 / 36577 | MATCH |
| axiom-workflow | 8b47aa65611d28395eab3695ce4ca90773e541800f72d4189838565268a7f235 / 27649 | efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76 / 31401 | MISMATCH |
| axiom-runtime | 90a725ed0ac96aa3378b3c9860f5357c8ef63b090662a19d362c24c8cc0aad0b / 15001 | 165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d / 17182 | MISMATCH |

Exact line comparison: installed Project lacks PR lines 502–539; installed Runtime
lacks PR lines 203–234. Workflow replaces installed lines 338–339 with PR 338–340,
replaces installed line 342 with PR 343, and lacks PR lines 411–464.
Work Item has no difference. Thus stable release current != AXM-39 PRESENT.
This maintainer session reads repository files; that is not proof either native
product conversation loaded them. Loaded revision remains UNKNOWN for both.

### E07 — Supported installation boundary and independent conversation records

No approved supported route was established to replace this release-owned host
with the PR pair. The branch release manifest still identifies 0.21.1; attempt-05's
official development archive is also 0.21.1. The current host is 0.21.2.
Read [upgrade version guard](../../../internal/install/upgrade.go): lower candidate
version yields `downgrade_refused`; divergent equal version yields
`divergent_equivalent_version`. This is source inspection of the current constraint,
not a newly executed upgrade preview. Attempt-05's refusal remains historical.
[Packaging](../../../scripts/build-release-archives.sh) accepts explicit version
metadata; supplying a higher invented version would violate this task's prohibition.
Installing PR skills while retaining the stable binary/docs would leave a forbidden
mixed baseline. Source install does not preserve the qualified release-owned pair;
an alternate prefix is not proof of qualification of this authorized native host.
No upgrade/apply, manual receipt/state edits, fake version, host mutation or
credential mutation performed. Existing valid host preserved; backup/rollback
not newly needed because no managed mutation was attempted. Prior backup evidence
is historical, not a current restore claim.

| Record | Authority / installed / loaded / auth | C01–C16 and AC1–AC5 native outcomes | Result |
| --- | --- | --- | --- |
| E07-Codex attempt-06 | T07 grant read back / release 0.21.2, three PR mismatches / UNKNOWN / subscription_observed | every native case NOT_RUN; no prompts dispatched, responses or rendering observed | NOT_RUN; TECHNICALLY_BLOCKED, BLOCKED_HOST_BASELINE |
| E07-Claude attempt-06 | T07 grant read back / release 0.21.2, three PR mismatches / UNKNOWN / subscription_observed | every native case NOT_RUN; no prompts dispatched, responses or rendering observed | NOT_RUN; TECHNICALLY_BLOCKED, BLOCKED_HOST_BASELINE |

Blocker concerns supported installation qualification, not material architecture,
product or design. No HUMAN_DECISION_REQUIRED gate invented. Missing action:
a supported, specifically authorized owned-installation path for the reviewed
PR binary/docs/four-skill pair compatible with current 0.21.2 ownership, followed
by backup/rollback and fresh installed/loaded read-back. Host authority is scoped;
it does not authorize bypassing the version guard or inventing version metadata.
No old-skill conversation, synthetic transcript or login-only result substitutes
for E07. Exact authorized [conversation protocol](examples.md#two-native-conversations--future-protocol-not_run)
remains pending without changes.

### E08 — Attempt-06 AC reconciliation and bounded handoff

| AC / cases | Historical local result | Attempt-06 native result | Aggregate |
| --- | --- | --- | --- |
| AC1 / C01–C06,C16 | PASS within E01–E06 limits | Codex/Claude NOT_RUN | PARTIAL |
| AC2 / C07,C08,C15,C16 | PASS within E01–E06 limits | Codex/Claude NOT_RUN | PARTIAL |
| AC3 / C09,C10,C13,C14 | PASS within E01–E06 limits | Codex/Claude NOT_RUN | PARTIAL |
| AC4 / C11–C13 | PASS within E01–E06 limits | Codex/Claude NOT_RUN | PARTIAL |
| AC5 / C01–C16 | regression evidence retained | both conversations NOT_RUN | PARTIAL |
| AC6 / scope and negative review | historical PASS; current two-document diff reviewed | no product mutation | PASS_SCOPED_REVIEW |

T01–T06 remain historical PASS; T07 BLOCKED_HOST_BASELINE; T08/E08 PARTIAL.
No demonstrated implementation violation; changes restricted to this Evidence and
Tasks checkpoint. Current local checks:

| Actual validation | Result |
| --- | --- |
| `go test ./internal/cli -run 'Test(OutputSeparation|WorkflowShowSynthetic|Presentation)' -count=1` | PASS, 0.875s |
| `go test ./internal/codexruntime -run 'Test(RenderedWorkflow|OperatorConversational|WorkflowSkill)' -count=1` | PASS, 1.318s |
| `./scripts/validate-repository.sh .` | PASS; Runtime behavioral scenarios SKIP/UNVERIFIED |
| Local Markdown paths and historical preservation assertions | PASS; 104 paths, attempts 01–05 Evidence and Tasks r1 unchanged |
| `git diff --check`; full two-document diff review | PASS |
| `./scripts/check-sensitive-files.sh --staged .` | PASS |
| `gitleaks git --staged --redact --no-banner` | PASS, no leaks |

Full package suite not repeated: no production code or skill bytes changed.
Final security checks are repeated against the actual staged documentation.
Input-head CI `facb807` read back PR Checks and delivery-metadata SUCCESS;
final-head CI must be checked separately after the documentation push.
Skipped native Runtime validator scenarios cannot qualify E07 or product acceptance.

technical_decisions[]: TECHNICAL_LOCAL; executor-ai; scope_impact:none.

- IMPL6-TD-01: compare all eight installed skill files against all four approved
  PR files, separately from release-current status; hashes and line differences
  distinguish release qualification from AXM-39 presence. Alternative inference
  from current receipts would falsely qualify three unchanged predecessor skills.
- IMPL6-TD-02: stop dependent sessions at incompatible PR installation boundary;
  retain stable host. Version-policy changes, fake metadata or skill-only replacement
  are outside this task. Source guard inspection is sufficient to reject those
  options without repeating a known unsupported installation attempt.
- IMPL6-TD-03: omit only two named overrides in temporary auth children, preserve
  existing legitimate subscription login; auth observation is not successful inference.
- IMPL6-TD-04: execute directly with deterministic checks; no verifiably strictly
  lower-capability model hierarchy, so no delegation.

**DF-01 DEFERRED_NOT_IMPLEMENTED:** stale predecessor Specifications index summaries
remain unchanged/out of scope; highlight at pre-merge. Existing #346 remains
separately deferred and NOT_IMPLEMENTED here. No optional implementation added.

remaining_work[]: qualify supported authorized PR pair on actual host; verify all
installed bytes and fresh loaded revisions; revalidate exact child auth; run one
native Codex and one native Claude conversation across C01–C16; reconcile E07/E08,
AC1–AC5 and T07/T08; verify final-head CI; then independent Review.
Next gate **RESUME_IMPLEMENT**. No merge/release/acceptance/closure/Done.
One attempt-06 stage.partial callback supplies actual pushed HEAD, final-head CI,
checks, technical decisions, limitations and remaining work with raw Linear read-back.
No demonstrated authorized automatic continuation callback beyond that stage report;
bounded handoff **AWAITING_EXECUTION_REPORT**.

## Attempt-07 — Verified backup and supported downgrade refusal

2026-10-24, actual Mac Darwin arm64. Local branch, remote branch and open Draft
PR #343 matched input HEAD `1d40aabe4a1a8a34a42d219e08cccb985e3db671`; tree clean.
Current Linear issue and all 48 comments read, hasNextPage=false; no attempt-07
completion found. [Continuation and scoped authority](https://linear.app/rgomids/issue/AXM-39#comment-78b7c91f)
and [dispatch receipt](https://linear.app/rgomids/issue/AXM-39#comment-919289ef)
read back. Authority is Maestro's delegated report of Rafael's 24/10/2026
approximately 19:33 BRT authorization, not independently authenticated human
consent. It permits supported alignment with backup/rollback and the previously
authorized two conversations; it does not override version/ownership gates.
Spec/Plan/Tasks r1, six AC, C01–C16, DESIGN NOT_APPLICABLE and non-goals retained.

### E07 — Actual host, ownership and complete private backup

Read-only `axiom --json version`, `compatibility inspect`, `docs inspect` and
both `runtime <runtime> status` commands PASS within their installation scope:
0.21.2 / c52f1ff78c91 / clean; valid_v1; docs verified/ok and compatible;
both receipts current, four equivalent skills, skillSetVersion/binaryCompatibility 3.
Binary SHA-256 `21a0d6884b97d5ca7b43634024d622532c51da3e3e3dfa8f75195d5e67c02057`;
installed manifest `1f59038c225cc47709eab44fad41a6257e00894003920e9c7a49aad2582b8cc5`.
Loaded revision remains UNKNOWN in both Runtimes. No native session started.

All eight installed files were read and compared, not inferred from receipts.
Each installed digest below applies independently to Codex and Claude:

| Skill | Installed SHA-256 in each root | PR SHA-256 | Comparison |
| --- | --- | --- | --- |
| axiom-project | a249ea87133eeefd53f7e6bccf8b5d083c2bf87802f8141ec85183d52d9a892d | 775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d | MISMATCH |
| axiom-work-item | ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3 | ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3 | MATCH |
| axiom-workflow | 8b47aa65611d28395eab3695ce4ca90773e541800f72d4189838565268a7f235 | efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76 | MISMATCH |
| axiom-runtime | 90a725ed0ac96aa3378b3c9860f5357c8ef63b090662a19d362c24c8cc0aad0b | 165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d | MISMATCH |

Neither skill lock had an open holder at observation. Release-state inventory
contained only installation.receipt; no interrupted-operation marker observed.
No symlinks, foreign owners or group/other-writable objects in backed-up scope.
A new retained PRIVATE backup contains 42 managed files: binary, installation
receipt, both skill receipts, eight skill files, installed docs and binding.
Manifest records exact source locations, SHA-256, UID/GID and modes, plus 18
directory records; raw paths and backup content remain private. Manifest SHA-256:
`67671b70e4ada94edbdae815aedf35a9a372a418acf844b522b5a111e841cf56`.
Backup files are 0600 under 0700 directories. Independent restoration rehearsal
into a private namespace verified all 42 hashes, UID/GID and original file modes.
Vendor credentials/configuration and unrelated skills were excluded.

[Pair restoration](../../commands.md#cli-output-and-help) requires stopping use,
preserving any upgraded root and restoring complete prior roots/receipts at
original paths with their matching binary, then checking status/digests.
This rehearsal is backup preparation, not an executed live rollback or a new
rollback guarantee. [Installer documentation](../../operational/installation.md)
explicitly provides no automatic rollback/downgrade or cross-root transaction.
Any later mutation still requires a supported complete-pair restoration plan.

### E07 — Official candidate verification and refusal

Inspected current [installer](../../../scripts/install-release.sh),
[source installer](../../../scripts/install-axiom.sh),
[upgrade guard](../../../internal/install/upgrade.go) and installation/ownership
references. Source install cannot preserve the release-owned complete docs pair.
Skill-only install/first-run from stable cannot supply PR content; PR-only skills
would mix baselines. A larger fabricated version or receipt edit is forbidden.
No supported route to align this exact PR pair on the current host was found.

Built the official generation-2 development archive from clean input HEAD,
using unchanged source manifest version 0.21.1, release=false, macos/darwin/arm64:

```sh
./scripts/build-release-archives.sh --version 0.21.1 --format 2 --development --only macos-arm64 --output <private-artifact-root>
./scripts/install-release.sh --verify-only --helper <verified-helper> --helper-sha256 <actual-helper-sha256> --archive <archive> --checksums <SHA256SUMS> --bin-dir <existing-bin-root> --receipt-dir <existing-receipt-root>
<verified-candidate> --json upgrade --archive <archive> --checksums <SHA256SUMS> --bin-dir <existing-bin-root> --receipt-dir <existing-receipt-root>
```

Build, pinned strict verifier and four archived/source skill comparisons PASS.
SourceCommit is the exact input HEAD; archive SHA-256
`e1f6f404fcb577653dba5004e683c1bd6d3dbfe02d84e227a6e2eeaa303cd9a7`;
candidate binary `b28f56501ee8b0338e8a37ad00dcee4115b21254411c304c92e863708e63b13d`;
verifier `08d184851d9b1a126bdb0480f16f2429dc814629877eb29e6875d45d10175bc0`.
The trusted local helper was built from the reviewed checkout, not trusted
merely through an adjacent checksum. No release artifact was published.

Actual read-only preview exits **1**, result **validation_failure**:
**Upgrade blocked before any effect: downgrade_refused**; next:
**Downgrade and divergent same-version replacement are not supported**.
Returned preview has no effects/digest; no apply attempted. Its error-envelope
provenance says development/unavailable/unknown, so it is not candidate build
qualification; archive metadata supplies the candidate identity above.
After refusal, all 42 managed files and 18 directories retain original hashes,
UID/GID, modes and exact inventory; no partial publication or marker.
Rollback **NOT_NEEDED_NO_MANAGED_CHANGE**. Stable host preserved.

| Native record | Installed / loaded / authentication | Actual observations |
| --- | --- | --- |
| E07-Codex attempt-07 | stable 0.21.2, three PR mismatches / UNKNOWN / NOT_REVALIDATED; attempt-06 subscription_observed only | NOT_RUN; no prompts, responses, inference or rendering |
| E07-Claude attempt-07 | stable 0.21.2, three PR mismatches / UNKNOWN / NOT_REVALIDATED; attempt-06 subscription_observed only | NOT_RUN; no prompts, responses, inference or rendering |

For EACH Runtime, C01, C02, C03, C04, C05, C06, C07, C08, C09, C10, C11, C12,
C13, C14, C15 and C16 are individually **NOT_RUN**. No synthetic transcript
substitutes for either conversation. No auth process was invoked after refusal;
no credential value viewed or persistent auth change made. Before later dispatch,
revalidate subscription login using only documented child-process omission of
OPENAI_API_KEY and CLAUDE_CODE_OAUTH_TOKEN if needed. Login is not inference.

### E08 — Reconciliation, technical decisions and blocked handoff

| AC / cases | Historical E01–E06 | Attempt-07 native | Aggregate |
| --- | --- | --- | --- |
| AC1 / C01–C06,C16 | PASS scoped | Codex/Claude NOT_RUN | PARTIAL |
| AC2 / C07,C08,C15,C16 | PASS scoped | Codex/Claude NOT_RUN | PARTIAL |
| AC3 / C09,C10,C13,C14 | PASS scoped | Codex/Claude NOT_RUN | PARTIAL |
| AC4 / C11–C13 | PASS scoped | Codex/Claude NOT_RUN | PARTIAL |
| AC5 / C01–C16 | synthetic regressions PASS | both conversations NOT_RUN | PARTIAL |
| AC6 / unchanged contracts | PASS historical | two-document scoped review | PASS_SCOPED_REVIEW |

T01–T06/E01–E06 historical scoped PASS preserved; T07/E07
**BLOCKED_HOST_BASELINE / TECHNICALLY_BLOCKED**; T08/E08 **PARTIAL**.
Outcome **BLOCKED**, because the indispensable supported host route is absent.
Concrete excluded actions remain AUTHORIZATION_REQUIRED and outside this grant:
forcing downgrade/version guard, fabricating version, editing receipts or mixing
stable binary/docs with PR skills. Additional permission alone cannot make an
unsupported route supported. No material architecture/product/design decision
was created; SDD remains closed at its approved revisions.

| Actual local validation | Result |
| --- | --- |
| `go test ./internal/cli -run 'Test(OutputSeparation\|WorkflowShowSynthetic\|Presentation)' -count=1` | PASS |
| `go test ./internal/codexruntime -run 'Test(RenderedWorkflow\|OperatorConversational\|WorkflowSkill)' -count=1` | PASS |
| `go test ./internal/install -run 'Test(UpgradeRefusalsHaveZeroEffects\|UpgradeSkillStaleAuthorityAndConcurrentInstallHaveZeroEffects\|UpgradeInterruptionAtEachOrderedEffectResumes\|SemverPrecedence)$' -count=1` | PASS |
| `./scripts/validate-repository.sh .` | PASS; Runtime behavioral scenarios SKIP/UNVERIFIED |
| Local Markdown paths and historical preservation assertions | PASS; 57 paths; attempts 01–06 and historical Tasks r1 byte-preserved |
| `git diff --check`; complete two-document scoped diff review | PASS |
| `./scripts/check-sensitive-files.sh --staged .` | PASS |
| `gitleaks git --staged --redact --no-banner` | PASS; no leaks |

Full suite not repeated for documentation-only changes. Final staged security
checks repeated after reconciliation. Input-head CI and Delivery metadata
SUCCESS; final pushed HEAD CI is recorded separately in the completion callback.
Runtime validator SKIP/UNVERIFIED remains distinct from native E07.

technical_decisions[] (TECHNICAL_LOCAL; executor-ai; scope_impact:none):

- IMPL7-TD-01: use Python 3 stdlib for private 42-file backup and isolated restore
  rehearsal, rather than a skill-only backup; includes docs/binding and receipts.
  Verified every file hash/owner/mode; no live restoration or credentials copied.
- IMPL7-TD-02: test the actual official 0.21.1 preview, then stop dependent work;
  alternatives (skill-only replacement, version inflation, source installation)
  violate complete-pair/ownership/version constraints. Official downgrade refusal
  and unchanged managed inventory verify this choice without scope expansion.
- IMPL7-TD-03: execute directly with deterministic tools; a strictly lower model
  capability hierarchy is unverified. No delegation or independent Review claimed.

**DF-01 DEFERRED_NOT_IMPLEMENTED:** stale predecessor Specifications index
summaries remain untouched; highlight at pre-merge. Existing [#346](https://github.com/rgomids/axiom/issues/346)
remains separate and NOT_IMPLEMENTED here. No optional implementation added.

remaining_work[]: establish supported authorized owned installation/restoration
for the complete PR binary/docs/four-skill pair on this host; qualify installed
and fresh loaded revisions; revalidate exact auth; run one native conversation
per Runtime with C01–C16; finish T07/T08/AC1–AC5, final-head CI and independent
Review when ready. Next gate **SUPPORTED_OWNED_PR_PAIR_REQUIRED**.
One new same-AXM attempt-07 stage.blocked callback supplies final pushed SHA,
actual CI, tests and ledger; no continuation watcher verified.
Handoff **AWAITING_EXECUTION_REPORT**. No merge, release, acceptance, close or Done.

## Attempt-08 — Qualified local candidate and native authentication failure

2026-10-24; **PARTIAL**, same open Draft PR #343 and branch. Initial local,
remote and PR HEAD `e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6`, clean tree.
All 51 Linear comments read, hasNextPage=false; no attempt-08 completion found.
[Current continuation/authority report](https://linear.app/rgomids/issue/AXM-39#comment-26e395da)
and [dispatch receipt](https://linear.app/rgomids/issue/AXM-39#comment-58b99763)
match this task. Delegated authorization is Maestro's reported Rafael grant
(2026-10-24 approximately 20:13 BRT), not independently authenticated human-session
proof. Applied only to supported local candidate qualification, complete-pair
installation and two native calls. No writer observed on scoped files or skill
locks; no concurrent source change observed. Strictly lower delegation capability
unverified: executor worked directly, no subagents or independent Review claimed.
Linear agent-context tool was advertised but returned tool-not-found; issue and
complete comment read-back succeeded. No private provider transcript imported.
Spec 012 r1, Plan r1, historical Tasks r1, AC1–AC6 and C01–C16 unchanged.
Four pillars: accepted scope, canonical product contracts and architecture/security
preserved; DESIGN NOT_APPLICABLE, no new visual surface.

### E07 — Version semantics, official candidate and exact provenance

Source release ledger remains 0.21.1, last changed by release commit
`921507c27c5b1ec5468508a6d48ce0e080e90a42`; installed stable source was
`c52f1ff78c915f00f7b9c4fea1e399bf1ee87b5e` / 0.21.2. Merge-base is the former.
Read the actual host-release range: release-delivery scripts/workflows differ,
but `internal/install` and `scripts/build-release-archives.sh` are byte-identical
between host source and this PR. No host release fixes silently imported.

Candidate version **0.21.3-axm39.8** is a private, unpublished next-patch candidate:
0.21.3 follows the current stable 0.21.2 patch under [existing version rules](../../../CONTRIBUTING.md#commit-messages);
`axm39.8` distinguishes this Work Item/attempt. Changed backward-compatible
presentation guidance supplies candidate content; exact source commit identifies
it independently of version ordering. This is not a recorded stable release or
claim that this source contains all later release-delivery changes. The [official
builder](../../../scripts/build-release-archives.sh) accepts the candidate through
its existing `--version` parameter and atomically propagates it to binary build
identity, archive name, docs descriptor/provenance and release metadata/checksums.
No hand-edited artifact, manifest, receipt, release ledger or changelog heading.
Changing `.release-please-manifest.json` would misstate release history; not done.
Numeric alignment alone was never used as compatibility proof.

First built official generation-2 archive with `--development`. Four skill
comparisons and temporary docs installation passed, but actual `--json version`
reported **development / unavailable / unknown**. [Provenance](../../../internal/provenance/provenance.go)
requires literal `development` for non-release informational identity; a SemVer
injected with release=false falls back to unknown provenance. [ADR-0016](../../decisions/0016-candidate-derived-skill-set-receipt.md)
and `Build.isCandidate` also reject receipt derivation from release=false.
This candidate was **DISQUALIFIED**, never applied to the real host. Archive
`9f855b95ce65753a32a78adaaf74666991b8477f2f968fb8b7c0c5454131b18a`;
binary `b261d0b0633ced1972df41122c064d657999181a36de7c01750a0154a2503384`.
Temporary docs success did not override that failure.

Resolved locally through existing clean-candidate build mode, omitting
`--development`, without changing code or provenance rules:

```sh
./scripts/build-release-archives.sh --version 0.21.3-axm39.8 --format 2 --only macos-arm64 --output <private-candidate-root>
./scripts/test-docs-offline.sh --archive <candidate-archive> --checksums <SHA256SUMS>
./scripts/install-release.sh --verify-only --helper <independently-built-helper> --helper-sha256 <actual-pin> --archive <candidate-archive> --checksums <SHA256SUMS> --bin-dir <owned-bin-root> --receipt-dir <owned-receipt-root>
```

All exit 0. This mode sets `release=true` for qualified clean SemVer identity;
it only builds local files, creates no tag/GitHub Release and makes no publication
or acceptance claim. No release workflow dispatched. Clean exact source
`e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6`, revision `e045cac1e3fd`, generation 2,
macos/darwin/arm64. Four archived skills exactly equal PR source/embedded content;
path/type inventory checked before extraction, strict pinned verifier PASS.

| Qualified artifact identity | SHA-256 |
| --- | --- |
| archive | `24b70b602db63bc55d06b4a8b9617ab487d4abcef120645e4e558c0a5cf8caf3` |
| binary | `d6582d9851e7a138b8b9a91085bfede35fe20f068ee84e5fbc1b78b346dc8683` |
| separately built verifier | `16298cfb6b18186711086506996fad5589681a41555159ed166c6bda313ac580` |
| docs inventory | `9982146a5e9c6793023c3178109188ed6bf0115c7c22d4e2f180f733c728e3ee` |
| docs provenance | `9f341d988b48fa9c651e963f89f5895acca6e0234492080d53ded5970ae87eab` |
| installed current four-skill manifest | `3471519797b11ddef021d1dea536538137f8747d9faaf2f96c110cf7d86fab50` |

The archive source is the input PR commit above. Later attempt-08 changes contain
Evidence/checkpoint documents only; they are not represented as a newly installed
binary source SHA. Final pushed Evidence HEAD and CI are recorded in the callback.
No historical PASS is reassigned to newly changed bytes.

### E07 — New backup, supported preview/apply and both-root read-back

Before apply, fresh actual host read-back again confirmed 0.21.2 / c52f1ff78c91 /
clean, docs verified/ok, valid_v1, current receipts. Read every installed skill:
Project/Workflow/Runtime mismatched PR in BOTH roots, Work Item matched.
A NEW private backup records **42 files and 22 directories**, SHA-256, UID/GID,
modes and exact docs inventory: binary, installation receipt, both skill receipts,
eight skills, complete active docs and binding. Extra directory scope includes
docs structural directories; count differs from attempt-07, not a rewritten
historical inventory. Backup objects 0600/private directories 0700; no credential
or unrelated skill content copied. Manifest SHA-256
`3c1cb1dd99b191e30cdc43d6eb26018efca467997a9f0788e21c184d3d0af57a`.
Isolated restoration rehearsal re-read ALL file hashes/UID/GID/modes and directory
UID/GID/modes: PASS. Fresh live comparison immediately before apply: PASS.
Backup retained privately; no live restoration or automatic rollback claimed.
Recovery plan remains the documented complete-pair preservation/restoration;
no force replacement, downgrade or receipt reconstruction authorized/performed.

Strict verify-only PASS and actual read-only candidate `upgrade` preview exit 0:
source 0.21.2 → target 0.21.3-axm39.8; direct/compatible, no persisted state to
transition, no leftovers. Preview classified persisted state absent_v1; earlier
installed compatibility inspection valid_v1 includes current skills, not Project
records. Digest:
`724d58bc8d771e58b3b5ed83f32ab1680f2a8f28ed9d1e6686b55eedd49b6591`.
Eight reviewed effects: docs generation/binding, binary, installation receipt,
three Codex skill updates, Codex skill receipt. Applied through the verified
candidate's supported `upgrade`, exact digest plus `--authorize-local`; exit 0,
all eight ledger effects confirmed, skills matches, skillReceipt current.
Then supported `axiom --json runtime claude install`, exit 0, converged Claude's
recognized owned skills/receipt. No cross-root atomicity claimed.

Fresh installed `version`, `docs inspect`, both Runtime `status` exit 0:
0.21.3-axm39.8 / e045cac1e3fd / clean; docs verified/ok/compatible; skill-set and
binaryCompatibility 3; receipts current. All EIGHT disk skills re-read and exactly
match PR source. Expected digests: Project `775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d`,
Work Item `ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3`,
Workflow `efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76`,
Runtime `165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d`.
Of original backup scope, exactly 10 expected files changed; 32 remained exact,
including all old docs/binding and both Work Item files. All original file
UID/GID/modes preserved. Negative tests cover foreign/unrecognized ownership;
no blanket claim that every unrelated user file was independently inventoried.
Rollback **NOT_NEEDED_INSTALL_SUCCEEDED**, live restore NOT_RUN. Host intentionally
retains qualified unpublished candidate, not stable 0.21.2.

### E07 — Actual native conversations, loading and authentication

[Sanitized exact prompt/replay input and observed excerpts](evidence-native-attempt08.md)
record scope and gaps. Executions used real installed subscription CLIs outside
maintainer checkout, without model override/delegation/credential edits. Omitted
only existing `OPENAI_API_KEY` and `CLAUDE_CODE_OAUTH_TOKEN` in child environments;
no value read/persisted. Immediately before invocation, both official
`runtime <runtime> auth` diagnostics exit 0, subscription_observed, local_observation,
usability unproven_until_real_dispatch. Codex method chatgpt, CLI 0.163.0; Claude
method claude.ai, CLI 2.1.310. This is NOT inference proof.

| Native record | Installed / loading / actual authentication and conversation |
| --- | --- |
| Codex | Exact pair above; actual four-skill file-read and SHA256 tool evidence identifies observed revision. Grouped reads truncated; individual Project/Work Item/Workflow reads repeated; complete Runtime context retention not independently attested. Native process exit 0, completed model response: subscription inference observed. Specific model/effort UNKNOWN, not present in consulted event metadata. |
| Claude | Exact pair above; loaded UNKNOWN. Real invocation exit 1: `Failed to authenticate: OAuth session expired and could not be refreshed`. Init names `claude-sonnet-5-5`, apiKeySource none; no successful inference or skill tool call. Subscription observation did not establish usability. |

Codex actual read-only Project list: success, `projects: []`. No configured
Project exists for live show/validate; none created/selected. Public Workflow r2
fixture and synthetic C01–C16 replay were supplied to this SAME real conversation;
replay is not fabricated model output and not live configured Project/Execution
Evidence. Historical fixture provenance preserved and explicitly distinguished.
One Codex and one Claude invocation only; no auth workaround, alternate key,
login/logout/refresh or retry after Claude failure. No Stage dispatch, external
Provider effect, visual rendering, AXM-12 E2E or human acceptance claimed.

| Case | Codex real conversation over labelled replay | Claude |
| --- | --- | --- |
| C01 | PASS_SCOPED_REPLAY; name+slug and structural-valid/readiness split | NOT_RUN |
| C02 | PASS_SCOPED_REPLAY; name-only/slug-only preserved | NOT_RUN |
| C03 | PASS_SCOPED_REPLAY; absent labels/full ID reason | NOT_RUN |
| C04 | PASS_SCOPED_REPLAY; both full IDs/collision/authority preserved | NOT_RUN |
| C05 | PASS_SCOPED_REPLAY; exact ID, command and full digest | NOT_RUN |
| C06 | PASS_SCOPED_REPLAY; partial inventory and hostile label inert | NOT_RUN |
| C07 | PASS_SCOPED_REPLAY; both summaries and advanced Agent details | NOT_RUN |
| C08 | PARTIAL; blockers promoted, concrete recovery next step missing | NOT_RUN |
| C09 | PASS_SCOPED_REPLAY; allowed/declared/unresolved/model unknown | NOT_RUN |
| C10 | PASS_SCOPED_REPLAY; resolved plan/login/no attempt separated | NOT_RUN |
| C11 | PASS_SCOPED_REPLAY; success/blocked/diagnostics and next inspection | NOT_RUN |
| C12 | PASS_SCOPED_REPLAY; exit-zero/invalid/effects unknown/reconcile | NOT_RUN |
| C13 | PASS_SCOPED_REPLAY; stale/truncated/inaccessible facts retained | NOT_RUN |
| C14 | PASS_SCOPED_REPLAY; unknown version alone no certification gate | NOT_RUN |
| C15 | PASS_SCOPED_REPLAY; sequential/DAG fallback, no rendering claim | NOT_RUN |
| C16 | PARTIAL; same-answer comparison, no separate explicit-entry turn/surface | NOT_RUN |

C08/C16 gaps remain baseline validation work, not optional deferred requirements.
Codex native real configured show/validate and actual graphical/narrow surface
behavior NOT_RUN. Installed inspector's loaded unknown is not overwritten or
forged by this separate session observation. Model-context completeness and exact
Codex model remain limitations, despite observed read-command/digest evidence.

### E08 — Actual validation, technical decisions and partial handoff

| Actual command/check | Result |
| --- | --- |
| `go test ./internal/cli -run 'Test(OutputSeparation\|WorkflowShowSynthetic\|Presentation)' -count=1` | PASS |
| `go test ./internal/codexruntime -run 'Test(RenderedWorkflow\|OperatorConversational\|WorkflowSkill)' -count=1` | PASS |
| Focused `internal/install` downgrade/conflict/stale-authority/interruption/SemVer/foreign-generation tests | PASS |
| Focused candidate receipt/embedded-identity `internal/install` tests | PASS |
| `go test ./internal/provenance ./internal/install -count=1` | PASS; complete two packages |
| Official generation-2 build, strict verification, four-skill archive equality | PASS; development candidate separately disqualified as above |
| `./scripts/test-docs-offline.sh --archive <archive> --checksums <SHA256SUMS>` | PASS for BOTH candidates; temporary-root docs/refusal only, native/network/source denial NOT_RUN |
| `./scripts/validate-repository.sh .` | PASS before Evidence edits; final document checks below/callback; behavioral Runtime scenarios SKIP/UNVERIFIED |

| AC | Current scoped evidence | Aggregate |
| --- | --- | --- |
| AC1 | Historical E01–E06 plus actual Codex replay C01–C06/C16; empty live list; Claude gap | PARTIAL |
| AC2 | Codex replay summary/detail and fallback; C08/C16 gaps; Claude absent | PARTIAL |
| AC3 | Codex provenance/Runtime replay; no Claude response | PARTIAL |
| AC4 | Codex blocked/effects-unknown replay; C08 next-step gap; Claude absent | PARTIAL |
| AC5 | Local regressions PASS; one Codex completed response, Claude auth failure | PARTIAL |
| AC6 | Evidence-only diff; production contracts/history unchanged | PASS_SCOPED_REVIEW |

T01–T06 historical scoped PASS remain historical; current focused/package tests
above are fresh. T07 **PARTIAL / BLOCKED_CLAUDE_AUTH**, T08 **PARTIAL**. Completion
requires remaining native coverage and review; no request for independent Review
while full readiness is absent. Exact pushed-head CI is read after push, never
inferred from input-head SUCCESS. Input CI run 40139652086 and Delivery metadata
40139652122/40139688503 were SUCCESS at e045cac.

technical_decisions[] (TECHNICAL_LOCAL; executor-ai):

- IMPL8-TD-01: assign private next-patch prerelease `0.21.3-axm39.8` through the
  existing official builder's atomic version parameter; keep stable release ledger
  untouched. Sources: builder, CONTRIBUTING version rules, actual archive metadata.
  Same 0.21.2/+metadata would be divergent-equivalent; 0.21.2 prerelease would be
  downgrade. Version is provenance for changed candidate bytes, not a compatibility
  promise or public release. Exact source, embedded skills and protected preview
  independently qualify it.
- IMPL8-TD-02: reject release=false candidate after actual unknown-provenance
  observation; use existing clean candidate mode per ADR-0016. No provenance or
  receipt bypass, code change, published release or new compatibility contract.
- IMPL8-TD-03: complete new 42-file/22-directory private backup and isolated
  restoration rehearsal, then exact-digest supported upgrade and owned Claude
  install. Retain old docs and matching backup; no automatic rollback claim.
- IMPL8-TD-04: empty real Project inventory means labelled public replay, not
  Project creation or fabricated live facts; record C08/C16 and loading/model gaps.
  Native Codex output is actual inference, distinct from synthetic input.
- IMPL8-TD-05: stop Claude after one actual OAuth failure despite local login
  observation; no credential mutation/alternate auth/retry. This is an operational
  authentication gate, not a material SDD/architecture decision or tooling question.

DF-01 **DEFERRED_NOT_IMPLEMENTED**, stale predecessor Specifications index,
remains separate; existing [#346](https://github.com/rgomids/axiom/issues/346)
unchanged/out of scope. No optional issue implemented or newly created.
remaining_work[]: restore usable existing Claude subscription through its owner;
revalidate exact child auth and run native conversation with exact installed
revision evidence; close C08/C16 and model/loading validation gaps with bounded
native coverage; reconcile T07/T08/AC1–AC5; independent Review only when ready.
No auth renewal performed. Candidate backup retained; host remains installed at
0.21.3-axm39.8. Next gate **CLAUDE_SUBSCRIPTION_USABILITY_REQUIRED**.
One same-AXM stage.partial callback records actual pushed Evidence HEAD, CI,
checks and this ledger with raw comment read-back. Handoff
**AWAITING_EXECUTION_REPORT**; no automatic continuation watcher established.
No merge, release publication, acceptance, close or Done.

Final documentary verification: complete scoped diff reviewed; 65 local Markdown
paths checked; historical attempts 01–07 are an exact byte prefix and historical
Tasks baseline byte-preserved. `git diff --check` PASS. Final repository validator,
staged sensitive-file scan and redacted Gitleaks results are read back before
commit and included in the exact pushed-head callback. Evidence-only commit does
not retag, rebuild or replace the already qualified source candidate.

## Attempt-09 — Native Claude response without qualified skill loading

2026-10-24 BRT (2026-10-25 UTC), **PARTIAL**. Local/remote/open Draft PR #343
matched expected `0825d24dc9ae056ad684641d9a695e02498d33c6`; tree clean at entry.
Same branch, no overlapping executor observed: the native Codex process in this
checkout was this executor's parent, not a second dispatch. Other Claude process
working directories were outside this checkout. This is bounded observation, not
a global lease. All 54 Linear comments read, hasNextPage=false; latest
[attempt-09 dispatch](https://linear.app/rgomids/issue/AXM-39#comment-85bdf9fa)
matched [the current prompt](https://linear.app/rgomids/issue/AXM-39#comment-77b2d1f5),
with no attempt-09 completion/competing dispatch before writes. Remote HEAD and
draft status re-read immediately before documentary edits.

Authority remains Maestro's report of Rafael's approximately 20:59 BRT grant,
not independently authenticated human-session consent. Scope: existing-subscription
Claude conversation, bounded Codex follow-up, T07/T08 Evidence and same-branch
commit/push/callback. No new package/version/install, credentials, Project effects,
Stage dispatch, merge/release/self-acceptance/Done. Linear field-note API returned
tool-not-found; issue/comments remained available. Direct execution, no subagents:
strictly lower capability hierarchy unverified and no useful independent writer.
Four pillars unchanged: accepted AC/non-goals, product contracts, architecture
and security; DESIGN NOT_APPLICABLE, no new product visual surface.

### E07 — Fresh installed and authentication preflight

Read-only version/docs/both Runtime status exit 0: **0.21.3-axm39.8 / e045cac1e3fd /
clean**, verified/ok/compatible docs, skillSetVersion/binaryCompatibility 3, both
receipts current. SourceCommit `e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6` and binary
SHA-256 `d6582d9851e7a138b8b9a91085bfede35fe20f068ee84e5fbc1b78b346dc8683`
unchanged. All eight files independently hashed/compared with source, exact match
to attempt-08 inventory. Re-read after conversations: still exact in both roots.
No assumption that later Evidence-only HEAD is installed binary source.

Exact child auth diagnostics exit 0, subscription_observed/local_observation,
Claude method claude.ai and Codex chatgpt. Existing subscription login only;
known API-key/token/provider overrides omitted in children without reading values.
No login/logout/refresh, key export, alternate account or auth repair by executor.
Successful actual Claude responses now prove usability for this invocation;
attempt-08's expired-session failure remains immutable historical Evidence.

### E07 — Conversations, loading, cases and failed prerequisites

[Exact public inputs, invocation scope, selected actual excerpts and per-case ledger](evidence-native-attempt09.md)
are authoritative for this attempt's native observations.

| Record | Actual result and limit |
| --- | --- |
| E07-Claude | One native CLI process, two interactive stream-JSON user turns, exit 0. Init `claude-opus-5-5`, apiKeySource none; effort UNKNOWN. Four installed skill Read calls and six Bash calls denied before execution. Only public local cases/Workflow reads succeeded. Loaded UNKNOWN; installed skill entries absent in this invocation, explicit skill-entry subcase NOT_RUN. Prompt-only replay did not qualify T07. |
| E07-Codex | Bounded follow-up, first natural turn then separate explicit `$axiom-project $axiom-workflow` turn on exact same thread through exec resume; both exits 0. Four individual complete tool-output contents exactly equal installed text; hashes identify revision. C08 concrete safe recovery now PASS_SCOPED_REPLAY. C16 textual semantic comparison observed, aggregate PARTIAL; no narrow GUI or independent harness re-expansion Evidence. Model/effort UNKNOWN. |

Claude's executor-selected isolation omitted skill discovery/authorized outside-cwd
read access; exact Bash allow rules also did not cover appended exit-reporting
suffixes. These are invocation limitations, not proven defects in installed
guidance. Native responses explicitly acknowledged absent skill loading. The
executor requested replay in the same initial prompt instead of gating it on
successful native loading; its subsequent prompt-only replay cannot repair that
failed prerequisite. No second Claude conversation or permission bypass attempted.

Claude replay ledger: C01/C03/C04/C06/C09/C10/C12/C13/C15 PASS_SCOPED_REPLAY
**prompt-only**, C02/C05/C08/C14 FAIL, C07/C11/C16 PARTIAL; all share missing
installed-loading prerequisite. C05 proposed unsupported command syntax, C08
abbreviated recovery reference, C14 made unknown version a gate. C16 ordinary
phrasing comparison was not real skill entry. No fixture-derived command executed.
No product-code/guidance change is justified from this unloaded session alone.

Codex C02–C06/C09–C15 retain attempt-08 passes only; C01/C07/C11 bounded comparison
also passed. C08 identifies Workflow owner inspection of exact validator/reference
and Work Item owner inspection of denied envelope before any proposed repair;
both blockers remain immediate, future change needs separate scope/preview/authority.
C16 separate actual explicit text turn preserved Unicode, full ID/digests, gate,
source/status, both blockers, confirmed no-dispatch and read-only next action.
It does not certify narrow/graphical behavior or automatic harness expansion.

No live configured Project show/validate performed. Codex did not repeat live
list; Claude's list request was denied. Attempt-08 empty list is historical only.
No real Workflow/Execution, graphical renderer, Stage dispatch or AXM-12 E2E
acceptance. Full Codex Runtime file output now observed untruncated; permanent
complete model-context retention remains unprovable. Inspector loaded unknown
is retained, not overwritten by either session or an invented receipt.

### E08 — Reconciliation, verification and next gate

| AC | Current Evidence | Aggregate |
| --- | --- | --- |
| AC1 | Historical identity regressions/Codex replay; explicit text comparison; Claude loading/C02/C05/C16 gaps | PARTIAL |
| AC2 | Codex summaries/details/C08 safe recovery/fallback; C16 surface and Claude C07/C08/loading gaps | PARTIAL |
| AC3 | Codex historical provenance cases; Claude unloaded C14 fails version-only rule | PARTIAL |
| AC4 | Codex safe C08/C11 next action; Claude C11/loading gaps; no live dispatch | PARTIAL |
| AC5 | Historical deterministic regressions retained; both native responses observed, but qualified Claude skill behavior and C16 coverage incomplete | PARTIAL |
| AC6 | Only Spec 012 Evidence/Tasks checkpoint changed; no canonical production/history semantics changed | PASS_SCOPED_REVIEW |

T01–T06 historical scoped PASS preserved. T07 **PARTIAL /
NATIVE_CLAUDE_LOADING_ENTRY_REQUIRED**; T08 **PARTIAL**. No request for independent
Review as a replacement for absent native Evidence. Input-head CI read back
SUCCESS: [PR Checks](https://github.com/rgomids/axiom/actions/runs/40142768287),
[Delivery metadata](https://github.com/rgomids/axiom/actions/runs/40142768292).
Those checks certify only input HEAD; exact pushed-head CI is queried separately
and included in the single completion callback. Fresh focused regressions:

| Actual command/check | Result |
| --- | --- |
| `go test ./internal/cli -run 'Test(OutputSeparation\|WorkflowShowSynthetic\|Presentation)' -count=1` | PASS, 0.764s |
| `go test ./internal/codexruntime -run 'Test(RenderedWorkflow\|OperatorConversational\|WorkflowSkill)' -count=1` | PASS, 1.317s |
| Local Markdown paths | PASS, 72 paths across three checkpoint documents |
| Historical preservation | PASS; complete attempts 01–08 Evidence prefix, Tasks r1 baseline, Spec/Plan/examples and attempt-08 native Evidence byte-exact |
| `git diff --check`, scoped content/diff review | PASS; only three Spec 012 documents |

Repository validator and staged sensitive-file/redacted secret scans are read back
before commit and recorded in the callback. Full Go package suite not repeated:
no production guidance, test, dependency or package bytes changed. Runtime validator
behavioral scenarios remain distinct from these native presentation observations.

technical_decisions[]: TECHNICAL_LOCAL, owner/decision_maker executor-ai,
scope_impact none; details and source observations linked above.

- IMPL9-TD-01: bounded read-only child environments and one two-turn Claude
  stream, no nested Runtime/delegation/external tools; preserve actual allow-rule
  failures rather than bypass or silently replace the conversation. Invocation
  settings failed loading/entry; gate future replay on actual native read success.
- IMPL9-TD-02: reuse exact public fixture/cases and historical Codex passes;
  follow up C08 plus C01/C07/C11 for C16, individually verify four full skill tool
  outputs against installed text. No grouped-read/context-completeness inference.
- IMPL9-TD-03: separate successful Claude subscription inference from failed
  skill qualification; retain replay failures and C16 PARTIAL. Do not infer an
  implementation defect, demand new credential renewal or change production
  guidance based on an unloaded response. Runtime flags are local technical
  choices, not a material HUMAN_DECISION_REQUIRED design gate.

**DF-01 DEFERRED_NOT_IMPLEMENTED**: stale predecessor Specifications index remains
outside scope, highlight at pre-merge. Existing [#346](https://github.com/rgomids/axiom/issues/346)
remains separate and NOT_IMPLEMENTED here. No new card, finding implementation,
ADR, release/tag, package version, reinstall, approval, merge or Done.

remaining_work[]: prepare correctly scoped native Claude skill loading/discovery
and prove entry/read revision **before** replay under a bounded conversation grant;
re-evaluate C01–C16 against installed guidance, retaining these observed failures;
preserve/resolve C16 narrow/graphical/harness-observation gaps without invented
support; reconcile T07/T08/AC and exact CI, then independent Review when ready.
Next gate **RESUME_IMPLEMENT / NATIVE_CLAUDE_LOADING_ENTRY_REQUIRED**.
No renewed authentication gate claimed. One deduplicated attempt-09 stage.partial
root callback supplies actual final SHA, tests/CI and ledger with raw read-back.
No authorized automatic continuation callback demonstrated beyond that receipt:
handoff **AWAITING_EXECUTION_REPORT**; no background execution claim.
