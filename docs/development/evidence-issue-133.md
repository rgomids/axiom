# Hierarchical CLI help — implementation Evidence

Technical contract: [GitHub #133](https://github.com/rgomids/axiom/issues/133).
Product tracking: [Linear AXM-10](https://linear.app/rgomids/issue/AXM-10/add-hierarchical-cli-help).
Scope: I133-T01–T03, one presentation-only delivery. Baseline: current `main`
`73dce6d` (v0.11.0), checked on 2026-10-08. GitHub #133 was open with
`status:active`, AXM-10 was In Progress, and all nine native predecessor links,
including #230, were closed. No overlapping help PR was open. These checks
caused no tracking mutations.

## Public command coverage matrix

Every row is exercised by the tree traversal in
`internal/cli/hierarchical_help_test.go`. Both aliases, output-mode precedence,
deterministic rendering and nil-Service execution are asserted at every node.

| Group/surface | Direct children / leaves |
|---|---|
| root | help, version, first-run, skill, runtime, project, integration, work-item, workflow, compatibility, artifact, recovery, upgrade; windows-permissions on Windows |
| skill | inspect |
| runtime | codex, claude, profile |
| runtime codex | install, status |
| runtime claude | install, status |
| runtime profile | validate, preview |
| project | configure, list, show, resolve, validate, archive, reactivate, init, reopen, update, install, context |
| project context | show, default-set, default-clear, session-set, session-clear, session-end |
| integration | list, show, validate, disable, enable, remove |
| work-item | create, select, list, show, update, comment, close, reopen, complete |
| workflow | start, advance, fact, resume, status, evidence, list, reconcile |
| compatibility | inspect, backup, export |
| artifact | cleanup, retire |
| recovery | inspect, apply |
| upgrade | standalone maintenance leaf |
| Windows windows-permissions | restore |

Windows recovery metadata, both aliases, value consumption, invalid group/leaf
guidance and fixed-order syntax are also exercised on the local host through
`TestHelpWindowsRecoveryRoutingWithoutWindowsEffects` and
`TestHelpWindowsRecoveryKeepsFixedOrderSyntax`. `commandTree` exposes this
surface only on Windows. The private `install-release` facade remains private.

## Acceptance criteria

| # | Criterion | Status | Executable / review Evidence |
|---|---|---|---|
| 1 | Root --help, -h and existing help without composition | PASS | Every-command nil-Service tests; executable precomposition black-box tests; existing Help compatibility test |
| 2 | Both aliases for every public group | PASS | Complete traversal and explicit Windows recovery routing tests |
| 3 | Both aliases for every public leaf | PASS | Same traversal, including no-flag, positional skill and maintenance leaves |
| 4 | Every public top-level surface exposed | PASS | Root direct-child assertions; independent typed parser-action AST inventory; Windows visibility assertion |
| 5 | Every valid direct group child exposed | PASS | Per-group direct-child assertions and registry-backed routing |
| 6 | Accurate usage and required/optional inputs | PASS | Actual FlagSet descriptions, repeatability and shared requirement assertions; presence-omission maintenance tests; conditional alternative/replay metadata |
| 7 | Concise valid invocation examples | PASS | Every leaf renders an example; validation example admission test prevents conflicting selector forms; Windows exact-order parser test; placeholders require substitution |
| 8 | Invalid input fails with closest help | PASS | Unknown paths, invalid supplied leaf flags/duplicates/types/positionals, existing strict-parser regressions and Windows group/leaf guidance assertions |
| 9 | JSON execution failures remain structured | PASS | Human/JSON invalid-help tests and executable strict-parser black-box tests; status/category preserved and help placed in next or historical nextAction |
| 10 | Help performs no state resolution or effects | PASS | All public nodes render with nil Service; isolated executable runs with invalid/absent/existing roots, Runtime/Provider traps and unchanged filesystem snapshots; main intercepts before composition |
| 11 | Parser/registry and flag consistency tests | PASS | Independent Go AST action discovery; FlagSet-backed help/parse tests; shared requirement assertions; Windows parser/help shared names and syntax |
| 12 | No duplicated domain/workflow logic | PASS | Independent engineering review and diff: registry/renderer contain presentation only; supplied syntax uses existing FlagSet checks; domain validators and use cases retain authority |
| 13 | Existing command compatibility preserved | PASS | Existing CLI and executable regression suites; complete race suite; Windows fixed-order admission retained; help aliases/guidance are the intentional changes |

These statuses assess technical delivery, not human acceptance, merge or release.
No live Provider/Runtime call is required or authorized for help validation.

## Validation

Executed successfully on the implementation tree:

- `go test ./internal/cli -count=1`
- `go test ./internal/cli -run 'Help|CommandRequirements' -count=1`
- `go test ./cmd/lingo -run 'TestHierarchicalHelpBlackboxPrecedesComposition|TestExecutableRejectsSingleHyphenSelectorFlagsBeforeEffects' -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- `GOOS=windows GOARCH=amd64 go build ./...`
- `./scripts/validate-repository.sh .`
- `./scripts/validate-agent-package.sh --maintainer-harness .`
- `./scripts/check-sensitive-files.sh .`
- `gitleaks detect --source . --no-git --redact`
- `git diff --check`

The initial package check without `--maintainer-harness` refused the repository's
existing Claude discovery symlinks, as the generated-Codex-package mode requires.
The repository validator already selects the maintainer mode; the explicit
maintainer invocation passed. No adapter or package policy was changed.

Local host execution does not prove native Windows filesystem/permission
restoration. That operation remains unchanged beyond its pure argument adapter
and error guidance. Native executable regression is a separate Windows CI lane;
remote checks are reported on the PR, never inferred from local tests.

## Implementation and review

- `internal/cli/commands.go`: presentation command tree, lookup and dispatch
  membership; new commands extend this registry.
- `internal/cli/help.go`: deterministic rendering, lexical help routing,
  execution-FlagSet syntax checks and safe nearest-help guidance.
- Existing flag builders: extracted where necessary so parsing and help use
  identical registrations. Context clear/end no longer register a selector
  that every supplied value previously caused the parser to reject.
- `internal/cli/help_requirements.go`: existing skill presence rules plus tested
  historical, conditional replay, maintenance and context presentation metadata.
  Application-only admission requirements are labelled explicitly.
- `cmd/lingo/main.go`: help interception before any application composition.
- `internal/cli/windows_permissions.go` and the Windows onboarding adapter:
  shared flag names and preserved separate-value, fixed-order recovery syntax.
- Focused tests plus updated legacy guidance assertions: input privacy, failure
  categories, known behavior and all help surfaces remain covered.
- `docs/commands.md`: delivered invocation and registration mechanism.

Delegation used bounded independent exploration/requirements, test authoring and
engineering review, with inherited runtime settings. Integration and final
technical claims remained with the main session. Review found and resolved an
invalid validation example, unsupported session syntax, missing Windows recovery
coverage, Windows nearest-group guidance and invalid supplied syntax under help.
Shared presence-aware CLI checks reject malformed selectors and conflicts even
when other required inputs are omitted for help; ordinary execution preserves
its existing validation order. Final focused re-review found no blocking finding.
No new Specification, dependency, framework, authority boundary or ADR was needed.

Pending human steps: PR review, required CI, merge and subsequent release under
separate authority. GitHub #133 and Linear AXM-10 remain open/In Progress; no human
acceptance is recorded by this Evidence.
