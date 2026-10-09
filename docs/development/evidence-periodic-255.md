# Evidence — periodic monitoring (#255)

Contract: [approved D1–D5 context](https://github.com/rgomids/axiom/issues/255#issuecomment-6089267928).
Base: current `origin/main`; the repository has no `master` branch.
Implementation: `feat/255-periodic-monitoring` in an isolated worktree.

## Local validation (2026-10-09)

- `python scripts/test-periodic-monitoring.py`: 40 tests executed: 39 passed, one POSIX-only
  process-group test skipped on Windows; all 40 pass on the final Linux Actions runner.
- `python scripts/check-automation-registry.py .`: PASS, 81 governed surfaces.
- `python scripts/check-adr-governance.py .`: PASS, 21 ADRs.
- Go 1.26.0 `go mod tidy -diff` in an isolated, network-disabled module: clean
  PASS; an intentionally unused local dependency FAIL; `go.mod` SHA-256 unchanged.
  Reproduce with a minimal Go module/package, then add an unused `require` with
  a local `replace` and rerun the same read-only command. No repository dependency changed.
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

## Completed acceptance matrix

The actual artifacts were downloaded with an exact two-member archive allowlist,
size/link checks, then validated by the production semantic validator. The real
reporter also validated each manifest against its checked-out subject SHA. Every
run below passed offline contracts and its incident job; controlled failures
correctly retain a failing workflow conclusion. Fixtures remain explicitly
identified and do not substitute for live checks.

| Scenario | Actions run | Observed incident state |
|---|---|---|
| New drift | [37996928972](https://github.com/joaby-oliveira/axiom/actions/runs/37996928972) | #1 open; occurrences 1 |
| Repeated drift | [37997007645](https://github.com/joaby-oliveira/axiom/actions/runs/37997007645) | Same #1; occurrences 2; no duplicate |
| First recovery | [37997084374](https://github.com/joaby-oliveira/axiom/actions/runs/37997084374) | #1 stays open; 1/2 confirmations |
| Confirmed recovery | [37997140828](https://github.com/joaby-oliveira/axiom/actions/runs/37997140828) | #1 closed; 2/2 |
| Recurrence | [37997216442](https://github.com/joaby-oliveira/axiom/actions/runs/37997216442) | Same #1 reopened; occurrences 3 |
| Recovery after recurrence | [37997296000](https://github.com/joaby-oliveira/axiom/actions/runs/37997296000), [37997371862](https://github.com/joaby-oliveira/axiom/actions/runs/37997371862) | #1 open at 1/2, then closed at 2/2 |
| Infrastructure | [37997446370](https://github.com/joaby-oliveira/axiom/actions/runs/37997446370) | Separate #2, explicitly INCONCLUSIVE/network; creation audit present |
| Infrastructure recovery | [37997520852](https://github.com/joaby-oliveira/axiom/actions/runs/37997520852), [37997594663](https://github.com/joaby-oliveira/axiom/actions/runs/37997594663) | #2 open at 1/2, then closed at 2/2 |
| Controlled security finding | [37997668213](https://github.com/joaby-oliveira/axiom/actions/runs/37997668213) | FAIL/vulnerability; no security issue or advisory content |
| Controlled security clean | [37997747405](https://github.com/joaby-oliveira/axiom/actions/runs/37997747405) | PASS; no new issue |
| Final live daily | [37997812275](https://github.com/joaby-oliveira/axiom/actions/runs/37997812275) | Tidy/integrity PASS; security FAIL/vulnerability; no public issue |
| Final live weekly | [37997827935](https://github.com/joaby-oliveira/axiom/actions/runs/37997827935) | PASS, all CLI surfaces and 42/57/27 suite test outcomes |

Incident histories: [sandbox #1](https://github.com/joaby-oliveira/axiom/issues/1)
and [sandbox #2](https://github.com/joaby-oliveira/axiom/issues/2). Issues may be
unavailable while the fork's original disabled-Issues setting is restored;
the table and Actions evidence preserve the validation record.

Final live runs observed `ee9f86563e3aba1eab755ddb81a7cc9ab727fd92`, whose
workflow/scripts/tests/baseline are identical to implementation `933f07f`.
Both dispatches were requested together: daily reporting completed at
22:12:00 UTC; weekly analysis began at 22:12:03 UTC. This confirms cross-layer
serialization including reporting. Repeated findings produced one tracked drift
issue; offline race/uncertain-write fixtures additionally test retry discovery.
The final runners passed all 40 offline tests, including the POSIX child-pipe case.
Normal candidate CI [37996889323](https://github.com/rgomids/axiom/actions/runs/37996889323)
passed all seven jobs. The live ruleset read-back retained the existing eight
required contexts; no periodic context, workflow dependency or release gate was added.

### Sanitized artifact example

Excerpt of the actual final weekly manifest (not a complete replay manifest):

```json
{
  "schema_version": 1,
  "run_id": "37997827935",
  "branch": "ci/255-periodic-sandbox",
  "subject_sha": "ee9f86563e3aba1eab755ddb81a7cc9ab727fd92",
  "layer": "weekly",
  "scenario": "live",
  "status": "PASS",
  "checks": [{
    "check": "cli", "component": "codex", "contract": "exec",
    "tool_version": "0.162.1", "status": "PASS", "category": "none",
    "baseline": {"--model": true, "--sandbox": true, "--json": true},
    "observed": {"--model": true, "--sandbox": true, "--json": true}
  }]
}
```

Full artifacts include fingerprints, durations, reproduction commands and safe
per-test records. Only `manifest.json`/`summary.md` are uploaded, retained 14 days.
No raw advisory, installation, help or Go test output was imported into this ledger.

### Cleanup, review and remaining authority

Both sandbox incidents were automatically closed and the open-issue count was
verified as zero before cleanup. The temporary remote branch was deleted, the
created workflow disabled (`disabled_manually`), and the two labels created only
for this validation removed. `has_issues` was restored to `false` and read back.
The fork's default branch, protections, releases, credentials and Actions
permission configuration were not modified. No production incident was created.

Engineering/security review found no remaining blocking implementation finding:
scope, least privilege, ownership, confidential output boundaries, timeout and
recovery behavior were checked against D1–D5 and existing repository decisions.
The operational acceptance matrix is complete; original failed runs remain failed.
Human review/merge and the normal delivery/release closure remain separate authority.

**Operational finding:** the live daily scan detected a vulnerability. Detailed
triage must be rerun privately under `SECURITY.md`; public evidence deliberately
contains only FAIL/category. No automatic upgrade or remediation was attempted.
This is a correctly reported monitoring result, not an assertion that the source
revision is vulnerability-free. GitHub's independent AI scan quota and the local
Windows limitations above are also explicitly unresolved; successful deterministic
CI and Linux monitoring evidence do not claim that the AI review succeeded.

### Reconciliation with advancing main

Before final delivery, `main` advanced to `fcf4f46` (Windows Server/native
acceptance, #288). Merge conflicts were limited to the automation catalog and
static release trigger allowlist. The catalog retains every upstream entry plus
the four monitoring entries; the allowlist retains both the upstream native
`workflow_call` exception and the periodic schedule/dispatch exception.

After reconciliation: monitoring contracts PASS (39 + one Windows POSIX skip),
automation coverage PASS (85 surfaces), ADR structure PASS (22 ADRs). A direct
diff confirms the monitoring workflow/scripts/baseline/schema and all six Go
suite packages are unchanged from the operationally tested code. ADR-0021's
new native release requirement is preserved. The old-head documentation CI
was cancelled as superseded; final-head normal CI evidence is recorded in PR #293.
