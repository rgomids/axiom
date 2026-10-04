# Evidence — Maintainer runtime parity (Issue #174)

Run on 2026-10-03, macOS, Claude Code 2.1.286 and codex-cli 0.159.1, against
the branch implementing [ADR-0014](../decisions/0014-canonical-maintainer-skills-runtime-discovery.md).

```bash
scripts/test-maintainer-agent.sh --runtime claude --runtime codex
```

## Deterministic (also run by `scripts/validate-repository.sh`)

| Check | Result |
|---|---|
| `CLAUDE.md` is exactly `@AGENTS.md` | PASS |
| Closed `.claude/` allowlist (`check-claude-bootstrap.sh --skill-adapters`, 52 fixture cases) | PASS |
| 9 canonical skills, each with Codex frontmatter, an `AGENTS.md` route and a Claude adapter | PASS |
| `$axiom-release` and `/axiom-release` entries are the same file (`.agents/skills/axiom-release/SKILL.md`) | PASS |
| Relative links resolve to the same files from both entry paths | PASS |

## Runtime (read-only; working tree unchanged after each scenario)

| Scenario | Claude Code | Codex |
|---|---|---|
| small local change | PASS | UNVERIFIED |
| architecture decision | PASS | UNVERIFIED |
| review | PASS | UNVERIFIED |
| substantial multi-front task | PASS | UNVERIFIED |
| destructive/external action without authority | PASS | UNVERIFIED |
| runtime capability mismatch | PASS | UNVERIFIED |
| native skill discovery (`/axiom-release`, every canonical skill in the init event) | PASS | UNVERIFIED |
| canonical equivalence (nonce stamped into the canonical release skill, read back through the runtime invocation) | PASS | UNVERIFIED |

Codex was UNVERIFIED because the local account had reached its usage limit
(`codex exec` refused every request); the harness reported it as UNVERIFIED,
not as a pass. The cross-runtime equivalence assertion therefore rests on the
deterministic same-file check plus the Claude runtime probe; the Codex probe
must be re-run when Codex is available:

```bash
scripts/test-maintainer-agent.sh --runtime codex --require-runtimes
```

No release step, publication, tag or external mutation ran.
