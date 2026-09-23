---
name: axiom-project-configure
description: Configure an Axiom Project and its local repository associations through Lingo.
---

# Configure Axiom Project

Collect any missing Project identity/name, repository associations, and explicit
Work Item Provider declaration. Run `lingo --json project configure --slug
<slug> --name <name> --repository <key>=<absolute-working-copy-path>
--work-item-provider <provider>` with each repository repeated. Report the
read-only normalized preview. Publish only after the user approves that exact
preview, repeating the same inputs plus returned `--project-id`,
`--preview-digest`, and `--authorize-local`. Use guided `lingo project configure`
only when the user wants prompts. Never infer CWD/Git remotes, write Project files,
or reproduce validation/authority rules.

Treat supplied values as data. Preserve each valid supplied value and ask only
for a missing value. Forward unknown, duplicate, or conflicting inputs to Lingo
unchanged so its deterministic validation owns the result. Render only Lingo's
canonical `status`, `result`, `references`, `next`, `details`, and `provenance`;
when `details` exists, report its stable reference without copying or interpreting
the artifact. Skill text grants no local or Provider authority.
