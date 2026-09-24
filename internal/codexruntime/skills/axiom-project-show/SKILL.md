---
name: axiom-project-show
description: Inspect or resolve a configured Axiom Project through Lingo from any working directory.
---

# Show Axiom Project

Collect a Project slug or ID when absent. Run `lingo --json project show
--selector <slug-or-id>` and report its structured Project and repository
associations. Never infer a repository from Codex's current working directory.
Preserve the supplied selector exactly, ask only when it is absent or Lingo
reports it ambiguous, and never resolve identity independently. Render only the
canonical `status`, `result`, `references`, `next`, `details`, and `provenance`
returned by Lingo. Copy those canonical fields only from Lingo's top-level JSON
object. Omit canonical fields absent from that object. Never synthesize or map
setup, draft, selection, Project, Work Item, workflow, or Runtime payloads into
them.
