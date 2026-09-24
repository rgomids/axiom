# Evidence — MVP Slice S5: Strict Selectors and Codex Convergence

## Claim and authority boundary

This record covers Specification 004 Slice S5 only: T14–T15. The final real
Codex Runtime observations ran against clean implementation revision
`8fa043f35a280ae5f5b27266954ea1bfa2d43f8f` on branch
`agent/s5-cli-codex-selectors`. Documentation reconciliation follows as a
separate revision and does not change the exercised Runtime, CLI, or application
behavior.

The first real seven-status run found a T15 defect: Codex synthesized `details`
from non-canonical `draft` and `workItem` payloads and emitted `details: null`
when Lingo omitted that top-level field. Revision
`72a1110e3c9c18fd241abed96333c09fadf649c4` made every installed skill copy only
canonical top-level fields, omit absent fields, and never map operation payloads
into them. Revision `8fa043f35a280ae5f5b27266954ea1bfa2d43f8f`
completed controlled upgrade of the exact prior Axiom-owned v2 receipt while
continuing to reject foreign receipts. Final Runtime Evidence was regenerated
after both corrections.

No real GitHub mutation, release/prerelease, S6 work, merge, Issue closure, or
human MVP acceptance occurred. Local mutations were confined to synthetic
temporary roots. Provider behavior used only a local fake executable.

## Environment

| Fact | Observation |
|---|---|
| Repository | `rgomids/axiom` |
| Branch | `agent/s5-cli-codex-selectors` |
| Exercised commit | `8fa043f35a280ae5f5b27266954ea1bfa2d43f8f` |
| Source state | `clean` during build and all final Runtime observations |
| PR relation at observation | PR #93 branch; local branch two commits ahead of then-remote head `c95a071ad0dc6f05425ba8f540e1a9692f814c87` |
| macOS | 27.0, build `26A428` |
| Architecture | `arm64` |
| Codex CLI | `codex-cli 0.155.1` |
| Model | `gpt-5.6-luna` |
| Reasoning | `low` through `-c 'model_reasoning_effort="low"'` |
| Evidence root | `/tmp/axiom-s5-evidence-c95a071/final` |

`codex exec --help` for the installed CLI exposed `--model`, `--config`,
`--ephemeral`, `--ignore-user-config`, `--ignore-rules`, `--strict-config`,
`--sandbox`, `--add-dir`, `--cd`, `--json`, and `--output-last-message`. The
chosen model/reasoning combination completed every final invocation. No fallback
model was used.

Isolation used an unrelated working directory, isolated `HOME`, isolated
Project/state/skill roots, an ephemeral Codex session, explicit writable temp
roots, and the host's existing `CODEX_HOME` only for authentication. Authentication
material was neither copied nor captured. Captures contain synthetic prompts,
JSONL events, stdout/stderr, last messages, and exit files; they contain no token,
cookie, API key, raw credential, or real Provider response.

The final installed skill-set v2 digests were:

| Skill | SHA-256 |
|---|---|
| `axiom-project-configure` | `d5271f6a24676c2f8111776cc797232784ad7e75318aca500f6ad73274510b60` |
| `axiom-project-show` | `2542254b45ef2c1ac67e09ae1d1924fd0648787836b9bbbe60480a6f09646bcc` |
| `axiom-work-item-create` | `8ecdd0553a999372522f7bc7ad0663e8474af7045f997c718e9c82e4799c5db3` |
| `axiom-work-item-run` | `b5ca1ecf4dd136ba5baa6c647b19080d2d129d539e31b27573e43694ae40982f` |
| `axiom-work-item-status` | `9fcfd0f9caf3a208e54d65cefab81d372cf1fa79c32ba3e1fe8efbc63d7f1990` |

## Real Codex invocation

Every final process used this host envelope, with the per-case temp roots and
prompt substituted explicitly:

```bash
HOME=<isolated-home> \
CODEX_HOME=<existing-host-codex-home> \
PATH=<head-built-lingo-bin>:$PATH \
LINGO_PROJECTS_ROOT=<isolated-projects> \
LINGO_STATE_ROOT=<isolated-state> \
AXIOM_CODEX_SKILLS_ROOT=<isolated-home>/.agents/skills \
codex exec \
  --ephemeral \
  --ignore-user-config \
  --ignore-rules \
  --strict-config \
  --sandbox workspace-write \
  --skip-git-repo-check \
  --add-dir <bounded-temp-root> \
  --cd <unrelated-cwd> \
  --model gpt-5.6-luna \
  -c 'model_reasoning_effort="low"' \
  --json \
  --output-last-message <capture>.last.txt \
  '<sanitized synthetic prompt>'
```

### Fully specified

The prompt supplied Project `dogfood-project`, Repository `main`, Work Item
`github:owner/repo#7`, and Execution
`319b5eea-1db6-4980-8259-8294a65e18ad`. Codex read the installed
`axiom-work-item-status` skill and ran exactly:

```bash
lingo --json workflow status \
  --project dogfood-project \
  --repository main \
  --work-item github:owner/repo#7 \
  --execution 319b5eea-1db6-4980-8259-8294a65e18ad
```

Question count was zero. Lingo exited `0`; Codex exited `0`. The final canonical
result was `success`, `Execution workflow operation completed`, reference
`execution:319b5eea-1db6-4980-8259-8294a65e18ad`, and provenance
`Axiom/development/8fa043f35a28/clean`. Event capture:
`final/full-input.stdout`; final response: `final/full-input.last.txt`.

### Partial selector

The prompt supplied the same Project, Repository, and Work Item while intentionally
omitting Execution. Codex asked exactly one question: `Execution selector is
required to inspect status. Please provide the execution ID.` It did not ask for
any supplied selector. Its only command read the installed skill; it did not
invoke Lingo. Therefore no CWD, Git remote, Provider, Runtime history, or global
identity fallback occurred. Captures: `final/partial.stdout` and
`final/partial.last.txt`.

## Seven-status semantic matrix

Each direct Lingo result was normalized to the presence and value of only
`status`, `result`, `references`, `next`, `details`, and `provenance`. The Codex
last message was normalized the same way and byte-compared with `cmp`. Missing
fields remained missing; no empty, null, or operation-specific substitute was
accepted. Canonical comparison files are
`final/status-<status>.{direct,codex}.canonical.json`.

| Status | Direct Lingo | Codex/Skill | Semantic match | Evidence |
|---|---|---|---|---|
| `success` | exit `0`; completed status read | Codex `0`; inner Lingo `0`; `axiom-work-item-status` | yes | `status-success.direct.json`, `full-input.*` |
| `failure` | exit `1`; invalid protected state root caused setup inspection failure | Codex `0`; inner Lingo `1`; `axiom-project-configure` | yes | `status-failure.{direct, codex*}` |
| `validation_failure` | exit `1`; unknown selector rejected before dispatch | Codex `0`; inner Lingo `1`; `axiom-work-item-status` | yes | `status-validation_failure.direct.json`, `validation-unknown.*` |
| `denied_authority` | exit `1`; stale Work Item digest denied before Provider call | Codex `0`; inner Lingo `1`; `axiom-work-item-create` | yes | `status-denied_authority.{direct, codex*}` |
| `partial` | exit `1`; fake Provider effect confirmed, controlled local linkage write denied | Codex `0`; inner Lingo `1`; `axiom-work-item-create` | yes | `status-partial.{direct, codex*}` |
| `interrupted` | exit `2`; local workflow `implementation` outcome `fail` committed revision 7 | Codex `0`; inner Lingo `2`; `axiom-work-item-run` | yes | `status-interrupted.{direct, codex*}` |
| `retryable_failure` | exit `1`; configured repository binding temporarily unavailable | Codex `0`; inner Lingo `1`; `axiom-project-show` | yes | `status-retryable_failure.{direct, codex*}` |

The `partial` row used a fake `gh` executable that returned synthetic Issue `8`
and then made the isolated local Work Item directory unwritable. This deterministically
proved confirmed external-effect truth plus incomplete local persistence without
network access. Permissions were restored after capture. The `retryable_failure`
row moved only the isolated repository binding and restored it with a shell trap.
The `interrupted` row used separate direct and Runtime copies of the revision-6
Execution fixture. These seams changed no repository file and contacted no real
Provider.

## Invalid selector matrix

| Case | Runtime observation | Lingo invocation | Result/effect |
|---|---|---|---|
| unknown | zero questions | exact command plus `--unknown-selector synthetic`; inner exit `1` | `validation_failure`; no application dispatch/effect |
| duplicate | zero questions | exact duplicate `--project dogfood-project`; inner exit `1` | `validation_failure`; no application dispatch/effect |
| conflict | zero questions | exact `--work-item ... --number 7`; inner exit `1` | `validation_failure`; no application dispatch/effect |
| missing Execution | exactly one Execution question | none | no Lingo/application/Provider effect |
| ambiguous Project slug | not exercised through real Codex in this run | none | deterministic protected-state integration fixture remains coverage; no Runtime claim made |

The three invalid Runtime event logs show no discovery or fallback command beyond
reading the installed skill when needed and invoking the exact explicit Lingo
command. Their canonical outputs were identical: `validation_failure`, result
`Explicit selector input is invalid`, next action `Remove unknown, duplicate, or
conflicting inputs and retry`, and clean head provenance.

## Commands and validation

Environment and host contract discovery:

```text
git status --short --branch
git rev-parse HEAD
gh pr view 93 --json ...
codex --version
codex exec --help
sw_vers
uname -m
```

Runtime Evidence used direct `lingo --json` commands paired with the real
`codex exec` envelope above. Raw final captures remain outside the repository at
`/tmp/axiom-s5-evidence-c95a071/final`; pre-remediation captures remain in the
parent temp root and are not used for final claims.

Final validation on clean `8fa043f35a280ae5f5b27266954ea1bfa2d43f8f`:

| Command | Exit | Observation |
|---|---:|---|
| `go test ./...` | 0 | all packages pass |
| `go test -race ./...` | 0 | all applicable packages pass |
| `go vet ./...` | 0 | no diagnostics |
| `go build ./...` | 0 | all targets build |
| `go mod verify` | 0 | all modules verified |
| `./scripts/dogfood-poc.sh` | 0 | isolated fake-Provider journey completed at workflow revision 13 |
| `./scripts/validate-repository.sh .` | 0 | repository, agent package, validator, and sensitive-file checks pass |
| `./scripts/check-sensitive-files.sh .` | 0 | worktree scan passes |
| `gitleaks detect --source . --no-git --redact` | 0 | no leaks found |
| `git diff --check` | 0 | no whitespace errors |

## Architecture and security review

- Lingo/application remains completion authority; skills only select and copy
  canonical top-level fields.
- Selector validity grants no mutation authority. Stale authority failed before
  Provider access.
- Exact prior Axiom-owned skill and receipt digests can upgrade; unknown content
  remains non-overwritable.
- Runtime roots, fake Provider, HOME, skills, working directory, and captures were
  bounded to temporary synthetic paths.
- No new architecture or ADR was required. The remediation tightened the already
  approved thin-adapter contract.

## Limitations and next gate

- Real Codex covered fully specified, missing-only, unknown, duplicate, conflict,
  and all seven terminal statuses. Ambiguous Project-slug behavior remains
  deterministic integration Evidence, not a real Codex observation in this run.
- Codex process exit `0` means the agent completed rendering; inner Lingo exits
  `0`, `1`, and `2` remain visible in command events and are recorded above.
- Temp captures are reproducible but intentionally not versioned as large logs.
- No real GitHub mutation was authorized or attempted.
- At documentation time, the two remediation commits were local and not yet on
  PR #93's remote branch. Remote PR readiness and CI therefore require a separate
  authorized push; no merge or human acceptance is inferred here.
