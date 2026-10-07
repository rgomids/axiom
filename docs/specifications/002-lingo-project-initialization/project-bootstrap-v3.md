# Issue #231 — Project bootstrap, context registry and readiness (portable v3)

Authority: [Issue #231](https://github.com/rgomids/axiom/issues/231), its
[Contract + Implementation Plan](https://github.com/rgomids/axiom/issues/231#issuecomment-6043012537),
its published Tasks (T231-01–T231-11) and explicit maintainer implementation
instruction. This additive contract refines ADR-0004 within its existing
ownership boundaries. It changes no v1/v2 semantics, no Accepted ADR, and no
credential, authority or Project/Repository identity rule.

Invariants kept: Project != Repository; Provider != Capability; Integration !=
Transport; Integration != MCP; Runtime != Model; Role != Model; Execution !=
Agent; portable intent != machine-local state; structural validity != readiness;
readiness != authority. No CWD, nearest checkout, chat/session history or prior
implicit selection ever identifies a Project or Repository. Git remote `origin`
has no semantic priority. Nothing defaults to Codex or to any model.

## 1. Portable schema v3

`schemaVersion: 3` is a new closed portable shape. It retains every v2 field and
its exact v2 semantics (including the #140 Runtime/Profile policy and its bounds,
and the prohibition of singular `runtime`) and adds four optional context
fields. Example:

```yaml
schemaVersion: 3
project:
  id: 123e4567-e89b-42d3-a456-426614174000
  slug: axiom
  name: Axiom
repositories:
  - key: core
    remote: https://github.com/rgomids/axiom.git
runtimes:
  - id: claude
modelProfiles:
  - key: careful
    runtimeRef: claude
    model: approved-model
runtimePreferences:
  - role: implementation
    complexity: high
    modelProfileRef: careful
providers:
  - key: work-items
    id: github
integrations:
  - key: work-items
    providerRef: work-items
    capabilities:
      - work-item
technologyContext:
  - key: infrastructure.terraform
    value: terraform
  - key: language.go
    value: go
documentationSources:
  - key: architecture
    kind: repository
    repositoryRef: core
    path: docs/architecture
  - key: product-notes
    kind: local-file
businessContext:
  text: Bounded product context.
  sourceRefs:
    - architecture
    - product-notes
  glossary:
    - key: work-item
      term: Work Item
      definition: A bounded unit of engineering intent tracked by Axiom.
```

### Compatibility

- `schemaVersion` must be integer 1, 2 or 3; anything else fails before effects.
- v1 and v2 keep their exact decode/encode contracts and remain readable and
  usable. The four v3 fields are unknown keys in v1/v2 and are rejected there.
- No automatic upgrade, migration, down-conversion or rewrite occurs on read,
  validation, readiness or any operation. A v1/v2 Project stays v1/v2 on disk.
- Projects created through `project configure` CREATE are v3. EDIT preview keeps
  the existing schema version of the Project it edits; post-create adoption of v3
  belongs to #230.
- Existing `businessContext.documents` keeps its v1 meaning (contained portable
  relative documents). v3 does not reinterpret it.
- Wherever the #140 contract says "v2" for Runtime policy, v3 behaves identically
  (allowlist projection, presence forms, bounds, canonical order).

### `technologyContext`

Optional; `absent` | `unconfigured` | `[]` | sequence, with the v2 collection
presence meanings. At most 64 closed mappings `{key, value}`:

- `key`: required, unique, lowercase token `^[a-z0-9][a-z0-9._-]{0,127}$`;
  logical identifier, no closed global taxonomy (detectors propose keys such as
  `language.go`, `package-manager.pnpm`, `container.docker`,
  `infrastructure.terraform`, `workflow.github-actions`);
- `value`: required, 1–256 bytes, single line, no control characters, not
  whitespace-padded, and subject to the existing identifier safety rule (no
  machine path, `file:` reference or secret-bearing payload).

A persisted fact is confirmed Project intent. Detection provenance/Evidence
lives only in the bootstrap preview, never in the manifest.

### `documentationSources`

Optional; `absent` | `unconfigured` | `[]` | sequence. At most 64 closed mappings:

| `kind` | other fields | rules |
| --- | --- | --- |
| `repository` | `repositoryRef`, `path` (both required) | `repositoryRef` names a declared Repository; `path` is repository-relative, 1–1024 bytes, `/`-separated, no empty, `.` or `..` component, no leading `/`/`~`, drive, backslash, `:`, `$`, backtick or control character |
| `local-file` | none | the key is the portable identity; the absolute path exists only in machine-local state (§2) |

`key` follows the technology key grammar and is unique. Any other `kind` fails
with `unsupported_source_kind`; fields not allowed for a kind fail. Future
Integration-backed sources (Notion, Confluence, Drive, …) are not modeled as local
files; they would require a later schema variant and keep Integration !=
Transport.

### `businessContext` additions (v3 only)

- `sourceRefs`: optional sequence (≤64, unique) of `documentationSources` keys;
  dangling references fail.
- `glossary`: optional sequence (≤128) of closed `{key, term, definition}`;
  `key` as above and unique; `term` 1–256 bytes single line; `definition`
  1–2048 bytes, no control character other than line feed. Neither is
  `unconfigured`-capable; `[]` is explicit emptiness.

Business text, glossary terms and definitions obey the existing portable safety
rule. Sensitive structural assignments (`password=…`, `token: …`, and the existing
normalized name families) are inspected across the complete input, independently
of URL component parsing, including bounded nested percent escaping and Markdown
wrappers. Names in ordinary prose (`password policy`, `token identifies a request`)
and ordinary URL paths/anchors are allowed when no assignment is present.

### Canonical encoding

Collections added by v3 are sets: `technologyContext`, `documentationSources`,
`glossary` sort by `key`; `sourceRefs` sorts lexically. Ordering never selects
anything. The whole manifest remains bounded by the existing 256 KiB / node /
depth limits, strict tags, no anchors/aliases/merge keys, and the closed-mapping
rule. Diagnostics carry fixed codes and schema paths only.

## 2. Machine-local documentation bindings (local record format 2)

Repository paths, credential bindings and Runtime state stay machine-local
exactly as today. A `local-file` source resolves through an installation-owned
binding:

```json
{"sourceKey":"product-notes","explicitPath":"/abs/path/product.md",
 "canonicalIdentity":"file:<64 hex>","observation":{"availability":"available",
 "basis":"present_metadata","observedAt":"…"}}
```

- The installation record gains `formatVersion: 2`, identical to format 1 plus a
  required non-empty `documentationBindings` array (≤64, unique `sourceKey`).
  Writers emit format 1, byte-identical to the previous writer, whenever there are
  no documentation bindings; format 2 only when bindings exist. Reading format 1
  never migrates it. Format 1 containing `documentationBindings` is malformed;
  format 2 with an empty array is non-canonical and rejected. Formats > 2 are
  `unsupported_newer`.
- `explicitPath` is absolute; `canonicalIdentity` is `file:` plus SHA-256 over the
  clean path and filesystem object identity, observed without following links.
- No document body, digest of content, secret or portable declaration has a slot.
- Installation accepts a binding only for a declared `local-file` source;
  dangling bindings fail. Unbound `local-file` sources are allowed (readiness
  warns).
- Compatibility window: format 2 is a new persisted-state format. Format 1 remains
  supported with direct compatibility. The first stable release shipping format 2
  must declare this window during release acceptance (Specification 004); this
  implementation performs no release acceptance. ADR-0004 places installation
  layout/version under the Local Installation Store, so no ADR changes.

## 3. Guided bootstrap (`project configure` CREATE)

`project configure` remains the only setup surface; no `project bootstrap`
command exists. CREATE builds one application proposal, previews it, and
publishes only after `--authorize-local` with the exact preview digest. EDIT keeps
its existing preview-only scope (#230 owns post-create lifecycle) and rejects the
bootstrap-only inputs below.

### Inputs (all explicit)

| Flag | Meaning |
| --- | --- |
| `--repository <key>=<absolute-path>` or `<absolute-path>` | explicit Repository locations (≥1, ≤32); a bare path gets a derived key proposal |
| `--repository-remote <key>=<locator>` / `<key>=none` | explicit remote identity choice; `none` keeps the Repository local-only |
| `--work-item-provider <id>` / `none` | requested `work-item` capability → Integration `work-items` → Provider |
| `--runtime <id>` | allowed Runtime (repeatable); must be locally configured |
| `--model-profile <key>` | allowed Model Profile (repeatable); copied as `{key, runtimeRef, model}` from local configuration |
| `--runtime-preference <role>/<complexity>=<profile>` | Runtime preference (repeatable) |
| `--technology <key>=<value>` / `--remove-technology <key>` | add/replace or drop a technology fact |
| `--documentation <key>=repository:<repo-key>/<path>` / `<key>=local-file:<absolute-path>` | documentation source (repeatable) |
| `--business-context <text>` | bounded business context (≤4096 bytes) |
| `--context-source <key>` | `businessContext.sourceRefs` entry (repeatable) |
| `--glossary <key>=<term>:<definition>` | glossary entry (repeatable) |

### Repository discovery (read-only, no Git process, no network)

For each explicit path: the path must be an absolute existing directory that is
not a link. `.git` is inspected with `lstat`: a directory, or a regular `gitdir:`
file (worktree; `commondir` honoured) whose target is a non-link directory.
A linked `.git`, unreadable or oversize metadata classifies the Repository as
`unsafe`/`unreadable`, which blocks publication. A directory without `.git` is a
valid local-only Repository. The repository `config` (regular file, ≤64 KiB) is
parsed for `[remote "<name>"] url` entries only; nothing is executed, fetched,
pushed or cloned, and global/system configuration is never read.

Each URL is normalized with the existing domain locator rule. URLs that are not
portable locators (local paths, embedded passwords, malformed) are never shown
and count only as `unsupported`. Classification:

| Status | Condition | Bootstrap behavior |
| --- | --- | --- |
| `local_only` | no candidate locator | Repository published without `remote` |
| `single` | exactly one distinct normalized locator | proposed; operator may override |
| `ambiguous` | two or more distinct locators | `repository_remote_ambiguous` blocks until `--repository-remote` |
| `incomplete` | `include`/`includeIf`/`insteadOf`/continuation lines or bounds (≤32 remotes, ≤64 URLs) exceeded | blocks until `--repository-remote` |

Aliases (different remote names) that normalize to one locator are one candidate.
Different transports (HTTPS vs SSH) are distinct locators. Candidates sort by
locator, names by name; `origin` is never preferred. An explicit
`--repository-remote` may name any valid portable locator (preview marks its
`source: operator`). Absolute paths enter only machine-local state.

A derived key lowercases the directory base name, maps other characters to `-`,
collapses/trims dashes and truncates to 63 bytes; an unusable or duplicate key
blocks (`repository_key_required`, `repository_key_conflict`) until an explicit
`<key>=<path>` is supplied.

### Runtime/Profile policy (#140 boundary)

Bootstrap reads the machine-local Runtime Profile configuration only to list
candidates (Runtime IDs with their allowlisted profiles and models). Choices must
name local candidates; each profile's Runtime must be a chosen Runtime;
preferences must reference chosen profiles. Only portable declarations are
written; adapters, executables, observations and credential references stay
local. No Runtime, profile or preference is ever chosen implicitly. With no
`--runtime`, the Project carries no Runtime policy: valid, with Execution
readiness `runtime_policy_unavailable`.

### Provider and capability

`--work-item-provider github` declares Provider `work-items` (`id: github`) and
Integration `work-items` with capability `work-item`. Any other identifier is
declared truthfully and reports `provider_unsupported`; `none` or omission leaves
the capability unmapped (`capability_mapping_missing`). Provider selection implies
no Transport and grants no authority; bootstrap performs no Provider call.

### Technology discovery (proposal only)

Detectors inspect file **names** only — no content read, no execution, no package
installation, no network. Bounds: depth ≤2, ≤4096 entries per directory, ≤256
directories per Repository, links never followed; `.git`, `node_modules`,
`vendor` and hidden directories are skipped (except `.github/workflows`).

| Signal (repository-relative) | Proposed fact |
| --- | --- |
| `go.mod` | `language.go=go` |
| `package.json` | `language.javascript=javascript` |
| `tsconfig.json` | `language.typescript=typescript` |
| `pnpm-lock.yaml` / `package-lock.json` / `yarn.lock` / `bun.lock`, `bun.lockb` | `package-manager.pnpm|npm|yarn|bun` |
| `Cargo.toml` | `language.rust=rust` |
| `pyproject.toml`, `requirements.txt` | `language.python=python` |
| `poetry.lock` / `uv.lock` | `package-manager.poetry|uv` |
| `Gemfile` | `language.ruby=ruby` |
| `pom.xml` | `build.maven=maven` |
| `build.gradle`, `build.gradle.kts` | `build.gradle=gradle` |
| `Dockerfile`, `Containerfile` | `container.docker=docker` |
| `compose.yaml`/`.yml`, `docker-compose.yaml`/`.yml` | `container.compose=compose` |
| `*.tf` (depth ≤2) | `infrastructure.terraform=terraform` |
| `.github/workflows/*.yml`/`.yaml` | `workflow.github-actions=github-actions` |
| `.gitlab-ci.yml` | `workflow.gitlab-ci=gitlab-ci` |

Each proposal carries `{repository, path}` Evidence (first match in byte order
plus a bounded count). More than one package manager in the same Repository marks
those facts `conflict` for review instead of choosing. Operator `--technology`
replaces/adds and `--remove-technology` drops before publication; the preview
digest covers the final set.

### Documentation, business context and glossary

Repository sources must reference a bootstrap Repository; their path is checked
lexically. Local-file sources take an absolute path that must be an existing
regular non-link file; its identity becomes a local binding and its content is
never read. `--context-source` must name a declared source.

### Preview, blockers and publication

The preview reports Project identity, Repositories (key, key source, local path,
Git state, remote status/candidates/choice), the capability mapping, Runtime
candidates and choices, technology proposals with Evidence, documentation sources
and bindings, business context, glossary, authority expectations, effects,
`blockers` and the digest. A preview with blockers can be reviewed but never
published (`bootstrap_blocked`). Missing optional intent is not a blocker; it
becomes readiness information. Publication keeps the existing protocol:
exact digest, portable publication, then local publication with truthful
partial-effect reporting.

## 4. Canonical Project readiness

One read-only application evaluator (`projectapp` readiness) composes the
validated portable Project, its installation record, Repository binding
observations, Provider support, credential-binding presence, the #140 Runtime
availability projection, and documentation resolution into a bounded,
deterministic `ReadinessReport`. It performs no mutation, Provider call or
Runtime dispatch and grants no authority.

### Report

- `structure`: `valid` | `invalid`; `effective`: `ready` | `partial` | `blocked`
  (`ready`: every operation ready; `partial`: at least one ready and one blocked;
  `blocked`: none ready).
- `repositories[]`: key, binding (`bound`/`missing`), availability
  (`available`/`unavailable`), remote (`declared`/`local_only`).
- `capabilities[]`: capability, Integration, Provider, status
  (`ready`/`missing`/`unsupported`), credential (`not_required`/`bound`/`unbound`).
- `runtime`: status (`ready`/`unavailable`/`blocked`) and the #140 blocker code.
- `documentation[]`: key, kind, status (`available`/`missing`/`unbound`/
  `stale`/`unsafe`/`repository_unavailable`). No content, no path.
- `context`: counts of technology facts, documentation sources, source refs,
  glossary entries, documents, policies; presence of business text.
- `authority`: readiness grants no authority; local effects need
  `--authorize-local`, Provider effects `--authorize-external`, each with an
  exact preview digest.
- `operations[]`: `work-item` and `execution`, each `ready`/`blocked` with its
  blockers. `blockers[]`/`warnings[]`: `{code, subject}` sorted by code then
  subject. Subjects are validated logical keys only.

### Codes

Blockers: `project_not_installed`, `project_source_unavailable`,
`project_state_invalid`, `installation_stale` (the portable source changed after
installation validated it), `recovery_required`, `repository_binding_missing`,
`repository_unavailable`, `capability_mapping_missing`,
`capability_mapping_ambiguous` (more than one Integration declares the
capability; none is chosen by order), `provider_unsupported`,
`runtime_policy_unavailable`, `runtime_resolution_blocked`.
Warnings: `credential_binding_missing` (a declared `credentialRef` has no local
binding; reported, not enforced, because no binding writer exists yet and the
implemented GitHub Provider authenticates ambiently through `gh`, preserving
pre-#231 behavior), `documentation_binding_missing`, `documentation_unavailable`,
`documentation_stale`, `documentation_unsafe`, `technology_context_absent`,
`business_context_absent`.
Bootstrap-only blockers: `repository_remote_ambiguous`, `repository_key_required`,
`repository_key_conflict`, `repository_unsafe`.

No code carries raw paths, parser/Provider/Runtime error text or secret values.

### Operation requirements

| Requirement | `work-item` | `execution` |
| --- | --- | --- |
| installed, structurally valid, no recovery state | ✓ | ✓ |
| every Repository bound and available | ✓ | ✓ |
| `work-item` → Integration → supported Provider | ✓ | ✓ |
| credential binding when that Integration declares `credentialRef` | warning | warning |
| Runtime policy declared and locally resolvable (#140) | — | ✓ |
| documentation / business / technology context | warning | warning |

Every current Execution is scoped to a linked Work Item, so `execution` includes
the `work-item` set. Capability mapping is one algorithm (`ResolveCapability`):
Integrations that declare the capability and reference a declared Provider; zero
→ missing; more than one → the conventional `work-items` Integration referencing
the `work-items` Provider when exactly that pair is declared (the pre-#231 rule
that Lingo's own setup writes), otherwise ambiguous; one → its Provider is `ready`
only when this build implements it (GitHub for `work-item`). The EDIT preview uses
the same mapping. The portable half of Runtime policy
(an allowed Runtime, at least one Model Profile, no `unconfigured` preferences) is
a domain rule shared with #140; without it Execution reports
`runtime_policy_unavailable` and no machine observation is made. Otherwise
availability is #140's own projection (`runtimeapplication.Availability`: at least
one allowed, enabled, installed and available Runtime with an allowed profile);
any #140 code (for example `no_allowed_match`, `runtime_unavailable`) yields
`runtime_resolution_blocked` with that detail. Readiness checks availability only;
selection stays the #140 resolver at operation time.

## 5. Pre-effect enforcement

Work Item create/select/show/comment/complete and `workflow resume` evaluate
their projection (`work-item`, `execution`) before any other step. `workflow
start` evaluates the shared and `work-item` requirements first; its Runtime
requirement is then the #140 request-specific projection (reviewed preview plus
fresh `Check`), so Runtimes are not observed twice and #140's reviewed-preview
contract is unchanged. A blocked projection of a structurally valid Project
returns its first blocker code as the operation category with a `preflight`
payload carrying the same codes as `project validate`, and produces zero Provider
calls, zero attempt/Execution writes and zero dispatch. Structural failures (not
installed, recovery, invalid or stale installation) keep each operation's
established category; those operations apply the same selection and source
checks and also fail before effects.
Readiness and authority are independent: a ready operation without authority is
still denied with its existing authority category; authority never bypasses a
blocker. No fallback to CWD, first Provider, first Runtime/profile or Codex.

### Limitations

`project install` (authored manifests) records Repository bindings only; a
`local-file` source of an installed authored v3 Project stays unbound (readiness
warns). Post-create binding maintenance belongs to #230. Technology detection
reads at most 4096 entries per directory; in a larger directory the subset is the
one the filesystem lists first.

## 6. Security and failure cases

- portable config rejects absolute/machine paths, `file:` references and
  secret-bearing values; local-only paths stay in the installation record;
- relative documentation paths reject traversal/escape; resolution walks each
  component with `lstat` and rejects links and special files;
- local-file bindings fail closed on missing, replaced (`stale`) or unsafe
  objects; content is never read;
- Git metadata is parsed, never executed; no shell, no `git` process, no network;
- technology detection reads names only;
- readiness, discovery and preview perform no writes;
- outputs exclude credential references' values, raw paths of bindings in
  readiness, document content and parser errors.

## 7. Acceptance Evidence

Completion Evidence lives in [evidence-issue-231.md](evidence-issue-231.md) and
maps each #231 acceptance criterion to tests: v3/v1/v2 codec matrices,
local format 1/2 fixtures, remote discovery fixtures, technology fixtures,
documentation resolution, capability/credential/Runtime readiness matrices, and
pre-effect no-mutation tests.

## 8. ADR assessment

No ADR changes. ADR-0004 already owns versioned portable evolution, the
portable/local split and no silent migration, and places installation layout and
version under the Local Installation Store. ADR-0001 (Project != Repository) is
preserved: Repositories remain explicit associations. ADR-0005 threat-model
properties apply unchanged to the new read paths (traversal, link and replacement
rejection, fail-closed). ADR-0007 publication/recovery is reused unchanged.
