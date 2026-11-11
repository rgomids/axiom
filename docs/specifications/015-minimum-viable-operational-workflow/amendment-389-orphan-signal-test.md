# Spec 015 amendment — #389 orphan signal test

2026-11-11; bounded implementation authorized by Rafael. Base `e5b9d10`.
References: [#389](https://github.com/rgomids/axiom/issues/389),
[#371](https://github.com/rgomids/axiom/issues/371) (related, out of scope),
[CR-S1-07](correction-cr-s1-07.md), [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md).

## Specification

The Linux observer must prove the live victim → runner → stub ownership chain,
stop the victim, and fail if any recorded process or member of the owned stub
group survives. Emergency cleanup remains failure, never evidence of success.
The test must use a bounded, generous wait informed by the test deadline and
must not mistake its own signal-delivery/wait assumptions for a product leak.
No product contract, CLI, dependency, ADR, vendor invocation or #371 fix.

## Plan and tasks (before implementation)

1. Inspect report/observation handshake, signal dispositions, process groups and
   `exec.Cmd.Wait` pipe behavior; distinguish demonstrated cause from hypotheses.
2. Run 20 SIGHUP repetitions with GOMAXPROCS=1 and two local CPU burners; attempt
   a controlled reproduction of the identified assumption before/after.
3. Change only the test harness, unless executable evidence proves a product
   defect. Preserve identity checks, exit checks and no-survivor assertions.
4. Run targeted checks, required full suite once, vet, repository/security
   validators and diff/format checks. Record actual results in separate evidence.
5. Review with Axiom review / Ponytail discipline; commit, push this branch,
   verify remote SHA, open one normal PR and observe head CI. No merge or status
   mutation; human review remains pending. Rollback is reverting this test fix.

Acceptance: before/after results and limitations recorded; all applicable checks
reported PASS/FAIL/NOT_RUN; no weakened safety assertion or unrelated edits.
