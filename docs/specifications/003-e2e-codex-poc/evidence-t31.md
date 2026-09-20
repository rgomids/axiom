# Evidence — T31 E2E contract

## Inputs inspected

- GitHub issues #19–#21 and #30–#40 on 2026-09-19.
- Specification 002, ADR-0001, ADR-0003, ADR-0004, conceptual/provider boundaries.
- Current CLI, local persistence, POC Evidence and dogfood script on `main`.
- Official OpenAI Codex skill documentation fetched on 2026-09-19:
  <https://developers.openai.com/es-419/docs/build-skills>.
- Bundled current `skill-creator` validator and naming instructions.

## Codex naming observation

Official documentation identifies explicit Codex invocation as `$skill-name` and
user-global discovery at `$HOME/.agents/skills`. The bundled validator applies:

```text
^[a-z0-9-]+$
```

Consequently `axiom:project-configure` is rejected; the valid reversible mapping
is `axiom-project-configure`. No fragile alias is specified.

## Deterministic documentation checks

```bash
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
git diff --check
```

## Review result

Specification compliance, architecture, security and Evidence lanes identified
no Blocker or Major. Existing ADRs decide the material boundaries; no new durable
architecture choice was introduced.
