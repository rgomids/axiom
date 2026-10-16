# ADR-0028 — MVP POSIX process-group supervision boundary

## Status

Proposed. Records the [AXM-8](https://linear.app/rgomids/issue/AXM-8/mvp-run-and-coordinate-stage-agents-through-cli-and-runtime-skills)
maintainer decision of 2026-10-15 (Linear comment
`247f9812-31d2-4c62-af63-7fd972d18f54`, "simplificação T43 + CI mínimo") for
I276-T43 of Issue #276. The decision approves the reduced MVP scope and its
implementation on PR #320; this text still needs maintainer acceptance before
the T43 wording of the accepted Plan is annotated. No accepted ADR is changed.
No merge, release or operational acceptance is inferred.

## Context

I276-T43 asks for process containment with `Stopped` true only when the whole
group is confirmed gone, and a negative test where a descendant escaping the
group yields `unknown`. [ADR-0025](0025-public-stage-dispatch-and-capability-proof.md)
requires each child to run in its own process group; it does not require
tracking processes that leave that group.

To also catch descendants that leave the group *and* lose ancestry to the
leader (`setsid` plus double fork), the implementation had grown host-wide
inference: a Linux child subreaper, a global registry of attempt leaders, and
on Darwin a before/after snapshot of every launchd-adopted process. The Darwin
snapshot cannot tell an attempt's orphan from any unrelated process the host
starts. On CI run 38832717686 it turned ordinary exits into
`Stopped=false`/`OutcomeUnknown=true` on macOS. The same inference reproduces on
Linux: an unrelated orphan reparented through the subreaper vetoes a clean stop.
Attributing such processes correctly needs causal tracing, a privileged service
or OS sandboxing, which the MVP does not take on.

## Decision

For Linux and macOS, T43 supervises the attempt's main process and its own
POSIX process group:

1. Spawn the leader with `Setpgid`, record its kernel start/group identity, and
   refuse to signal or confirm when that identity changed (PID reuse fence).
2. Stop by `SIGKILL` to that group only; never signal another group or process.
3. Report `Stopped` only when the group is observably empty (`ESRCH`, or no
   live member with that process group id), within the existing bound.
   Otherwise, and on any identity or observation error, the result stays
   unconfirmed and the classifier keeps `unknown`.
4. A descendant reachable through the live leader's ancestry but outside the
   group is a *known escape*: confirmation is vetoed and it is left running.
5. Descendants that deliberately leave the group and lose ancestry to the
   leader are outside the MVP guarantee. They are neither tracked, signalled
   nor counted, and their effects are not proof of anything.
6. Exit code 0 is never proof of workflow success; a stage still needs its
   recorded result (unchanged classifier).

The Linux subreaper, the global leader registry and the Darwin adoption
snapshot are removed because their only purpose was the universal tracking
now outside scope. Process-table reads remain only for group membership and
leader ancestry.

## Alternatives considered

| Alternative | Trade-offs |
| --- | --- |
| A — group supervision with known-escape veto (chosen) | Small, deterministic, no privileges; external host activity cannot veto a stop. Deliberately decoupled descendants are not contained. |
| B — keep host-wide orphan inference | Catches more escapes in theory but misattributes unrelated processes, makes macOS stops unconfirmable on busy hosts and stays non-causal. |
| C — causal tracing, privileged helper or OS sandbox | Could contain decoupled descendants; needs platform entitlements, new infrastructure and security review beyond the MVP. |

## Consequences

### Positive

Stop confirmation depends only on the attempt's own group and proven ancestry,
so it is reproducible on Linux and macOS runners. The supervisor no longer
becomes a subreaper or reads unrelated processes into attempt outcomes.

### Negative / trade-offs

A Runtime that daemonizes a child out of its group can leave work running after
`Stopped` is reported for the group. This is a documented limit, not a
confirmation of that work's absence. The accepted T43 negative test wording
("descendant escaping the group yields `unknown`") narrows to *observable*
escapes; this changes the meaning of "confirmed stop" for #276 and is the part
that becomes harder to change later once stop receipts are persisted.

## Revisit when

A Runtime is shown to daemonize work during attempts, a platform offers
unprivileged descendant containment (for example cgroup ownership or a process
sandbox) that can be adopted without new privileges, or Windows job objects
re-enter scope.
