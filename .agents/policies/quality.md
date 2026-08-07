# Quality Policy

## Evidence hierarchy

Prefer:

1. executable tests/checks;
2. static analysis;
3. reproducible commands;
4. concrete diff inspection;
5. reasoned judgment.

## Review expectations

Review proportional to risk:

- behavior against acceptance criteria;
- test adequacy;
- regression risk;
- dependency direction;
- unnecessary complexity;
- concurrency/error handling where relevant;
- observability;
- operational impact;
- documentation consistency.

Avoid universal style dogma such as arbitrary function line limits.

## Change discipline

- Keep changes scoped.
- Avoid opportunistic refactors unless they directly reduce implementation risk.
- Preserve existing behavior unless change is intentional.
- When modifying legacy areas, prefer minimal, reversible changes.
