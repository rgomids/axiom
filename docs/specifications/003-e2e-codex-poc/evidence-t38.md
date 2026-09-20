# Evidence — T38 Equivalent Lingo and Codex Entrypoints

## Delivered behavior

- Direct Lingo defaults to concise human output and needs no Codex Runtime.
- `lingo --json ...` emits stable operation/status/category fields plus typed
  Project, Work Item or workflow payloads where applicable.
- `project show` now implements the command referenced by the global skill and
  returns resolved repository associations without consulting caller CWD.
- Workflow JSON returns `repositoryPath`, `currentGate`, every gate state,
  artifact reference and SHA-256 digest. Codex does not reconstruct paths or
  workflow state.
- `lingo help` maps every installed `$axiom-*` skill to stable Lingo commands.
- Embedded skills contain explicit `lingo --json` templates, collect missing
  conversational inputs, and leave domain validation and authority to Lingo.
- Runtime install upgrades only exact prior Axiom skill bytes. Changed/unowned
  content remains a deterministic `codex_skill_conflict`.

## Reproduction

```bash
go test ./internal/cli ./internal/codexruntime ./cmd/lingo -count=1
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

Executable black-box coverage validates Project, Work Item, interrupted workflow
and repository path payloads through the JSON surface. Unit coverage validates
human output, help mapping, exact skill delegation, deterministic categories and
known-version skill upgrade without weakening unowned-conflict protection.

## Authority equivalence

Skills supply no implicit `--authorize-external`. Create/comment/complete still
fail before provider mutation without that explicit flag. Project resolution,
artifact containment, gate ordering and Evidence validation remain the same
application methods for human and JSON callers.

Literal `axiom:<skill>` remains unsupported by the current standalone Codex skill
name contract. `$axiom-<skill>` is the already documented reversible POC mapping.
