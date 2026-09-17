# T03 — Implementation Evidence

The initial implementation record below is historical. The subsequent
[post-human-review revision](#post-human-review-revision--2026-09-15) records the
three Major findings, corrections, current implementation SHA and fresh validation.

## Authority, baseline and delivery — 2026-09-15

- Specification / Clarifications / Plan / Tasks and ADR-0001–0004 are authority.
  Explicit human authorization covers **T03 only**. No material contradiction or
  need to change an approved architecture/contract was identified.
- Baseline: `378338cc0af79eaaec8b17e3cada44629515c043`, merged
  [PR #7](https://github.com/rgomids/axiom/pull/7). `git fetch origin main` and
  `gh pr view 7 --repo rgomids/axiom --json state,mergedAt,mergeCommit,url`
  returned exit 0; merge time `2026-09-15T03:37:48Z`. Local `main` and
  `origin/main` were advanced to that exact SHA. Initial worktree was clean.
- Branch: `codex/t03-strict-manifest-codec`.
- **Final implementation SHA: `94f42fb70279459787d460f3327615c0eac45d7a`.**
  It contains codec, tests, fixtures and dependency pins. The subsequent
  documentation-only commit records this SHA without a self-referential hash.
- **T01: Accepted / merged. T02: Accepted / merged.
  T03: Ready for human implementation review. T04–T21: Not started.**
  T03 merge does not authorize T04. No self-approval or merge.

Authority and context inspected: [Specification](spec.md), [Clarifications](clarifications.md),
[Plan](plan.md), [Tasks](tasks.md), ADR-0001–0004, Constitution, repository policies,
existing domain/application code and [T01](evidence-t01.md)/[T02](evidence-t02.md)
Evidence. Domain, application contracts, historical Evidence and ADRs are unchanged.

## Delivered component and decisions

[`internal/manifest/`](../../../internal/manifest/) is an in-memory adapter:

```text
UTF-8 bytes -> bounded YAML nodes -> closed schema/security validation
-> manifest DTO -> project.State -> project.New -> validated Project
Project -> explicit DTO mapping -> canonical YAML -> checked semantic round trip
```

- Node validation precedes typed DTO decoding. No permissive map decoding,
  implicit scalar-to-string conversion, extension bag or YAML types in domain.
- `optional[T]` retains absence/unconfigured/present, including explicit empty
  collections; optional string DTOs treat `unconfigured` as ordinary string data.
  Profile `state` is the specific approved union. All configured shapes follow
  Plan §2. Business Context `{}`, empty text and empty documents remain distinct.
- `Decode`/`Encode` return fixed codes and safe schema paths/numeric indices using
  domain issue values. Raw parser errors, rejected keys/scalars, URLs and panic
  values never enter results. Syntax/structural checks fail fast deterministically;
  domain validation retains its existing sorted independent issues.
- `Codec` implements the existing T02 `ManifestCodec` without changing T02.
  Its coarse port result uses existing `ValidationPhase / ArtifactsField /
  InvalidSnapshot`, category `validation`, remedy `correct_input`; callers needing
  detailed safe paths can use the pure codec functions. Richer presentation is
  deferred; this adapter does not expand the application diagnostic vocabulary.
- Ordering comes from explicit DTO field order and existing domain normalization.
  Keyed declarations sort by key; document/capability order, text and opaque case
  survive. Encoder uses two-space indentation, UTF-8 and a final newline. Scalars
  are emitted with their schema types, quoting ambiguous strings as needed.
- Encoder rejects zero-invalid Projects and applies the same wire/security limits
  to domain-created Projects. It returns no partial bytes on failure and verifies
  `decode(encode(p))` with `Project.Equivalent` before returning bytes. A domain
  value passing T01 alone is not proof of wire security or resource admissibility.
- Version support is exactly `!!int` with scalar value `1`. Missing, string, bool,
  null, float, other integers and alternative numeric spellings (`01`, `0x1`,
  `+1`, separators) fail; no numeric coercion or migration. Explicit standard
  string typing is respected (`!!str 1` is string data). YAML 1.2 `yes/on/no/off`
  remain strings; timestamp/bool/numeric nodes cannot silently become strings.

## Parser experiment and rationale

Selected **`go.yaml.in/yaml/v3` v3.0.5**, exact checksums in `go.sum`.
Upstream: [YAML organization repository](https://github.com/yaml/go-yaml),
[versioned package](https://pkg.go.dev/go.yaml.in/yaml/v3@v3.0.5).
This is the YAML organization's maintained fork. Registry inspection found v3.0.5
as latest v3 and only release candidates through v4.0.0-rc.6 for v4. Stable v3 API,
node evidence and experimentally demonstrated subset support motivated selection.
No broad ecosystem comparison or untested claim of parser safety is implied.

Before adoption in the repository module, copied `parser_contract_test.go` into
an isolated temporary module with the candidate pin. Executed
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -mod=mod -v ./...`: **exit 0**.
Candidate downloads were explicit implementation setup with network/checksums;
subsequent repository checks ran with proxy and checksum lookup disabled.
The parser itself has no runtime module dependency beyond the standard library.

Direct dependency contract tests, retained in the component:

| Test | Observed behavior and adapter consequence |
|---|---|
| `TestParserContractNodeEvidence` | Root/deep duplicates survive node parsing; anchors, alias targets, merge tags and custom tags remain visible. Adapter must reject them before DTO decode; library defaults alone are insufficient |
| `TestParserContractScalarTyping` | Int/float/bool/null/string/timestamp tags and scalar spellings remain inspectable. Explicit strings and YAML 1.2 word strings remain strings |
| `TestParserContractStreamAndHostileBounds` | Decoder exposes second document and EOF separately; undefined aliases/malformed syntax fail; depth 10,001 fails; 20,000-node sequence remains inspectable without expansion |
| `TestTreeLimitBoundaries` | Adapter accepts tree depth 32 and rejects 33, counting root as depth 1 |

No YAML unmarshalling into an unrestricted map or native domain object is used.
No aliases are resolved: node parse exposes references; tree validation rejects
anchors/aliases before the dependency's native DTO decoder can expand them.
Reimplementing YAML lexical syntax would add a larger parsing/security burden.
The selected dependency plus the restricted adapter passed the retained tests.

## Resource bounds

| Bound | Enforcement |
|---|---|
| 262,144 bytes (256 KiB) | Input length checked before parser allocation; invalid UTF-8 also fails before parsing |
| Depth 32 | Iterative AST validation, root depth 1, keys/values counted, document wrapper excluded |
| 16,384 nodes | Same traversal; scalar/container/key nodes count; document wrapper excluded |

**Allocation limitation:** v3 exposes no configurable streaming node budget. The
node/depth acceptance limits run after construction of a byte-bounded AST. The
parser additionally caps flow/indent nesting at 10,000 internally; this was tested.
The input byte ceiling bounds parser work/data, and aliases never expand. These
are explicit acceptance/resource controls, not a per-call deadline or allocator
sandbox. Decoder also contains a sanitized panic guard around node parsing.
Fatal runtime failures such as machine-wide out-of-memory cannot be recovered.

`BenchmarkHostileBounds -benchtime=1x -benchmem` observed on Apple M1/macOS:

| Synthetic input | Time | Allocated bytes | Result |
|---|---|---|---|
| Byte ceiling + 1 | 64,625 ns | 32 | Controlled rejection |
| 10,001 nested flow sequences | 12,326,000 ns | 5,501,232 | Controlled rejection |
| Dense sequence at byte ceiling | 62,016,083 ns | 44,798,760 | Controlled rejection |

Single-run measurements illustrate cost, not guaranteed maxima. At-limit byte and
node inputs pass their checks; limit+1 fails. Canonical quoting/indentation can make
an otherwise compact valid source exceed the output byte ceiling: Encode then
returns a size issue and no bytes. Limits are implementation constants, not a new
schema version or permission to consume arbitrary resources.

## Structural security policy v1

- Closed shapes reject secret/value/password/token bags and all local-only fields,
  including paths, executable location, observations, installation/formatVersion,
  cache/process metadata. No local DTO exists in this component.
- Logical references exclude key-value payloads, whitespace/control bytes,
  interpolation, URL/query syntax and machine paths. Known sensitive `name:value`
  payloads also fail. Ordinary logical names, including `token`, remain valid;
  no value-pattern/entropy scanner or credential catalog is invented.
- Identifier fields reject machine-path forms and control bytes. Free Project name
  and Business Context text remain data; they are not scanned or evaluated.
- URI user-info rejects passwords and non-SSH user-info; conventional SSH account
  names remain permitted. `file://` locators represent forbidden local paths.
- Known sensitive query names are URL-decoded once, case-folded and compared after
  removing hyphens/underscores. Fixed families: token/accessToken/refreshToken/
  idToken/authToken/oauthToken; password/passwd/pwd; apiKey/key; secret/clientSecret;
  signature/sig; credential/authorization/auth; X-Amz signature/credential/security
  token; X-Goog signature/credential. Invalid query escapes/separators fail closed.
  This versioned finite policy has explicit fixtures; benign query data and
  case-sensitive paths remain unchanged. No URL is fetched.
- Both encode and decode enforce these rules. Synthetic sentinels are absent from
  returned diagnostics. Input buffers remain unchanged, invalid Projects/snapshots
  cannot escape as usable results, and the codec accepts no writer/observer/secret
  source. AST/import inspection verifies no ambient effect API in production.

This is structural exclusion, not proof of secret absence. Unrecognizable secret
values can resemble valid identifiers or free text. Optional text/document scanning,
real document containment/existence and physical artifact allowlisting belong to
later authorized Tasks. No scanner or AI security validation was added.

## Task → FR/SEC/AC → tests

All rows cover **T03 portions** of acceptance criteria, not full slice acceptance.

| Traceability | Executed tests |
|---|---|
| T03; FR-009/012/015; AC-01/04/06/11/12; H3/H8; ADR-0004 | `TestGoldenRoundTrip`, `TestSchemaVersion`, `TestEquivalentStatesCanonicalBytes`, `TestOrderedListsAndOpaqueContent` |
| T03; FR-015/016; AC-06/12 | `TestYAMLAbuse`, `TestClosedMappingsAtEveryShape`, `TestStrictScalarTypesAndShapes`, `TestRequiredFieldsAndNestedScalarTypes`, direct parser contract tests |
| T03; FR-015; SEC-001; AC-06/08 | `TestByteDepthAndNodeLimits`, `TestTreeLimitBoundaries`, `BenchmarkHostileBounds`, `FuzzDecodeSafeRoundTrip` |
| T03; FR-009/012; AC-01/04/11/15; H3/H9 | `TestDeclarationStatesPairwise`, `TestNestedPresencePairwise`, `TestEquivalentStatesCanonicalBytes`, `TestDomainValidationAndNormalization` |
| T03; FR-008/015/016; SEC-001; AC-06/08/12 | `TestStructuralSecretAndLocalStateExclusion`, `TestURLSensitiveParameterPolicyV1`, `TestEncoderRejectsInvalidUnsafeOrOversizedState`; sentinel/input-preservation assertions shared by rejection cases |
| T03; FR-015/016; SEC-001; AC-06/08/12; inward dependency boundary | `TestT02PortAndReadSnapshotFailureHasNoEffects`, `TestManifestProductionDependencyBoundary`, unchanged domain/application checkers and regression suites |

External-package tests exercise actual codec/domain/T02 snapshot integration.
Private tests inspect dependency nodes and resource boundary behavior. Goldens are
synthetic, hand-authored expected output; no real credentials/provider data.
Pairwise tests compare semantic equivalence and canonical byte distinctions.
Functional black-box coverage is at the in-memory codec API; no CLI exists.

## Reproduction and results

Platform: **macOS 26.6.2 (25G83), Darwin arm64, Apple M1**.
Toolchain: **Go 1.26.1**. Run from repository root, after explicitly preparing the
pinned dependency cache as documented in [commands](../../commands.md#t01t04-validation):

```bash
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go test ./...
go test -race ./...
go vet ./...
go build ./...
go test -coverprofile=/tmp/axiom-t03-coverage.out ./...
go mod verify
go test -fuzz=FuzzDecodeSafeRoundTrip -fuzztime=30s -parallel=2 ./internal/manifest
go test -fuzz=FuzzDecodeSafeRoundTrip -fuzztime=200000x -parallel=2 -timeout=60s ./internal/manifest
go test -run '^$' -bench=BenchmarkHostileBounds -benchtime=1x -benchmem ./internal/manifest
go list -f '{{.ImportPath}}: {{join .Imports ", "}}' ./internal/...
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

| Check | Exit / result |
|---|---|
| Candidate contract tests, before adoption | 0; all direct dependency cases passed |
| Initial repository tests before implementation | 1 expected: candidate not yet in repository go.mod; subsequent red golden tests reported missing fixtures before fixture delivery |
| Full tests / race / vet / build | 0 / 0 / 0 / 0 |
| Coverage | 0; manifest **96.1%**, domain **99.6%**, application **100.0%** statements |
| Fuzz campaigns | 0 / 0; 90,395 executions in timed run; further 200,000 executions passed |
| Hostile benchmarks / module checksum verification | 0 / 0 |
| Static dependency inspection | 0; YAML only in manifest, inward imports, no domain/application changes |
| Domain/application checker and negative fixtures | 0; pure boundaries preserved; seven/eleven forbidden fixtures rejected without executing fixture code |
| Repository validator, shell syntax, diff and staged scan | 0; harness/package, regression and sensitive-path/content checks passed |
| Documentation link/gate/contract preservation checks | 0; relative links and latest lifecycle consistent; normative source bodies, historical Evidence and ADRs preserved |

Some defensive parser/encoder error paths are not reached by valid closed DTOs;
coverage is supporting evidence, not proof of correctness. Full tests were rerun
after final security/test edits. Fuzz campaigns preceded the final additional
reference-payload/file-URL rejection branches; deterministic tests cover those.

## Review, limitations and exclusions

Self-review covered Specification compliance, DTO/domain separation, input resource
bounds, scalar typing, duplicate/unknown keys at every approved object shape,
canonical/presence semantics, security and dependency direction. No blocking finding
or architecture/authority expansion identified. Human implementation review remains
required. No independent agent review or human acceptance is claimed.

- **Linux execution unverified**; only macOS results above. Go/package portability
  is not evidence of Linux behavior. No filesystem, permissions, TOCTOU, crash,
  concurrency, installation or CLI acceptance guarantee is claimed.
- `gitleaks`, `markdownlint`, `markdownlint-cli2`, `lychee`, `shellcheck` unavailable;
  dedicated scanner/linter/external-link checks remain unverified. No such tools
  were installed. Existing scans and public-safe fixture review are separate.
- Node limits apply after byte-bounded AST construction, as quantified above.
  No unbounded stream API, caller-increased limits or alias expansion.
- Excluded: T04–T21, local record codec, persistence/atomic protocol/native roots,
  init/update/install/reopen use cases, filesystem adapters, CLI, Runtime/Provider/AI,
  Git engine, command runner, generic persistence/Workspace/DI/migration frameworks,
  migrations, split manifests, original formatting preservation and source rewrite.
- Documentation changes reconcile only T03; T21 is not started. Domain normalization,
  T02 contracts, ADR decisions and approved schema shapes remain unchanged.

**Stop at human implementation review. Do not merge. Do not start T04.**

## Post-human-review revision — 2026-09-15

### Authority and exact revision

- Read PR #8 description, all current reviews and issue/inline comments through
  `gh pr view 8 --repo rgomids/axiom --json title,body,state,headRefName,headRefOid,baseRefName,comments,reviews,url`
  and paginated `gh api repos/rgomids/axiom/pulls/8/comments` and
  `gh api repos/rgomids/axiom/pulls/8/reviews`. One human review, three Major
  findings, no issue or inline comments at intake. Review:
  [2026-09-15 human review](https://github.com/rgomids/axiom/pull/8#pullrequestreview-5212615197).
- Revision baseline: `aea1f1efaed5bc2161a651a18ee05368d859dbeb`, clean branch
  `codex/t03-strict-manifest-codec`, confirmed against fetched remote head.
- Specification, Clarifications H1–H11, Plan, Tasks, ADR-0004, domain, adapter,
  existing tests and this Evidence were inspected before editing production code.
  Scope remains T03; approved contracts require no amendment.
- **Corrected implementation SHA: `48e90be5bd9179fbf5047364d3b0462d1be7435a`.** A subsequent
  documentation commit records the implementation hash without self-reference.
- **T03: Ready for human re-review. T01/T02: Accepted / merged.
  T04–T21: Not started.** No approval, merge or next-Task authority is inferred.

### Findings, root causes and corrections

| Human finding | Root cause | Correction and regression Evidence |
|---|---|---|
| Major: `sourceHint` accepts relative local paths | Generic machine-path check covered rooted forms only; URL check required `://` | Logical-reference validation rejects `.`/`..` path components, rooted suffixes, backslashes, drive-relative forms and `file:` URIs. Tests cover direct, namespaced, YAML-escaped and percent-escaped forms in encode and decode |
| Major: structural secret payloads accepted as identifiers | `referencePayload` ran only for logical fields and recognized only `:` | Apply the existing finite sensitive-name families to all nine schema-classified identifier fields and logical references, with `:` and `=` delimiters, case/hyphen/underscore folding and whitespace around the name. No entropy/value scanner or catalog |
| Major: Encode materializes oversized YAML before checking `MaxBytes` | Unrestricted `bytes.Buffer` followed by Decode's byte check | Private `io.Writer` rejects crossing writes before allocation/copy, retains at most `MaxBytes` bytes/capacity and stays failed. Encode/Close overflow maps to `byte_limit`, returns nil bytes and bypasses round-trip parsing on that failure. Successful output still passes unchanged Decode/Equivalent validation |

Additional equivalent bug found during review: `repositories[].remote` accepted
`file:/synthetic`, `FILE:relative` and `file:../synthetic` as SSH shorthand.
The adapter now rejects the `file` scheme before the `://` check. The domain's
locator normalization and public contracts remain unchanged. Ordinary SSH/HTTPS
locators, case-sensitive paths and benign escaped query data still round-trip.

Security checks inspect percent-decoded views without changing stored values.
Nested escapes cannot hide the same path/payload syntax; malformed escapes fail
closed. A per-value inspection budget of four times the original byte length
bounds repeated decoding work; excessively nested escaping fails closed. This
applies to logical/identifier syntax only. Remote query-name handling retains its
existing single URL-query decoding policy. No environment/store value is resolved.

Positive tests preserve environment names, Keychain/Secret Service/libsecret/
Windows Credential Manager/runtime-managed logical entries, namespace-style secret
references, ARN-style identifiers, slash-separated logical names and opaque model
IDs. Single-letter namespaces remain permitted in identifier fields; the
drive-relative restriction is specific to logical references. Bare `token`,
`password` and other sensitive-family names remain valid identifiers. Human names
and Business Context text are not subjected to this structural payload policy.
An ordinary slash-separated name cannot establish a filesystem location by itself;
these checks reject explicit local syntax, not every string that could name a file.

### Regression tests and negative controls

- `TestReviewLogicalPaths`, `TestReviewLogicalFieldVariants`: path rejection and
  legitimate references across all six logical schema fields; encode/decode and
  semantic preservation. Original examples and equivalent encodings included.
- `TestReviewIdentifierPayloads`: all nine identifier fields, both delimiters and
  all existing sensitive-name families; positive opaque identifiers and names.
- `TestReviewEscapedSyntaxAndFreeText`, `TestReviewLocalURIRemotes`: encoded names/
  delimiters, YAML escapes, local URI variants, preserved human text and remotes.
- `TestBoundedOutputWriter`: arbitrary tested chunk sizes, exact limit, empty
  writes, oversized first write, crossing write without partial copy, sticky
  failure and retained buffer capacity. No dependency chunk-size assumptions.
- `TestEncodeStopsAtWriterBound`: valid domain Projects with 1 MiB plain text and
  128 KiB text whose YAML escaping exceeds the ceiling. Observes the writer used
  by the Encode implementation: overflow flag set, retained buffer within bound,
  nil public output, `byte_limit`; post-serialization Decode alone cannot explain
  the observed writer rejection.
- `TestEncodeOutputByteBoundary`: canonical byte lengths `MaxBytes-1`, `MaxBytes`
  and `MaxBytes+1`, final newline, valid round trip and exact rejection boundary.
- Existing T02 snapshot integration now includes the path/payload findings;
  fuzz seeds include nested escaping, local URI, payloads and valid store references.
  Existing goldens and declaration/nested-presence matrices remain unchanged.

Before production fixes, new security regressions failed with missing structural
rejections. Re-running against baseline `security.go` reproduced both original
security bugs (exit 1); restoring corrected code passed. An encode negative control
temporarily restored unrestricted `bytes.Buffer` serialization plus subsequent
Decode: `TestEncodeStopsAtWriterBound` failed with `Encode did not enforce the
writer bound` (exit 1), despite the later byte-limit rejection. Restoring the writer
passed. Both temporary substitutions were restored before final validation/commit.
This distinguishes writer enforcement from a test that only checks the returned code.

Initial writer tests failed to compile before `boundedOutput`/`encodeWithOutput`
existed (exit 1, expected). An initial result-reporting shell wrapper used zsh's
read-only `status` variable and failed after running the tests; subsequent wrappers
used `review_exit`, and the negative-control commands completed as recorded above.

### Executed validation

Final platform: **macOS 26.6.2 (25G83), Darwin arm64, Apple M1; Go 1.26.1**.
All Go commands used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; no downloads or
dependency changes. The required full command list in the original reproduction
section was rerun as applicable below; the historical extra 200,000-execution
campaign and historical coverage percentages are not fresh results.

| Command | Fresh result |
|---|---|
| `go test -run '^TestReview' ./internal/manifest` | Exit 1 against baseline security; exit 0 after fixes |
| `go test -run '^TestReviewLocalURIRemotes$' ./internal/manifest` | Exit 1 before additional URI fix; covered by final passing suite |
| `go test -run '^TestEncodeStopsAtWriterBound$' ./internal/manifest` | Exit 1 with unrestricted-buffer negative control; corrected writer passed |
| `go test ./...` | Exit 0; passed |
| `go test -race ./...` | Exit 0; passed |
| `go vet ./...` | Exit 0; passed |
| `go build ./...` | Exit 0; passed |
| `go mod verify` | Exit 0; passed |
| `go test -fuzz=FuzzDecodeSafeRoundTrip -fuzztime=30s -parallel=2 ./internal/manifest` | Exit 0; 380,888 executions; passed |
| `go test -run '^$' -bench=BenchmarkHostileBounds -benchtime=1x -benchmem ./internal/manifest` | Exit 0; measurements below |
| `go run ./scripts/check-project-domain.go` | Exit 0; passed |
| `bash scripts/test-check-project-domain.sh` | Exit 0; passed |
| `go run ./scripts/check-projectapp.go` | Exit 0; passed |
| `bash scripts/test-check-projectapp.sh` | Exit 0; passed |
| `./scripts/validate-repository.sh .` | Exit 0; passed |
| `git diff --check` / `git diff --cached --check` | Exit 0; passed |
| `./scripts/check-sensitive-files.sh --staged .` | Exit 0; passed |

Final hostile-input benchmark, one iteration per case:

| Case | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| Byte ceiling + 1 | 2,750 | 32 | 1 |
| Depth 10,001 | 5,130,208 | 5,496,040 | 9,153 |
| Dense sequence at byte ceiling | 44,550,833 | 44,804,080 | 393,319 |

### Review and remaining limitations

- Self-review covered each finding, equivalent syntax, scope, dependency direction,
  sanitized failures, positive compatibility and all declaration-presence forms.
  No remaining blocking finding identified; human re-review remains required.
- Writer tests prove retained output length/capacity and controlled failure at the
  writer boundary. They do **not** prove total Encode heap usage or a deadline:
  the caller's Project, detached state/DTOs, encoder scalar/event working memory
  and transient buffer growth allocations are outside that bound. No arbitrary
  total-memory claim or dependency-internal chunk-size assumption is made.
- Decoder node/depth checks still follow byte-bounded AST construction. Benchmarks
  are observations, not guaranteed maxima. Structural exclusion is not proof of
  secret absence; no generic scanner or free-text secret detection was introduced.
- **Linux was not executed.** Filesystem, CLI, persistence, permission, crash and
  install guarantees remain outside T03. T04 and later Tasks remain unstarted.
- `gitleaks`, `markdownlint`, `markdownlint-cli2`, `lychee`, `shellcheck` unavailable
  in PATH; dedicated checks remain unverified. No tools installed. Repository
  sensitive-file/content checks and manual synthetic-fixture review are separate.
- Only adapter implementation/tests, this Evidence and CHANGELOG change. Domain,
  T02 ports, normative Specifications/Plan/Tasks and accepted ADRs remain unchanged.
  README and command instructions already describe the same package/API/toolchain.

**Ready for human re-review. No approval or merge. T04 not started.**
