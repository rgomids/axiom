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

## CR-002 correction (PR #257)

`net/url` accepts comma, semicolon and equals in a host/reg-name. The previous
prose URL extraction validated and consumed the entire token before inspecting
its surrounding assignments, so host-valid delimiters could hide sensitive
assignments. `safeProseURL` now checks parsed authority components with the shared
`referencePayload` / `sensitiveParameterV1` classifier before consumption. It also
rejects a rooted prose suffix swallowed as a URL path after an authority comma
or semicolon, including Markdown-wrapped suffixes and rooted assignments.
The check is scoped to v3 prose; legacy scalar `safeURL` behavior
is unchanged. URL path/query punctuation remains governed by URL syntax, and
Markdown/quote boundaries continue to retain adjacent prose for inspection.
The package remains pure, with only standard-library dependencies.

`TestPortableProseStructuralSafety` rejects comma/semicolon assignments for
password, token, api_key, client_secret and authorization, case variants
PASSWORD/Api_Key/Access_Token, colon assignments, multiple assignments, URI and
nested escaping, SSH userinfo assignments, rooted authority suffixes and
Markdown/quoted adjacent assignments. Fresh-context attack review reproduced
Markdown-wrapped authority assignments and rooted suffixes; those cases now
have permanent regressions in all three boundary tests. Authority and prose
share punctuation, adjacent-assignment and suffix inspection. It retains
ordinary URL paths/queries,
encoded spaces, Markdown links, non-sensitive authority assignments, SSH git
userinfo, IPv6 hosts and ordinary prose mentioning password/token.
`TestV3PortableProse` applies the bypass matrix to businessContext.text,
glossary.term and glossary.definition at decode and direct domain/encode
boundaries. `TestBootstrapRejectsUnsafePortableProse` applies that matrix to
SetupInput business/glossary fields and proves InvalidContextInput, zero preview,
nil manifest, a nonpublishable proposal and zero storage calls.
`TestOlderSchemaProseCompatibility` now explicitly round-trips these host-shaped
payloads in v1/v2 business text, preserving the historical contract. Existing
goldens remain part of full-suite validation. CR-001 validation and the dogfood
Runtime-unavailable resume/zero-mutation regression are preserved.

Before implementation, the new targeted regression run failed on the vulnerable
head; after implementation, all three affected packages passed uncached.


### CR-002 validation (macOS darwin/arm64, Go 1.26.1, 2026-10-07)

| Command | Result |
| --- | --- |
| `go test ./... -timeout 10m` | PASS |
| `go test -race ./... -timeout 10m` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=linux go build ./...` | PASS |
| `GOOS=windows go build ./...` | PASS |
| `go mod verify` | PASS |
| `./scripts/validate-repository.sh .` | PASS |
| `./scripts/dogfood-poc.sh` | PASS |
| `go run ./scripts/check-architecture.go domain` | PASS |
| `go run ./scripts/check-architecture.go application` | PASS |
| `bash scripts/test-check-project-domain.sh` | PASS |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `gitleaks dir . --no-banner --redact` | PASS |
| `git diff --check` | PASS |

Linux/Windows builds are cross-compilation only. Optional live Runtime behavior
remains SKIPPED/UNVERIFIED; dogfood uses controlled Provider/Runtime stubs.
Final fresh-context security review found no blocking finding after the
adversarial regressions above were incorporated. Delegation was limited to
independent security review; implementation and validation stayed with the
Orchestrator.

### CR-002 authority grammar completion (2026-10-07)

The earlier CR-002 correction at `df9481c2eb34d8f2ff75d1aa6c983990df876189`
covered only part of the authority boundary class. `authoritySubDelimiter`
now centrally defines the complete RFC 3986 sub-delims set (`!$&'()*+,;=`).
`authorityProseDelimiter` combines that grammar with prose wrappers and the
userinfo separator. `=` remains an assignment operator and `:` remains intact
for `referencePayload` / `unsafeProseReference`; no sensitive-name list or
classifier was duplicated. Path/query punctuation is still consumed using URL
syntax rather than split as authority prose.

All eleven sub-delimiters are boundaries for rooted authority-to-path
transitions. Drive prefixes split between authority and URL path are also
checked together, preserving ordinary host/port and IPv6 syntax. Authority
parts remain in surrounding prose inspection so whitespace, Markdown or quotes
cannot detach a sensitive key from its adjacent assignment operator. A bare
mention of a sensitive key still remains valid without an assignment.

Concrete regression evidence:

- `TestPortableProseStructuralSafety`: all eleven delimiters with sensitive
  assignments, SSH userinfo, Unix rooted paths and Windows drive/path
  transitions; required case variants; percent and nested percent encodings;
  multiple assignments; adjacent operators with whitespace, Markdown and
  quotes. Ordinary documentation URLs, path/query punctuation, non-sensitive
  authority assignments, SSH git userinfo, IPv6 and bare key mentions pass.
- `TestV3PortableProse`: the same rejection matrix at all three boundaries
  (`businessContext.text`, `glossary[].term`, `glossary[].definition`), with
  decode rejection and direct domain/encode rejection. Required valid URLs
  round-trip through all three boundaries.
- `TestBootstrapRejectsUnsafePortableProse`: the same matrix proves
  `InvalidContextInput`, zero preview, nil manifest, nonpublishable proposal
  and zero storage calls for all three context fields.
- `TestOlderSchemaProseCompatibility`: original and newly covered authority
  payloads continue to round-trip unchanged in schema v1/v2. Scalar `safeURL`
  and historical goldens remain unchanged; strengthened policy is prose v3.

New tests first reproduced failures against the previous head. Independent
fresh-context security review then reproduced adjacent-operator and Windows
rooted-path bypasses; both classes gained permanent regressions and fixes.
Implementation and integration remain with the Orchestrator; delegation is
limited to independent security attack/re-review with inherited model/effort.

Final independent re-review identified no blocking finding after 462 bounded
adversarial combinations and nested-encoding checks. Earlier attack review
also checked 1,584 delimiter/key/operator/wrapper combinations. These are
local structural checks, not a claim of exhaustive proof or remote CI success.

#### Completion validation (macOS darwin/arm64, Go 1.26.1)

| Command | Result |
| --- | --- |
| `go test ./... -timeout 10m` | PASS |
| `go test -race ./... -timeout 10m` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=linux go build ./...` | PASS |
| `GOOS=windows go build ./...` | PASS |
| `go mod verify` | PASS |
| `./scripts/validate-repository.sh .` | PASS |
| `./scripts/dogfood-poc.sh` | PASS |
| `go run ./scripts/check-architecture.go domain` | PASS |
| `go run ./scripts/check-architecture.go application` | PASS |
| `bash scripts/test-check-project-domain.sh` | PASS |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `gitleaks dir . --no-banner --redact` | PASS |
| `git diff --check` | PASS |

Linux/Windows results are cross-compilation only. Dogfood uses controlled
Provider/Runtime stubs; optional real Runtime scenarios remain SKIPPED/UNVERIFIED.
Remote CI and human re-review are separate from these local results.

## CR-002 responsibility separation (2026-10-07)

Previous head: `e7c5752fb568aef12d718bdbfe479a936bfe3e8d`. Earlier CR-002
corrections and validation above remain historical. Architecture classification:
**internal refactor preserving existing contract**. ADR-0001/0004/0005/0007 remain
unchanged: portable/local separation, schema evolution, filesystem threat model
and publication authority are preserved. No new ADR, persisted format, CLI or
readiness contract. The v3 Specification clarifies the existing no-secret rule.

Root cause: URL-token consumption removed components before the structural
secret policy inspected them. Authority-specific repairs left path/fragment
assignments reachable. New regression cases reproduced those failures at
portableconfig, manifest and bootstrap boundaries before implementation.

Responsibilities now separate inside the pure `internal/portableconfig` package:

- `secret_policy.go` owns the existing sensitive-name taxonomy, historical scalar
  assignment helper and independent `ContainsSecretBearingValue`. The policy
  inspects complete original prose and each bounded percent-decoded layer; it
  does not consume `url.URL` components. Name comparison retains case folding
  and dash/underscore normalization. A sensitive name followed by `:`/`=` is
  unsafe, including an empty payload as in the previous fail-closed policy;
  ordinary words are allowed. Whitespace and prose/Markdown wrappers may connect
  name/operator; reference separators terminate that association.
- `safeProseStructure` owns UTF-8/control checks, machine paths and references;
  `safeProseURL` retains structural URL/machine-reference checks. Historical
  `safeURL` userinfo/query checks and scalar behavior remain unchanged.
- `SafeProse` composes the policies through shared bounded normalization. Each
  full layer reaches secret inspection **before** URL extraction. Neither URL
  consumption nor Markdown splitting defines the secret scanner's input. The
  normalization budget is unchanged (four times original input bytes), and
  exhaustion fails closed. No duplicate decoding pass or sensitive-name list.

Removed complexity: authority parts no longer return to surrounding prose;
`safeProseParts` no longer interprets sensitive assignments; machine-reference
suffix inspection no longer doubles as secret detection. Authority delimiter and
rooted-path checks remain because they protect legitimate structural boundaries.
No network, environment credentials, arbitrary file reads, new dependencies or
runtime Gitleaks integration were added.

Regression Evidence:

- `TestContainsSecretBearingValue`: focused input/expected/reason matrix for
  plain/whitespace/colon assignments, normalization, all URL components, userinfo,
  Markdown/quotes, percent/nested/assembled escapes, multiple assignments,
  normalization exhaustion and benign prose/URLs.
- `TestSecretPolicyAcrossReferenceWrappers`: existing sensitive-name families
  across prose and reference wrappers; `TestProsePolicyComposition` independently
  checks structural acceptance versus secret rejection and structural refusals.
- `TestPortableProseStructuralSafety`: direct `SafeProse` regressions, retaining
  prior machine-path, file-reference and URL authority coverage.
- `TestV3PortableProse`: text, glossary term and definition each reject unsafe
  decode/domain/encode state and round-trip legitimate prose/URLs unchanged.
- `TestBootstrapRejectsUnsafePortableProse`: the same three fields yield
  `InvalidContextInput`, zero preview, nil manifest, invalid/nonpublishable
  proposal and zero publication storage calls.
- `TestBootstrapAcceptsBenignSecurityProse`: those three fields keep valid
  previews, publishability and portable round-trips. Existing multiline checks
  remain. `TestOlderSchemaProseCompatibility` adds path/fragment/nested-escape
  cases without changing v1/v2 semantics or goldens;
  `TestHistoricalScalarURLPolicy` protects scalar URL behavior explicitly.

Orchestrator owns implementation/integration/validation/documentation. Delegation
was limited to one independent fresh-context security review with inherited
model/effort and read-only authority. That review found a false positive: scanning
across `/`/`#` linked a path named `token` to a fragment operator. The general
name/operator association rule was narrowed to prose wrappers; text fragments and
empty-key queries now remain valid, with permanent policy and real-boundary
regressions. Final independent review reported no blocking finding after 144
bounded adversarial combinations and additional encoding/path/benign probes.
This is evidence for declared structural assignments, not a general claim that
all credential patterns are impossible.

### Final local validation

Validation uses macOS darwin/arm64, Go 1.26.0. The first unaligned invocation of
`bash scripts/test-check-project-domain.sh` failed because its
`GOTOOLCHAIN=local` selected Homebrew Go 1.24.5; setting only PATH then exposed an
inherited Go 1.26.1 GOROOT mismatch. Aligning both GOROOT and PATH to Go 1.26.0
passed; no repository workaround or installed-toolchain modification was made.
Linux/Windows builds prove cross-compilation only. Repository validation skips
optional real Runtime behavioral scenarios (SKIPPED/UNVERIFIED); dogfood uses
controlled Provider/Runtime stubs. Remote CI and human re-review are separate.

| Command | Result |
| --- | --- |
| `go test ./... -timeout 10m` | PASS |
| `go test -race ./... -timeout 10m` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=linux go build ./...` | PASS |
| `GOOS=windows go build ./...` | PASS |
| `go mod verify` | PASS |
| `./scripts/validate-repository.sh .` | PASS |
| `./scripts/dogfood-poc.sh` | PASS |
| `go run ./scripts/check-architecture.go domain` | PASS |
| `go run ./scripts/check-architecture.go application` | PASS |
| `bash scripts/test-check-project-domain.sh` | PASS |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `git diff --check` | PASS |
| `gitleaks dir . --no-banner --redact` | PASS |

Remote CI on previous head `e7c5752` was re-read as PASS. CI for this correction
is PENDING until its commit is pushed and the exact new head is checked; earlier
CI is not evidence for the corrected bytes. Human re-review remains pending.
