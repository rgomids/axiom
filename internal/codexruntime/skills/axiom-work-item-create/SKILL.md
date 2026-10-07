---
name: axiom-work-item-create
description: Create or select a GitHub-backed Axiom Work Item through Lingo.
---

# Create Axiom Work Item

To inspect accepted arguments before starting this workflow, run
`axiom --json skill inspect axiom-work-item-create` and report its
`skill` payload. Inspection requests stop there: do not collect inputs or execute
any command below. The binary owns argument names, descriptions, required and
conditional inputs, and accepted forms; do not maintain a separate argument list.
For workflow invocation, preserve the guided behavior below.

Start from whatever intent the user provides, including one short problem
statement. Do not ask the user to fill a schema or repeat information already
present. Reuse supplied facts across the canonical sections. Preserve their
wording as user-authored; record paraphrases, synthesis, and inferred content
as Axiom-authored elaboration, even when the user later approves it.

Conduct a focused conversation in the user's language. First extract the
problem, desired outcome, context, scope, constraints, non-goals, and acceptance
expectations already present. Ask only for material gaps or ambiguity, using
plain questions such as "What should work differently when this is fixed?",
"Where does this happen?", or "How could we check that it is solved?". Explain
a section through its practical meaning when needed. Ask a small related group
of questions at a time and incorporate each answer before asking again.

Propose safe draft content when reasonable, identifying assumptions for review.
For example, focus scope on the stated problem; say that no additional context,
constraints, or exclusions were supplied when that is all that is known. Do not
invent environments, actors, deadlines, technical solutions, commitments, or
acceptance evidence. If an unknown would materially change delivery or safety,
ask instead of filling it with boilerplate. Derive concrete acceptance checks
from the agreed observable outcome; do not substitute "tests pass" for product
acceptance. Keep unresolved material questions out of a supposedly final draft.

Collect only missing target facts: the Project selector, Project repository key,
and explicit GitHub `owner/repository`. Do not infer the target from the Runtime
working directory. Build all seven canonical sections, useful as upstream
Specification input: explain the problem, observable outcome, relevant context,
bounded scope, preserved constraints, explicit exclusions, and verifiable
acceptance expectations. Information acquisition does not depend on body
presentation, Work Item type, label inference, or workflow-run follow-ups.

Collect or
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
never creating another Issue as a metadata repair.

Pass verbatim user facts through `--intent`, `--problem`, `--desired-outcome`,
`--context`, `--scope`, `--constraints`, `--non-goals`, and `--acceptance` as
appropriate. Pass each synthesized or inferred section through repeatable
`--elaborated-section <name>=<content>`, using canonical names `problem`,
`desired_outcome`, `context`, `scope`, `constraints`, `non_goals`, and
`acceptance_expectations`. Use one source per section; combine facts and inference
in an elaborated section rather than misattribute the combination to the user.
Keep the original short statement in `--intent`; when elaborating the problem,
include that statement verbatim in the proposed problem section too. Quote values safely;
never turn user text into executable shell syntax.

Run `axiom --json work-item create` with those facts first, without authority.
Use Lingo's returned missing questions to refine the conversation; do not ask
for known information again. Present the complete returned final draft, its
authorship and assumptions, target, effects, expected revision, and digest for
human validation before creating the external Issue. Invite corrections and
produce a fresh preview when any fact changes. Draft review is not mutation
authority; require explicit authority for the exact preview.

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
