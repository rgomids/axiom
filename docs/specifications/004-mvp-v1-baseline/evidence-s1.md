# S1 Evidence — Result, provenance and protected local substrate

## Scope and claim

This record covers the authorized remainder of S1 in GitHub Issue #75: T02 and
T03 after accepted/merged T01. It supports the claim that Axiom can create and
read one bounded, durable, machine-local Markdown detail artifact by opaque ID,
and that delivered local stores preserve old-or-new commit truth and fail closed
when publication state is uncertain.

This is implementation Evidence, not human acceptance. It does not authorize or
claim S2–S7, cleanup/recovery mutation, Provider effects, Git remote mutation,
Runtime installation, migration, release, or physical power-loss durability.

## Delivered contracts

- `internal/detailartifact` owns closed metadata v1, initial Plan guardrails,
  structural sensitive-value rejection, correlation, SHA-256 integrity,
  retention classification, and required/optional completion materialization.
- `internal/local.ArtifactStore` owns
  `<state-root>/artifacts/v1/objects/<prefix>/<uuid>/` and publishes exactly
  `metadata.json` plus `details.md` as one protected directory.
- `internal/local` owns the shared F0–F8 vocabulary, versioned bounded recovery
  marker, private staging, protected create/update, canonical confirmation,
  committed-state reporting, and fail-closed reader checks.
- Work Item and workflow records now carry exact observed byte revisions for
  stale-authority rejection. Project and installation stores retain their
  existing anchored directory/file publication and now write versioned attempt
  metadata under the same logical protocol.
- Coordination order is managed state root, store namespace, Project/object
  scope. Locks are kernel-backed and acquired broad-to-narrow. A conflict does
  not wait, retry, or infer lock ownership from PID/time.

Commit point: successful protected rename of the complete canonical file or
directory. Before that point, the prior complete generation remains authority.
After canonical bytes are confirmed, later acknowledgment/secondary/cleanup
failure reports `Committed=true`; it never claims rollback. Any surviving stage
or marker makes the relevant reader return `recovery_required`.

## Bounds and capacity

| Guardrail | Enforced value | Failure behavior |
|---|---:|---|
| Markdown per artifact | 1 MiB | reject before publication |
| Metadata per artifact | 64 KiB | reject; no field truncation |
| Captured output represented per artifact | 256 KiB | reject before publication |
| Live artifacts per state root | 10,000 | explicit capacity error; no eviction |
| Aggregate artifact content | 1 GiB | explicit capacity error; no eviction |

Lookup accepts only a lowercase UUID v4 and returns `artifact:<uuid>` after
validating object identity, closed metadata, content size/digest, owner-only
mode/ACL, regular-file type, and link count. Paths are never lookup authority.

## Fault and reader matrix

| Stage | Injected/observed case | Required and verified truth |
|---|---|---|
| F0 | cancellation/error after coordination | no canonical mutation; prior readable |
| F1 | no progress, short-write loop, ENOSPC/EDQUOT/error before complete stage | no commit; absent stage cleaned or interruption recognized |
| F2 | interrupted/corrupt/unsafe prepared object | no publication; reader fails closed when stage survives |
| F3 | cancellation/stale expected revision/target recheck | prior authority; stale write rejected |
| F4 | prior-generation boundary / pre-publication error | prior remains complete; uncertainty remains marked |
| F5 | collision/rename publication error | old or new complete canonical object only |
| F6 | interruption after rename before acknowledgment/confirmation | committed only when canonical generation is confirmable; otherwise recovery required |
| F7 | acknowledged primary plus secondary failure | primary remains committed; caller can classify truthful partial |
| F8 | cleanup error/identity uncertainty | canonical commit preserved; exact leftovers retained and readers fail closed |

Artifact tests exercise F0–F8 over the multi-file directory publication. Work
Item tests exercise F0–F8 over shared single-file create/update publication.
Workflow stale-revision tests use the same single-file implementation. Existing
Project and installation process tests exercise real process death before and
after their directory/file commit points. Multi-process artifact and Project
tests use pipe barriers as the oracle; sleeps only keep the helper alive after
the barrier.

## Security and confinement observations

- Synthetic token/private-key/raw-chat sentinels are rejected and not persisted.
- Invalid UTF-8, control characters, oversized content/metadata, unknown or
  duplicate metadata fields, invalid correlation, and digest mismatch fail.
- Symlink, hard-link, wrong type, permissive mode/ACL, collision, replacement,
  unsafe root, and arbitrary-path lookup cases fail closed.
- Isolated portable and Repository sentinel trees remain byte-identical after
  artifact creation; production code has no Provider, Git, Runtime, network, or
  process-execution port.
- Capacity failures preserve existing objects and perform no automatic cleanup.

## Reproduction

Run from repository root without Makefile indirection:

```bash
go test -race -shuffle=on -count=3 \
  ./internal/detailartifact ./internal/local ./internal/completion \
  ./internal/provenance ./internal/workitem ./internal/workflow
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
./scripts/test-install-axiom.sh
./scripts/dogfood-poc.sh
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
gitleaks dir . --no-banner --redact
git diff --check
```

The final S1 run recorded every command above with exit 0. The focused race run
was repeated three times with shuffled test order. No remote Provider or Git
mutation occurs in these commands; dogfood uses synthetic local/fake boundaries.

## Native observation and deferred matrix

Native execution for this delivery:

| Field | Observation |
|---|---|
| OS | macOS 27.0, build 26A428 |
| Kernel / architecture | Darwin 27.0.0, arm64 |
| Filesystem | APFS (`/dev/disk3s3s1`) |
| Go | go1.26.1 darwin/arm64 |
| baseline `main` | `5d4ca4babe8b63a33da3d8d20cfd2b19415c36a5` |

T22 retains the approved obligation for native Ubuntu 26.04 amd64/arm64 ext4
and final exact-target reruns. This S1 delivery does not convert current CI or
cross-compilation into those native claims.

## Remaining limitations

- Explicit artifact cleanup is T18; guided mutation of recognized recovery state
  is T19. S1 preserves and diagnoses uncertainty but does not auto-repair/delete.
- The guardrails are approved initial defaults, not benchmark-derived production
  capacity promises. Later dogfood/RC review may propose changes without changing
  artifact identity or ownership semantics.
- Same-UID arbitrary hostile interleavings, physical power loss, and media
  durability remain excluded by ADR-0005.
- S2 was not started.
