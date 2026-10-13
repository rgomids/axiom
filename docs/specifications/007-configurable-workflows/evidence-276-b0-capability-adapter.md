# Issue #276 — B0 capability, envelope and adapter evidence

Observation date: 2026-10-13. Scope: the user-authorized implementation of
[I276-T10/T11/T13/T18/T19/T20](plan-tasks-276-stage-agent-execution.md), on
`feat/276-stage-agent-execution`, following baseline commit `265e24e`.
This records internal implementation and synthetic checks, not merge, acceptance,
real capability proof or authorization to execute a Runtime.

## Implementation and tests

| Task | Implemented boundary | Coverage |
|---|---|---|
| T10 | Pure capability evidence vocabulary, states, exact Runtime/variant/model binding, 30-day maximum age, invalidation and scoped confinement | Fresh matching evidence; every binding/model element changed; cross-variant refusal; expiry/future timestamps; interface-only/auth-only/unverified facts cannot prove; required confinement facts and explicitly untested roots; strict/redacted codecs |
| T11 | Machine-local `capability-evidence/v1` protected conditional publication, read-back, conditional invalidation, inventory and publication recovery | Round-trip/invalidate; expired evidence retained but unusable; stale invalidation; tampered digest; concurrent creation; F2–F8 interruption recognition; unsafe evidence refused before filesystem effects |
| T13 | Pure envelope contracts with a local filesystem adapter; explicit variant writable/negative roots; private scratch Git/nonce/inbox/sentinel; manifest identity checkpoints; quarantine and exact cleanup with durable cleanup intent | Unsafe roots/traversal/strict JSON; both variant manifests; create/digest/quarantine/reopen; existing path; replaced directory; symlinks; stale cleanup; interrupted preparation and cleanup at directory/manifest/root deletion boundaries |
| T18 | Version-gated internal constant argv and output bound for Codex 0.159.1 and Claude 2.1.296 | Four Runtime/variant golden argv; configured bypass/alternate/unknown arguments refused; unknown versions; clean disjoint working/inbox roots; pinned Claude coordination command prefix |
| T19 | Closed environment plus frozen explicit environment shared by auth and spawn; mandatory internal subscription-auth preparation | Allowed login/vendor/locale/correlation variables; provider tokens dropped; unknown variables dropped; duplicates rejected; missing login roots; missing authenticator; credential references refused; auth Runtime/version/digest equality; exact preflight/spawn bytes |
| T20 | Bounded deterministic adapter packet; read-only compiled-input projection; stdin attachment and attempt-correlated digest receipt; mandatory recorder port before preparation exposes stdin | Golden packet digest; graph-child/single-agent projections; required fields and coordination record bound; no Runtime/model/effort fields; oversized/credential-shaped/chat-marker/invalid UTF-8/missing-pin/unsafe-reference/path refusals; failed/missing receipt recorder cannot expose stdin |

Changed files (relative to the repository root):

- `internal/capabilityproof/envelope.go`, `evidence.go`, `evidence_test.go`
- `internal/local/capability_store.go`, `capability_store_test.go`
- `internal/local/probe_envelope.go`, `probe_envelope_identity_posix.go`,
  `probe_envelope_identity_windows.go`, `probe_envelope_inventory.go`,
  `probe_envelope_recovery.go`, `probe_envelope_test.go`
- `internal/local/inventory.go`, `recovery.go`,
  `state_compatibility_contract_test.go`
- `internal/runtimeadapter/stage_invocation.go`, `stage_invocation_test.go`,
  `packet.go`, `packet_test.go`
- `internal/graphapplication/stage_invocation.go`, `stage_invocation_test.go`,
  `task_packet.go`, `task_packet_test.go`
- `internal/executiongraph/scheduler.go`
- This evidence document.

## Executable evidence and limitations

The requested module environment was used with `GOFLAGS=-mod=mod`,
`GOPROXY=off`, and `GOMODCACHE=/home/box/go/pkg/mod`. The supplied
`GOCACHE=/home/box/.cache/go-build` fails with `read-only file system`; subsequent
checks use writable `GOCACHE=/tmp/axiom-276-go-build`.

- `gofmt -l .`: empty output.
- `go vet ./...`: passes.
- `go test ./...`: executed; fails in filesystem-dependent packages under this
  sandbox. `/` is owned by UID 65534 while the process EUID is 1000. Existing
  `internal/local/platform_posix.go` ancestor checks accept only root-owned or
  current-user-owned ancestors, so protected storage refuses before publication.
  The unchanged `TestExecutionStoreRoundTripsClosedPrivateState` independently
  reproduces this failure. No ownership/permission policy was relaxed.
- `go test ./internal/capabilityproof ./internal/runtimeadapter
  ./internal/graphapplication ./internal/executiongraph`: passes.
- Local unsafe-before-effects, strict-envelope-JSON, unsafe-envelope-policy and
  state-root writer inventory checks pass. Store publication, concurrency,
  invalidation, expiry retention and interruption/recovery tests compile but
  remain unverified on a supported filesystem because of the ancestor failure.
- Native envelope lifecycle tests require an explicit
  `AXIOM_TEST_PROBE_STATE_ROOT` outside repositories and prohibited temporary or
  vendor roots, on a filesystem satisfying the existing private-storage policy.
  They skip when that fixture root is absent. POSIX lifecycle execution and
  Windows execution are unverified here; Windows local-package test
  cross-compilation passes.

## Preserved boundaries and remaining integration

No real Runtime, vendor/model call, interface observation or probe was executed.
No network tooling, new dependency, CLI wiring or commit was introduced.
Graph1/Graph2 codecs, compatibility fixture bytes and `internal/runtimeprofile`
remain unchanged; the compiled #306 input/tuple types are consumed read-only.

The new invocation preparation seam is separate from historical configured
invocations. Later admission must derive effect flags from reviewed authority,
match packet inbox/Axiom pin to the invocation binding, revalidate variant-bound
capability evidence, and persist the packet receipt in its successor attempt
ledger. B0 provides and tests the mandatory recorder port; it does not supply
that later ledger or dispatch composition. Historical attempts gain no new
persisted field.

Envelope cleanup can resume with its durable intent even after recorded roots,
the envelope manifest, or the envelope root are removed. An interruption between
an exclusive mkdir and its first durable identity checkpoint leaves ambiguous
residue quarantined for operator review; automatic cleanup refuses to invent
ownership. Recovery tests for recognized partial states require the native
fixture described above.

Engineering/security review corrected leading chat-role and invalid UTF-8
filtering, unsupported-version negative-test masking, missing Claude coordination
execution, variant-root overbreadth, and interrupted envelope cleanup recovery.
Final static recovery review identified no remaining blocking finding; filesystem
behavior is subject to the explicit verification limits above.
