# Issue #306 — Local Runtime and Model Profile management Plan

## Status and authority

**Proposed; Planning in progress.** Only Planning is authorized. Tasks,
production implementation, vendor probes, merge, release, Issue closure and
product acceptance remain separately gated.

Technical owner: [Issue #306](https://github.com/rgomids/axiom/issues/306).
One branch, `feat/306-local-runtime-profiles`, and one Draft PR serve the entire
lifecycle. Human approval must identify the exact commit and artifact blobs;
material changes invalidate the affected approval.

## Investigation baseline

Main inspected on 2026-10-11: `5d52327b0367ba08d872b50e804df5c7d63a3e89`
(v0.16.0). Working tree was clean and detached. No existing #306 branch or
owning open PR was found before creating this branch from current main.

Reuse `internal/runtimeprofile`, `internal/runtimeapplication`, the protected
`internal/local/runtime_profile_store.go` and #231 Project readiness. Existing
local storage and selection do not constitute supported production authoring.
Investigation covers versioned contracts, exact preview/apply, credential
references, capability truth, persistence/recovery and conversational routing.

This initial revision records the Planning transition, not a complete Plan or
approval. The complete Plan, proposed contract amendment and mandatory security
review will follow in the same PR before human review.
