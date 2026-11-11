# Issue #387 — ancestor ownership diagnostic Evidence

Date: 2026-11-11. Base: `e5b9d10880cfc210356b8c49d639b1718b494e01`.
Branch: `fix/387-codex-sandbox-ancestor-owner`.
Scope/plan/decision proposal: [amendment](amendment-387-ancestor-owner.md).
This is technical Evidence, not MVP acceptance or compatibility approval.

## Reconciliation and implementation

The resumed worktree contained interleaved uncommitted implementations that did
not build. Reconciled to one typed `ErrUnsafe`-compatible refusal, one bounded
category codec, the existing application/presentation boundaries and one amendment
in Specification 004. Removed the duplicate Spec 015 note and incomplete global
CLI preflight. Ownership, mode, sticky-directory, link, identity and publication
rules remain unchanged. No state formats, dependencies, CLI flags or brokers added.

The shared POSIX ancestor check reports `filesystem_root` or `ancestor`, numeric
UID and `mapped`/`unmapped`/`unknown` UID-map observation. Installation inspection,
selection, readiness/admission, Project catalog/portable validation and Runtime
configuration preserve the cause. CLI results use the existing diagnostic envelope;
no private pathname or repair/reconfiguration suggestion is emitted.

## Deterministic and manual Evidence

Injected file-stat ownership covers trusted root/euid, foreign ownership with
ordinary/sticky modes, writable root rejection and sticky root acceptance. The
injected rooted walker rejects before creation; InstallationStore Inspect/Select/
ListInstalled retain the cause. Category parsing rejects private path classes and
malformed facts; UID-map cases distinguish mapped/unmapped/unknown. Application
regressions cover catalog/lifecycle and Runtime authoring cause preservation.
CLI tests render mapped results through the real JSON and text presentation path.

A locally built binary was run inside `bwrap --unshare-user --ro-bind / /`
with synthetic temporary HOME, `LINGO_STATE_ROOT` and `LINGO_PROJECTS_ROOT`.
The observed filesystem root UID was 65534 and its UID-map observation unmapped.
All 16 command/format combinations returned the specific refusal, exit 1, bounded
facts and guidance without misleading categories, private paths or repair advice:

- Project show (`--selector`), validate (`--slug` and `--project`), list;
- Runtime Profile list and validate;
- Work Item list and Workflow list (`--project`);
- each in terminal and JSON formats.

Both synthetic roots remained empty. Initial manual testing found legacy
`project validate --slug` still erased the cause; corrected its existing storage
mapping and repeated the entire matrix successfully. No model, authentication
probe or real Axiom state was used. Before-patch manual sandbox repro: NOT_RUN;
the reported baseline and source cause were read from #387 and its linked Evidence.

## Validation matrix

Final results are recorded after command completion; no pending command is PASS.

| Check | Result | Evidence / scope |
| --- | --- | --- |
| Build | PASS | final `GOMAXPROCS=2 go build ./...`; locally built CLI also passed synthetic bwrap matrix |
| Vet | PASS | final `GOMAXPROCS=2 go vet ./...`, exit 0 |
| Targeted tests | PASS | local/projectapp/runtimeapplication `-run 'Ancestor\|UIDMap' -count=1 -p=1`; final compiled CLI test binary `TestAncestorOwnerDiagnostic` passed JSON/text presentation |
| Full suite | PASS | final source `go test ./... -count=1 -p=1 -timeout=40m`, exit 0, 32 packages ok, no FAIL |
| Repository validator | PASS | `./scripts/validate-repository.sh .` with `PATH` excluding vendor CLIs, exit 0 (see note below) |
| Sensitive files | PASS | worktree and staged `./scripts/check-sensitive-files.sh`, exit 0 |
| Formatting | PASS | final `gofmt -l` empty on all changed/new Go files |
| Diff whitespace | PASS | `git diff --check` |
| Installed docs generation | PASS | existing `axiom-docs-build`; 26 outputs, inventory/provenance hashes generated in a temporary directory |
| Manual namespace matrix | PASS | 16 refusals, synthetic roots unchanged |
| Native macOS/Windows | NOT_RUN | no host execution in this task; CI is separate Evidence |
| Gitleaks | NOT_RUN | tool unavailable; required sensitive-file checker executed |

## Earlier runs and failure triage

The first full invocation was started while diagnostic tests were still being
corrected. It reported the old presentation-test failure, the missing replaced
skill revision (corrected and its ownership test rerun PASS), a 10-minute CLI
package timeout in an unrelated black-box command test, and
`TestPOSIXProcessContainment/decoupled_orphan_in_another_attempt_group` failing
with “descendant did not start”. The obsolete failing invocation was interrupted
with SIGTERM before completion; it is FAIL/incomplete, never PASS. One intermediate
targeted invocation also ended with signal 143; final targeted results above are
from completed runs.

The containment subtest's first isolated rerun also failed. A subsequent isolated
run of the entire `TestPOSIXProcessContainment` passed in 6.562 seconds without
changing that package. Host CPU/memory pressure and swap use were observed;
scheduling pressure is an inference, not a proven new product defect. Final full
validation uses `-p=1` and `-timeout=40m` on the final source.

Validator note (pre-existing, unrelated to this change): with the default host
`PATH`, `scripts/test_model_advisor_eval.py::test_unavailable_runtime_exits_nonzero`
failed (`0 != 1`). The test removes only its stub directory from `PATH`, so an
installed `claude` CLI elsewhere on `PATH` stays resolvable and the
"unavailable runtime" premise does not hold; that run may have invoked the real
CLI once. The rerun with `PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin`
passed. The test isolation defect is a deferred finding for a separate card.

## Review correction CR-392-F1

Review of `c938fed` found that `project list` still erased the cause when the
installation records were readable but an installed Project's independent
portable source sat under a foreign-owned ancestor: `ReadInstalledProject`
returned bare `ErrUnsafe` and the catalog reported `invalid_project_state`
with repair advice. (The manual matrix above did not cover this split, because
its state root shared the untrusted ancestor and `ListInstalled` refused first.)

The adapter now returns the typed `AncestorOwnerError` for its source root and
`ProjectCatalog.ListProjects` maps any `StorageCategory()` refusal before the
generic unsafe mapping. Other unsafe sources keep `invalid_project_state`. The
only additions are an unexported source-root opener on `InstallationStore`
(nil in production, same `existingPrivateRoot`) and extraction of the existing
list presentation into `projectListResult`, both for deterministic tests.

- `internal/local` `TestProjectListPortableSourceAncestorOwnerRefusal`: real
  store and catalog with an injected ancestor checker; bounded category,
  no private directory name, state root unchanged; generic unsafe stays
  `invalid_project_state`.
- `cmd/lingo` `TestProjectListPortableSourceAncestorOwner`: catalog → list
  presentation → `project list` in text and JSON; refusal facts and
  troubleshooting link present, no repair/reconfigure/invalid/private path;
  generic unsafe still renders repair advice.
- Both tests were observed failing with the respective fix disabled.

### Correction validation

| Check | Result |
| --- | --- |
| Targeted tests | PASS (`internal/local`, `internal/projectapp`, `cmd/lingo` ancestor tests, `-count=1`) |
| Full suite | PASS (`go test ./... -count=1 -p=1 -timeout=40m`, exit 0, 32 packages ok, no FAIL) |
| Vet / gofmt | PASS (`go vet ./...` exit 0; `gofmt -l cmd internal` empty) |
| Repository validator | PASS (`validate-repository.sh .` with vendor CLIs excluded from `PATH`) |
| Sensitive files / diff whitespace | PASS (`check-sensitive-files.sh`; `git diff --check`) |
| Manual namespace repro of the split case | NOT_RUN (covered by injected checker; no chown/sandbox needed) |

## Review and remaining decision

Ponytail/Axiom engineering and security self-review found no blocking defect in
the diagnostic change. Assume the existing local CLI/process concurrency model.
No trust exception, shortened ancestor chain or new mutation path is introduced;
UID-map facts never grant authority. Existing generic unsafe/recovery categories
remain for their original cases. Documentation/commands and all four shipped
Runtime skills now explain the refusal; release tooling generates installed-doc
metadata from those sources instead of committing generated bundles.

**DECISION_PENDING (Rafael):** restricted Linux sandbox compatibility remains
unresolved. A writable root retains the same full ancestor chain; unmapped UID
cannot prove host root; an euid-only anchor omits replaceable ancestors; an external
broker needs its own authentication, object authority and lifecycle contract.
Recommend a separately reviewed narrow broker design while retaining fail-closed
ownership. ADR-0005/0007 and #265's explicitly approved Windows solution do not
permit a POSIX exception; ADR-0028 changes process supervision only. No ADR is
accepted here. Use `Refs #387`, not a closing keyword. No acceptance, merge,
release, status change or real vendor validation is claimed.

Compatibility proposal published as [DECISION_PENDING (Rafael)](https://github.com/rgomids/axiom/issues/387#issuecomment-6796782134).
