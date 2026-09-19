# Evidence — T36 GitHub Work Items

## Delivered behavior

- Provider-neutral Work Item application ports separate resolution, repository
  identity, provider actions and local linkage.
- The GitHub adapter resolves the configured repository's `origin` using bounded
  `git` execution, then invokes only specific `gh issue` commands.
- Create, comment and complete require `--authorize-external`; select/show remain
  read/local operations.
- GitHub authentication stays in the existing `gh` credential source. No token or
  credential value has a state or Evidence field.
- Closed JSON linkage records Project ID, repository key, `owner/repo`, Issue
  number, canonical HTTPS URL and state under protected local state.
- Provider success followed by local write failure is reported as
  `provider_committed_local_failed`, never rollback.

## Reproduction

```bash
go test ./internal/workitem ./internal/local -run 'WorkItem|GitHub' -count=1
go test ./cmd/lingo -run ExecutableMinimalLifecycleAndFailurePaths -count=1
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

Deterministic fakes cover Git remote resolution, create/read/comment/close,
authority denial before provider calls, exact URL/repository/number validation,
local round trip and truthful post-provider failure. Live GitHub mutation is
reserved for #39 dogfooding.

## Security limits

Commands use argv without a shell, absolute resolved executables, a 15-second
timeout and 64 KiB combined output bound. Provider errors are mapped to fixed
categories; raw stderr, Work Item content and credentials are not rendered.
This is not a general GitHub client and supports only one GitHub `origin` remote.
