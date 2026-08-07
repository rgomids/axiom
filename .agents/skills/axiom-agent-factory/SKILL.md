---
name: axiom-agent-factory
description: Design, render, validate, and package Codex agent harnesses using Axiom's emerging agent model.
---

# Axiom Agent Factory

## Purpose

This skill is both useful functionality and dogfood for the future Axiom product.

Use it to create another agent/harness from a project need.

The current supported native target is:

```text
codex
```

Other targets may be represented in a vendor-neutral blueprint but must not be claimed as operational renderers until implemented and validated.

## Workflow

### 1. Gather context

Read supplied project files before asking questions.

Resolve:

- project/agent name;
- objective;
- target;
- repository state;
- stack;
- architecture;
- validation commands;
- constraints;
- workflows;
- skills;
- specialist roles;
- providers/integrations;
- risk/approval requirements;
- definition of done;
- package scope.

Ask only for unknowns that materially change artifacts.

### 2. Normalize blueprint

Start from:

`.agents/templates/agent-blueprint.yaml`

Do not write target-specific file paths into vendor-neutral sections.

### 3. Create artifact plan

Every planned file must have:

- path;
- responsibility;
- why it is needed;
- source blueprint concepts.

Remove files without clear responsibility.

### 4. Render Codex

Read:

`references/codex-renderer.md`

Minimum native package:

```text
AGENTS.md
.agents/skills/<skill>/SKILL.md   # only when needed
```

Additional artifacts may be added only with clear purpose.

### 5. Validate

Check:

- target isolation;
- no secrets;
- no empty directories;
- no vague placeholders;
- no duplicated policy blocks;
- `AGENTS.md` remains a router rather than a monolith;
- workflows are represented as skills;
- package matches blueprint;
- assumptions are visible;
- validation commands are real or explicitly unresolved.

Use `scripts/validate-agent-package.sh <package-dir>` when applicable.

### 6. Package

Create a ZIP whose contents are intended to be extracted into the repository root.

Recommended name:

```text
<project>-codex-agent.zip
```

Do not include:

- `.git/`;
- `.env`;
- private keys;
- caches;
- build outputs;
- node_modules/vendor;
- unrelated user files;
- internal reasoning/scratchpads.

## Design heuristic

Prefer, in order:

1. short global instruction;
2. skill;
3. specialist subagent only when isolation/expertise is genuinely useful;
4. deterministic script;
5. external provider/tool only when needed.

Avoid "do everything" skills.

## Completion

Return:

- package path;
- exact tree;
- assumptions;
- validation status;
- unresolved decisions.
