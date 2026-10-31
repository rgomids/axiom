# S1-05 — canonical workflow implementation checkpoint (PARTIAL)

[AXM-42](https://linear.app/rgomids/issue/AXM-42),
[human S1 authority](https://linear.app/rgomids/issue/AXM-42#comment-32f02fa5),
[S1-04 receipt](https://linear.app/rgomids/issue/AXM-42#comment-3dbdf961),
[existing PR #366](https://github.com/rgomids/axiom/pull/366).
This attempt does not complete S1 or advance to engineering/security review.
Next gate: `resume_implement`. Maestro publishes and reads back the receipt;
executor has no Linear access and made no callback/dispatch.

Source implementation commit: `989da5f192618d7dbc7eb393abfc20fe0fed4c6d`;
manifest-pin correction: `efff9decaafa80cda63d76e0dc61693f7fa0e0ef`.

## Source and authority

Verified physical worktree `/home/grokbot/axiom-wt/axm-42`, registered worktree,
clean status, branch `spec/axm-42-minimum-viable-workflow`, origin
`https://github.com/rgomids/axiom.git`, local/remote/PR HEAD all
`1ea08c8447e2692ddc9cb3a1f1d4695af93c83bf`, and ancestry before edits.
No clone, switch, reset, clean, stash, rebase or force push.
Spec 015 correction-04, Plan/Tasks, Research A–L, ADR-0037 and its precise
accepted partial supersessions control. Historical investigation-only wording
is not treated as revoking the supplied later S1 authority. S2/S3, real cutover,
live vendor tests and Product Accepted remain outside authority.

The final receipt records the full final SHA (an Evidence commit cannot embed
its own hash). The final source validation uses initial HEAD plus source patch
SHA256 `4323d87e6ecb87d6b330810d8e9d40e1d4639c343029e3e0fd495d22cb3f59de`
(`git diff --binary 1ea08c8447e2692ddc9cb3a1f1d4695af93c83bf efff9decaafa80cda63d76e0dc61693f7fa0e0ef -- internal cmd`). Final full validation was restarted after the authoring diagnostic and
manifest-pin corrections;
a further focused per-agent timeout test was added and run separately.
Documentation and measurement are reconciled afterward. CI is reported separately for the exact pushed HEAD.

## S1.1–S1.8 matrix

| Task | Result in this attempt | Remaining acceptance |
| --- | --- | --- |
| S1.1 | Preserved accepted ADR-0037 and historical supersession annotations | No new material decision or acceptance claimed |
| S1.2 | Extended the same format-4 snapshot with configured agent controls and before-gate timing; strict DAG/bounds validation before status derivation | Remaining legacy owner removal is S1.7, not a second supported engine |
| S1.3 | Public explicit machine-local defaults, optional Profiles and bare Runtime bindings; immutable ordinary retry and safe provenance; no candidate scoring/auth preflight | Full A–H/K consumer retirement remains open |
| S1.4 | Same fenced engine preserved; per-agent timeout and configured retry ceiling enforced; no change to process ownership/unknown-effect rules | S1-wide certification depends on structural retirement and remaining controls |
| S1.5 | Public two-stage stub progression with no Profile catalogue; before/after gates, dependency readiness, assigned required outputs, attempts/timeouts retained | Exact configured validators and authorized command checks still refused before creation; input/context lowering and all schema-2 bounds need complete reconciliation |
| S1.6 | Existing private attempt ingress fixtures preserved | No S2 parallel transport, routing or integration claim |
| S1.7 | Producers/consumers measured explicitly; no new legacy engine introduced | NOT COMPLETE: lifecycle commands/stores/codecs, Graph/Single readers/writers, predictive policy/auth consumers still present |
| S1.8 | Additional isolated active/unknown disposition and unexpected-file negatives; installed operator stop-or-block guidance reconciled | Cannot certify cutover complete until S1.7 and remaining control mapping finish; actual state operations NOT_RUN |

## Changed boundaries and focused proof

`LINGO_INVOCATION_DEFAULTS` is explicitly configured machine-local input: a strict
JSON array of at most 16 `field=value` strings and 64 KiB, using runtime, model,
effort, instructions and optional profile. No ranked candidate or new durable
policy store. Per-agent run > Workflow agent > local defaults; Profile values
supply defaults at their own level. Every level is checked for conflicting
assignments even when shadowed. Failure responses never copy raw JSON or values.
The instructions supplied by a Workflow remain above local instructions.

Format-4 Runtime configuration now permits zero named Profiles and empty Profile
allowlists. `runtime binding create --runtime codex` can preview/apply only the
binding; Profile-specific fields without a Profile refuse. Historical formats
keep their prior validation rules. Dangling Profile references still refuse;
selected Profiles remain subject to explicit allowlists. No generated placeholder
Profile, capability/complexity catalogue or vendor support claim.

Ordinary public retry resolves the frozen tuple against current Runtime binding,
repository and Workflow restrictions, without re-reading changed local default
selectors. It preserves the original instructions, Profile digest and provenance;
only explicit new selection permits another tuple, as another attempt.

Configured output-only stages retain before/after human-gate timing, per-agent
DAG, assigned required outputs, timeouts and maximum attempts. Cyclic/dangling
agent graphs and invalid limits refuse before any Execution write. The same
serial loop schedules ready agents; this does not implement S2 parallelism.
Canonical bounds now admit 32 agents and 64 outputs/artifacts, matching those
schema-2 dimensions. The aggregate byte bound remains 256 KiB and combined
instruction bounds remain stricter than some full definitions; complete schema
coverage is not claimed. Stage timeouts reject integer overflow before conversion.

Before-gates produce awaiting_gate with zero attempts. A real human approval is
required before spawn. After-gates still require successful outputs. Gate
receipts must encode a human actor and approved disposition; cancellation and
unknown effects cannot grant approval. Dependency-ordered status refresh avoids
relying on declaration order. Both gate timings on one canonical stage refuse,
rather than letting one approval satisfy two positions.

Configured validators remain fail-closed. The retained Workflow contract uses
`artifact-schema`, `evidence-check` and `human-review` with policy references;
the old implementation validates typed retained artifact metadata/correlation
and exact result-bound human review. Presence of an output file alone is not
an equivalent validator. No validator is skipped or substituted with a gate.
Porting those controls plus optional authorized command argv remains required.

Focused tests include `TestPublicWorkflowRunMachineLocalDefaults` (reviewed
removal of the fixture binding/Profile and creation of a bare binding, then two
stages), `TestPublicWorkflowRunInvalidLocalDefaultsHaveNoExecution`,
`TestPublicWorkflowRunDeniedLocalDefaultRuntime`,
`TestPublicWorkflowRunOptionalDefaultProfileAndDirectOverride`,
`TestPublicWorkflowRunOrdinaryRetryIgnoresChangedDefaults`,
`TestPublicWorkflowRunBeforeGateRequiresApprovalBeforeSpawn`,
`TestMapWorkflowDefinitionValidatesBeforeMapping`,
`TestExecutionBeforeGateAndAgentDependencies`,
`TestExecutionInvalidAgentControlsRefuseBeforeAttempts`,
`TestAuthoringBindingWithoutProfileCatalogue`, and
`TestV4RuntimeBindingDoesNotRequireProfileCatalogue`.
All processes here are deterministic local stubs, not vendor inference or
Provider mutations. No actual Codex/Claude CLI test invocation occurred; no
ambient model fallback or live vendor substitution.

## Structural retirement and measured survivors

[Deletion report](deletion-s1-05.json) measures the approved candidate union plus
every added internal/cmd Go file. Its supplemental expanded-union delta also
accounts for modified existing Go files outside that envelope, avoiding hiding
replacement growth. Physical lines include blanks/comments; exported symbols
are declaration counts, not semantic reachability. Numerical S3 targets are not
an S1 gate.

| Candidate + added union | Baseline | Current | Delta |
| --- | --- | --- | --- |
| Production files / lines | 88 / 21,872 | 79 / 20,284 | -9 / -1,588 |
| Production exported types / funcs | 287 / 351 | 251 / 318 | -36 / -33 |
| Test files / lines | 84 / 18,320 | 75 / 15,150 | -9 / -3,170 |
| Test exported functions | 481 | 403 | -78 |

Expanded-union delta including changed preexisting files outside the original
envelope: production -13 files/-1,997 lines/-40 types/-52 funcs;
tests -12 files/-3,500 lines/-85 funcs. This attempt adds code; these net
reductions accumulate prior S1 work and do not claim new structural deletion.

[Consumer occurrence inventory](consumers-s1-05.json) contains 253 production
lexical producer/consumer references plus 117 relevant declarations and the exact matching patterns. It is a
bounded reproducible inventory, not a zero-callers or reachability certificate.
Important remaining chains:

| Chain | Concrete surviving producer/consumer | Ownership and required replacement |
| --- | --- | --- |
| Lifecycle | `internal/cli/commands.go`, `cmd/lingo` workflow dispatch → `workflow.Service`, configured/gates → `local.WorkflowStore` Create/Save/Load/List | S1 obsolete lifecycle retirement remains open; linked Work Item resolution must survive independently |
| Delivery readiness | `cmd/lingo/delivery.go` loads Graph/Single attempts/authority/coordination before composing results | S1 must remove obsolete readers by consuming canonical results; S3 retains responsibility for authorized remote delivery/rework, not permission to delete its safety behavior early |
| Child publishing | `cmd/lingo/stage_publish.go` persistedChildResolver loads Graph/Single/Workflow stores and leases | S1 old resolver retirement remains open; S2 owns real bidirectional routing/dedupe/parallel integration |
| Shared composition | `cmd/lingo/stage_service.go` stageStores still opens Graph/Single/Workflow stores for delivery/coordination | Not a justified permanent survivor; remove after canonical consumers are ported, not by hiding constructors |
| Predictive policy/auth | `cmd/lingo/runtime_policy.go`, `runtimeapplication/policy_v4.go` and old lifecycle admission | S1 retirement remains open. Canonical run uses explicit resolver and bounded effective selector reader, not these predictions |
| Legacy persistence | Graph/Single stores, authority/dispatch-input codecs, Workflow historical readers/writers and inventory classifiers | Old operational readers/writers are not justified by historical bytes. Retire after callers are removed; preserve inert history/backup without conversion |
| Process primitives | `executiongraph.OSProcessRunner` and process correlation used by canonical engine | Justified low-level survivor: replacing it loses PID/start/group/timeout/cancel safety. Reusing this process boundary is simpler than another supervisor and creates no Graph ledger |
| Delivery owners | independent packet/publication/decision/rework state | S3 owns their replacement with Execution delivery; no new remote mutation or S3 implementation in this attempt |

No obsolete format, store family or command was newly deleted in S1-05. S1.7
remains a substantive required implementation task, not deferred optional work.

## Cutover proof and operational limits

Only isolated `t.TempDir` state was used. Existing negatives cover held leases,
interrupted publications, unknown/post-cutover namespaces, linked/unreadable
roots/files/backups, changed source bytes, forged paths and byte-verified backup.
New negatives cover active/unknown/incomplete arbitrary raw bytes requiring
human disposition even without a held lease, unexpected regular files at known
root names and an empty human identity. Inspection never decodes/resumes those
bytes. Operator disposition is required; absence of a held lease is not proof
of a stopped process. The helpers perform no stop, migration or real switch.

The installed Workflow reference now states the stop-or-block sequence, raw
backup verification, process identity/escape limits, unknown-effect disposition,
old-format refusal and rollback without replay. The command reference, installed
Runtime/Work Item skills, shared embedded skill ownership history and architecture
point to the actual partial path. Published ADR text/history is unchanged.

## Validation ledger

Final local results and corrected-source CI read-back are reconciled below.
The initial full run overlapped later source edits and is diagnostic only.
An intermediate full run exposed the existing `profile required` diagnostic
regression for incomplete optional Profile fields. Restored the exact original
field-required diagnostic through strict Profile candidate validation; the
existing test is unchanged and passes. No failed run is reported as PASS.
An all-workflow `-timeout=5m` run timed out at
`TestDeliveryReviewMissingAndWaiverAuthorityFailClosed` while validating retained
legacy schema/state; stageexecution passed. No assertion was weakened; the
required complete run retains `-timeout=30m`.
CI at `989da5f192618d7dbc7eb393abfc20fe0fed4c6d` FAILED in Tests solely
because `TestEveryPublishedSharedRevisionStaysOwned` required the actual new
embedded manifest digest `cb1a07df3b7b664f6c3b770747e834c4e839eb5da4251b1151078689661ac9b5`.
Added that digest to the existing pin list, preserving all prior pins and test
assertions; full codexruntime package passed afterward (21.091s). No fictitious
skill publication was recorded. Formatting, vet and pinned Staticcheck passed
in that source CI run before its test failure.
A first bare-binding test failed because the shared token-list validator still
required a Profile allowlist; fixed only for format 4 and rerun successfully.

Exact focused commands (the table supplies their results):

```sh
go test ./cmd/lingo ./internal/workflow ./internal/stageexecution ./internal/runtimeapplication -run 'TestPublicWorkflowRun|TestExecution|TestEngine|TestResolveInvocation' -count=1 -timeout=5m
go test ./internal/runtimeprofile ./internal/runtimeapplication ./cmd/lingo -run 'TestV4RuntimeBinding|TestAuthoringBindingWithout|TestPublicWorkflowRunMachineLocal' -count=1 -timeout=5m
```

| Command / source | Result |
| --- | --- |
| Focused public/engine/invocation command above | PASS; cmd 15.800s, workflow 2.184s, stageexecution 17.631s, runtimeapplication 0.189s; before final diagnostic/pin corrections |
| Focused bare-binding/default command above | PASS after optional empty-token-list correction; public bare-binding two-stage fixture 4.114s |
| `go test ./internal/runtimeapplication ./internal/runtimeprofile -count=1 -timeout=5m` | PASS after diagnostic repair; 0.750s / 0.401s |
| `go test ./internal/codexruntime -count=1 -timeout=10m` | PASS after actual manifest pin; 21.091s |
| `go test ./internal/stageexecution -run TestEnginePerAgentTimeout -count=1 -timeout=5m` | PASS, 2.144s; real local process stops at agent timeout with longer stage default |
| `go test ./internal/local -run TestCutover -count=1 -timeout=5m` | PASS, 0.398s; isolated fixtures only |
| `go build ./...` | PASS |
| `go vet ./internal/... ./cmd/...` | PASS; repeated after final production changes |
| `go test ./... -count=1 -timeout=30m` | PASS, exit 0 at corrected source efff9de; cmd/lingo 238.311s, local 129.304s, workflow 89.066s, stageexecution 12.395s, codexruntime 20.270s |
| `bash scripts/check-go-quality.sh format` | PASS, Go 1.27.3 linux/arm64 |
| `go run scripts/check-architecture.go domain` | initial FAIL encoding/hex; final PASS (12 files), no allowlist change |
| `go run scripts/check-architecture.go application` | FAIL initial and final: strconv, unchanged baseline |
| `bash scripts/test-check-project-domain.sh` | PASS |
| `bash scripts/test-check-projectapp.sh` | FAIL, independently reproduced with initial-HEAD scripts; existing UTF-8 fixture/allowlist mismatch |
| `python3 scripts/check-adr-governance.py` | PASS, 38 ADRs |
| `python3 scripts/check-automation-registry.py` | PASS, 99 surfaces; no durable script added |
| `bash scripts/check-claude-bootstrap.sh --skill-adapters .` | PASS |
| `bash scripts/check-sensitive-files.sh .` and `--staged .` | PASS independently; staged scan before each commit |
| `bash scripts/validate-repository.sh .` | FAIL on inherited ignored `.s1-03-lingo-test.log`; no cleanup performed |
| `bash scripts/test-docs-offline.sh` | PASS installed bundle integrity/link/negative checks on isolated linux-arm64 archive at source 989da5f; qualification PARTIAL: source/network denial, native Runtime and pinned v1 installer unverified |
| `git diff --check`; JSON parsing and local Evidence link checks | PASS |
| Race / local staticcheck / gitleaks | NOT_RUN: gcc / staticcheck / gitleaks absent; no substitute installed |
| CI at source `989da5f192618d7dbc7eb393abfc20fe0fed4c6d` | FAIL Tests, sole skill-manifest pin failure; prior Formatting/Vet/Staticcheck/Build passed; fixed in efff9de |
| CI at corrected source `efff9decaafa80cda63d76e0dc61693f7fa0e0ef` | PASS: PR Checks + delivery-metadata, exact head read-back; [CI run](https://github.com/rgomids/axiom/actions/runs/41222161775) completed 2026-10-31 17:02:44 UTC |
| Exact final Evidence SHA CI | Read back separately in the final receipt; never substitute prior-SHA status |
| Live vendor tests, actual cutover, Provider remote effects, merge/release/acceptance | NOT_RUN, not authorized |

Two superseded complete-suite diagnostics were terminated (exit 143) after
corrections; a third superseded run was likewise terminated after CI identified
the missing pin. SIGINT did not end the first two; owned test trees were stopped
with PID handles and SIGTERM. No installed-state process was inspected/stopped.
Their partial package output is not final-suite PASS. The last full command runs
against corrected production plus actual manifest-pin bytes; final result above.


## technical_decisions[]

- **Local defaults:** explicit bounded environment input versus a new persisted
  default-policy object. Chose the former: existing resolver precedence and
  public configuration without a new lifecycle. Trade-off: operator/session
  environment must be configured; execution freezes effective provenance.
- **Optional catalogue:** permit bare format-4 Runtime binding versus generating
  a placeholder Profile. Chose bare binding; existing format-1/2/3 semantics and
  nonempty referenced Profile checks remain strict. Verified public no-catalogue
  progression and V4 codec/authoring negatives.
- **Retry:** validate frozen invocation under current restrictions versus
  resolving changed defaults before restoring old values. Chose frozen inputs
  first, preserving revocation checks and immutable ordinary retry even when
  defaults become malformed. Fresh selection remains explicit.
- **Configured controls:** persist bounded per-agent controls and exact before
  timing in the same snapshot; keep unimplemented validators refused. Replacing
  validators with human gates or output existence would weaken accepted intent.
- **Architecture baseline:** replace domain digest decode/re-encode with the
  equivalent anchored 64-lowercase-hex syntax check. No allowlist change. Leave
  the unrelated application read-model dependency problem explicit for the
  remaining consumer retirement rather than widen imports or conceal it.
- **Delegation:** direct work. No verified capability hierarchy strictly below
  Orchestrator was available; no models were hardcoded for delegation, no
  specialist findings or independent review are claimed. DESIGN NOT_APPLICABLE.

Evidence refs for these decisions: this PR's final SHA in the receipt, changed
source/test paths above and the two JSON reports. No new ADR/material baseline
change was introduced.

## deferred_findings[]

- **Application architecture baseline:** initial and final
  `go run scripts/check-architecture.go application` report
  `FAIL: non-allowlisted application import: strconv`, first at
  `internal/projectapp/active_workflow.go`. The file is unchanged from initial
  `1ea08c8447e2692ddc9cb3a1f1d4695af93c83bf`; its Workflow read model also
  depends on workflowdefinition, outside the existing pure application
  allowlist. Exact scope/provenance is preserved; no allowlist was weakened.
  The import does not certify a runtime security failure, but the required
  architecture check remains FAIL and review readiness is not claimed.
- **Inherited ignored artifact / repository validator:**
  `bash scripts/validate-repository.sh .` FAILS on prohibited path
  `.s1-03-lingo-test.log`, an inherited ignored artifact (`.gitignore:36`,
  36,764 bytes, mtime 2026-10-31 12:52:55 -0300, raw-byte SHA256
  `d6ee23dd4c98846202ec0a08f20bfb2b95aa17ea4453af2d551474f63c73cf28`).
  Content was not read, staged, imported or deleted; no cleanup/quarantine was
  performed under the explicit no-clean instruction. Staged source scanning
  passes separately. This is not evidence of a credential in the file, only
  the actual validator refusal. Safe artifact disposition remains unresolved.
- **Checker fixture baseline:** `bash scripts/test-check-projectapp.sh` FAILS
  at `domain profile accepted an application-only allowance`: the fixture expects
  `unicode/utf8.ValidString` to be forbidden in domain although the current
  domain allowlist explicitly permits it. Reproduced using only scripts
  extracted from initial HEAD into an isolated temporary directory, exit 1.
  Checker blob `c977de43b8a5be1dc437c5c282370d7c4c4e83c5`, fixture blob
  `0c5b000b2375528381c270fa2d691c1167600fbe`; both unchanged. No fixture or
  allowlist weakening; this optional discovery is deferred for pre-merge display.
- Historical Model Advisor synthetic failure and builtin SDD r1/r2 drift remain
  preexisting deferred findings from prior Evidence, not repaired or reclassified.
- PR title/body still describe documentation-only work. PR metadata and review
  text read-back did not establish authority for executor metadata mutation;
  proposed replacement below is ready for Maestro's authorized publication.

## remaining_work / bounded handoff

1. Resume exact Workflow validator lowering/execution: typed artifact/evidence
   correlation, result-bound human review and optional explicitly authorized
   command checks. Preserve stop conditions and immutable attempts; no Policy
   catalogue, invented gate or silently skipped control. Finish full input/context
   and schema-2 bounds reconciliation, including instruction/aggregate size.
2. Use the producer/consumer inventory to port canonical readiness, child context
   and lifecycle views, then retire S1 A–H/K legacy commands/readers/writers and
   codecs with replacement safety tests. Keep actual S2 collaboration/integration
   and S3 remote/rework requirements distinct; do not delete their required
   safety outcomes prematurely.
3. Reconcile remaining architecture application baseline in that scoped consumer
   work; full required validation and exact-head CI must pass before claiming
   review readiness. Recompute deletion union after changes.
4. Finish S1.8 after structural retirement; actual installed-state inventory,
   backup/stop/cutover remains separately authorized and NOT_RUN.

Proposed PR title: `feat(workflow): AXM-42 canonical Execution — partial S1`.
Proposed body: `Implements the public canonical Execution path with explicit
machine-local invocation defaults, optional Profiles, immutable per-attempt
selection and supported configured gates/agent controls. S1 remains partial:
exact validators and structural A–H/K consumer retirement are unfinished. See
implementation-s1-05.md for validation, deletion/cutover evidence and remaining
work. No real cutover, live vendor E2E, S2/S3 delivery or Product Accepted.`
Preserve `Related-Issues: none` and `Completes-Issues: none`.

No Linear publication, callback, merge, release, deployment or acceptance.
**REPORT_PENDING** — Maestro publishes this attempt's single receipt and
reads it back; next gate `resume_implement`.
