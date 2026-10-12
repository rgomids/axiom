# Issue #306 implementation Evidence

## Authority and receipt boundary

Implementation authorization: [PR #310 comment 6109821395](https://github.com/rgomids/axiom/pull/310#issuecomment-6109821395). Unified ownership: [comment 6109651213](https://github.com/rgomids/axiom/pull/310#issuecomment-6109651213). Approved Tasks baseline: commit `16c2aa034ffc98009ce2e04c1d243d76eb645ea8`, blob `72bb0ed65c08c422536f7867af4a0f4daa97ac28`. The approved planning artifact remains unchanged. This separate receipt records implementation, rather than retroactively changing planning acceptance.

Repository: `rgomids/axiom`; Issue #306; existing Draft PR #310; branch `feat/306-local-runtime-profiles`. The commit containing this receipt identifies its source tree. Remote CI must match that commit or its subsequent documentation-only consolidation before final technical completion. No merge, release, Issue closure, Ready transition, external model run or human acceptance is authorized by this receipt.

**Current checkpoint:** 23 units have implementation coverage below. Local tests and synthetic integration are classified independently. Supported native Linux/Windows CI is **UNVERIFIED** until exact-head receipts arrive. Therefore T15/T16 and the final X07 consolidation are **UNVERIFIED**, and full Issue completion is not asserted. GATE-I306-INTEGRATED remains **BLOCKED** by missing authoritative external receipts.

## Task acceptance matrix

Paths below are relative to the repository. Test names are executable receipts, not claims of vendor capability.

| Task | Implemented contract and acceptance coverage | Evidence owner/test | Classification |
| --- | --- | --- | --- |
| I306-T01 | Canonical configuration, observation, authoring, conditional publication/recovery ports; application owns exact authority; CLI projects sanitized reports | `runtimeapplication/authoring.go`, `runtimeprofile/v2.go`, `local/runtime_profile_authoring.go`; application authoring tests | PASS local |
| I306-T02 | Mandatory Profile effort, exact closed v2 member names, no credentials, version dispatch; canonical tombstone1 and immutable v1 decoding | `runtimeprofile/v2_test.go`; current corpus `local-v2.json`, `tombstone-v1.json`; old fixtures retained | PASS local |
| I306-T03 | Exact Project/local intersection, Profile-first uniqueness, model/effort proof consumption and complete PE-04 tuple; ambiguity before proof, no complexity-derived effort | `runtimeapplication/policy_v2_test.go`, runtimeprofile resolution tests; capability evidence injected | PASS synthetic |
| I306-T04 | Exact snapshot token binds bytes, file/ancestor identity and generation; locks, conditional create/update/retire/recreate, monotonic revision and read-back | `local/runtime_profile_authoring_test.go`: lifecycle, identity, contention and multiprocess tests | PASS macOS native; other native rows UNVERIFIED |
| I306-T05 | Local1/2 and tombstone recognition, supported malformed versus future-version diagnosis; scoped F0–F8 recovery preserving unrelated scopes | local runtime authoring/recovery tests and `TestRuntimeProfileInventoryInvalidSupportedV2IsMalformed` | PASS macOS native; other native rows UNVERIFIED |
| I306-T06 | Explicit effort for every legacy Profile; explicit legacy-reference removal; preview alone writes nothing; no implicit rewrite | `TestAuthoringMigrationRequiresEveryExplicitEffortAndLegacyConsent`; protected restart chain; smoke CLI migration | PASS local |
| I306-T07 | Input/candidate redaction, safe identifiers, native subscription-auth boundary, no credential store/import; version/mode/source/proof-scoped overrides | runtimeapplication canaries; runtimeadapter effort-source negative tests; subscription-auth tests; independent review | PASS local/synthetic; vendor proof UNVERIFIED |
| I306-T08 | Typed authoring operations, exact effect preview/apply, stale replay and no-op refusal, cascade and tombstone/recreation, uncertainty and scoped recovery | `runtimeapplication/authoring_test.go`, local fault/recovery tests | PASS local |
| I306-T09 | Canonical binding/profile CLI metadata and thin adapters; duplicate/unknown flag refusal and structural candidate validation | `internal/cli/runtime_authoring.go`, `cmd/lingo/runtime_authoring_test.go`, metadata/help contracts | PASS real CLI with synthetic executable inventory |
| I306-T10 | Existing readiness distinguishes missing/retired/unsafe/recovery/legacy effort from model/auth/proof gaps; bounded catalog includes archived Projects and reports unknown references honestly | `cmd/lingo/runtime_impacts_test.go`, bootstrap readiness tests; two-Project snapshots unchanged | PASS local/synthetic |
| I306-T11 | Fourth thin `axiom-runtime` skill in shared Codex/Claude catalog, manifest/upgrade/install/help parity; historical three-skill digest retained | codexruntime/install/domain-skill tests; bounded installer smoke | PASS macOS native Axiom installation; Linux/Windows UNVERIFIED |
| I306-T12 | Passive executable/integration discovery with no vendor process, authentication or forced authoring | runtimebootstrap passive callback/process-refusal tests; executable first-run tests | PASS local/synthetic |
| I306-T13 | Optional descriptive fast/balanced/deep guidance; operator-supplied model/token, one/three/four Profiles, no universal effort tiers or inferred availability | skill source plus `TestExecutableRuntimeAuthoringRestartAndRetirement` four inventories/four Profiles per Runtime | PASS local/synthetic |
| I306-T14 | Production CLI writes reviewed intent in isolated roots; four installed states; replay, refusal, remove/recreate and restart; shared-profile impacts without Project mutations | executable runtime authoring; application migration/recovery; two-Project impacts; smoke journal | PASS local Axiom; installed executables and positive capability scenarios synthetic |
| I306-T15 | Anchored roots/private permissions, symlink/reparse/identity refusal, locking, replacement/concurrency and recovery; fresh/upgrade/reinstall inventory | local/install native suites and CI verify matrix | PASS macOS native; Linux/Windows UNVERIFIED |
| I306-T16 | Independent local acceptance/checkpoint, command/help reconciliation, security/native classification and explicit external handoffs | This receipt's local matrix, validation, review and handoff sections | UNVERIFIED pending T15 native CI |
| I306-X01 | Workflow schema2 excludes Agent effort and reserved effort-capability channels; closed codecs; schema1 historical semantics preserved | workflowdefinition v2/encoding/negative tests and validator examples | PASS local |
| I306-X02 | Explicit per-agent Profile mapping, old/new effort comparison, exact Project/local policy, immutable source revision, independent publication/selection; source token checked under lock and before commit | projectapp migration tests; native same-byte replacement publication refusal test | PASS local/synthetic |
| I306-X03 | Compiler/StagePlan2 takes controls only from `resolvedProfile`; per-agent tuple and input/proposal digest; historical compiler1 retained | workflowcompiler v2 and protected chain; current public single/mixed StagePlan journey | PASS synthetic |
| I306-X04 | Execution3 freezes schema2 snapshot and initial Profile tuple; restart/clone isolation and rebind refusal; historical records inspectable; new legacy start/resume refused | workflow binding tests, protected chain, current start and historical public fixture tests | PASS local/synthetic |
| I306-X05 | Graph2 freezes per-child tuple/controls; source inspection and exact proof guard before each attempt; legacy current dispatch refused; no replacement dispatcher | executiongraph/graphapplication current tests; runtimeadapter source tests; protected graph restart | PASS synthetic and native file inspection; actual vendor attempt UNVERIFIED |
| I306-X06 | Addressable builtin default-sdd r2/schema2, no vendor/model/effort imposed; builtin1 bytes/digest retained; confirmed Project selection stays separate | workflowdefinition v2 tests, current start/StagePlan journey, passive four inventories and separate-selection smoke | PASS synthetic/local |
| I306-X07 | Protected migration/restart chain, negative proof/model/source drift, no rebind, current admission versus historical readers, append-only six-family corpus | `TestProfileV2ProtectedMigrationCompilerGraphRestartChain`, current/legacy admission suites, version matrix below | UNVERIFIED final exact-head/native consolidation; local chain PASS |

## Version and ownership matrix

| Family | Historical read/inspection | Current write/new admission | Continuation/downgrade rule |
| --- | --- | --- | --- |
| Local1 | Preserved closed historical reader and original fixture bytes | Explicit migration only into Local2; v1 writer cannot overwrite Local2/tombstone | Legacy absence of effort blocks current use; no inferred effort |
| Local2 | Protected reader, private-root inventory and restart | Conditional reviewed authoring; exact token/revision/effects | Legacy Save/Create refused over v2; malformed supported2 differs from unknown3 |
| Tombstone1 | Canonical recognized retirement metadata | Final compound removal; recreation increments prior generation | Retired intent cannot resolve; no fresh-generation reset |
| Workflow1/default-sdd1 | Original bytes, digests and historical compiler retained | Current Start refuses schema1; operator publishes separately mapped schema2 | No implicit migration/selection; old immutable revisions retained |
| Workflow2/default-sdd2 | Closed schema2 document and addressable builtin2 | Confirmed publication/selection; resolution gets Profile effort only | Agent effort/direct or capability-based override rejected |
| StagePlan1/2 | Historical compiler/plan semantics retained independently | Current public StagePlan requires Execution3/schema2; proposal authority separate | Old plan/approval cannot authorize new family |
| Execution1/2/3 | Legacy inspection/advance/reconciliation fixture regressions retained; Execution3 restart | Current production Start requires schema2 + PE-04 tuple and writes3 | Current Resume refuses legacy contract; later configuration cannot rebind existing3 |
| Graph1/2 | Historical graph reading and golden receipts preserved | Current graph application requires2 and resolved tuples | Legacy current dispatch refused; each new attempt rechecks tuple and effort sources |

Current corpus lives in `internal/workflowcompiler/testdata/issue306/`, alongside rather than replacing historical fixtures. `SHA256SUMS` hashes trimmed canonical JSON bytes. The protected chain recomputes every document and compares frozen bytes after restart. Only test workspace paths are normalized to `/synthetic/issue306`; protected records use actual isolated paths. No production identity or authorization is normalized.

Effort-source inspection consumes an authoritative externally supplied version/executable/mode/precedence declaration. It requires scoped consent, recognizes bounded configuration syntax, binds exact bytes plus object/ancestor identity, treats managed drop-in inventory explicitly as a directory, and refuses missing/unknown/unbounded inputs. It neither invents a supported vendor version nor obtains G-3 evidence. runtime-default emits no Axiom effort override; explicit selectors derive only from the frozen Profile. Existing vendor-managed subscription authentication remains the auth owner.

## Executed validation

Local environment: macOS arm64, isolated Axiom state, no vendor authentication/model processes. Go tests use the module-selected Go1.26 toolchain. The quality script uses Go1.26.1 with consistent GOROOT/PATH and Staticcheck `2026.2.1 (0.8.1)`; the shell's older Go1.24.5 and an inconsistent GOROOT produced setup errors, corrected before the passing quality run. No dependency upgrade was made.

| Command | Result and boundary |
| --- | --- |
| `env -u OPENAI_API_KEY -u CODEX_API_KEY go test ./...` | PASS full local suite; host API-key variables removed for subscription-test isolation |
| `env -u OPENAI_API_KEY -u CODEX_API_KEY go test -race ./...` | PASS full local race suite |
| `go test -race ./internal/local ./internal/projectapp -run 'TestWorkflowMigration\|TestWorkflowPublication' -count=1` | PASS post-review source-identity regression, native filesystem + synthetic mapping |
| `go test ./internal/workflowcompiler -run TestProfileV2ProtectedMigrationCompilerGraphRestartChain -count=2` | PASS frozen corpus and restart chain, injected capability evidence |
| `go vet ./...` | PASS local |
| `scripts/check-go-quality.sh all` with consistent Go1.26.1 environment | PASS formatting, `go mod tidy -diff`, pinned Staticcheck including tests |
| `scripts/validate-repository.sh .` | PASS repository, schemas/examples, security/governance/bootstrap structural checks; optional maintainer behavioral evaluation UNVERIFIED |
| `scripts/dogfood-poc.sh` | PASS production local migration/separate selection/current refusal; positive current and historical lifecycle tests explicitly SYNTHETIC |
| `git diff --check` | PASS |
| `gitleaks dir . --no-banner --redact` | PASS worktree scan; staged sensitive-file and `gitleaks git --staged --no-banner --redact` checks also PASS before commit |

Changes after the full race run are additionally checked in affected packages. The full local suite was rerun after review fixes and passed. Native Windows/Linux cross-builds also passed; staged scan passed; exact-head CI outcomes will be reconciled before technical completion. Cross-compilation is a build receipt only and cannot satisfy T15 native Windows/Linux gates.

## Independent engineering/security review

A separate reviewer inspected the consolidated diff and ran nine affected package suites. Findings corrected before publication:

1. Effort-source scope/content digest did not bind every ancestor/object replacement or explicitly distinguish a managed drop-in directory. The source reader now checks anchored identity, stable bytes and bounded directory membership. Unknown TOML/JSON syntax and duplicates fail closed.
2. Workflow migration consumed semantic configuration digest without the exact source snapshot identity. The observation token now participates in preview authority and is checked under the shared root lock and precommit callback. Native same-byte file replacement and synthetic identity drift regressions pass.
3. Smoke skill count still expected three instead of four. Reconciled installation inventory and current-versus-historical admission checks; no fabricated production model proof was added.
4. Invalid supported Local2 was diagnosed as a future version. Inventory now distinguishes malformed2 from unsupported3, with a regression test.

No remaining blocking correctness/security finding was identified in the reviewed scope. This is engineering review, not human acceptance. Native Windows ACL/reparse execution remains a required CI receipt. The single-child execution dispatcher remains #276-owned; this Issue freezes Profile controls in compiler/StagePlan/Execution/Graph boundaries and does not claim real dispatch completion.

## Handoffs and remaining gates

- **G-3 / #276 — BLOCKED external functional readiness:** production observation does not acquire positive Runtime/model/effort/no-override proof. Supply authoritative exact version, executable digest, invocation mode, observation/configuration/source identity and precedence proof. Unknown or stale evidence blocks resolution/attempt before vendor invocation.
- **#305 — owner preserved:** local Runtime authoring does not modify portable Project policy. A Project mismatch reports the existing policy-owner action. Workflow publication and active selection have independent confirmation.
- **#276 — actual agent dispatch/coordination UNVERIFIED:** synthetic dispatcher/guard regressions test consumption and zero-effect refusals; no authenticated or paid inference receipt exists.
- **#278 — real E2E/MVP/AXM-7 R-3 BLOCKED:** requires separately authorized real-world receipts and human acceptance. Tests, CI and this document confer no acceptance.
- **#277/G-4 — NOT_APPLICABLE to local configuration proof:** delivery/rework and external Provider linkage are not authored by this change; required integrated scenarios must acquire their own receipts.
- **Native Linux/Windows and final CI — UNVERIFIED:** supported platforms remain required. Do not label cross-builds or synthetic executables native success. Complete T15, then T16/X07 exact-head consolidation, after actual required checks pass.
- **Human review/merge — pending:** PR remains Draft. No self-approval, Ready transition, Issue closure, merge or release.
