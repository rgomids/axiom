# CR-S1-06 correction: cutover inventory catalog

Attempt `CR-S1-06-a27f3c9` on [AXM-42](https://linear.app/rgomids/issue/AXM-42) /
[PR #366](https://github.com/rgomids/axiom/pull/366), from
[review 5729356147](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5729356147).
Executor: Claude Opus 5.5 (`claude-opus-5-5`). This is a scoped correction, not
an independent review: `INDEPENDENT_REVIEW_PENDING`. It does not re-review the
PR's other files. [`review-s1-full.md`](review-s1-full.md) and its ledger are
historical records and stay unchanged.

## Identity

| Item | Value |
| --- | --- |
| Base | `440e44a68638e08e901fd3f87f5b15cd395a6365` |
| HEAD before correction (local = origin = PR head, worktree clean apart from ignored `tmp/`, `scripts/__pycache__/`) | `a27f3c97fe211850f113c18a3a9942f51ed42e88` |
| Correction commit | recorded in the PR and the AXM-42 event |

## Finding

`cutoverKnownRoots` omitted `capability-evidence`, `capability-probes` and
`project-context`. `local.StateRootDirectories` lists all three. `InspectCutover`
therefore reported `unknown_state_entry` for each one and blocked every normal
historical state root that held them. Of the 15 stable-v1 corpus snapshots,
11 have `capability-evidence` and 14 have `project-context`. None has
`capability-probes`, so that fixture is synthetic. The prior
review recorded this as deferred D-02. S1.8 requires a truthful, usable
inventory, so this correction reclassifies the catalog part of D-02 as in scope.
The "no operator command" part stays deferred by design.

## Disposition by stored meaning

| Root | Meaning and source | Treatment |
| --- | --- | --- |
| `capability-evidence/v1` | Capability Proof records. No writer since admission certification was retired (#334); stable-v1 corpus | Legacy family: the namespace must be exactly `v1`, every file is hashed into the inventory and byte-backed-up, never decoded |
| `capability-probes` | Probe envelopes (`envelope.json`, `workspace/`, `inbox/`, `sentinel/`). Leftovers were quarantined by default with `.axiom-recovery-probe-quarantined`. No writer since #334 | Legacy family: the whole subtree is hashed and backed up. The quarantine marker is settled history only as a regular file directly inside an envelope. Anywhere else, or as a directory, it still BLOCKS as `interrupted_publication`. A `.probe-cleanup-*.json` intent BLOCKS as `interrupted_probe_cleanup` |
| `project-context/v1` | Live Project context, read and written by `InstallationStore` and used by `execution run` | Known live root, like `projects` and `work-items`: stays in place and is not part of the legacy backup. A file in place of the directory still BLOCKS |

Unchanged fail-closed checks: unknown top-level names, symlinks, irregular
files, unknown namespaces, `.axiom-stage-`/`.axiom-recovery-` leftovers, held or
unreadable leases, unreadable records or directories, and changed or omitted
bytes. Byte-level backup verification is unchanged too. There is still no
public cutover or migration command.

## technical_decisions[]

- **TD-C1:** retired evidence and probe envelopes are legacy families, not
  ignored roots. Ignoring them would drop history from the backup.
  Alternative: list them as known live roots, which keeps the bytes in place
  but out of the backup.
- **TD-C2:** Project context is live state, so it is excluded from the legacy
  backup, as other live roots already are. Alternative: back it up too, which
  would treat current state as history.
- **TD-C3:** exempt only the exact quarantine marker at envelope depth. Do not
  decode probe manifests (the cutover never decodes legacy state). Block on
  cleanup intents.
- **TD-C4:** a drift test asserts every `StateRootDirectories` entry is in
  `cutoverKnownRoots`. A future state root then fails a test instead of
  silently blocking cutover.

## Evidence

| Check | Result |
| --- | --- |
| BEFORE (fix absent; the new tests compiled with a temporary marker constant, since removed) | FAIL as expected: `TestCutoverCatalogCoversEveryInventoriedStateRoot` (3 missing roots), `TestCutoverHistoricalRecognizedRootsAreInventoriedAndBackedUp` (`unknown_state_entry` × 3) |
| AFTER `go test ./internal/local -run TestCutover -count=1 -v` | PASS: 14 tests, including 10 fail-closed subcases (truly unknown root, unknown evidence namespace, interrupted probe cleanup, staging leftover, marker at wrong depth or as a directory, workspace symlink, evidence interrupted publication, Project context replaced by a file) |
| `go test ./internal/local -count=1` | PASS |
| `go build ./...`, `go vet ./...` | PASS |
| `go test ./... -count=1 -timeout=30m` | PASS (exit 0; 32 packages `ok`, 0 `FAIL`) on the corrected tree |
| `go run cmd/gofmt -l cmd internal scripts`, `git diff --check` | PASS (no output) |
| `scripts/check-go-quality.sh`, `check-sensitive-files.sh`, `validate-repository.sh`, `test-docs-offline.sh`, staticcheck | NOT_RUN locally (shell approval unavailable); CI on the pushed SHA is authoritative |
| Vendor/E2E, real installed-state cutover | NOT_RUN, not authorized |

## Remaining limitations

- Probe workspace files over the per-file read bound (4 MiB) or in non-private
  directories BLOCK as `inventory_incomplete`. The cutover fails closed instead
  of taking a partial backup. The operator then needs a separately authorized
  disposition.
- An unquarantined envelope (crash before the marker) is backed up as inert
  bytes. Stopping any old probe process stays with the operator's
  `AttemptsDisposed` disposition: no lease exists to probe.
- D-01, D-03, D-04 and D-05 stay deferred, and D-02's "no operator command"
  part stays deferred by design.
- Independent full review of PR #366 is still the next gate.
