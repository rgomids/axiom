# AXM-37 — Implement v3 save state (WIP)

Stopped at the user's explicit SAVE STATE request because the Codex five-hour
budget was below 10%. This is a continuation checkpoint, not implementation
completion, a new Execution, a new approval gate or SDD Review.

## Authority and repository

- Same Draft [PR #335](https://github.com/rgomids/axiom/pull/335), branch
  `docs/axm-37-offline-operational-documentation`.
- Entry local/remote HEAD: `4b9c4fb409b79d98e0c52e6a00384c7c8fa42bf8`, clean worktree.
  PR open/Draft, entry CI PASS (PR Checks run 39512817048 and delivery metadata
  run 39512856925). These results do not cover this checkpoint's new code.
- [H1 acceptance](acceptance-h1.md) binds Spec/Clarification/Plan/Tasks R1 and
  ADR-0034 to that exact entry SHA. The full Implement v3 prompt authorizes
  T01–T16; it remains the scope for continuation after this requested stop.
- Linear read-back: In Progress, blocks AXM-39/AXM-12, no blocked-by relation.
  No Linear writes, canonical Execution reconstruction or successor dispatch.
- Other Codex/Claude processes existed; no unrelated worktree changes observed.
  Three bounded lower-capability delegates handled authority, corpus and rendering;
  all stopped before checkpoint. Integration/checks were performed in the main session.

## Completed and partial work

| Task / Evidence | State in this checkpoint | Owners / exact remaining work |
| --- | --- | --- |
| Authority / CR-001 | PASS for documentary correction | `acceptance-h1.md`, ADR-0033/0034 and both indexes; historical Plan/Tasks approvals retained, H1 accepted replacement recorded. Finding [CR-001](https://github.com/rgomids/axiom/pull/335#discussion_r4331346389) corrected in source; review thread not self-resolved |
| T01 / E01 | PARTIAL | `internal/operationaldocs/records.go`: G/D/skills identities, bounded TSV validation and closed flat JSON helper with tests. Full vectors, metadata/binding/receipt codecs, rejects/bounds and schema integration remain |
| T02 / E02 | NOT_RUN for R1 | No native descriptor association implementation or C2 regression executed. Historical C2 remains FAIL; historical G1 remains BLOCKED for its former baseline, not a new R1 permission gate |
| T03 / E03 | NOT_RUN for R1 | Prior transition models preserved; no production installation-wide coordination changes |
| T04 / E04 | NOT_RUN | Safety checkpoint over completed T01–T03 remains required before write integration |
| T05 / E05 | PARTIAL | `docs/operational/`: ten English authored pages and exact basename allowlist. Local journey links close after generation. Corpus content/source mapping and final installed behavior still need review/qualification |
| T06 / E06 | PARTIAL | `internal/cli/documentation*.go`, `cmd/axiom-docs-build/main.go`: real-tree Markdown/man rendering, inventory/provenance/man-map, quota checks and generation. Current tree only; final inspect tree, exhaustive per-command flag/default parity and final C1 freeze remain |
| T07 / E07 | NOT_RUN | v2 producer, shell/Go pre-execution verification, descriptor injection and full package/old-reader matrix not implemented |
| T08 / E08 | NOT_RUN | Immutable generation/binding publisher not implemented |
| T09 / E09 | NOT_RUN | No installer lifecycle/receipt integration or actual concurrent/fault qualification |
| T10 / E10 | NOT_RUN | No installed read-only resolver/snapshot consumer |
| T11 / E11 | NOT_RUN | Public `docs inspect` is absent. Temporary incomplete tree edits were removed; public routing remains unchanged |
| T12 / E12 | NOT_RUN | Active skills unchanged; no installed inspect/help or native man integration |
| T13 / E13 | NOT_RUN | Exact-artifact V1–V7 and v1 migration/independent-return qualification outstanding |
| T14 / E14 | NOT_RUN | V8 offline host controls/journey outstanding |
| T15 / E15 | NOT_RUN | V9 real fresh Codex/Claude sessions and independent controls outstanding |
| T16 / E16 | NOT_RUN | Integrated fan-in/readiness and independent review handoff outstanding |

AC-01–08 remain **NOT_RUN for the integrated R1 product**. AC-06 has partial
local renderer evidence only. V1–V9 are NOT_RUN for final R1 artifacts; Q6 and
the historical v0.1.0 migration-floor objective remain UNVERIFIED. No qualification
is inferred from unit checks, generated text or prior native experiments.

## Validation actually performed

- PASS: `go test ./internal/cli ./cmd/axiom-docs-build ./internal/operationaldocs`.
  The build command currently has no dedicated Go test file; full corpus execution
  was checked separately. Core tests bind descriptor inputs, exact active skills,
  duplicate/escaped-duplicate/unknown/null/trailing JSON refusals and renderer output.
- PASS: `go vet ./internal/cli ./cmd/axiom-docs-build ./internal/operationaldocs`.
- PASS: `go build ./...`.
- PASS: `./scripts/validate-repository.sh .` and
  `python3 scripts/check-adr-governance.py .` (35 ADRs).
- PASS: two stable full documentary builds with version `0.0.0-axm37.wip`,
  `--source-root .`, and entry SHA supplied to `--commit`; byte/hash comparison of
  all 28 files; operationaldocs.Validate before output publication; 37 local links
  and target anchors checked after generation.
- Stable measurement: 25 outputs, 412068 content bytes, 14 man pages;
  inventory 2655 bytes, man-map 9391 bytes, provenance 75298 bytes;
  generation `b30b4c3fd4d81ffa3cfb98a90c07fdf440b5ff0ba3facd08e94455cfb626ebb1`.
  This is dirty-worktree development output, not release provenance or C1 freeze.
  Final `docs inspect` adds a leaf/page and requires regeneration before packaging.
- PASS: `git diff --check` and `gitleaks dir . --no-banner --redact --exit-code 1 --log-level error`.
  Staged sensitive-file check and final push/CI read-back are recorded in the PR
  save-state comment with the actual resulting commit SHA.

An earlier test attempt caught the unfinished `docsInspectAction` reference;
it was removed before the passing checks above. An initial two-build comparison
differed while a delegate was still editing generator inputs; the stable rerun
above passed after all delegate writes stopped. Neither failed intermediate run
is final artifact qualification. Full `go test ./...`, native suites, shell/Go v2
differential checks, release packaging and real Runtime trials were not run.

## Next exact action for Claude

1. Fetch/read the save-state comment on PR #335, verify its commit against remote
   HEAD, local branch/worktree/CI/reviews and concurrent activity. Reuse this branch,
   PR and logical Implement; read AGENTS.md and relevant skills/policies. No new PR,
   Execution or human microgate.
2. Review this WIP and accepted R1 contracts. Complete T01 codecs/vector negatives
   and T05/T06 content/parity/source mapping. Renderer provenance includes all 27
   production CLI sources; audit other transitive registration/transform inputs.
   Generation takes supplied commit without proving clean source equality; packaging
   must enforce exact source/artifact provenance. Do not treat scratch measurements
   as a final quota freeze.
3. Continue causal graph: T01→T02/T03→T04 and T01/T05→T06→T07;
   T04/T07→T08→T09 and T04/T07→T10→T11→T12. Regenerate T06 after final T11;
   T09/T12→T13/T14/T15→T16. Preserve C1 coverage/measurement before T07.
4. Explicit compatibility point to reconcile: current v1 release row token is
   `macos-27`; accepted documentary R1 uses `macos`. Preserve historical v1 bytes
   and closed readers, make new row normalization explicit and test both routes.
   Also resolve how complete release metadata is independently available to the
   resolver and how binding gets the staged binary inode before binary commit;
   these were identified, not implemented or approved by inference.
5. Use only previously authorized disposable hosts/observers for native rows.
   Their availability/authority was not freshly established in this checkpoint.
   Record missing rows NOT_RUN/UNVERIFIED and group real residual decisions.

No release/merge/force-push, Linear transitions, review-thread resolution, installed
Runtime mutation, credential/infra changes or functional acceptance occurred.
