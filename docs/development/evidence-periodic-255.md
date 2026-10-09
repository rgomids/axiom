# Evidence — periodic monitoring (#255)

Contract: [approved D1–D5 context](https://github.com/rgomids/axiom/issues/255#issuecomment-6089267928).
Base: current `origin/main`; the repository has no `master` branch.
Implementation: `feat/255-periodic-monitoring` in an isolated worktree.

## Local validation (2026-10-09)

- `python scripts/test-periodic-monitoring.py`: 36 tests executed: 35 passed, one POSIX-only
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

Pending a confirmed sandbox repository. No production monitoring incident has
been created, no release/branch settings have changed, and no operational
Actions run is claimed. Issue #255 remains incomplete until real clean and
controlled failure/recovery runs and lifecycle evidence are recorded here.
