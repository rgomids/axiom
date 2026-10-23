# AXM-39 — Presentation examples and verification protocol

All examples are synthetic expectations for [Specification 012](spec.md), not
captured conversations, canonical payload fixtures or executed product tests.
Origins and authority follow P1–P2, S1–S2, R1–R3 and L1. No new wire keys,
commands, models, Runtime defaults or installation prerequisites are defined.

## Content regression cases

Synthetic Project IDs:
`11111111-1111-4111-8111-111111111111` and
`11111111-1111-4111-8111-111111111112`.
These intentionally share long prefixes: a short UUID cannot disambiguate them.

| Case | Canonical facts supplied to future test | Required content / forbidden inference |
| --- | --- | --- |
| C01 | name `Órbita`, slug `orbita`, unique | `Órbita (orbita)`; no UUID needed in routine narrative |
| C02 | name only `Órbita`; then slug only `orbita` | Existing human identifier only; no invented second field |
| C03 | Neither human field; first full ID | `Project sem nome/slug`, full ID, reason for fallback |
| C04 | Both full IDs, identical display pair; then collision on name-only query | Both exact IDs and labels; unresolved sensitive target until exact selection; no first-match/prefix choice |
| C05 | Unique human label; user asks full ID; copyable operation/preview/Evidence has exact selector/digest | Full requested ID; authority-bearing source bytes unchanged, including full digest |
| C06 | Partial inventory or missing canonical ID; escaped control characters in label | Uncertainty explicit; no generated ID or claim of global uniqueness; display text never executes |
| C07 | Two inspected Stages, with explicit I/O, gates, owners and criteria | Both summaries include essentials; advanced details retrievable, not fabricated from Stage names |
| C08 | Validator unavailable; timeout/limit or effect ceiling blocks work | Exact risky constraint appears initially, despite advanced classification; other requested details remain accessible |
| C09 | Agent allows Codex/Claude; `profileRef: policy-default`; Project declares `codex-test`; no resolved plan | Allowed/declaration stated, selection/resolution not established; no concrete model, effort or host-based preference |
| C10 | Matching resolved plan; separate successful login observation; no attempt | Cite distinct sources; resolution/login do not establish execution, compatibility, billing or acceptance |
| C11 | Success result, structurally valid, blocked with `runtime_policy_unavailable / policy_unconfigured`, confirmed no dispatch | Example below; do not suggest absent certification/probe as remedy |
| C12 | Prior attempt with unknown effects, multiple blockers, timeout or invalid result despite exit 0 | All material facts survive summary; reconcile before retry; no “nothing ran” without evidence |
| C13 | Stale/conflicting references, inaccessible detail or truncated result | Gap and smallest safe inspection; no merged/current-state fiction |
| C14 | Unknown Runtime version; otherwise admissible configuration and authority | No version registry/probe/qualification demand; no promised success |
| C15 | Sequential/DAG Workflow, independent agents, renderer unavailable | AXM-38 identity/order/dependencies preserved; text fallback, no claim of visual rendering |
| C16 | Same facts across natural entry and explicit skill, narrow/plain-text surfaces, Unicode labels | Equivalent identity, state, blocker and gate meaning; no color/hover dependency |

Future deterministic tests compare required facts, forbidden claims and exact
authority-bearing strings against canonical inputs, rather than one mandatory
wording snapshot. Each case needs positive assertions and an omission/invention
negative (for example, reject C09 if it says “Codex selecionado”). Use existing
content-test conventions at implementation planning; no new registry or test
framework selected by Specify. Matrix is a reproducible oracle, not proof tests
exist or passed. C01–C06 apply to conversational `project list/show/validate`
and natural-language follow-up wherever relevant facts are available.

## Representative expected turns

**Identity:** “Quais Projects existem?”

> Órbita (orbita).

**Collision follow-up:** “Use Órbita.” (two matching candidates)

> Há dois Projects com esse nome. Para identificar o alvo com segurança:
> Órbita (orbita) — ID: 11111111-1111-4111-8111-111111111111;
> Órbita (orbita) — ID: 11111111-1111-4111-8111-111111111112.
> Indique qual ID corresponde ao Project desejado.

**Runtime:** “O implementation já usa Codex?” (C09)

> A definição permite Codex e Claude. O Agent declara `policy-default`;
> o Project declara `codex-test`. Esses dados não confirmam seleção nem
> resolução para este Stage. Execução não observada nesta fonte.

**Blocked validation:** “Está pronto?” (C11)

> A validação foi concluída com sucesso e a estrutura é válida. A execução
> NÃO está pronta e não foi realizada: `runtime_policy_unavailable /
> policy_unconfigured`. Falta policy de Runtime para este Project. Próxima ação:
> inspecionar a configuração e a operação instalada de edição de policy;
> qualquer alteração exige preview e authority próprios. Esta consulta não
> autorizou execução ou inferência.

**Stage:** “Explique intake e implementation.” (C07)

For each actual inspected Stage, summarize purpose, I/O meaning, gate kind/timing,
Agent/Role, allowed/declared/resolved Runtime/Profile and completion criterion.
If inspection did not return those facts, state the gap. Do not supply a canned
Intake gate or claim implementation tests passed merely from the Stage ID.
On “detalhes do Agent”, recover declared validators, retries, limits, timeout,
concurrency, effect ceilings and failure policy from the same exact revision.
Any blocking validator/authority restriction appears in the first answer.

## Two native conversations — future protocol, NOT_RUN

Origin: CARD AC5 and ACCEPTED AXM-12 ownership as linked by [Spec 012](spec.md).
One conversation in native Codex, one in native Claude, each using a compatible
installed binary/packaged skill pair outside a maintainer checkout. Product
operation must not require the maintainer's personal paths or source tree.
Identify exact host/surface/build, installed Axiom version/content identity,
skill revision and canonical Project/Workflow references before interpreting
results. Unknown compatibility/installation remains unknown.

1. Check dependency delivery, actual installed pair and authorization for the
   concrete session/reads. Real Runtime invocation, inference, credentials,
   cost or external effects require separate specific authority. Otherwise
   retain NOT_RUN; never reuse another user's authorization.
2. Begin naturally: list Projects, inspect/validate one, request full ID and
   exercise an available safe collision/absence example. Label any replayed
   input as synthetic; it cannot prove an installed collision was observed.
3. Inspect the exact Workflow, then `intake`, `implementation` and an Agent;
   ask essential and advanced questions. Check diagram/text consistency and
   risk promotion, distinguishing rendering observations from text assertions.
4. Ask allowed vs selected/resolved Runtime/Profile, including `policy-default`
   and a declared `codex-test` when the source actually contains them. Exercise
   blocked/unknown and successful-read cases without dispatching a Stage merely
   to make a positive presentation example.
5. Record sanitized input/result refs and visible answer excerpts, observed
   behavior, failed/untested cases and reviewer verdict per AC/case. Keep exact
   canonical IDs/digests in safe authority records; publish only classified
   material. No raw credentials, host paths or full private chat export.

Evidence record per conversation: host/build, date, installed binary/skills
identity, authorization reference, canonical input reference/revision, case/AC,
sanitized visible response, result PASS/FAIL/NOT_RUN, gaps and next action.
These are Evidence metadata, not a new product store/schema. Natural-language
fluency and semantic fidelity need native review; deterministic content tests
alone do not prove them. Failed/partial conversations remain evidence, never
silently replaced by a successful synthetic transcript. Neither conversation
establishes AXM-12 inference/E2E acceptance or generic vendor capability.
