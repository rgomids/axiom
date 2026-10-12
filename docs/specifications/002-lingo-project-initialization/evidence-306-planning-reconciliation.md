# Issue #306 — Planning reconciliation Evidence

## Scope and authority

2026-10-11; documentation-only reconciliation in Draft PR #310.
Previous reviewed/remote HEAD verified before edits:
`2096509b03daa9fb345ec245cb9070bbb18a3479`.
Main verified: `5d52327b0367ba08d872b50e804df5c7d63a3e89`.
Working tree was clean; existing branch/PR reused.

Sources read: [maintainer D1–D5/onboarding record](https://github.com/rgomids/axiom/issues/306#issuecomment-6108025323)
and [textual REQUEST CHANGES](https://github.com/rgomids/axiom/pull/310#issuecomment-6108025068).
Direction is confirmed; exact reconciled Plan/Spec/Security/ADR artifacts remain
Proposed / Awaiting Human Approval. No Tasks, implementation, schema/fixture edits,
authenticated probes, Issue/Linear status change, merge or release.

## Integrated source findings

- Local format 1 has no Profile effort; Workflow schema 1 requires Agent effort.
- Current compiler derives controls from that Agent and reports requestedEffort.
- Configured Execution format 2 accepts only Workflow schema 1; existing graph
  resolutions do not persist the proposed complete Profile/effort snapshot.
- Both default-sdd-r1 fixtures have identical bytes: ten engineer/medium agents,
  policy-default, both Runtime constraints, Workflow runtime-default effort.
- policy-default has no vendor priority: compiler previews constraints separately
  and demands one successful match. Both candidates can require an explicit choice.
- runtimebootstrap detection is passive, but Run installs Axiom integrations;
  read-only onboarding inventory must reuse detection without installation effects.
- Production observer proves executable identity/integration only; auth does not
  prove model/effort. New contracts need G-3 evidence, never manufactured proof.

Read-only contract and onboarding investigations inherited Orchestrator settings;
security/consistency reviews used bounded files and outputs. No capability-profile
mapping for cheaper explicit model overrides was available, so inheritance was
used without a higher model or invented selection. Orchestrator spot-checked
critical functions; agents accept no artifacts. Any review findings are integrated
before publication. This is design evidence, not execution evidence.

## Impact matrix — changes proposed for future authorized implementation

| Surface | Proposed impact | Current reconciliation effect |
| --- | --- | --- |
| Spec 002 | Local authoring v2 effort, auth exclusion, retirement, consent/onboarding | Proposed amendment updated; portable policy contract unchanged |
| Spec 007 | Workflow schema 2 forbids effort; migration/default-sdd/snapshot contracts | Separate proposed amendment added; accepted spec/amendments retained |
| ADR-0023 | Local skill/onboarding and explicit inventory extension | Proposed revision only |
| ADR-0024 / ADR-0022 | Sole effort owner, versioned snapshot/migration; future bounded partial supersession | New Proposed ADR; accepted ADR-0022 unedited |
| Local RuntimeProfile/store | Format 2, strict tombstone, preview-bound publication, shared diagnostics | No Go/source/store state changed |
| Workflow schema/fixtures | Future schema2 + default-sdd-r2; retain schema1/r1 historical references | No schema/fixture edits |
| Compiler/resolver/dispatch | Effort from Profile only; v2 tuple/proof/drift checks; legacy admission blocks | No code or live dispatch |
| Execution/graph/Stage Plan | Proposed configured3, graph/proposal2, StagePlan/result2; immutable tuple | Historical formats and bytes unchanged |
| Bootstrap/skills/installers | Passive first-use guidance; optional neutral templates; axiom-runtime parity | No skill/catalog/install changes |
| Project/readiness | Existing exact intersection, separate #305 authority, scoped diagnostics | No portable policy/operational mutation |
| Compatibility/recovery | Supported old reader paths, new corpus/version/window; no universal downgrade | No corpus changes; behavior remains future Evidence |

## Accepted/source baselines preserved

The following SHA-256 hashes were captured/read and compared against the remote
reviewed HEAD. Their bytes must remain unchanged through this documentary revision.

| Path | SHA-256 |
| --- | --- |
| `internal/runtimeprofile/runtimeprofile.go` | `c75c1031af93d0a14fd7c0ea079f7fb92b879c19b410110f53080db6575a583e` |
| `internal/local/runtime_profile_store.go` | `a8026cb9693cd03a17bf11f1ef42e067d5e4d8ebee3410abafe3b082deb98ba2` |
| `internal/runtimebootstrap/bootstrap.go` | `6d91f838fa76a975c1ef980f7f35f83e5348356a4303f26ed76e02e5cfebd769` |
| `internal/workflowdefinition/workflow.schema.json` | `a11760414057ab93735f12f8c169dedcd2ea9923fb4d8cbb537ea786ee504dd3` |
| `internal/workflowdefinition/default-sdd-r1.json` | `97066c66485a3df865601e108dd4d8e82c84d6216869a3c71ff93cf155378d00` |
| `docs/specifications/007-configurable-workflows/workflow.schema.json` | `a11760414057ab93735f12f8c169dedcd2ea9923fb4d8cbb537ea786ee504dd3` |
| `docs/specifications/007-configurable-workflows/examples/default-sdd-r1.json` | `97066c66485a3df865601e108dd4d8e82c84d6216869a3c71ff93cf155378d00` |
| `docs/specifications/007-configurable-workflows/spec.md` | `c8a098285feb4df95faa7228c4e47445f26c958c202f159f898586a03d5746ca` |
| `docs/decisions/0022-dedicated-workflow-conversational-surface.md` | `ca81f87ac6633b1c102cf1f88d7f3b9c0877ee143570235ab6e9fbcb49deacb0` |

## Documentary validation

Checks executed for this reconciliation:

- `./scripts/validate-repository.sh .` — PASS, including 25 ADR structures;
  vendor behavioral scenarios explicitly SKIPPED/UNVERIFIED.
- Local Markdown target/anchor check — PASS (146 targets, seven anchors).
- Accepted/source baseline hash + `git show 2096509:<path>` comparison — PASS
  (nine selected files, identical bytes; source/schema/fixture/accepted docs).
- D1–D5 / PE / ON / acceptance marker check — PASS; semantic consistency review
  additionally corrected unique-candidate selection, auth-versus-storage semantics,
  legacy reference migration, no-override source inspection and rollback generation.
- Diff scope check — PASS, nine changed/new Markdown paths only;
  `git diff --check` / staged whitespace check — PASS.
- `./scripts/check-sensitive-files.sh --staged .` — PASS.
- Scoped `gitleaks dir` on Spec 002, Spec 007 and ADR docs — PASS, no detected leaks.
- `scripts/delivery-issues.sh check-pr` — PASS; Related-Issues #306 and
  Completes-Issues none. No completion/closing projection is requested.

Final exact-head remote and PR read-back are reported in the PR review record,
including CI classification and any suite skipped for documentation-only changes.
These checks validate artifacts, not proposed production behavior.
No fresh broad Go suite, vendor/native auth or model inference is required for
this documentary scope. Prior baseline Go tests are explicitly historical in
Security, not current tests of these proposed versions.

Expected controls: repository/ADR governance validation; local Markdown links;
D1–D5/PE/ON traceability and cross-document consistency review; diff scope/whitespace;
accepted-code/schema/fixture hash preservation; staged sensitive-file and scoped
secret scans; PR metadata; exact remote-head/read-back before publication.

## Limits and next action

Documentary/synthetic CI cannot prove model access, effort support, actual first-use
operation or cross-platform migration. New format/legacy-continuation/no-override
semantics and cross-cutting implementation ownership still require exact human
review with Plan/Spec/Security/ADR acceptance. STOP before Tasks/Implementation.
Final publication SHA/artifact blob identities belong to the PR review record.
