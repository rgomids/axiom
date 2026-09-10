# Frozen Specification — Record Creation

## Scope

Add record creation and retrieval behavior to the controlled fixture. This is
experimental input, not an Axiom product specification.

## Functional requirements

- **FR-001 — Caller-owned identifier:** The caller MUST provide an identifier
  between 8 and 32 ASCII alphanumeric characters. The system MUST preserve that
  identifier exactly.
- **FR-002 — Conflict safety:** Creating a record with an existing identifier
  MUST fail with a conflict result and MUST NOT change the existing record.
- **FR-003 — Retrieval:** A successfully created record MUST be retrievable by
  the same identifier and return the stored payload.
- **FR-004 — Rollback:** A failed release MUST support restoring the previously
  deployed behavior without losing records accepted before rollback.

## Acceptance criteria

- **AC-001:** Given identifier `ORDER2026`, creation and retrieval return
  `ORDER2026` unchanged with the original payload.
- **AC-002:** Given an existing `ORDER2026`, a second creation returns conflict
  and the original payload remains retrievable.
- **AC-003:** Record performance should be good.

## Non-goals

- Authentication, authorization, tenancy, UI, provider integration, and
  production deployment are outside this fixture.
