# Minimal-intent Work Item interview — issue #134

Status: implemented for review; human acceptance remains separate.

## Scope and contract

This follow-up refines FR-008–FR-011 without changing the seven-section domain,
Provider rendering, type/label policy, or workflow execution. It starts from
one sentence through the shared Codex/Claude Runtime skill. Natural-language
extraction stays at the Runtime boundary; Lingo validates canonical sections,
authorship, input budgets, missing questions, and exact mutation authority.
No model, paid service, or inference dependency is added to the CLI.

The Runtime reuses facts already in the conversation, explains missing concepts
with plain questions, and proposes safe draft content only when an unknown
would not materially change delivery. Unknown environments, actors, solutions,
deadlines, and commitments must not be invented. Material gaps require answers.
"No additional context supplied" records the limits of available information;
it is not evidence that no relevant context exists. Concrete acceptance checks
must follow the agreed observable outcome and remain useful to Specification.

A complete final preview contains problem, desired outcome, context, scope,
constraints, non-goals, and acceptance expectations, plus attribution, target,
effects, expected local revision, and digest. All assumptions are reviewed.
Human draft validation precedes explicit external authority for the exact
preview. Changed facts or attribution invalidate authority. Cancellation,
incomplete input, rejected input, and stale authority cause no Provider create.

## Implementation plan and boundaries

- Refine the embedded shared create skill with extraction, focused questions,
  bounded assumptions, complete review, and unchanged Lingo authority.
- Expose the existing `SectionInput.Elaborated` through repeatable CLI
  `--elaborated-section name=content`; never misattribute Runtime text to users.
- Reuse both supplied and elaborated sections in terminal prompting and retain
  the plain terminal interview as a deterministic fallback without inference.
- Reject conflicting authorship sources instead of silently discarding one.
- Register the previous shared skill revision for safe Codex/Claude upgrades.
- Verify parser, composed preview, authored sections, concrete acceptance,
  cancellation, stale authority, security checks, and shared Runtime ownership.

No new ADR is needed: the neutral domain and thin Runtime/Provider boundaries
remain unchanged. Rollback is a revert of these scoped changes; any subsequently
published skill revision must remain in shared ownership history.

## Acceptance scenarios and evidence

| Issue criterion | Scenario / evidence |
| --- | --- |
| One sentence yields a complete reviewed draft | `TestMinimalIntentInterviewProducesReusableDraftWithoutMutation` starts with "Export fails when I select multiple rows.", receives six missing questions, adds conversational answers and bounded proposals, and renders all seven sections. |
| Supplied information is reused | Original problem, observable outcome, and acceptance text retain user authorship; `TestInterviewReusesSuppliedAndElaboratedSectionsBeforeExactAuthority` asks only for the missing acceptance check. |
| Missing information is conversational | Shared skill asks about observable changes, occurrence, and checking success in the user's language; terminal prompts explain missing concepts. Runtime semantic interview quality needs human observation, not just string assertions. |
| Final draft validated before mutation | CLI preview precedes authority; yes binds the exact digest, no/empty answer cancels; composed stale/corrected previews leave local state and Provider ledger untouched. Existing domain exact-authority tests cover the authorized create. |
| Useful upstream context | Composed renderer retains every canonical section, concrete multi-row export acceptance, bounded scope, unknown context, exclusions, and correct attribution. |

The scripted scenario models Runtime reasoning. The additional bounded agent
observation below exercises model-generated questions and elaboration, with a
synthetic user and isolated Provider. Neither substitutes for human acceptance
or proves discovery and behavior in every installed Runtime. No live external
Issue creation is needed for this information-acquisition change.

### Internal behavioral review, 2026-10-04

A separate Codex agent read the exact versioned create skill and conducted a
Portuguese interview. The controller supplied synthetic user replies and
executed the agent's argument vectors against the real composed Lingo binary.
The binary used an isolated Project and a fake GitHub executable that only
served the label catalog and rejected mutations. All ten changed code, test,
and skill files in the Linux build copy matched the reviewed worktree bytes.
This was a controlled agent conversation, not an installed CLI discovery test
or an independent human usability study.

| Step | Observed behavior |
| --- | --- |
| Minimal intent | Started from "O CSV perde linhas quando exporto vários pedidos." Asked where export happens and for a concrete example, rather than requesting seven schema fields. |
| Existing facts | User supplied 12 selected orders, 10 exported orders, the desired one-row-per-order outcome, unchanged columns/format, and no PDF or screen changes. The agent reused these facts without repeating their questions. |
| Explain verification | When asked what verification meant, the agent explained selecting 12 orders and checking all 12 in the CSV. It clarified whether the header counted and whether other selections were affected. |
| Complete preview | Lingo returned all seven sections. Verbatim outcome, constraints and exclusions retained user authorship; synthesized context, scope and checks had Axiom authorship. The agent presented the complete returned JSON, including target, labels, effects, expected revision and digest, before any authority. |
| User correction | A synthetic correction limited the failure to more than ten orders. The agent preserved the original intent verbatim, updated problem/context/checks, marked the combined problem as elaboration, and requested a fresh preview without external authority. Lingo returned a different digest. |
| Review without publication | After the agent presented the corrected complete preview, the synthetic user approved the draft but explicitly declined publication. The agent ended without requesting another command. Project/state file hashes remained unchanged; the Provider ledger contained exactly two label-catalog reads and no mutation. |

One recoverable invocation error occurred: the agent initially invented
`--github-repository`. Lingo rejected the unknown flag without mutation; after
consulting the command documentation, the agent retried with the documented
`--provider-repository`. This is an observed Runtime reliability limitation,
not evidence that the first attempt always succeeds.

An additional independent adversarial review found no blocking regression in
CLI parsing, section attribution, authority, cancellation, or shared skill
upgrades. It re-ran the native Windows CLI/domain tests successfully. Delegation
was bounded to an independent reviewer and a fresh interview agent, both using
inherited model/effort; the main reviewer executed commands and checked the
returned previews, file hashes and Provider ledger.

## Validation

Observed on 2026-10-04 with Go 1.26.0, against base `453144f`:

| Check | Observed result |
| --- | --- |
| Linux/WSL `go test -race ./...` | Passed, including the composed interview and shared Runtime ownership tests. |
| Linux `go vet ./...`, `go build ./...`, `go mod verify` | Passed. |
| Linux `./scripts/validate-repository.sh .` | Passed; sensitive files, maintainer harness, bootstrap invariants and automation inventory validated. Its optional live-runtime scenarios were explicitly skipped. |
| Linux `./scripts/dogfood-poc.sh` | Passed; bounded synthetic Provider workflow completed at revision 17. |
| Windows `go test ./internal/cli ./internal/workitem -count=1` | Passed. |
| Windows `go vet ./...`, `go build ./...`, `go mod verify` | Passed. |
| Windows `go test ./... -timeout 10m` | Failed in filesystem/storage and Runtime-installation tests, including portable Project publication. |
| Clean Windows base `453144f`: `go test ./cmd/lingo -run '^TestExecutableGuidedProjectConfiguration$' -count=1` | Reproduced the same `Portable Project publication failed` symptom without this change. Other native-suite failures were not individually isolated. |
| Independent bounded review | No blocking findings; reviewed scope, authorship, exact authority, tests, and shared upgrade history. |
| Remote CI at implementation commit `f59e0b8` | Linux, macOS, Windows, release-contract and delivery-metadata passed; [CI run](https://github.com/rgomids/axiom/actions/runs/37231134100). |
| Native Git Bash repository check during review | Stopped because the local Claude skill adapter is checked out as a copy rather than a symlink; repository checks use the native Linux validation copy. |

The final source was checked for whitespace errors. Tests use synthetic Provider
targets and temporary state. No live external Issue was created. A human Runtime
interview is not inferred from automated checks or the synthetic-user agent
observation. The dedicated gitleaks scanner was unavailable; repository sensitive-file
checks and staged-content review remain the executed security evidence.
Human acceptance and release publication remain separate decisions.
