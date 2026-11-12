# #388 — Missing exact Workflow reference

Authorized bounded bug fix, 2026-11-11. Implements the existing Specification
007 exact-reference contract; no selection fallback, new flags, persistence or ADR.

## Specification

For `project workflow show`, `select`, and `remove`, reject absent workflow key,
revision (zero/unprovided), or digest before reference lookup. Report
`missing_required_input` with the first absent flag in workflow/revision/digest
order, using the existing `missing_required_input:--<flag>` details vocabulary.
Point humans to `project workflow list` / `project show` for the exact reference.
A complete unknown reference continues to return `workflow_not_found`. Existing
preview/apply authority and JSON/exit conventions remain intact.

## Plan and tasks

1. Validate missing reference fields in the shared Project application path;
   retain reference validity and lookup semantics for complete inputs.
2. Present the application refusal through the existing completion/diagnostic
   adapters, naming the missing flag and discovery commands.
3. Complete the shipped skill operation example and discovery metadata examples;
   preserve historical skill snapshots and their hashes.
4. Test each missing field, all missing, unknown and successful exact references
   for all three operations; check human/JSON diagnostics and documentation parity.
5. Run required quality/security checks, review the connected paths, record
   results in [Evidence](evidence-388-missing-reference.md), deliver one open PR.

No material architecture/security/contract decision is pending. No vendor
inference or real Axiom state access is needed.
