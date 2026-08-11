# Quickstart: Validate Documented Command Deprecation Process

Run from the sample repository root. Commands add no dependency, mutate no provider, and require no
network. Expected result: every command exits successfully unless marked as an availability probe.

## 1. Required files

```sh
test -f README.md
test -f CHANGELOG.md
test -f docs/commands.md
test -f docs/contributing.md
test -f docs/deprecations/README.md
test -f docs/deprecations/template.md
```

## 2. Required template headings

```sh
for heading in \
  'Status' \
  'Affected Command' \
  'Replacement' \
  'Rationale' \
  'Affected Users' \
  'Migration Steps' \
  'Notice / Release Target' \
  'Rollback Plan' \
  'Owner' \
  'Approval Evidence' \
  'Relevant Links'; do
  rg -Fqx "## $heading" docs/deprecations/template.md
done
```

## 3. Known repository-local links

```sh
test -f docs/commands.md
test -f docs/contributing.md
test -f docs/deprecations/README.md
test -f docs/deprecations/template.md
test -f CHANGELOG.md
rg -n '\[[^]]+\]\([^)]+\)' README.md docs CHANGELOG.md
```

Review the reported Markdown links; each repository-local target above must exist. No general link
checker is added by this feature.

## 4. Existing command invariants

```sh
test "$(rg -c '^## `example (validate|package)`$' docs/commands.md)" -eq 2
rg -Fqx 'Validates repository documentation before review.' docs/commands.md
rg -Fqx 'Packages validated repository documentation for distribution.' docs/commands.md
```

## 5. Whitespace

```sh
! rg -n '[[:blank:]]+$' README.md CHANGELOG.md docs specs/001-command-deprecation-process
```

## 6. Scope

```sh
git status --short --untracked-files=all
git status --porcelain=v1 --untracked-files=all \
  | cut -c4- \
  | rg -v '^(\.agents/|\.experiment/|\.specify/|specs/|README\.md$|CHANGELOG\.md$|docs/)'
```

Expected second command output: empty. Review first command to distinguish official Spec-Kit
infrastructure and scenario inputs from feature implementation files.

## 7. Sensitive content

```sh
command -v gitleaks
rg -n '(AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY-----|gh[pousr]_[A-Za-z0-9_]{20,})' \
  README.md CHANGELOG.md docs specs/001-command-deprecation-process
```

`command -v gitleaks` is an availability probe. If unavailable, report scanner coverage as
unverified. High-confidence fallback search must produce no matches.

## 8. Recorded outcomes

### Baseline before implementation — 2026-08-11

- PASS — `docs/commands.md` contained exactly `example validate` and `example package` with the
  frozen descriptions.
- PASS — `docs/deprecations/` did not exist; no command was documented as deprecated.
- PASS — initial feature scope contained only official Spec-Kit infrastructure, scenario inputs,
  constitution, and specification/design artifacts.
- NOT CREATED — repository-root `.gitignore`; no generated technology artifacts exist, and a
  non-Markdown file would violate the documentation-only constitution. Official `.specify/.gitignore`
  remains unchanged.

### User Story 1 acceptance — 2026-08-11

- PASS — explicit documented-command deprecation requires a record.
- PASS — command removal can occur only after an earlier record and lifecycle transitions.
- PASS — editorial correction preserving name, description, behavior, and lifecycle requires no
  record.
- RESIDUAL — whether rename, replacement, or incompatible behavior change independently triggers a
  record remains a human decision; documentation does not infer an answer.
- PASS — all four stages, responsible humans, normal notice, and urgent-security exception content
  are explicit.

### User Story 2 acceptance — 2026-08-11

- PASS — `docs/deprecations/README.md` links the reusable template and states the frozen path and
  `NNNN-short-name.md` shape.
- PASS — template contains all eleven exact required headings and instructions for explicit absence.
- PASS — every known repository-local link target used by the index and template exists.
- PASS — agent located and structurally reviewed the template well inside 10 minutes.
- UNVERIFIED — SC-002's human contributor completion time; no separate human timed exercise occurred.
- RESIDUAL — number allocation and collision authority remain unspecified by the frozen answer.

### User Story 3 acceptance — 2026-08-11

- PASS — reviewer checklist blocks approval when migration, rollback, authority, approval, or other
  mandatory content is absent.
- PASS — blank reusable template is correctly classified as incomplete and not approval-ready.
- PASS — a hypothetical fully populated record is approval-ready only after all lifecycle, notice,
  exception, migration, rollback, link, and authority checks are satisfied.
- PASS — reviewer commands cover file presence, exact headings, known local targets, whitespace,
  command invariants, and final scope without executable automation.
- UNVERIFIED — SC-003's agreement between two independent human reviewers; only one agent review was
  available.

### Final implementation validation — 2026-08-11

- PASS — all six required repository files exist.
- PASS — all eleven template headings and all seven process headings match exactly.
- PASS — every currently documented repository-local Markdown link resolves to an existing file.
- PASS — no trailing whitespace found across `README.md`, `CHANGELOG.md`, `docs/`, or feature
  artifacts. `git diff --check` also passed; direct search covered untracked files omitted by Git diff.
- PASS — `example validate` and `example package` headings and descriptions remain byte-for-word
  identical; `git diff -- docs/commands.md` is empty.
- PASS — changed and created paths are limited to official Spec-Kit infrastructure, scenario inputs,
  feature artifacts, `README.md`, `CHANGELOG.md`, `docs/contributing.md`, and
  `docs/deprecations/*.md`.
- PASS — no application source, executable validator, dependency manifest, CI file, provider
  integration, credential, or network requirement was added.
- PASS — high-confidence sensitive-pattern fallback found no AWS access key, private-key header, or
  GitHub token pattern.
- UNAVAILABLE — `gitleaks`, `lychee`, and `markdown-link-check`; full secret scanning and general link
  scanning remain unverified. Known local targets were checked explicitly.
- PASS — initial `$speckit-analyze` found no critical issue, one high consistency issue, and three
  medium coverage/consistency issues. Owning artifacts were corrected. Second read-only analysis
  found 18/18 buildable requirements covered, zero critical/high material inconsistencies, three
  declared residual ambiguities, and zero constitution conflicts.
- HUMAN INTERACTION — one batched response from `.experiment/operator-answers.md`; no additional
  human response or recommended multiple-choice answer was substituted.
- RESIDUAL — trigger coverage beyond explicit deprecation, `NNNN` allocation/collision authority,
  and provider-neutral proof of a published notice-bearing release remain undecided.
- ARCHITECTURE — repository-local Markdown, indexed individual records, four-stage human lifecycle,
  and one-off validation were surfaced. No ADR exists or is warranted because no runtime,
  dependency, external interface, provider boundary, or difficult-to-reverse technical architecture
  was introduced.
- DIFFICULTIES — frozen batch did not answer the additional trigger-boundary question; generated
  checklists remained incomplete by design; Git diff does not inspect untracked whitespace; external
  scanners were unavailable. Each limitation is preserved rather than inferred away.
- CHECKLIST GATE — `requirements.md` is 14/16 and `deprecation-requirements.md` is 0/20 because the
  latter is an unevaluated requirements-quality question set. Explicit user instruction to run the
  full remaining sequence authorized implementation without marking unanswered items complete.
- CONVERGENCE — PASS. `$speckit-converge` checked 13 functional requirements, 5 success
  criteria, 7 acceptance scenarios, 4 plan decisions, 7 constitution principles, and 18 completed
  tasks. It found zero actionable gaps, appended no task, and left `tasks.md` byte-for-byte unchanged.

### Shared completion criteria audit — 2026-08-11

1. PASS — durable intake and one-batch clarification evidence: `.experiment/brief.md`,
   `.experiment/operator-answers.md`, and `spec.md` clarifications.
2. PASS — specification contains acceptance scenarios and explicit non-goals.
3. PASS — `plan.md`, `research.md`, `data-model.md`, and `quickstart.md` define the artifact design.
4. PASS — `tasks.md` contains 18 ordered, story-mapped tasks; all are complete.
5. PASS — implementation is the smallest coherent set: process/index, template, three entry-point or
   lifecycle updates, and no fictional record.
6. PASS — two read-only analysis runs and one clean convergence result cover consistency.
7. PASS — deterministic commands and dated outcomes appear above; unavailable scanners are named.
8. PASS — README, contributor guidance, and `CHANGELOG.md` received lifecycle treatment.
9. PASS — architecture decisions and explicit no-ADR rationale are recorded.
10. PASS — difficulties, three residual ambiguities, one human batch, and unverified outcomes are
    explicit.
11. PASS — scope checks found no application code, executable validator, dependency, provider
    mutation, credential, network requirement, or existing-command change.
