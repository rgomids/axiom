# Adapter Failure Evidence

## Tested behavior

| Case | Injection / observation | Result |
|---|---|---|
| Spec-Kit unavailable | `SPECIFY_LAUNCHER=/path/that/does/not/exist` | Exit 69, `adapter_error=spec_kit_unavailable`, no workspace/result |
| Pinned ref unavailable | Real `uvx` resolution of `v0.0.0-axiom-missing` | Exit 1, raw upstream Git resolution error visible, no capability result; partial temp workspace requires deletion |
| Output incompatible | Fixture lacks neutral contract fields | Exit 65, `adapter_error=incompatible_output`, no normalized file |
| Existing normalized destination | Pre-existing valid file | Exit 73, `adapter_error=output_exists`, existing bytes preserved |
| Capability failure | Injected executable exits 42 | Wrapper exits 70, `adapter_error=capability_failure`, no result |
| Converge finds gaps | Actual pinned run | Only disposable `tasks.md` changed: 9 lines and 6 appended tasks |

`test-failures.sh` deterministically covers the first, third, fourth, and fifth cases.
The invalid-ref output and time are retained in `evidence/failures/`.

## Upgrade check

At execution time, GitHub's latest stable release was still `v0.16.2`, the
pinned version used here. The annotated tag resolved to
`4871b485f97c7fa452ec58eba325d87536c55c34`. There was no newer stable version,
so no meaningful pinned-vs-newer execution could be performed. Running the same
tag twice would not test upgrade compatibility.

Upgrade migration, changed output compatibility, and rollback across two real
stable versions remain unverified. A new stable release is the smallest
external-state change needed for that test.

## Operational observations

- The actual adapter failed closed and preserved raw events.
- The post-run review added an output-exists guard so a failed new run cannot
  leave a stale normalized result looking current.
- The invalid-ref path exposes the upstream error but does not normalize that
  init failure; callers must interpret exit 1 and clean the partial temp tree.
- Converge is intentionally mutating, so cleanup must discard the entire
  disposable workspace rather than copy `.specify/` or modified tasks back.
- During adapter development, an inspection command omitted its temporary
  `cd` and initialized Spec-Kit at the Axiom root. Git showed only known
  untracked generated paths; all eleven exact directories were moved to
  recoverable quarantine and root state was reverified. The final adapter uses
  a subshell with explicit `cd`. This demonstrates real working-directory and
  cleanup risk even with isolated intent.
