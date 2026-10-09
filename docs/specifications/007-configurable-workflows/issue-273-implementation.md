# Issue #273 — Project workflow authoring

Scope: WF-001, WF-002, WF-005 / AC-001, AC-002, AC-005 of accepted
Specification 007 and ADR-0020. The maintainer's 2026-10-09 request authorizes
implementation and a PR; it does not authorize merge, release or human acceptance.
Initial baseline: `917c00d`; rebased onto `fcf4f46` on `main` before the PR.
Review and acceptance of this implementation remain human decisions.

## Implementation plan and boundaries

1. Validate the complete portable contract with Draft 2020-12 and deterministic
   semantic checks; identify content with RFC 8785 JCS plus SHA-256.
2. Extend Project/manifest schema to 4, preserving schema 1/2/3 readers/writers
   and explicit declaration presence; upgrade only upon explicit selection.
3. Publish immutable companion content, then append its bounded revision index
   through ADR-0007; select only indexed, readable published content. Refresh the
   local portable observation through the existing Project edit publisher.
4. Expose authoring under Project CLI/domain skill operations, retaining the
   Execution meaning of the existing top-level `workflow` commands.
5. Exercise invalid/stale edits, interrupted publication/recovery, retained
   history and native executable journeys; reconcile inventory and compatibility.

No new scheduler, execution format, Runtime dispatch or Provider effect is part
of this slice. #274 owns configurable admission/binding and legacy resume, #275
owns runtime/profile/effort planning, and #276 owns dispatch. Existing sequential
execution behavior is preserved; selecting a definition here does not claim that
the legacy executor consumes it. Those slices must refuse an unselected Project
when admitting a **new configurable** execution.

## Contracts and recovery

`project workflow list/show/validate/create/edit/select/remove/recover` is the
public operation family. Canonical completion and provenance remain top-level;
the `workflowAuthoring` payload contains exact references, definitions,
diagnostics, prerequisites, effects, `projectRevision` and `previewDigest`.
`axiom-project` forwards these as `workflow.<operation>` for Codex and Claude;
there is no additional workflow skill.

Preview is read-only. Apply repeats the same intent with `--expected-revision`
equal to `projectRevision`, `--preview-digest` and `--authorize-local`. The digest
binds operation, Project ID, portable and local revisions, current selection,
prior and proposed workflow identities, complete index and effect list.
Index/content changes invalidate authorization even if the manifest is unchanged.
Definition publication retains the existing local chain locks while checking the
local wire and acquiring the portable lock. Selection rechecks the reviewed
index inside the portable writer lock, including concurrent index-only changes.
Same identity/content is unchanged; different content at that identity conflicts.

Companions live at `workflows/<workflowId>/<revision>.json` and
`workflows/index.json` below the protected Project working copy. The index is
bounded to 1024 identities / 1 MiB. Retired identity assignments are never reused.
`remove` retires the index entry and conservatively retains content so historical
revisions remain inspectable. It refuses built-ins, the current selection,
explicit references, and uncertain local inventories. Current format-1 Executions
carry no configurable reference; an unknown future record, recovery object or
unsupported format blocks retirement. #274 must provide complete binding and
reference inventory before relaxing that conservative refusal.

Publication uses existing file fencing, CAS, private ownership, no-link checks,
read-back and recovery markers. Content precedes index; selection follows both.
Interrupted file protocols use `recovery inspect/apply`. A fully published orphan
is never selectable or silently adopted: provide its exact JSON to
`project workflow recover --file`, review the new preview and explicitly apply.
A confirmed content publication followed by index failure reports `partial`;
selection's cross-store failure uses the existing Project edit recovery protocol.

Mutation follows #230's existing Project administrative ownership boundary:
installed Projects under the Lingo projects root. An external recorded source
is refused as `unsupported_portable_source`, preserving it rather than writing
through an unowned path. Optional retired content deletion is deferred; no
background pruning or destructive filesystem operation is exposed.

Static validation checks strict JSON/UTF-8, unknown/missing fields, integer and
byte limits, unique IDs, protected gates, lifecycle/checkpoints, declared I/O,
prior-stage references, DAG cycles, connected integration owner and supported
capability vocabulary. Project context/profile/policy keys resolve against
Project declarations. Readiness prerequisites explicitly retain runtime/profile/
effort, validator availability and digest-bound local context/artifact resolution
for planning; authoring never claims these observations were performed.

## Verification map

| Issue acceptance criterion | Executable evidence |
|---|---|
| Fresh Project explicitly selects default without defining stages | `TestExecutableWorkflowDefaultCustomEditSelectAndRetire`; seed/frozen-digest test |
| Custom stage/agent create, inspect, edit, validate and select; Runtime parity | Executable journey; approved mixed Codex/Claude fixtures; canonical routing/discovery and shared embedded skill tests |
| Invalid/stale edits preserve selection; identical apply safe | Executable journey; `TestWorkflowPreviewBindsIndexContentAndLocalRevision`; immutable publication test |
| Portable definitions exclude host paths/credentials/observations | Closed schema, portable prose policy and negative definition/reference tests |
| Old revisions inspectable; retention never breaks references; legacy readable | Retirement/reference tests, schema 1/2/3-to-4 tests, writer inventory and frozen append-only corpus |

`TestWorkflowInterruptedContentAndIndexRecovery` injects interruptions at F3–F8
for both content and index. Existing ADR-0007 tests retain F0–F8 coverage.
The definition byte-window test covers content larger than the historical 256 KiB
record limit without broadening other record limits.

Run from the repository root:

```bash
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
python3 docs/specifications/007-configurable-workflows/validate_examples.py
```

The `v0.12.0-issue273` compatibility snapshot is an **unreleased test candidate**
generated by actual writers, not evidence of release or operational acceptance.
Windows storage tests require the repository's private temporary-directory/ACL
preconditions; an unsafe host directory is a refusal, never grounds to weaken
the storage policy. Cross-platform CI remains the native regression gate.

## Maintainer verification record — 2026-10-09

- Ubuntu: complete `go test -race ./...` passed, including executable authoring,
  historical upgrade ownership, immutable publication and F3–F8 recovery.
- Rebased tree: `go vet ./...`, `go build ./...`, `go mod verify`, repository
  governance and all approved example/digest checks passed. Governance covered
  22 ADRs and 81 automation surfaces. Legacy end-to-end dogfood reported `pass`.
- The repository's pinned `check-go-quality.sh all` passed with Go 1.26.0 and
  Staticcheck v0.8.1. Race tests used Go 1.27.2. The newer toolchain's export-data
  format is unsupported by that pinned analyzer; the supported Go 1.26 run is
  the quality result, with no gate alteration.
- Windows: build and Project/application/manifest/CLI/definition tests passed.
  Full storage/upgrade tests refused this host's temporary-directory ACLs
  (`unsafe_target` and private-directory checks). Native hosted regression CI
  must supply Windows/macOS evidence; no protection was weakened.
- Initial hosted macOS execution exposed a lexical source-path comparison for
  the trusted root-owned `/var` alias. Publication now reuses `trustedCanonical`
  from the existing Project edit boundary; a native alias regression test was
  added. User-owned links remain refused.
- Staged sensitive-file scan and diff checks passed. Gitleaks was unavailable.
  Shared-skill behavior was checked deterministically; real Codex/Claude
  behavioral sessions and human operational acceptance remain unverified.

## Dependency implementation choices

Adopt `github.com/santhosh-tekuri/jsonschema/v6` v6.0.2 (Apache-2.0) for full
Draft 2020-12 validation rather than the documentation checker's subset. Compile
only the bundled schema; no network schema loading is used.
Adopt `github.com/cyberphone/json-canonicalization` at
`19d51d7fe467` (Apache-2.0) for the accepted RFC 8785 wire identity. Its six
upstream conformance input/output vectors and license are vendored as **test
data**, including Unicode/UTF-16 sorting, string controls and ECMAScript numbers.
The schema forbids floats/null and checks bounds before canonicalization.
`golang.org/x/text` is the validator's indirect Unicode dependency (BSD-3-Clause).
No paid service, CLI login, runtime library or process execution is introduced.
These are implementation choices under the accepted schema/JCS decision, not
changes to ADR-0020 or to historical graph/proposal digests.

## Integration with Issue #232

Merge `2d7a7cd` from `main` to reconcile the human-first presentation contract.
Workflow authoring now presents its exact canonical event through the shared
Markdown/JSON renderer. Three-level routing metadata keeps the workflow family;
its examples and embedded skill use human output by default. Both pre-integration
skill sets remain owned for upgrades, alongside the combined embedded revision.
`TestWorkflowAuthoringPresentsExactCanonicalEvent` covers successful previews,
stale authority and partial publication without dropping workflow fields or
repeating an operation for presentation.
