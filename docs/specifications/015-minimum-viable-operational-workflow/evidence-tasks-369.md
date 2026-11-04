# Evidence — #369 Tasks / attempt-01

2026-11-04. Companion to [Tasks revision 1](tasks-369-pre-spawn-launch.md).
This is decomposition Evidence; no runtime behavior is implemented or validated.

## Baseline and sources

Clean worktree before edits. Fetched origin/main and canonical branch; local,
remote and PR HEAD all `9fad368ded09fc2f70e0b88d23fef5c4588869e7`.
Main/base `375960907ce1ba4799830e2f0a90ae978598ea23`; PR #370 open/draft,
head fix/369-launch-refused, repository rgomids/axiom. Read all 13 Issue comments
available at inspection, approved baseline/HD-369-D/Plan receipt and favorable
Plan review #5763007250. Public publication authority is explicit in the task.

Read approved amendment, Plan and dependency Evidence; constitution/product
context, ADR-0037, quality/security/documentation policies and canonical SDD/review
skills. Used Ponytail ultra and Ponytail-review discipline. External graph query:
`GRAPHIFY_BACKEND=none graphify query 'launch refusal execution accounting diagnose' --graph /home/grokbot/.graphify/axiom/graphify-out/graph.json --budget 800`.
It returned 708 nodes, truncated to 25; used only for navigation. Plan source
inventory and targeted existing fixture/package discovery support scope, not graph
inference. No graph generated in the repository; no dependencies installed.

## Decomposition review

What this change does: versions eight implementation tasks traceable to P1–P6 and
AC-A–D. Each task names owners/scope, prerequisites, concrete outputs, acceptance,
positive/negative checks, Evidence and stop conditions. All remain PLANNED.

- Specification: all four AC included; HD-369-D remains exactly prospective with
  two lifetime deterministic refusal slots and conservative pending/charged states.
  No historical rescue, r2 budget change, automatic retry or auth prerequisite.
- Architecture: one canonical format-4 aggregate/store/engine; downgrade proof is
  an actual old-reader harness before additive writers, rerun against real bytes.
  Retained identity/double-read/selector/authority/CAS/fence controls stay required.
  No material contradiction with ADR-0037 or retained controls identified.
- Verification: P3 owns source-level parser bounds and ported proof fixtures; P4a
  owns pure boundary, P4b actual CLI route; P5a owns codec/transitions, P5b engine
  reservation/recovery; P6 joins them. B/D are independent of P3. Critical paths
  and shared-file coordination are explicit; dependencies are acyclic.
- Security: canaries across persisted/text/JSON/log sinks, no-effect spies and
  tree snapshots, unknown downgrade refusal before effects, hostile TOML limits,
  frozen history/clone isolation and crash/race boundaries are mandatory negatives.
- Product/design: truthful five-dimension diagnosis; design NOT_APPLICABLE (CLI).
  No new UI, product promise, ADR, automation or future execution authority added.

No blocking finding identified in this documentation review. The smallest useful
split is eight tasks; no agent delegation needed. This is self-review, not an
independent implementation/security review or merge approval.

## Tasks-stage checks

PASS (executed for these documentation changes):
- `./scripts/validate-repository.sh .` — repository/documentation/governance checks;
  script-selected checks only, not future AC behavioral validation.
- `./scripts/check-sensitive-files.sh .` and
  `./scripts/check-sensitive-files.sh --staged .`.
- `git diff --check`, `git diff --cached --check`, staged path/content inspection.
- Local Markdown link checks; assertions for eight unique task IDs, required task
  sections, P1–P6 and AC coverage, exact decoder pin/limits, acyclic dependencies,
  forbidden runtime/config/corpus changes and completion marker/JSON shape.

Dedicated `gitleaks` executable unavailable. Sensitive-file checks remain required
and passed; no equivalent consolidated secret-scanner result is claimed.

NOT_RUN: all future task commands, old-reader harness, new parser integration,
new schema/accounting/crash/race/read-only runtime tests, full Go/platform suite,
AI/vendor battery, host reproduction, authentication/inference, production/E2E.
The prior Plan isolated proof was read, not rerun. No runtime/test/dependency/ADR
files changed. The validator may execute its existing local contract checks; this
is not implementation Evidence for AC-A–D.

## Publication receipt

Final commit/push/remote SHA/PR head and CI observation are recorded in the single
Tasks completion comment on [Issue #369](https://github.com/rgomids/axiom/issues/369),
with stable event ID and read-back validated JSON. This avoids a self-referential
commit SHA in versioned bytes. CI is observed separately, never assumed complete.
Next gate: implement after Tasks review/gates; no implementation begun here.
