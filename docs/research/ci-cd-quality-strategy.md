# CI/CD Quality Strategy and Release Validation Gates

## Status

Research for [#154](https://github.com/rgomids/axiom/issues/154), executed on
2026-10-05/06 against `main` at `e59bfe9` (v0.5.0 is the latest stable
release). It refines the issue's
[initial framing](https://github.com/rgomids/axiom/issues/154#issuecomment-6007508302)
and [failure-class matrix v0](https://github.com/rgomids/axiom/issues/154#issuecomment-6007550025)
checkpoints.

This document is a recommendation. It changes no workflow, test, script,
ruleset, release or product code, and it is not an ADR. Every statement about
the current state is marked by its source; inferences are labelled
**(inference)**. Decisions that need a human are listed in
[Open human decisions](#open-human-decisions).

Revision 2 (2026-10-06) adds [Structural quality](#structural-quality) in
response to the [PR #226 review](https://github.com/rgomids/axiom/pull/226).
It is reconciled into the failure-class matrix (F32–F38), the test taxonomy,
the target pipeline, the gap analysis (G12–G18) and slices 15–22.

Reading order: [Summary](#summary) →
[#153 tracer](#trigger-regression-153-as-tracer) →
[Recommended target pipeline](#recommended-target-pipeline) →
[Implementation decomposition](#implementation-decomposition).

## Summary

1. **The PR/main baseline is stronger than the issue assumed.** Required
   `upgrade-journeys (linux|macos)` checks (#188) already run real published
   N, v0.1.1 and POC state through a candidate installer on every PR and
   `main` push. The fast layers (Go race tests on three OSes, contracts,
   dogfood smoke) take about 4 minutes of wall-clock time. They are
   deterministic and hermetic.
2. **The release boundary is the weak point.** The release flow builds once
   and publishes the same bytes. Prepare → envelope → publish preserves
   artifact identity by digest. But the prepared bytes are only *structurally*
   verified: the closed file set, checksums, manifests, headers, build info,
   and a `version` smoke on 1 of 4 rows. **Nothing installs, upgrades or runs
   the prepared bytes before publication.** Upgrade journeys test a
   source-equivalent rebuild (`9999.0.0-acceptance.N`), not the candidate.
3. **#153 was a policy gap and a mechanism gap.** No candidate-level upgrade
   gate existed (mechanism). Even with one, the gate would have encoded the
   ambiguous HD-4 refusal as correct, because supported upgrade sources were
   never declared (policy). #186 had no policy ambiguity: the same missing
   gate alone let it ship. Both bugs came from **older state generations**,
   not from N-1, so an `N-1 → N` rule alone is insufficient.
4. **Platform coverage:** macOS arm64 and Linux amd64 have native hosted
   coverage. **Linux arm64 has no execution anywhere in CI**, even though
   `ubuntu-24.04-arm` is free for this public repository. **Windows client
   amd64 has no native hosted runner.** `windows-2022` Server is a regression
   proxy only. #126 was closed without its automation.
5. **Recommendation:** add a *candidate acceptance* stage **inside the
   existing prepare run**. It runs on native rows against the exact uploaded
   bytes. Bind its Evidence into the publication envelope. Add a bounded
   post-publish smoke through the public installer. Declare a
   generation-based upgrade-source policy. No extra stage is added to PR
   latency.
6. **Structural quality (revision 2):** CI has `go vet` and GitHub-managed
   CodeQL, Code Quality and Dependabot, and nothing else.
   - **Blocking gates (deterministic, objective defect):**
     - `gofmt`/`go mod tidy`;
     - curated `staticcheck`, including unused code (12 findings locally, 2
       of them false positives);
     - `govulncheck`. This is the most valuable addition: nothing checks the
       standard library or toolchain that every published binary embeds.
       It runs in binary mode on the prepared bytes at release.
   - **errcheck:** promoted to blocking in stages, for durability-critical
     packages.
   - **Signals only:** coverage (per package and diff, no global
     percentage), complexity delta and duplication.
   - **Scheduled:** mutation testing, scoped to the critical decision core.

## Method

| Phase | Source |
|---|---|
| Current state | `.github/workflows/*.yml`, `.github/rulesets/*.json`, `scripts/` (every validation script and installer), 139 `*_test.go` files, `scripts/automation-registry.json`, Specifications 004/006, ADR-0015/0016/0017, `.agents/policies/quality.md` |
| History | Issues #153, #186, #126, #80, #81; PRs #187, #188; `gh release list`; live rulesets and environments (`gh api`) |
| Cost/duration | `gh run list` / `gh run view` for the last 30 runs of each workflow, as of 2026-10-06 |
| External research | Primary GitHub, Go and Sigstore documentation only, after the gap map ([External research](#external-research)) |

Verification: key subagent findings were re-checked directly. These were:
no workflow invokes `install.sh` or `test-install-bootstrap.sh`; journey 2
seeds no state; `v0.1.0:internal/install/upgrade.go:211` already contains
the `state_incompatible` gate; #126 is closed as `COMPLETED` with no
implementing PR; and the repository is public.

---

## Axiom current state

### Workflows and gates

| Workflow | Trigger | Jobs / runner | Role | Gate |
|---|---|---|---|---|
| `ci.yml` | PR, push `main`, dispatch (Release PR) | `verify` × {ubuntu-24.04, macos-15, windows-2022}; `release-contract` (ubuntu); `upgrade-journeys` × {ubuntu-24.04 → `linux-amd64`, macos-15 → `macos-27-arm64`} | Product regression | **Merge-blocking**: all 6 contexts are required (`main.json`, strict) |
| `delivery-metadata.yml` | PR, dispatch | ubuntu | PR title / Issue relationship | Merge-blocking (required) |
| `release-please.yml` | dispatch by `release.sh start` | ubuntu | Release PR from validated `main` SHA; dispatches CI on the Release PR | Release start control |
| `release-artifacts.yml` | dispatch by `release.sh prepare` | `prepare` on ubuntu-24.04 | **Release prepare**: preflight, build 4 rows once, verify, notes, upload `axiom-release-<tag>` (30 days) | **Release-blocking**: `verify-prepared-release.sh` requires `conclusion == success` |
| `publish-release.yml` | dispatch by `release.sh publish` | `preflight`, `publish` (environment `release`, required reviewer) | **Release publish**: download the prepared run, re-verify, recompute the envelope, which must equal the human-authorized `preview_digest`, then publish | Human authority + environment approval |
| `delivery-sync.yml` | push `main` | ubuntu | Issue/Project delivery projection | Informational (post-merge effects) |
| `issue-label-policy.yml`, `deploy-landpage.yml` | Issues / `site/**` | ubuntu | Governance / site | Not product gates |

Other release-boundary controls (fact):

- `release.sh status` blocks `prepare` unless the required CI on the release
  revision is `success` (`revision_ci`, `scripts/release.sh:593`). That is
  client-side; the workflows themselves do not re-check it, except for
  repair.
- The tag ruleset forbids tag update and deletion. Immutable releases are
  enabled (`docs/security/repository-security.md`).
- `release.sh verify --download` reads the published release back after
  publication. It is digest-only: it re-runs `verify-release-artifacts.sh` on
  downloaded assets and **does not install them**.

### Validation-surface inventory

**Subject** is the artifact the surface exercises:

- **S**: source tree, in-process;
- **B**: binary built from source in the job;
- **C′**: release archive rebuilt from the same source, which is *not* the
  candidate;
- **P**: the prepared candidate bytes;
- **R**: a published release asset.

**Env** is the execution environment:

- **H**: hosted runner;
- **F**: fakes/stubs (fake `gh`, `curl`, `codex`/`claude`, `sw_vers`);
- **N**: native for a supported row.

Durations are medians from recent successful runs (fact). Other costs are
estimates.

| # | Surface | Failure class it protects | What it actually proves | Subj. | Env / platform | Stage | Blocking | Determinism / flake risk | Duration | Evidence |
|---|---|---|---|---|---|---|---|---|---|---|
| V1 | `go test -race ./...` (687 tests / 139 files) | Unit/domain, component, CLI, fault, security-FS, installer, compatibility | In-process and real-FS behaviour; SIGKILL crash points; injected ENOSPC/EIO; symlink/ACL. About 11 `go build` black-box CLI runs | S, B | H+F; Linux ext4, macOS arm64 APFS | PR, main | Merge | High. Risks: 50 ms real-clock timeouts (`executiongraph`), `time.Sleep(time.Hour)` helpers killed on a marker, 52 `range map` uses | ~1.7 min (linux), 2.5 min (macOS) | Log only |
| V2 | `go test ./... -timeout 10m` (Windows, no `-race`) | Same, plus Windows-only tests (junctions, ACL, 6 install tests) | Windows FS semantics on **Server 2022**; POSIX-shell tests skipped (`testfs.POSIXShell`) | S, B | H; windows-2022 NTFS (not an approved row) | PR, main | Merge | High; NTFS timestamp caveat handled | ~4.2 min (job) | Log |
| V3 | `go vet`, `go build`, `go mod verify` | Build/static/module integrity | Compiles for the host; module sums match | S | H × 3 | PR, main | Merge | Deterministic | in V1/V2 | Log |
| V4 | `validate-repository.sh` (calls `check-*`, harness, registry, ADR governance, sensitive files) | Repository-invariant and governance drift, committed secrets | Static hygiene; **not** product behaviour | S | H; Linux, macOS | PR, main | Merge | Deterministic | seconds | stdout |
| V5 | `dogfood-poc.sh` | CLI lifecycle E2E regression | First-run, configure, Work Item, full workflow lifecycle, Provider projection replay; **`version=development`** binary | B | H+F | PR, main | Merge | High | seconds | One JSON stdout line |
| V6 | `test-windows-install.ps1`, `test-windows-bootstrap.ps1` (PS 5.1 and 7) | Windows installer/bootstrap regression | Install contract with a go-built `axiom.exe`; bootstrap refusal over an in-memory transport | B | H+F; Server 2022 | PR, main | Merge | High | in V2 | stdout |
| V7 | `release-contract`: `test-release-pipeline.sh`, `test-release-flow.sh` (2.6k lines), `test-delivery.sh`, `test-release-corrections.py`, `test-release-pr-checks.py` | Release-flow and authority regressions | Build/verify scripts on a clean clone with tamper mutations; orchestration logic against a stateful fake `gh`; workflow static shape | S, C′ | H+F | PR, main | Merge | High | ~4.0 min | stdout |
| V8 | `upgrade-journeys` → `test-upgrade-journeys.sh` | Upgrade, persisted-state compatibility, historical-format transition, reinstall idempotency, skill-receipt convergence | **R → C′**, through the candidate archive's *bundled* installer, on 2 rows. Journeys: (1) N with N-written non-empty state plus the frozen v1 corpus → byte-preserved, `valid_v1`, usable, first-run no-op, rerun `unchanged`; (2) v0.1.1 → skills/receipt convergence, **no state seeded**; (3) POC → N → candidate preserve/rebuild with an independent inventory | R, C′ | H+F (fake Runtimes and Providers, `sw_vers` stub); linux-amd64, macOS arm64 | PR, main, Release PR | Merge | Medium: network downloads of published assets (`curl --retry 4`), the POC `go build`, availability of the newest tag's assets | ~1.0–1.2 min | Step summary (key=value); nothing retained |
| V9 | `release-artifacts.yml`: `release-preflight.sh`, `build-release-archives.sh`, `verify-release-artifacts.sh` | Artifact integrity, wrong revision/tag | **P**: closed set, SHA256SUMS, closed entries/modes, MANIFEST, metadata, installer/skills equal to source, executable header per row, `go version -m` (`vcs.revision`, `vcs.modified=false`), `--json version` on **linux-amd64 only** | P | H; ubuntu-24.04 | Release prepare | Release | Deterministic (preflight reads the remote) | ~0.7–1.0 min | `release-evidence.txt`, `preflight.txt`, `prepare-metadata.txt` in the uploaded artifact (30 days) |
| V10 | `verify-prepared-release.sh` + `publish-release.sh --envelope` | Artifact identity drift between prepare and publish; changed preview | Same prepared run, successful dispatch from `main`; re-verification equals prepared Evidence; notes regenerate identically; the envelope (tag, revision, run, every artifact SHA-256, effects) equals the authorized digest | P | H | Release prepare (status), publish | Release | Deterministic | <1 min | Envelope lines, step summary |
| V11 | `release.sh verify --download` | Publication regression | Published tag/revision/latest/assets equal the prepared set; assets re-verified structurally | R | Maintainer host | Post-publish | Manual | Deterministic | — | stdout |
| V12 | Corpus/inventory guards (`TestEveryV1WriterIsRecognizedByInventory`, `TestStableCorpus*`, published-receipt fixtures) | Persisted-state compatibility at the component layer | Every v1 writer is classified; every frozen snapshot resolves to `direct`; every published receipt is recognized | S | H | PR, main (in V1) | Merge | Deterministic | in V1 | — |
| V13 | Manual-only durable scripts: `test-install-bootstrap.sh`, `test-install-posix-facade.sh`, `test-install-axiom.sh`, `test-release-archives.sh`, `test-s7-native.sh`, `test-s7-security.sh`, `test-codex-skills.sh`, `test-check-project*.sh` | Canonical `install.sh` bootstrap, facade, native-row and security regressions | Real logic against fake transport/fixtures | S, B, C′ | Maintainer host | **Manual** (registry: no callers) | None | High | — | stdout |
| V14 | `scripts/acceptance/s9-rc-*.py`, `s9-graph-runner.go` | Real install, upgrade and Runtime journey on a published RC | Real network, the real `install.sh`, published RC; Runtime executables located, never executed by the installer path; real Codex/Claude graph runs in T24 | R | Maintainer host; human-attested Linux rows | Manual, per RC | Human acceptance | Medium | Hours (human) | Ledgers, `acceptance-evidence.json` (T24) |
| V15 | `TestPOSIXInstallerBaselineDifferential`, `TestRecognizedPOCCrossFilesystemArchive` | Installer differential; cross-FS archive | Env-gated (`AXIOM_POSIX_BASELINE_SCRIPT` is unset in CI, so it is skipped); cross-FS runs only on Linux via `/dev/shm` | S | H | PR (partial) | Merge when executed | — | — | — |
| V16 | GitHub code scanning: CodeQL default setup (`actions`, `go`, `javascript-typescript`, `python`; `default` suite) and the ruleset `code_quality` rule (severity `errors`) | Security-pattern defects (taint, injection, workflow misuse); code-quality findings of severity *error* | Query-based findings on source. `CodeQL`/`Analyze (*)` are **not** required status checks. The `code_quality` rule gates merges on error-severity Code Quality findings; which analysis feeds it is not distinguishable in the check runs (**unverified**) | S | GitHub-managed | PR, main | CodeQL: informational alerts (1 open, in `site/`); Code Quality: merge rule | Query-deterministic | GitHub-managed | Code-scanning alerts |
| V17 | Dependabot alerts + security updates; secret scanning with push protection | Known-vulnerable module versions; leaked secrets | Module-level advisory match on `go.mod` (no reachability, **no standard-library/toolchain coverage**); secret patterns on push | Manifest | GitHub-managed | Continuous | Security updates open PRs; push protection blocks pushes | Deterministic | — | Alerts |

Structural-quality tooling beyond V3, V16 and V17 (format checks,
staticcheck, errcheck, coverage, complexity, dead code, duplication,
govulncheck, mutation) is **absent**. See
[Structural quality](#structural-quality).

Notable facts from the inventory:

- **No test makes a real external call.** There is no `net/http`,
  `httptest` or listener in Go tests. GitHub is faked through a `gh` shell
  stub or the test binary itself. Codex and Claude are no-op stubs;
  execution-graph invocations use interface fakes.
- **`install.sh`, the canonical public installer, never runs in CI.** Its
  only automated tests use a fake `curl` and are manual-only (V13). Real
  `releases/latest` behaviour is exercised only by manual RC acceptance
  (V14).
- **Stable-corpus freshness is not enforced.** `.agents/policies/quality.md`
  says to freeze the writer output "before release" with
  `AXIOM_FREEZE_STATE_CORPUS`. The test only freezes when the variable is set
  (`internal/local/state_compatibility_contract_test.go:148`). Unfrozen
  drift in writer output fails nothing. Only `snapshots/v0.4.0` exists for
  10 stable releases (it is valid when later writers emit identical bytes,
  but nothing proves that they do).
- **Evidence is mostly stdout.** Only V8 writes a step summary. The only
  durable release Evidence is the prepared artifact, kept 30 days. There are
  no JSON or JUnit reports and no retained failure logs beyond GitHub's log
  retention.

### Artifact flow and identity

```text
release.sh start ─► release-please.yml ─► Release PR (CI dispatched on its head)
   human merges ─► release commit R on main ─► ci.yml push (incl. upgrade-journeys on C′(R))
release.sh status: revision_ci(R)==success ─► release.sh prepare
release-artifacts.yml run P:  checkout R ─► preflight ─► build 4 rows ONCE ─► verify (structural) ─► upload axiom-release-<tag>
release.sh status: download P ─► verify-prepared-release ─► envelope(preview_digest) ─► HUMAN authorizes digest
publish-release.yml: download P (run-id) ─► re-verify ─► envelope == authorized ─► env `release` approval ─► draft ─► assets ─► publish ─► read-back
release.sh verify --download (post-publish, digest-only)
```

| Identity property | Current state | Evidence |
|---|---|---|
| Revision | Full SHA input; `HEAD == revision`; clean tree; first-parent of `main`; release commit for stable | `release-preflight.sh`, `verify-release-artifacts.sh` (`vcs.revision`, `vcs.modified=false`) |
| Tag / version | Preflight; the version is embedded in the binary via `-ldflags` and checked in `release-metadata.txt` | `preflight.txt`, `release-evidence.txt` |
| Workflow run | `prepare-metadata.txt` (run URL, attempt, workflow ref, Go version); the envelope binds `prepared_run` | Prepared artifact |
| Hashes | Per-archive SHA-256, per-executable SHA-256, per-bundle MANIFEST digest, `SHA256SUMS` digest | `release-evidence.txt`; envelope |
| Handoff | Download by **name + run-id**. Not pinned by `artifact-id` or artifact digest; the content is re-verified against the revision and must equal the authorized envelope | `publish-release.yml` |
| Provenance | Go build info only. No artifact attestation. Release attestation comes from GitHub immutable releases (external fact, below) | — |
| **Build once → publish same bytes** | **Yes**: publish never rebuilds | V9/V10 |
| **Validate the same bytes** | **No.** Behavioural validation (V8) runs on C′, rebuilt with a different version string (`9999.0.0-acceptance.N`). By construction those are different bytes, and the real version string is never exercised in an upgrade. P gets structural checks plus 1-row `version` | `ci.yml` build step |

A rebuild could not prove identity even if desired (inference, supported by
external research): `tar`/`gzip` embed mtimes and ownership, and the binary
embeds the version. Rebuild-and-compare is therefore not a viable substitute.
Validation must consume P itself.

### Platform reality

The support policy comes from Spec 006 and ADR-0015 and is defined by OS
family and architecture: Linux amd64 and arm64, macOS arm64, and Windows
client amd64. OS version is Evidence, not eligibility.

| Row | Hosted coverage today | Fidelity |
|---|---|---|
| `macos-27-arm64` | `macos-15` (arm64, APFS) in V1, V5, V8 | **Native** for the row (family/arch); OS version recorded only implicitly |
| `linux-amd64` | `ubuntu-24.04` (ext4) in V1, V5, V8 | **Native** for one distro/filesystem (Ubuntu/ext4) |
| `linux-arm64` | **None.** Built and header-checked only (V9) | Never executed |
| `windows-amd64` (client) | `windows-2022` **Server** in V2 and V6 (no upgrade journey) | Regression proxy; the CI comment itself says it is "not an approved installation row" |

Simulated contracts are F-class fakes, not native acceptance:

- the `sw_vers` stub (for pre-ADR-0015 bundles);
- fake `curl` (bootstrap);
- fake `gh`;
- Runtime stubs;
- errno injection.

**#126 today (fact):** it is closed as `COMPLETED` with no implementing PR.
`test-s7-native.sh` has no caller, and `ubuntu-24.04-arm` has never appeared
in any workflow. Linux amd64/arm64 native Evidence was human-attested for
v0.1.2-rc.2 (T24) and has not been repeated since. The closure does not
reflect delivered automation.

### Cost and reliability baseline

- The repository is **public**: standard hosted runners, including arm64 and
  macOS, cost nothing.
- PR feedback: about 4.2 min median wall-clock, bounded by `verify (windows)`
  and `release-contract`.
- Release prepare takes about 1 min. Publish takes about 1 min plus human
  approval.
- Recent failures (30 runs per workflow):
  - CI: 2 failures. One was a genuine Windows test failure on a PR branch.
  - Release workflows: 6 failures. All were control-plane or metadata
    defects (missing delivery metadata, 403 on draft creation, notes
    mismatch in recovery). None was a flaky test.
  - Reruns (`run_attempt > 1`): rare; the visible causes were cancellations.
  - No evidence of flaky product tests in the sampled window.

---

## Failure-class matrix

This is v1, refined from the v0 checkpoint. Changes from v0:

- **Split:** *Upgrade compatibility* into *owned upgrade mechanics* and
  *Runtime skill/receipt convergence* (#186 is a distinct mechanism).
  *Platform-specific regression* into *OS/arch execution*, *filesystem
  semantics* and *hosted-proxy divergence*.
- **Added:** *Upgrade-source policy ambiguity*, *Compatibility corpus drift*,
  *Candidate version-string-dependent behaviour*, *Canonical bootstrap
  installer regression* (pre-publication, distinct from post-publish
  selection) and *External-dependency-induced gate failure*.
- **Merged:** *Contract regression* is folded into the component and CLI
  rows, because no separate stable external API surface exists yet.
  *Evidence insufficiency* and *Flaky gate* moved to cross-cutting policy
  ([Evidence](#evidence), [Flakiness](#flakiness-and-failure-policy)).
- **Removed as a class:** *Performance/timeout regression*. It becomes a
  timeout policy attribute of every gate; no product performance contract
  exists.

Notation: the **Detecting surface (today)** column names V-numbers from the
inventory. **✓** means trustworthy protection at the right subject. **◐**
means partial (wrong subject, wrong stage, fake environment or a missing
row). **✗** means none.

| ID | Failure class | Example | Detecting surface (today) | Subject / env / stage | Today | Target boundary | Gap |
|---|---|---|---|---|---|---|---|
| F01 | Build / compilation | Does not compile; a row cannot be produced | V3; V9 | S/H/PR; P/H/prepare | ✓ | PR; prepare | — |
| F02 | Static / module / repository invariant | `go.sum` drift; registry or ADR rule | V3, V4 | S/H/PR | ✓ | PR | — |
| F03 | Unit / domain | Wrong rule result | V1, V2 | S/H/PR | ✓ | PR | — |
| F04 | Concurrency / race | Unsafe shared state | V1 (`-race`, not on Windows) | S/H/PR | ✓ (◐ on Windows) | PR | Low |
| F05 | Component integration | Stores or coordination disagree | V1, V2 | S/H/PR | ✓ | PR | — |
| F06 | CLI behaviour | Exit code, envelope or rendering | V1 black-box, V5 | B/H+F/PR | ✓ | PR | — |
| F07 | Security / filesystem boundary | Traversal, symlink, ACL, overwrite | V1, V2; manual V13 (`test-s7-security`) | S/H/PR | ✓ (Linux/macOS/Server) | PR | Windows client NTFS is not native |
| F08 | Recovery / interruption | Crash leaves ambiguous state | V1 (SIGKILL, five-boundary resume) | S/H/PR | ✓ component | PR (component); candidate (one smoke boundary) | No candidate-level interruption check (low priority) |
| F09 | Fresh install / first-run | Clean host cannot install or bootstrap | V8 implicitly (installs N); V6 (Windows); V14 manual | C′ or R/F/PR; manual | ◐ | **Prepare (P, native rows)** | **Not on P**; linux-arm64 never |
| F10 | Reinstall idempotency | Rerun mutates or fails | V8 rerun `unchanged` | C′/F/PR | ◐ | PR (C′) + **prepare (P)** | Not on P |
| F11 | Owned upgrade mechanics | N → N+1 receipt/binary swap fails | V8 journey 1 | R→C′/F/PR | ◐ | PR + **prepare (R→P)** | Not on P; no Windows journey |
| F12 | Persisted-state compatibility (#153) | State valid under N blocks N+1 | V8 (1, 3), V12 | R→C′; S | ◐ | PR + **prepare (R→P)** | Not on P |
| F13 | Historical-format transition (#153) | POC preserve/rebuild wrong | V8 (3), V1 (15+5 POC tests) | R→C′; S | ◐ | PR + **prepare** | Not on P |
| F14 | Runtime skill/receipt convergence (#186) | Upgrade ends `partial`; Claude conflict | V8 (2), V1 receipt fixtures | R→C′; S | ◐ | PR + **prepare** | Journey 2 seeds no state (inference: acceptable for this class) |
| F15 | **Upgrade-source policy ambiguity (new)** | A gate encodes a refusal that the product contract forbids (HD-4 vs FR-065) | **None** | — | ✗ | Spec/ADR + generation registry, checked at PR | **Policy gap** |
| F16 | **Compatibility corpus drift (new)** | A writer's output changes and no snapshot is frozen before release | V12 only checks existing snapshots | S | ✗ | **Prepare** (freshness check) | Freeze not enforced |
| F17 | **Version-string-dependent behaviour (new)** | Ordering, receipt or compatibility window under the real version | Nothing runs the real version except the 1-row `version` smoke | — | ✗ | **Prepare (P)** | C′ uses the synthetic version |
| F18 | Release artifact integrity | Wrong entries, checksum, metadata, header | V9, V10, V7 mutations | P/H/prepare | ✓ | Prepare | — |
| F19 | Artifact identity (validate ≠ publish) | Tests pass on a rebuild, not on published bytes | Publish identity ✓ (V10); validation identity ✗ | — | ◐ | Prepare → publish binding | **Core gap** |
| F20 | OS/arch execution | Binary or installer fails on a supported row | V1/V8 macOS arm64, Linux amd64 | B, C′ | ◐ | Prepare (P) per row | **linux-arm64: none**; Windows client: proxy |
| F21 | Filesystem semantics | APFS/ext4/NTFS rename, ACL, space, cross-device | V1 (APFS, ext4, Server NTFS; cross-FS Linux only) | S | ◐ | PR (component) + manual native Evidence | No cross-FS on macOS in CI; NTFS client not native |
| F22 | Hosted-proxy divergence (new split) | Passes on Server 2022 / Ubuntu, fails on client or another distro | None | — | ✗ | Supplemental Evidence (scheduled/manual) | Accept as residual risk with explicit rows |
| F23 | Canonical bootstrap installer (pre-publication) | `install.sh` selector, checksum or metadata logic broken | V13 manual; V6 Windows | B/F/manual | ◐ | **PR** (fake transport) + **prepare** (P via local transport) | `install.sh` never in CI |
| F24 | Canonical selection / published install | `releases/latest` resolves wrong; published asset not consumable | V14 manual per RC; V11 digest-only | R | ✗ automated | **Post-publish smoke** | No automation |
| F25 | Runtime bootstrap | Codex/Claude discovery or skill install | V1, V5, V8 with stubs | F | ◐ (contract only) | PR (contract) + scheduled/manual real | No real Runtime check |
| F26 | Provider integration (GitHub) | Real `gh`/API behaviour differs from fake | V1 fakes; T24 manual | F | ◐ (contract only) | PR (contract) + scheduled sandbox | No real Provider check |
| F27 | Secret / credential handling | Token in logs or artifacts | V4 sensitive files; workflow token scoping | S | ◐ | PR + workflow review | Out of scope beyond policy |
| F28 | Release-flow regression | Wrong revision, rebuild, authority bypass | V7 (fakes), V9/V10 real | S/F; P | ✓ | PR + prepare/publish | Real GitHub only exercised at release time (accepted) |
| F29 | Publication regression | Assets/tag inconsistent | `publish-release.sh` read-back; V11 | R | ✓ (digest) | Publish | — |
| F30 | Delivery metadata | Wrong Issue relationship or notes | Delivery-metadata check, V7, V10 notes regeneration | S | ✓ | PR / prepare | — |
| F31 | **External-dependency gate failure (new)** | The newest tag's assets are missing or the network fails, so a required check fails unrelated to the PR | V8 depends on it | — | — | Classified infra failure (policy) | Failures are not classified |

Structural-quality classes, added in revision 2 (see
[Structural quality](#structural-quality)):

| ID | Failure class | Example | Detecting surface (today) | Subject / env / stage | Today | Target boundary | Gap |
|---|---|---|---|---|---|---|---|
| F32 | Formatting / module hygiene drift | Unformatted file; `go.mod`/`go.sum` not tidy | None enforced (gofmt clean only by maintainer discipline; `go mod tidy -diff` clean today) | S | ✗ | PR (blocking) | No check |
| F33 | Statically detectable defect | Dead store, impossible condition, misuse of an API | V3 `go vet`; V16 CodeQL (security-oriented) | S | ◐ | PR (blocking, curated `staticcheck`) | No bug-pattern analysis beyond vet |
| F34 | Unchecked error on an effectful call | Ignored `Close`/`Sync`/`Rename` error loses durability or hides partial failure | None | S | ✗ | PR (signal → blocking in critical packages) | No check |
| F35 | Dead / unreachable code | Unused helper that suggests a missing call; stale API | None (6 unused functions found locally) | S | ✗ | PR (unused, blocking); scheduled (reachability report) | No check |
| F36 | Reachable known vulnerability (module **or standard library/toolchain**) | Advisory in a stdlib package the binary calls | V17 (modules only, no reachability) | S; P | ◐ | PR (on dependency/toolchain change), scheduled `main`, **prepare (binary mode on P)** | No stdlib/toolchain or reachability check; release toolchain is the exact `go` directive patch |
| F37 | Test-adequacy erosion | Changed lines not exercised; tests execute code without asserting on it | None (no coverage collected in CI) | S | ✗ | PR (diff coverage signal); scheduled (mutation, critical packages) | No visibility |
| F38 | Maintainability hotspot growth (signal, not a defect) | A critical function's complexity keeps growing; duplicated logic diverges | None | S | — | PR (delta signal); scheduled report | No visibility |

---

## Trigger regression: #153 as tracer

### What happened (fact)

- The stable channel resolved v0.2.0 on macOS arm64.
- The archive verified, and the bundled installer refused with
  `owned upgrade refused: … state_incompatible`.
- The host held state written by the `v0.1.0-poc.1` prerelease (POC
  workflow records).
- `internal/install/upgrade.go` accepted only `AbsentV1` and `ValidV1`. That
  gate is present at `v0.1.0:internal/install/upgrade.go:211`, came in with
  S7 PR #100, and shipped in every stable release from v0.1.0 to v0.4.1.
- It was fixed in v0.4.2 by #188 (preserve → rebuild → reconfigure,
  ADR-0017) and #187 (#186).

### Classification

| Question | Answer |
|---|---|
| 1. Failure class | F13 historical-format transition + F12 persisted-state compatibility. **Root cause also F15**: HD-4 (recognized POC state is preserved, no automatic migration) contradicted FR-065/T39 (convergent owned upgrade). The installer's refusal was *correct under one approved contract*. |
| 2. Which checks could have detected it | At the time: none. S9 RC acceptance (T24/AC-19) tested `rc.1 → rc.2` owned upgrade, which uses the same state generation and almost empty state. Go tests asserted the refusal *as intended behaviour*. Today: V8 journey 3 and the POC Go tests detect it on C′. |
| 3. Why it reached a release | (a) No supported-upgrade-source declaration: nobody had stated "a host holding recognized POC state is a supported upgrade source". (b) No gate ran *older-generation non-empty state → candidate*. (c) Upgrade testing in T24 meant "previous RC with fresh state", not "each state generation users can have". |
| 4. Which gate should have blocked | **Release-prepare candidate acceptance**: the upgrade-source matrix over declared *state generations* (POC generation included), through the installer, against the prepared bytes. PR coverage (V8 today) is early regression feedback, not the release guarantee. |
| 5. Artifact | The prepared v0.2.0 `macos-27-arm64` archive (P), the exact bytes the stable channel later served. |
| 6. Environment | Native macOS arm64 (APFS). Hosted `macos-15` is sufficient under ADR-0015, with Runtime stubs. No real Runtime or Provider was needed: the failure happens before any external call. |
| 7. Evidence | Per journey: source generation and version, the archive SHA-256 under test (equal to the envelope), the pre-upgrade classification, the installer result category, the preservation manifest digest, the post-upgrade classification, the usability probes, the rerun result, row and OS facts, run and attempt. Its digest is bound into the publication envelope. |

**Calibration result:** the mechanism (candidate gate) is necessary but not
sufficient. Without F15, a candidate gate would have passed v0.2.0 while
asserting the refusal. #186 (v0.1.1 → v0.4.1 ended `partial`) had no policy
ambiguity: a generation-based candidate gate alone would have blocked it.
Both bugs came from **older generations**: POC state and five-skill
receipts. Neither came from N-1. **An `N-1 → N` matrix would have missed
both.**

### The required regression path, evaluated

| Step | Covered today on C′ (PR/main, V8) | Required on P (release prepare) |
|---|---|---|
| published/supported N → install | ✓ (N, v0.1.1; POC via a source build of the tag) | Same sources; N must be the release being superseded |
| Representative non-empty persisted state | ✓ journey 1 (N-written plus frozen corpus) and journey 3 (POC); ✗ journey 2 | Same |
| Prepared candidate N+1 | ✗ (C′, version `9999.0.0-acceptance.N`) | **P, real version** |
| Canonical installer | ◐ bundled `install.sh` from the candidate archive; public `install.sh` bootstrap not used | Bundled installer **and** public `install.sh` over a local transport serving P (real `releases/latest` only post-publish) |
| Transition (preserve/rebuild/migrate) | ✓ | Same |
| Verify N+1 usable | ✓ (project show, workflow status, Runtime status, first-run) | Same |
| Reinstall → idempotent no-op | ✓ (`unchanged`, archive unchanged) | Same |

PR regression coverage stays. It is cheap (about 1 min) and catches the
class before merge. Release-candidate acceptance is a different
responsibility: it proves *these bytes* are safe to publish.

---

## Structural quality

Added in revision 2, in response to the
[PR #226 review](https://github.com/rgomids/axiom/pull/226). Behavioural
gates prove what the product does. Structural signals look at the code
itself.

Principles applied:

- A **binary gate** is allowed only for a deterministic tool that reports an
  objective defect with a low false-positive rate.
- **Heuristic metrics start as signals.** They are never a global score:
  - no `coverage >= X%`;
  - complexity is judged per function and as a PR delta.
- Nothing duplicates an existing protection without a demonstrated benefit.

### Local measurement (2026-10-06, `e59bfe9`, Go 1.26.1, macOS arm64)

These measurements were run once as exploratory Research in a scratch
location. No tool was added to the repository.

| Signal | Command | Result |
|---|---|---|
| Coverage | `go test -count=1 -coverprofile ./...` (~47 s, no `-race`) | 73.7% of statements in-process. Per package: 45.1% (`cmd/lingo`) to 99.1% (`internal/project`); `install` 83.0%, `local` 73.8%, `compatibility` 77.3%, `codexruntime` 72.4%. Black-box tests execute a separately built binary, so their coverage is **not counted**. That is why `cmd/lingo` reads low. |
| gofmt | `gofmt -l .` | Clean |
| Module hygiene | `go mod tidy -diff` | Clean |
| goimports | `goimports -l .` | 38 files differ, all in import *grouping* (third-party imports are not in a separate group). This is style, not a defect. |
| staticcheck | `staticcheck` 2026.2.1 (v0.8.1) `./...` (~19 s cold) | 12 findings: 6 `U1000` unused functions (e.g. `install.syncDirectory`, `codexruntime.syncPath`, `local.clearAttempt`); 3 `ST1005` capitalized error strings; 1 `SA4006` unused value in a test; 1 `SA4000` and 1 `SA4004` that are **false positives or intentional idioms** (repeated stateful `Decode` calls; "return first issue" loop). No product correctness defect was found (inference). The locally installed v0.6.1 cannot load a Go 1.26 module, so the tool version is coupled to the Go toolchain. |
| Cyclomatic complexity | `gocyclo` | Average 6.43. Non-test functions: 114 over 15, 26 over 30. Top: `install.(Service).Apply` 97, `InstallReleasePOSIX` 73, `workflow.ValidState` 69, `install.InstallRelease` (Windows) 56, `install.(Service).previewIn` 55, `cli.RunInteractive` 55. Hotspots concentrate in the installer and upgrade path, which is the most release-critical code. |
| Dead code (reachability) | `deadcode ./cmd/...` | 175 functions unreachable from `cmd/lingo`. Most are deliberate: `executiongraph` (55) and `projectapp` (41) serve the S8 graph and acceptance tooling, which has no CLI surface (`s9-graph-runner.go` is `//go:build ignore`), and some are test-only helpers. **High false-positive rate for a gate.** |
| govulncheck | `govulncheck ./...` (~6 s) | Results are intentionally not recorded in this public document; any finding is handled privately under [SECURITY.md](../../SECURITY.md). |
| errcheck, cognitive complexity, duplication, mutation | — | **Not measured.** These tools are not installed, and this Research did not adopt third-party tools only to measure. Any estimate below is labelled. |

Release toolchain (fact):

- `go.mod` declares `go 1.26.0` with no `toolchain` directive.
- `setup-go` with `go-version-file` installs exactly that patch. The v0.5.0
  prepare run 37394812687 logged `go version go1.26.0 linux/amd64`.
- Published binaries therefore embed the 1.26.0 standard library.
- Nothing checks standard-library or toolchain advisories, and Dependabot
  (V17) does not cover the standard library.

### Assessment

**Stage key:**
- **PR-B**: merge-blocking.
- **PR-S**: PR signal: a report in the job summary that does not block.
- **Sched**: scheduled on `main`.
- **Prep-B**: release-blocking at prepare.

| Item | Current state | Failure class it detects | Value for Axiom | False-positive risk | Cost / PR latency | Nature | Stage | Recommendation |
|---|---|---|---|---|---|---|---|---|
| **Test coverage** | Not collected anywhere; black-box and E2E coverage invisible | F37: changed code not exercised | Medium. Behaviour is guarded by contract tests, not percentages. **Diff coverage on critical packages** (`install`, `compatibility`, `local`, `codexruntime`, `manifest`) shows untested new branches in review | Global %: high (misleading). Diff coverage: low–medium (platform-gated files, defensive branches) | Profile collection on the existing Linux test run: ~+10–20% of that job (estimate); no extra job | Heuristic | **PR-S** (per-package plus diff report); later include black-box coverage via `go build -cover` and `GOCOVERDIR` | **Adopt as a signal.** No global threshold. Never blocking. Uncovered changed lines in critical packages are something to justify in review. |
| **Cyclomatic complexity** | Not checked | F38: hotspot growth (indirectly, untested paths) | Medium. It locates where branches outrun tests: the installer and upgrade hotspots are 55–97 | High as an absolute threshold (114 functions would fail at 15) | Seconds | Heuristic | **PR-S**: report functions whose complexity *increases* past a local threshold (e.g. 15), or new functions above it; **Sched**: hotspot report | **Signal (delta-based) only.** Use it to target mutation testing and refactoring, not to gate. |
| **Cognitive complexity** | Not checked; not measured | F38 (readability) | Low **in addition to** cyclomatic: it is largely redundant as a signal | Medium | Seconds | Heuristic | — | **Do not adopt separately.** Report one complexity metric. Revisit only if the cyclomatic signal proves unhelpful in review. |
| **staticcheck** | Absent; `go vet` only (V3) | F33 bug patterns; F35 unused code | High. Objective defects at near-zero runtime cost, and the code base is mostly clean already (12 findings, 2 FP) | Low with a curated check set; per-line suppression for justified idioms | ~20 s cold, less when cached; one Linux job (+ `GOOS=windows`/`darwin` passes for platform files) | Deterministic for a pinned version | **PR-B** after a one-time baseline fix | **Adopt as blocking.** Pin the tool version to the Go toolchain (the coupling above), enable `SA*` plus `U1000`, and start `ST*` as signal. Tool adoption is a hypothesis to approve in its slice. |
| **errcheck** | Absent; not measured | F34: ignored errors on effectful calls | **High for Axiom specifically.** Correctness and durability depend on `Close`/`Sync`/`Rename`/`Remove` results in `local`, `install`, `compatibility` and `codexruntime` | Medium (estimate): `fmt.Fprint*` to stdout/stderr, deferred `Close` on read-only files. Needs an exclusion list | Seconds | Deterministic, but its *relevance* is heuristic | **PR-S** first; **PR-B** for critical packages once the baseline is clean | **Adopt, promoted in stages.** Blocking first where durability is a product contract. Not duplicated by `go vet` or staticcheck. |
| **Dead code** | Absent | F35 | Medium. Unused helpers in security- and durability-relevant code deserve review; e.g. unused `syncDirectory`/`syncPath` *may* indicate an intended durability call that is missing (**inference, to verify in the slice**) | `U1000`: low. Whole-program `deadcode`: high (graph/acceptance code with no CLI surface, platform files) | Included in staticcheck; `deadcode` ~4 s | `U1000` deterministic; reachability heuristic | `U1000` via staticcheck **PR-B**; `deadcode` **Sched** report | **Adopt `U1000` only as a gate.** Keep reachability as a scheduled report, triaged by a human. |
| **Duplication** | Absent; not measured | F38: divergent copies | Low. Known duplication is deliberate (POSIX/Windows platform pairs; workflow job-isolation blocks in `publish-release.yml`) | High | Seconds–minute | Heuristic | **Sched** report, optional | **Do not gate.** Optional scheduled report; adopt only if a divergence defect is observed. |
| **govulncheck** | Absent. Dependabot covers module advisories without reachability (V17). Nothing covers the standard library or toolchain | F36 | **High.** Every published binary embeds the standard library. Axiom's security boundary relies heavily on standard-library filesystem and archive APIs, and the release toolchain is pinned to a single patch version | Low: symbol-level reachability reports only called vulnerable code | ~6 s source mode; seconds per binary | Deterministic for a given database snapshot. The result changes when the external database changes (F31-like) | **PR-B** only when a PR changes `go.mod`/`go.sum`/toolchain (otherwise **PR-S**); **Sched** daily on `main`; **Prep-B** in binary mode on the **prepared binaries (P)** with an explicit human waiver path recorded in the envelope | **Adopt.** Pair it with a toolchain patch policy (a `toolchain` directive or explicit patch bumps) so fixes are reachable. Route scheduled results per SECURITY.md (see [Open human decisions](#open-human-decisions)). |
| **Formatting / import hygiene** | Not enforced in CI; clean today by convention | F32 | Medium. Zero-FP checks that keep reviews focused; `go mod tidy` protects module integrity | gofmt and tidy: none. goimports: 38 files of grouping churn with no defect value | Seconds | Deterministic | **PR-B** for `gofmt -l` and `go mod tidy -diff` | **Adopt `gofmt` and `tidy` as blocking.** **Do not adopt goimports grouping as a gate** (churn without a defect); revisit only as a one-time formatting change. |
| **Mutation testing** | Absent; not measured | F37: tests that execute code without detecting changes in it | **High for a small critical core**: compatibility classification/transition policy, upgrade policy, manifest parser, release preflight rules. Low elsewhere | Medium (equivalent mutants need triage) | Minutes to hours per package (estimate): far too slow for PRs | Heuristic (score) on top of deterministic execution | **Sched** (weekly) on selected packages; manual before changing a critical policy | **Adopt later, scheduled and scoped.** Surviving mutants become test Issues. Never a PR gate or a global score. Tool choice is an unadopted hypothesis. |

### What already has equivalent protection

- `go vet` (V3) overlaps a subset of staticcheck. Keep both: vet is the
  toolchain baseline, and staticcheck adds bug patterns and `U1000`.
- CodeQL (V16) is security-query oriented. It does not replace staticcheck,
  errcheck or govulncheck. The ruleset `code_quality` rule already blocks
  error-severity Code Quality findings, so new gates should not duplicate
  that rule's findings. Revisit once its feeding analysis is confirmed.
- Dependabot (V17) covers module advisories. govulncheck adds reachability
  and, crucially, the standard library and toolchain.
- Behavioural guards (V1, V8, V12) remain the primary quality evidence.
  Coverage and mutation only measure *test adequacy* around them.

---

## External research

Only questions raised by the gap map were researched, from primary sources
fetched on 2026-10-05.

| Topic | Finding | Source | Consequence for Axiom |
|---|---|---|---|
| Runner labels | `ubuntu-24.04-arm` is GA (arm64). `macos-15`, `macos-26` and `macos-latest` are arm64. `windows-11-arm` is arm64; no hosted Windows *client* amd64 label exists. `windows-2022` is retained. `macos-latest` moved to macOS 26 in mid-2026; no deprecation is announced for `macos-15`. | [GitHub-hosted runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners), [image migrations 2026-05-14](https://github.blog/changelog/2026-05-14-github-actions-upcoming-image-migrations/), [runner-images](https://github.com/actions/runner-images) | Linux arm64 native is available at no cost. The Windows client row cannot be native on hosted runners. Record the image version in Evidence; plan for a `macos-15` deprecation. |
| Billing | Standard hosted runners are free for public repos. Private per-minute rates: Linux x64 $0.006, Linux arm64 $0.005, Windows $0.010, macOS $0.062 (about 10× Linux). | [Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions), [runner pricing](https://docs.github.com/en/billing/reference/actions-runner-pricing) | Cost is a constraint only if the repository becomes private. Then macOS dominates. |
| Artifact handoff | v4+ artifacts are immutable (overwrite creates a new id). `upload-artifact` outputs `artifact-id` and `artifact-digest`. `download-artifact` supports `artifact-ids` and verifies the digest (`digest-mismatch: error` by default in current majors). Retention is 1–90 days for public repos. File modes are not preserved (zip). | [upload-artifact](https://github.com/actions/upload-artifact), [download-artifact](https://github.com/actions/download-artifact) | Pin acceptance and publish to the prepare `artifact-id` plus digest. Keep the tar.gz and `SHA256SUMS` inside: the artifact digest covers the zip wrapper, not the inner archives. The current pin is download-artifact v4.3.0; digest enforcement for that version was not verified. |
| Attestations | `actions/attest` (attest-build-provenance v4 wraps it) gives SLSA v1.0 Build L2 provenance bound to the subject digest, free for public repos. Verify with `gh attestation verify`. Immutable releases (GA 2025-10-28) lock tag and assets and generate a release attestation (`gh release verify`). | [Artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations), [Immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases) | Optional hardening: attest P at prepare and verify at acceptance and publish. Not required to close the #153 gap. |
| Gating across runs | Environments support required reviewers, wait timers and custom protection apps. There is no native `needs:` across workflow runs; `workflow_run` is limited and runs privileged. | [Environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments), [Events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows) | Putting acceptance jobs **in the prepare run** reuses the existing check `prepared run conclusion == success`. No new cross-run trust is needed. |
| Reproducible builds | `CGO_ENABLED=0 go build -trimpath` is reproducible for the same toolchain and sources. `tar`/`gzip` are not deterministic by default. | [go.dev/blog/rebuild](https://go.dev/blog/rebuild) | A rebuild cannot stand in for P. Validate P. |
| Reports, summaries | `go test -json` → JUnit via `gotestsum`. Step summaries: 1 MiB per step, 20 per job. | [Workflow commands](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands) | Keep the summary short; retain structured JSON as an artifact. |
| Go vulnerability and coverage tooling (revision 2) | `govulncheck` reports only vulnerabilities whose symbols are reachable. It covers the standard library for the toolchain in use, and it can scan compiled binaries (`-mode=binary`). Go 1.20+ builds coverage-instrumented binaries (`go build -cover`, `GOCOVERDIR`), so integration and black-box runs can contribute coverage. The `toolchain` directive pins the toolchain separately from the `go` language version. | [Go vulnerability management](https://go.dev/doc/security/vuln/), [Coverage for integration tests](https://go.dev/doc/build-cover), [Go toolchains](https://go.dev/doc/toolchain) | Binary-mode scanning of **P** binds vulnerability Evidence to the exact bytes being published. Black-box coverage removes the misleading low `cmd/lingo` figure. A toolchain patch policy is needed for stdlib fixes to reach releases. |
| Retries | No native step retry. Re-running jobs keeps `run_id` and increments `run_attempt`; the limit is 50 re-runs within 30 days. | [Re-run workflows](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/re-run-workflows-and-jobs) | Evidence must carry `run_attempt`. A rerun after failure must be justified by failure category. |

---

## Options considered

The options below differ in *where candidate acceptance runs and what it
binds*. The PR and main stages are the same in all of them; see the
recommendation.

| | A. Status quo + PR hardening | B. Acceptance inside the prepare run (**recommended**) | C. Separate acceptance workflow bound by the envelope | D. Publish an RC, accept the published RC, then publish stable | E. Self-hosted native lab |
|---|---|---|---|---|---|
| What it is | Keep V8 on C′. Add linux-arm64 and `install.sh` to PR. Release relies on `revision_ci` | Add native-row jobs to `release-artifacts.yml` (`needs: prepare`). They download P by artifact id/digest and run E2E plus the upgrade matrix on P. Run success is already required by `verify-prepared-release.sh` | New `candidate-acceptance.yml` dispatched with `prepared_run`. The envelope adds `acceptance_run` and the Evidence digest. Publish verifies both | Use the existing RC channel as the acceptance subject; T24-style per RC | Operator-owned runners for exact OS rows (Windows client, other distros, macOS versions) |
| Confidence (#153/#186 class) | Medium: regression on the same source, not the same bytes; F17 open | **High**: same bytes, real version, native rows | High | High for the RC bytes, **but stable bytes differ** (version embedded; different release commit) | High fidelity, but adds operational trust and maintenance |
| Feedback latency | PR +0 to 1 min (arm64 in parallel) | PR +0; prepare +3 to 6 min (parallel jobs, estimate) | PR +0; release +1 dispatch cycle | Days (manual RC cycle) | Variable |
| Cost (public repo) | Free | Free (private: about $0.5–1.5 per release, mostly macOS; estimate) | Free | Human hours per RC | Hardware and upkeep |
| Complexity | Low | **Low–medium**: one workflow, existing trust gate | Medium: new cross-run binding in envelope and publish | Low in CI, high in process | High |
| Reproducibility | Good | Good (Evidence pins P digest and inputs) | Good | Manual ledgers | Depends on the lab |
| Native fidelity | macOS arm64, Linux amd64 (+arm64) | macOS arm64, Linux amd64/arm64; Windows Server proxy | Same as B | Whatever the operator runs | Highest |
| Maintenance | Low | Low; shares `test-upgrade-journeys.sh` with PR | Medium | High | High |
| Fits authority model (ADR-0011/0012) | Yes | **Yes, unchanged**: the envelope already binds `prepared_run` | Requires envelope/publish change | Yes | Yes |
| Verdict | Necessary but insufficient | **Recommended** | Fallback if prepare-run duration or permissions become a problem | Keep as an *optional* risk-driven step, not the gate | Not justified now (no demonstrated failure that hosted rows miss) |

A single combined prepare → accept → publish workflow (one run with
`needs:` and an environment gate) was also considered. It moves publication
authority from the reviewed envelope digest to an environment click. That
weakens ADR-0011/0012 and is rejected.

---

## Recommended target pipeline

### Stages

```text
PR (merge-blocking, ~5 min budget)
  F01–F08, F23 (fake transport), F28, F30   deterministic, hermetic
  structure job (Linux, ~1 min, parallel):  gofmt -l, go mod tidy -diff, staticcheck (pinned, curated, + GOOS passes)  [blocking]
                                            govulncheck  [blocking only if go.mod/go.sum/toolchain changed]
                                            errcheck  [signal → blocking for critical packages]
                                            coverage per package + diff coverage, complexity delta  [signal only]
  F09–F14 on C′ (upgrade-journeys)          regression feedback; linux-amd64, linux-arm64, macos-arm64
  F15 generation-registry check             a new persisted/receipt generation requires a declared baseline

main (push)
  same required set (post-merge truth for revision_ci)
  scheduled (non-blocking): all-stable-sources sweep, real Provider sandbox, flake detector,
                            govulncheck daily, mutation testing (weekly, critical packages),
                            deadcode / complexity hotspot reports

release prepare  ── release-artifacts.yml, ONE run ──
  prepare:  preflight → build once → structural verify → F16 corpus freshness
            → govulncheck -mode=binary on every prepared binary (F36; waiver only by human, recorded in envelope)
            → upload P (artifact-id, digest)
  accept-<row> (needs: prepare; matrix linux-amd64 / linux-arm64 / macos-arm64 / windows-proxy):
            download P by id+digest → verify SHA256SUMS == prepare evidence
            → candidate smoke (version provenance on THIS row, real version)
            → fresh install via bundled installer AND public install.sh over local transport
            → upgrade-source matrix (declared generations) with non-empty state
            → usable probes → reinstall no-op → refusal smoke (downgrade)
            → upload acceptance-evidence-<row>.json
  evidence: aggregate → acceptance-evidence.json (digest) into the prepared set
  run conclusion == success  ⇒  release-blocking (already enforced)

authorization
  envelope adds: acceptance_evidence_sha256, rows, attempt  → human authorizes the digest (unchanged model)

release publish
  download P by prepared run (+ artifact id) → re-verify → envelope == authorized → environment → publish
  NO rebuild; NO re-test of behaviour

post-publish (operational detection, not prevention)
  published-smoke on linux-amd64 / macos-arm64 / windows-proxy:
    public install.sh/install.ps1 --version vX.Y.Z  and  channel stable
    → asset digest == envelope → version provenance → reinstall unchanged
  failure ⇒ incident + fix-forward release (immutable releases: no in-place repair)
```

### Layer responsibilities (test taxonomy)

| Layer | Owns | Does not own |
|---|---|---|
| Unit | Domain rules, parsers, policy matrices | Filesystem or process effects |
| Component / package integration | Stores on a real FS, crash and fault injection, security-FS, compatibility classification, corpus | Installer packaging, real versions |
| Black-box CLI | Exit codes, envelopes, selector and authority semantics of a built binary | Archive/installer integrity |
| Contract | Release-flow scripts against a fake `gh`; Provider adapters against fakes; Runtime discovery against stubs | Real service behaviour |
| Smoke | "Minimally consumable" for a subject: starts, reports the correct provenance, installs, reinstalls no-op | Breadth |
| Candidate E2E | Install → first-run → configure → Work Item → workflow on **P** | Real Runtimes/Providers |
| Upgrade / compatibility | Declared generation → candidate with non-empty state, transition, usability, idempotency | Every historical version |
| Native platform | Same E2E per supported row on its OS family/arch | Distro/OS-version permutations (supplemental) |
| Real Runtime / Provider acceptance | Live Codex/Claude/GitHub behaviour versus the contract | Ordinary merge gating |
| Structural / static (revision 2) | Objective code defects found without execution: format, module tidiness, vet, curated staticcheck (incl. unused code), unchecked errors in critical packages, reachable vulnerabilities | Behaviour; style preferences without a defect (import grouping, naming) |
| Quality signals (revision 2) | Coverage per package and on changed lines, complexity delta, hotspot, dead-code and duplication reports, used to *direct* review and testing | Gating; global scores or percentages |
| Test adequacy (revision 2) | Scheduled mutation testing of the critical decision core (compatibility/upgrade policy, manifest parser, release preflight rules) | PR latency; whole-repository scores |

### Blocking versus informational

| Gate | Stage | Blocking |
|---|---|---|
| Current 7 required contexts, plus `upgrade-journeys (linux-arm64)` and `install-bootstrap` contracts | PR/main | **Merge** |
| `accept-*` jobs on P, corpus freshness, acceptance Evidence present and bound | Prepare | **Release** (run success; envelope) |
| Windows Server proxy acceptance | Prepare | **Release**, labelled `proxy`; native client Evidence handled per the [Windows decision](#open-human-decisions) |
| Published smoke | Post-publish | Informational for publication (already immutable); **blocks closing the release as verified** and opens an incident |
| All-stable sweep, real Runtime/Provider acceptance, flake detector | Scheduled/manual | Informational. Blocking only when the release contract changes that integration (see below). |
| `gofmt -l`, `go mod tidy -diff`, curated `staticcheck` (incl. `U1000`) | PR/main | **Merge** |
| `govulncheck` (source mode) | PR | **Merge** only when the PR changes `go.mod`/`go.sum`/toolchain; otherwise signal (a new advisory is an external event, F31) |
| `govulncheck -mode=binary` on P | Prepare | **Release**; an explicit human waiver is recorded in the envelope |
| `errcheck` | PR | Signal, then **merge** for critical packages once the baseline is clean |
| Coverage / diff coverage, complexity delta | PR | Signal only (job summary). Never blocking; no global threshold |
| Mutation testing, `deadcode`, duplication, complexity hotspots, daily `govulncheck` | Scheduled | Informational; findings become Issues (vulnerabilities are routed per SECURITY.md) |

### Environment matrix

| Row | PR/main | Prepare (P) | Post-publish | Supplemental |
|---|---|---|---|---|
| macOS arm64 (`macos-15`, then the successor label) | V1, V5, V8 | accept | smoke | Operator native Evidence on new major macOS |
| Linux amd64 (`ubuntu-24.04`) | V1, V5, V8 | accept | smoke | Other distros: manual |
| **Linux arm64 (`ubuntu-24.04-arm`)** | **add** V1 + V8 | **accept** | smoke (optional) | — |
| Windows amd64 (`windows-2022` Server proxy) | V2, V6 | accept (proxy, incl. `install.ps1` upgrade from N) | smoke | Client amd64 native: decision required |

Every Evidence record names the runner image and version, OS version,
architecture and filesystem type. These are facts, not eligibility
(Spec 006).

### Compatibility (upgrade-source) policy

Proposed for a Specification/ADR; not decided here.

1. **N (latest published stable)** is always a supported source. This is the
   issue's `N-1 → N`: the candidate supersedes N.
2. **One baseline per persisted generation ever shipped in a stable
   release.** A generation is a distinct combination of:
   - state/portable format (the v1 corpus snapshots);
   - installation/skill-set receipt format (pre-#186 five-skill receipts,
     represented by v0.1.1);
   - a recognized historical format with a declared transition (POC,
     represented by `v0.1.0-poc.1`).

   Today's baselines are therefore N, v0.1.1, POC and the frozen v0.4.0
   corpus. This is exactly what V8 already exercises.
3. **Adding a generation requires a declared baseline in the same PR.**
   This is the F15 check: a new receipt format, kind or encoding without a
   registry entry fails PR. **Retiring a baseline requires an explicit
   compatibility-window decision** (FR-026), never silent pruning.
4. **Every stable release as a source is not a gate.** A scheduled weekly
   *all-stable-sources sweep* (10 stable tags today at about 20–30 s
   each, estimate) is non-blocking. It detects generations the model missed; a
   finding becomes a new declared baseline.
5. **Prereleases, POC and RC states are not supported sources unless
   declared** (as POC is, by ADR-0017). Downgrades stay refused and are
   smoke-checked.
6. **Fixtures:** keep frozen state corpora (append-only, already enforced)
   *and* real published assets. Assets are immutable releases, so their
   availability is a stable dependency. **Enforce corpus freshness at
   prepare (F16)**: writer output for the candidate must be byte-covered by
   a frozen snapshot, or prepare fails with the freeze command.

### Smoke versus E2E

| Suite | Subject | Purpose | Size / time | Where |
|---|---|---|---|---|
| Main smoke | B (`dogfood-poc.sh`, exists) | The CLI lifecycle still composes | Seconds | PR/main |
| Candidate smoke | P, per row | Correct bytes for *this* row start and report the exact version, revision and `clean`; fresh install succeeds | <30 s per row | First step of `accept-*` (fail fast) |
| Full candidate E2E | P, per row | Fresh install (bundled installer + public `install.sh` over a local transport), first-run, configure, Work Item (fake `gh`), workflow, upgrade matrix, reinstall no-op, downgrade refusal | Minutes per row (estimate 2–4) | `accept-*` |
| Published smoke | R through the public path | Public installer, channel selection, CDN/redirect, digest equals envelope, reinstall no-op | <1 min per row | Post-publish |

### Immutable artifact flow

```text
prepare: build once → P (artifact-id, artifact-digest, SHA256SUMS, per-archive sha)
accept-*: consume P by artifact-id; refuse unless SHA256SUMS digest == prepare evidence
envelope: binds prepared_run, artifact sha set, acceptance_evidence_sha256 (+ run_attempt)
publish: consume P by run (and artifact-id); refuse unless envelope == authorized digest
read-back: published digests == envelope; GitHub release attestation (immutable releases)
optional: actions/attest on P at prepare; gh attestation verify in accept and publish
```

Acceptance writes **only** Evidence files. It never mutates P. A new upload
gets a new artifact id; a separate Evidence artifact name prevents any
rewrite of P.

### Evidence

Gate Evidence is a closed, bounded JSON document (proposed
`axiom-gate-evidence/v1`). A step summary shows a short excerpt, and the
document is retained with the prepared set (≥ the publish window, today 30
days). The release record links it. **Raw logs are never the only Evidence.**

| Field | Content |
|---|---|
| `gate`, `suite`, `row` | e.g. `candidate-acceptance`, `upgrade-matrix`, `linux-arm64` |
| `subject` | `kind` (`prepared`/`published`/`rebuilt`), `tag`, `version`, `revision`, `sha256sums_sha256`, `archive_sha256`, `artifact_id`, `artifact_digest` |
| `run` | Workflow, run id, `run_attempt`, workflow ref, job |
| `environment` | Runner label and image version, OS name and version, architecture, filesystem type, Go version, shell |
| `inputs` | Upgrade sources (tag plus asset sha), fixture/corpus digests, exact command line |
| `journeys[]` | Id, source generation, steps with `result`, installer category, classification before/after, preservation manifest digest, timings |
| `result` | `pass` / `fail`, `failure_category` (see below), counts |
| `started_at`, `finished_at` | UTC |

### Flakiness and failure policy

- **Failure categories** are recorded in Evidence:
  - `product` — deterministic defect;
  - `test_defect`;
  - `infrastructure` — runner, network or GitHub API;
  - `external_dependency` — a published asset or a real service;
  - `timeout`.
- **Retries:**
  - Allowed only *inside* transport steps (`curl --retry`) and for whole-job
    re-runs whose failure is categorized `infrastructure` or
    `external_dependency`. The Evidence keeps every attempt.
  - **Forbidden** for `product` and `test_defect`, for any assertion step,
    and for release gates once the failure output shows an assertion
    failure.
  - A pass on rerun after an assertion failure is recorded as `flaky`. It
    is not a pass.
- **Timeouts:**
  - Every job and step is bounded. Current job bounds are 20–30 min.
  - Proposed suite budgets: PR suites at ≤ 2× median; `accept-*` at
    15 min per row.
  - A timeout is `timeout`, never retried silently.
- **Quarantine:**
  - A test with an observed nondeterministic result moves to a quarantine
    list in the same PR that records the observation.
  - The list is versioned, and each entry has an owner capability (from the
    registry set) and an expiry.
  - A quarantined test runs and reports but does not block.
  - Quarantine never applies to a test that guards a release-critical
    failure class (F09–F19); those are fixed or the gate is held.
- **Ownership:** each blocking suite maps to a registry capability, for
  example `compatibility`, `installation`, `release`, `cli-e2e` or `ci`.
  Named people remain a human decision.
- **Promotion to blocking:**
  - deterministic by design: no real network or clock dependence beyond
    bounded transport retry;
  - it maps to a failure class;
  - a failing control was demonstrated, as #188's negative control against
    v0.4.1 did;
  - ≥ 10 consecutive clean runs at the target stage;
  - documented runtime.
- **Retirement:** a gate whose flaky rate exceeds about 2% over 50 runs, or
  that has not detected or guarded a class for which a cheaper layer is
  equivalent, is demoted to informational by a reviewed change that names
  the replacement protection.

### Runtime and Provider integration strategy

These distinctions are kept throughout:

- Runtime ≠ Model;
- Provider ≠ Integration;
- Integration ≠ Transport.

| Level | Subject | Where | Blocking |
|---|---|---|---|
| Deterministic fake/contract | Runtime discovery and skill install (stubs, isolated `HOME`); GitHub adapter (fake `gh`, simulated status/rate-limit); graph invocation fakes | PR | Merge |
| Recorded-contract drift | Version and help/feature probes of real `codex`, `claude` and `gh` executables, with no credentials or model calls; compares the surfaces Axiom depends on | Scheduled (weekly) | Informational; a drift opens an Issue |
| Sandbox Provider acceptance | GitHub: a dedicated sandbox repository with a fine-grained token scoped to it. Covers Work Item create, projection, label and comment, and replay idempotency | Scheduled, plus manual before a release that changes the GitHub Integration | Blocking **only** for releases whose delivered Issues change that Integration |
| Real Runtime acceptance | Codex/Claude login with a real model call (T24-style graph run) | Manual (operator credentials); optionally scheduled with a budgeted key | Blocking only when the release contract changes Runtime bootstrap or execution; otherwise supplemental Evidence |

Supporting rules:

- **Secrets:** environment-scoped secrets, used only by jobs that are not
  triggered by PRs and never on `pull_request` from forks.
- **Quotas, rate limits and cost:** bounded attempts, recorded in Evidence
  (`usage: unavailable` is acceptable, as in T24).
- **External outage:** categorized `external_dependency`. A release proceeds
  with recorded human acceptance of the gap.

### Performance and cost

| Stage | Today | Target (estimate) | Notes |
|---|---|---|---|
| PR wall-clock | ~4.2 min | ~4.5–5 min | linux-arm64 jobs run in parallel; `install-bootstrap` contracts take seconds |
| PR job-minutes | ~15 | ~18–20 | Free (public) |
| Release prepare | ~1 min | ~5–8 min | Four parallel `accept-*` jobs (estimate: 2–4 min E2E plus downloads) |
| Publish | ~1 min + human | Unchanged | No re-test |
| Post-publish | Manual read-back | +2–3 min | Automated smoke |
| Scheduled | None | Weekly sweep plus drift probes, ~10–15 job-min | Free (public) |
| If private | n/a | macOS is about 10× Linux. Per release about 4 macOS min (≈ $0.25) plus Linux/Windows (≈ $0.10). PR macOS is ≈ $0.25 per run | Becomes a concern only at high PR volume |
| PR structure job (revision 2) | None | ~1 min in parallel (staticcheck ~20 s cold, govulncheck ~6 s, gofmt/tidy seconds); wall-clock +0–0.5 min | Coverage profile adds ~10–20% to the Linux test job (estimate); no extra job |
| Prepare `govulncheck -mode=binary` | None | Seconds per binary | Runs inside the existing prepare job |
| Scheduled structural suites | None | Daily govulncheck (seconds); weekly mutation on 4–5 packages (tens of minutes to hours, estimate); reports (seconds) | Free (public). Mutation is the only material cost |
| Maintenance | — | One shared journey script for PR and prepare (subject is a parameter); a generation registry; an Evidence schema; pinned analyzer versions bumped with the Go toolchain; errcheck exclusions; triage of signals and surviving mutants | The largest burdens are keeping journey 3 (POC build) working as Go evolves and keeping analyzer versions aligned with the toolchain |

---

## Gap analysis

| ID | Current | Target | Missing capability | Classes |
|---|---|---|---|---|
| G1 | Prepared bytes are verified only structurally; behaviour is tested on C′ | Native-row acceptance consumes P in the prepare run | `accept-*` jobs; `test-upgrade-journeys.sh` accepting a prepared set and the real version; download by artifact id/digest | F09–F14, F17, F19 |
| G2 | Upgrade sources are implicit, defined only by V8 arguments | Declared generation baselines; PR check for undeclared generations | Spec/ADR + machine-readable generation registry + check | F15 |
| G3 | Corpus freeze is voluntary | Prepare fails when writer output is not frozen | Freshness check at prepare | F16 |
| G4 | Linux arm64 is never executed | arm64 in PR (V1/V8) and accept | `ubuntu-24.04-arm` rows | F20 |
| G5 | `install.sh` bootstrap is never in CI | Fake-transport contract at PR; local-transport install of P at accept | Wire `test-install-bootstrap.sh` (and posix facade) into CI; a local transport mode for the acceptance harness | F23 |
| G6 | No published-path automation | Post-publish smoke | Workflow after publish, or `release.sh verify --smoke` | F24 |
| G7 | Windows: no upgrade journey; Server proxy only | Windows proxy acceptance with an `install.ps1` upgrade from N; explicit client-row stance | PowerShell journey; decision | F11, F20, F22 |
| G8 | Evidence is stdout or a step summary | `axiom-gate-evidence/v1` JSON retained and bound in the envelope | Schema, emitters, envelope field | All release-critical |
| G9 | No failure categories, retry rules or quarantine | Policy plus Evidence fields | Policy text (quality policy), quarantine list | Cross-cutting |
| G10 | No real Runtime/Provider checks outside manual T24 | Scheduled drift probes plus sandbox Provider acceptance | Sandbox repo, scoped secret, workflow | F25, F26 |
| G11 | #126 closed without automation | Superseded by G4 + G7 + supplemental native Evidence | Reconcile #126 with a reference to this research | F20–F22 |
| G12 | Formatting and module tidiness are clean only by convention | Blocking `gofmt -l` and `go mod tidy -diff` | Structure job | F32 |
| G13 | No standard-library/toolchain vulnerability check; release toolchain is the exact `go` directive patch | Conditional PR gate, daily scheduled scan, binary-mode release gate on P; toolchain patch policy | `govulncheck` integration, `toolchain` policy, envelope waiver field | F36 |
| G14 | Static analysis limited to `go vet` (+ security-oriented CodeQL) | Curated, pinned staticcheck as a blocking gate | Baseline cleanup (12 findings), suppression policy, pinned version | F33, F35 |
| G15 | Ignored errors on effectful calls are undetected | errcheck signal → blocking in critical packages | Exclusion list, staged promotion | F34 |
| G16 | No coverage collected; black-box coverage invisible | Per-package and diff-coverage signal, incl. `go build -cover` binaries | Coverage report in the job summary | F37 |
| G17 | No maintainability signals | Complexity delta signal; scheduled hotspot/dead-code/duplication reports | Report generation | F38 |
| G18 | Test adequacy of the critical decision core is unmeasured | Scheduled, scoped mutation testing | Tool adoption decision, package scope, triage flow | F37 |

Smaller, non-blocking notes:

- Windows has no `-race`.
- `check_darwin_nocgo_test.go` probably never runs (inference: cgo is on by
  default on macOS runners).
- Cross-FS archive tests run only on Linux.
- Upgrade-journey failures print only 20 lines, and journey 1 continues
  after a seeding failure.
- The `macos-15` runner is labelled `macos-27-arm64` (legacy naming per
  ADR-0015, but Evidence should record the real OS version).

---

## Implementation decomposition

These are bounded slices in dependency order. **None is authorized by this
research.** Each needs its own Issue and the normal SDD flow.

| # | Slice | Depends on | Outcome | Size |
|---|---|---|---|---|
| 1 | **Upgrade-source and release-acceptance policy** (Specification amendment + ADR candidate): generation-based supported sources, the candidate-acceptance gate on prepared bytes, the Windows client stance | — | Contract that the gates encode (closes F15 at the policy level) | S (doc) |
| 2 | **Gate Evidence schema** `axiom-gate-evidence/v1` plus a JSON emitter in `test-upgrade-journeys.sh` (failure categories, environment facts); summary stays short | — | Structured Evidence usable by every later slice | S |
| 3 | **Journey harness takes a prepared set**: `--candidate` accepts a directory plus the expected `SHA256SUMS` digest and the real version; journey 2 seeds state; fail-fast on seeding failure | 2 | One harness for both C′ (PR) and P (prepare) | S–M |
| 4 | **Candidate acceptance in `release-artifacts.yml`**: `accept-{linux-amd64, macos-arm64}` jobs consuming P by artifact id/digest; acceptance Evidence uploaded; envelope binds `acceptance_evidence_sha256`; `test-release-pipeline.sh`/`test-release-flow.sh` contracts updated | 1, 3 | **Closes the #153/#186 release-boundary gap (G1)** | M |
| 5 | **Linux arm64 rows**: `ubuntu-24.04-arm` in `verify` and `upgrade-journeys` (PR) and `accept` (prepare) | 4 (for accept); PR part independent | G4 | S |
| 6 | **Generation registry and PR check**: a machine-readable baseline list consumed by `ci.yml` and the accept jobs; fails when a new receipt or state generation lacks a baseline | 1, 3 | G2 | M |
| 7 | **Corpus freshness check at prepare** | 1 | G3 | S |
| 8 | **Canonical bootstrap in CI**: `test-install-bootstrap.sh` and `test-install-posix-facade.sh` as PR checks; local-transport public `install.sh` install of P in `accept-*` | 4 | G5 | S–M |
| 9 | **Windows proxy acceptance**: PowerShell upgrade journey (N → P via `install.ps1`) on `windows-2022` in PR and prepare | 1, 3 | G7 (proxy part) | M |
| 10 | **Post-publish smoke** (public installer, `--version` and channel stable, digest equals envelope, reinstall no-op) and an incident path | 2 | G6 | S |
| 11 | **Flakiness and failure policy** in `.agents/policies/quality.md`, with a quarantine list and owner capabilities | 2 | G9 | S |
| 12 | **Scheduled non-blocking suites**: all-stable-sources sweep, Runtime/`gh` drift probes | 3, 6 | Model-error detection | S |
| 13 | **Sandbox Provider acceptance** (GitHub sandbox repo, scoped secret) | 1, 11 | G10 (Provider) | M |
| 14 | Optional: **artifact attestation** of P at prepare, verified in accept and publish | 4 | Provenance hardening | S |
| 15 | **Formatting and module hygiene gate**: `gofmt -l` and `go mod tidy -diff` in a PR `structure` job; required context | — | G12 | S |
| 16 | **Toolchain patch policy and govulncheck**: `toolchain`/patch-bump policy; PR gate when `go.mod`/`go.sum`/toolchain change (signal otherwise); daily scheduled scan on `main`; `-mode=binary` on P at prepare with a human waiver recorded in the envelope | 2; [decision 4](#open-human-decisions) | G13 | M |
| 17 | **Curated staticcheck as a blocking gate**: pinned version aligned with the toolchain; `SA*` + `U1000` blocking, `ST*` signal; one-time baseline fix of the 12 findings, with justified suppressions for the 2 false positives; `GOOS` passes for platform files. Includes checking whether the unused sync helpers indicate a missing durability call (a separate product Issue if so) | 15 (shares the job) | G14 | S |
| 18 | **errcheck staged promotion**: signal with an exclusion list, then blocking for `local`, `install`, `compatibility` and `codexruntime` | 11, 17 | G15 | S–M |
| 19 | **Coverage signal**: per-package and diff coverage on critical packages in the job summary; later black-box/E2E coverage via `go build -cover` and `GOCOVERDIR`; no threshold | 2 | G16 | M |
| 20 | **Complexity delta signal** (one metric) and a scheduled hotspot report | 19 (shared report) | G17 | S |
| 21 | **Scheduled mutation testing** of the critical decision core; surviving mutants become test Issues; the tool choice is approved in this slice | 11, 19 | G18 | M |
| 22 | Optional: scheduled **dead-code reachability and duplication** reports | 17 | G17 (supplemental) | S |

Sequencing note: slices 15 and 17 have no behavioural dependency. They are
the cheapest structural improvements and can proceed in parallel with
slices 1–3.

Reconciliation is also recommended: comment on #126 that its scope is
superseded by slices 5, 9 and the Windows decision. Do not reopen it
automatically.

## Risks

- **Hosted-runner image drift.** `macos-15` will be deprecated eventually,
  and Windows image routing has had reported anomalies (inference from an
  unverified runner-images issue). Mitigation: record the image version in
  Evidence; treat a label change as a reviewed workflow change.
- **External dependency in required checks.** Published-asset downloads and
  the POC tag build can fail for reasons unrelated to the change.
  Mitigation: categorize as `external_dependency`; immutable releases make
  the assets stable.
- **Generation model incompleteness.** A defect from an unmodelled
  generation still escapes the gate. The weekly sweep detects it; it does
  not prevent it.
- **Prepare-run duration and retention.** Acceptance extends prepare by
  minutes. The 30-day retention bounds the authorization window, as today.
- **Fake fidelity.** Runtime stubs and fake `gh` cannot detect real
  behaviour drift; this is mitigated only by the scheduled and manual
  levels.
- **Windows client.** No hosted native client amd64 exists. Without a
  decision, Windows releases ship on proxy Evidence.
- **Analyzer/toolchain coupling.** An analyzer built for an older Go cannot
  load this module (observed with staticcheck v0.6.1 against Go 1.26).
  Mitigation: pin analyzer versions and bump them with the toolchain.
- **Signal fatigue.** Coverage, complexity and mutation reports lose value
  if they are noisy. Mitigation: report deltas only, on critical packages,
  in a short summary.
- **Vulnerability results in public logs.** Public CI output of
  vulnerability scans can disclose reachability for released binaries
  (see decision 4).
- **Process load.** More Evidence and gates add review surface for one
  maintainer. Keep the Evidence bounded and machine-checked rather than
  human-read.

## Open human decisions

Only blocking decisions are listed.

1. **Supported upgrade sources** (blocks slices 1, 4 and 6). Adopt the
   generation-based policy: N plus one baseline per shipped generation plus
   declared historical formats, retired only by an explicit window
   decision. Alternatives:
   - every stable release, which is a growing matrix;
   - N only, which would have missed #153 and #186.

   Recommendation: generation-based. It is hard to change later because it
   becomes a public support promise.
2. **Windows client amd64 release Evidence** (blocks slice 9's gate
   semantics). Options:
   - (a) the Server proxy is release-blocking, and native client Evidence is
     supplemental and manual per minor release;
   - (b) a self-hosted Windows client runner;
   - (c) human-attested native client Evidence required per stable release.

   Recommendation: (a) now, revisited if a client-only defect appears.
3. **Whether real Runtime/Provider acceptance can block a release**
   (blocks slice 13). Recommendation: blocking only for releases whose
   delivered Issues change that Integration or Runtime contract; otherwise
   supplemental.
4. **How vulnerability-scan results are surfaced in this public repository**
   (blocks slice 16). CI logs and job summaries are public. SECURITY.md
   forbids publishing unpatched vulnerabilities. Options:
   - (a) PR and prepare gates print advisory IDs publicly. The advisories
     are already public, and the gates block *before* a vulnerable change
     merges or a release publishes. Scheduled scans of `main` emit only
     pass/fail publicly and route details privately (for example a private
     security advisory draft).
   - (b) Every scan emits only pass/fail publicly.
   - (c) Every scan is fully public.

   Recommendation: (a). It is hard to change later only in the sense that
   published logs cannot be recalled.

Third-party structural tools (staticcheck, errcheck, a mutation tool) stay
hypotheses until each slice approves its adoption (AGENTS.md). Approving
them is a normal slice review, not a blocking decision for this research.

Recommended next artifact: slice 1 as a Specification amendment plus an ADR
candidate (*Release candidate acceptance on prepared bytes and
generation-based upgrade sources*). This research does not create that ADR.
