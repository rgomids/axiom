---
name: axiom-work-item-create
description: Create or select a GitHub-backed Axiom Work Item through Lingo.
---

# Create Axiom Work Item

Collect the Project selector, Project repository key, explicit GitHub
`owner/repository`, and these structured sections: problem, desired outcome,
context, scope, constraints, non-goals, and acceptance expectations. Run
`lingo --json work-item create` with those facts first, without authority. Report
the returned draft, target, effects, expected revision, and digest for review.

Only after the human grants authority for that exact preview, repeat the same
facts with `--preview-digest <digest> --authorize-external`. Changed facts require
a new preview. For an existing Issue, run `lingo --json work-item select --project
<selector> --repository <key> --provider-repository <owner/repository> --number
<number>` first; after review, repeat with its digest and `--authorize-local`.
Report the canonical structured result. Never call GitHub directly, infer a Git
remote, retry an ambiguous create blindly, or treat linkage as human acceptance.
