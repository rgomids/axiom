# Issue #303 — T01/T02 implementation Evidence

Scope: GitHub [#303](https://github.com/rgomids/axiom/issues/303) / AXM-7.
T01 and T02 only; #303 is not complete. No technical/product acceptance inferred.

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

## Limits and next action

- Delegation not used: coupled catalog/table changes are cheaper to integrate
  sequentially; deterministic checks suffice for the bounded review.
- Minimal new SKILL file only. Existing Project definition prose deliberately
  untouched under T02's tables/operations-sentence scope; T03 must reconcile it
  and add the full instructional content before product use.
- Installer inventories and global skill-name help remain at the preceding
  two-skill set until T04; upgrade/history/receipt proof belongs to T05–T07.
  This Draft PR proves catalog/inspect routing, not installed discovery.
- T03–T13, including T08a `activeWorkflow`, NOT_RUN/not implemented here.
  Full #303 R-1 acceptance runner not delivered; no overall R-1 completion claim.
  R-2/R-3: `deferred_to_278`. G-1/#305, G-2/#306, G-4 consent and
  `--file` hardening #307 remain with their owners.
- Next recommended task: T03, after Code Review of this bounded T02 commit
  and separate implementation authority. Request review of ownership,
  metadata compatibility, readiness argument parity and scope boundaries.
- Stop here: no merge, release, Issue closure, Linear transition or AXM-8 unblock.

Resume from the Draft PR branch/HEAD, recheck main and review state, read this
record and the accepted Plan/T03. Do not treat this partial delivery as T13.
