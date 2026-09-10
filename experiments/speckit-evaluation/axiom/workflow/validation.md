# Validation — Documented Command Deprecation Process

## Context

- Working directory:
  `/Users/rgomids/Projects/axiom/experiments/speckit-evaluation/evidence/workspaces/axiom-run`
- Pre-change `docs/commands.md` SHA-256:
  `828f711ab9dbae62c2217f6ec44a1fdc54262f8178916a883dbacf7392a8e5df`
- Validation uses repository tools and shell commands only. No validator was
  added to the sample repository.

## Deterministic command results

### V-001 — Required files

```sh
test -f docs/deprecations/README.md && test -f docs/deprecations/template.md && test -f README.md && test -f docs/contributing.md && test -f CHANGELOG.md
```

- Exit: `0`
- Outcome: pass.

### V-002 — Required headings

```sh
for heading in '# Command deprecations' '## Record convention' '## Lifecycle and authority' '## Create a record' '## Notice and urgent security exception' '## Reviewer checklist' '## Record index'; do rg -F -x "$heading" docs/deprecations/README.md >/dev/null || exit 1; done; for heading in '# Command Deprecation NNNN — Short name' '## Status' '## Affected command' '## Replacement' '## Rationale' '## Affected users' '## Migration steps' '## Notice or release target' '## Rollback plan' '## Owner' '## Approval evidence' '## Relevant links' '## Urgent security exception'; do rg -F -x "$heading" docs/deprecations/template.md >/dev/null || exit 1; done
```

- Exit: `0`
- Outcome: pass; process and template headings exist.

### V-003 — Lifecycle, notice, and review gate

```sh
rg -F 'Proposed -> Approved -> Deprecated -> Removed' docs/deprecations/README.md docs/deprecations/template.md >/dev/null && rg -F 'at least one published release containing the' docs/deprecations/README.md docs/deprecations/template.md >/dev/null && rg -F 'Missing required migration steps or rollback plan blocks approval.' docs/deprecations/README.md >/dev/null
```

- Exit: `0`
- Outcome: pass.

### V-004 — Internal Markdown links

```sh
ruby -e 'failed=[]; ARGV.each do |file|; File.read(file).scan(/\[[^\]]+\]\(([^)]+)\)/).flatten.each do |target|; next if target.match?(/\A(?:https?:|mailto:|#)/); path=target.split("#",2).first; candidate=File.expand_path(path,File.dirname(file)); failed << "#{file}: #{target}" unless File.exist?(candidate); end; end; abort failed.join("\n") unless failed.empty?' README.md docs/contributing.md docs/deprecations/README.md docs/deprecations/template.md experiment-output/*.md
```

- Exit: `0`
- Outcome: pass; checked local links resolve.

### V-005 — Whitespace

```sh
! rg -n '[[:blank:]]+$' README.md CHANGELOG.md docs/contributing.md docs/deprecations experiment-output && git diff --check
```

- Exit: `0`
- Outcome: pass; no trailing whitespace or diff whitespace errors.

### V-006 — Existing command preservation

```sh
test "$(shasum -a 256 docs/commands.md | awk '{print $1}')" = '828f711ab9dbae62c2217f6ec44a1fdc54262f8178916a883dbacf7392a8e5df'
```

- Exit: `0`
- Outcome: pass; command names and descriptions remain byte-for-byte unchanged.

### V-007 — Implementation scope

```sh
test "$(git diff --name-only | sort | tr '\n' ' ')" = 'CHANGELOG.md README.md docs/contributing.md ' && test "$(find docs/deprecations -maxdepth 1 -type f | sort | tr '\n' ' ')" = 'docs/deprecations/README.md docs/deprecations/template.md '
```

- Exit: `0`
- Outcome: pass; implementation contains three tracked Markdown edits and two
  new Markdown documents.

### V-008 — Executable-bit exclusion

```sh
test -z "$(find docs/deprecations experiment-output -type f -perm -111 -print)"
```

- Exit: `0`
- Outcome: pass; no new document is executable.

### V-009 — Sensitive-content scan

```sh
/Users/rgomids/Projects/axiom/scripts/check-sensitive-files.sh --directory .
```

- Exit: `0`
- Output: `PASS: sensitive-file checks passed (directory)`
- Outcome: pass for prohibited sensitive paths and high-confidence private-key,
  GitHub-token, and AWS-key patterns covered by the existing Axiom scanner.

### V-010 — No current deprecation record

```sh
test "$(find docs/deprecations -maxdepth 1 -type f ! -name README.md ! -name template.md | wc -l | tr -d ' ')" = '0'
```

- Exit: `0`
- Outcome: pass; process introduction did not deprecate a command.

### V-011 — Local template-use simulation

```sh
sed 's/NNNN/0001/g; s/Short name/Example command/g' docs/deprecations/template.md | rg -F -x '# Command Deprecation 0001 — Example command' >/dev/null
```

- Exit: `0`
- Outcome: pass; template can produce a locally named record header without a
  provider or network.

### V-012 — Consolidated scanner availability

```sh
command -v gitleaks
```

- Exit: `1`
- Output: empty.
- Outcome: `gitleaks` unavailable; not installed automatically.

## Rejected check attempt

An initial functional check proposed copying the template to a generated `/tmp`
file and deleting it afterward:

```sh
tmp_record="$(mktemp /tmp/axiom-deprecation-record.XXXXXX.md)" && cp docs/deprecations/template.md "$tmp_record" && test -s "$tmp_record" && rg -F -x '## Migration steps' "$tmp_record" >/dev/null; outcome=$?; rm -f "$tmp_record"; exit $outcome
```

- Process exit: none; execution guard rejected `rm -f` before launch.
- Evidence status: not executed and not counted as a pass.
- Safe replacement: V-011, which performs a non-writing stream simulation.

## Test-boundary mapping

- Unit-like documentation checks: V-001 through V-003.
- Integration check: V-004 validates links between documentation artifacts.
- Functional black-box check: V-011 simulates local template naming.
- Regression and scope checks: V-005 through V-010.

## Coverage limits

- Human-readable quality, policy coherence, and faithful interpretation require
  semantic review; passing commands alone do not prove them.
- `gitleaks` coverage is unverified because the tool is unavailable.
- No runtime, dependency, API, database, provider, or network behavior exists to
  test.

## Final closure suite

All commands below ran after implementation, review, reconciliation, and
creation of every required evidence artifact.

### C-001 — Evidence artifact presence

```sh
for f in experiment-output/intake.md experiment-output/specification.md experiment-output/clarifications.md experiment-output/plan.md experiment-output/tasks.md experiment-output/decisions.md experiment-output/validation.md experiment-output/review.md experiment-output/execution.md; do test -s "$f" || exit 1; done
```

- Exit: `0`.

### C-002 — Final required headings

```sh
for heading in '# Command deprecations' '## Record convention' '## Lifecycle and authority' '## Create a record' '## Notice and urgent security exception' '## Reviewer checklist' '## Record index'; do rg -F -x "$heading" docs/deprecations/README.md >/dev/null || exit 1; done; for heading in '# Command Deprecation NNNN — Short name' '## Status' '## Affected command' '## Replacement' '## Rationale' '## Affected users' '## Migration steps' '## Notice or release target' '## Rollback plan' '## Owner' '## Approval evidence' '## Relevant links' '## Urgent security exception'; do rg -F -x "$heading" docs/deprecations/template.md >/dev/null || exit 1; done
```

- Exit: `0`.

### C-003 — Final local-link resolution

```sh
ruby -e 'failed=[]; ARGV.each do |file|; File.read(file).scan(/\[[^\]]+\]\(([^)]+)\)/).flatten.each do |target|; next if target.match?(/\A(?:https?:|mailto:|#)/); path=target.split("#",2).first; candidate=File.expand_path(path,File.dirname(file)); failed << "#{file}: #{target}" unless File.exist?(candidate); end; end; abort failed.join("\n") unless failed.empty?' README.md docs/contributing.md docs/deprecations/README.md docs/deprecations/template.md experiment-output/*.md
```

- Exit: `0`.

### C-004 — Final whitespace

```sh
! rg -n '[[:blank:]]+$' README.md CHANGELOG.md docs/contributing.md docs/deprecations experiment-output && git diff --check
```

- Exit: `0`.

### C-005 — Frozen-input and command hashes

```sh
test "$(shasum -a 256 docs/commands.md | awk '{print $1}')" = '828f711ab9dbae62c2217f6ec44a1fdc54262f8178916a883dbacf7392a8e5df' && test "$(shasum -a 256 .experiment/operator-answers.md | awk '{print $1}')" = 'de1b7de2e6466c5200cddc15dc781a4c8fdb05dc27e6fb5f1d067313127b025a'
```

- Exit: `0`.

### C-006 — Exact workspace scope

```sh
LC_ALL=C diff -u <(printf '%s\n' '.experiment/operator-answers.md' 'CHANGELOG.md' 'README.md' 'docs/contributing.md' 'docs/deprecations/README.md' 'docs/deprecations/template.md' 'experiment-output/clarifications.md' 'experiment-output/decisions.md' 'experiment-output/execution.md' 'experiment-output/intake.md' 'experiment-output/plan.md' 'experiment-output/review.md' 'experiment-output/specification.md' 'experiment-output/tasks.md' 'experiment-output/validation.md') <({ git diff --name-only; git ls-files --others --exclude-standard; } | LC_ALL=C sort -u)
```

- Exit: `0`.

### C-007 — Executable bits and no current record

```sh
test -z "$(find docs/deprecations experiment-output -type f -perm -111 -print)" && test "$(find docs/deprecations -maxdepth 1 -type f ! -name README.md ! -name template.md | wc -l | tr -d ' ')" = '0'
```

- Exit: `0`.

### C-008 — Final sensitive-content scan

```sh
/Users/rgomids/Projects/axiom/scripts/check-sensitive-files.sh --directory .
```

- Exit: `0`.
- Output: `PASS: sensitive-file checks passed (directory)`.

### C-009 — Required workflow sections

```sh
for heading in '## Facts' '## Requirements' '## Assumptions' '## Unknowns' '## Non-goals'; do rg -F -x "$heading" experiment-output/intake.md >/dev/null || exit 1; done; for heading in '## Scenarios' '## Acceptance criteria' '## Explicit non-goals' '## Constitution check'; do rg -F -x "$heading" experiment-output/specification.md >/dev/null || exit 1; done; rg -F -x '## HC-001 — Policy and artifact contract' experiment-output/clarifications.md >/dev/null && rg -F -x '## Shared completion-criteria comparison' experiment-output/execution.md >/dev/null
```

- Exit: `0`.

### C-010 — Completed task and execution status

Two initial status-count commands were invalid checks:

```sh
rg -F -x '- **Status:** Completed' experiment-output/tasks.md | test "$(wc -l | tr -d ' ')" = '7' && rg -F -x 'Completed. Implementation, review, documentation reconciliation, and final' experiment-output/execution.md >/dev/null
```

- Exit: `1`.
- Output: `rg: unrecognized flag -`.
- Cause: leading hyphen in the fixed-string pattern required `--`.

```sh
rg -F -x -- '- **Status:** Completed' experiment-output/tasks.md | test "$(wc -l | tr -d ' ')" = '7' && rg -F -x 'Completed. Implementation, review, documentation reconciliation, and final' experiment-output/execution.md >/dev/null
```

- Exit: `1`.
- Output: empty.
- Cause: count command substitution was placed in the pipeline consumer and did
  not count the upstream `rg` output as intended.

Corrected check:

```sh
test "$(rg -F -x -- '- **Status:** Completed' experiment-output/tasks.md | wc -l | tr -d ' ')" = '7' && rg -F -x 'Completed. Implementation, review, documentation reconciliation, and final' experiment-output/execution.md >/dev/null
```

- Exit: `0`.
- Outcome: all seven tasks and final execution status are completed.
