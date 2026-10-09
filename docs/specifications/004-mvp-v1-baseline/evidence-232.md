# Issue #232 Evidence — human-first canonical result presentation

Issue: [#232](https://github.com/rgomids/axiom/issues/232) (AXM-11). Executed
2026-10-09 on macOS 27.0.1, arm64, `go1.26.1`, from base `1fede3251885`
plus this branch. Technical completion only: review, merge, release, the real
Runtime observation below and human acceptance remain separate gates.

## Delivered surface

```text
application use case
-> canonical completion event (the exact `--json` bytes, unchanged)
-> presentEvent (internal/cli/presentation.go)
   -> default: deterministic Markdown rendered from that JSON event
   -> --json: the canonical JSON event itself
```

- One renderer. Every completion emitter (`project`, `integration`,
  `work-item`, `workflow`, `runtime`, `first-run`, `version`, maintenance,
  readiness, operational, context and `skill inspect`) now marshals its
  canonical JSON event once and hands it to `presentEvent`. The per-command
  human formatters are removed; human views of Work Item, workflow
  and Runtime results no longer print raw JSON.
- The human view is a pure function of the canonical JSON event: status label
  plus canonical status code, Axiom-authored result, `Next`, `References`,
  `Details`, scalar payload fields, one section per structured payload in JSON
  order, `Provenance` footer. No status, effect or authority rule lives in the
  renderer; no model or Runtime is called.
- JSON marshalling is unchanged for every emitter: each JSON branch reuses its
  previous encoding. No canonical field, payload shape, status or exit code
  changed. Comparing `--json` output of a `main` (`1fede3251885`) build and this
  branch in one sandbox for `version`, `project list|show|validate|archive`,
  `integration list`, `workflow list|status`, `work-item list`,
  `runtime profile validate`, `first-run`, `skill inspect` and
  `runtime codex status` gave byte-identical output except two intended value
  changes: `skill inspect` mode `example` strings no longer carry `--json`, and
  the Runtime status skill SHA-256 values reflect the new skill text.
- Provider-authored and local values render literally (code spans, fenced
  blocks for multi-line text); C0/C1 controls and bidirectional overrides are
  escaped, so a value cannot inject Markdown structure or terminal sequences.
- Bounded, never truncated: the human view may use at most twice the JSON
  bound of its emitter. A literal code span can nearly triple a value made of
  backticks, so a view beyond that bound keeps the canonical summary (status,
  result, next, references, details, provenance) with the same exit code and
  names the withheld payload, pointing to `--json`
  (`TestPresentationOversizedViewKeepsCanonicalOutcome`, from the #291 review
  reproduction with a validated `PrepareSetup` preview). Every canonical field
  is bounded, so the summary always fits
  (`TestPresentationWorstCaseCanonicalSummaryFits`). Skills present that
  summary and may repeat only a read-only or preview command with `--json`.
- Both Runtime skills (`axiom-project`, `axiom-work-item`) invoke the default
  human view, present Lingo's Markdown exactly as returned, read follow-up
  values (preview `digest`, `previewDigest`, `executionId`, `revision`) from
  their labelled fields, and use `--json` only on explicit user request.
  `skill inspect` examples follow the same form. The replaced v0.12.0
  two-skill revision is registered in `sharedSkillHistory`, so owned upgrades
  keep replacing it; `SkillSetVersion`/`BinaryCompatibility` stay `2`.

## Acceptance criteria

| Acceptance criterion | Evidence |
|---|---|
| Representative Project, Work Item and Execution Runtime invocations no longer display raw JSON by default | Skill routing tables and `skill inspect` examples use `axiom <command>` (`TestCanonicalSkillRoutingTableConvergesWithInspection`, `TestSkillOperationMetadataMatchesExecutableParser`, `TestCanonicalRoutingContract` reject `--json` in a route); `TestPresentationGoldenScenarios` covers Project, Work Item and Execution payloads through `emitResponse`; local observation below |
| Equivalent canonical input produces stable human-readable output | `TestPresentationIsDeterministicAndModeIndependentInOutcome`; golden `.md` fixtures |
| Explicit JSON returns the canonical representation without semantic loss | golden `.json` fixtures; unchanged JSON assertions across `internal/cli` and `cmd/lingo` black-box tests |
| Confirmed effects, warnings, Evidence references and provenance cannot disappear | `TestPresentationPreservesEveryCanonicalValue` walks every JSON key and leaf value of every scenario and requires its literal rendering in the human view |
| CLI and supported Runtime integrations produce semantically equivalent results | Matrix below: both skills route to the same `axiom` command, parser and emitter as the CLI |
| Success, no-op, partial, validation failure, denied authority, interrupted/recovery-required, external Provider failure | Scenario fixtures below plus the per-status `completion-*.md` matrix |
| No additional LLM call to format a result | `TestPresentationMakesNoModelOrRuntimeCall` restricts the renderer's imports to the standard library and `internal/completion` |

## Golden fixtures

[`internal/cli/testdata/presentation/`](../../../internal/cli/testdata/presentation/)
holds one `.json` (canonical) and one `.md` (human) file per scenario, rendered
through `emitResponse`, plus `completion-<status>.md` for every canonical status.
Regenerate with `go test ./internal/cli -run 'TestPresentationGoldenScenarios|TestCompletionGoldenMatrix' -update-presentation`.

| Scenario | Status / exit | Payload |
|---|---|---|
| `success-project-show` | `success` / 0 | `project` |
| `no-op-work-item-close` | `success` / 0, `work_item_already_closed` | `change`, `workItem` |
| `partial-work-item-create` | `partial` / 1, confirmed reference + details | `workItem` |
| `validation-failure` | `validation_failure` / 1 | — |
| `denied-authority-project-archive` | `denied_authority` / 1, `authority_denied` | `operational` |
| `interrupted-execution` | `interrupted` / 2 | `workflow` |
| `recovery-required-execution` | `partial` / 1, two confirmed references | `workflow` |
| `provider-failure-unconfirmed` | `failure` / 1, `publication_uncertain` | `workItem` |
| `provider-failure-retryable` | `retryable_failure` / 1, `temporarily_unavailable` | `workItem` |
| `hostile-provider-values` | `success` / 0, backticks, ESC and RLO in a Provider URL | `workItem` |

## CLI / Runtime semantic-equivalence matrix

| Domain | Runtime skill operation | Lingo command (CLI and skill) | Emitter → `presentEvent` |
|---|---|---|---|
| Project | `axiom-project` `configure` | `axiom project configure` | setup / edit |
| Project | `axiom-project` `list`, `show`, `validate` | `axiom project list|show|validate` | project list / project / readiness |
| Project | `axiom-project` `archive`, `reactivate`, `integration` | `axiom project archive|reactivate`, `axiom integration …` | operational / integration / edit |
| Work Item | `axiom-work-item` `create` | `axiom work-item create|select` | work item |
| Work Item | `axiom-work-item` `list`, `show`, `update`, `comment`, `close`, `reopen` | `axiom work-item …` | work item lifecycle |
| Execution | `axiom-work-item` `run` | `axiom workflow start|advance|fact|resume|reconcile` | runtime resolution / workflow |
| Execution | `axiom-work-item` `status` | `axiom workflow status|evidence|list` | workflow / execution list |

The skill adds no parser, emitter or rendering of its own; the same binary call
produces the same canonical event and the same Markdown for both entrypoints.

## Validation

| Command | Result |
|---|---|
| `go vet ./...` | pass |
| `gofmt -l internal cmd` | no output |
| `go test ./...` | pass (all packages, including `cmd/lingo` black-box tests) |
| `./scripts/validate-repository.sh .` on a clean copy of the tracked tree | pass |

## Bounded local observation

A development binary built from this branch ran in an isolated sandbox `HOME`
with no Provider credentials and no Runtime installed; only sandbox-local files
were written. Default output (excerpt; sandbox path redacted):

```text
$ axiom project list
### Succeeded (`success`)

No configured Projects

- **projects:** _none_

**Provenance:** product `Axiom` · version `development` · revision `1fede3251885` · sourceState `dirty`

$ axiom project configure ... --preview-digest <digest> --authorize-local   # without --project-id
### Authority denied (`denied_authority`)

Project setup authority is missing or stale

- **Next:** Review the current preview and authorize its exact digest

$ axiom project show --selector obs
### Succeeded (`success`)

Project resolved

- **References:**
  - `project:2b8656d3-eae7-4295-bf17-9518b0010875`
  - `repository:app`

#### project

- **id:** `2b8656d3-eae7-4295-bf17-9518b0010875`
- **slug:** `obs`
- **source:** `<sandbox>/home/.axiom/projects/obs`
- **repositories:**
  - key `app` · path `<sandbox>/repo` · availability `available`
...

$ axiom --json project list
{"status":"success","result":"No configured Projects","provenance":{...},"projects":[]}
```

## Not executed

- **Real bounded Runtime observation.** A real Codex or Claude session reading
  the installed skills and presenting the Markdown requires real Runtime
  dispatch with subscription credentials, outside this delivery's authority.
  It remains pending separate authorization; the mocks and local CLI
  observation above do not substitute for it.
- Published historical Evidence that quotes the previous `key: value` human
  format is unchanged and remains historical.
