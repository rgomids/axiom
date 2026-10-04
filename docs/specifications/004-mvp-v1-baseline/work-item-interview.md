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

The scripted scenario models Runtime reasoning; it does not claim an autonomous
model interview was measured. A human Runtime walkthrough should start with a
single sentence, answer only material questions, correct one proposed assumption,
inspect the complete new preview, and decline creation to observe the boundary.
No live external Issue creation is needed for this information-acquisition change.

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

The final source was checked for whitespace errors. Tests use synthetic Provider
targets and temporary state. No live external Issue was created. Native macOS,
remote CI, and a human Runtime interview are not inferred from these local
checks. The dedicated gitleaks scanner was unavailable; repository sensitive-file
checks and staged-content review remain the executed security evidence.
Human acceptance and release publication remain separate decisions.
