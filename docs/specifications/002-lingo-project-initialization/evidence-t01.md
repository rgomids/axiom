# T01 — Implementation Evidence

## Authority and delivery status — 2026-09-14

- Specification: **Approved**. Plan: **Approved**. Tasks: **Approved**.
- Implementation: **Authorized**, explicitly limited by the human to **T01**.
- Baseline: `main` / merged [PR #5](https://github.com/rgomids/axiom/pull/5),
  `f70c7e5e762c408db3093753e6dabbc74d65f035` (merge confirmed through GitHub).
- Delivery revision: the commit containing this Evidence and the accompanying
  `internal/project/` implementation; use `git rev-parse HEAD` after checkout to
  identify the exact reviewed revision. Branch: `agent/spec-002-t01`.
- **T01 implemented; ready for human implementation review. T02–T21: Not started.**
  Agent checks do not approve or merge the PR, or authorize the next DAG unit.

Read in full before code: [Specification](spec.md), [H1–H11 and original history](clarifications.md),
[approved Plan](plan.md), [Tasks](tasks.md), ADR-0001/0003/0004, Constitution,
architecture/quality/security/documentation policies and repository security controls.
Current human authorization supersedes historical pending lifecycle gates only.
No requirement, approved Task definition, Plan contract or ADR was reinterpreted.

## Delivered boundary

[`internal/project/`](../../../internal/project/) contains only portable domain
values and behavior. `New(State)` validates complete input and returns an immutable
Project snapshot; failed validation returns no valid Project. `State()` returns
detached values. `Propose(Intent)` preserves ID and all unsupplied fields, materializes
the complete proposal and revalidates retained references. `Equivalent` compares
normalized valid intent, with keyed collections sorted and all presence retained.

Internal, reversible representation choices within Plan §1/§2/§4:

- UUID input uses canonical lowercase, hyphenated UUID v4 with RFC variant bits.
  No entropy allocation; supplied identity is preserved. Slug uses the exact
  approved ASCII grammar; names and opaque values retain case/content.
- `Declaration[T]` represents absent, explicit unconfigured or present values.
  Go nil and empty slices inside a present collection both mean explicitly `[]`;
  neither means absence. The field-specific validator rejects forbidden forms.
  `ModelProfile.State` admits only absence or the explicit unconfigured form.
- `Change[T]` records whether a field was supplied. Intent uses explicit whole-field
  replacement, an internal operation encoding left open by Plan §4: omitted change
  retains the field; setting an absent declaration removes it. A supplied object or
  collection replaces that whole field; it is not an implicit nested merge. No wire
  patch syntax, JSON/YAML merge-patch standard or future CLI API is selected.
- Identity and schema version are not mutable intent fields. No independent fork,
  ownership, slug registry or Repository CRUD service exists.
- Locator normalization changes only URI scheme/DNS-host case (SSH shorthand:
  unambiguous DNS-host case). User, port, escaping, path case, suffix, slash, query
  and fragment survive. No SSH/HTTPS coalescing or semantic alias claim. Ambiguous
  drive-like shorthand and invalid syntax fail safely. URI IPv6 literals remain
  byte-preserved. Password-bearing user-info and non-SSH user-info are rejected.
- Domain issues contain fixed field paths/indices and codes, sorted by field/code.
  Application categories, remedies, severities and outcomes remain T02 work.

Project != Repository, Axiom != Lingo, Runtime != Model, Provider != Transport and
Integration != MCP remain intact. No filesystem/YAML/CLI/port/adapter dependencies,
vendor catalog, discovery, Git, network, secret reads or Runtime execution.
Go module is introduced solely to build/test this authorized domain, using the
installed Go toolchain and standard library; no third-party dependency or framework.

## Requirement and behavior Evidence

Tests use external `project_test` consumers of the public internal-package API,
synthetic in-memory data and observable results rather than private methods.

| T01 contract / traceability | Executed behavior tests |
|---|---|
| Minimal/configured state; FR-001/002/005/006/007/009; AC-01/02/05/11; H3; Plan §2 | `TestMinimalAndConfiguredProject`, `TestConfiguredShapesAndReferenceIntegrity` |
| Canonical v4 version/variant, immutable identity, mutable nonunique name; FR-001; AC-04; H2; ADR-0004 | `TestUUIDVersionVariantAndCanonicalIdentity`, `TestMinimalAndConfiguredProject`, `TestProjectOwnsSnapshotsAndIntentInputs` |
| Full slug grammar; FR-017; AC-13; H2 | `TestSlugGrammar` |
| Unique keys and normalized locators; FR-002/004; AC-02/04; ADR-0001; Plan §5 | `TestDuplicateDeclarations`, `TestLocatorNormalizationIsConservative` |
| All top-level optional forms; FR-005/006/007/009/012; AC-01/04/05/11; H3 | `TestAllTopLevelDeclarationForms`, `TestEquivalencePreservesPresenceAndIgnoresKeyedOrder` |
| Nested optional strings/collections/transport/profile state; Plan §2; AC-04/05/11 | `TestNestedOptionalStrings`, `TestNestedCollectionsAndTransportPresence`, `TestConfiguredShapesAndReferenceIntegrity` |
| Complete proposed state; omission retention; explicit removal with dangling Provider/credential/Runtime reference; FR-018; AC-15; H4/H9; ADR-0004 | `TestPartialIntentRetainsCompleteStateAndValidatesRetainedReferences`, `TestCompleteIntentAndOrderedData` |
| Deterministic equivalence, keyed order, opaque case, document order, no state aliasing; FR-012/018; AC-04/15 | `TestAllKeyedCollectionsNormalizeWithoutChangingOpaqueData`, `TestProjectOwnsSnapshotsAndIntentInputs`, `TestDeterministicSafeIssues` |
| Lexical document references and untrusted text; FR-009; AC-11 | `TestDocumentReferencesAreLexicalAndTextIsUninterpreted` |
| Pure domain / no application I/O; Plan §1; ADR-0003 | Standalone AST import/symbol check, checker rejection fixtures, complete source inspection; domain tests contain no environment/filesystem/process/network calls |

T01's retained-reference removal Evidence uses actual approved portable references
(Provider, credential and Runtime). Plan §4's illustrative Repository-removal case
does not introduce a `repositoryRef` field into the closed schema. No such field
was added; local binding continuity remains outside T01.

## Reproduction and executed results

Environment: macOS 26.6.2, Darwin arm64, `go version go1.26.1 darwin/arm64`.
Run from the repository root with an installed Go 1.26 toolchain:

```bash
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go test -cover ./...
go test -race -shuffle=on -count=10 ./internal/project
go vet ./...
go build ./...
go run ./scripts/check-project-domain.go
bash scripts/test-check-project-domain.sh
./scripts/validate-repository.sh .
for script in scripts/*.sh; do bash -n "$script" || exit; done
git diff --check
git diff --cached --check
./scripts/check-sensitive-files.sh --staged .
```

| Evidence | Result |
|---|---|
| Initial tests before implementation | Expected exit 1: no non-test Go files in package; red construction baseline |
| Domain test suite | Exit 0; **99.6% statement coverage**; coverage is supporting evidence, not proof of correctness |
| Race detector, shuffled order, ten runs | Exit 0 |
| `go vet ./...`, `go build ./...` | Exit 0; domain package only, no executable CLI |
| Domain AST boundary check | Exit 0; six domain source/test files checked; pure import/symbol allowlist, no compiler directives or I/O test helpers |
| Boundary-check regression fixtures | Exit 0; pure fixture accepted; seven network/process/environment/output/reflection/import-alias fixtures rejected without executing fixture code |
| Repository validator | Exit 0; harness/package checks, both existing Bash regression suites, sensitive-file scan and whitespace check |
| Shell syntax, worktree/staged whitespace and staged sensitive-file scan | Exit 0 |

The standalone AST checker performs source I/O outside domain tests. Go compilation,
test reporting and verification tools necessarily read/write their own inputs,
caches and reports; those effects are not domain behavior. The allowlist plus
source inspection establishes that this implementation and its test scenarios use
pure operations; it is not an OS sandbox or a claim that the Go runner performs
zero syscalls. Test fixtures use no `TempDir`, environment reads, secrets, network,
external processes, Provider discovery or Runtime execution.

## Review and limitations

Reviewed complete implementation/test/tool/documentation diff against T01, Plan
§1/§2/§4/§5, H2–H4/H9 and Accepted ADR boundaries. No material contradiction,
new durable architectural decision or blocking finding identified. Authored by
the repository agent under explicit human T01 authority; acceptance remains human.

- No CLI, YAML/local codec, application authority/ports, persistence or installation.
  Integration/filesystem/CLI AC evidence and Linux behavior remain **unverified**;
  only the T01 domain portions of referenced ACs are delivered.
- Document checks are lexical only; existence, safe actual containment, permissions
  and text inspection are deferred. Portable validity beyond T01 still needs the
  approved codec/application/security tasks.
- Structural secret-value and sensitive-query policies belong to T03/T18; T01
  does not claim complete secret detection or a complete security validator.
- No remote alias discovery or semantic Repository equivalence; distinct syntactic
  locators stay distinct. No operational readiness or binding availability inferred.
- `gitleaks`, `markdownlint`, `markdownlint-cli2`, `lychee` and `shellcheck` unavailable;
  dedicated scanner/linter/external-link coverage remains unverified. No tools or
  dependencies installed. Existing frozen experiments and ADR text are untouched.
- T21 is not started: these documentation edits only reconcile T01 delivery and
  explicit lifecycle authorization, as required for this unit.

**Return to human implementation review. Do not advance to T02/T03/T04 without
new human instruction after T01 review.**
