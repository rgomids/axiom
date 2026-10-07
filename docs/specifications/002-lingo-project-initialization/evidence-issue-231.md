# Issue #231 — implementation Evidence

Contract: [project-bootstrap-v3.md](project-bootstrap-v3.md). Base revision:
`419523c` (`main`, release 0.7.0). Implementation is local and unpublished:
no push, PR, Issue update, release or Provider mutation was performed.
Implementation does not imply human acceptance or Issue closure.

## Task completion

| Task | Result | Evidence |
| --- | --- | --- |
| T231-01 | Additive contract frozen; Spec 002 spec/plan/index reconciled; no ADR change | [contract](project-bootstrap-v3.md) §8 |
| T231-02 | Portable schema v3 domain + strict codec | `internal/project/context.go`, `context_test.go`; `internal/manifest/context_v3_test.go`, `testdata/v3-*.yaml` goldens |
| T231-03 | Local installation record format 2 (documentation bindings) | `internal/local/{codec,dto,mapping,validation}.go`, `documentation_record_test.go`, `documentation_inventory_test.go`, `testdata/documentation-v2.json` |
| T231-04 | Read-only Repository/Git remote discovery | `internal/projectdiscovery/repository.go`, `gitconfig.go`, `repository_test.go` |
| T231-05 | Name-only technology discovery | `internal/projectdiscovery/technology.go`, `technology_test.go` |
| T231-06 | Documentation resolution adapters | `internal/local/readiness_adapters.go`, `readiness_adapters_test.go`, `file_identity*.go` |
| T231-07 | Guided bootstrap proposal (`project configure` CREATE) | `internal/projectapp/setup.go`, `bootstrap_test.go` |
| T231-08 | Canonical readiness evaluator | `internal/projectapp/readiness.go`, `capability.go`, `readiness_test.go`; `internal/runtimeapplication` `Availability`, `availability_test.go` |
| T231-09 | Pre-effect enforcement | `cmd/lingo/readiness.go` `preflight`; call sites in `cmd/lingo/main.go`; `TestBlockedPreflightPerformsNoEffects` |
| T231-10 | CLI/operator surfaces | `internal/cli/bootstrap_input.go`, `readiness.go`, help; `cmd/lingo/bootstrap_readiness_test.go`; `docs/commands.md` |
| T231-11 | Regression, security validation, reconciliation | this document; validation below |

## Acceptance criteria → Evidence

| #231 acceptance criterion | Evidence |
| --- | --- |
| Bootstrap from local repository locations without CWD identity | `TestBootstrapFromExplicitRepositoriesResolvesAmbiguityAndSeparatesLocalState` (CWD set to another checkout whose remote never appears); `TestDocumentationResolverRepositorySources` (CWD not a fallback) |
| Candidate remotes discovered, never silently canonical when ambiguous | `projectdiscovery` repository tests (aliases collapse, HTTPS≠SSH, `origin` sorted second and not preferred, include/insteadOf/continuation → incomplete); `TestBootstrapRemoteDiscoveryRules`; black-box refusal of a blocked publish even with digest + authority |
| Work Item capability/provider configured during setup | `TestBootstrapPreviewIsDeterministicAndBounded`, `TestSetupProposalCapabilityAndEffectsAreExplicit` (ready/missing/unsupported) |
| Portable config has no absolute paths or secrets | `TestV3RejectsMalformedContext` (`technology path/secret/file`, `source absolute/traversal/secret path`); `TestV3PortableProse` (business text, glossary term and definition: secret/path/file/escaping/sensitive URL rejection, safe prose round-trip); `TestBootstrapRejectsUnsafePortableProse` (`InvalidContextInput`, zero proposal/preview/manifest and no publication storage calls); black-box manifest scan for workspace paths, notes path/content, CWD remote, `codex` |
| Technology context deterministic, reviewable, editable | `projectdiscovery` technology tests (bounds, links ignored, depth, conflicts, determinism, no `os/exec`/`net`/`syscall` import guard); `TestBootstrapTechnologyProposalsAreEditable` (Evidence per Repository, remove/replace, digest covers edits) |
| Documentation sources registered and resolved without re-entry | `TestBootstrapDocumentationContextAndGlossary`; black-box: local binding in installation format 2 and `project validate` resolving both sources; stale replacement warning |
| Business context and glossary persisted as bounded context | `TestSchemaV3RejectsInvalidContext`, `TestV3CollectionBounds`, `v3-configured` golden; black-box manifest contains `glossary:` |
| `project validate` reports readiness, warnings and exact blockers | `TestReadinessMatrix`, `TestReadinessRenderingIsBoundedAndStable`, black-box `partial` report with `runtime_policy_unavailable` |
| Incomplete Projects fail only unsatisfiable capabilities | `TestReadinessMatrix` (missing Runtime policy blocks only `execution`; documentation warnings never block); `TestBootstrapAuthorsRuntimePolicyFromLocalCandidatesOnly` (executable removed → `partial`, Work Item still ready) |
| Tests: multi-repo, multiple remotes, missing bindings, missing capability/provider, documentation resolution, incomplete readiness | `TestBootstrapMultiRepositoryAndDerivedKeys`; remote fixtures; `repository binding missing` / `repository unavailable` / `capability mapping missing` / `provider unsupported` / `two unconventional integrations are ambiguous` / `credential binding absent warns without blocking` readiness rows; resolver tests; incomplete-readiness rows |

## Matrices (summarized; full cases in the named tests)

**v3 compatibility.** v1 and v2 goldens unchanged; v3 fields rejected in v1/v2
(`unknown_key` at codec, `unsupported_field` in domain); `schemaVersion: 4`,
quoted, float or prefixed versions rejected; v3 Runtime policy validation equals
v2 (`TestSchemaV3RetainsV2RuntimeSemantics`); `absent`/`unconfigured`/`[]`/list
round-trip (`TestV3DeclarationStatesRoundTrip`); readiness on a v1 Project leaves
bytes untouched (`TestReadinessReadsOlderSchemasWithoutRewriting`).

**Portable vs local.** Absolute Repository paths, local-file paths and file
identities exist only in the installation record; a record without documentation
bindings is still written as byte-identical format 1
(`TestFormatOneStaysFormatOneWithoutDocumentation`); format 2 rejects bodies,
content digests, relative paths, malformed identities, duplicates and empty
arrays; format 1 with bindings is malformed; format 3 is `unsupported_newer`.

**Remote discovery.** local-only, single, alias collapse, ambiguous (incl.
HTTPS vs SSH), incomplete, unsafe `.git`/config links, gitdir/commondir
worktrees, oversize/malformed config, password/local-path/query/fragment URLs
counted as `unsupported` and never echoed.

**Technology.** Go; Node+TypeScript+pnpm; npm+pnpm conflict flagged; Docker +
Compose; Terraform at depth ≤2 only; GitHub/GitLab workflows; skipped
`node_modules`/`vendor`/hidden directories; links ignored; directory named like a
signal ignored; >256 directories and >4096 entries bounded; no file content read.

**Documentation resolution.** Repository file/directory available; missing;
traversal → unsafe; symlinked component or leaf → unsafe; unusable Repository →
`repository_unavailable`; local-file bound/unbound/missing/replaced (`stale`)/
link (`unsafe`); content never read and never returned.

**Capability / credential / Runtime.** ready; missing; unsupported (`linear`);
ambiguous (two unconventional Integrations); conventional `work-items` pair kept
with an extra Integration (pre-#231 compatibility); credential bound/unbound
(unbound is a warning; no item reference or source kind in output); Runtime: policy absent → `runtime_policy_unavailable`
without observation; `no_allowed_match`, `runtime_unavailable`, disabled,
model mismatch, missing observer → `runtime_resolution_blocked`; v3 identical to
v2; bootstrap rejects non-local Runtimes (`codex` not configured locally) and
never defaults one.

**Pre-effect.** For Projects lacking the capability (`""`) or with an
unsupported Provider (`linear`): Work Item create/select/comment, workflow start
and resume all fail with the readiness code as category and a `preflight`
payload; the Provider stub records zero calls; Project roots, state root and
workspace are byte-identical before/after; `project validate` reports the same
code. Ready + no authority keeps existing authority denial
(`TestExecutableMinimalLifecycleAndFailurePaths`); ready + authority keeps the
existing effect path (same test, dogfood suites).

## Security assertions

No `os/exec`, `net` or `syscall` in `projectdiscovery` (import guard); Git
metadata parsed, never executed; no global/system Git config read; readiness and
bootstrap previews scanned for workspace paths, document bodies, credential item
references; manifest scanned for local paths and content; `gitleaks` clean.

## Supported platform limitations

Validated on macOS (darwin/arm64) by the test suite; Windows builds compile
(`GOOS=windows`) but Windows-specific symlink tests are skipped and file identity
on Windows (`file_identity_windows.go`) is compiled, not exercised here. Linux was
not exercised in this session. No live GitHub, Codex or Claude Runtime was
exercised; Provider and Runtime behavior uses the repository's existing stubs.

## Independent review

A fresh-context reviewer found no Blocker and no security leak. One Major was
fixed before completion: an unbound declared `credentialRef` would have blocked
existing Projects that GitHub authenticates through `gh`, with no command able to
bind it; it is now a warning (contract §4). Four Minors were fixed: discovery
skips non-portable Evidence names (`TestNonPortableEvidenceNamesAreNeverProposed`),
the EDIT preview uses `ResolveCapability`, the conventional `work-items`
Integration keeps the pre-#231 mapping when several Integrations declare
`work-item`, and the portable inventory treats supported versions 1–3 that fail to
decode as malformed rather than newer
(`TestPortableManifestVersionWindowIncludesSchemaThree`).

## Validation (macOS darwin/arm64, Go 1.26, 2026-10-07)

| Command | Result |
| --- | --- |
| `go test ./... -timeout 10m` | pass (no failures) |
| `go test -race ./... -timeout 10m` | pass (no failures, no data race) |
| `go vet ./...` | pass |
| `go build ./...` (native, `GOOS=windows`, `GOOS=linux`) | pass |
| `go mod verify` | all modules verified |
| `./scripts/validate-repository.sh .` | pass; runtime behavioral scenarios skipped (not requested; UNVERIFIED) |
| `git diff --check` | clean |
| `gitleaks dir . --no-banner --redact` | no leaks found |

## CR-001 correction (PR #257)

The initial Evidence and review above are historical. CR-001 showed that plain
v3 business/glossary strings bypassed structural portable safety despite their
text bounds. `internal/portableconfig` now owns the existing scalar policy and
sensitive-name classifier without YAML/domain/application dependencies. The
codec retains that exact scalar policy for v1/v2; only v3 prose opts into
`SafeProse`. `project.PortableContextValue` combines the existing UTF-8, bounds,
whitespace and control-character rules with that shared policy. Bootstrap checks
all three fields before putting context in preview/state; domain validation
also rejects direct v3 construction, and the v3 shape protects decode/encode.
No sensitive-name list was duplicated.

`TestV3PortableProse` covers `businessContext.text`, `glossary.term` and
`glossary.definition`: token/password/api-key assignments, POSIX/Windows paths,
`file:` references, nested URI escaping, sensitive URL queries and unsafe values
embedded in prose. Normal prose, multiword terms, ordinary URLs and literal
percent signs round-trip; text/definitions accept multiline. Direct unsafe v3
state fails `project.New` and produces no encoded manifest.
`TestBootstrapRejectsUnsafePortableProse` proves `InvalidContextInput`, an invalid
and nonpublishable proposal, a zero `SetupPreview`, nil `Manifest`, and zero
storage calls from `PublishConfigured` with the rejected Project.
`TestPortableProseStructuralSafety` additionally covers JSON/whitespace-separated
assignments, URL passwords, encoded sensitive query names, rooted assignments,
UNC/home paths, encoded controls and excessive nested escaping.
`TestBootstrapAcceptsPortableProse` verifies exact multiline business/definition
text and multiword terms in preview and an equivalent manifest round-trip.
`TestOlderSchemaProseCompatibility` keeps previously accepted v1/v2 text readable
and round-trippable; existing fixture/golden tests verify unchanged older bytes.

Independent correction review reproduced Markdown-wrapped secrets/paths and
sensitive query parameters hidden after commas; regressions now reject them at
both codec/bootstrap boundaries. A further review found rooted colon assignments
and safe URL fragments misread as paths; the policy now inspects colon/equal
suffixes and checks complete URLs before tokenizing only surrounding prose.
Adjacent prose after Markdown links is retained and inspected; regression cases
cover a secret/path immediately after a link without whitespace.
The architecture profiles were stale at the original PR head (pure v3 symbols
missing); their explicit allowlists were reconciled. Local variable selectors
are distinguished from imported package selectors, with positive/negative
fixtures preserving enforcement.

### Correction validation (macOS darwin/arm64, Go 1.26.0, 2026-10-07)

| Command | Result |
| --- | --- |
| `go test ./... -timeout 10m` | PASS |
| `go test -race ./... -timeout 10m` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=linux go build ./...` | PASS (cross-compile only) |
| `GOOS=windows go build ./...` | PASS (cross-compile only) |
| `go mod verify` | PASS |
| `./scripts/validate-repository.sh .` | PASS; optional live Runtime behavioral scenarios SKIPPED/UNVERIFIED |
| `./scripts/dogfood-poc.sh` | PASS; no dogfood source changes |
| `go run ./scripts/check-architecture.go domain` | PASS |
| `go run ./scripts/check-architecture.go application` | PASS |
| `bash scripts/test-check-project-domain.sh` | PASS with Go 1.26.0 GOROOT/bin in PATH and matching GOROOT |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `./scripts/check-sensitive-files.sh --staged .` | PASS |
| `gitleaks dir . --no-banner --redact` | PASS |
| `git diff --check` | PASS |

Fresh-context review reproduced and drove the boundary regressions described
above; no blocking security finding remained after correction. Live Provider
and Runtime behavior and Linux/Windows execution remain UNVERIFIED.
