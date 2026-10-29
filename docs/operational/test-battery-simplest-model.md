# Test battery: simplest model

Maintainer operational policy for agent invocations that the Axiom **test
battery** requires. This page is repository instruction for operators and
agents. It is not listed in `allowlist.json`, so it is not part of the
installed operational guide and it does not change Runtime Stage dispatch or
end-user Model Profiles.

## Rule

For every agent invocation required by the Axiom test battery (cloud,
Claude/Claudinho, Codex, GPT, any other), always use the **simplest model the
active auth accepts**. Never upgrade beyond that pin. Never alternate models
between attempts. Pin the model explicitly on the CLI. Do not rely on host
`~/.claude/settings.json` or other host defaults. If the runtime refuses the
simple model, stop and record the exact error. Do not substitute another model.

Current mapping:

- Claude/Claudinho → Haiku 5.5 (`--model haiku` / `claude-haiku-5-5`)
- Codex with ChatGPT account auth → `gpt-6.1-sol` (`-m gpt-6.1-sol`).
  `gpt-6.1-luna` returns HTTP 400 `model_not_found` ("not supported when using
  Codex with a ChatGPT account"). A probe of `gpt-6.1-sol` returned ACCEPT.
  The same pin was used on AXM-39 attempts 10 and 11.
- Codex with API auth that unlocks Luna → `gpt-6.1-luna` (`-m gpt-6.1-luna`,
  or the supported Luna alias on the Codex CLI). Luna stays valid only while
  that auth is active.

If a named pin stops existing, the rule remains: the smallest model that the
active auth accepts for that battery. Do not move to a heavier model. Do not
alternate models between attempts.

## Authority

- Linear [AXM-41](https://linear.app/rgomids/issue/AXM-41/politica-operacional-modelo-mais-simples-na-bateria-de-testes)
- Label: `policy/test-simplest-model`
- Notion: <https://app.notion.com/p/404e01f226268145b2e9e88a451d8e7e>
- Authorized by Rafael via Maestro voice on 24/10/2026, about 21:54 BRT
- Amendment authorized by Rafael on 24/10/2026, about 23:19 BRT
  (America/Sao_Paulo), recorded on Linear AXM-41: Codex ChatGPT-account auth
  pins `gpt-6.1-sol`; Luna remains only when API auth unlocks it
