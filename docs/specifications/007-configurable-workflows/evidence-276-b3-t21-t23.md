# I276-T21 / I276-T23 implementation record

Scope: [T21 and T23](plan-tasks-276-stage-agent-execution.md#i276-t21--consume-frozen-resolvedprofile-and-effort-source-checks-at-dispatch), Plan §§3.4–3.5 and 3.10, amendment A3/A6, #306 PE-03–PE-06/PE-05a, and the successor boundary in ADR-0026. This records implementation under the maintainer's explicit task request; it does not change historical approval statuses or claim runtime proof, release readiness, or completion of the public dispatch journey.

## Implementation

- `runtimeadapter.BuildFrozenStageInvocation` derives explicit effort or no-override semantics exclusively from the validated tuple, using the existing stage argument/environment builder.
- `graphapplication.PrepareFrozenStageInvocation` consumes the delivered policy `Check` boundary with capability-backed observations. It compares the current tuple with the persisted tuple, refuses substitution, requires subscription authentication, and calls the delivered effort-source interface before and after authentication on every attempt. Missing/changed Profile or proof blocks before invocation delivery.
- `coordination.Ingestor` consumes sequence-ordered inbox candidates under an attempt fence, validates lineage and chain envelopes, checks the per-attempt terminal reserve, resolves question/contract references against canonical streams, assigns canonical revisions, and publishes through the existing coordination store. Canonical source provenance includes the trusted inbox epoch. Crash replay recognizes already-published identical records without duplicating them.
- `local.AttemptInbox.Consume` shares the Append directory lock, persists a bounded cursor/rejection journal through the existing protected publication protocol, and drains before persisting the seal. Previous-epoch candidates remain quarantined in the private inbox and cannot qualify outcomes. Append refuses sealed inboxes, including after reopen.
- `local.CoordinationFence` pins the existing run lease and checks a freshly loaded running-attempt binding. `Ingestor.Classify` supplies the final-drain/classifier seam for the successor supervisor's terminal CAS. The caller must load canonical attempt state and share the same lease with terminal writes.

The private inbox journal is outside canonical state-root storage. Historical Graph codecs/corpus, resolution algorithms, source inspection implementation and coordination CLI reads are unchanged. Supervisor/public command composition remains with its owning tasks.

## Test source added

All added tests use fake boundaries, fake child provenance, or inert temporary files. They do not launch Codex/Claude, run probes, use network/inference/Providers, or read/write HOME.

- Explicit and runtime-default effort for both adapters; checks repeated for each attempt.
- Drift of every tuple field; missing proof; refusal to fall back to runtime-default; external selectors, source-digest drift, selectors changing during authentication, and safe categorization of opaque inspection failures.
- Delivered resolver checks for removed, tombstoned, edited and disabled local Profiles and absent/wrong-model proof.
- Delivered X05 effort inspection against fake executable bytes and temporary declared settings, including selectors inserted after the clean inspection.
- Two fake children exchanging a question/answer across canonical streams; concurrent drains; blocker before exit and exact classifier correlation.
- Forged lineage, stale graph/chain revisions, missing question/contract reference, previous-epoch quarantine, capacity exhaustion with terminal reserve, terminal-state fencing and sealed late publication.
- Crash replay/reopen, live/closed lease fencing, and publication racing the final drain.

## Verification and limitations

- `gofmt` applied to all changed Go files; formatting and diff whitespace checked.
- No `go test` command was run, as explicitly requested. Added test behavior remains unexecuted.
- An offline `go vet` attempt stopped before analysis: the available `/usr/bin/go` is 1.24.4, while `go.mod` requires 1.26.0. Compilation/static type analysis remains unverified; no toolchain download was attempted.
- Manual review covered frozen tuple ownership, fake-only test boundaries, lease/inbox lock order, sealed late writes, canonical-only classifier input, bounded diagnostics, recovery refusal, and historical format isolation.
- No commit or external effects were performed. The existing untracked B2 evidence file was preserved.
