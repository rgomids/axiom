# Attempt-11 — Haiku 5.5 model gate unverified; replay withheld

2026-10-24 BRT (2026-10-25 UTC). Sanitized operational observations only.
Input local/remote/PR HEAD `9c213665bf258aeb3f8ccf41fe8a83aca5ee9fb3`;
clean worktree, same open Draft PR #343 and branch
`docs/axm-39-conversational-presentation`. Linear inventory: 60 comments,
final page `hasNextPage: false`; attempt-11 prompt/dispatch present, no matching
attempt-11 completion receipt. Dispatch acceptance does not prove execution.

## Authority and unchanged installed state

User authorizes only Claude Haiku 5.5, requested `claude-haiku-5-5`, with positive
effective-model metadata before inference; no fallback or blind retry.
No delegated agent. No new Codex inference. No host settings, credentials,
installations, receipts, versions or permissions changed to overcome refusal.

Fresh official read-only `axiom --json version`, `docs inspect`,
`runtime claude status` and `runtime codex status` all exit 0. Candidate remains
`0.21.3-axm39.8`, clean source `e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6`;
binary SHA-256 `d6582d9851e7a138b8b9a91085bfede35fe20f068ee84e5fbc1b78b346dc8683`.
Docs verified/ok; both receipts current; eight skills equivalent, observed files
match. Manifest `3471519797b11ddef021d1dea536538137f8747d9faaf2f96c110cf7d86fab50`.
These are inspector observations, not new direct source-byte comparisons or
session loading proof. Inspector still reports loaded unknown. Attempt-10's
native Skill/Read/hash evidence belongs only to its earlier drifted session.

## E07-Claude — Exact pre-inference gate result

Installed CLI help/version checked: **Claude Code 2.1.310**.
[Official model configuration](https://code.claude.com/docs/en/model-config)
supports full model IDs; an explicit flag alone is not resolved identity.
Python 3 standard-library subprocess/select collector launched ONE process
outside the maintainer checkout, in the system temporary directory. Exact flags:

```text
claude -p --model claude-haiku-5-5
  --input-format stream-json --output-format stream-json --verbose
  --no-session-persistence --setting-sources user
  --settings '{"disableAllHooks":true}' --strict-mcp-config
  --mcp-config '{"mcpServers":{}}' --no-chrome
  --permission-mode dontAsk --tools Read,Skill
```

No fallback flag. No prompt argument. Stdin held open with **zero bytes/user
messages submitted**. Child environment omitted inherited `ANTHROPIC_*`,
`CLAUDE_CODE_*`, `CLAUDE_CONFIG_DIR`, `CLAUDE_MODEL`, `CLAUDECODE` overrides by
name, without inspecting/exporting their values; existing user configuration
remained untouched. Hooks disabled only for this process, MCP empty, no write,
shell or agent tools exposed. No settings/permission bypass or auth operation.

Bound: 15 seconds waiting for machine-readable output; after no record arrived,
collector requested process termination and drained both streams. Observed:

```json
{
  "requested_model": "claude-haiku-5-5",
  "resolved_model": null,
  "system_init_records": 0,
  "user_messages_sent": 0,
  "stdout": "",
  "stderr": "",
  "exit_code": 0,
  "elapsed_seconds": 15.43
}
```

**No CLI/provider refusal or error text was emitted.** This is an unverified
model gate, not a fabricated provider rejection, unavailable-model finding or
authentication verdict. Exit 0 and requested ID prove neither resolved identity
nor availability. No user inference was requested; no assistant/tool response
observed. No claim about unobserved provider internals. No second launch,
alternate alias/model, resume, control-protocol experiment or retry followed.
Gate failed **before native entry/loading and replay** because no supported
machine-readable effective identity was observed without submitting inference.

## E07 — Case ledger and provenance

| Case | Historical Codex result | Claude attempt-11 |
| --- | --- | --- |
| C01 | PASS_SCOPED_REPLAY | NOT_RUN |
| C02 | PASS_SCOPED_REPLAY | NOT_RUN |
| C03 | PASS_SCOPED_REPLAY | NOT_RUN |
| C04 | PASS_SCOPED_REPLAY | NOT_RUN |
| C05 | PASS_SCOPED_REPLAY | NOT_RUN |
| C06 | PASS_SCOPED_REPLAY | NOT_RUN |
| C07 | PASS_SCOPED_REPLAY | NOT_RUN |
| C08 | PASS_SCOPED_REPLAY | NOT_RUN |
| C09 | PASS_SCOPED_REPLAY | NOT_RUN |
| C10 | PASS_SCOPED_REPLAY | NOT_RUN |
| C11 | PASS_SCOPED_REPLAY | NOT_RUN |
| C12 | PASS_SCOPED_REPLAY | NOT_RUN |
| C13 | PASS_SCOPED_REPLAY | NOT_RUN |
| C14 | PASS_SCOPED_REPLAY | NOT_RUN |
| C15 | PASS_SCOPED_REPLAY | NOT_RUN |
| C16 | PARTIAL | NOT_RUN |

Every attempt-11 case withheld at the same model gate. No new native/synthetic
replay, selectors/digests, intake/implementation views, blocked recovery or
graphical/narrow surface evidence. Codex history and C16 independent-harness
gap preserved; no current-session loading inferred from installed files.

## E08 — Reconciliation and technical decisions

Attempt outcome **BLOCKED** (`stage.blocked`); overall Implement **PARTIAL**.
T01–T06 historical scoped PASS; T07/T08 **PARTIAL**. AC1–AC5 **PARTIAL**;
AC6 **PASS_SCOPED_REVIEW**, unchanged canonical contracts and evidence-only scope.
E07-Claude NOT_RUN for qualified native behavior; E08 partial reconciliation.
No independent Review qualification or AXM-12 E2E/acceptance claim.

technical_decisions[] — TECHNICAL_LOCAL; executor-ai; scope impact none:

- IMPL11-TD-01: exact full model ID rather than mutable `haiku` alias, user
  source retained for installed discovery, zero user input before metadata.
  Evidence: invocation/result above and installed CLI help. Flag does not prove
  resolution; stop rather than infer precedence or submit a loading prompt.
- IMPL11-TD-02: bounded Python 3 standard-library collector, then terminate/drain
  after 15 seconds without metadata; no new SDK/dependency, unsupported control
  request or second invocation. Reversible process-only choice; model remained
  unverified and grant was not silently reused.
- IMPL11-TD-03: append Evidence and prepend checkpoint, preserve all earlier
  artifacts; no product fix justified by absent metadata. Scope review checks
  only these documentation changes; historical PASS is not a new test result.

Next gate **HAIKU_5_5_MODEL_REFUSED_OR_UNVERIFIED**. Remaining work: establish a
supported pre-inference effective-model verification path under renewed bounded
authority; prove native entry/loading in that authorized session; replay C01–C16;
resolve or retain C16 gaps; reconcile T07/T08/AC/E07/E08 and seek Review only
after qualification. No automatic retry or background monitor.

DF-01 **DEFERRED_NOT_IMPLEMENTED**; GitHub #346 separate. No domain/CLI/Execution,
registry, host or auth changes; no merge, release, self-approval, human acceptance
or Done. Final SHA, actual validators/security/CI and single root Linear comment
with raw marker/JSON read-back are recorded in the matching attempt-11 callback,
avoiding a self-referential commit. Handoff **AWAITING_EXECUTION_REPORT**.
