# P1 contract closure — resumption 3

**Development contract, pending T04/G1 review; not production qualification.**
The exact accepted [revision-1 tables](contracts-proposed.md) govern versions,
fields/types/order, layouts, states/reasons/exits, the identity DAG and shared man
mapping. [D1/D3 acceptance](acceptance-resume-2.md) and [C1](acceptance-resume-3.md)
qualify those tables without editing them. No new public schema or codec is added.

## T01 deliverables and executable checks

[Record vectors](contract-vectors-proposed.json) contain exact positive UTF-8
metadata/receipt/TSV bytes, binding object and compact-JSON-plus-LF binding hash,
two synthetic output records and four discovery envelopes. These are record-level
fixtures, deliberately not an installable full release/corpus. The preserved
[CLI map](cli-coverage-proposed.json) defines every existing leaf, flag/default,
shared page/anchor and the future docs-inspect leaf. It does not claim rendered
T06 coverage. Expected skill manifests retain the existing four active names;
historical ownership inventories remain separate.

[Closed oracles](contract-oracles-resume-3.json) bind their inputs and preserved
v1 owners by SHA-256. A scratch consistency check replayed the v1 pins, exact
control hashes, metadata/binding/receipt links, generation DAG, output lengths and
hashes, 105 unique leaves/549 registrations, 15-page selection and four state/exit/
path/loaded invariants: **14 checks PASS, exit 0**. This does not execute a future
v2 parser. Replay: hash pinned `git show <baseline>:<path>` bytes; split ordered
positive KV at the first `=`; recompute each referenced SHA-256; encode binding in
its listed order with compact JSON plus LF; recompute the documented generation
preimage; compare inventory lengths/hashes and exact coverage/state invariants.

The oracle records 148 per-key removal/duplicate/reorder/unknown-key test
specifications for the two KV formats. Reordering means exchange with an adjacent
key (the last exchanges with its predecessor); no exchange with a nonexistent
key. JSON duplicate/escaped-duplicate, unknown/trailing values and all other
negative record/archive vectors remain required exactly as revision 1 specifies.
**These mutation outcomes are specifications, not executed codec tests.**

Quota boundary clarification: for each of 27 measured/bounded surfaces, N−1 and N
are inside that quota; N+1 refuses before candidate execution. Other invariants
still apply, so an inside-quota malformed record is not accepted. The historical
vector labelled `quota-boundaries` does not mean all three values reject. Zero
candidate calls applies to rejected complete inputs, never as evidence fabricated
from a documentary vector. C1 retains every initial safety limit during development.

## Producer/consumer ownership and pre-execution oracle

The JSON matrix distinguishes pinned v1, future v2 shell/Go, historical migrations,
reinstall/upgrade and unqualified/copied/downgrade cases. T07 owns archive production
and differential shell/Go validation; T09 owns receipts/effects; T11 owns discovery
presentation. Shell/Go must validate checksum, archive closure/types/quotas, exact
metadata/skill/control schemas, cross-record identities and every payload before
candidate execution. Invalid-input sentinel remains untouched; old consumers
reject v2. No runtime/skill format is changed. Positive candidate execution and
Codex receipt authority remain ADR-0016's existing contract; Claude remains separate.

Pinned v1 tests may pass now; future v2 consumer/migration tests remain NOT_RUN.
All historical native v1→v2 migrations remain UNVERIFIED. The existing 25-source/
11-receipt/three-oldest-archive inventory is [bounded E01 evidence](evidence-resume-2.md),
not a qualified v0.1.0 support claim. No fixture or historical bytes are rewritten.

## T02 experiment boundary

Probe pin-at-start against descriptor-plus-independent-OS-proof. Linux uses
`/proc/self/exe`; macOS uses `proc_pidinfo(PROC_PIDREGIONPATHINFO)` at a function
address in the executable's mapped image. Compare image device/inode against an
opened, no-follow physical binary; hash pinned bytes, select binding by digest and
exact original binary/parent identities. Equal version labels are irrelevant.
Unprovable identity refuses. Experiment layout follows accepted generation/binding
paths with synthetic complete three-output inventories, not a production corpus.

Pin all three validated controls, then validate and copy every inventoried content
file into one bounded memory snapshot before exposing content. Use anchored roots,
no-follow regular-file opens, owner/mode/link checks, boundary identity rechecks and
inventory closure. Replacing content before its validated open must refuse;
replacement after snapshot must consume retained A bytes. A future JSON path open
cannot inherit this snapshot and needs fresh validation. Missing syscall/observer
proof remains a gap; no pathname-only algorithm is adopted.

Cgo's native API bridge in the disposable harness is experimental, not a new
production dependency or selected Go adapter. Existing `golang.org/x/sys`, stdlib
`os.Root`, hashes/JSON and local confinement helpers remain the production options;
no additional library, cryptography or generic parser is adopted. Mechanism
selection and its bounded security assumptions require T04 and human disposition.
