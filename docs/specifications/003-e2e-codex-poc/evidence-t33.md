# Evidence — T33 Codex Runtime

## Delivered behavior

- `lingo runtime codex install` publishes five thin Axiom skills at Codex user scope.
- `lingo runtime codex status` verifies exact installed content.
- Default root follows current official Codex documentation:
  `$HOME/.agents/skills`; `AXIOM_CODEX_SKILLS_ROOT` isolates tests.
- Install is idempotent, refuses symlinks/unknown or changed content, and rolls
  back only directories created by the failed attempt.
- Skills use the validated `$axiom-<skill>` naming and delegate to Lingo.

## Reproduction

```bash
./scripts/test-codex-skills.sh
go test ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
```

The bundled `quick_validate.py` was present but its PyYAML dependency was absent
from both system and workspace Python on this host. No dependency was installed.
The repository test executes the same supported-name predicate plus embedded
frontmatter/delegation checks in Go and reports the unavailable optional validator.

Unit tests cover absent/configured/idempotent status, unowned conflict,
attempt-local rollback and symlink refusal. Global discovery from an unrelated
Codex session is reserved for the final dogfooding run because it requires a new
host session after installation.

## Naming divergence

Literal `axiom:<skill>` remains unsupported for standalone Codex skills. The
installed names are `axiom-project-configure`, `axiom-project-show`,
`axiom-work-item-create`, `axiom-work-item-run`, and `axiom-work-item-status`.
