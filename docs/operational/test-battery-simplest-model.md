# Test battery: simplest model

Maintainer operational policy for agent invocations that the Axiom **test
battery** requires. This page is repository instruction for operators and
agents. It is not listed in `allowlist.json`, so it is not part of the
installed operational guide and it does not change Runtime Stage dispatch or
end-user Model Profiles.

## Rule

For every agent invocation required by the Axiom test battery (cloud,
Claude/Claudinho, Codex, GPT, any other), always use the **simplest available
model**.

Current mapping:

- Claude/Claudinho → Haiku 5.5 (`--model haiku` / `claude-haiku-5-5`)
- Codex → GPT 6.1 Luna (`-m gpt-6.1-luna` or the supported Luna alias on the Codex CLI)

If Haiku or GPT 6.1 Luna stop existing, the rule remains: always the smallest
possible model for that battery. Never upgrade to a heavier model (no
Sonnet, Opus, or sol as fallback). Never alternate models between attempts.
Pin the model explicitly on the CLI. Do not rely on host
`~/.claude/settings.json` or other host defaults. If the runtime refuses the
simple model, stop and record the exact error. Do not substitute another model.

## Authority

- Linear [AXM-41](https://linear.app/rgomids/issue/AXM-41/politica-operacional-modelo-mais-simples-na-bateria-de-testes)
- Label: `policy/test-simplest-model`
- Notion: <https://app.notion.com/p/404e01f226268145b2e9e88a451d8e7e>
- Authorized by Rafael via Maestro voice on 24/10/2026, about 21:54 BRT
