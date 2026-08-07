---
name: axiom-sdd
description: Run Axiom changes using a lightweight Spec-Driven Development flow from intent through validated implementation.
---

# Axiom SDD

## Goal

Prevent prompt-driven implementation without a stable statement of desired behavior.

## Flow

```text
intake
→ specify
→ clarify
→ plan
→ tasks
→ implement
→ review
→ reconcile
```

Not every change requires every phase as a separate file.

## Intake

Capture:

- problem;
- desired outcome;
- users/actors;
- constraints;
- known evidence;
- explicit non-goals.

## Specify

Define behavior without prematurely prescribing implementation.

Include as applicable:

- scenarios;
- functional requirements;
- acceptance criteria;
- invariants;
- errors/edge cases;
- non-functional constraints.

## Clarify

Ask only questions whose answer materially changes:

- behavior;
- architecture;
- security;
- data ownership;
- compatibility;
- rollout;
- scope.

Prefer a reasoned default when the choice is reversible and low-risk.

## Plan

Map the specification to:

- affected repositories/modules;
- contracts;
- data changes;
- architecture;
- observability;
- tests;
- rollout/rollback;
- migrations;
- documentation.

Identify ADR candidates explicitly.

## Tasks

Split only when it improves execution, validation, ownership, or rollback.

A task should have:

- objective;
- scope;
- dependencies;
- acceptance/evidence;
- repository impact.

## Implementation

Use `axiom-implement`.

## Review

Use `axiom-review`.

## Reconcile

Use `axiom-document`.

## Rule

Never silently change the specification to match an implementation shortcut.
