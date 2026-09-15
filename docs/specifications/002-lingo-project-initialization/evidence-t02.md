# T02 — Implementation Evidence

## Authority, baseline and delivery — 2026-09-14

- Specification / Plan / Tasks: **Approved**. Human authorization: **T02 only**,
  conditional on PR #6 being merged into `main`.
- `gh pr view 6 --repo rgomids/axiom --json number,state,mergedAt,mergeCommit,baseRefName,headRefName,url`
  returned exit 0, `MERGED`, base `main`, merge time `2026-09-14T19:58:54Z`.
- Baseline SHA: `ffc9af9b7158de62f94671472c9ddedf36439802`, merge of
  [PR #6](https://github.com/rgomids/axiom/pull/6). `git pull --ff-only origin main`
  returned exit 0; `git merge-base --is-ancestor` confirmed T01 incorporation.
  Checkout was clean on `main`; new branch `agent/spec-002-t02` starts at this SHA.
- **Delivery SHA (implementation, tests and boundary checkers):
  `5e18a482105627cbb5935456b01411b36957e260`.** The subsequent documentation-only
  commit adds this Evidence and lifecycle reconciliation without changing delivered
  code. This separation makes the tested code revision exact and non-self-referential.
- **T01: Accepted / merged. T02: Ready for human implementation review.
  T03–T21: Not started.** No merge, release or next-Task authority is inferred.

Read in full: AGENTS.md, Constitution, Specification 002, clarifications including
H1–H11, approved Plan, Tasks, T01 Evidence, ADR-0001/0003/0004,
architecture/quality/security/documentation policies and existing `internal/project/`.
Current explicit authorization supersedes earlier lifecycle gates only.
T01 Evidence, approved contract bodies, ADRs, domain code and frozen experiments
remain unchanged. No durable architecture decision beyond approved boundaries arose.

## Delivered contracts

[`internal/projectapp/`](../../../internal/projectapp/) owns its ports and imports
only domain and allowlisted standard-library operations. Domain imports neither
application nor infrastructure. No concrete adapter, parser dependency or CLI exists.

- **Complete snapshots:** `ReadSnapshot` decodes via injected `ManifestCodec`,
  requires a valid complete domain value and all referenced documents, and detaches
  byte/slice inputs. It rejects duplicate/ambiguous relative identifiers and a
  document masquerading as `axiom.yaml`. Full manifest/document content is retained;
  stores never receive `project.Intent`, manifest patches or unrestricted path writes.
- **Exact revisions:** distinct portable/local types distinguish zero-invalid,
  explicitly missing and present content. Portable revision hashes length-framed
  manifest plus sorted document names/content using SHA-256; local revision hashes
  exact observed record bytes. Per-artifact digests support metadata-only records.
  Digests detect changes, not identity, authenticity or secret absence. Restoring a
  recorded portable digest does not establish current freshness.
- **Destinations:** opaque, comparable in-process capability identities; no writable
  filesystem path in either writer method. Trusted future adapters bind identities
  to inspected directory objects and keep bindings protected. Root discovery or
  constructing an identity grants no approval. Local destination cannot equal its
  portable source identity; physical alias/overlap checks remain future work.
- **Authority:** private immutable preview owns operation, complete before/after
  intent, expected revisions, old/new destinations and exact replacement/removal
  set. Create/move require an absent target; update preserves ID and destination,
  move requires explicit changed slug and destination. Local preview binds the
  Project ID, source revision, expected record revision and full local proposal.
  Trusted human/system confirmation refers to that exact preview; every fresh
  preview needs new consent. Revocation is shared across copies using an atomic
  flag. No AI or Git approver/operation is exposed.
- **Submission:** `AuthorizePortable`/`AuthorizeLocal` create sealed requests only
  after deterministic checks. `ApplyPortable`/`ApplyLocal` are small contract gates:
  no port calls on denial, no ambient reads or metadata allocation, and cancellation
  before submission prevents the call. They do not implement init/update/install.
  Future writers must recheck permits and revisions, including the separate absent
  move/create target, under their actual commit protection. A permit is not a lock.
- **Local metadata and observations:** source capability, Project ID/slug, content
  and artifact digests, reference metadata, bindings, presence observations and
  attempt metadata are separate from portable state. No secret-value field or
  credential-reading port. Checkout/Runtime observations have no execution methods;
  presence never proves authentication, capabilities, model or workflow readiness.
  Clock and identity allocation are injectable, without ambient allocation here.
- **Safe issues / optional inspection:** fixed phase/field/code/index vocabulary;
  category, severity and remedy derive from fixed codes. Ordering is deterministic
  by phase, field, code and index, including unknown enum values. Optional scanner
  finding/unavailable/failure produces warnings; it cannot invalidate configuration
  solely through unavailability or certify secret absence. No scanner is adopted.
- **Outcomes:** portable and local `MutationResult`s, readiness and actual commit
  status remain independent. Zero commit status means unknown/recovery required.
  Pre-commit failure, confirmed mutation, confirmed portable mutation with local
  failure, no-op and recovery-required uncertainty are distinct. Known committed
  state survives error/cancellation; no rollback is reported. A local success does
  not hide a failed portable mutation. Commit status means Project persistence,
  not Git commit; no Git result participates in local success.

These are internal, reversible representations within Plan §1/§4/§6/§7/§9,
not a public wire API, durable grant protocol or persistence mechanism.
`Confirm` assumes trusted presentation/system policy has actually obtained consent;
this in-process contract does not authenticate a principal or sandbox malicious
linked code. T05/T06/T07 must prove commit-time revocation/concurrency/confinement;
T17 must connect human interaction to this explicit confirmation boundary.

## Task → requirements → executable Evidence

All test names below are in `internal/projectapp/`, use external `projectapp_test`
consumers and synthetic in-memory fixtures. They cover T02 portions of the listed
ACs, not full Specification acceptance.

| T02 component / traceability | Executed test or check |
|---|---|
| Exact authority, no caller bypass; FR-010/011/018/019; SEC-002/004; AC-15; H5/H9; ADR-0003/0004 | `TestAuthorityDenialCallsNoStore`, `TestAuthorityBindsCompleteWriteSetIntentOperationAndDestinations`, `TestRevocationSharedAcrossAuthorityCopies`, `TestPreviewRejectsInvalidOperationIdentityAndScope` |
| Portable/local authority separation and no implicit Git; FR-021; SEC-002; AC-10/18; H7/H11 | `TestLocalAuthorityCannotAuthorizePortableAndViceVersa`, `TestDeniedEffectsAndCancelledGate`; AST import/symbol checker |
| Complete snapshots, expected revisions and removal/move write set; FR-018; SEC-004; AC-15; H9/H10; Plan §4/§6 | `TestSnapshotsAreCompleteDetachedAndRevisioned`, `TestWriteSetContainsDeletionsAndMoveDestinations`, `TestRevisionMetadataCanBeRecordedWithoutBodiesOrFreshnessClaims` |
| Reference-only metadata and inspection warnings; FR-008; SEC-001; AC-08; Plan §1/§7/§9 | `TestLocalSnapshotKeepsReferenceMetadataDetached`, `TestDeterministicSafeIssueOrderingAndInspectionWarnings`, `TestBoundaryFailuresCannotBeReinterpretedAsSuccess`; denied-secret-read trap and AST check |
| Safe deterministic results; FR-016; AC-12; Plan §7 | `TestDeterministicSafeIssueOrderingAndInspectionWarnings`, `TestOutcomeClassificationPreservesActualCommit` |
| Actual pre/post/unknown commit, partial local failure; FR-013/021; SEC-004; AC-07/12/18; H7/H10/H11 | `TestOutcomeClassificationPreservesActualCommit`, `TestInjectedMetadataAndFaultResults`, `TestBoundaryFailuresCannotBeReinterpretedAsSuccess` |
| Fault, entropy/clock and concurrency seams; FR-013/018; SEC-004; AC-07/15; Plan §9 | `TestInjectedMetadataAndFaultResults`, `TestBarrierSeamSupportsConflictAndLateRevocation` |
| No application filesystem writes, process/network/secret reads; SEC-001/002; AC-08/10/18; inward imports; ADR-0003 | `TestDeniedEffectsAndCancelledGate`, all denial matrices; `check-projectapp.go` and eleven negative checker fixtures; unchanged domain checker and its seven negative fixtures |

`portableSpy`/`localSpy` capture calls and inject pre/post/unknown commit outcomes.
`deniedEffects` has separate write/process/network/secret counters behind a denied
writer trap. `fakeEntropy` substitutes identity allocation; `fakeClock` supplies a
fixed instant. `barrierStore` pauses submissions with channels, then checks a shared
revision and permit: one synthetic CAS wins; revocation while both are paused
prevents either commit. No sleeps, processes, filesystem writes or real entropy
are used by these application scenarios. These fixtures are colocated test helpers
for extension by downstream Tasks, not fake production infrastructure.

## Reproduction and results

Platform: **macOS 26.6.2 (25G83), Darwin arm64**.
Toolchain: **`go version go1.26.1 darwin/arm64`**. Existing Go 1.26 module unchanged;
standard library only; no tools or dependencies installed.

Run from repository root:

```bash
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go test -cover ./...
go test -race -shuffle=on -count=10 ./...
go vet ./...
go build ./...
go run ./scripts/check-project-domain.go
bash scripts/test-check-project-domain.sh
go run ./scripts/check-projectapp.go
bash scripts/test-check-projectapp.sh
./scripts/validate-repository.sh .
for script in scripts/*.sh; do bash -n "$script" || exit; done
git diff --check
git diff --cached --check
./scripts/check-sensitive-files.sh --staged .
```

| Executed check | Exit / observed result |
|---|---|
| First contract tests before implementation | 1, expected: no non-test Go files in `internal/projectapp` |
| `go test -cover ./...` | 0; domain **99.6%**, application **100.0%** statement coverage |
| `go test -race -shuffle=on -count=10 ./...` | 0; both packages pass all ten runs |
| `go vet ./...` / `go build ./...` | 0 / 0; no executable CLI produced |
| Domain source boundary / regression fixtures | 0 / 0; six source/test files; seven forbidden fixtures rejected |
| Application source boundary / regression fixtures | 0 / 0; seven source/test files; eleven forbidden fixtures rejected, never executed |
| Repository validator | 0; harness/package checks, existing regression suites, worktree sensitive-file and whitespace checks |
| Shell syntax / worktree and staged whitespace | 0; all root scripts and both diffs |
| Staged sensitive-file scanner | 0; implementation and documentation staged content inspected separately |
| Baseline preservation / documentation links and gates | 0; domain, module, T01 Evidence, ADRs, experiments and approved normative bodies unchanged; local links/gates checked |

Initial development also caught and corrected a test variable type mismatch before
passing the suite. Final self-review corrected local/portable destination overlap
by capability identity and prevented local success from hiding portable failure;
regression tests cover both. A revision-metadata round trip is tested so later local
codecs can store digests without reconstructing bodies or claiming freshness.
No unresolved blocking engineering/security finding was identified for T02 scope.

## Explicit limitations and next gate

- **Not validated:** any real filesystem commit, confinement, symlink/hard-link/
  ancestor/TOCTOU race, permission/ACL, crash recovery, durability, two-process
  protocol or Linux behavior. The in-memory CAS/barrier is not filesystem Evidence.
- **Not delivered:** strict YAML/local codec, parser choice, complete structural
  secret validation, physical artifact allowlist, native roots, CLI, use cases,
  Git backing, checkout inspection, Runtime/Provider/AI execution, secret resolution
  or T03+. `ReadSnapshot` trusts the injected codec's contract; local construction
  validates identity/source/revision and detaches the proposal, not T04's full
  metadata/wire security rules. Future codecs/use cases/adapters must enforce the
  remaining validation before any real write. No production adapter can be wired
  safely from T02 alone.
- Destination identities and preview grants are process-local and nonserializable.
  Adapter binding, safe destination display, authenticated/durable/retry authority
  and commit/revocation linearization remain future implementation obligations.
  No lock, staging layout, syscall protocol, transaction engine, Git authority
  format, transport or automation decision was taken here.
- Application tests and allowlisted source inspection prove no application I/O
  routes in this slice. They are not an OS sandbox or syscall interception:
  Go compiler/test runner and standalone verification tools perform their own
  filesystem/cache/report I/O. No application black-box CLI coverage is claimed.
- `gitleaks`, `markdownlint`, `markdownlint-cli2`, `lychee` and `shellcheck` are
  unavailable. Dedicated secret/lint/external-link coverage remains unverified.
  Local scanners and reviewed synthetic fixtures do not prove absolute absence
  of secrets. Coverage percentages do not prove correctness.
- Documentation updates reconcile T02 only; T21 is not started. Historical T01
  Evidence and approvals remain preserved. Actual acceptance is human-owned.

**Stop at human implementation review. Do not merge or authorize T03/T04/T05.**
