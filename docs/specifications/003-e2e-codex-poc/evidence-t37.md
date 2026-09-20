# Evidence — T37 Executable Workflow

## Delivered behavior

- Lingo starts one workflow only from a resolved Project, configured repository
  and locally linked GitHub Work Item.
- Workflow state persists the fixed Specification, clarification, Plan, Tasks,
  implementation, review, Evidence, reconciliation and completion sequence.
- Every technical gate records one repository-relative artifact reference and
  SHA-256 digest. Reads are anchored with `os.Root`, bounded to 1 MiB and cannot
  escape the resolved repository through traversal or symlinks.
- A failed gate persists `interrupted`; status remains inspectable and `resume`
  resets only that gate for a new attempt.
- Completion can run only after every technical gate passes and requires
  `--authorize-external`. Provider completion preceding a failed local write is
  reported truthfully. Both `provider_committed_local_failed` and
  `work_item_completed_workflow_write_failed` preserve the confirmed Issue URL,
  identity and `CLOSED` state in the workflow/CLI Result.
- Workflow rules live in Lingo application behavior. Codex skills remain thin
  entrypoints.

## Reproduction

```bash
go test ./internal/workflow ./internal/local ./internal/cli ./cmd/lingo -count=1
go test ./cmd/lingo -run ExecutableMinimalLifecycleAndFailurePaths -count=1
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

The executable black-box journey uses fake bounded `git`/`gh` transports and a
real temporary repository association. It starts from CLI environment state,
links Work Item 7, advances four gates, records an implementation failure,
inspects and resumes it, advances through reconciliation, verifies denied
completion without authority, then completes with authority.

## Negative coverage

- wrong gate order and invalid outcome;
- advance after interruption without explicit resume;
- absent, traversing and escaping-symlink artifact references;
- interrupted state and explicit resume;
- completion without external authority;
- missing workflow and unavailable Project/repository through existing resolver
  categories;
- repository binding changed after workflow start;
- strict closed local JSON and private file mode.
- external completion state propagation when Work Item linkage or workflow
  persistence fails after the Provider commit.

Live GitHub mutation and the real unrelated-CWD Codex dogfood remain assigned to
T39/#39. Final human acceptance remains outside T37.
