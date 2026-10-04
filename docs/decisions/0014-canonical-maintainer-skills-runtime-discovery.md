# ADR-0014 — Canonical maintainer skills with runtime-native discovery

## Status

Accepted on 2026-10-03 by the human decision recorded in
[Issue #174](https://github.com/rgomids/axiom/issues/174#issuecomment-5975167793).
Upstream adoption of its implementation remains subject to PR review.

This ADR supersedes only the repository bootstrap-policy fragment that
rejected every `.claude/` entry, materialized in `AGENTS.md` ("Maintainer
runtimes") and `scripts/check-claude-bootstrap.sh`. No earlier ADR recorded
that prohibition, so no ADR is superseded. ADR-0003 remains in force and
compatible.

## Context

Codex and Claude Code are both maintainer runtimes of this repository.
`AGENTS.md` is the single canonical maintainer policy and the root
`CLAUDE.md` contains only `@AGENTS.md`.

Maintainer skills live in `.agents/skills/<skill>/SKILL.md`. Codex scans
`.agents/skills` from the working directory up to the repository root, so
`$axiom-release` is invocable natively. Claude Code discovers project skills
only under `.claude/skills/<skill>/SKILL.md`, so a Claude maintainer could
read a skill when `AGENTS.md` routed to it but could not invoke
`/axiom-release`. The validators rejected any `.claude/` entry, and Issue #174
required that native Claude discovery be decided explicitly rather than
worked around.

Verified on 2026-10-03 from official documentation:

- Claude Code (`code.claude.com/docs/en/skills`): a project skill entry under
  `.claude/skills/` may be a symlink to a directory elsewhere on disk; Claude
  Code reads `SKILL.md` from the target and loads it once even when several
  locations point at the same target.
- Codex (`learn.chatgpt.com/docs/build-skills`): repository skills are read
  from `.agents/skills` in every directory from the working directory to the
  repository root; symlinked skill folders are followed.

Executable evidence (Claude Code 2.1.286, 2026-10-03): with
`.claude/skills/axiom-release -> ../../.agents/skills/axiom-release`, the
headless init event lists `axiom-release` in `skills` and `slash_commands`,
and `/axiom-release` expands the canonical instructions.

## Decision

1. `.agents/skills/<skill>/SKILL.md` is the canonical, runtime-neutral
   definition of each Axiom maintainer skill. It is not a Codex-owned
   directory, although Codex discovers it natively. Each skill has exactly one
   maintained implementation.
2. A runtime may expose canonical skills only through a thin discovery
   adapter that adds no instructions. For Claude Code the adapter is
   `.claude/skills/<skill>`, a relative symlink whose target is exactly
   `../../.agents/skills/<skill>`. No `SKILL.md` is copied.
3. Every canonical maintainer skill has a Claude adapter. Both entry paths are
   three directories below the repository root, so relative links inside a
   skill resolve to the same files from either path.
4. `CLAUDE.md` stays a regular file whose entire content is `@AGENTS.md`.
   `.claude/skills/` is the only approved exception under `.claude/`, for
   runtime-native skill discovery only. It does not authorize
   `.claude/settings*`, agents, commands, hooks, Claude-specific policy or
   prompts, copied skills, or any other `.claude/` content.
5. The validators enforce a closed allowlist in repository mode
   (`check-claude-bootstrap.sh --skill-adapters`): `.claude/` and
   `.claude/skills/` are real directories containing only adapters; each
   adapter is a symlink with the exact target above, named after an existing
   canonical skill whose directory, `.agents/` and `.agents/skills/` are real
   (non-symlink) directories inside the repository, whose `SKILL.md` is a
   regular file declaring the same `name`; every canonical skill has an
   adapter; no other `SKILL.md` declares a canonical maintainer skill name.
   Anything else fails.
6. Generated Codex packages are unchanged: the Agent Factory's Codex renderer
   emits neither `.claude/` nor symlinks, and the package validator rejects
   both by default. Maintainer runtime support is not Agent Factory renderer
   support; a Claude renderer needs its own Specification and validation.

## Alternatives considered

- **Keep the prohibition (status quo).** No new surface, but Claude
  maintainers cannot invoke skills natively and parity stays asymmetric.
- **Copy each `SKILL.md` into `.claude/skills/`.** Works on every platform,
  but creates two maintained implementations and drift. Rejected.
- **Generated copies checked by a sync validator.** Avoids symlinks but still
  duplicates content in the tree and adds a generation step. Rejected while
  symlinks are supported by both runtimes.
- **Move canonical skills to `.claude/skills/` and symlink Codex.** Makes a
  vendor path canonical and inverts runtime neutrality. Rejected.
- **Claude plugin or user-level installation.** Moves maintainer behavior
  outside the repository and its review. Rejected.

## Consequences

- `/axiom-release` (Claude) and `$axiom-release` (Codex) resolve to the same
  file; editing a canonical skill changes both runtimes at once.
- Symlinks become a reviewed repository surface. They are restricted to
  exact relative targets inside `.agents/skills/`, so a link cannot point
  outside the repository or to an arbitrary file.
- Adding a canonical skill requires adding its adapter; the validator fails
  otherwise.
- On checkouts without symlink support (for example Git for Windows with
  `core.symlinks=false`), adapters materialize as small text files and Claude
  Code native discovery does not work there; the canonical skills and
  `AGENTS.md` routing still do. Repository validation runs on Linux and
  macOS, as before.
- Any further runtime-specific structure still requires a recorded decision.

## Revisit when

- either runtime changes or drops symlinked skill discovery;
- Claude Code supports a configurable project skill root or `.agents/skills`;
- a third maintainer runtime needs an adapter;
- a Claude Agent Factory renderer is specified.
