# Implement Evidence 010 — resumption 3

**P1 IN_PROGRESS; G1 not approved; T05+ NOT_RUN.**
Same Implement continuation, Draft PR #335, entry head
`790b1a2e648146700a63b42c70e18669861f0c8c`; clean entry worktree.
Canonical Execution remains unknown; none recreated or dispatched. Linear read
In Review, blockedBy empty, blocks AXM-39/AXM-12 unchanged. Entry-head CI and
Delivery metadata confirmed SUCCESS (CI run 39500006601); not new-head Evidence.

[C1](acceptance-resume-3.md) disposes only the quota-freeze ordering, appended to
Plan/Tasks without rewriting accepted history. [Contract closure](contracts-p1-resume-3.md)
and [oracles](contract-oracles-resume-3.json) record T01 consistency PASS separately
from NOT_RUN future codec/migration tests. v0.1.0 is still an unqualified objective;
public quotas remain conditional on full T05/T06 measurements before T07.

## E02 — experiment in progress

Disposable cgo/Go reader and Python synchronization harness; no product code.
Local macOS arm64/APFS completed 22 cases plus fresh B (exit 0). Initial harness
run refused valid fixture because it expected 16 binding keys instead of accepted
15; corrected. Mutating helpers explicitly chmod only disposable fixture parent
directories for controlled faults. Second run passed 20 cases; final expanded
local run passed 22. These are harness revisions, not hidden product failures.

Linux native amd64/arm64 passed the same 22 cases plus fresh B at
`fa599696db5a85b2a7abead971eca3a6ce5a0fd8`, run 39503787947.
[Sanitized per-host results](probe-results-resume-3.json) preserve exact hashes,
barriers, refusals and consumed content digests. Temporary branch-only read-token workflow
is authorized solely for these bounded experiments and will be removed with its
registry entry after collection. Source hashes and sanitized outcomes will be
retained. No cross-build, Docker/emulation or missing job counts as native proof.

## Remaining graph and authority

T01 documentary/model closure awaits fan-in review; T02 native fan-in in progress;
T03 depends on adequate T01/T02; T04/G1 remains pending complete E01–E03 and human
review/disposition. T05–T16, G2/G3 and AC-01–AC-08/full V1–V9 NOT_RUN.
No automatic SDD review, merge, release, closure, functional acceptance or status/
blocker changes. Normal push and same Draft PR updates only.

No delegation: runtime exposes overrides but no demonstrable strictly inferior
model ordering, so main Orchestrator performs integration/probes. This is not an
independent review. Next: collect both native jobs, assess failures, then T03/T04
or exact blocked checkpoint. Human G1 approval remains mandatory before T05+.

## E03 — transition fixture model in progress

After both native E02 jobs completed, a separate scratch Python model exercised
installation-derived persistent flock, different receipt roots, stale expected
binary/lock revisions and read-back barriers. Local macOS passed 21 transition
rows, three foreign/link lock refusals, lock replacement and process-concurrency
checks. Two child processes exited 0; same-installation contender was busy while
another installation acquired independently. No production lock is implemented.

Early harness failures: multiprocessing startup needed a main guard; new lock
content needed descriptor rewind and private directory creation. A substantive
macOS EACCES rejected moving an already-0500 generation across parents; revised
model stages within `generations/` and renames within that same parent. No
permission repair of existing material is proposed. Errors and their corrected
source revision remain distinct from production fault qualification.

Linux E03 native jobs pending. Stage-write/space/quota/permission/sync and rename/
post-commit errors are **deterministic injected outcomes**, not physical storage
failures. Binary rename/read-back and retained complete-generation checks are
actual filesystem operations. Runtime receipt/ack placeholders are model effects,
not candidate proof or Runtime convergence. Material algorithm adoption awaits T04
and human G1 disposition; no automatic SDD review or T05+.

## E02 follow-up — release-compatible build experiment

Inspection found the current builder forces CGO_ENABLED=0 and prepares all archives
on Ubuntu. Before asking for a build change, a bounded alternative experiment kept
that setting: stdlib Syscall6 on Darwin, `/proc/self/exe` descriptor on Linux.
Local native macOS arm64/APFS passed **24 cases plus fresh B**, including an actual
API-error-path control (invalid read-only region-info flavor) and a deliberately
closed binary descriptor. `go version -m` confirmed CGO_ENABLED=0/darwin/arm64.
Linux native repeats pending; no release adapter or ABI support claim adopted.

Local Apple SDK C offsetof/sizeof measurements returned 1272-byte region record,
96-byte device offset and 104-byte inode offset. Primary Apple XNU libproc source
and proc_info_private header corroborate the read-only query selector, but direct
syscall/ABI stability is a material review question, not proven by source lookup.
No Apple implementation is copied into product code; no dependency/build/workflow
publication change follows. This preserves the earlier cgo positives separately.

Pinned v1 shell/Go dispatch checks also passed: v1 passive Go load accepted;
proposed exact v2 record and unknown metadata version rejected. Shell invalid-input
sentinels remained absent (exit 1), temporary Go test removed. These old-layout
fixtures do not prove complete v2 archives or future v2 consumers. Existing facade
fixture suite exit 0, failures=0; four affected Go packages passed.
