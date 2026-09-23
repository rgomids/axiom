# Contribution Policy

## Core rule

The Axiom repository agent is a contributor and MUST follow the repository
contribution workflow.

The canonical workflow lives in [CONTRIBUTING.md](../../CONTRIBUTING.md). The
agent must also use the real [Pull Request template](../../.github/PULL_REQUEST_TEMPLATE.md),
follow [documentation governance](../../docs/documentation.md), and apply the
[security policy](security.md). This policy defines agent behavior; it does not
duplicate those sources.

## Required behavior

For every contribution, the agent must:

- use the current branch and commit conventions;
- keep one small, coherent scope and exclude unrelated changes;
- respect Specifications, ADRs, explicit authority, and Task/Slice boundaries;
- run applicable repository validations and record reproducible Evidence from
  commands actually executed;
- reconcile affected documentation;
- assess security impact and state known limitations;
- preserve human approval boundaries.

Before opening a Pull Request, inspect the branch, complete diff, commits, PR
title and description, related references, Evidence, documentation impact,
security impact, limitations, and absence of out-of-scope changes.

When opening a Pull Request, use the repository template and replace every
placeholder with truthful content or an objective not-applicable explanation.

## Prohibited behavior

The agent must not:

- push directly to `main`;
- open an empty or context-free Pull Request;
- use generic final commit messages such as `update`, `changes`, `fix`, or
  `wip`;
- include unrelated changes;
- treat green CI as sufficient Evidence;
- treat technical merge as human acceptance;
- advance to another Task or Slice without explicit authority;
- merge its own Pull Request without explicit human authorization.

Future changes to the canonical contribution sources apply automatically. When
this policy conflicts with them, stop and reconcile the conflict instead of
inventing a parallel workflow.
