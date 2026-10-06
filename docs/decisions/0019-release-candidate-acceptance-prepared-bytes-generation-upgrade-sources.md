# ADR-0019 — Release candidate acceptance on prepared bytes and generation-based upgrade sources

## Status

Status: Accepted
Accepted: 2026-10-06

Accepted by the explicit human decisions recorded on 2026-10-06:

- in the Research
  [Accepted human decisions](../research/ci-cd-quality-strategy.md#accepted-human-decisions)
  of Issue [#154](https://github.com/rgomids/axiom/issues/154) (merged in PR #226);
- in the maintainer-authored contract of
  Issue [#227](https://github.com/rgomids/axiom/issues/227);
- in the Windows proxy scope and effective-date decision in
  [#227 comment 6009062641](https://github.com/rgomids/axiom/issues/227#issuecomment-6009062641).

The normative release contract becomes effective when the #227
Specification/ADR pull request is accepted and merged to `main`. PR review and
merge remain separate gates. This ADR authorizes no workflow, installer,
product code, Windows Server support, production eligibility bypass or release
publication.

The concrete placement of candidate acceptance (Alternatives D and E) is
**not** decided here. It is chosen by the implementation Slice within the
invariants below.

The observable contract is
[Specification 004 — Release-candidate acceptance and supported upgrade sources](../specifications/004-mvp-v1-baseline/spec.md#release-candidate-acceptance-and-supported-upgrade-sources)
(FR-068–FR-076, AC-50–AC-57).

## Context

Two regressions reached stable users and calibrate this decision:

- **#153** (persisted / historical upgrade failure). A host holding state
  from the `v0.1.0-poc.1` prerelease could not upgrade to v0.2.0: the
  installer refused it as `state_incompatible`. The refusal gate shipped in
  every stable release from v0.1.0 to v0.4.1, and v0.4.2 fixed it
  (ADR-0017). Two gaps let it ship:
  - *policy:* nobody had declared which persisted generations are supported
    upgrade sources. HD-4 and FR-065 disagreed, and tests asserted the
    refusal as intended behaviour;
  - *mechanism:* no candidate-level gate ran older-generation, non-empty
    state through the candidate.
- **#186** (skill / receipt convergence failure). Upgrading from v0.1.1 to
  v0.4.1 ended `partial`, because receipts written by earlier releases did not
  converge. It had no policy ambiguity; the missing candidate gate alone let
  it ship (ADR-0016).

Both failures came from **older persisted generations** (POC state,
five-skill receipts), not from the immediately previous release. An
`N-1 -> N` rule would have missed both.

The current state of `main` was verified for this decision:

- **Build once, publish the same bytes.** `release-artifacts.yml` builds the
  four release rows once from the release revision with the real version. It
  verifies them *structurally*: closed set, checksums, manifests, metadata,
  executable headers and build provenance, with a `version` probe on one row.
  It retains them as the prepared artifact. `publish-release.yml` downloads
  that prepared run, re-verifies it, recomputes the publication envelope, and
  publishes only if the envelope equals the human-authorized digest
  (ADR-0011, ADR-0012). Publication never rebuilds.
- **Behavioural validation runs on a different subject.** The
  `upgrade-journeys` checks in `ci.yml` exercise real published N, v0.1.1
  and POC state through a candidate installer. Since #188 they are
  merge-blocking on every pull request and `main` push. However, that
  candidate is rebuilt from source with the synthetic version
  `9999.0.0-acceptance.<run>`. Its bytes are never the prepared bytes, and the
  real version string is never exercised in an upgrade.

Pull-request regression coverage on a rebuilt candidate is valuable early
feedback, but it is not release Evidence. Release Evidence must be about the
exact bytes users will receive. Artifact identity at publication proves only
that the published bytes are the prepared bytes. It does not prove that those
bytes install, upgrade or run correctly. A rebuild cannot stand in for them:
the version is embedded, and archives carry metadata, so even a reproducible
binary yields different release files.

Platform reality constrains where Evidence can come from:

- macOS arm64 and Linux amd64 run natively on hosted runners.
- Linux arm64 has hosted arm64 runners but runs nowhere in CI today.
- No hosted Windows client amd64 runner exists, and production installers
  refuse Windows Server (Specification 006 FR-E03, ADR-0015).

## Decision

1. **Candidate acceptance consumes the prepared immutable bytes.** Its subject
   is the prepared candidate set produced once from the release revision. It
   never uses a rebuilt or source-equivalent candidate.
2. **The validated bytes are the bytes eligible for publication.** Any
   difference in revision, preparation or bytes is a new candidate that needs
   its own acceptance.
3. **Publication does not rebuild.** It publishes exactly the accepted bytes.
   Acceptance never modifies them; it only produces Evidence.
4. **Candidate acceptance belongs to the release boundary.**
   - A stable release cannot be published until acceptance has passed for
     its exact prepared set on every release row.
   - Before any effect, publication verifies passing release-critical Evidence
     bound to the digests of the files being published.
   - The human publication authority is given over a candidate whose
     acceptance outcome is bound to it.
   - The existing authority model is unchanged: preparation needs no
     publication authority, and acceptance grants none (ADR-0011, ADR-0012).
5. **Pull-request regression coverage stays** as early, merge-blocking
   feedback on rebuilt candidates.
6. **Pull-request regression does not replace candidate Evidence.** Neither
   does a published release candidate, whose bytes differ from the stable
   set.
7. **Supported upgrade sources follow a persisted-generation policy.** A
   persisted generation is a distinct shape of Axiom-owned persisted content
   that a published release writes and a later upgrade must converge: state
   and portable formats, installation receipts, and Runtime skill-set
   receipts. Declared historical formats (today RecognizedPOC) are sources
   with their declared transition.
8. **N is a mandatory baseline.** N is the latest published stable release the
   candidate supersedes. Its generation stays inside the candidate's
   compatibility window.
9. **Each supported persisted generation keeps one declared baseline.** The
   baseline is an immutable published release plus representative non-empty
   state it produced, or a frozen corpus of that state. A change introducing
   a new generation declares it, and its forward paths, before the candidate
   that first ships it passes acceptance. Every historical stable release is
   **not** a blocking source. A scheduled historical sweep may cover all of
   them, but it is non-blocking.
10. **Baselines are retired only by an explicit compatibility-window
    decision**, recorded with the outcome for hosts that still hold that
    generation. Silent pruning is never a retirement.
11. **Windows: hosted Windows Server is the bounded release-blocking proxy**
    for the Windows client amd64 row.
    - On the exact prepared Windows bytes it covers:
      - identity and digest binding;
      - version and revision provenance;
      - direct prepared `axiom.exe` CLI behaviour that does not treat Server
        as a supported client host;
      - the real prepared installers' fail-closed Server refusal, with zero
        effects.
    - It claims no Windows client install, upgrade or reinstall.
    - Specification 006 FR-E03/AC-E04 and ADR-0015 are unchanged. Server
      stays unsupported and refused, and no acceptance-only production
      override exists.
    - Windows install, upgrade and reinstall mechanics stay merge-blocking
      through the source-level test seam.
    - Native Windows client Evidence is supplemental.
12. **Real Runtime/Provider acceptance is not universally release-blocking.**
    It blocks only a release that changes that Integration or Runtime
    contract. Otherwise deterministic contract validation is the ordinary
    protection and real Evidence is supplemental. Supplemental Evidence never
    becomes blocking implicitly.
13. **Release-critical Evidence and its disclosure.**
    - Release-critical Evidence is bounded, structured and bound to the
      artifact digests, tag/version, revision and row.
    - Raw logs are never its only form.
    - Public vulnerability-scan output is `PASS`/`FAIL` only. Details follow
      the private `SECURITY.md` flow.

### Invariants

```text
validated candidate bytes == publishable candidate bytes
```

```text
a supported persisted generation cannot silently disappear from the upgrade contract
```

- Publication never rebuilds, and acceptance never mutates the prepared set.
- No stable publication happens without passing acceptance Evidence bound to
  the exact digests being published.
- Supplemental Evidence (native Windows client, non-blocking real
  Runtime/Provider runs, other distributions or OS versions, the historical
  sweep) is never silently promoted to blocking or counted as blocking
  Evidence.
- The Windows proxy never implies Windows Server support or client-installer
  Evidence.

### Transition

Once this ADR and its Specification amendment are merged, a manual candidate
acceptance satisfies the contract until automated acceptance exists. It runs
over the downloaded prepared set with Evidence bound to its exact digests. For
Windows it follows the bounded proxy scope.

## Alternatives considered

| | Alternative | Benefit | Cost / complexity | Authority / Evidence impact | Outcome |
|---|---|---|---|---|---|
| A | **Status quo:** regression on a rebuilt candidate only | Already exists; fast; merge-blocking | None added | Evidence is about different bytes and a synthetic version; release-boundary gap stays open (the #153/#186 class) | **Rejected** as the release guarantee; kept as PR feedback (Decision 5) |
| B | **`N-1 -> N` only** | Small, simple matrix | Lowest | Would have missed both #153 and #186, which came from older generations | **Rejected** |
| C | **Every historical stable release as a blocking matrix** | Maximal coverage | Grows linearly per release; slower and more fragile prepare; mostly redundant sources within one generation | Blocking on redundant sources raises the external-dependency failure surface without new failure classes | **Rejected** as blocking; kept as a non-blocking scheduled sweep |
| D | **Acceptance inside the prepare boundary** (same run as preparation) | Reuses the existing "prepared run succeeded" gate; no new cross-run trust; envelope model unchanged | Longer prepare run (minutes, estimate) | Acceptance outcome is inherent to the prepared run that publication already binds | **Deferred** to the implementation Slice; the Research recommends it |
| E | **Separate candidate-acceptance workflow** bound to the prepared run | Independent retry and isolation; prepare stays short | New cross-run binding: the envelope and publication must also verify the acceptance run and its Evidence digest | More moving parts in the authority chain; still compatible with the invariants | **Deferred**; the fallback if D's duration or permissions become a problem |
| F | **Published RC as the acceptance subject** (accept RC, then publish stable) | Exercises the real public install path | Days of latency per release; manual cycle | **The stable bytes differ from the RC bytes** (embedded version, different release commit), so it cannot satisfy invariant 1 | **Rejected** as the gate; an RC stays optional additional Evidence and remains the MVP acceptance subject (AC-48) |
| G | **Self-hosted native lab** (Windows client, other distros, OS versions) | Highest platform fidelity | Hardware, upkeep, runner security and trust | Adds an operator-owned trust domain; no demonstrated defect that hosted rows or the proxy miss | **Deferred**; revisit on concrete client-only or install-path divergence |

A single combined prepare -> accept -> publish run, gated only by environment
approval, was also considered. It would move publication authority from the
reviewed envelope digest to an environment click, weakening ADR-0011 and
ADR-0012. It is rejected.

## Consequences

### Positive

- **Release confidence:** behavioural Evidence is about the exact bytes users
  receive, under the real version string. The #153/#186 classes are blocked at
  the release boundary, not only at merge.
- **Compatibility commitment** is explicit and reviewable:
  - N and one baseline per generation;
  - a declared transition for each historical format;
  - generations cannot disappear silently.
- **Evidence** becomes bounded and machine-checkable, and is bound to digests.
  Authorization is given over a candidate whose acceptance outcome is known.

### Negative / trade-offs

- **Prepare latency:** each stable release waits for acceptance on every row.
  The Research estimates a few minutes per row in parallel.
- **PR latency:** unchanged by this decision. Pull-request regression already
  exists, and new rows belong to their own Slices.
- **Maintenance cost:**
  - a shared journey harness for rebuilt and prepared subjects;
  - a versioned generation declaration;
  - an Evidence schema;
  - keeping historical baselines (POC build, old published assets) runnable
    as toolchains evolve.
- **Artifact and Evidence retention:** acceptance Evidence must be retained at
  least until the publication it supports completes (FR-075). Today the
  prepared set's 30-day retention bounds that authorization window.
- **Failure and retry:** any change to the prepared set invalidates its
  acceptance. A failed acceptance blocks publication. A retry is a new attempt
  recorded in Evidence and never erases the failure. Infrastructure or
  external-dependency failures (for example, unavailable published baseline
  assets) can delay a release without a product defect, so failures need
  categories (a later Slice).
- **Platform residual risk:**
  - Windows client install, upgrade and reinstall of the exact bytes have no
    blocking Evidence. They rely on the source-level seam, on shared Go
    upgrade logic exercised on the POSIX rows, and on supplemental native
    client Evidence.
  - Hosted-image drift and unmodelled distributions remain residual risks.
- **Future persisted generations:** every new persisted shape costs a declared
  generation, its forward paths and a baseline. Removing one costs an explicit
  compatibility-window decision. The added friction is intended.
- **Interim process:** until automation exists, each stable release needs a
  manual acceptance over the prepared set. This adds maintainer effort per
  release.

## Revisit when

- a concrete client-only or Windows install-path divergence is found
  (Decision 11);
- a hosted Windows client amd64 runner, or an accepted native lab, becomes
  available;
- the scheduled sweep finds a generation the model missed, repeatedly;
- prepare-run duration or permissions make Alternative E preferable to D;
- the repository becomes private and runner cost changes the trade-offs;
- the release authority model (ADR-0011, ADR-0012) changes.

## Related

- [Specification 004 — Release-candidate acceptance and supported upgrade sources](../specifications/004-mvp-v1-baseline/spec.md#release-candidate-acceptance-and-supported-upgrade-sources)
- [Specification 006](../specifications/006-installer-host-eligibility/spec.md),
  [ADR-0015](0015-installer-host-eligibility-os-family-architecture.md):
  host eligibility, unchanged.
- [ADR-0011](0011-command-driven-release-start.md),
  [ADR-0012](0012-release-publication-credential.md): release start and the
  publication authority, unchanged.
- [ADR-0016](0016-candidate-derived-skill-set-receipt.md) (#186) and
  [ADR-0017](0017-recognized-poc-preservation-archive.md) (#153): the
  calibration regressions.
- [CI/CD quality strategy Research](../research/ci-cd-quality-strategy.md)
  (#154, PR #226).
