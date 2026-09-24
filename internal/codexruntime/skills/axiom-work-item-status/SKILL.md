---
name: axiom-work-item-status
description: Inspect Axiom Work Item workflow status and Evidence through Lingo.
---

# Show Axiom Work Item Status

Collect only missing Project, Project-scoped Repository, exact Work Item, and
Execution selectors. Run `lingo --json workflow status` and, when requested,
`lingo --json workflow evidence` with `--project <uuid-or-slug> --repository
<key> --work-item github:<owner>/<repository>#<number> --execution <id>`.

Never infer selectors from CWD, Git, Provider, Runtime chat, or global discovery.
Never classify workflow state independently. Forward unknown, duplicate, or
conflicting input to Lingo validation. Render only canonical `status`, `result`,
`references`, `next`, `details`, and `provenance`; report a `details` reference
without copying or interpreting its artifact. Copy those canonical fields only
from Lingo's top-level JSON object. Omit canonical fields absent from that object.
Never synthesize or map setup, draft, selection, Project, Work Item, workflow, or
Runtime payloads into them. This skill performs no mutation and grants no
Provider authority.
