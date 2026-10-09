# Issue #272 / AXM-4 — CLI subscription authentication preflight

Tracking: GitHub [#272](https://github.com/rgomids/axiom/issues/272) (canonical
contract), Linear
[AXM-4](https://linear.app/rgomids/issue/AXM-4/mvp-verify-local-cli-subscription-authentication-before-runtime)
(mirror), parent Epic #15.

## Authority boundary

The 2026-10-08 start request authorized bounded development, local validation
and metadata synchronization. It did not authorize live Codex/Claude inference,
merge, release, Issue closure or human acceptance. Read-only vendor status and
version commands are inspection, not inference.

No new ADR: the change reuses ADR-0003 (Lingo owns policy; skills stay thin),
ADR-0008/0009 (Execution records and the parent/child graph) and the existing
reviewed Runtime/Profile binding. It adds one machine-local check at the
existing dispatch boundary; it changes no persisted Execution record format.

## Contract

`runtimeadapter.AuthPreflight.Check` inspects one effective child invocation:
executable, environment (an invocation without environment inherits the Axiom
process environment, exactly as `executiongraph.OSProcessRunner` does, through
the shared `executiongraph.EffectiveEnvironment`) and working directory.

Order (first failure wins, fail closed):

1. Executable identity (`ExecutableIdentity`), basename and, at dispatch, the
   reviewed `executableDigest`.
2. Environment overrides — names only:
   Codex `OPENAI_*`, `AZURE_OPENAI_*`, `CODEX_API_KEY`;
   Claude `ANTHROPIC_*`, `CLAUDE_CODE_USE_*`, `CLAUDE_CODE_SKIP_*`,
   `CLOUD_ML_*`, `VERTEX_*`, `CLAUDE_CODE_OAUTH_TOKEN`,
   `AWS_BEARER_TOKEN_BEDROCK`, `CLAUDE_CODE_API_KEY_HELPER_TTL_MS`; plus any
   `CODEX_*` / `CLAUDE_CODE_*` name containing `TOKEN`, `API_KEY` or `URL`
   (for example `CLAUDE_CODE_API_KEY_FILE_DESCRIPTOR`).
   Duplicate keys are ambiguous.
3. Configuration overrides — key names only, values never retained:
   Codex `<CODEX_HOME or HOME/.codex>/{config,managed_config}.toml`,
   `<cwd>/.codex/config.toml` and `/etc/codex/{config,managed_config}.toml`
   (`model_provider` ≠ `openai`, any `model_providers`, any `*base_url`,
   `preferred_auth_method` ≠ `chatgpt`, `forced_login_method` ≠ `chatgpt`, and
   inline tables/arrays mentioning those keys);
   Claude `<CLAUDE_CONFIG_DIR or HOME/.claude>/settings.json`,
   `<cwd>/.claude/settings.json`, `<cwd>/.claude/settings.local.json`, the
   platform managed settings and its `managed-settings.d/*.json` drop-ins (`apiKeyHelper`, `awsCredentialExport`,
   `awsAuthRefresh`, `gcpAuthRefresh`, `forceLoginMethod` ≠ `claudeai`, and the
   environment names above under `env`). Unresolvable roots, unreadable or
   unparseable files are `unproven`.
4. `<exe> --version` (validated version token required).
5. Vendor status surface: `codex login status` (exact lines `Logged in using
   ChatGPT`, `Logged in using an API key…`, `Not logged in`) or
   `claude auth status --json` (only `loggedIn`, `authMethod`, `apiProvider`
   are decoded; identity fields are never decoded).
6. Executable identity again: the status commands ran the file.

Status commands get exactly the effective environment, no stdin, a 10 s
timeout and 8 KiB of output that is classified and discarded.

| Status | Meaning | Dispatch |
| --- | --- | --- |
| `subscription_observed` | Vendor status reports ChatGPT / claude.ai first-party login; no override observed | allowed |
| `unproven` | Unrecognized, ambiguous, timed out, unreadable configuration, changed executable | blocked |
| `incompatible` | API-key/provider/helper/config override, API-key login, credential reference | blocked |
| `unavailable` | Not logged in, executable missing | blocked |
| `unsupported` | No recognized status surface on the installed version | blocked |

Every report carries `usability: unproven_until_real_dispatch` and
`revalidation: before_each_dispatch`. A positive status is not usability or
billing proof: vendor status can report a cached login whose token expired,
and on Codex 0.159.1 / Claude 2.1.295 the status surface keeps reporting the
subscription login while `CODEX_API_KEY` / `ANTHROPIC_API_KEY` would take
precedence in `codex exec` / `claude --print`. That is why environment and
configuration are inspected independently (observed read-only 2026-10-08).

## Dispatch integration

`graphapplication.LocalConfiguration.SubscriptionAuth` enables the
subscription scenario. Then `policyInvocations.ResolveInvocation`:

1. keeps the existing reviewed preview → `Check` → binding match (Runtime,
   Model Profile, model, credential reference, executable identity);
2. refuses a credential reference before any credential is resolved;
3. resolves the invocation, re-checks its effective argv with the reviewed
   argument guard, requires a reviewed `executableDigest`, and runs the
   preflight on it bound to that digest;
4. blocks with `executiongraph.ErrAuthenticationBlocked` (scheduler category
   `authentication_blocked`, no attempt, no process, no persisted change); a
   changed executable also wraps `runtimeapplication.ErrStale`.

Each dispatch and retry repeats the check; no earlier report authorizes a later
dispatch. `LocalService.AuthenticationEvidence()` returns the latest sanitized
report per child. Without `SubscriptionAuth`, behavior is unchanged.

`axiom runtime <codex|claude> auth` is the read-only readiness surface: it
checks the executable on `PATH` with the shell's environment and working
directory (what a child without explicit environment would inherit). Only
`subscription_observed` completes; everything else is `validation_failure`.

## Evidence and sanitization

`AuthReport` fields are enums, a validated version token, the executable
digest and environment/configuration key names (`env:NAME`,
`codex_config:<scope>:<key>`, `claude_settings:<scope>:<key>`; unsafe names
become `unnamed`). Raw output, values, paths, credentials and account
identifiers never enter it. `evidenceKind` distinguishes `fake` (scripted
tests) from `local_observation` (real status/version commands). Neither is a
real Runtime probe.

## Known limitations

- The subscription scenario is opt-in (`SubscriptionAuth`). No current Lingo
  command dispatches graph children; only the historical S9 acceptance runner
  composes `LocalService`, and it does not enable it. Making it mandatory is a
  pending human decision.
- Codex `config.toml` is scanned line by line (no TOML dependency adopted);
  multi-line constructs are not interpreted beyond the listed keys, and
  parent-directory project configs are not walked.
- Not inspected: Claude `~/.claude.json` (account state), macOS MDM / Windows
  registry managed preferences, Codex `--profile`/`-c` (already rejected by the reviewed
  argument guard), credential stores, network gateways outside env/config.
- Token expiry, plan limits and actual billing are not observable without a
  real dispatch.
- The window between the final identity check and process start is not
  closed; the scheduler starts the process immediately after the check. The
  executable digest covers the resolved file only, not an interpreter or
  modules behind a script wrapper (for example npm-installed launchers).

## Real probe envelope (Pending — requires separate authorization)

Not executed. Requires an explicit maintainer authorization naming this
envelope. Preconditions: Codex and Claude installed on `PATH`, logged in with
their subscriptions; `axiom runtime codex auth` and `axiom runtime claude auth`
both report `subscription_observed` in the probe shell; no override variables.

Expected identity at preparation (2026-10-08, read-only): Codex `0.159.1`
(method `chatgpt`), Claude `2.1.295` (method `claude.ai`); record the
`executableDigest` from the preflight immediately before the probe.

Invocation boundaries (one call each, empty temporary directory as cwd,
`env -i` with only `HOME` and `PATH`, no credential reference):

```bash
env -i HOME="$HOME" PATH="$PATH" axiom --json runtime codex auth
env -i HOME="$HOME" PATH="$PATH" codex exec --skip-git-repo-check "Reply with exactly: axiom-probe-ok"
env -i HOME="$HOME" PATH="$PATH" axiom --json runtime claude auth
env -i HOME="$HOME" PATH="$PATH" claude --print "Reply with exactly: axiom-probe-ok"
```

Expected output: each preflight `subscription_observed`; each probe exits 0 and
its output contains `axiom-probe-ok`. Record only: date, versions, method,
executable digest, exit code, whether the marker appeared, and `evidenceKind:
real_probe`. Do not record raw output beyond the marker, account, organization,
plan or usage data. Failure (non-zero exit, login prompt, quota or billing
message) is recorded as the concrete vendor limitation by category; never retry
with an API key or another provider. Recovery: none required — the probe
changes no Axiom state; a vendor login problem is fixed by the human.
