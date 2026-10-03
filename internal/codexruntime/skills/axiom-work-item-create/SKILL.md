---
name: axiom-work-item-create
description: Create or select a GitHub-backed Axiom Work Item through Lingo.
---

# Create Axiom Work Item

Collect the Project selector, Project repository key, explicit GitHub
`owner/repository`, and these structured sections: problem, desired outcome,
context, scope, constraints, non-goals, and acceptance expectations. Collect or
propose a Work Item type using delivery intent: `story` delivers a concrete
user/product benefit, `bug` corrects observed faulty behavior, and `task` covers
technical or operational activity. Supply the proposed type with `--type`.
Do not silently turn implementation activity into a story. For a story, collect
`--beneficiary` (who benefits) and `--value` (what concrete benefit becomes
possible), keeping implementation details in scope. Ask for missing value or
propose task when the conversation only describes implementation activity.

Use repeatable `--classification` for explicitly requested provider
classifications (GitHub label names), preserving them as user intent. Otherwise
let Lingo infer a type label from the provider's existing catalog. Report
`draft.providerDocument.metadata`, including the proposed labels, and notices
alongside the type and story value in the complete final draft before authority.
Unsupported explicit classifications must be corrected, never silently omitted;
missing inferred labels are an explicit preview notice. Report partial results
for unapplied or unverified metadata, keeping the existing Issue reference and
never creating another Issue as a metadata repair. Run
`axiom --json work-item create` with those facts first, without authority. Report
the returned draft, target, effects, expected revision, and digest for review.

Only after the human grants authority for that exact preview, repeat the same
facts with `--preview-digest <digest> --authorize-external`. Changed facts require
a new preview. For an existing linked or candidate Issue, use exact selector
`github:<owner>/<repository>#<number>`. Run `axiom --json work-item select
--project <uuid-or-slug> --repository <project-scoped-key> --work-item
<exact-selector>` first; after review, repeat with its digest and
`--authorize-local`.
Never call GitHub directly, infer a Git remote, retry an ambiguous create blindly,
or treat linkage as human acceptance. Ask only for missing or Lingo-reported
ambiguous inputs. Do not parse identity, classify status, inspect persistence, or
grant authority. Preserve transported user text as user-authored.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `draft`, `selection`, `workItem`

Copy canonical completion fields only from Lingo's top-level JSON object. Omit
canonical fields absent from that object. Never derive, synthesize, or reinterpret
a canonical field from `draft`, `selection`, `workItem`, or another
operation-specific payload. Preserve and report those returned payloads separately
according to their original semantics. In particular, the create preview must
retain the complete `draft`, its `target`, `effects`, expected revision, preview
digest, and every other review/authorization fact returned by Lingo. Never
reinterpret `draft`, `target`, `effects`, `selection`, or `workItem` as `details`
or another canonical field. Keep every nested field in its original Lingo JSON
position; do not extract, duplicate, rename, or relocate payload fields.
