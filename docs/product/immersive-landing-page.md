# Immersive landing integration

Issue [#316](https://github.com/rgomids/axiom/issues/316) evolves the bootstrap
surface accepted by #86. The maintainer requested the reviewed immersive
prototype as the replacement, with reusable elements maintained in Axiom DS
and separate pull requests for team approval. This document records the new
candidate; it does not rewrite the historical bootstrap Evidence or claim
review, merge, deployment or full MVP acceptance.

## Contract

- Axiom DS owns landing presentation, sections, tokens and browser behavior.
  Axiom owns the product integration and accuracy of its installation/capability
  claims. Source changes happen upstream; a pinned bundle is imported for review.
- `site/` is a checked-in static snapshot of `packages/landing/dist` from the
  full DS Git revision in `site/.axiom-landing-source.json`. Every file has a
  SHA-256 digest. There is no runtime CDN, frontend framework, paid service or
  build-time dependency on another checkout to deploy an accepted snapshot.
- The original transparent mascot is shipped unchanged; SVG crops remove the
  wordmark and tagline while preserving the gold detail. Packaging local assets
  is intentional so a reviewed page does not depend on moving raw `main` URLs.
  The structural validator compares the logo bytes to the canonical product
  `docs/assets/axiom-logo-white.png`.
- The narrative is one continuous five-scene transformation: an idea is born,
  its work fragments, Axiom connects the existing pieces, a six-stage workflow
  makes boundaries visible, and verifiable results lead to a human choice.
  The idea remains the protagonist; the large Axiom mark enters at the third
  scene. The six workflow stages belong to scene four, not six more scenes.
- All 25 storyboard beats are retained, five per scene. Fragmentation includes
  six work fronts and tool/context tiles; orchestration separately presents the
  map, five configurable stage cards, dependency graph, human gate and six-row
  execution view. Delivery presents convergence, five artifact rows, five review
  checks with two human alternatives, the delivered result and next possibilities.
  The same 25 illustrated beats remain available without motion or JavaScript.
- Native scroll controls the narrative reversibly. System reduced motion,
  manual pause and absent JavaScript reveal five complete static scenes; no
  essential explanation depends on animation. Institutional content, installation
  and documentation remain directly reachable without completing the story.
- The opening fades the gold screen to black before a single supernova reveals
  the idea. The nucleus fades out before the control plane enters, stays hidden
  through the central Axiom reveal and becomes the luminous origin as the
  configurable flow forms. The original idea returns connected to the next
  possibilities after the verifiable delivery. Its identity and explanation persist;
  no repeated flash or timed scene advancement is added.
- Pacing uses 200svh per beat, a reading hold, quintic geometry interpolation and
  a short scroll follower. Connection signals precede staggered card blooms.
  Ambient globe rotation, light pulses, wave lines and decorative square zooms
  add movement without advancing scenes. Pause/reduced motion disable ambient
  loops; fixed-frame previews remain deterministic.
- Sample model assignments, PR #123, 12/12 tests, progress 3/6 and approval labels
  follow the storyboard. Visible conceptual/example captions distinguish these
  illustrations from actual product capabilities, real CI results or acceptance.
- Illustrations are conceptual explanations, never product GUI screenshots or
  live execution results. Diff, tests and Evidence are examples; scrolling
  never authorizes or accepts a real delivery. Accept and rework remain human
  alternatives.
- PT/EN translations remain text. Static formatting belongs to HTML elements,
  never parsed translation strings. Blocked browser storage is supported.
- Each terminal command has its own copy control and selection fallback.
  Installation names only the currently supported hosts: macOS arm64 and Linux
  amd64/arm64. Commands follow the current README and command reference; provider
  configuration/authentication and human confirmation remain separate steps.
- Available foundation, MVP delivery and discovery remain distinct. This
  marketing change does not authorize CLI/domain implementation or accept the
  complete configurable delivery journey.

## Editorial revision

The [editorial review of PR #317](https://github.com/rgomids/axiom/pull/317#pullrequestreview-5512493953)
requested fragmentation before the solution, a visible transformation reusing
existing work, and a return to the original idea at the verifiable outcome.
The [motion handoff](https://github.com/rgomids/axiom/pull/317#issuecomment-6160104548)
and five storyboard references guide the new prototype. Their timings and
composition are design proposals. The maintainer's subsequent request authorizes
implementation of the prototype; it does not convert illustrative concepts
into delivered product functionality or imply editorial/visual acceptance.

The design source remains DS PR #19. The product imports its versioned bundle;
reviewers separately assess narrative comprehension and visual quality. Automated
checks verify reproducibility, continuity, accessibility fallbacks and product
regressions, not a visitor's understanding or human approval.

## Upgrade and validation

Follow [Website development](../development/getting-started.md#website-development).
The importer checks the DS checkout revision, refuses dirty canonical sources,
symlinks and unsafe paths, and refuses to overwrite files it does not own.
Only files recorded by the prior manifest (or the enumerated legacy bootstrap
files on first import) may be replaced or retired. Review the complete diff.

The retained shell validator delegates to the cross-platform Python structural
checker. It validates source revision, exact file digests, resource/anchor
resolution, canonical/social metadata, the original brand bytes, safe text
boundaries and product installer parity. Offline Chromium tests cover language
round trips and DOM XSS, blocked storage, keyboard tab navigation, narrow width,
reduced motion, complete no-JavaScript scenes, reversible native scroll and copying
one command at a time. Import tests verify
reproducibility, ownership protection and path traversal refusal.

## Approval, rollout and rollback

The DS pull request must be approved first so the imported revision has a
reviewable upstream source. The product pull request is a separate review gate.
GitHub Pages remains restricted to `main`, after validation; a branch push or
pull request cannot deploy. No merge or publication is performed by preparing
these changes. After merge, verify the actual Pages run and served URL before
claiming deployment. Reverting the integration commit restores the preceding
page and validator as one coherent unit.

## Limits

Chromium checks do not establish Safari/Firefox or real-device acceptance.
Social preview caches and the served Pages deployment are external checks after
merge. No Go product logic changes, and page validation does not replace Go CI.
