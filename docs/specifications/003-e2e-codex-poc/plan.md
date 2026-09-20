# Plan — Specification 003: E2E Codex POC

## Status

Approved and implementation-authorized by the 2026-09-19 human execution request.

## Architecture

```text
Codex global skill --thin invocation--> Lingo CLI
                                      -> application services
portable Project <-> strict codecs   -> protected local catalog/state
resolved repository                  -> bounded workflow/Evidence
GitHub port -> gh transport          -> GitHub Issue
```

Keep existing `project`, `projectapp`, `manifest` and `local` contracts. Add
cohesive application packages for installation/runtime bootstrap, catalog and
configuration, Work Items, and workflow. CLI parses inputs and renders stable
results; it does not own rules.

## Delivery mapping

| Issue | Planned implementation | Verification |
|---|---|---|
| #32 | installer plus version/source inspection | isolated PATH root, rerun/conflict/failure tests |
| #33 | Codex adapter and five thin skills | validator, isolated user root, unrelated-CWD discovery |
| #34 | closed Project catalog and repository resolver | ID/slug, ambiguity, moved/missing, multi-repository tests |
| #35 | atomic configure application + CLI | CLI black-box and thin-skill delegation |
| #36 | Work Item port, GitHub `gh` adapter, local linkage | fake transport plus live Evidence |
| #37 | resumable ordered workflow and Evidence | gate/failure/resume/completion tests |
| #38 | coherent CLI/JSON/skill mapping | black-box equivalence matrix |
| #39 | clean-room dogfood script and report | local run plus Linux/macOS checks |
| #40 | docs/status/final report | traceability audit and final branch review |

## Persistence

Extend the existing state root using closed JSON files and safe filesystem
operations. Project catalog entries are derived from validated portable Projects
and local repository bindings, addressed by Project ID. Workflow and Work Item
state are separate from `installation.json`, so Specification 002 wire contracts
do not silently change. Writes use no-replace creation or expected-byte atomic
replacement, fail closed on unknown artifacts and retain recovery Evidence.

## Command surface

Target commands:

```text
lingo version
lingo runtime codex install|status
lingo project configure|show|resolve
lingo work-item create|select|show|complete
lingo workflow start|advance|status|evidence
```

Commands support `--json` where skills need deterministic interpretation.
Mutation commands require explicit flags for external GitHub effects. Guided
prompts may be added where practical; repeatable non-interactive flags are the
acceptance baseline.

## Validation strategy

- unit tests for state machines, authority and selectors;
- adapter integration tests with isolated roots and fake `gh`;
- executable black-box tests from unrelated CWD;
- installer/skill conflict and idempotency tests;
- workflow failure/resume and bounded-output tests;
- live GitHub dogfooding only after deterministic tests;
- repository, race, vet, build, module and sensitive-file checks per issue;
- final `scripts/dogfood-poc.sh` covers the complete E2E journey.

## Rollback and limits

Intermediate issue PRs squash into the permanent POC branch and never into
`main`. Local test roots are disposable. Installer never removes unowned content.
Live GitHub mutations are traceable by Issue/PR URL. No schema migration,
cross-machine state or automatic recovery framework is introduced.
