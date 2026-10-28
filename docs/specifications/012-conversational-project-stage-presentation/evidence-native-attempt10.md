# Attempt-10 — Native loading proved; replay withheld for model drift

2026-10-24 BRT (2026-10-25 UTC). Selected sanitized observations only;
no raw transcript, private paths, configuration values, account or auth data.
[Implementation reconciliation](evidence-implementation.md#attempt-10--native-loading-proved-replay-withheld-for-model-drift).

## Baseline, authority and preflight

Input local/remote/PR HEAD `3ccf3e9d6802231faa01fda2d1de87674e8caa55`;
branch `docs/axm-39-conversational-presentation`, same open Draft PR #343.
Working tree clean. Complete Linear comment inventory returned
`hasNextPage: false`; latest records were the [attempt-10 prompt](https://linear.app/rgomids/issue/AXM-39#comment-621539e2)
and [dispatch receipt](https://linear.app/rgomids/issue/AXM-39#comment-3daf7ac9).
No competing later dispatch/completion appeared at pre-edit recheck. A dispatch
receipt is not proof another executor started or completed. PR body still cites
older attempt-07 results; exact current HEAD and append-only Evidence control.
Only current Spec 012 Evidence/Tasks are changed here, as authorized.

Authority remains Maestro's delegated report of Rafael's approximately 21:13 BRT
grant, not independently authenticated human-session Evidence: ONE bounded
Claude subscription conversation, existing installed pair, loading/entry before
replay, no other model/key/cost/host, no reinstall, credentials or provider writes.
The explicit no-other-model bound is retained rather than inferred away.

Official read-only `axiom --json version`, `docs inspect`, `runtime codex status`
and `runtime claude status` exit 0: unpublished installed candidate
`0.21.3-axm39.8`, clean source
`e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6`; docs `verified/ok`, both receipts
`current`, eight skills match. Binary SHA-256 remains
`d6582d9851e7a138b8b9a91085bfede35fe20f068ee84e5fbc1b78b346dc8683`.
Official `skill inspect axiom-project` confirmed existing `--selector` show
syntax and separate mutation authority; no new syntax was adopted.
Executor's live `project list` after the conversation returned `projects: []`,
exit 0. This outside-session observation does not count as Claude list/show/validate.

## Supported configuration and unexpected model drift

Local `claude --help` and version **2.1.310** were checked without inference.
Official [skills documentation](https://code.claude.com/docs/en/skills) identifies
personal discovery under the `user` setting source; omitting it hides those
entries. Official [permissions documentation](https://code.claude.com/docs/en/permissions)
distinguishes `Read(//absolute/path)` from `/path` relative to settings, and
requires each compound shell subcommand to match independently.

One native process, outside the maintainer checkout, using streamed JSON input
and output. Loading-only first user turn; fixtures/replay deliberately withheld.
Sanitized invocation profile (path-bearing arguments refer only to the four
installed owned skill directories):

```text
claude -p --input-format stream-json --output-format stream-json --verbose
  --no-session-persistence --setting-sources user
  --settings <session-only JSON> --strict-mcp-config
  --mcp-config {"mcpServers":{}} --no-chrome
  --permission-mode dontAsk --tools Read,Bash,Skill
  --add-dir <installed axiom-project directory>
  --add-dir <installed axiom-workflow directory>
  --add-dir <installed axiom-runtime directory>
  --add-dir <installed axiom-work-item directory>
```

Session JSON disables hooks, denies edit/write/NotebookEdit/Agent/web tools,
and allows only exact four Skill names, exact absolute owned SKILL.md/public
fixture reads, four individual `shasum -a 256 <owned SKILL.md>` commands and
bounded read-only Axiom version/docs/status/list/skill-inspection commands.
No broad shell allow rule, permission bypass, plugin fabrication, copied skill
package, credential export, host settings edit or security-check disabling.
Inherited API-key/token/provider override variables were omitted by name from
the child environment; their values were not inspected or exported.

**Executor configuration mistake:** enabling `user` restored discovery but also
activated its existing model setting. Init reported **`claude-sonnet-5-5`**,
`apiKeySource: none`, `permissionMode: dontAsk`, whereas attempt-09 had actually
reported **`claude-opus-5-5`** with user settings omitted. No `--model` flag was
passed; nevertheless the effective model changed. This violates the fixed
no-other-model envelope for this attempt. The executor should have checked this
configuration interaction before starting inference. No claim of model-preserving
execution or complete authority conformance is made.

On observing init drift, executor closed further input. Already running loading
operations completed; process exit **0**, elapsed **18.11 seconds**. One user
message submitted; **zero replay/follow-up user turns**. The event collector
stopped when closing input, so final assistant summary/result event was not
captured; completed-turn verdict and final permission-denials array **UNKNOWN**.
Captured individual tool results below succeeded. No second invocation, model
switch, resume, fallback, login refresh or replay was attempted. Effort UNKNOWN.
This is a native streamed CLI surface, not witnessed terminal GUI/rendering.

## E07-Claude — Direct discovery, native entry and complete loading proof

Init lists all four skills in both `skills` and `slash_commands`. First visible
response: “All four skills ... are listed as available native skill entries.”
Four actual `Skill` calls individually requested `axiom-project`,
`axiom-workflow`, `axiom-runtime`, `axiom-work-item`; each returned
`Launching skill: <name>`, followed by native expansion of that installed body.
After removing only the runtime-added base-directory header and comparing to
the installed body after YAML frontmatter, each expansion is **exactly equal**
(disregarding terminal newline only). This is an observed native tool entry,
not slash-like ordinary prose or a pasted substitute. No separate user slash
turn/comparative replay was sent.

Four actual individual `Read` calls used offset 1, limit 2000. Removing only
number/tab display prefixes reconstructs each complete installed file **exactly**,
including frontmatter and terminal newline. This checks actual output equality,
not substring matching or extrapolation from grouped/truncated reads. Four
individual exact Bash hash requests each returned the expected digest, with
`is_error: false`; no shell suffix was appended. Per-command numeric exit code
was not present in the captured content; success is observed tool success,
not a fabricated exit code. No captured Skill/Read denial.

| Skill | Installed bytes | Session Read equality / native expansion | Session SHA-256 |
| --- | --- | --- | --- |
| Project | 34213 | exact / exact body | `775ab63f1c428156cce595cfe587039462d8e0a713b156fcce90f61c479d158d` |
| Workflow | 31401 | exact / exact body | `efdfbcd65cdffd3b3e3003e86caa8eb81f904ed9501c8f7bf22139a908154a76` |
| Runtime | 17182 | exact / exact body | `165fa61ab665ea819805b2343a00e61b24bce50a32252ee85446039f7b31636d` |
| Work Item | 36577 | exact / exact body | `ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3` |

Each installed file also byte-matches the immutable source SHA above, in BOTH
managed roots before/after. Loaded content revision is directly identified for
these reads/expansions, **within the model-drifted session only**. Permanent
model retention, semantic application and authorized-model acceptance remain
unproven. Official inspector still says `loaded: unknown`; no receipt changed.
No actual Axiom command ran inside Claude. No product-skill byte mismatch found;
therefore no guidance/product correction or new contract is justified here.

## E07-Codex and individual case ledger

No new Codex inference, session or harness expansion. Preserve [attempt-08](evidence-native-attempt08.md)
and [attempt-09](evidence-native-attempt09.md) scoped passes/complete-read proof.
Historical Claude attempt-09 FAIL/PARTIAL/prompt-only results remain immutable;
new NOT_RUN entries below do not replace or erase them.

| Case | Codex combined historical result | Claude attempt-10 qualification |
| --- | --- | --- |
| C01 | PASS_SCOPED_REPLAY | NOT_RUN; no live/synthetic question |
| C02 | PASS_SCOPED_REPLAY | NOT_RUN; absence variants withheld |
| C03 | PASS_SCOPED_REPLAY | NOT_RUN |
| C04 | PASS_SCOPED_REPLAY | NOT_RUN |
| C05 | PASS_SCOPED_REPLAY | NOT_RUN; no command/digest replay |
| C06 | PASS_SCOPED_REPLAY | NOT_RUN |
| C07 | PASS_SCOPED_REPLAY | NOT_RUN; neither Stage replayed |
| C08 | PASS_SCOPED_REPLAY | NOT_RUN; no recovery replay |
| C09 | PASS_SCOPED_REPLAY | NOT_RUN |
| C10 | PASS_SCOPED_REPLAY | NOT_RUN |
| C11 | PASS_SCOPED_REPLAY | NOT_RUN; no blocker replay |
| C12 | PASS_SCOPED_REPLAY | NOT_RUN |
| C13 | PASS_SCOPED_REPLAY | NOT_RUN |
| C14 | PASS_SCOPED_REPLAY | NOT_RUN; no version-only admission inference |
| C15 | PASS_SCOPED_REPLAY | NOT_RUN; no graph/rendering |
| C16 | PARTIAL | NOT_RUN; real entry succeeded, semantic/surface comparison withheld |

All Claude cases withheld because invocation failed the model-preserving
authority boundary. Loading success alone supplies none of their semantic
assertions. No new narrow/graphical Codex surface or independent harness
expansion was available/observed within this Claude-only grant; gaps retained.

## E08 — Aggregate, remaining gate and limitations

T01–T06 historical scoped PASS. T07 **PARTIAL**, dependent Claude replay
**TECHNICALLY_BLOCKED / MODEL_CONFIGURATION_DRIFT**. T08 **PARTIAL**.
AC1–AC5 **PARTIAL**; AC6 **PASS_SCOPED_REVIEW** for unchanged canonical
contracts/doc-only diff, with the invocation authority deviation disclosed.
No READY_FOR_INDEPENDENT_REVIEW claim. Neither Review nor green CI replaces
missing qualified native conversations or C16 surface Evidence.

Post-session official docs/status and eight source-byte comparisons passed;
installed binary, skills and receipts unchanged. No installation/receipt/version,
Project/Workflow/Stage/provider write. Source/Spec/Plan/examples/attempts 01–09
remain preserved. No merge, release, acceptance, issue closure or Done.

Next gate **RESUME_IMPLEMENT / MODEL_PRESERVING_NATIVE_CLAUDE_REQUIRED**:
verify a supported per-session discovery/read configuration that retains the
original effective model without changing host settings; any further inference
needs separate explicit bounded authority because this attempt's one invocation
was consumed. Do not silently retry. Then replay C01–C16 after loading/entry,
retain/resolve C16 surface limits, reconcile T07/T08/AC and seek independent
Review only when Evidence qualifies. No renewed-credential gate is asserted.

DF-01 **DEFERRED_NOT_IMPLEMENTED**: predecessor Specifications index summaries;
GitHub #346 remains separate. Exact final PR HEAD, final-head CI, actual checks
and single same-AXM completion callback/read-back belong to the matching
attempt-10 Linear receipt, avoiding a self-referential commit. No background
continuation integration observed; handoff **AWAITING_EXECUTION_REPORT**.
