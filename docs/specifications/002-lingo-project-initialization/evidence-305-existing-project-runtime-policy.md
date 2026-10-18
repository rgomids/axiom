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
| T03 | 01, 06, 07, 09, 10, 13, 14 | V03 | NOT RUN / pending |
| T04 | 07, 10, 14 | V04 | NOT RUN / pending |
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
