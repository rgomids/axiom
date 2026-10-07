# Issue #140 — portable Runtime/Profile policy v2

Authority: [Issue #140](https://github.com/rgomids/axiom/issues/140),
[Plan](https://github.com/rgomids/axiom/issues/140#issuecomment-6029914946),
[Tasks](https://github.com/rgomids/axiom/issues/140#issuecomment-6029934192),
and explicit implementation instruction. This additive contract refines ADR-0004
and ADR-0009; it does not change their ownership boundaries or v1 semantics.

## Closed wire contract

```yaml
schemaVersion: 2
project:
  id: 123e4567-e89b-42d3-a456-426614174000
  slug: example
  name: Example
runtimes:
  - id: claude
  - id: codex
modelProfiles:
  - key: careful
    runtimeRef: claude
    model: approved-model
  - key: worker
    runtimeRef: codex
    model: approved-model
runtimePreferences:
  - role: implementation
    complexity: high
    modelProfileRef: careful
```

V2 retains every v1 field and its declaration semantics except singular `runtime`,
which is forbidden and replaced by `runtimes`. `schemaVersion` must be integer 1
or 2; unsupported versions fail before effects. Every mapping remains closed.
No automatic upgrade, down-conversion, migration or Project rewrite occurs.
Identity remains required. The three policy collections remain optional:

- absent: no declared intent for that collection;
- `unconfigured`: explicit unresolved intent;
- `[]`: explicitly configured with no members;
- nonempty sequence: declared members, never implicit defaults.

All three empty/unresolved forms are preserved by canonical round trips. A valid
portable Project may have no executable policy. Execution then blocks. Absent or
empty preferences apply no preference; explicit `runtimePreferences: unconfigured`
blocks execution until the unresolved selection intent is reconciled.
`runtimes` contains closed mappings with required opaque nonblank `id`, unique
within the collection. At most 8 Runtimes, 32 profiles and 32 preferences are
supported in v2. Existing identifier safety checks prohibit paths and secrets.
`modelProfiles` keeps v1's closed shape: required unique `key`; either
`state: unconfigured` alone, or required `runtimeRef` and `model` with no `state`.
Configured profiles must reference a declared Runtime ID. Unconfigured profiles
are not execution candidates. `runtimePreferences` contains closed mappings with
required logical `role`, `complexity`, `modelProfileRef`; each (role, complexity)
pair is unique and must reference a configured profile. Role and complexity tokens use lowercase
letters, digits, dash, underscore or dot, at most 256 bytes; profile references retain the existing logical-key rules. No wildcard or ranking
is introduced. Identifiers/model values are not trimmed or case-folded.

Canonical encoding orders Runtimes by ID, profiles by key and preferences by
(role, complexity). Ordering never chooses a Runtime. Other v1 normalization and
presence rules remain unchanged. No stage selector is introduced: sequential
workflow gates exist, but graph child capability requests have no stable stage
field. Role and complexity preferences cover the existing shared resolver input.

## Application projection and compatibility

V1 reads and writes exactly its historical contract. Its configured singular
Runtime plus configured profiles projects into a legacy single-Runtime execution
allowlist; absent/unconfigured Runtime or absent/empty/unconfigured profiles
blocks. Multiple eligible profiles without a declared preference are ambiguous.
V1 never implies Codex and stays v1 on disk.

V2 execution candidates are the intersection of portable configured profiles and
machine-local configuration. IDs, Runtime association and concrete model must
agree exactly. Local profile allowlists, enabled state, supported complexities,
proven capabilities and installed/available observations further constrain that
intersection. Project preferences are projected to the existing runtimeprofile
resolver; global local preferences cannot invent Project intent. An explicit
operator Runtime selector may narrow candidates, never widen them.

Local adapter/executable bindings, environment, credentials, availability, Runtime
version and process state never enter portable intent. Profile capabilities and
complexity support remain local observations/configuration in this slice.
`runtimeprofile` remains the only selection algorithm. Missing, incompatible,
unavailable and unsupported candidates block; unresolved multiplicity blocks.

Preview includes Project ID, requirements, choice/model/version, configuration and
observation revisions, Project/configuration/observation digests and blocker code.
Digests bind actual snapshots as well as revision counters. Dispatch re-reads
Project/local configuration and observations before invocation, validates exact
preview equality, and checks the concrete command binding's model agrees. Drift
blocks and requires fresh preview; dispatch cannot silently choose another profile.
Preview performs no Runtime dispatch and grants no external mutation authority.
Output excludes credential references, environment values and raw adapter errors.

## Constitution and validation

Project != Repository; Runtime != Model; Role != Model; Execution != Agent.
Portable intent remains independent of machine observations. Deterministic parsing,
projection, resolver and freshness checks precede effects. No architecture waiver,
new dependency or ADR supersession is required. T140-02–T140-08 prove closed-schema
round trips, v1 compatibility, blockers, both concrete adapters, pre-dispatch
freshness and presentation safety through tests and versioned completion Evidence.

## Compatibility window and release generation

This implementation supports portable schemas 1 and 2 concurrently, with no
retirement of v1, and introduces no machine-local persisted-state generation.
Portable schema 2 is a new portable shape under Specification 004 FR-073/FR-074.
It has no published baseline yet. Before the first stable release shipping it,
release acceptance must declare that immutable release as its generation baseline
and freeze representative nonempty v2 content. Existing published baselines and
stable-v1 corpora remain supported and append-only. This implementation does not
perform release acceptance, tag creation or publication.

The CLI observation file is explicitly supplied machine-local operator inventory
using the existing Observation contract, not a new Axiom store or a claim that
Lingo independently proved capabilities. File reads are bounded and closed-schema;
future checks re-read the file. Programmatic execution sources must provide current
Runtime observations on every check. Recorded observations cannot prove changes
outside their observation source; live Runtime/provider acceptance remains separate.
