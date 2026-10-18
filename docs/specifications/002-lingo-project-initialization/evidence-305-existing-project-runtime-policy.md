# AXM-34 / G-1 implementation Evidence

Status: **Implementation in progress**, 2026-10-18. No product acceptance,
Execution transition, merge, release or issue completion is claimed.

Authority: approved Tasks at `dea54ea39f87f3a8cbe9c43fbda3e302e55402ac`,
[human receipt](https://github.com/rgomids/axiom/pull/327#issuecomment-6246118505),
and the operator's explicit G1-T01–T09 implementation request. Approved
[amendment](amendment-305-existing-project-runtime-policy.md),
[Plan](plan-305-existing-project-runtime-policy.md) and
[Tasks](tasks-305-existing-project-runtime-policy.md) retain their historical bytes.
The only delivery PR is [#327](https://github.com/rgomids/axiom/pull/327), branch
`docs/axm-34-project-runtime-policy-specification`; technical issue #305 remains open.

## Preflight

Executed: `git status --short`, `git branch --show-current`, `git rev-parse HEAD`,
`git ls-remote origin refs/heads/docs/axm-34-project-runtime-policy-specification`,
`gh pr view 327`, `gh issue view 305`, `gh pr list --state open`, PR comments/reviews,
and direct Linear AXM-34 issue/relations/comments reads. Initial local/remote/PR
HEAD was the approved Tasks SHA; working tree clean. PR #327 was the only open
PR associated with AXM-34 (#317/#328 unrelated). `PR Checks` run `39100475104`
and delivery-metadata were SUCCESS at that SHA. No code validation is inferred
from these documentary-head checks. Linear had no blockedBy relation.

Canonical Execution ID, revision, current gate, active/suspended state and derived
lifecycle remain **unknown**: no checkout `.axiom` ledger or exposed canonical
endpoint was available. Provider prompts/approvals do not reconstruct that truth.
No new canonical Execution was started and no existing one was transitioned.
Remote HEAD and worktree are checked again at publication to detect competing work.
No delegation: a strictly lower model hierarchy could not be verified.

## Progress matrix

AC mappings retain the approved Tasks matrix; partial test coverage is not full AC
passage. PASS below applies only to the described unit and checks.

| Task | G1-AC | Oracle | State / Evidence |
| --- | --- | --- | --- |
| T01 | 01–04, 12, 14 | V01 | Implemented pure draft/candidate, tests below; integration still depends on T02–T07 |
| T02 | 04, 07, 08 | V02 | Pure compatibility implemented/tested below; integrated snapshot/immutable-consumer proof pending T03/T09 |
| T03 | 01, 06, 07, 09, 10, 13, 14 | V03 | Read-only sealed preview core and synthetic tests implemented; full readiness parity/integrated passage pending; continuation below |
| T04 | 07, 10, 14 | V04 | Guarded adapter increment and synthetic/multiprocess tests implemented; full coverage closure pending; continuation below |
| T05 | 01, 05, 06, 09–11, 14 | V05 | NOT RUN / pending |
| T06 | 01, 02, 05, 08, 11, 13, 14 | V06 | NOT RUN / pending |
| T07 | 01, 03, 06, 09, 12–15 | V07 | NOT RUN / pending |
| T08 | 01, 09, 14, 15 | V08 | NOT RUN / pending |
| T09 | 05, 08, 11, 12, 14, 15 | V09 | NOT RUN / pending |

## T01 — pure draft/candidate

Production paths: `internal/projectapp/policy_draft.go`, `policy_view.go`,
`edit.go`, `edit_publish.go`. Tests: `policy_draft_test.go`, `policy_bounds_test.go`.
The draft is opaque validated in-memory intent, carried separately from
`EditAuthority`; it cannot carry identity, paths or authority. It extends
`EditIntent` and refuses unrelated simultaneous edits. Canonical CLI transport is
still pending T07; no policy option or callable CLI journey is claimed.
`ApplyEdit` explicitly refuses policy publication until T02–T06 are integrated.

Draft JSON uses optional `runtimes`, `modelProfiles`, `runtimePreferences` objects.
Each has required `mode`: `absent`, `unconfigured`, `replace` with `values`, or
`members` with optional `add`, `replace`, `remove` arrays. Removal identities are
strings for Runtime IDs/Profile keys and `{role, complexity}` for preferences.
Members retain portable shapes; Profile `state: unconfigured` is mutually
exclusive with model/Runtime. Null, aliases, duplicates and unknown fields refuse.
An empty document object is a policy-only no-op. Empty member mode configures `[]`.

Trade-off: `encoding/json` alone accepts duplicate/case-alias fields. Existing
`strictjson.Decode` already composes duplicate/Unicode/closed-shape validation
with the adopted JCS library; reused it and added the manifest's 16,384-node,
depth-32 limit (keys and values, root depth 1). No new dependency, generic decoder
or storage framework. Pure collection merge is shared across the three collections
only. Final candidate uses `Project.Propose`, portable scalar safety and the
existing manifest codec; no local candidate seeding or defaults.

Tests exercise all forms/omission, additions to absent/unconfigured, exact members,
explicit dependency repair, same-model named Profiles, duplicate/conflicting or
missing targets, dangling/unconfigured references, unsafe model path, unsupported
v1/v4 policy edits, unchanged complete v2/v3 round trips, final 8/32/32 and
independent overflow, operation 16/64/64 bounds, exact byte/node/depth bounds,
malformed JSON/Unicode/shape/unknown fields. Semantic diff includes presence and
safe removed/replaced member values in deterministic order, excludes unrelated
credential/context fields, and has no local paths.

The synthetic Project UUID is `123e4567-e89b-42d3-a456-426614174000` (existing test
fixture). Pure previews preserve `context.md` and `policy.md` bytes/digests,
complete unrelated fields, local bindings, source state, UUID/schema and no-op
manifest formatting/local wire. A test initially detected no-op local slice
reconstruction changing bytes; policy intent now preserves the observed local
wire exactly and avoids checkout observation. These tests have no storage writer,
Provider, Runtime invocation, authentication or secret-value reader capability.
They do not prove guarded native publication or whole-journey output safety.

Commands executed successfully (exit 0):

- `go test ./internal/projectapp ./internal/project ./internal/manifest`
- `go vet ./internal/projectapp`
- `git diff --check`

`gitleaks` is unavailable (`command -v gitleaks`: exit 1), not installed.
V02–V09, CLI/skill/native conversation and native storage acceptance remain NOT RUN.
No readiness of a real local Runtime or Profile is asserted.

## Contract observation for T02

`project.workflowSelection` and manifest schema validation allow recorded Workflow
selection only in schema 4; `Project.SelectWorkflow` explicitly upgrades to 4.
G-1 remains exactly v2/v3. Their recorded active Workflow is therefore `none`.
Do not admit v4 policy edits or silently upgrade v2/v3 to manufacture a selection.
The portable compatibility helper must preserve the existing Workflow semantics
and can be tested separately on exact definitions; those tests do not imply v4
G-1 support. No accepted contract or ADR is changed.

## T02 — exact portable Workflow compatibility

`policy_workflow.go` resolves the exact active definition through
`ResolveActiveWorkflow`, recomputing the full validated definition digest and
respecting published index entries, immutable built-in revisions and explicit
none. The report projects logical references, schema, all Stage/Agent requirements,
complete before/after Profile/Runtime associations and introduced/unresolved/resolved
incompatibility. Both introduced and unresolved incompatibility block compatibility.
Absent local configuration/proof does not enter this portable check. Report overflow
refuses with `workflow_impact_limit`; no truncated compatibility success.

Explicit Profile refs ignore preferences. Historical schema-1 `policy-default`
follows the exact role/complexity preference, then exposes the full admissible
portable set without selecting one. Schema 2 treats that spelling as an explicit
Profile key. The small portable association/Runtime-constraint helpers are reused
by `ValidateWorkflowProfilesV3`; its existing local owner/model/effort gates and
error ordering remain intact. No compiler/dispatch behavior or local format changes.

`policy_workflow_test.go` covers built-in/project definitions, schemas 1/2,
explicit/default precedence, multiple admissible keys, Runtime constraints,
preference changes, missing/unconfigured candidates, retained/new incompatibilities,
explicit repair, none, missing/tampered/retired/index-drift/unindexed definitions,
compatible reselection refusal, safe raw-error suppression, output overflow and
unchanged input Project. Existing compiler/Profile-chain tests were rerun.
Four fixed impact scopes disclose immutable bindings and fresh future admission;
Execution/Work Item inventory remains unknown/uninspected. Native publication,
existing immutable-record byte comparison and live conversation are still pending.

PASS (exit 0): `go test ./internal/projectapp ./internal/workflowcompiler
./internal/workflowdefinition`. T01 commit: `3b0ed19`; T02 tested as its working-tree
descendant. Final exact committed-head validation is recorded below at consolidation.

A safety review also closed a pre-exposure seam: policy proposals cannot yield a
usable legacy `PublicationRequest` and `ApplyEdit` refuses them until guarded
publication integration. No CLI option exposes policy editing in these units.

## Consolidated AC progress

No full G1-AC passage or integrated G-1 completion is asserted by this increment.
The approved Tasks matrix remains the complete acceptance checklist.

| Criterion | Existing unit Evidence | Remaining oracle / Task |
| --- | --- | --- |
| G1-AC01 | v2/v3 pure draft/candidate round trip | V03/V05–V08, T03/T05–T08 full journey |
| G1-AC02 | V01 forms/partial edits/no-op bytes/companions | V06, T06 observed persisted state |
| G1-AC03 | V01 strict shape/references/bounds | V07, T07 transport and safe diagnostics |
| G1-AC04 | V01 dependent edits, V02 portable exact refs | T03 integrated compatibility gate |
| G1-AC05 | No Project/registry writer exposed | V05/V06/V09, T05/T06/T09 actual A/B isolation |
| G1-AC06 | Portable helper has no local-presence gate | V03/V05/V07, T03/T05/T07 readiness/save |
| G1-AC07 | V02 exact resolution/requirements/before-after | V03/V04, T03/T04 snapshot integration and guards |
| G1-AC08 | V02 four fixed scopes, unknown inventory | V06/V09, T06/T09 immutable record byte comparisons |
| G1-AC09 | Policy ApplyEdit/request fail closed pending guards | V03/V05/V07/V08 full sealed authority journey |
| G1-AC10 | No guarded policy publication yet | V03–V05, T03–T05 dependency/writer coverage and races |
| G1-AC11 | Existing recovery protocol left unchanged | V05/V06/V09, T05/T06/T09 policy fault/read-back proof |
| G1-AC12 | V01 same-schema round trip, v1/v4 policy refusal | V07/V09, T07/T09 full compatibility/CLI regression |
| G1-AC13 | No proposed-snapshot readiness yet | V03/V06/V07, T03/T06/T07 canonical equivalence |
| G1-AC14 | V01 strict parsing/scalars, V02 bounded logical output | V03–V09 all integrated output/storage/security boundaries |
| G1-AC15 | No CLI/skill option exposed | V07–V09, T07–T09 inspection/transport/conversation |

## Review and continuation

Orchestrator review found and corrected three issues before publication:
no-op local reconstruction, unguarded legacy publication-request exposure, and
legacy schema-1 authoring permission expansion during helper reuse. Regression
tests cover all three. Workflow impact serialization now checks the cumulative
byte budget per complete row before retaining more rows, avoiding an oversized
aggregate JSON allocation. No blocking finding was identified for the bounded
pure units; this is self-review, not independent/human approval of G-1.

The session budget requires consolidation before starting T03's sensitive
observation/publication chain. This is **in progress**, not a discovered product
blocker and not completion of nine Tasks. No further implementation authority is
needed for the already authorized scope; carry it into continuation. No subagents
were spawned. Full V01/V02 integrated passages remain open where the tables say
pending; T03–T09 are unimplemented. Do not remove policy publication refusals merely
to expose a CLI option. The legacy `EditPreview` still contains manifest/paths and
must never be serialized unchanged for G-1.

Next action is T03, following the approved dependency order. Inspection notes:

- `ProjectReadiness.evaluate` currently calls `Runtime.Availability` by UUID,
  which reloads persisted policy. Compose proposed Project/retained installation
  snapshots with the same evaluator instead of accidentally assessing old policy.
- `runtimeapplication.EvaluatePolicyV3` is the existing pure candidate evaluator;
  `Service.CurrentAvailability` supplies canonical request-independent blockers.
  Do not introduce another resolver or locally configured save prerequisite.
- Production `runtimePolicySource.Load` and
  `runtimeObservationSources.observer` live in `cmd/lingo/runtime_policy.go`.
  `ExecutableObserver.Observe` reads executable identity and integration facts;
  it launches/authenticates nothing, but observations have fresh `ObservedAt`.
  Do not bind fresh clock values into deterministic preview identity. Audit all
  actual consulted sources before choosing the read-only composition. Required
  unreadable snapshots must refuse; observed absence and unknown proof are explicit.
- T04 must resolve every dependency/reader/writer/lock/boundary/test row in Tasks.
  `ProjectEditPublisher.PublishEdit` already locks state root → projects → target;
  `OperationalStore.CommitOperational`, `RuntimeProfileStore.Publish` and Workflow
  publication share root coordination. Runtime store exposes `inspectLocked` for
  lock-held reads; public `Inspect` under a held exclusive root would self-lock.
  Portable source/companions need complete-generation checks through commit.
  External executable/integration/proof sources need an actual guard or explicit
  fail-closed refusal; a final uncoordinated read is not proof of exclusion.
- T05/T06 must extend existing portable-then-local publication/recovery and add
  observed canonical read-back; preserve no-op bytes and classify post-commit
  uncertainty honestly. `PublicationRequest` and `ApplyEdit` policy refusals are
  deliberate temporary safety gates, to replace only after T04–T06 proof.
- T07 extends the existing configure/edit adapter, its actual help/inspection and
  guided/noninteractive tests. Draft wire semantics are above; option spelling is
  still undecided, not an invented existing CLI option. T08 canonical skill source
  is `internal/codexruntime/skills/axiom-project/SKILL.md`; preserve frozen published
  skills/corpus. T09 consolidates all approved V01–V09 cases and native Evidence.

```yaml
objective: Complete G1-T01–G1-T09 in PR 327 under the approved amendment/Plan/Tasks
state: in_progress
completed_units:
  - T01 pure draft/candidate and semantic policy projection; commit 3b0ed19
  - T02 pure exact Workflow compatibility and authoring reuse; commit 4816372
remaining_units:
  - T03 implementation/validation; depends T01/T02; safe sealed preview and canonical proposed readiness, V03
  - T04 implementation/security/concurrency; depends T03; complete guarded dependency/writer table, V04
  - T05 implementation/fault validation; depends T04; exact replay and truthful publication/recovery, V05
  - T06 implementation/validation; depends T05; observed read-back and classification, V06
  - T07 CLI implementation/integration; depends T01–T06; real transport/help/inspection/guided parity, V07
  - T08 skill integration; depends T07; thin canonical conversational routing, V08
  - T09 integrated security/regression/native Evidence; depends T01–T08; all AC/oracles with explicit native limits, V09
decisions_and_assumptions:
  - Approved amendment/Plan/Tasks and accepted ADRs preserved byte-for-byte
  - G1 only schemas 2/3; active recorded Workflow is none there; schema-4 policy editing remains refused
  - Canonical Execution state unknown; no new Execution or workflow transition
  - No verifiable strictly lower model hierarchy; execute in main session
authority_granted_and_pending:
  - Granted: scoped implementation/tests/docs, commits/push and updates to same branch/PR 327
  - Not granted: merge/self-approval/release/issue closure/product acceptance/infra/secrets/scope expansion
validation_status:
  - Unit and regression commands/results recorded in this Evidence and PR
  - V03–V09 and native G1 journey NOT_RUN
  - Staticcheck unavailable locally; full Go quality gate not locally passed
next_action: Revalidate remote HEAD/comments/concurrent changes, then implement T03; do not create another PR or infer canonical Execution state
```

## Consolidated verification receipt

Code head: `4816372e33d194c73f551a1ea4310fd5b6e12770`, following
`3b0ed19`. GitHub PR head read-back confirmed this exact code head, Draft/Open,
with only #327 open on the canonical branch. Initial push (exit 128) found a stale
`/usr/bin/gh` credential-helper executable path. Command-scoped use of the already
installed `/usr/local/bin/gh` helper succeeded (exit 0); no persistent Git config,
credential, permission or protection changed.

Environment: `go version go1.27.2 linux/amd64`; `uname -srm` reported Linux
6.12.94+ x86_64; `stat -f -c %T .` reported `ext2/ext3`. This is local unit/regression
Evidence, not qualification of every supported native storage matrix row or the
repository's Go 1.26 CI toolchain.

| Command / check | Result |
| --- | --- |
| `go test ./internal/projectapp ./internal/project ./internal/manifest ./internal/workflowcompiler ./internal/workflowdefinition` at code head above | PASS, exit 0 |
| `go test -race ./internal/projectapp` at code head above | PASS, exit 0 |
| `go test -race ./internal/projectapp ./internal/workflowcompiler ./internal/manifest` before final report-budget hardening | PASS, exit 0; Project application race suite repeated afterward |
| `go test ./...` on the T02 worktree before final report-budget hardening | PASS, exit 0; includes CLI, compatibility, local storage and Workflow regression suites |
| `go vet ./...` before final report-budget hardening; `go vet ./internal/projectapp` afterward at code head above | PASS, exit 0 |
| `./scripts/validate-repository.sh .` | PASS, exit 0; optional runtime behavioral scenarios explicitly SKIPPED/UNVERIFIED |
| `./scripts/check-go-quality.sh all` | FAIL/UNVERIFIED, exit 1: Staticcheck executable absent; formatting and `go mod tidy -diff` stages passed; no tools installed |
| `git diff --check`, staged content/path review and `./scripts/check-sensitive-files.sh --staged .` before each code commit | PASS, exit 0 |
| New Go comment scan | No added comments (search exits 1 for no matches) |
| Diff of approved amendment/Plan/Tasks against `dea54ea39f87f3a8cbe9c43fbda3e302e55402ac` | Empty; approved bytes preserved |
| V03–V09, G1 native Linux/macOS publication/fault qualification and native Codex/Claude conversation | NOT_RUN: these implementation chains remain pending |

SHA-256 source fingerprints (code head above):

| Path under `internal/projectapp/` | SHA-256 |
| --- | --- |
| `policy_draft.go` | `507a6e0b62025c16cb00f2c703c45e28fc7ce2a376ef92cc7793cab212192221` |
| `policy_view.go` | `e8fa6fa843cff94424f6e2cc8eb9ab76f8ff64b55732f6010b271d4ee316a1c4` |
| `policy_workflow.go` | `6bb12d70c3848ea862d3e653e09f2ce182c20f3fba7f6b5964b622b0cf720ea9` |

CI for the final handoff commit is reported separately in PR #327, never inherited
from the previously green documentary SHA. Technical/human review requested for
the bounded increment; continuation of T03–T09 is required before G-1 review for
full delivery. No local Runtime readiness, product acceptance or Work Item
completion is inferred from any test or CI outcome.


## Continuation increment — sealed preview and publication guards

This section supersedes the earlier continuation handoff, without retconning its
receipts. G-1 remains incomplete. T01/T02 were reused, not reimplemented.

Preflight: worktree clean at `a9ec90421452a6725460045b418fefe235f719ca`;
remote branch and PR head matched. Read the exact handoff comment
`6250512102`, #305, #327 comments/reviews/checks and open PR inventory. Only #327
belongs to AXM-34. Approved amendment/Plan/Tasks diff against `dea54ea` is empty.
Direct Linear issue/relations/comments reads confirmed In Progress, no blockedBy,
and the exact Tasks approval. No provider narrative was imported. Canonical
Execution remains unknown; no new Execution or transition. No delegation because
strict model inferiority cannot be verified.

Code increment `3c4cf2daf7475c8ea8ea790d5532b9cc348f47a4` adds the T03 core:
`PreviewPolicyEdit`, its opaque proposal and separate safe DTO, the local
`PolicyEditObserver`, and metadata-only Runtime evaluation. Selection happens once;
the local adapter rechecks selected installation and manifest bytes while reading
operational/configuration/catalog facts. The proposal seals full candidate via the
existing EDIT digest plus dependency tokens, policy/diff, exact Workflow report,
admission, scoped readiness and allowed effects. No fresh timestamp enters identity.
Preview output omits manifest/context/name, paths and credential references.
Dependencies are internal hashes, not portable fields or output paths.

Readiness uses the proposed Project, existing `EvaluatePolicyV3` and
`EvaluateReadiness`, with retained installation observations explicitly disclosed as
`retained_installation_metadata_no_new_proof`. Work Item does not inherit Runtime
blockers. No executable lookup/launch, auth/network/Provider call, credential-value
read or proof acquisition is wired. Unknown live evidence remains
`inventory_unavailable`; stable outer `runtime_policy_unavailable` and
`runtime_resolution_blocked` accompany fixed inner reasons. This does not establish
real machine readiness. `project validate` still uses its existing composition:
full preview/validate/read-back observation parity remains to be integrated.

The T04 adapter increment adds optional exact `PolicyDependencies` to existing
`EditPublicationRequest`. With them, publication validates policy-only changes and
installation bookkeeping preservation before I/O. Root → projects → installation
locks remain held through portable commit. It checks dependencies before creating
recovery state and again under the portable lock at `beforeCommit`, using lock-held
readers rather than nested `Inspect`. Missing stores stay missing. Workflow
compatibility is recomputed under protection. Legacy EDIT requests retain existing
behavior. Application policy apply and legacy policy `PublicationRequest()` remain
closed; no CLI flag or conversational route is exposed by this increment.

| Dependency | Reader / token | Writer / coordination | Protected check / tests |
| --- | --- | --- | --- |
| Source manifest and object/ancestor identity | `policyFileToken`, selected snapshot and exact manifest CAS | `PortableStore.Update`, namespace/source locks | `policyGuard` inside `PortableStore.Update` beforeCommit; same-byte inode replacement and stale token tests |
| Installation wire, identity, UUID/slug | `PolicyEditObserver`, `policyFileToken`, exact LocalExpected | `ReplaceRecord` / `ProjectEditPublisher`, state-root → projects → installation | exact bytes plus identity before commit; stale/identity replacement and policy-only-scope tests |
| Operational state/absence | `observeOperationalRecord`, hash of exact revision/absence | `CommitOperational`, shared state-root chain | lock-held reread before commit; absent→present test |
| Workflow catalog/index/selected definition or none | `readWorkflowCatalog`, hash of complete catalog including wire and documents | `PublishWorkflow`, portable namespace lock; selection writers use existing EDIT | catalog digest and exact compatibility under portable lock; changed catalog test |
| Runtime/Profile config/absence and authorization epoch | `RuntimeProfileStore.inspectLocked`, existing exact Token | `RuntimeProfileStore.Publish` / recovery / epoch publication, state-root coordination | lock-held token reread through commit; missing-store→present and stale token tests |
| Live executable/auth/capability observations | None consulted | None added | explicit unavailability; no freshness or proof claim |
| Companion documents / unsupported sources | Existing single-manifest owned PortableStore only | No new writer/support | source confinement and complete-candidate decoding refuse unsupported inputs; full G-1 companion qualification remains open |

Actual new tests in `internal/local/policy_edit_test.go` and
`policy_edit_guard_test.go` cover deterministic safe preview, caller-copy isolation,
unchanged portable/local bytes, absent stores, candidate model change instead of
old-policy reload, corrupt configuration, unsafe/unsealed facts, canonical
unconfigured blocker, positive guarded publication despite local config absence,
all five stale tokens, controlled changes to each dependency immediately before
commit, policy-only scope refusal, and two native processes with the same request
(exactly one applied, one conflict; 15-second process deadline). Process tests ran
three times. These are bounded synthetic tests, not full V03/V04 passage.

Executed checks for this increment (exit 0 unless noted):

- `go test ./internal/projectapp ./internal/runtimeapplication ./internal/local`.
- `go test ./... && go vet ./...`: PASS, exit 0 (CLI 182.894s), before
  the final policy/local prior-generation coherence check; targeted race/vet below
  repeated after that three-line guard correction.
- `go vet ./internal/projectapp ./internal/runtimeapplication ./internal/local`.
- `go test -race ./internal/projectapp ./internal/runtimeapplication ./internal/local`
  on the initial preview increment; local suite 100.846s.
- `go test ./internal/local -run 'TestPolicyPublication' -count=3` and scoped vet.
- `go test -race ./internal/local -run 'TestPolicy' -count=1` after guard additions;
  repeated after canonical unconfigured-blocker ordering correction.
- `./scripts/check-go-quality.sh format`, `./scripts/check-go-quality.sh tidy`.
- `./scripts/validate-repository.sh .`: PASS; native conversational behavioral
  scenarios explicitly skipped/unverified.

Environment: Go 1.27.2, Linux 6.12.94+ x86_64, filesystem reported `ext2/ext3`.
Native mechanisms are existing directory-descriptor locks, os.Root confinement,
no-follow private-file checks and ADR-0007 file publication. This does not qualify
macOS, every filesystem, hostile arbitrary same-UID races or power loss. No new
dependency/tool installation. Existing standard-library codecs/SHA-256 and adopted
lock/publication primitives suffice; no generic transaction engine was added.
Gitleaks remains unavailable; staged sensitive-file checks remain required.

At read-back of preview commit `3c4cf2d`, PR Checks run `39132593482` and
delivery-metadata `39132593487` were SUCCESS. These results do not apply to later
commits. Final guard commit and its CI are reported in the PR publication receipt.

### Remaining work / next session

T03 and T04 are **in progress**, not complete. Full V03 requires observation parity
with canonical validate/read-back, broader Workflow/local-state/capability/effort
cases, explicit actionable repair guidance and output-bound/security closure.
Full V04 requires the approved exhaustive drift/writer/absence/content-counter and
native supported/unsupported-source coverage, beyond this guard core. Do not
start the dependent T05 exposure until these passage criteria are satisfied.
T05 exact application replay/no-op/recovery, T06 observed read-back, T07 CLI/guided
transport, T08 thin axiom-project route and T09 integrated/native Evidence remain
unimplemented. No G1-AC01–15 full passage is claimed; the original gap matrix above
remains binding, with V03/V04 partial evidence added here.

```yaml
objective: Complete G1-T01–G1-T09 in sole PR 327
state: in_progress
completed_units:
  - T01/T02 pure units at 3b0ed19/4816372, unchanged in this continuation
  - T03 sealed preview core at 3c4cf2d, with scoped synthetic checks
  - T04 dependency guard core, see final publication receipt
remaining_units:
  - T03 readiness parity and exhaustive V03 closure
  - T04 complete guard coverage and V04 closure, after T03
  - T05 exact replay/no-op/recovery, after T04
  - T06 observed canonical read-back, after T05
  - T07 canonical CLI/guided transport, after T06
  - T08 thin conversational route, after T07
  - T09 integrated AC/V/native Evidence, after T08
constraints:
  - preserve approved amendment/Plan/Tasks and existing T01/T02
  - no application policy apply or CLI exposure until guarded replay/read-back
  - canonical Execution unknown; do not infer absence or create a competing one
  - same branch and PR; normal pushes; no merge/release/acceptance/closure
validation_status: see executed checks above and final PR receipt
next_action: finish T03 parity/negative cases, then close T04 coverage before T05
```

## r3 — readiness parity and dependency identity

Resumed from clean `788e181f99124ac26ade6393e237b16dc64a251a`; remote/PR
matched, both checks passed, only #327 is open for AXM-34. Direct Linear read:
In Progress, no blocking relations. Canonical Execution remains unknown; no
Execution or Provider transition. No delegation: strict relative model hierarchy
cannot be verified. Approved amendment/Plan/Tasks remain byte-identical.

T03 now supplies fixed actionable Runtime guidance. Canonical readiness accepts
an optional snapshot-aware probe; production schemas 2/3 consume the same pure
policy evaluator and inspected configuration as the proposed snapshot, without
executable, authentication, capability or credential acquisition. Missing local
facts remain blockers, never permission or portable invalidity. T04 now binds
operational file identity and all consulted Workflow files in the source token,
including exact contents via existing catalog validation. Standard library and
existing locks/hash/codec suffice; no new dependency or generic transaction layer.

Executed, exit 0: `go test ./internal/local ./internal/projectapp`; targeted
`go test ./internal/local ./internal/projectapp -run 'Policy|Readiness'`;
`go test -race ./internal/local ./internal/projectapp -run 'Policy|Readiness'`.
Tests include matching-but-unproven, missing, disabled, wrong Runtime snapshot
parity/guidance, exact operational-file replacement, same-revision changed Runtime
content, source relocation and hardlink refusal immediately before commit.
Initial disabled-case expectation was corrected to the existing evaluator's
`runtime_unavailable`; the application and proposed result already agreed.

These checks extend V03/V04, not native machine readiness or a claim of exhaustive
G1-AC passage. The local portable store accepts manifest plus the existing Workflow
catalog; unsupported companion layouts remain refused by the existing reader.
Application tests retain full companion snapshots; no source-layout expansion is
inferred. Native macOS and runtime conversations remain NOT RUN.

### r3 — T05 exact application replay

`ApplyPolicyEdit` re-observes and seals the candidate, denies mismatched or partial
UUID/digest/local authority, and preserves portable validity independently of local
execution readiness. Publication reuses the existing guarded portable-then-local
protocol. No-op enters the same protected dependency verification and returns
`unchanged` without staging, recovery or revision writes. Legacy unsealed policy
publication remains closed. Tests cover preview, partial/missing/wrong authority,
cancellation, stale replay, no-op bytes, precommit failure, interrupted recovery
and post-portable partial outcome, followed by existing owned recovery readers.

Executed, exit 0: `go test ./internal/local ./internal/projectapp -run
'Policy|Readiness'`; `go vet ./internal/local ./internal/projectapp ./cmd/lingo`;
`go test ./cmd/lingo -run 'Readiness|RuntimePolicy'`. The bootstrap regression now
asserts uninspected facts in canonical validate and separately retains the actual
explicit Runtime resolver's authentication/executable blocker assertions. T05's
application path is implemented; detailed publication faults reuse the existing
protocol and remain subject to final integrated V05/V09 evidence. T06 read-back
and CLI exposure remain next, not implied by these application tests.

### r3 — T06 observed read-back

Successful publication and guarded no-op now perform a new canonical selection
by exact UUID and a no-change preview through the same safe observer/evaluator.
Read-back compares actual manifest/local bytes and source identity to the expected
candidate, exposes expected and observed revisions, and separates `confirmed`,
`drift`, `unverified` and `recovery_required`. Runtime/operational/Workflow snapshot
change is disclosed independently of confirmed policy bytes. The confirmed commit
outcome is retained if read-back fails; nothing retries or rolls back automatically.

Executed, exit 0: targeted policy/readiness suites; targeted policy/readiness race
suites (before the additional A/B case); `go vet ./internal/local
./internal/projectapp`. `TestPolicyReadBackTruthAndDrift` covers schemas 2/3,
no-op, failed reader after commit, a later authorized writer before read-back and
changed local Runtime configuration. `TestPolicyRemovalPreservesOtherProjectRegistryAndBindings`
removes A's complete Runtime/Profile/preference set and verifies A's observed
revision, B's portable/local bytes and read-back, shared registry bytes, and all
unrelated A installation bindings. Synthetic UUIDs are the existing `123e4567...4000`
and `123e4567...4001` fixtures, never real provider data.

### r3 — T07 canonical CLI and guided transport

Actual interface: `project configure --project <uuid-or-slug> --policy-draft
'<strict-json>'`. `{}` reads policy; absent replay is validate/dry-run/preview.
Inline JSON reuses the bounded strict parser and avoids another input-file reader
or secret/path-reading surface. Policy intent cannot mix with unrelated EDIT or
CREATE. The existing exact three-part replay tuple alone enables publication.
`policyEdit` is a bounded safe report, with independent publication/read-back;
legacy `edit` output is not used. Help and inspection derive the real flag, while
mode authority explicitly describes policy-only intent and observed read-back.

Integration found and fixed retained-vs-observed Repository availability: preview
now reuses the canonical Repository/Documentation metadata probes, binds their
safe status maps in a `Readiness` dependency digest, and rechecks them before
portable commit. No document body, credential value or executable is read. This
completes operation-report parity rather than weakening the assertion to Runtime
codes only. Existing no-follow metadata checks and bounded commit checks suffice;
no new watcher, lock registry or dependency is introduced.

Executed, exit 0: `go test ./internal/local ./internal/projectapp ./cmd/lingo -run
'Policy|Readiness'`; `go test ./internal/cli ./cmd/lingo -run
'Policy|Readiness|SkillInspect|Strict'`. CLI tests prove schemas 2/3 installed
journeys; structured/guided identical previews and no implicit consent; exact apply
and observed read-back/validate operation parity; no-op bytes; stale replay;
unknown/unsafe/malformed/oversized/duplicate/mixed/partial-authority refusals;
tampered-schema/stale-installation refusal without adoption; safe output scans.
Strict flag regression now includes policy-draft. This is real CLI adapter testing,
not native host conversation or full native storage qualification.
