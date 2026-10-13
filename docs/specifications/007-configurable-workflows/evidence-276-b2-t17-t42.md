# #276 B2 — I276-T17 and I276-T42 implementation evidence

Implemented under the explicit maintainer request on
`feat/276-stage-agent-execution`, against the task plan §3.9, amendment A1/A2
and ADR-0026. Recorded contract approval statuses are unchanged.

The per-variant capability observer consumes only matching fresh evidence,
keeps model results scoped, and supplies the evidence digest through the
existing Profile proof reference consumed by StagePlan2. Runtime policy
composition intersects current facts with the freshly loaded configuration;
removed/disabled Profiles remain unknown and Profile edits invalidate proof.
Version acquisition is injected separately and rechecks executable identity.
Absent current facts never produce proof. Observation and Profile formats are
unchanged.

The capability service checks declared, effect-derived, implicit and model
requirements independently of Profile declarations. Every Check reloads the
full current binding and evidence. Executable/version/mode/adapter/pinned
axiom/Profile/model/effort/configuration/effort-source drift fails closed;
preview-to-dispatch drift returns capability_stale and ErrStalePreview
(CR-007 case 9). This is the consumption/revalidation boundary for the later
stage admission and dispatch tasks, not public CLI or probe wiring.

Show reports states, confinement scope and all five not_tested categories.
Invalidate binds local actor confirmation to the exact reviewed target,
binding, evidence reference and wire revision, then uses the existing store's
conditional invalidation. Preview copies its Profile tuple. Unauthorized,
changed-binding and stale-wire applies perform no mutation.

Validation used the installed Go 1.26 toolchain directly with GOPROXY=off,
GOSUMDB=off, GOTOOLCHAIN=local and GOCACHE=/tmp/axiom-go-cache.

- `go test ./internal/capabilityproof ./internal/runtimeapplication`: passed.
- `go test ./cmd/lingo -run TestRuntimeVersionObserver`: passed.
- `go build ./...`: passed (read-only module stat-cache warning).
- `go vet ./...`: passed.
- `git diff --check`: passed.
- Broader tests for internal/local, internal/workflowcompiler and cmd/lingo
  were attempted with default and private workspace TMPDIR. They fail with
  storage-safety errors including unsafe project storage and unsafe_source;
  those suites remain unverified in this sandbox. No safety check was relaxed.

Tests cover matching variants, all required-capability omissions, unknown and
unsupported evidence, expiry, different bindings, full-tuple drift, Profile
configuration edits/removal/disable, resolver evidence-reference propagation,
version-observation executable replacement, local invalidation authority,
stale previews and immutable preview tuples. Review found no remaining
blocking issue within these use-case boundaries; real dispatch and probe
operation are not demonstrated by synthetic tests.

No new persisted kind or wire format is introduced. These tasks require no
additional corpus freeze; the previously introduced capability_evidence kind
retains its existing corpus obligations. Historical Graph1/2 and corpus bytes
are unchanged. No real probe, inference, provider call or network command was
run. This session issued no commit command; HEAD changed concurrently from
51af05d to e012d7e during implementation. That commit was preserved, and later
test/documentation changes remain uncommitted.
