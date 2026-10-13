# #276 B1 — I276-T01 and I276-T32 implementation evidence

Scope: local implementation under the maintainer's explicit request on
`feat/276-stage-agent-execution`, without commit/push, network, real Runtime
probes, inference, provider calls or publication. This record does not change
Specification/ADR acceptance statuses, claim an accepted successor-contract
blob, mark a Task merged, or claim operational acceptance.

T01 adds the separate Graph3 codec, inspection version dispatch, domain/update/
canonical-record validation, valid and invalid vectors and the
[field-contract appendix](appendix-276-successor-graph-field-contract.md).
The historical `graph.go` codec and Graph1/Graph2 fixtures remain unchanged.
The original Graph1 stable-corpus and Graph2 delivered fixture have hardcoded
SHA-256 expectations and exact round-trip tests. Each valid Graph3 vector is
refused by the original reader. No work is reassigned to delivered I306-X05.

T32 adds an immutable `validator-policies/v1/<digest>.json` local artifact.
The `validatorpolicy` domain defines a closed format-1 definition with Project
identity, 1–64 commands (unique logical reference, exact argv, exact environment,
bounded output) and a human-operator review binding the definition preview
digest, actor, UTC time and separate operation-authority reference. Encodings
are compact JSON plus LF, at most 64 KiB; identifiers/digests and command sizes
are bounded. The local store checks the exact stored bytes against their content
address, reuses private-directory locks/publication, and is registered in state
inventory and recovery enumeration. This additive kind follows the existing
release-time frozen-corpus process; historical snapshots are not edited.

`stageexecution.ValidatorPolicyService.Preview` is read-only. Apply requires
explicit local authority, `human-operator` source and the exact reviewed
preview digest. Agent-record/Plan sources, missing authority and stale reviews
are refused. Changed commands/environment cannot reuse a review. Command
resolution requires the dispatch-bound artifact digest and the owning Project,
and returns copied `gitworkspace.ValidationCommand` values. Shell executables
and shell-launch wrappers are rejected; execution uses argv directly and an
explicit environment (including an empty environment rather than inherited
host credentials). This is reviewed operator policy under ADR-0005, not OS
confinement or executable-content authentication. T25/T33 own public preview/
integration admission and execution-time policy resolution; no public command
is wired here.

## Validation

All Go commands use `GOFLAGS=-mod=mod GOPROXY=off`, a writable `/tmp` Go cache,
and a workspace-owned private temporary directory. No dependency fetch was
performed.

- Graph package, validator-policy domain, stageexecution preview/apply and
  strict JSON tests pass (also with `-race`), including every reason, epoch progression,
  cancellation, fact append/mutation, missing/foreign records, strict decoding,
  version dispatch, exact historical digests and old-reader refusal.
- Policy tests reject shell argv, unreviewed command changes, wrong digest,
  nonoperator sources, missing authority, wrong Project and duplicate fields.
- Filesystem policy-store, inventory and recovery
  tests are implemented but cannot pass in this host: `/` is owned by UID
  65534 (`nobody`), while the test process is UID 1000. The unchanged
  `local.ancestorSafe` requires ancestors owned by root or the current user,
  so private storage rejects the host with `unsafe project storage` before
  the new publication/recovery path runs. Existing storage tests fail at the
  same boundary. The positive filesystem policy-create/reference and
  publication crash-window tests remain unverified on a compatible host.
- The real-local-Git integration test isolates review/create/reference via an
  in-memory policy store and executes the mapped argv in temporary integration
  worktrees: the complete `internal/gitworkspace` suite also passes with
  `-race`. Concrete filesystem publication is covered separately.
- The complete Go suite was attempted; its filesystem-dependent tests fail
  at this ownership boundary. This is not a green-suite claim and storage
  ownership checks were not weakened.

- `go build ./...`, `go vet ./...`, Go format, `git diff --check`,
  ADR governance, automation registry and sensitive-file checks pass.
- `TestEveryStateRootWriterDirectoryIsInventoried` passes; the remaining
  filesystem writer/load/freeze guards hit the ancestor ownership blocker.

## Review and remaining evidence

Review covers the closed successor/version boundary, historical immutability,
classification/qualification distinction, append-only facts, actor/authority
binding, exact-byte policy addresses, argv-only execution, private publication
and inventory/recovery coverage. No blocking code finding remains from local
review. Full filesystem/integration verification, separate contract acceptance
and later consumer Tasks remain explicit boundaries; this is not evidence of
real capability, lease/process fencing, canonical stream ingestion, protected
multi-object dispatch authority, installed command operation or T40 smoke.

## Host verification (outside Codex sandbox)

The ownership blocker above is specific to the Codex sandbox. On the box host
(UID owning ancestors), the complete `go test ./...` suite passes, including
filesystem policy publication/recovery and the writer/inventory guards, after
freezing `validator_policy` in stable-v1 snapshot `v0.12.0-issue276.b1` and
making the top-level directory comparison order-independent.
