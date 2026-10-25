# Attempt-16 — Integrated remediation; native qualification remains partial

2026-10-25, macOS arm64. Same open Draft [PR #343](https://github.com/rgomids/axiom/pull/343),
branch `docs/axm-39-conversational-presentation`. Initial local/remote HEAD
`6e7c005c44684fdf711114289417bb48030792f5`, clean worktree. Corrected source
`da39492665b6350385701f7f3d1b30f643f335ab`; final documentary HEAD is separately
recorded with exact CI in the same-AXM completion event. Input-SHA CI is not reused.

## E01 — Baseline, authority and scope

Read [Spec r1](spec.md), [Plan r1](plan.md), [Tasks r1](tasks.md),
[C01–C16](examples.md), [attempt-15 ledger](evidence-native-attempt15.md) and
[observations](observations-native-attempt15.md), authenticated
[attempt-15 PARTIAL receipt](https://linear.app/rgomids/issue/AXM-39#comment-ecb2dce7)
and [attempt-16 prompt](https://linear.app/rgomids/issue/AXM-39#comment-0491329a).
77 Linear comments inventoried through hasNextPage false; prompt/dispatch present,
no existing attempt-16 completion. AXM-36/37/38 status Done read back; installation
and native facts verified independently. PR open/Draft, auto-merge absent.

Rafael's bounded authorization is supplied in this request and reported by Maestro,
a delegated AI, not independently authenticated human identity. It covers the
in-scope reversible owned install, isolated qualification, native pinned sessions,
scoped commits/push and final Linear receipt performed here. No new credentials,
login/provider changes, privileged effects, production policy/Workflow mutation,
paid-resource provisioning, merge, release, acceptance or Done authority inferred.

Only three packaged guidance files, two tests and complete four-skill history/pin
changed. Work Item reviewed unchanged: existing ownership/result/source distinctions
remain sufficient. Compatibility and skill-set stay 3. No canonical CLI/domain/
schema/flag registry/admission/Execution/renderer change; no SDD reopening or ADR.
Historical records and approved baselines unchanged; only this new ledger,
observations and an appended implementation-Evidence pointer are added.

Process/cwd inventory found no overlapping executing writer before source edits;
other Claude workers had other directories. A later same-directory Codex process
was identified as the computer-use app-server service, not a second dispatched
Implement battery. No observed concurrent source mutation. Serial source/manifest
ownership throughout. No subagents: capability ordering not established.

## E02–E06 — Reproduction, correction and deterministic validation

New guidance test failed before edits with seven absent answer-audit obligations;
passed after edits. A separate pre-correction pinned Claude diagnostic returned
exit 0 and reproduced gratuitous C02 UUIDs on installed .15. Its self-reported
loading is not promoted to complete tool-read proof. Actual answers retained in
[new observations](observations-native-attempt16.md). Inspection confirms Work Item
owns `workflow stage status`; Project owns the empty `--policy-draft '{}'` read.
These are not reported as nonexistent commands.

Cross-cutting correction: audit the outgoing answer for label-vs-authority scope,
all three success/valid/blocked dimensions, unknown-vs-negative Runtime claims,
case-local Profile sources, complete Workflow digest and inspection-first action.
The source already had related rules; native failures demonstrate incomplete
application, not a new domain requirement. Tests cover packaged obligations and
independent public candidate answers with omission/invention/truncated-digest
negatives. They do not enforce probabilistic model output or prove native/visual ACs.
Complete replaced four-skill revision appended; all historical pins/ownership retained.

Executed, exit 0 unless explicitly noted:

- Regression-first `go test ./internal/codexruntime -run TestConversationalAttempt16AnswerAudit -count=1`: exit 1 before correction, exit 0 after.
- `go test ./internal/codexruntime ./internal/cli -run 'Test(ConversationalAttempt|PresentationAttempt|EveryPublishedSharedRevision)' -count=1`.
- `go test ./internal/cli ./internal/codexruntime ./internal/runtimebootstrap ./internal/compatibility ./cmd/lingo`: all five packages PASS; cmd/lingo 423.143s.
- `go vet ./internal/cli ./internal/codexruntime ./internal/runtimebootstrap ./internal/compatibility ./cmd/lingo`.
- `go build ./cmd/lingo`; generated binary moved to private qualification storage.
- `gofmt` on changed Go files; no remaining formatting diff.
- `./scripts/validate-repository.sh .`: repository/link/structure/security checks PASS.
- Full scoped diff review, `git diff --check`, staged sensitive-file scans and redacted Gitleaks scans: PASS. Final documentary scans/CI recorded in completion receipt.

Scoped review: no public contract/architecture/security-boundary change identified.
This is executor scope review, not independent Code Review or AC acceptance.
Required native failures below remain blocking regardless of deterministic PASS.

## Supported owned installation and rollback provenance

Initial pair `0.21.3-axm39.15`, source aff1accef3a0a5ada730d350409fc7153d4c06a0;
both receipts current/equivalent, docs verified. Fresh bounded backup of 167 owned
objects: binary, install receipt, owned docs generations/bindings, both skill
receipts and eight skill files/directories. Backup manifest SHA-256
`31b86c7acf5742a1ce8eee6e59265751853923f1a6b0d4594f1d3ea77aef318e`.
Separate restore rehearsal matched UID/GID/mode/mtime and every file byte.
Live baseline rechecked before apply. No credential/unrelated config copied;
backup and rehearsal retained privately. No live rollback claimed; recovery uses
that bounded verified baseline, never hand-edited receipts or mixed pairs.

Official builder on clean source da39492665b6350385701f7f3d1b30f643f335ab produced
private unpublished candidate `0.21.3-axm39.16`, format 2, macos-arm64. This is a
unique qualification version, not a forced version, tag, release or publication.
Strict helper hash/archive checksum/path/type/skill-byte verification passed.
Preview direct/compatible, absent_v1, no persisted transition, no leftovers; eight
effects only: docs generation/binding, binary/install receipt, three Codex skills,
skill receipt. Reviewed exact preview digest:
`69eb74b9531b4361409aa0a9c00260eddbc17dd67fa52811fae5b3baed077f67`.
Apply plus `--authorize-local` exit 0. Supported Claude install converged its owned
set; repeated supported installs successful/current/equivalent. Complete files
and receipts read back; docs verified; all eight skill files equal source after
native sessions. Inspector loaded UNKNOWN is separate from session full-read proof.

```sh
./scripts/build-release-archives.sh --version 0.21.3-axm39.16 --format 2 --only macos-arm64 --output <private-candidate-root>
./scripts/install-release.sh --verify-only --helper <built-helper> --helper-sha256 <verified-hash> --archive <candidate> --checksums <SHA256SUMS> --bin-dir <owned-bin-root> --receipt-dir <owned-receipt-root>
<candidate-binary> --json upgrade --archive <candidate> --checksums <SHA256SUMS> --bin-dir <owned-bin-root> --receipt-dir <owned-receipt-root>
# Same inputs, reviewed digest above, then --preview-digest <exact-digest> --authorize-local
axiom --json runtime claude install
axiom --json runtime codex install
axiom --json runtime claude status
axiom --json runtime codex status
axiom --json docs inspect
```

| Artifact | SHA-256 |
| --- | --- |
| Archive | `045276faba99e1e987429bc9631affcb0f062710ff09d17556571b641898ff4e` |
| Installed binary | `8a64655d36b74662950d7b67bfb935491fdcf3c730b0f6b510574ad27b38bd5f` |
| axiom-project | `fddb0c49313c98f5b726acb9eb6356cc6c47c4cebf7a521079758fb1f0b8f647` |
| axiom-workflow | `20b6d727b97505623df2cf3adb570bb042a5c02a74c05e6ffcf5d196f8d44270` |
| axiom-runtime | `aeed4e7cfc8c81c37935d630086ab7920b422421d5876119b72841ec06931bd9` |
| axiom-work-item | `ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3` |

Embedded shared manifest pin: `e29183f85292f1aeb7fe37ed6a5688e66583e293340d249666208d76b7526696`.
Installed binary source is da39492665b6350385701f7f3d1b30f643f335ab, not automatically
the later documentary HEAD.

## E07 — Real Project, native sessions and source isolation

One Project in fresh isolated LINGO_PROJECTS_ROOT/LINGO_STATE_ROOT, provider none,
local-only qualification Repository, no Runtime policy or Workflow selection edit.
UUID `0c96e863-80eb-431f-aa2e-c3b8bafa774b`, name `Órbita AXM-39 — qualificação local`,
slug `axm39-native-qualification`; CREATE preview digest
`486d2a70e0bc137696aac18f73d624f207250fba6681bf110fec39d08824bc40`.
Create/apply and five read commands exit 0. Each native session actually ran:

```sh
axiom --json project list
axiom --json project show --selector 0c96e863-80eb-431f-aa2e-c3b8bafa774b
axiom --json project validate --project 0c96e863-80eb-431f-aa2e-c3b8bafa774b
axiom --json project workflow list --project 0c96e863-80eb-431f-aa2e-c3b8bafa774b
axiom --json project workflow show --project 0c96e863-80eb-431f-aa2e-c3b8bafa774b --source builtin --workflow default-sdd --revision 2 --digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068
```

Result: success, structurally valid, effective/execution blocked by
capability_mapping_missing and runtime_policy_unavailable / policy_unconfigured.
No policy repair/dispatch to manufacture readiness. Supported archive preview/apply
exit 0, digest `f1b9805f858094f95b1385724411b7729fb355c836cce335cdd3a181b7df5d69`.
Read-back: isolated default list empty; include-archived shows that Project archived;
ambient inventory empty. Records retained, no unsupported delete.

Codex CLI 0.163.0; Claude CLI 2.1.310. Read-only login diagnostics report ChatGPT
login / Claude firstParty oauth login. No credential file read, login mutation,
provider change or key fallback. Child process environment strips named provider/
model overrides without inspecting their values. Auth is not model/inference proof.

Both sessions begin minimal real inference `Reply exactly OK. Do not use tools or
load skills.` before native metadata inspection; this is not pre-inference model
verification. No case input before metadata/full-read gates. Every subsequent
invocation pinned; no fallback, model substitution or resume sandbox drift.

```text
codex -m gpt-6.1-sol -a never exec -s read-only --json --skip-git-repo-check -C <native-scratch> 'Reply exactly OK. Do not use tools or load skills.'
codex -m gpt-6.1-sol -a never -c 'sandbox_mode="read-only"' exec resume --json --skip-git-repo-check <same-thread-id> -
claude --model claude-haiku-5-5 -p --input-format stream-json --output-format stream-json --verbose --no-session-persistence --setting-sources user --settings <session-only-restricted-JSON> --strict-mcp-config --mcp-config '{"mcpServers":{}}' --no-chrome --permission-mode dontAsk --tools Read,Skill,Bash --add-dir <four-owned-skill-directories> <qualification-root>
```

Codex nine actual process exits 0; all nine native turn_context records show
model gpt-6.1-sol, effort medium, sandbox read-only, approval never. Tool audit:
exactly four complete installed cat reads and five Axiom read commands, all exit 0.
Every full file appears byte-for-byte in its actual tool output. Hidden native
Skill resolver UNKNOWN; complete file reads are positively observed.

Claude one stream process exit 0, nine successful results, zero permission denials.
Init and all 34 assistant messages exactly claude-haiku-5-5, apiKeySource none,
permissionMode dontAsk. Session-only hooks disabled, strict empty MCP, Read/Skill/
five Axiom read-command permissions; Write/Edit/Agent/web denied. No bypass or
host settings edit. Four native Skill launches plus four complete Read results
match each installed file after reconstructing numbered lines including EOF;
explicit C16 launches Project/Workflow again. Context retention remains UNKNOWN.
Native routing metadata is not independent backend attestation or billing proof.

Codex thread `01a1734e-2a68-7a51-a196-8cfdf9a5496d`; Claude session `7c32e133-554b-4d8c-9043-8950482a426c`.

C10 has exclusive Plan/Profile/login facts, no borrowed Project declaration.
C16 natural and explicit entries use identical actual four-result bodies;
unredacted body SHA-256 `8d2ff259e11cf028ce1d48d15cbac4d655b5e698943b2462207c59615b1d8db4`.
Codex textual semantics PASS_SCOPED; Claude preserves textual parity but repeats
omissions/overclaims. LIVE remains actual CLI data; replay remains synthetic
SCOPED_REPLAY even when answered by real native inference. No model self-score is
used as the executor's independent verdict.

## C16 visual — bounded supported alternatives, NOT_RUN

Previous iTerm2 refusal preserved; not retried or bypassed. Actual attempt-16:

| Method | Actual result |
| --- | --- |
| `cua.getState()` | apps inventoried; browsers empty |
| `cua.getApp('com.openai.codex')` | `Computer Use is not allowed to use the app 'com.openai.codex' for safety reasons.` |
| `cua.getApp('com.anthropic.claudefordesktop')` | `js execution timed out; kernel reset, rerun your request` after 30.0214 seconds; no app state or screenshot |
| `cua.createBrowserTab('iab', 'about:blank', { visible: false })` | `Browser is not available: iab` |

No supported accessible native/harness renderer established through these bounded
alternatives. No UI injection, terminal-security bypass, screenshot synthesis,
re-rendered text masquerading as native observation, width claim or graphical PASS.
C16.visual/narrow/independent-rendered-observation = NOT_RUN. A supported authorized
observer remains necessary; unavailable UI is a technical/tool-access gap, not a
new architecture/design decision or implied permission to widen access.

## E08 — Case/AC/task reconciliation and remaining work

PASS_SCOPED means only exact sources/answers in [observations](observations-native-attempt16.md).
Historical attempt-15 and earlier PASS/FAIL/PARTIAL remain immutable. A replay pass
cannot erase an actual LIVE failure or certify a graphical facet.

| Case | Codex attempt-16 | Claude attempt-16 | Scope / remaining finding |
| --- | --- | --- | --- |
| C01 | FAIL_LIVE; PASS_REPLAY | FAIL_LIVE; PASS_REPLAY | Both add real Project UUID during LIVE Workflow explanation; operational necessity not established; routine replay labels pass |
| C02 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Name-only/slug-only; no gratuitous UUID or invented field |
| C03 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Both missing: full fallback ID/reason |
| C04 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Both full collision IDs; name-only follow-up unresolved |
| C05 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Requested full ID/selector/digest exact; no authority invented |
| C06 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Partial inventory unknown, missing ID not actionable, escaped hostile Unicode label inert |
| C07 | PASS_SCOPED_LIVE | PARTIAL | Both Stage summaries faithful; Claude expands advanced implementation only, intake advanced fields not fully re-expanded; historical scoped PASS preserved |
| C08 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Validator/timeout/limit/ceiling/unknown effects retained; no retry/gate invention |
| C09 | PASS_SCOPED_LIVE_REPLAY | PASS_SCOPED_LIVE_REPLAY | Allowed/declarations/unknown resolution distinct; codex-test only C09 source; C16 certainty defect assessed below |
| C10 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Exclusive Plan/Profile/login sources; execution/compatibility/billing/acceptance unknown |
| C11 | PASS_SCOPED_LIVE_REPLAY | PASS_SCOPED_LIVE_REPLAY | Command success + valid structure + blocked, exact blockers and inspection-first; C16 omission separately remains |
| C12 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Prior unknown effects/timeout/authority/result-invalid despite exit 0; reconcile before retry |
| C13 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Stale/inaccessible/truncated source not promoted to current truth |
| C14 | PASS_SCOPED_REPLAY | PASS_SCOPED_REPLAY | Unknown version alone not admission failure/certification demand |
| C15 | PASS_SCOPED_LIVE_REPLAY | PASS_SCOPED_LIVE_REPLAY | Complete real sequential order/gates plus independent replay DAG/full digests/text fallback; no graphics claimed |
| C16 | PARTIAL | PARTIAL | Codex text parity PASS_SCOPED; visual NOT_RUN. Claude natural/explicit both omit command success, assert no selected/resolved/executed Runtime and closed gates without matching evidence; parity alone not correctness |

T01–T06 historical scoped PASS preserved; current deterministic correction/package/
security checks PASS. T07/T08 and E07-Codex/E07-Claude/E08 PARTIAL. AC1 PARTIAL
(LIVE UUID); AC2 PARTIAL (Claude intake advanced details and rendered observation);
AC3 PARTIAL (Claude C16 certainty); AC4 PARTIAL (Claude C16 command-status omission/
gate certainty); AC5 PARTIAL (native failures/visual gap); AC6 PASS_SCOPED_REVIEW,
not independent Code Review. No requirements converted to optional.

One coherent answer-audit/history revision and one post-install qualification
battery per Runtime were completed. Actual failures retained; no same-source
rerun for a favorable answer, model fallback, host-global prompt override or new
formatter/response-enforcement pipeline. Existing explicit guidance is not a
supported deterministic guarantee of model compliance. Remaining fidelity needs
further evidence-backed in-scope remediation; visual facet needs a genuinely
accessible authorized observer. Neither gap is solved by CI, installation, synthetic
oracles, declaring Runtime absence, or a new acceptance criterion.

## technical_decisions[] and handoff

All TECHNICAL_LOCAL, decision_maker executor-ai, scope_impact none, evidence this
ledger and source da39492665b6350385701f7f3d1b30f643f335ab:

- **IMPL16-TD-01:** outgoing-answer audits in existing three guidance owners plus
  complete four-skill history and focused oracle negatives. Alternative new CLI/
  schema/formatter or model-specific product policy rejected outside baseline;
  Work Item unchanged. Native failures limit the result despite deterministic PASS.
- **IMPL16-TD-02:** clean private .16 candidate, verified bounded backup/restoration
  rehearsal and digest-bound supported owned upgrade/convergence. Alternative
  receipt editing/version forcing rejected; private candidate is not a release.
- **IMPL16-TD-03:** exact pins and explicit Codex resume confinement every call,
  full source reads before case input, C10 isolation/C16 identical real bodies,
  retain actual failures. Alternative repeated same-source PASS-seeking or model
  substitution rejected; deterministic and native results remain separate.
- **IMPL16-TD-04:** bounded visual alternatives via documented computer-use API;
  stop after actual refusals/timeout/unavailability, retain NOT_RUN. Alternative
  bypass or fabricated screenshot rejected; no new access or settings changes.

DF-01 DEFERRED_NOT_IMPLEMENTED; existing [finding](evidence-specify.md) and GitHub
[#346](https://github.com/rgomids/axiom/issues/346) untouched. No new optional scope.
Next gate **RESUME_IMPLEMENT**, overall **PARTIAL**, not READY_FOR_REVIEW.
No real independently authorized continuation callback established by this stage;
checkpoint **AWAITING_EXECUTION_REPORT** after one same-AXM completion event and
read-back. No request for human merge, self-approval, release, acceptance or Done.
