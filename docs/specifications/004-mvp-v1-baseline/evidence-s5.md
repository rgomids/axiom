# Evidence — MVP Slice S5: Strict Selectors and Codex Convergence

## Claim and authority boundary

This record covers the explicitly authorized Specification 004 Slice S5 only:
T14–T15. Work started from clean `main` revision
`f32a345eac2f19dfef91a460a4db9cc8928bd454`, the PR #91 merge, on branch
`agent/s5-cli-codex-selectors`.

The validated implementation and tests are committed at
`04ebccde6ce8e5dfdcec2f259b95e079fc76ec4a`. Documentation/Evidence reconciliation
follows separately so this record can name that immutable implementation revision.

No real Provider mutation, release/prerelease action, S6 work, Issue mutation,
or human MVP acceptance was performed or inferred. The real Codex observations
used isolated temporary Project, state, HOME, skill, and working roots; Codex and
Lingo were restricted to read-only status behavior and no GitHub command ran.

## T14 — strict CLI selector path

The canonical selector vocabulary is:

```text
--project <exact Project UUID or installation-unique slug>
--repository <exact Project-scoped Repository key>
--work-item github:<owner>/<repository>#<number>
--execution <exact Execution ID, except workflow start>
```

The CLI rejects unknown, duplicate, conflicting, or malformed inputs before
application dispatch. Scalar flags are single-valued. `--work-item` conflicts
with the pre-S5 split target flags instead of silently winning. Shell-like
characters are rejected as selector data and never interpolated.

The application path revalidates Project resolution, Repository membership,
exact linked Provider resource/external ID, and applicable Execution identity.
Workflow start rejects a supplied Execution identity; later workflow operations
reject a mismatched identity. Selector validity grants no mutation authority.

### Deterministic selector observations

| Case | Expected/observed result | Prompts | Effects |
|---|---|---:|---:|
| Project slug and UUID | Same installed Project/application path | 0 | read-only |
| Repository `main` in selected Project | Exact scoped binding | 0 | read-only |
| `github:owner/repo#7` | Exact protected local link | 0 | read-only |
| Exact matching Execution | Canonical `success` status result | 0 | read-only |
| Missing Execution only | Only `Execution ID:` | 1 | read-only |
| Unknown flag | `validation_failure` | 0 | 0 |
| Duplicate `--project` | `validation_failure` | 0 | 0 |
| `--work-item` plus `--number` | `validation_failure` | 0 | 0 |
| Malformed/shell-like Work Item | `validation_failure` | 0 | 0 |
| Unknown Repository/Work Item | `validation_failure` | 0 | 0 |
| Mismatched Execution | `validation_failure` | 0 | 0 |
| Unrelated CWD | Same explicit resolution | 0 | read-only |

The executable black-box test byte-compares the protected Execution record before
and after the invalid selector matrix. Bytes remain identical. The fake Provider
ledger receives no call for these read-only or parser-failure cases.

`internal/cli` recording-service tests also compare parsed full selectors with
the direct operation-shaped application input. The CLI adds presentation syntax;
it does not introduce a second resolver or workflow use case.

## T15 — thin Codex path

### Installed skill identity

Isolated installation reported skill set `2`, binary compatibility `2`, and all
five skills as `equivalent`:

| Skill | SHA-256 |
|---|---|
| `axiom-project-configure` | `05d8e420f440529df3bd75a521f3d9493d5cefe3d9fc16ddb1da9ffeed553cd1` |
| `axiom-project-show` | `d7f86666dd2036b53a4cdbe2d6b67d096936f59b164ae9573806fbb9a40d97fd` |
| `axiom-work-item-create` | `9b6d28569d02a97ff0273d08a9abd6bc70ec573050c2bd9f3cc5dd40e984fcaf` |
| `axiom-work-item-run` | `49d269602abedde05dc357135dc9262f6146bccc87cb97784790659f5eed37a4` |
| `axiom-work-item-status` | `9f4d5063347eb47ef38d7c7789f27fb13f3a224880e53915080b0ba9bbe8ec5d` |

Known Axiom-owned v1 digests remain recognized only for controlled installer
upgrade. Foreign or modified content remains non-overwritable. Deterministic
tests also prove missing, partial, equivalent, owned-older, modified/foreign, and
binary-incompatible states.

### Host contract discovery and bounded real invocation

Host observed: `codex-cli 0.155.1`; standalone skill directory
`$HOME/.agents/skills/<skill>/SKILL.md`; invocation through `$axiom-work-item-status`.
The run used `codex exec --ephemeral --ignore-user-config --ignore-rules
--sandbox read-only --skip-git-repo-check` from an unrelated isolated directory.
Authentication remained in the host's existing Codex home and was neither copied
nor recorded. Evidence contains no credentials, raw chat history, or unrestricted
model output.

Fully specified observation:

- Codex discovered and read the isolated v2 skill.
- It executed exactly `lingo --json workflow status` with the supplied Project,
  Repository, Work Item, and Execution selectors.
- It asked zero questions.
- Lingo exited `1` with canonical `validation_failure` because the isolated state
  intentionally contained no selected Project; Codex process exited `0` after
  rendering the same canonical fields.
- The isolated binary truthfully reported development provenance at base revision
  `f32a345eac2f` with `dirty` source state because the bounded observation ran
  before the implementation commit; it did not invent the later commit identity.
- Result/provenance meaning was preserved; no Provider command or mutation ran.

Partial observation:

- Project, Repository, and Work Item were supplied; Execution was absent.
- Codex asked exactly one question: `Qual é o ID da execução (--execution)?`
- It did not invoke Lingo and did not infer Execution from CWD, Git, Provider,
  Runtime history, or global discovery.

### CLI/Codex semantic matrix

The shared `completion.Result` and renderer golden matrix owns classification;
skills only forward JSON and render its fields.

| Canonical status | CLI meaning | Codex meaning |
|---|---|---|
| `success` | Requested result/effects confirmed | Same canonical field |
| `failure` | Non-validation, non-retryable failure | Same canonical field |
| `validation_failure` | Selector/state/schema/invariant rejected | Same canonical field |
| `denied_authority` | Exact authority absent/refused/stale | Same canonical field |
| `partial` | Confirmed effects plus incomplete work | Same canonical field/references/next |
| `interrupted` | Persisted truth preserved before terminal completion | Same canonical field/references/next |
| `retryable_failure` | Named bounded safe retry boundary | Same canonical field/references/next |

Human and JSON renderers are golden-tested for all seven rows. Stable `details`
references are regression-tested in both formats. Artifact create/read, digest,
identity, confinement, and sanitized Markdown lookup remain owned by the T02
artifact boundary; skills report the canonical reference without copying or
interpreting artifact content. Provenance is central and unchanged transported
user text remains user-authored.

## Commands executed

All listed commands completed with exit `0` unless an expected inner CLI failure
is stated:

```text
git fetch origin --prune
git merge --ff-only origin/main
gh pr view 91 --json number,state,mergedAt,mergeCommit,url,title
gh issue view 79 --json number,state,title,body,url
gh issue view 15 --json number,state,title,body,url
go test ./internal/cli ./internal/workflow ./internal/workitem ./cmd/lingo
go test ./internal/codexruntime ./cmd/lingo ./internal/cli ./internal/workflow ./internal/workitem
go test ./...
git diff --check
go build -o <isolated-root>/lingo ./cmd/lingo
<isolated-env> lingo --json runtime codex install
<isolated-env> lingo --json runtime codex status
<isolated-env> codex exec --ephemeral --ignore-user-config --ignore-rules --sandbox read-only ... <fully-specified prompt>
<isolated-env> codex exec --ephemeral --ignore-user-config --ignore-rules --sandbox read-only ... <partial prompt>
```

Final repository-wide validation after documentation reconciliation:

| Command | Exit | Observation |
|---|---:|---|
| `go test ./...` | 0 | all packages pass |
| `go test -race ./...` | 0 | all applicable packages pass |
| `go vet ./...` | 0 | no diagnostics |
| `go build ./...` | 0 | all packages/build targets compile |
| `go mod verify` | 0 | all modules verified |
| `./scripts/dogfood-poc.sh` | 0 | isolated fake-Provider journey completed at workflow revision 13; historical path remained compatible |
| `./scripts/validate-repository.sh .` | 0 | repository, agent package, validator behavior, and sensitive-file checks pass |
| `./scripts/check-sensitive-files.sh .` | 0 | worktree scan passes |
| `gitleaks detect --source . --no-git --redact` | 0 | no leaks found |
| `git diff --check` | 0 | no whitespace errors |

## Architecture and security review

- `Axiom != Lingo`, `Project != Repository`, `Execution != Agent`, and
  `Skill != workflow truth` remain unchanged.
- Application services and protected stores resolve exact identities; command
  handlers adapt syntax and host prompts only.
- Skills contain no status classifier, persistence/recovery rule, Provider call,
  independent identity resolver, or authority grant.
- Resolution completes before mutation dispatch. Parser/application failures have
  zero side effects and do not consult ambient identity.
- No ADR is required: selector meanings, Lingo ownership, thin skill direction,
  Execution scope, compatibility, and detail ownership were already approved.

## Limitations and next gate

- Real Codex Evidence proves host discovery, zero-question full input, missing-only
  questioning, exact Lingo invocation, and canonical validation rendering. It does
  not replace deterministic success/failure matrices or claim Provider behavior.
- No real GitHub mutation was authorized or attempted in S5.
- Historical split `--number` paths remain executable for pre-release regression,
  but are not promoted as the S5 selector contract.
- Native release-row and clean RC acceptance remain T22/T24 obligations.

S5 is technically ready for review after final repository validation. Human MVP
acceptance and S6 authorization remain separate decisions.
