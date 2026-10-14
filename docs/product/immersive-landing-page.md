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
  claims. Source changes happen upstream, then a reviewed bundle is imported.
- `site/` is a checked-in static snapshot of `packages/landing/dist` from the
  full DS Git revision in `site/.axiom-landing-source.json`. Every file has a
  SHA-256 digest. There is no runtime CDN, frontend framework, paid service or
  build-time dependency on another checkout to deploy an accepted snapshot.
- The original transparent mascot is shipped unchanged; SVG crops remove the
  wordmark and tagline while preserving the gold detail. Packaging local assets
  is intentional so a reviewed page does not depend on moving raw `main` URLs.
  The structural validator compares the logo bytes to the canonical product
  `docs/assets/axiom-logo-white.png`.
- The gold opening transitions into the dark page through native scrolling.
  The organic control-plane diagram is explanatory, with a readable transcript.
  System reduced motion and the manual motion control reveal complete content.
- PT/EN translations remain text. Static formatting belongs to HTML elements,
  never parsed translation strings. Blocked browser storage is supported.
- Each terminal command has its own copy control and selection fallback.
  Installation names only the currently supported hosts: macOS arm64 and Linux
  amd64/arm64. Commands follow the current README and command reference; provider
  configuration/authentication and human confirmation remain separate steps.
- Available foundation, MVP delivery and discovery remain distinct. This
  marketing change does not authorize CLI/domain implementation or accept the
  complete configurable delivery journey.

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
reduced motion and copying one command at a time. Import tests verify
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
