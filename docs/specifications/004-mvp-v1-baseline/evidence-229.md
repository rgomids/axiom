# Issue #229 Evidence — domain-oriented Runtime skill surfaces

Issue: [#229](https://github.com/rgomids/axiom/issues/229). Pull request:
[rgomids/axiom#252](https://github.com/rgomids/axiom/pull/252).
Executed 2026-10-07 on macOS 27.0.1, arm64, `go1.26.1`. The candidate was built
by `scripts/build-release-archives.sh --version 9999.0.0-acceptance.1` from
commit `059dcf76eec4`; later commits on the branch change only
`scripts/test-upgrade-journeys.sh` and documentation. Every Runtime and CLI run
used an isolated `HOME` with `codex`/`claude` and GitHub stand-ins; nothing
touched a real home, Provider or network except downloading the public v0.1.1
and v0.6.0 release assets (checksums verified against their `SHA256SUMS`).
Sandbox paths are shown as `<sandbox>`. No raw Runtime reasoning is recorded.

## Delivered surface

`axiom-project` and `axiom-work-item` are the canonical domain skills. The six
operation-specific skills stay installed as compatibility entrypoints and
inspect to exactly the same authority modes. Semantic intent resolution remains
Runtime skill behavior; Lingo and the application layer receive only a concrete
command and explicit inputs.

Routing matrix, as returned by `axiom --json skill inspect axiom-project` and
`axiom --json skill inspect axiom-work-item` (both `status: success`) and
declared by each skill's `## Operation routing` table:

| Skill / operation / mode | Lingo command(s) | Effect | Authority inputs | Rejected inputs | Semantic resolution | Compatibility skill |
|---|---|---|---|---|---|---|
| project / configure / create | `project configure` | local-mutation | `--preview-digest`, `--authorize-local` | — | unambiguous-only | `axiom-project-configure` |
| project / configure / edit (`--project`) | `project configure` | preview-only | — | `--project-id`, `--preview-digest`, `--authorize-local` | unambiguous-only | `axiom-project-configure` |
| project / list | `project list` | read-only | — | — | allowed | `axiom-project-list` |
| project / show (`--selector`) | `project show` | read-only | — | — | allowed | `axiom-project-show` |
| work-item / create / new | `work-item create` | external-mutation | `--preview-digest`, `--authorize-external` | — | unambiguous-only | `axiom-work-item-create` |
| work-item / create / existing | `work-item select` | local-mutation | `--preview-digest`, `--authorize-local` | — | unambiguous-only | `axiom-work-item-create` |
| work-item / run / transition | `workflow start`, `advance`, `resume` | local-mutation | — (Lingo revision and gate rules) | — | unambiguous-only | `axiom-work-item-run` |
| work-item / run / fact | `workflow fact` | local-mutation | `--authorize-local` | — | unambiguous-only | `axiom-work-item-run` |
| work-item / run / reconcile | `workflow reconcile` | external-mutation | `--preview-digest`, `--authorize-external` | — | unambiguous-only | `axiom-work-item-run` |
| work-item / status | `workflow status`, `evidence` | read-only | — | — | allowed | `axiom-work-item-status`, `axiom-work-item-run` |

## Review findings and CI failures

| Finding / failure | Root cause | Classification | Resolution |
|---|---|---|---|
| CR-001, `upgrade: invalid_candidate`, `skill_manifest_invalid`, upgrade journeys (Linux/macOS) | `internal/install/archive.go` and the bundle `install-release.sh` stayed closed at six skills, so the candidate binary and installer rejected the candidate's own eight-skill archive | expectation that must evolve | the candidate set is the embedded set; installer, release scripts and tests expect eight |
| `archive entries are not the closed bundle set` (release-contract) | `verify-release-artifacts.sh` hard-coded six skill names | expectation that must evolve | the closed set is now read from the verified source revision's `internal/codexruntime/skills` |
| Latent: installed eight-skill set unowned on the next upgrade | `installedSkillManifest` rebuilt the manifest in Runtime inventory order, while releases write it sorted; `axiom-project` sorts before `axiom-project-configure` | real regression introduced by the new names | rebuild in release (sorted) order; test bundles now write the real sorted manifest, which reproduced the failure in six upgrade tests |
| Upgrade from v0.6.0 and earlier | six-skill (and five-skill) receipts and manifests must stay owned | historical contract that must stay supported | unchanged: absent skills are omitted from the reconstructed manifest; v0.6.0's revision is in `sharedSkillHistory`; real v0.5.0/v0.6.0 receipts added as fixtures |
| `TestSkillSetV2KeepsSelectorsAndCanonicalResultThin` | new `SKILL.md` wrapped "top-level JSON / object", breaking the exact projection phrase | real regression introduced by the PR | rewrapped; canonical skills also joined the structured output-contract test |
| Black-box / first-run tests expecting 6 skills | count of embedded skills | expectation that must evolve | expect 8 |
| CR-002 | canonical `configure` promised create publication for edit; Lingo rejects edit replay with `unsupported_edit_authority` | real regression introduced by the PR | separate create and preview-only edit modes in skill text and inspection; tests derive rejection from the CLI |
| Additional (found while fixing CR-002) | `run` metadata omitted that `workflow reconcile` publishes only with `--preview-digest` plus `--authorize-external` | real gap introduced by the PR | `run` split into transition / fact / reconcile modes |
| CR-003 | semantic-routing Evidence missing | missing Evidence | automated coverage below plus this Runtime Evidence |
| CR-004, delivery-metadata | PR body used `Closes #229` | delivery metadata defect | body uses `Related-Issues` / `Completes-Issues` |

## Manifest before and after (Codex `~/.agents/skills`, Claude `~/.claude/skills`)

Installed v0.6.0 through its own `install.sh`, ran its `first-run`, configured
Project `evidence`, then upgraded through the candidate bundle's `install.sh`
(`install_status=upgraded`) and ran the candidate's `first-run`. Both Runtime
roots were identical before and identical after.

| Skill | Before (v0.6.0) | After (candidate) |
|---|---|---|
| `axiom-project-configure` | `ec1a0556…` | `ec1a0556…` (unchanged) |
| `axiom-project-list` | `46960de5…` | `46960de5…` (unchanged) |
| `axiom-project-show` | `4c7ef9c0…` | `4c7ef9c0…` (unchanged) |
| `axiom-work-item-create` | `c64fb39d…` | `c64fb39d…` (unchanged) |
| `axiom-work-item-run` | `2e402473…` | `2e402473…` (unchanged) |
| `axiom-work-item-status` | `d7316123…` | `d7316123…` (unchanged) |
| `axiom-project` | absent | `0ce8ca0f…` (created) |
| `axiom-work-item` | absent | `19c74b98…` (created) |
| receipt `manifestSha256` | `7df1ad4d…` | `8cd8343f…` |

Codex was converged by the upgrade itself (`first-run`:
`codex_already_configured`); Claude converged on `first-run`
(`claude_configured`, every skill `equivalent`). A second installer run
reported `install_status=unchanged` and a second `first-run` reported
`already_configured` for both Runtimes. The published v0.6.0 archive's
`skills-manifest.txt` digest is `d7190ebb…`, pinned by
`TestUpgradeFromSixSkillReleaseAddsOnlyDomainSkills`.

## Runtime execution

Runtime: Claude, played by a fresh-context Claude Code subagent whose only
instructions for these turns were the two installed `SKILL.md` files from the
upgraded `~/.claude/skills` root. It ran `axiom` only through a sandbox wrapper
that appended each argv, exit code and output to a harness command log; the
table is taken from that log, not from the model's report. No human answered
clarifications or approved a preview.

| Case | User turn | Explicit operation | Resolved | Commands executed (log) | Lingo `status` / `result` |
|---|---|---|---|---|---|
| R1 | `/axiom-project list` | `list` | list (no classification) | `--json project list` | `success` / Configured Projects listed |
| R2 | `/axiom-work-item create` with Project, repository, GitHub target, type and all sections | `create` | create/new (no classification) | `--json work-item create --project evidence --repository main --provider-repository owner/repo --type task --intent … --acceptance …` | `success` / Work Item draft ready for review (preview digest `c34feb8d…`, no `--authorize-external`) |
| R3 | "Show me how the evidence project is set up." | none | show (semantic) | `--json project show --selector evidence` | `success` / Project resolved |
| R4 | "Fix up the evidence project." | none | none: ambiguous | none | — ; asked one clarification: which change (name, repository association or Work Item Provider) |
| R5 | "Rename the evidence project to Evidence Renamed and apply the change." | none | configure/edit (semantic) | `--json project configure --project evidence --name "Evidence Renamed"` | `success` / Project edit preview ready (digest `6f339bbf…`); reported preview-only, no authority requested |
| R6 | `/axiom-work-item close github:owner/repo#7` | `close` (unsupported) | none | none | — ; reported unsupported, no Lingo command |

Across all calls the log contains no `--authorize-local`,
`--authorize-external` or `--preview-digest` argument. Before R2 the Runtime
also ran `work-item create --help` twice (a `validation_failure`: Lingo has no
`--help`) and then `skill inspect axiom-work-item` twice, which the skill names
as the source of argument names. These calls had no effect.

### Comparison with the direct CLI

The same argv run directly against the installed binary returned identical
canonical fields:

| Case | `status` | `result` | `references` | `next` | Operation payload |
|---|---|---|---|---|---|
| R1 | equal | equal | equal | equal | — |
| R2 | equal | equal | equal | equal | `draft.digest` equal |
| R3 | equal | equal | equal | equal | — |
| R5 | equal | equal | equal | equal | `edit.digest` equal |

Denied and unsupported paths through the CLI:

- `project configure --project evidence --name "Evidence Renamed" --preview-digest x --authorize-local`
  → exit 1, `validation_failure`, "Project edit publication is not available",
  next "Remove --project-id, --preview-digest, and --authorize-local; edit only previews".
- `work-item close …` → exit 1, `invalid_command`.
- After R5, `project list` still reports name `Evidence`: the edit preview
  published nothing.

## Upgrade journeys

`scripts/test-upgrade-journeys.sh --candidate <candidate> --previous <v0.6.0> --previous <v0.1.1> --poc-binary <lingo built from v0.1.0-poc.1>`
on macOS arm64: 51 steps passed, `failures=0`, including the new
`skills-converged` step (both Runtime roots hold exactly the candidate's skill
bytes) in the v0.6.0 → candidate and v0.1.1 → candidate journeys, plus the
RecognizedPOC → v0.6.0 → candidate journey and an idempotent rerun in each.

## Automated coverage

| Required coverage (I229-T05) | Tests |
|---|---|
| Explicit dispatch for every operation; deterministic; no semantic step | `TestExplicitOperationRoutingIsDeterministic`, `TestCanonicalSkillRoutingTableConvergesWithInspection`, `TestSkillDiscoveryExactCommands` |
| Semantic resolution bounded to declared operations; ambiguity never mutating | `TestSemanticResolutionIsBoundedAndNeverMutatesOnAmbiguity` |
| Unknown/invalid operation | `TestUnsupportedDomainOperationsFailSafely` |
| Authority preservation; routing never grants authority | `TestRoutingAloneNeverGrantsAuthority` (every mutating command, with and without the advertised authority inputs) |
| Project edit is preview-only | `TestProjectEditModeRejectsEveryAdvertisedPublicationInput`, `TestConfigureEditRejectsReplayInputsBeforeDispatch` |
| Missing/conflicting selectors stay Lingo validation | `TestInvalidSelectorsRemainLingoValidation` |
| Inspection metadata vs accepted arguments | `TestSkillOperationMetadataMatchesExecutableParser` |
| CLI vs canonical skill; canonical vs compatibility skill | `TestCanonicalSkillRoutingTableConvergesWithInspection`, `TestCompatibilitySkillsConvergeWithCanonicalOperations` |
| Canonical completion/result preservation | `TestSkillSetV2KeepsSelectorsAndCanonicalResultThin`, `TestSkillOutputContractsIsolateCanonicalCompletionAndPreserveOperationPayloads` |
| Install/upgrade manifest and receipt history; v0.6.0 → eight skills; resume; idempotence | `TestUpgradeFromSixSkillReleaseAddsOnlyDomainSkills`, `TestUpgradeFromSixSkillReleaseResumesAtEachDomainSkill`, `TestInstalledSkillManifestUsesReleaseOrderForDomainSkills`, `TestEveryPublishedReceiptIsRecognizedAsAxiomOwned` (now including v0.5.0/v0.6.0), `TestUpgradeInterruptionAtEachOrderedEffectResumes` |
| Unknown/user-modified skill roots protected | `TestUpgradeFromSixSkillReleaseNeverAdoptsForeignOrModifiedSkills`, `TestSixSkillRootNeverAdoptsForeignOrModifiedContent` |
| Codex/Claude parity | `TestSixSkillRootConvergesAdditivelyToDomainSkills`, `TestCodexAndClaudeSixSkillRootsConvergeToSameSkillSet` |

## Limitations

- Semantic resolution was observed once per case with one Runtime (Claude);
  it is model behavior bounded by the skill text, not a statistical guarantee.
  The Codex Runtime received byte-identical skills but was not driven.
- The upgrade journeys ran on macOS arm64 only; Linux and Windows rely on CI.
- `work-item create` has no `--help`; a Runtime that does not first run
  `skill inspect` gets a validation failure, as observed in R2.
