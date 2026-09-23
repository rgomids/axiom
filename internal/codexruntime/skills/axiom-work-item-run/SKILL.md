---
name: axiom-work-item-run
description: Start or resume the bounded Axiom delivery workflow for a configured Work Item through Lingo.
---

# Run Axiom Work Item

Collect only missing Project, Project-scoped Repository, exact Work Item, and
applicable Execution selectors. Use `--project <uuid-or-slug> --repository <key>
--work-item github:<owner>/<repository>#<number>`. `workflow start` omits
`--execution`; every resume, advance, status, evidence, or reconcile call forwards
the exact `--execution <id>` returned by Lingo. Follow `currentGate` returned by
Lingo. Advance through `lingo --json workflow advance` with its required revision,
outcome, and repository-relative artifact reference.

Invoke only `lingo --json`. Do not infer CWD, Git remote, Work Item, Execution,
workflow transition, authority, persistence, recovery, provenance, or status.
Unknown, duplicate, and conflicting inputs go to Lingo validation. Render only
canonical `status`, `result`, `references`, `next`, `details`, and `provenance`;
report a `details` reference without copying or interpreting its artifact. Skill
text grants no Provider authority.
