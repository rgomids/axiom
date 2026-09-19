# Evidence — T34 global Project resolution

## Delivered behavior

- `lingo project resolve --selector <slug-or-id>` scans protected ID-addressed
  installation records, never caller CWD.
- Exact canonical ID and observed slug are supported.
- Duplicate slugs fail as `project_ambiguous`; no implicit selection occurs.
- Portable source and every repository binding must resolve to an absolute,
  existing non-symlink directory.
- Multiple repository bindings remain intact in the resolved application value.
- No absolute repository path is added to `axiom.yaml`.

## Reproduction

```bash
go test ./internal/local -run ResolveProject -count=1
go test ./internal/cli ./cmd/lingo
go test ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

The integration tests change process CWD to an unrelated temporary directory,
resolve by slug and ID, retain two repository associations, reject duplicate
slugs, and report a removed repository path without fallback scanning.

## Limitations

This issue adds read-only resolution over the existing strict local record.
Project configuration and binding publication belong to #35. Repository Git
identity/remote matching remains conservative metadata and is not re-observed here.
