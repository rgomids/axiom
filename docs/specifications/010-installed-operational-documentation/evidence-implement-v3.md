# AXM-37 — Implement v3 Evidence

Same Draft [PR #335](https://github.com/rgomids/axiom/pull/335), same logical
Implement, baseline R1 accepted by [H1](acceptance-h1.md). This is implementation
Evidence, not SDD Review, merge, release or functional acceptance. It supersedes
the intermediate [save state](save-state-implement-v3.md), which stays as history.

Status words: **PASS** (executed here, result retained), **PARTIAL** (executed
for part of the criterion), **NOT_RUN**, **UNVERIFIED** (no sustaining run).
Local macOS arm64 runs are the author's host; CI runs are the PR's ubuntu amd64
runner (`PR Checks`). Neither is the qualified native matrix of T13–T15.

## Task traceability

| Task / Evidence | State | What exists and what was run |
| --- | --- | --- |
| T01 / E01 | PASS (codecs) | `internal/operationaldocs`: closed metadata 2, receipt 2 (generation formula checked), binding JSON (canonical emit, hash over compact JSON + LF), TSV and descriptor/skills identities. Per-key remove/duplicate/swap/empty, BOM-less CR/LF/no-final-LF, quota-before-parse and JSON duplicate/unknown/trailing/range negatives; the accepted [positive vectors](contract-vectors-proposed.json) replay through the production codecs. The 148-case oracle count is a specification; the executed mutation set is the table-driven generator in `kv_test.go`, not a case-by-case replay of that list |
| T02 / E02 | PARTIAL | Descriptor-based association is exercised by `resolve_test.go` and `docs_publish_test.go`: C2 (disk bytes replaced under a running A) refuses, same-label/different-bytes B only verifies with its own descriptor, moved binary, alias, symlinked content, permissions, corrupt/conflicting binding. Run locally on macOS arm64 and in CI on linux amd64. **linux arm64 NOT_RUN** (no runner in PR checks). No claim about the original executing image: `imageProof` is always `unavailable` by design |
| T03 / E03 | PARTIAL | `docs_publish.go` adds an installation-wide `lock` inside the documentation namespace; tests: concurrent publishers with different receipt roots never both own the binary, a held lock refuses without effects, fault boundaries (`generation_staged`, `generation_committed`, `binding_published`, `before_binary_commit`) leave no binary/receipt/marker and retry converges, conflicting generation and foreign files are preserved. A stale lock needs manual removal, as the receipt lock already does. Not a transaction engine and not covered by the process-level fault matrix of the earlier fixtures |
| T04 / E04 | PARTIAL | Author-side engineering/security pass over T01–T03 and the write path (path anchoring, ownership/mode checks, exact-match shell greps, bindings directory permissions). It found and fixed three items (see "Findings fixed"). It is **not** an independent review; that is the next SDD stage |
| T05 / E05 | PASS (content as authored) | `docs/operational/` ten English pages + allowlist; generated corpus has all required classes; local links close (`scripts/test-docs-offline.sh` re-checks them on the installed copy). Editorial quality is for Review |
| T06 / E06 | PASS | `cmd/axiom-docs-build` + `internal/cli/documentation.go`: two identical builds (byte-for-byte), 106 public leaves / 549 flag registrations asserted by `TestDocumentationRenderingDeterministicAndComplete` against the accepted coverage map, 15 man pages, man-map anchors on the full command path. Provenance hashes the non-test `internal/cli` sources; other transitive inputs are not hashed |
| T07 / E07 | PARTIAL | Archive generation 2: producer (`build-release-archives.sh --format 2`, `--only` row filter), Go verifier (`archive_v2.go`) and shell verifier (`install-release.sh`) validate layout, modes, strict sorted manifest, metadata, skills, documentary records and cross-hashes before candidate execution. Real archives for all three rows pass the Go loader; the shell journey rejects four resealed bad archives and the pinned v1 installer rejects v2 before any effect; the v1 metadata reader rejects generation 2. Shell/Go differential is selected cases, not an exhaustive same-set proof. Release workflows and `scripts/install.sh` still produce/consume generation 1 (default unchanged); the v2 row token is `macos` and v1 keeps `macos-27` |
| T08 / E08 | PASS (publisher) | Immutable generation tree (0400 files, 0500 dirs, staged then `PublishNew`), per-binary binding files, exact-bytes no-op, conflicts refused and preserved |
| T09 / E09 | PARTIAL | Fresh and upgrade paths publish generation → binding (staged binary inode) → binary commit → receipt 2; v1→v2 migration preserves foreign files; same-candidate reinstall is a true no-op; missing generation is repaired by reinstall; A→B keeps B verified and A's descriptor refused; downgrade refused. Receipt 2 binding hash is computed from the committed binary and cross-checked with the pre-commit binding. Fault tests use an in-process seam, not killed processes |
| T10 / E10 | PASS (fixtures) | `Resolve`: read-only, no network/extraction/repair/Runtime, returns an in-memory snapshot, refuses on every listed mismatch; `TestResolveNeverWrites` |
| T11 / E11 | PASS | `axiom [--json\|--human] docs inspect`; suffix/extra input rejected; states map to success/failure/validation_failure with exit 0/1; paths only when verified |
| T12 / E12 | PARTIAL | Four skills carry an identical "Installed documentation" section; `docs inspect` reports `skills.expected` (only when the binary's skills reproduce the installed manifest digest), per-Runtime `observed` receipt/files and `loaded: unknown`. Skill history (`sharedSkillHistory`, pinned digest) updated. Man selection remains a documented session recipe; no code launches or diagnoses a reader |
| T13 / E13 | NOT_RUN | Exact-artifact V1–V7 on three native rows needs disposable hosts |
| T14 / E14 | PARTIAL | `scripts/test-docs-offline.sh` (local macOS arm64; CI linux amd64 pending): installs without checkout, `env -i`, cwd `/`, verifies, reads all 26 files and links, refuses tamper/copy/modified binary. Network/maintainer-document denial was **not** enforced or observed; linux arm64 NOT_RUN |
| T15 / E15 | NOT_RUN | Real new Codex and Claude sessions with independent controls |
| T16 / E16 | PARTIAL | This document, user docs (`docs/commands.md`), PR comment; independent review handoff below |

## Acceptance and validation coverage

| Item | State |
| --- | --- |
| AC-01 offline reading | PARTIAL (T14 above) |
| AC-02 association | PARTIAL (resolver oracle; hosts above) |
| AC-03 publication/upgrade | PARTIAL (in-process faults) |
| AC-04 | NOT_RUN (not independently evidenced) |
| AC-05 compatibility | PARTIAL (v1↔v2 selected cases; v0.1.0 floor **UNVERIFIED**) |
| AC-06 reference parity | PASS (leaf/flag parity, determinism) |
| AC-07 man | PARTIAL (pages, map and anchors; reader selection UNVERIFIED) |
| AC-08 agents | NOT_RUN |
| V1–V7, V9 | NOT_RUN. V8 PARTIAL as T14. Q6 and migrations: not qualified |

## C1 quota measurement (no limit changed)

Generation `87313c25…` (development build): 26 outputs, 414 514 content bytes,
largest file 191 591 bytes (`cli.md`), 29 files with controls, 126 man-map rows.
All inside the initial limits (64 outputs, 1 MiB per file, 8 MiB total, 256 KiB per
control), so the initial limits are retained as the public bounds; nothing was
trimmed or raised. Values are from a dirty development build, not release provenance.

## Findings fixed during implementation

1. `docs inspect` section anchors dropped the command path for `docs` (empty `.SH`).
2. Shell provenance/man-map membership used regex `grep`; now fixed-string exact match.
3. The resolver did not check the `bindings` directories' ownership/permissions; now it does.
4. Staticcheck `U1000` on two unused helpers (CI failure at `cf69bdc`); removed.

## Known limits and decisions left to the human

- Archive generation 1 reading now ends at the first gzip member (previously a
  concatenated second member was also read). No supported producer emits one.
- Generation 2 is opt-in (`--format 2`). Making it the release default, the
  bootstrap `install.sh`/asset naming (`macos` vs `macos-27`) and the release
  workflows are untouched and need a decision before publication.
- The descriptor proves association with an intact installation, not the bytes of
  the image that is running. A same-UID attacker who rewrites binary, binding and
  generation together is not defended against (as ADR-0034 states).
- `docs inspect` reads Codex/Claude skill roots (read-only) but never starts a
  Runtime or man reader.
- No GC of old bindings/generations; no rollback; stale documentation lock needs
  manual removal.

## Full-suite note

Local `go test ./...` (macOS arm64): every package passed except one run of
`internal/executiongraph` `TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation`
(process timing, package untouched by this change). It failed in one repeated
`-count=3` run, then passed with `-count=1` here and at baseline `4b9c4fb`:
treated as an environmental flake under load, not investigated further.

## Reproduce

```sh
go build ./... && go vet ./... && go test ./...
./scripts/test-docs-offline.sh
./scripts/validate-repository.sh . && python3 scripts/check-adr-governance.py .
```

## Next action

Independent SDD Review of the implementation; then, with disposable hosts and
real sessions authorized, T13–T15 on the three native rows. No merge, release,
Linear transition or unblocking of AXM-39/AXM-12 was performed.
