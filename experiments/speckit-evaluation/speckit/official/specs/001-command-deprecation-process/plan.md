# Implementation Plan: Documented Command Deprecation Process

**Branch**: `001-command-deprecation-process` | **Date**: 2026-08-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-command-deprecation-process/spec.md`

**Note**: This template is filled in by the `$speckit-plan` command; its definition describes the execution workflow.

## Summary

Add a small repository-local Markdown process for documented command deprecations. Implement one
indexed record directory, one reusable template, contributor-facing process guidance, README and
changelog lifecycle updates, and deterministic one-off validation commands. Preserve both existing
commands and descriptions. Keep three residual choices explicit: additional trigger classes,
four-digit number allocation, and evidence that a notice-bearing release was published.

## Technical Context

**Language/Version**: Markdown (CommonMark-compatible repository documentation)

**Primary Dependencies**: None; validation uses already-available local shell, Git, and text-search
commands without adding repository dependencies

**Storage**: Repository-local Markdown files

**Testing**: Deterministic local checks for file presence, required headings, known local links,
trailing whitespace, sensitive patterns, allowed file scope, and unchanged command descriptions

**Target Platform**: Offline local repository checkout

**Project Type**: Documentation-only sample repository

**Performance Goals**: Contributor can locate the template and create a structurally complete record
in 10 minutes or less

**Constraints**: No application code, executable validator, dependency, CI, provider integration,
credential, or network requirement; no command deprecation or command-description change

**Scale/Scope**: Two existing documented commands, zero current deprecations, one process page, one
template, and one empty index

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- PASS — documentation-only output; no application behavior.
- PASS — `example validate` and `example package` names and descriptions remain invariant.
- PASS — process, template, index, decisions, and validation evidence use repository-local Markdown.
- PASS — creation and review work offline without provider, credential, or network dependence.
- PASS — maintainer approval and command-owner transitions remain explicit human gates.
- PASS — validation uses deterministic local commands where practical and records manual limits.
- PASS — no executable validator, dependency, CI workflow, or operational integration is planned.
- PASS after Phase 1 — research, data model, and quickstart preserve every gate above.

## Project Structure

### Documentation (this feature)

```text
specs/001-command-deprecation-process/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── spec.md
├── checklists/
│   ├── requirements.md
│   └── deprecation-requirements.md
└── tasks.md
```

No `contracts/` directory is generated because the feature exposes no software or external-system
interface. The contributor-facing Markdown template is the durable record contract.

### Source Code (repository root)

```text
README.md
CHANGELOG.md
docs/
├── commands.md                         # unchanged
├── contributing.md                     # add process entry point
└── deprecations/
    ├── README.md                       # process, lifecycle, index, reviewer checklist
    └── template.md                     # reusable record template
```

**Structure Decision**: Keep all realized process artifacts under `docs/deprecations/`, link them
from existing contributor entry points, and keep planning or validation evidence under the official
feature directory. No source or test directory is created because no executable behavior exists.

## Complexity Tracking

No constitution violations or complexity exceptions.

## Architecture Decision Treatment

Surfaced decisions: repository-local Markdown storage, indexed individual records, explicit
four-stage human-controlled lifecycle, and one-off deterministic validation instead of automation.
No ADR is warranted: this slice adds no runtime architecture, external interface, dependency,
provider boundary, or difficult-to-reverse technical choice. Decisions and alternatives are fully
captured in `spec.md` and `research.md`.
