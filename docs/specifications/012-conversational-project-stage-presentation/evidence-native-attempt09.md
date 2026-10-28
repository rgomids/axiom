# Attempt-09 — Native loading boundary and bounded follow-up

2026-10-24 BRT (2026-10-25 UTC). [Implementation reconciliation](evidence-implementation.md#attempt-09--native-claude-response-without-qualified-skill-loading).
Selected sanitized observations over public synthetic inputs, not a provider
transcript export. No personal paths, account data or authentication material.

## Inputs, authority and observed surfaces

Input Evidence HEAD `0825d24dc9ae056ad684641d9a695e02498d33c6`; installed
candidate source remains `e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6`, version
`0.21.3-axm39.8`. These identities are deliberately different.
[Attempt-09 prompt/authority report](https://linear.app/rgomids/issue/AXM-39#comment-77b2d1f5)
and [dispatch](https://linear.app/rgomids/issue/AXM-39#comment-85bdf9fa)
were read back; no completion or competing dispatch observed before writes.
Reported authorization is Maestro's relay of Rafael's approximately 20:59 BRT
grant, not independently authenticated human-session Evidence.

Both conversations ran outside the maintainer checkout on Darwin arm64, using
existing subscription CLIs. No model override, delegation, login/logout/refresh,
alternate key, installation, Project mutation, Stage dispatch or Provider write.
Child environments omitted API-key/token/provider override variables by name;
no values were read/exported. Official Axiom auth diagnostics immediately before
each conversation returned exit 0, `subscription_observed` / `local_observation`:
Claude `claude.ai`, Codex `chatgpt`. These diagnostics alone did not prove inference.

Exact [attempt-08 cases](evidence-native-attempt08.md#replay-input) reused without
changes; SHA-256 `9034b3a76071ba7134c6566e0759ce422c93b3d8e5fdfd8ce2b281456ae3bddd`.
Exact public [Workflow fixture](../../../internal/codexruntime/testdata/rendered-workflow/default-sdd-r2.json)
file SHA-256 `67eb107befe04198b57a56dbf548234dd26ad99f324aa11ad04779d8182fad81`;
definition default-sdd r2 digest
`1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068`.
Embedded historical producer revision `4bf046f79d7d5006fea25ff2b5dc4dafc8a48fdc`
remains historical, not newly produced native/installed Evidence. Fixture Project
`123e4567-e89b-42d3-a456-426614174000` is distinct from synthetic Órbita.

Claude Code **2.1.310**, one native process, two interactive stream-JSON user
turns, exit **0**. Init directly reports `claude-opus-5-5`, `apiKeySource: none`,
permission mode `dontAsk`. Successful model responses observed; effort UNKNOWN.
This is a native CLI conversation with streamed input/output, not a witnessed
terminal GUI or graphical/narrow viewport.

Codex CLI **0.163.0**, bounded new follow-up thread and explicit second user turn
through `exec resume` of that exact thread; both exits **0**. Read-only sandbox,
approval never, user configuration omitted, multi-agent feature disabled. Exact
model/effort UNKNOWN in consulted event output. No use of `--last` or unrelated
session history. Execution metadata persistence is not an agent-authorized write.

## Prompts and permission outcome

Claude used the [same substantive attempt-08 prompt](evidence-native-attempt08.md#sanitized-prompt)
and exact cases above, with its own installed root. Added instructions: read each
complete skill individually, hash all four, restrict commands to read-only Axiom
inspection/list and hashing, give concrete C08 recovery, reserve C16 for a separate
explicit-entry user turn, and stop any denied action without workaround.

Invocation restricted tools to Read/Bash/Skill, explicit allow rules and
`dontAsk`; omitted setting sources, disabled hooks, Chrome and external MCP
configuration. **This executor setup did not expose installed user skills or
admit outside-working-directory skill reads.** The six Bash requests also appended
an exit-reporting suffix not covered by the exact command allow rules. No expanded
permission or alternative tool was used to bypass these denials. Exact permission
matching/root cause was not separately reproduced with another inference call.
This is an unqualified invocation, not proof of an installed-product discovery bug.

Four Read requests for installed SKILL.md, one hash request and five Axiom command
requests were denied before execution. They have **no shell exit code**. Only
local cases/Workflow Read calls succeeded. Claude continued with explicitly
synthetic prompt/fixture-only answers despite the failed session-loading check;
that continuation **does not qualify T07**. The executor should have staged this
loading check before requesting the replay. No second Claude conversation was
launched under the one-conversation bound.

Second actual Claude user turn began `/axiom-project`, requested the same
synthetic C01/C07/C11 and explicit Workflow entry, exact IDs/digests, Unicode,
authority, both blockers and next action. Actual response:

> Entrada explícita indisponível. O comando /axiom-project não existe nesta sessão e não rodou.

Init listed neither installed Axiom skill; no Skill invocation occurred. This
is an attempted explicit entry, **NOT_RUN** as a real skill entry. Phrasing
comparison happened, but cannot close C16 or loading.

Codex first prompt restricted work to C08 and C01/C07/C11 needed for C16; requested
individual full installed reads/hashes, same synthetic fixture, concrete safe
recovery and no repeat of other attempt-08 cases. Second actual user turn began
`$axiom-project $axiom-workflow`, asked equivalent facts on the observed plain-text
stream and explicitly withheld GUI/rendering claims. Native response applied the
installed bytes read in turn one. No separate harness re-expansion event was
observed; no claim that a new automatic skill expansion was independently proved.

## E07-Codex — Actual reads and selected excerpts

Four individual `cat <installed-root>/<skill>/SKILL.md`, `cat cases.txt`,
`cat workflow.json`, four-file `shasum -a 256`, and `axiom --json docs inspect`
all exit **0**. Complete contents of each four skill tool results were compared
directly with actual installed text: **exact equality**, not a substring/truncation
heuristic. Hashes equal the [qualified four-skill inventory](evidence-implementation.md#e07--new-backup-supported-previewapply-and-both-root-read-back).
Runtime's full 17,182 bytes were included individually this time. This supplies
actual complete-read/content-revision Evidence, unlike grouped attempt-08 reads;
permanent model-context retention still cannot be independently attested.
Inspector continues to report `loaded: unknown`; no receipt was overwritten.

- C08: “a menor próxima inspeção é conferir, pelo owner axiom-workflow, a referência exata e a configuração de validate-result/builtin-sdd-v1”. Response also identified Work Item ownership of plan/dispatch/envelope, kept both blockers immediate, prohibited direct write/validator removal/Runtime fallback/redispatch, and required separate scope, preview and exact authority for future change. **PASS_SCOPED_REPLAY** for the missing recovery step; inspection itself not executed.
- C01: “Órbita (orbita) é o único Project no inventário fornecido.” Structure-valid/readiness-unknown preserved; fixture Project kept separate.
- C07: both Stage essentials, input/output, declared gate/timing, owner/Role, allowed/declaration vs resolution and completion criteria; advanced validator, attempts, timeout, concurrency, effects and failure policy recovered from the exact fixture.
- C11: “status success indica conclusão bem-sucedida da operação relatada, não execução pronta.” Both `runtime_policy_unavailable` and `policy_unconfigured`, confirmed no-dispatch and next read-only policy inspection preserved; absent C11 target not assigned to Órbita.
- C16 explicit response: “A entrada explícita não acrescenta poderes nem exige um novo despacho.” Full requested synthetic ID `11111111-1111-4111-8111-111111111111`, Unicode name/slug and authority digest `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa` retained. Semantic comparison succeeded on this text stream; no narrow GUI was observed, so aggregate C16 **PARTIAL**.

No new live Project list/show/validate attempted by Codex; attempt-08 empty live
list remains its historical observation, not a fresh empty-inventory assertion.

## E07-Claude — Actual responses and nonconformance

Actual response explicitly acknowledged no installed skill guidance and no live
Project observation. Auth failure from attempt-08 did **not** recur. Successful
subscription inference does not imply correct presentation or loaded revision.

- Boundary: “Nenhuma das quatro skills pôde ser lida e nenhum comando axiom ou shasum rodou.” Loaded revision **UNKNOWN**, hashes unavailable inside this session. Outside-session disk equality is not substituted.
- C02 answered that name and slug point to the same Project, rather than independently presenting the supplied name-only/slug-only absence variants: **FAIL**.
- C05 supplied `axiom --json project show 11111111-1111-4111-8111-111111111111`, admitting syntax was unverified. Existing supported syntax requires `--selector`: **FAIL**, command was never executed. See [installed-source operation table](../../../internal/codexruntime/skills/axiom-project/SKILL.md).
- C07 omitted implementation output/completion criteria from the initial summary: **PARTIAL**; advanced fields alone do not repair that summary.
- C08 promoted both blockers and proposed read-only inspection, but abbreviated the definition digest in that recovery reference: **FAIL** against exact authority/reference preservation. No fixture-derived command executed.
- C11 retained both blockers/no-dispatch, but next step jumped to configuring policy without first identifying safe inspection/preview and separate change authority: **PARTIAL**; no configuration performed.
- C14: “A prontidão continua desconhecida até a versão ser observada.” **FAIL**: unknown version alone became an admission gate, contradicting R2 and the [existing Runtime guidance](../../../internal/codexruntime/skills/axiom-runtime/SKILL.md). This prompt-only response is not evidence of a regression in bytes that were never loaded.
- C16 stated installed explicit entry unavailable, compared ordinary phrasing only and omitted the requested authority digest: **PARTIAL**, actual skill-entry subcase **NOT_RUN**.

## Per-case reconciliation

PASS_SCOPED_REPLAY denotes semantic response to labelled synthetic facts only.
Even Claude replay passes below **do not validate installed skill behavior**:
all Claude cases share the failed loading/entry prerequisite. No configured
Project show/validate, real Stage execution or AXM-12 E2E acceptance established.

| Case | Codex combined Evidence | Claude attempt-09 observed replay |
| --- | --- | --- |
| C01 | PASS_SCOPED_REPLAY; attempt-08 + bounded comparison | PASS_SCOPED_REPLAY; prompt-only |
| C02 | PASS_SCOPED_REPLAY; attempt-08 retained | FAIL; absence variants not independently presented |
| C03 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; absent labels/full ID reason |
| C04 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; both full IDs/ambiguity/no mutation |
| C05 | PASS_SCOPED_REPLAY; attempt-08 retained | FAIL; unsupported copyable command |
| C06 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; uncertainty/inert hostile label |
| C07 | PASS_SCOPED_REPLAY; attempt-08 + follow-up | PARTIAL; incomplete essentials |
| C08 | PASS_SCOPED_REPLAY; concrete recovery added | FAIL; abbreviated recovery reference |
| C09 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; allowed/declaration/unresolved |
| C10 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; resolution/login/no attempt split |
| C11 | PASS_SCOPED_REPLAY; attempt-08 + follow-up | PARTIAL; unsafe/incomplete next-action framing |
| C12 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; reconcile unknown effects before retry |
| C13 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; stale/inaccessible/truncated uncertainty |
| C14 | PASS_SCOPED_REPLAY; attempt-08 retained | FAIL; version-only admission gate |
| C15 | PASS_SCOPED_REPLAY; attempt-08 retained | PASS_SCOPED_REPLAY; text topology/fallback only |
| C16 | PARTIAL; separate explicit text turn now observed, no narrow GUI/harness expansion proof | PARTIAL; explicit skill entry NOT_RUN |

Next technical gate: **NATIVE_CLAUDE_LOADING_ENTRY_REQUIRED**, within
**RESUME_IMPLEMENT**. Prepare an invocation whose supported scope includes the
owned skill directories and exposes their native entries; verify loading before
replay. Do not request login renewal or treat corrected runtime flags as a material
human design decision. Further native inference must stay within an explicitly
bounded conversation grant; this attempt does not silently repeat its one Claude
conversation. Preserve C16 surface gaps. Independent Review cannot replace them.
