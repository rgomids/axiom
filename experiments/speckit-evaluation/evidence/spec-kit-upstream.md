# Current Official Spec-Kit Profile

Evaluated source: GitHub Spec-Kit `v0.16.2`, commit
`4871b485f97c7fa452ec58eba325d87536c55c34`.

## Recommended installation

Official documentation recommends a persistent source install pinned to a
release tag:

```sh
uv tool install specify-cli \
  --from git+https://github.com/github/spec-kit.git@v0.16.2
```

The experiment used the official one-time alternative:

```sh
uvx --from git+https://github.com/github/spec-kit.git@v0.16.2 \
  specify init --here --force --integration codex --script sh \
  --ignore-agent-tools
```

## Current initialized structure

Relevant generated paths observed:

```text
.agents/skills/speckit-*/SKILL.md
.specify/
├── integration.json
├── integrations/*.manifest.json
├── memory/constitution.md
├── scripts/bash/
├── templates/
└── workflows/
specs/<feature>/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/
└── tasks.md
```

The active feature is resolved through `.specify/feature.json` or an override;
the Git extension is optional and was not installed.

## Current Codex support

Codex is a native skills-based integration. It installs skills to
`.agents/skills` and invokes them as `$speckit-<command>`. Skills mode is the
Codex default in the evaluated source. The experiment observed ten installed
skills: constitution, specify, clarify, plan, checklist, tasks, analyze,
implement, converge, and taskstoissues.

## Recommended full flow

```text
constitution
-> specify
-> clarify
-> plan
-> checklist
-> tasks
-> analyze
-> implement
-> converge
```

- `constitution`: creates or updates governing principles and synchronizes
  dependent templates.
- `specify`: produces user stories, requirements, outcomes, assumptions, and
  non-goals from outcome-focused input.
- `clarify`: asks up to five targeted questions per pass and writes answers into
  the current spec.
- `plan`: produces implementation design and supporting artifacts.
- `checklist`: creates requirement-quality questions.
- `tasks`: creates ordered, path-specific, story-mapped work.
- `analyze`: read-only cross-artifact consistency and coverage analysis.
- `implement`: executes checked tasks.
- `converge`: compares artifacts with implementation and appends legitimate
  remaining work when needed.
- `taskstoissues`: can publish tasks to issue tracking and was deliberately not
  used because provider mutation was outside scope.

## Other current capabilities

- Integrations for many coding agents, with manifest-aware install, switch,
  status, update, and uninstall behavior.
- Extensions for new commands, integrations, and quality gates.
- Presets for overriding templates, commands, and terminology.
- Workflows with commands, shell steps, conditions, loops, fan-out/fan-in,
  gates, pause/resume, and persisted run state.
- Bundles that compose integrations, extensions, presets, steps, and workflows
  with version and provenance metadata.
- Project-local template overrides.
- Self-check and self-upgrade commands plus manifest-aware integration upgrade;
  feature specifications, plans, tasks, constitution, source, and Git history
  are documented as preserved by the upgrade path.

## Relevant boundaries

- The core artifact flow is feature-centric within one initialized project.
- `SPECIFY_INIT_DIR` can target a member project from another directory, which
  helps monorepos, but this experiment found no native equivalent of Axiom's
  logical Project spanning independent repositories with combined Execution,
  Evidence, Release, provider, and living-document lifecycle concepts.
- Workflow shell steps run with local user privileges; official documentation
  states that workflow requirements are advisory rather than a runtime
  capability sandbox. Catalog or downloaded workflows require source review.
- Extensions and presets improve reuse but add catalog, version, and template
  compatibility considerations.

## Canonical sources

- [Release](https://github.com/github/spec-kit/releases/tag/v0.16.2)
- [README at evaluated commit](https://github.com/github/spec-kit/blob/4871b485f97c7fa452ec58eba325d87536c55c34/README.md)
- [Installation](https://github.com/github/spec-kit/blob/4871b485f97c7fa452ec58eba325d87536c55c34/docs/installation.md)
- [Agentic SDD](https://github.com/github/spec-kit/blob/4871b485f97c7fa452ec58eba325d87536c55c34/docs/reference/agentic-sdd.md)
- [Integrations](https://github.com/github/spec-kit/blob/4871b485f97c7fa452ec58eba325d87536c55c34/docs/reference/integrations.md)
- [Workflows](https://github.com/github/spec-kit/blob/4871b485f97c7fa452ec58eba325d87536c55c34/docs/reference/workflows.md)
- [Upgrade guide](https://github.com/github/spec-kit/blob/4871b485f97c7fa452ec58eba325d87536c55c34/docs/upgrade.md)
