# Issue #303 — implementation Evidence

Scope: GitHub [#303](https://github.com/rgomids/axiom/issues/303) / AXM-7.
Full accepted implementation scope T01–T13, including T08a.
Final convergence is recorded in the validation appendix of [Draft PR #308](https://github.com/rgomids/axiom/pull/308). No human/product acceptance is inferred.

## T01 — approved baseline gate: PASS

- Base: `3aa2501da4a7003b00ed14f3204e669ed54078ea`; branch: `feat/303-workflow-skill`.
- [Contract PR #304](https://github.com/rgomids/axiom/pull/304): `MERGED`,
  merge commit equals the base, merged at `2026-10-10T22:19:33Z`.
- [Human acceptance](https://github.com/rgomids/axiom/issues/303#issuecomment-6102714056):
  maintainer `rgomids`, exact Plan/Addendum approval at reviewed head
  `a548d44ea0e0f432eca80ebde80c420616771334`.
- [ADR/amendment acceptance](https://github.com/rgomids/axiom/issues/303#issuecomment-6099596104):
  reviewed head `7db80ac95b5e83cd8cc9aaaadc785633c3ee3fa9`.
- Fetched `origin/main` and PR #304 head. `git diff FETCH_HEAD <base> --`
  the Spec 007 directory and ADR-0022: empty. Approved PR bytes are on main.
- Compared both accepted reviewed heads with the base. Differences are the
  authorized Status/index acceptance reconciliation, historical annotations
  and acceptance-state references. No material contract divergence found.
  Original accepted blobs remain in Git history; this implementation changes
  none of the five contract files below.
- Linear read-only observation: AXM-7 `In Progress`, `Blocks AXM-8` present.
  Some descriptive contract-status prose remains stale there; repository
  contract and explicit human ledger control implementation. No Linear mutation.
- Working tree clean before branch creation. Local main ref was preserved;
  implementation branch created directly from fetched approved `origin/main`.

Observed base blobs (`git rev-parse <base>:<path>`):

| Artifact | Blob |
|---|---|
| `docs/specifications/007-configurable-workflows/spec.md` | `0fac282ee371706538319d0d6852a4ededebd75f` |
| `docs/specifications/007-configurable-workflows/amendment-303-workflow-skill.md` | `66897da8deb449ae24acc493d04ba815f1c40dfd` |
| `docs/decisions/0022-dedicated-workflow-conversational-surface.md` | `8c5c187f95b70a4b60637b618efc09369ab9ca48` |
| `docs/specifications/007-configurable-workflows/addendum-303-r10-active-workflow.md` | `1a48cb8c9a39fa06567db505a49d9be1c3e297be` |
| `docs/specifications/007-configurable-workflows/plan-tasks-303-workflow-skill.md` | `ba9db01b346f1890939c5e354f4e0e1481cd68be` |

Accepted original blobs: amendment `9d1895b6ab2dfea7e6c8ab4cd026bb111f31ce97`,
ADR `13a52a62e1273ec8c928f064563d297c3798d152`,
Addendum `14300b70226c807d2648e8e6b30181f23be3cae4`,
Plan `75f689946689cf61bc40872dbb61e4fc44f00e85`.

## T02 — catalog split: PASS (bounded local checks)

- Seven `definition.*` operations exclusively owned by `axiom-workflow`;
  canonical `project_workflow_*` actions, modes and authority strings preserved.
- `workflow.select` remains exclusively Project-owned. Work Item metadata unchanged.
- `configuration.readiness` has three separate read-only modes:
  `configuration` (profile validate/preview), `runtime` (Codex/Claude status),
  `authentication` (Codex/Claude auth). No new executable command or parser.
- SKILL routing tables, catalog/inspect and current #230 matrix block match.
  No aliases or forwarding. Frozen matrix sections unchanged.
- New regression checks cover exclusive command ownership, exact operation names,
  argument/help parity, valid preview inputs and rejected mutation/login flags.
  Existing dispatch probe extended with controlled readiness services.

Validation executed successfully:

| Command/check | Result |
|---|---|
| `go test ./internal/cli ./cmd/lingo -run 'Skill\|Routing\|Catalog'` | PASS |
| `go test ./internal/cli` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| Base-versus-change decoded inspect comparison | PASS |
| `git diff --check` | PASS |
| `./scripts/check-sensitive-files.sh --staged .` | PASS |
| `gitleaks detect --source . --no-git --redact` | PASS; no leaks |

Inspect comparison built an isolated source archive of the base with
`git archive <base>` / `go build -o <base-binary> ./cmd/lingo`, plus the changed
binary with `go build -o <changed-binary> ./cmd/lingo`. Invoked only
`--json skill inspect` on both; excluded completion provenance and compared
all fields of decoded `skill` metadata:

- Project retained operations/commands (including `show`, `list`, `workflow.select`)
  equal the corresponding base metadata exactly.
- Work Item entire `skill` metadata equal the base exactly.
- Migrated definition metadata equal the base except `workflow.` → `definition.`.
- 43 unique command routes across three owners; no duplicated command.
- SHA-256 of sorted-key compact JSON for retained Project `skill`:
  `d6833a4c4db62f20d0e29beedc05afdc230c93f7b7928730f0d392bfe509c030`.
- SHA-256 for unchanged Work Item `skill`:
  `1efd1d09b2f0139298be9629d295c95a7d13ff31adb1493c0e6696ff7964e14a`.

Initial checks caught missing readiness dispatch probes and an existing
instructional-text assertion automatically applied to the new catalog entry.
Probes added; original Project/Work Item assertion coverage retained. Workflow
instructional assertions belong to T03. Final checks above pass.

Engineering/security self-review: no blocking finding within T02. Application,
Execution, parser and authority behavior untouched. Readiness execution tests
use controlled services; no real auth probe, credential access, paid inference
or product Provider effect. Repository publication is limited to the authorized
implementation branch and Draft PR.

## Full implementation execution

The earlier T01/T02 checks above remain historical Evidence. The full execution
preserves those contracts and delivers the following additional scope.

| Task | Implementation and deterministic Evidence |
|---|---|
| T03 | Three complete embedded skills; `workflow_skill_content_test.go` checks schema coverage, ownership, authority and limitations |
| T04 | Runtime/archive/installer/help inventories contain three skills; runtime/bootstrap, POSIX facade and release archive tests |
| T05 | Shared immutable v0.15.0 history and official receipts; [provenance](../../../internal/codexruntime/testdata/published-receipts/v0.15.0/PROVENANCE.md); final embedded digests pinned |
| T06 | Both Runtime official-receipt convergence, reinstall, foreign conflict, historical rollback tests; release upgrade journeys |
| T07 | Installed skill bytes, inspect, help and catalog parity; removed routes fail closed with no alias or forwarding |
| T08 | Additive preservation tests; canonical Project selection and Work Item run/status/plan metadata retained |
| T08a | Exact selected/none/unresolvable read model in Project show/list, JSON/human and mixed list tests; state snapshots prove no writes |
| T09 | Runtime scratch draft journey; validation, separate creation/selection previews and approval, draft drift and refusal snapshots |
| T10 | Offline bounded [R-1 runner](../../../scripts/acceptance/stage-plan-r1.py), closed JSON schema, unit tests and actual canonical test markers |
| T11 | Existing Runtime/Profile/policy/effort/credential failure coverage; additive multiple named Profile test; bounded contract/security review |
| T12 | CLI/install/README/site/index reconciliation, R10 exception and #278 handoff below |
| T13 | Exact clean revision runner, complete checks, independent review and remote head CI recorded in PR validation appendix |

AC-015 rows 1–10 map respectively to: accepted unchanged blobs/index; three-skill
installation/history/journeys; routing/preservation/R10; content/draft/binding;
R-1 A–H report; deferred native handoff; final convergence; protected compatibility;
Profile security; independent review. A row passes only with its actual final
validation record. Until all required checks and review pass, T13 is blocked and
`Completes-Issues: none` applies.

### Regression analysis

The reviewed CI run [38091372395](https://github.com/rgomids/axiom/actions/runs/38091372395)
failed because the partial delivery had not converged embedded skill inventory,
shared published history and installation expectations. Full implementation adds
the third skill throughout the installation paths and the verified v0.15.0 set;
no required check is removed or weakened. The older parser must refuse the new
archive with zero effects; the new installer performs supported convergence.

Full local tests also exposed an obsolete two-skill black-box count, updated to
three. An inherited host authentication variable caused a synthetic test to
observe an additional override: validation uses an isolated environment with API
key variables removed, without reading or printing their values. An ignored
preexisting Claude agent worktree causes repository structure validation in the
primary checkout to reject its nested adapter; final validators use a clean
implementation checkout, preserving that preexisting worktree.

### Reproduction and final records

Run `go test ./...`, `go build ./...`, `go vet ./...`, `go mod verify`,
`scripts/check-go-quality.sh all`, and `scripts/validate-repository.sh .` from a
clean checkout of the final PR SHA. Run the runner unit tests with
`python3 -m unittest discover -s scripts/acceptance -p 'test_stage_plan_r1.py'`.
Run R-1 with `python3 scripts/acceptance/stage-plan-r1.py --source "$PWD"
--target-revision <full-final-SHA> --output <outside-source-report.json>`.
The report is bound to actual test results and the exact clean source revision;
no declared release version is inferred from synthetic artifact builds.

The PR appendix retains the actual R-1 report, final SHA, command exit results,
independent review verdict and final CI run IDs. Temporary host log paths are not
portable acceptance Evidence. Published receipt provenance and test fixtures are
committed; final-run provenance belongs to that exact revision's appendix.

### Explicit limitations and #278 handoff

R-1: use the actual final report result. R-2 and R-3: `deferred_to_278`.
No real vendor session, inference, Provider dispatch or human E2E is claimed.
The additive read model never selects, repairs, defaults or falls back; an
unreadable portable manifest cannot supply an observable reference and reports
unresolvable without guessing. Missing Project directories retain the original
show failure category.

The #278 handoff must supply: exact implementation revision and R-1 report;
Runtime/version and named Profile identity; Project policy intersection;
separate technical and product acceptance; G-2 availability/resolution proof;
G-4 explicit consent for each native effect; native A–H/Report H Evidence and
R-2/R-3 verdicts without inferring real availability from synthetic tests.
#305/#306 retain their configuration owners. The accepted #307 regular-file
`--file` limitation remains documented and unfixed. No Lane L artifact, fixed
model tier, credential mutation, successor implementation, merge, release,
Issue closure, Linear transition or AXM-8 unblock is included.
