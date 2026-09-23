# Landing page — Acceptance Evidence

## Authority and status — 2026-09-23

- Scope: [Issue #86](https://github.com/rgomids/axiom/issues/86) — bootstrap public
  landing page. Delivered in [PR #88](https://github.com/rgomids/axiom/pull/88).
- This is a parallel product surface. It does not touch domain contracts,
  Specifications, or MVP delivery slices. S1 remains delivered, S2 remains
  authorized with T04–T07 implemented and awaiting human review, and S3–S7 remain
  unauthorized.
- Baseline: `main` at `aaf54be`, merge of
  [PR #89](https://github.com/rgomids/axiom/pull/89).
- Delivery branch: `docs/86_added_landing_page`.
- **Landing page implementation: Completed. Evidence: Produced. Human acceptance:
  Pending. Production deployment: Not yet observable — see Limitations.**

Checks, review, commit, or merge do not provide human acceptance.

## What is published

`site/` holds the whole published surface: `index.html`, `styles.css`, `app.js`,
`rain.js`. The Pages workflow uploads that directory verbatim; it assembles
nothing and copies nothing into it.

## Canonical asset reuse

`docs/assets/axiom-logo.png` and `docs/assets/axiom-logo-github.png` remain the only
versioned copies of the identity assets. The landing page consumes them through the
absolute raw URLs of those canonical files:

```text
https://raw.githubusercontent.com/rgomids/axiom/main/docs/assets/axiom-logo.png
https://raw.githubusercontent.com/rgomids/axiom/main/docs/assets/axiom-logo-github.png
```

The same two URLs serve the header logo, the hero logo, the favicon, the Apple
Touch icon, the Open Graph image, and the Twitter Card image. `blob` URLs are not
used for images: `blob` returns GitHub's HTML page, not the asset. No relative
path (`assets/…`, `../docs/assets/…`) crosses the published boundary, and no copy
of either asset exists under `site/`.

## Reproducible checks

### Landing page validation

```bash
AXIOM_LANDING_PAGE_REMOTE=1 ./scripts/validate-landing-page.sh .
```

```text
PASS: site/ contains every file the Pages artifact publishes
PASS: canonical identity assets exist under docs/assets/
PASS: no canonical asset is duplicated inside site/
PASS: no residual landing_page/ reference in the repository
PASS: site/ loads no image through a GitHub blob URL
PASS: site/ references no identity asset through a relative path
PASS: favicon, Apple Touch icon, Open Graph, Twitter, header and hero use canonical raw URLs
PASS: declared image dimensions match the canonical assets in docs/assets/
PASS: canonical URL, Open Graph and Twitter metadata are present
PASS: dark is the default, light requires an explicit stored choice, and the choice persists
PASS: prefers-reduced-motion is honoured by both the stylesheet and the animation
PASS: README.md documents the public URL and the local development command
PASS: every published file answers over http://localhost:33547
PASS: every canonical asset URL answers successfully
PASS: landing page validation passed
```

The local serving check binds a free ephemeral port so it stays deterministic when
port 8000 is already taken. The remote asset checks need network and are skipped
unless `AXIOM_LANDING_PAGE_REMOTE=1`.

### Repository and Go validation

```bash
./scripts/validate-repository.sh .
go test ./...
go vet ./...
go build ./...
go mod verify
```

```text
PASS: repository bootstrap validation passed
ok   github.com/rgomids/axiom/cmd/lingo           3.551s
ok   github.com/rgomids/axiom/internal/cli        0.013s
ok   github.com/rgomids/axiom/internal/codexruntime 0.127s
ok   github.com/rgomids/axiom/internal/completion 0.013s
ok   github.com/rgomids/axiom/internal/detailartifact 0.194s
ok   github.com/rgomids/axiom/internal/local      1.360s
ok   github.com/rgomids/axiom/internal/manifest   0.861s
ok   github.com/rgomids/axiom/internal/project    0.013s
ok   github.com/rgomids/axiom/internal/projectapp 0.010s
ok   github.com/rgomids/axiom/internal/provenance 0.006s
ok   github.com/rgomids/axiom/internal/workflow   0.017s
ok   github.com/rgomids/axiom/internal/workitem   0.032s
go vet: no findings
go build: no output
all modules verified
```

### Reference hygiene

```bash
grep -R "landing_page" . --exclude-dir=.git --exclude=validate-landing-page.sh
grep -RE 'github\.com/rgomids/axiom/blob/[^"]*\.(png|jpe?g|gif|svg|webp|ico)' site
grep -RE 'src="(\.\.?/|assets/)' site
grep -R "raw.githubusercontent.com/rgomids/axiom/main/docs/assets" site
```

The first three produce no output. The fourth lists the six canonical references
in `site/index.html`: `og:image`, `twitter:image`, `icon`, `apple-touch-icon`, the
header `<img>`, and the hero `<img>`.

The only remaining `blob` URL in `site/` is the footer link to `LICENSE`, which is
a document a reader opens, not an asset the browser loads.

### Canonical asset URLs

```bash
curl -fI https://raw.githubusercontent.com/rgomids/axiom/main/docs/assets/axiom-logo.png
curl -fI https://raw.githubusercontent.com/rgomids/axiom/main/docs/assets/axiom-logo-github.png
```

```text
200 image/png  .../docs/assets/axiom-logo.png
200 image/png  .../docs/assets/axiom-logo-github.png
```

### Local serving

```bash
python3 -m http.server 8000 --directory site
```

`/`, `/index.html`, `/styles.css`, `/app.js` and `/rain.js` all answer `200`. No
request escapes `site/`, because the only non-local requests are the two absolute
raw asset URLs above.

### Rendering, assets, theme and motion

All browser observations below were produced against that local server with
headless Chrome, and each command is reproducible.

Loaded assets, read from the live DOM:

```text
OK  1983x793   https://raw.githubusercontent.com/rgomids/axiom/main/docs/assets/axiom-logo-github.png
OK  1254x1254  https://raw.githubusercontent.com/rgomids/axiom/main/docs/assets/axiom-logo.png
DECLARED  og:image          .../axiom-logo-github.png
DECLARED  twitter:image     .../axiom-logo-github.png
DECLARED  icon              .../axiom-logo.png
DECLARED  apple-touch-icon  .../axiom-logo.png
stylesheets loaded: 1
body computed background: rgb(5, 8, 6)
```

Every `<img>` reports `complete` with a non-zero `naturalWidth`, so no asset is
broken. The declared `width`/`height` attributes were corrected to the canonical
PNG dimensions (1983×793 and 1254×1254); `validate-landing-page.sh` now compares
them against the PNG headers so the reserved box cannot drift from the asset.

Dark mode as the default, independent of the OS preference:

```bash
google-chrome --headless --dump-dom --blink-settings=preferredColorScheme=1 http://127.0.0.1:8000/
google-chrome --headless --dump-dom --blink-settings=preferredColorScheme=2 http://127.0.0.1:8000/
```

Both report `<html lang="en" data-theme="dark">` on a first visit with empty
storage. The OS colour scheme is deliberately not consulted.

Toggle and persistence, driven from a same-origin probe page:

```text
1. first visit (storage empty): data-theme=dark stored=null
2. after one toggle click:      data-theme=light stored="light" aria-label="Switch to dark theme"
3. after reload with stored light: data-theme=light stored="light"
4. toggled back:                data-theme=dark stored="dark"
5. after reload with stored dark:  data-theme=dark
```

`prefers-reduced-motion`:

```text
normal   reduce matches=false  rain layer present=true   drops after 3s=7  .btn transition=0.15s
reduced  reduce matches=true   rain layer present=false  drops after 3s=0  .btn transition=1e-05s
```

The background animation never starts under reduced motion, and the stylesheet
collapses transitions.

Viewports:

```bash
google-chrome --headless --window-size=1440,900 --screenshot=desktop.png http://127.0.0.1:8000/
google-chrome --headless --window-size=390,844  --screenshot=mobile.png  http://127.0.0.1:8000/
```

Both render the dark hero with both logos, readable copy, and both CTAs. At
390 px the header collapses to a single row, the hero stacks, and the CTA buttons
become full width; no horizontal overflow appears.

### Links and metadata

Every outbound `href` in `site/index.html` was requested:

```text
200  https://github.com/rgomids/axiom
200  https://github.com/rgomids/axiom#documentation
200  https://github.com/rgomids/axiom#project-status
200  https://github.com/rgomids/axiom/blob/main/LICENSE
200  https://raw.githubusercontent.com/.../axiom-logo.png
404  https://rgomids.github.io/axiom/           (canonical; see Limitations)
```

`## Documentation` and `## Project status` both exist in `README.md`, so both
anchors resolve.

### Supply chain

`.github/workflows/deploy-landpage.yml` carries `pages: write` and
`id-token: write`, so every Action is pinned to an immutable commit SHA, matching
the control already used by `.github/workflows/poc-verification.yml`:

```text
actions/checkout@11d5960a326750d5838078e36cf38b85af677262             # v4
actions/configure-pages@983d7736d9b0ae728b81ab479565c72886d7745b      # v5
actions/upload-pages-artifact@56afc609e74202658d3ffba0e8f6dda462b719fa # v3
actions/deploy-pages@d6db90164ac5ed86f2b6aed7e0febac5b3c0c03e         # v4
```

The `checkout` SHA is the same one `poc-verification.yml` already reviewed. Each
SHA was resolved from the official tag of the intended version and is annotated
with that version for readability.

The workflow uploads `./site` unchanged. It copies no logo into `site/`.

## Expected deployment

- Artifact: the contents of `site/`, uploaded by
  `actions/upload-pages-artifact` and deployed by `actions/deploy-pages` in the
  `github-pages` environment, under the `pages` concurrency group with
  `cancel-in-progress: false`.
- Public URL: <https://rgomids.github.io/axiom/>, recorded as the page's
  `<link rel="canonical">` and documented in [README](../../README.md#website) and
  [README.pt-BR](../README.pt-BR.md#site).

## Limitations

- The deploy workflow only triggers on `push` to `main`, so **the production Pages
  deployment cannot be observed before this PR merges**. The public URL currently
  answers `404`. Everything that does not depend on that deployment was validated
  deterministically above; the Pages result, the served artifact, and the live
  public URL must be confirmed after merge.
- The Open Graph and Twitter images are validated as reachable canonical URLs.
  How a given social platform crops or caches them is outside this Evidence.
- Screenshots were produced with headless Chrome at two viewports. No
  cross-browser or real-device matrix was run; #86 does not require one for this
  bootstrap.
- No Lighthouse run is recorded. #86 asks for reasonable performance,
  accessibility and SEO, not a scored threshold.
- The absolute raw URLs pin the `main` branch. An asset renamed or moved under
  `docs/assets/` would break the page; `validate-landing-page.sh` fails closed if
  the canonical files or the references stop matching.
