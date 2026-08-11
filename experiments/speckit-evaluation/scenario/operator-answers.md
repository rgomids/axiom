# Frozen Operator Answers

These answers were fixed before either execution. They may be revealed only
after an approach records its own clarification questions.

1. **Lifecycle and authority:** use `Proposed -> Approved -> Deprecated -> Removed`.
   A proposed record changes no command status. A repository maintainer must
   approve transition to `Approved`; command owners execute later transitions.
2. **Location and naming:** keep records in `docs/deprecations/` using
   `NNNN-short-name.md`; maintain an index in `docs/deprecations/README.md` and a
   reusable `docs/deprecations/template.md`.
3. **Required content:** status, affected command, replacement or explicit
   absence, rationale, affected users, migration steps, notice/release target,
   rollback plan, owner, approval evidence, and relevant links.
4. **Notice and exception:** removal normally follows at least one published
   release containing the deprecation notice. An urgent security removal may
   bypass that minimum only when the record states rationale, impact, owner,
   approval, and compensating migration guidance.
5. **Enforcement:** no executable validator or CI in this slice. Provide a
   reviewer checklist and use deterministic repository checks for file presence,
   required headings, links, whitespace, and scope.
