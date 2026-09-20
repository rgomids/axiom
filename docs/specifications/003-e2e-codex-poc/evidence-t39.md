# Evidence — T39 End-to-End Dogfooding

Structured observations: [evidence-t39.json](evidence-t39.json).

## Deterministic complete journey

`./scripts/dogfood-poc.sh` now creates isolated install, Project, local-state,
global-skill, repository and provider roots. From an unrelated CWD it proves:

```text
repository install -> Lingo on PATH -> Codex skill install/status
-> missing skill detection/repair -> Project configure/show
-> missing and moved Project failures -> denied GitHub mutation
-> Work Item create/link -> ordered workflow -> failed implementation
-> inspect -> resume -> review/Evidence/reconciliation
-> denied completion -> authorized completion
```

The 2026-09-19 macOS run exited 0 with:

```json
{"evidenceVersion":1,"evidence":"axiom_e2e_dogfood","cwdIndependent":true,"globalSkillCount":5,"workItem":7,"workflow":"completed","binarySha256":"0a529495e080db4c90d8e1190d35df8345b32b3336b0cebfb6c187895b0cb600","workflowSha256":"dd8835e5b3d610eeb83e1d9038639fdc92997fa313b956950c6438baec596752","result":"pass"}
```

Project IDs are generated per isolated run, so workflow-state hashes are
per-run observations rather than fixed golden values.

Reproduce:

```bash
bash -n scripts/dogfood-poc.sh
./scripts/dogfood-poc.sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
gitleaks git --no-banner --redact --log-opts='--all'
```

## Real user-global Runtime observation

From `$HOME`, the repository installer updated `$HOME/.local/bin/lingo` to
`poc-5ae90de0e619`; SHA-256 was
`0a529495e080db4c90d8e1190d35df8345b32b3336b0cebfb6c187895b0cb600`.
`lingo runtime codex install` upgraded the five exact prior Axiom skills in
`$HOME/.agents/skills`; status returned `codex_ready`.

Lingo then configured logical Project `axiom-e2e-poc`, selected live GitHub
Issue #39 and started its workflow. A new projectless Codex task
`01a0bb43-9b1a-7513-92d3-b56d9dbee8bf` started at:

```text
/Users/rgomids/Documents/Codex/2026-09-19/axiom-poc-runtime-dogfood
```

It discovered and explicitly invoked `$axiom-work-item-status` using only
Project `axiom-e2e-poc`, repository key `main`, and Work Item `39`. Its exact
read-only Lingo command contained no filesystem path. Returned JSON reported
`currentGate=implementation` and resolved
`repositoryPath=/Users/rgomids/Projects/axiom`.

The projectless task's attempt to write the resolved repository correctly hit
the task sandbox's separate user-approval gate. No authority was silently
expanded. It proposed one bounded help-mapping assertion; this primary task,
already authorized for the repository, applied it and reran the dogfood script.
Thus global discovery and path resolution are directly proved; cross-sandbox
write approval remains an explicit host control, not an Axiom failure.

## #19 and #20 reconciliation

New E2E coverage closes previous gaps for bindings, Runtime, Provider, guided
configuration and workflow integration. It does not prove physical power-loss
durability, every filesystem rename/sync fault, or arbitrary hostile same-UID
interleavings required by the unresolved #19 threat-model gate. #19 remains
open. Because #20 explicitly depends on deterministic persistence/recovery
Evidence, #20 also remains open pending that gate; no green E2E check is treated
as acceptance or waiver.
