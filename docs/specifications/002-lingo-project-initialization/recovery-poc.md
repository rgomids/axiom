# POC filesystem recovery — operator procedure

This procedure applies only to the one-manifest Project POC. `recovery_required`
stops normal access when a recognized interrupted stage or attempt marker remains.
The CLI does not repair or delete recovery artifacts. Unknown artifacts are never
cleaned automatically.

## Classify the observed state

| Observation | Publication state | Safe interpretation |
|---|---|---|
| Stage or temporary file exists; final target absent | Not published | No new Project or installation record is authoritative. |
| Update temporary file exists; old `axiom.yaml` remains | Not published | Old complete manifest remains authoritative. |
| Attempt marker exists; create/install final target exists, or updated manifest matches the proposed new bytes | Published, durability uncertain | The rename was observed. A crash may have occurred before directory sync or marker cleanup; do not claim the operation was durably acknowledged. |
| Attempt marker exists; updated manifest still matches the prior bytes | Not published, unless the proposed bytes were identical | Prior manifest remains authoritative; compare exact saved bytes before cleanup. |
| Final target exists without interrupted artifacts | Normal candidate | Validate/reopen through Lingo; physical presence alone is not validity. |
| Unknown artifact, malformed final target, unsafe link or ownership | Invalid or unsafe | Preserve all files and investigate. No automatic cleanup. |

`Create` publishes the Project directory with no replacement. `Update` atomically
replaces the one manifest. `Install` publishes one ID-addressed local record with
no replacement. Portable and local roots do not form one transaction. A local
installation failure does not roll back a valid portable Project.

## Manual recovery

1. Stop all Lingo processes using the affected roots. Record command, operation,
   slug or Project ID, platform, and the `recovery_required` JSON event.
2. Make a separate, access-restricted copy of the affected portable Project root
   or local ID directory, including hidden files, before moving any artifact.
   Record file names, types, permissions, sizes and SHA-256 hashes of regular files.
3. Inspect the copied evidence and the original with no-follow tools. Identify
   only the interrupted artifacts from this operation. An unfamiliar artifact,
   link, ownership change or conflicting operation stops recovery for review.
4. Determine whether the final target is absent, contains the prior complete
   value, or contains the newly published complete value. Validate its schema,
   immutable ID and expected revision against the original intent and saved
   Evidence. A surviving published value after a crash has uncertain prior
   durability; record that fact even if it validates now.
5. After explicit operator review, move only positively identified stage,
   temporary and attempt artifacts into an access-restricted quarantine outside
   the active root. Retain that quarantine and the initial copy for audit. Do
   not remove unknown artifacts or replace a final target to force success.
6. Run `lingo project validate --slug <slug>` and `lingo project reopen --slug
   <slug>` again. For a local install, rerun `lingo project install --source
   <absolute-project-path>` only when no final record was published. Record
   results and hashes. If classification remains ambiguous, leave
   `recovery_required` unresolved.

This is an explicit operator procedure, not an automatic recovery guarantee.
The POC does not prove recovery under all disk-full, ACL, hostile same-user race
or abrupt power-loss conditions. Those gaps remain in [POC Evidence](evidence-poc.md).
