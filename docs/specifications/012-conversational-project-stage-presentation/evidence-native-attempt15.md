# Attempt-15 — Integrated presentation corrections; qualification remains partial

2026-10-25, macOS arm64. Same open Draft [PR #343](https://github.com/rgomids/axiom/pull/343).
Initial local/remote/PR HEAD `1a641c4f77340ad13539408a644449ee5d4e11be`, clean tree.
Source correction commit `aff1accef3a0a5ada730d350409fc7153d4c06a0`.
All 74 Linear comments inventoried through `hasNextPage: false`: attempt-15 prompt
and dispatch present, no previous attempt-15 completion. Other observed Claude
processes had different working directories; only this executor observed in this
worktree. No subagents: available capability ordering did not justify delegation.

## Baseline and actual scope

[Specification r1](spec.md), [Plan r1](plan.md), [Tasks r1](tasks.md),
[C01–C16](examples.md), [failed/partial attempt-14](evidence-native-attempt14.md),
[attempt-15 prompt](https://linear.app/rgomids/issue/AXM-39#comment-1619543b).
Rafael's 25 October approximately 09:53 BRT instruction was relayed by Maestro,
a delegated AI; provenance reported, not independently authenticated. The user
explicitly supplied the bounded implementation/install/native/commit/push/receipt
grant for this execution. No new credential, login, provider setting, paid service,
privilege, production Project policy, Workflow selection or execution authority.

Three existing packaged skills changed: early scoped-answer procedures for human
labels, partial-inventory uncertainty, per-case source isolation, complete Stage
purpose/I/O/validator/output correlation, Runtime certainty and inspection-first
readiness. Work Item reviewed unchanged. Complete replaced four-skill revision
appended to shared history; new embedded manifest pinned. Compatibility/skill set
remain 3. No domain, CLI flags/output/schema, Registry, admission or Execution code
changed. Spec/Plan/Tasks and historical Evidence remain unchanged; implementation
Evidence receives only an appended continuation pointer. DF-01 remains deferred;
GitHub #346 untouched.

New guidance regression failed against unchanged source (12 missing early rules),
then passed after correction. New independent synthetic answers/omission and
invention controls cover C06–C11/C15/C16 source isolation. These tests validate
packaged guidance and offline oracles, **not probabilistic native compliance**.
Historical C01–C16 scoped passes are preserved, not retroactively relabelled.

## Supported owned installation and rollback qualification

Initial pair: private `0.21.3-axm39.14`, source e67018f9c3c5, both receipts current,
four skills equivalent, docs verified. Fresh bounded backup: **131 objects**,
manifest SHA-256 `9e8bbe86fd7eb09c04312a8dbdcf2531a707f03772edf9c4d9256920b696465a`.
Owned binary, installation receipt, full owned documentation generations/bindings,
eight skill files/directories and both skill receipts only. UID/GID, permissions,
mtime and file bytes independently compared after separate restore rehearsal:
PASS. Live baseline compared again before apply. No credential or unrelated
configuration copied. Backup/rehearsal retained privately; no live rollback claimed.
On failure: inspect exact ownership/ledger, then use the verified bounded restore
plan; never force a version, hand-edit receipts or mix binary/docs/skill pairs.

Clean source aff1acc, official builder, unique private **unpublished** candidate
`0.21.3-axm39.15`; no release/tag/publication claim:

```sh
./scripts/build-release-archives.sh --version 0.21.3-axm39.15 --format 2 --only macos-arm64 --output <private-candidate-root>
./scripts/install-release.sh --verify-only --helper <built-verifier> --helper-sha256 <independent-hash> --archive <candidate> --checksums <SHA256SUMS> --bin-dir <owned-bin-root> --receipt-dir <owned-receipt-root>
<candidate-binary> --json upgrade --archive <candidate> --checksums <SHA256SUMS> --bin-dir <owned-bin-root> --receipt-dir <owned-receipt-root>
# Same inputs, reviewed preview, plus:
# --preview-digest e0ae68e2f4a7e1eab5a37c376a4f38ad373f71c715d50badeb07789bd9361d51 --authorize-local
axiom --json runtime claude install
axiom --json runtime codex install
axiom --json runtime codex status
axiom --json runtime claude status
axiom --json docs inspect
```

All actual product commands exit 0. Strict pinned verifier reports
`archive_verify_success: format=2 platform=macos architecture=arm64`.
Archive member paths/types checked; archived skills match source bytes. Preview:
direct/compatible, absent_v1, no persisted transition, no leftovers; exactly eight
effects (docs generation/binding, binary, receipt, three Codex skills, skill receipt).
Apply confirmed. Claude supported install converged its owned set; repeated installs
for each Runtime stayed successful/current/equivalent. Two private harness response
assumptions initially failed after successful commands (verifier text was not JSON;
install has no `result` field). Observed successes were reconciled by read-back;
upgrade apply was **not** repeated. These were harness parsing errors, not product
failures or unknown mutations. No cross-root atomicity claim.

| Artifact | SHA-256 |
| --- | --- |
| Candidate archive | `7df22e855673cd24419415814a3aadf0ec86922416cd4cfb01e7b60f2f4340dc` |
| Installed binary | `ea7caa6e3647826a62db4e76c04271ec4989f513389c958dce345b3140367dcd` |
| Embedded manifest | `725010f55b3b9af755ab6520913088e6479074467b3fb15c46086c1e64b3fc0c` |
| Project skill, both roots | `42b86e7c5cd6a9929545382d640ee1957e60d2ec6b8076551f7ed60b7fc95604` |
| Workflow skill, both roots | `68e9d8d5b8b5f303e99b7fd7a6c18c7bf964aea6b3e9b1346a1705c6f64f2639` |
| Runtime skill, both roots | `b35c1b2781f9b2b7939f8a86bf0aa8d6a9741ef9b9ee580b66b2f3573e9252db` |
| Work Item skill, both roots | `ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3` |

Read-back: installed source aff1acc, docs verified/ok, compatibility 3, all four
skills equivalent and receipts current for both Runtimes. Inspector loaded remains
unknown; session reading/launch evidence below is a different observation.
Final documentary commit is not the installed binary's source commit.

## Real isolated Project and input provenance

New isolated `LINGO_PROJECTS_ROOT` and `LINGO_STATE_ROOT`, supported Project CREATE
preview/apply only; provider none, local qualification Repository. No policy or
Workflow mutation to manufacture readiness. Project
`65616e9c-ee54-4d65-862e-af47d535ab0b`, name `Órbita AXM-39 — qualificação local`,
slug `axm39-native-qualification`; CREATE digest
`551aa76f1e63f4d80859fae4b03087ed37aaa29a9ea779f35b836f7f54405b42`.
Both sessions actually ran these five independent reads, each exit 0:

```sh
axiom --json project list
axiom --json project show --selector 65616e9c-ee54-4d65-862e-af47d535ab0b
axiom --json project validate --project 65616e9c-ee54-4d65-862e-af47d535ab0b
axiom --json project workflow list --project 65616e9c-ee54-4d65-862e-af47d535ab0b
axiom --json project workflow show --project 65616e9c-ee54-4d65-862e-af47d535ab0b --source builtin --workflow default-sdd --revision 2 --digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068
```

Executor's initial show used unsupported `--project` and received explicit selector
validation_failure, with no mutation. Installed help supplied `--selector`; failed
capture preserved separately, corrected read exit 0. No canonical-store patch.
Real validation: command success, structure valid, effective blocked;
capability_mapping_missing for Work Item and Execution; execution also
runtime_policy_unavailable/policy_unconfigured. Business/technology warnings remain.
Builtin default-sdd r2 inspected, not selected. This Project cannot dispatch;
no native journey included stage execution or provider effects.

C01–C06, C08–C14 and synthetic DAG C15 use separately labelled public
**SCOPED_REPLAY** inputs answered through native inference. C10 has its own turn,
exclusive Plan/Profile/auth facts; no Project declaration supplied. C07 and the
sequential C15 definition come from actual installed default-sdd r2. C16 natural
and explicit prompts contain exactly the same four real read results; no fixture
or previous-case Project policy is substituted. CLI captures are operational
facts from this isolated Project, not production or AXM-12 E2E.

Supported cleanup archived only the qualification Project, preview digest
`faf9ef55973f5fc5406a36e086ec8d68adb19427c25291f564b956978684b1fb`,
apply/read-back exit 0. Default isolated list empty; include-archived shows exactly
that UUID archived. Ambient list observed empty after qualification. Isolated
records retained for audit/reversal; no unsupported delete. All eight installed
skill files still equal source after native sessions. Installed candidate retained.

## E07 — Actual native metadata, loading and controls

Codex CLI 0.163.0, Claude CLI 2.1.310. Read-only auth: Codex reports ChatGPT login;
Claude loggedIn true, oauth_token, firstParty. No credential file read/export/edit,
login refresh or API key. Child environment removed named provider/model overrides
without reading or printing their values. Authentication is not inference evidence.

Each conversation began with `Reply exactly OK. Do not use tools or load skills.`
This caused minimal real inference before metadata inspection; it is not claimed
as pre-inference model verification. No substantive case input before the exact
native metadata and full-file gates. All subsequent input/response/exit captures
retained privately; only classified public test inputs and visible answers are
published in [sanitized observations](observations-native-attempt15.md).

Codex thread `01a1732e-0764-7280-ba9b-7d80f4963258`: nine successful turns,
all actual process exits 0, nine native `turn_context` records with exact
`model: gpt-6.1-sol`, effort medium, approval never. No model drift/fallback.

```text
codex -m gpt-6.1-sol -a never exec -s read-only --json --skip-git-repo-check -C <native-scratch> 'Reply exactly OK. Do not use tools or load skills.'
codex -m gpt-6.1-sol -a never exec resume --json --skip-git-repo-check 01a1732e-0764-7280-ba9b-7d80f4963258 -
# Remaining turns after control reconciliation:
codex -m gpt-6.1-sol -a never -c 'sandbox_mode="read-only"' exec resume --json --skip-git-repo-check 01a1732e-0764-7280-ba9b-7d80f4963258 -
```

**Control discrepancy, not hidden:** initial turn recorded read-only; resumes
01–05 recorded danger-full-access despite the initial invocation's read-only flag.
The harness did not initially pin sandbox again on resume. When identified,
executor paused only its own controller, let its already launched case finish,
then stopped that controller before C08/C11–C16 input. The deliberate held-input
stop produced FileNotFoundError; it was not a failed native inference or a model
availability error. Same thread continued once with explicit supported config
above; contexts 06–08 positively recorded read-only/never. No model or permission
fallback, ambient config mutation or replay-for-PASS. Audit of **all nine tool
calls**: four installed `cat` reads and five exact Axiom read commands only.
No write/dispatch/provider action observed. Earlier turns are not retrospectively
certified read-only sandboxed; that limitation remains part of E07.

Native Codex tool warning: ignored unrecognized ambient `features.rmcp_client`.
No configuration repair. Explicit model pins plus actual metadata controlled model
choice; warning is not model fallback. This evidence is runtime routing metadata,
not independent backend attestation.

Explicit `$axiom-project $axiom-workflow $axiom-runtime $axiom-work-item` entry,
then four separate full `cat` outputs, each exit 0. Executor decoded actual native
tool output blocks and verified each installed file's entire bytes as a substring
of its output. Every file matched corrected source. No native Skill-tool launch
acknowledgement exists for Codex; hidden resolver/loading retention stays unknown.
Proven complete installed-file reading is not silently downgraded to unknown,
nor promoted to hidden native-resolver certification.

Claude session `2fc7bc09-ac29-4e48-850f-4baa4db9ee04`, one stream process,
nine successful results, exit 0, no permission denials. Init and every assistant
message model exactly `claude-haiku-5-5`, apiKeySource none, permissionMode dontAsk.

```text
claude --model claude-haiku-5-5 -p --input-format stream-json --output-format stream-json --verbose --no-session-persistence --setting-sources user --settings <session-only-restricted-JSON> --strict-mcp-config --mcp-config '{"mcpServers":{}}' --no-chrome --permission-mode dontAsk --tools Read,Skill,Bash --add-dir <four-owned-skill-directories> <qualification-root>
```

Session-only settings disable hooks; permit Read for exact four installed skill
files and four qualification captures, Skill for four domain names, Bash only
Project list/show/validate and Workflow list/show. Deny write/edit, Agent and web
tools. No permission bypass or host settings change. Four actual Skill launches,
four complete Read calls, five Bash reads; explicit C16 launched Project/Workflow
Skill again. Full Read outputs reconstructed from numbered lines including final
empty EOF line match each complete file byte-for-byte. Native Skill launch and
full byte reading are independently observed facets; neither proves perpetual
internal context retention.

## C16 graphical facet — actual tool observation, NOT_RUN

Computer-use inventory returned running iTerm2, ChatGPT, Claude and other apps;
no connected browser. Executor attempted the directly relevant graphical terminal:

```javascript
await cua.getApp('com.googlecode.iterm2');
```

Tool refused: `Computer Use is not allowed to use the app 'com.googlecode.iterm2' for safety reasons.`
No terminal UI was opened or controlled; no screen-width measurement, screenshot
or independent visual harness observation was possible through that attempted
surface. Other listed apps do not establish access to these exact pinned CLI
sessions. No alternative native-session graphical renderer was established.
The facet is **NOT_RUN**, not unsupported on all macOS hosts, not PASS and not
permission to bypass the tool restriction. No HTML/text replay, fake screenshot,
or model self-score substituted for independent graphical observation.

Natural/explicit C16 had identical actual fact bodies, SHA-256
`9e5c269fdc40b6098f93b6887a5d2f151fadca3bfd91706ea44d75c7324a35ec`.
Both runtimes supplied plain-text answers with Unicode human label, blocked state,
source and inspection-first next step. Requested narrow wording proves text only.

## Per-case assessment and AC reconciliation

PASS below always means **PASS_SCOPED_OBSERVATION**, not generic native acceptance.
SCOPED_REPLAY is synthetic data through real native inference. LIVE means real
isolated Project/installed definition. Verdict is executor assessment, not model
self-evaluation, independent Code Review or human acceptance.

| Case | Codex | Claude | Source/turn and actual scope |
| --- | --- | --- | --- |
| C01 | PASS | FAIL | 03 identity replay: Codex routine label; Claude prints full UUID while saying it is internal. LIVE 02 opening labels work; broader technical identity view is distinct |
| C02 | PASS | FAIL | 03 independent name-only/slug-only sources: Claude again appends UUIDs. Historical attempt-14 PASS preserved; this is a newly observed failing case |
| C03 | PASS | PASS | 03 replay: missing-both fallback, full ID and reason |
| C04 | PASS | PASS | 03 replay: both exact colliding IDs, repeated name-only choice unresolved; no sensitive action |
| C05 | PASS | PASS | 03 replay: complete requested selector/digest; consultation grants no execution |
| C06 | PASS | PASS | 03 replay: NOT_ESTABLISHED, inert escaped hostile label, missing ID/action guard; no injected file created |
| C07 | PASS | PASS | LIVE 02: actual intake/implementation purpose, I/O, common Agent/Role/constraints, validator and output existence/correlation; requested advanced detail. Not observed completion |
| C08 | PASS | PASS | 06 independent risk-r7 replay: content-check unavailable, timeout/limit, forbidden write ceiling, unknown effects, no invented gate state; not a real injected validator fault |
| C09 | PASS | PARTIAL | 04 replay: exact allowed/declarations and no invented model. Claude opens “Não” and says no selected/resolved Runtime rather than consistently qualifying these as unobserved. LIVE answer more carefully preserves usage uncertainty |
| C10 | PASS | PASS | 05 exclusive Plan/Profile/auth turn: no imported Project declaration; observed login vs resolution vs unobserved attempt/compatibility/billing/acceptance. No resolved model/effort claim without linkage |
| C11 | PASS | PARTIAL | LIVE 02 and replay 06: both blockers/no-dispatch scope/inspection-first next action retained. Claude replay omits command success and valid structure, although LIVE response preserves them |
| C12 | PASS | PASS | 06 replay: invalid result despite exit 0, timeout, both blockers and unknown effects; reconcile before retry |
| C13 | PASS | PASS | 06 replay: stale/conflicting references, inaccessible/truncated source; no freshness inferred from revision label |
| C14 | PASS | PASS | 04 replay: no registry/probe/certification/version gate, no success guarantee |
| C15 | PASS | FAIL | LIVE 02 complete ten-Stage order/gates plus 06 same-Stage DAG replay. Claude preserves DAG/independence/empty gates but shortens supplied full digest to `aaaa…aaaa`; exact AXM-38 identity fidelity fails |
| C16 | PARTIAL | PARTIAL | LIVE fact body 07 natural / 08 explicit: plain-text semantic parity observed. Graphical/narrow/independent visual facet NOT_RUN; Codex hidden resolver unknown; Claude native Skill launches and full reads proven |

No attempt-14 or earlier record overwritten. Approved historical scoped PASS,
including C02–C05/C12–C14, retains its original source/status. New failures do not
reclassify history and cannot be concealed to maintain an all-green matrix.

T01–T06 historical scoped PASS preserved; current correction regression/package
checks PASS. T07 **PARTIAL**, E07-Codex/E07-Claude **PARTIAL**; both actual pinned
conversations occurred but required presentation/graphical acceptance incomplete.
T08/E08 **PARTIAL**: honest new crosswalk and callback, not implementation closure.

| AC | Result | Evidence boundary |
| --- | --- | --- |
| AC1 | PARTIAL | Claude C01/C02 UUID violations; C16 visual gap; offline identity/collision/negative controls pass |
| AC2 | PARTIAL | Actual Stage summaries/criteria observed; Claude C15 full-reference failure and C16 visual gap |
| AC3 | PARTIAL | C10 source isolation improved, exact pins/bytes proven; Claude C09 certainty and Codex resolver limits remain |
| AC4 | PARTIAL | Inspection-first next action improved; Claude C11 replay loses success/structure dimensions |
| AC5 | PARTIAL | Two nine-turn native conversations, per-case captures; required failures and independent visual NOT_RUN remain |
| AC6 | PASS_SCOPED_REVIEW | Reviewed scoped skill/history/test/Evidence diff; no domain/flags/Registry/schema/admission/Execution change; not independent CR |

## Actual local verification

| Command / check | Result |
| --- | --- |
| `go test ./internal/codexruntime -run '^TestConversationalAttempt15Guidance$' -count=1` before/after | FAIL/1 before (12 missing rules), PASS/0 after |
| `go test ./internal/cli -run 'Test(OutputSeparation|WorkflowShowSynthetic|Presentation)' -count=1` | PASS/0 |
| `go test ./internal/codexruntime -run 'Test(RenderedWorkflow|OperatorConversational|WorkflowSkill|Conversational|EveryPublished)' -count=1` | PASS/0; includes both-root upgrade/conflict/interruption/restoration contracts |
| `go test ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./cmd/lingo ./internal/install` | PASS/0; codexruntime 16.341s, runtimebootstrap 0.852s, cli 1.035s, cmd/lingo 410.883s, install 80.557s |
| `go vet ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./cmd/lingo ./internal/install` | PASS/0 |
| `go build ./...` | PASS/0 |
| `./scripts/check-go-quality.sh format` | PASS/0 |
| `./scripts/validate-repository.sh .` | PASS/0 |
| `git diff --check`; historical prefixes/bytes and links | Final documentary precommit check below and same-AXM receipt |
| `./scripts/check-sensitive-files.sh --staged .`; scoped redacted Gitleaks | Source commit PASS/0; final documentary results in same-AXM receipt |
| Exact final pushed-head CI | Read after final push; report exact SHA/conclusions/URLs in completion event, not input-SHA CI |

No new full local `go test ./...`, race suite or AXM-12 E2E claimed. All canonical
product commands and nine native turns per runtime have separately retained exits.
Native transport success is not semantic PASS. No test retries or prompt rewrites
performed to force native PASS; control correction continued only unsent cases.

## technical_decisions[] and bounded handoff

All TECHNICAL_LOCAL, executor-ai, scope_impact none; evidence is this document and
source commit aff1acc in the same PR.

- **IMPL15-TD-01:** early scoped-answer procedures in three existing skills, complete
  prior four-skill history, compatibility 3 unchanged. Alternative schema/domain
  formatter or model-specific product policy rejected: outside baseline. Work Item
  reviewed unchanged; deterministic content and negative tests are not AI compliance.
- **IMPL15-TD-02:** clean official private prerelease candidate .15 and supported
  digest-bound owned upgrade/Claude convergence, verified backup/restore rehearsal.
  Alternative manual file/receipt replacement or version forcing rejected. No live
  rollback needed; no release/tag claimed. Installation/reading/acceptance separate.
- **IMPL15-TD-03:** independent C10 source scope and identical actual C16 fact bodies,
  one pinned conversation each; synthetic cases labelled SCOPED_REPLAY, not live.
  Preserve failed answers and unused graphical facet instead of replay/self-score
  or PASS-seeking repetition. Native model metadata positively checked, no fallback.
- **IMPL15-TD-04:** after observed Codex resume sandbox drift, stop own controller
  before unsent cases, explicitly pin supported sandbox config on remaining resumes
  and verify actual metadata. Alternative ignoring discrepancy rejected. Observed
  operations audited read-only; earlier sandbox metadata retained as limitation.

**DF-01: DEFERRED_NOT_IMPLEMENTED**, existing predecessor-index summary finding;
no new optional finding/issue, no #346 change. Required failing cases remain
in scope, not deferred. One consolidated remaining reason: corrected instructions
still lack complete native fidelity (Claude C01/C02/C09/C11/C15), and exact pinned
CLI graphical/independent observation was refused by the available terminal tool.
No deterministic supported response-enforcement or alternative graphical interface
was established in this pass; none was invented or added to the product.

Next gate **RESUME_IMPLEMENT**; not REVIEW or RECONCILE_IMPLEMENT. Further bounded
in-scope correction and honest requalification of remaining facets are required.
This report is the completion receipt for the performed attempt, not a claim that
all approved work completed. Same-AXM root stage.partial event with exact final
SHA/CI, duplicate check and API read-back; no genuine independent-review continuation
callback established, so **AWAITING_EXECUTION_REPORT**, without ceremonial approval.
No self-review approval, ready-for-review conversion, merge, release, acceptance,
Done, card closure or unrelated write.
