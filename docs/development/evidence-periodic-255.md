# Evidence — periodic monitoring (#255)

Contract: [approved D1–D5 context](https://github.com/rgomids/axiom/issues/255#issuecomment-6089267928).
Base: current `origin/main`; the repository has no `master` branch.
Implementation: `feat/255-periodic-monitoring` in an isolated worktree.

## Local validation (2026-10-09)

- `python scripts/test-periodic-monitoring.py`: 40 tests executed: 39 passed, one POSIX-only
  process-group test skipped on Windows (to be executed on Linux).
- `python scripts/check-automation-registry.py .`: PASS, 81 governed surfaces.
- `python scripts/check-adr-governance.py .`: PASS, 21 ADRs.
- Existing Go suites: first attempts found incomplete temporary Go installs;
  an official Go 1.26.0 archive was downloaded with its published SHA-256
  verified into isolated temporary storage. Tests execute with that toolchain.
- With a task-local test temporary directory, existing `githubissues`,
  `executiongraph`, `graphapplication`, `compatibility` and `runtimeprofile`
  suites PASS. `runtimeadapter` still fails its existing auth-preflight tests
  with `configuration_unreadable`, followed by a panic after the refusal.
  The initial default TEMP also failed filesystem ancestor-trust checks.
  No product code or host ACL was changed to bypass those checks. Full suites
  must be verified on the Linux scheduled runner.
- `actionlint 1.7.12 -shellcheck= -pyflakes= .github/workflows/periodic-monitoring.yml`:
  PASS. The official binary archive was checksum-verified in temporary storage;
  ShellCheck/Pyflakes were not available on this Windows host.
- `bash scripts/check-sensitive-files.sh .`: PASS (worktree).
- `./scripts/validate-repository.sh .` on this Windows checkout stops at existing
  Claude skill adapters checked out as plain files (`core.symlinks=false`). The
  host lacks symbolic-link privileges. Full validation remains required on a
  native-symlink checkout; partial checks do not substitute for it.

## Operational validation

Normal PR CI on the initial implementation:
[run 37994525582](https://github.com/rgomids/axiom/actions/runs/37994525582).
`go-quality`, `verify (linux)`, `verify (macos)` and both upgrade-journey checks
passed; Linux/macOS include native-symlink aggregate repository validation.
The release-contract static trigger allowlist rejected the new `schedule`
event. A narrowly named `periodic-monitoring.yml` exception now permits only
schedule/dispatch, preserving the delivery closure and existing trigger guards.
The corrected head's [run 37995147313](https://github.com/rgomids/axiom/actions/runs/37995147313)
passed all seven CI jobs (including release-contract and Windows verify), plus
delivery metadata. These normal PR runs are not D5 periodic-workflow/lifecycle
proof. GitHub's independent AI scanning job hit a monthly model quota; no code
finding or successful AI review is claimed.

Human authority confirmed `joaby-oliveira/axiom` as the temporary sandbox.
Its `main` and repository protections/releases/credentials were untouched.
The fork had no registered workflows; a branch-only echo bootstrap registered
the new workflow path ([37996172734](https://github.com/joaby-oliveira/axiom/actions/runs/37996172734)).
The exact candidate definition was then restored; no filename alias was needed.

- Controlled clean weekly [37996203214](https://github.com/joaby-oliveira/axiom/actions/runs/37996203214):
  PASS, both jobs succeeded, all 36 then-current offline tests passed on Linux,
  versioned safe artifact uploaded/downloaded, no unnecessary issue.
- Live weekly [37996245268](https://github.com/joaby-oliveira/axiom/actions/runs/37996245268):
  correctly failed and uploaded safe evidence. Operational testing exposed
  gh's colon-free USAGE heading, Claude's inert npm wrapper when postinstall
  is disabled, and normal adapter JSON exceeding the initial capture cap.
  Fixes now recognize the semantic heading, use the pinned optional native ELF
  directly without install scripts, and bound each pipe to 1 MiB.
  Sandbox evidence now identifies its actual dispatched branch.
- Reporting in that live failure returned HTTP 410: the fork's Issues feature
  was disabled. No incident was created. The human explicitly authorized its
  temporary enablement and restoration, which is now underway.
- Corrected live weekly [37996767428](https://github.com/joaby-oliveira/axiom/actions/runs/37996767428):
  PASS in both jobs, all 39 then-current offline tests passed on Linux, all CLI
  probes passed, and the suites recorded 42 adapter, 57 failure and 27 compatibility
  test outcomes. No incident was created.
- Controlled drift [37996928972](https://github.com/joaby-oliveira/axiom/actions/runs/37996928972)
  created sandbox issue #1; repeated drift
  [37997007645](https://github.com/joaby-oliveira/axiom/actions/runs/37997007645)
  updated that same issue to two occurrences. Both analyses failed accurately
  while reporting succeeded. The list endpoint lagged creation, omitting its
  first audit comment. Reporting now includes validated write responses when
  emitting audit comments; the added eventual-consistency regression test passes.

Issue #255 remains incomplete until the corrected live layers and controlled
create/update/recovery/recurrence matrix have operational evidence. No production
monitoring incident has been created. Earlier failed runs remain failed Evidence.
