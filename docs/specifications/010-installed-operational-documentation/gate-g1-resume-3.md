# T04 / G1 — resumption 3 submission

**G1 BLOCKED. D2 is not qualified; no T05+ or algorithm adoption.**
Same AXM-37 Implement and Draft PR #335. This is P1 integration/self-assessment,
not independent review, SDD `review`, functional acceptance or a new Execution.
[C1](acceptance-resume-3.md) is accepted; it does not approve G1.

## E01–E03 fan-in

| Input | Executed evidence | Qualification withheld |
| --- | --- | --- |
| E01 / T01 | [Contract closure](contracts-p1-resume-3.md), [oracles](contract-oracles-resume-3.json); 14 consistency checks; exact v1 pins; 105 leaves/549 registrations/15 pages; pinned Go accepts v1, rejects exact v2 record/unknown version; pinned shell rejects both invalid records, untouched execution sentinels | Future v2 consumers and full archive/migration matrix NOT_RUN. Record fixtures are not an installable v2 corpus. Historical native migrations UNVERIFIED. C1 quotas are development budgets; final freeze requires full T05/T06 measurement/coverage before T07 |
| E02 / T02 | [Three native rows](probe-results-resume-3.json): CGO-disabled prototype passed 24 cases + fresh B per host; earlier cgo prototype passed 22 + fresh B. A survives atomic replacement, same-label bytes, post-launch PATH, symlink/spaces, unlink, copy/relocation refusals, ancestry/content changes and complete three-output snapshots. Native API-error/stale-descriptor controls added | **Supplemental macOS in-place-before-pin whole-image byte oracle FAIL below.** Neither prototype selected. No production ACL/permission adapter, all-filesystem, man/offline/Runtime qualification. Explicit alias-hop-boundary rows remain NOT_RUN |
| E03 / T03 | [Three native fixture rows](transition-results-resume-3.json): 21 transitions, two-process same-target/different-receipt-root contention, independent targets, alias identity, stale revisions, three unsafe-lock refusals and lock replacement; complete generation → binding → actual binary rename/read-back | Python fixture model, not production installer qualification. Storage errors injected; receipt/skills/ack are model effects, not candidate/Runtime proof. Results predate the supplemental D2 counterexample and require reconciliation with a future qualified mechanism |

Latest native run at `1206ba9b02767ed23d52fd6dc4df3fb429e44384`:
[amd64](https://github.com/rgomids/axiom/actions/runs/39505487190/job/118913220022),
[arm64](https://github.com/rgomids/axiom/actions/runs/39505487190/job/118913220395),
both SUCCESS on their **bounded 24-case suite**, repeating the 21 T03 model rows.
That green workflow does not contain or override the subsequent local negative
control. Its overall D2 qualification is therefore **UNQUALIFIED**, not PASS.
macOS local rows: native arm64/APFS, identical source/harness hashes. Linux `stat`
reported `ext2/ext3`; no broader filesystem claim. Docker not rerun: previous
arm64 primitive PASS complementary, amd64 emulated image-proof FAIL negative.

## C2 — executing-file mutability before binary pin

**Material D2 blocker; human assessment required before selecting a mechanism.**
In the supplemental native macOS experiment, `open(O_WRONLY|O_APPEND)` on a live
executable succeeded. Between initial OS-image observation and binary-descriptor
pin, append changed its bytes/hash while preserving device/inode and active A.
A synthetic internally consistent binding for that changed file selected the
fixture B generation. The reader returned `pinned`, active tag A, with **all three
B content hashes**, not A's generation or refusal. Process exit 0 is the observed
program result; the experimental identity oracle result is **FAIL**.

This falsifies the tested rule "mapped device/inode equals opened file, therefore
its later full-file hash identifies the launched image" under this controlled
case. It does **not** prove every possible OS mechanism infeasible or a production
vulnerability: no product resolver exists. Appended bytes need not change mapped
instructions. The binding is a synthetic changed input, not an authenticated
release. The distinction between whole-file identity, mapped code identity,
input authenticity and the ADR-0005 boundary must be reviewed explicitly; matching
local hashes cannot solve authenticity. Do not silently classify this controlled
mutation as an excluded continuously hostile same-UID interleaving.

### Exact replay protocol

Use the `main.go`/`run.py` heredocs at
[1206ba9 workflow source](https://github.com/rgomids/axiom/blob/1206ba9b02767ed23d52fd6dc4df3fb429e44384/.github/workflows/axm37-p1-experiment.yml).
Compile A/B with CGO_ENABLED=0 and their existing `-X main.tag=A/B` flags. On native
macOS, create A/B fixtures using that harness's unchanged `fixture` function:

1. Start A and wait for its first `image` record, before sending its first newline.
2. Append exact UTF-8 bytes `AXM37-controlled-append\n` to A's target with
   O_WRONLY|O_APPEND. Observe unchanged target device/inode and changed SHA-256.
3. Copy B's complete synthetic generation to A's `generations/<B-generation>`.
   Copy B's binding object, replacing only binaryPath, parentDevice/parentInode,
   binaryDevice/binaryInode and binarySha256 with A's current physical facts and
   modified bytes. Emit compact JSON plus LF; store at the accepted binding path
   keyed by modified binary SHA and exact binding SHA. Retain A's old binding and
   complete generation. This is a hostile-input model, not publication authority.
4. Resume every existing newline barrier. Compare active tag A, original A binary
   SHA/generation, modified same-inode SHA and actual consumed hashes. Observed
   `replacementContentAccepted=true`, `originalContentAccepted=false`.

Exact observed hashes, phases, API/source/harness identity and OS/filesystem are
in [the counterexample object](probe-results-resume-3.json). Linux in-place variant
NOT_RUN; one supported native counterexample suffices to withhold adoption. No
further dependent experiment or production was started after this finding.

### Mechanism alternatives, not adopted

| Direction | Trade-off / disposition needed |
| --- | --- |
| Preserve exact identity requirement and strengthen the mechanism — recommended | Demonstrate that a changed executing file is refused or retains the original complete A snapshot, including the startup boundary; independent kernel/image metadata and immutable snapshot timing need a new measured proposal. An inode/hash-only guard is insufficient. Further P1 work stays within existing authority; it cannot accept G1 or invent a stronger OS guarantee |
| Change the supported mutation/threat or byte-identity requirement | Material contract/architecture change, not an implementation shortcut. Requires explicit human disposition and ADR-0018 process where applicable; preserve the failed oracle. Do not narrow AC/D2 or call this case out of scope by inference |

A separate implementation choice remains: the cgo/libproc route proved earlier
versus a CGO-disabled direct Darwin syscall prototype. Current builder forces
CGO_ENABLED=0 and prepares all archives on Ubuntu. The latter passed the bounded
24-case native suites, using SDK-measured region-record size 1272, device offset
96 and inode offset 104. [Apple's libproc wrapper](https://github.com/apple-oss-distributions/xnu/blob/main/libsyscall/wrappers/libproc/libproc.c)
and [private query selector definitions](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/proc_info_private.h)
corroborate the query, not ABI stability or original whole-file identity. No
Apple implementation is copied into product code. Raw ABI adoption or a native
macOS/cgo build topology needs review; neither repairs the counterexample merely
by existing. No new dependency, builder, release workflow, SDK or infrastructure
change adopted. Harder to change after shipment: OS/ABI support and release build
envelope, binding identity and public discovery guarantees.

## D4 fixture proposal — not a production adoption

Installation key: physical parent device/inode plus exact `axiom` leaf, never
binary inode/hash or receipt root. Proposed persistent lock:
`<parent>/.axiom-docs/axiom/installation.lock`, owner-only regular 0600, nlink 1,
no-follow; bounded recognized bytes `axiom-install-lock-v1\n<dev>:<ino>\naxiom\n`.
New internal record/path remains proposed; existing marker/Runtime formats unchanged.
Nonblocking flock must bind/recheck lock object/content revision and every target/
effect revision under lock and before commit. A replacement file can acquire a
second OS flock; old-holder identity and stale-preview lock-revision checks must
refuse. Never unlink/recreate busy/unknown locks. Incomplete initialization may
conservatively refuse; no blind recovery.

Stage complete generation under the same `generations/` parent, verify bytes,
set immutable modes, publish final key, then complete binding before binary rename.
Cross-parent rename of an already-0500 directory failed EACCES on macOS; same-parent
model passed. Preserve/refuse conflicting/incomplete objects. Confirmed B remains
confirmed across receipt/skills/ack/cleanup errors. Unknown rename or marker state
requires inspection plus fresh exact authority; no automatic retry, generation GC
or rollback. Counts include preserved conflicting directories, not certified valid
generations. Actual Go integration must reuse existing confinement/ACL helpers;
stdlib/x/sys suffice as candidates, no lock framework/parser library adopted.

## Status and recoverable handoff

| Unit | Status / exact next requirement |
| --- | --- |
| T01 / E01 | PARTIAL: development contracts/oracles reconciled; full preserved-format/pre-execution and historical support claims remain qualified; any algorithm change needs reconciliation |
| T02 / E02 | PARTIAL / BLOCKED: positive native suites retained; supplemental macOS identity oracle FAIL; do not select either candidate without resolving C2 and remaining boundary coverage |
| T03 / E03 | PARTIAL: transition/lock fixture model passed; cannot adopt production lock/layout against unqualified D2; reconcile after T02 |
| T04 / E04 | PARTIAL: fan-in submitted **G1 BLOCKED**, independent P1 engineering/security review and exact-revision human material disposition pending |
| T05–T16 | BLOCKED / NOT_RUN; no dependent production started |
| G2/G3, AC-01–AC-08, full V1–V9 | NOT_RUN; no synthetic/native primitive result closes them |

Next: human assessment of the C2 counterexample and material mechanism direction;
then revalidate head/authority and resume T02 proof, reconcile T01/T03, complete
independent P1 review and obtain explicit human G1 approval. C1 still blocks T07
on incomplete coverage/over-quota results; any material revision needs comparison
and human disposition. No automatic SDD review, new Execution/PR, Linear change,
AXM-39/12 unblocking, merge, release, closure or functional acceptance.
