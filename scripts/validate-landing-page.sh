#!/usr/bin/env bash
# Reproducible acceptance checks for the public landing page (issue #86).
# Deterministic checks run offline; the remote asset checks require network and
# are skipped unless AXIOM_LANDING_PAGE_REMOTE=1.
set -euo pipefail

TARGET="${1:-.}"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

pass() {
  printf 'PASS: %s\n' "$1"
}

skip() {
  printf 'SKIP: %s\n' "$1"
}

[[ -d "$TARGET" ]] || fail "target directory does not exist: $TARGET"
ROOT="$(cd "$TARGET" && pwd -P)"
SITE="$ROOT/site"
RAW_BASE="https://raw.githubusercontent.com/rgomids/axiom/main/docs/assets"
LOGO="$RAW_BASE/axiom-logo.png"
LOGO_GITHUB="$RAW_BASE/axiom-logo-github.png"
PUBLIC_URL="https://rgomids.github.io/axiom/"

# --- Published tree -----------------------------------------------------------

site_files=(
  "index.html"
  "styles.css"
  "app.js"
  "rain.js"
)

for relative in "${site_files[@]}"; do
  [[ -s "$SITE/$relative" ]] || fail "landing page file is missing or empty: site/$relative"
done
pass "site/ contains every file the Pages artifact publishes"

# --- Canonical assets ---------------------------------------------------------

canonical_assets=(
  "docs/assets/axiom-logo.png"
  "docs/assets/axiom-logo-github.png"
)

for relative in "${canonical_assets[@]}"; do
  [[ -s "$ROOT/$relative" ]] || fail "canonical asset is missing: $relative"
done
pass "canonical identity assets exist under docs/assets/"

if find "$SITE" -type f \( -name 'axiom-logo.png' -o -name 'axiom-logo-github.png' \) -print -quit | grep -q .; then
  fail "canonical asset is duplicated inside site/"
fi
if [[ -d "$SITE/assets" ]]; then
  fail "site/assets/ exists; identity assets must stay only in docs/assets/"
fi
pass "no canonical asset is duplicated inside site/"

# --- Asset reference hygiene --------------------------------------------------

# This checker and its Evidence are excluded: both necessarily spell the legacy
# name in order to describe the check itself.
if grep -rq "landing_page" "$ROOT" --exclude-dir=.git \
  --exclude=validate-landing-page.sh --exclude=evidence-landing-page.md; then
  fail "residual reference to the removed landing_page/ directory"
fi
pass "no residual landing_page/ reference in the repository"

# A blob URL renders GitHub's HTML page, so it may link a document for a reader
# but must never be used for an asset the browser loads.
if grep -rEq 'github\.com/rgomids/axiom/blob/[^"]*\.(png|jpe?g|gif|svg|webp|ico)' "$SITE"; then
  fail "site/ loads an image through a GitHub blob URL; blob serves HTML, not the raw asset"
fi
pass "site/ loads no image through a GitHub blob URL"

if grep -rEq 'src="(\.\.?/|assets/)' "$SITE"; then
  fail "site/ references an identity asset through a relative path"
fi
pass "site/ references no identity asset through a relative path"

expected_refs=(
  "<meta property=\"og:image\" content=\"$LOGO_GITHUB\">"
  "<meta name=\"twitter:image\" content=\"$LOGO_GITHUB\">"
  "<link rel=\"icon\" href=\"$LOGO\" type=\"image/png\">"
  "<link rel=\"apple-touch-icon\" href=\"$LOGO\">"
  "src=\"$LOGO_GITHUB\""
  "src=\"$LOGO\""
)

for reference in "${expected_refs[@]}"; do
  grep -Fq -- "$reference" "$SITE/index.html" \
    || fail "canonical asset reference is missing from site/index.html: $reference"
done
pass "favicon, Apple Touch icon, Open Graph, Twitter, header and hero use canonical raw URLs"

# Declared width/height must match the canonical PNGs, or the reserved box has
# the wrong aspect ratio and the hero shifts once the remote image arrives.
python3 - "$ROOT" <<'PYEOF' || fail "declared image dimensions do not match the canonical assets"
import re
import struct
import sys

root = sys.argv[1]
html = open(root + "/site/index.html", encoding="utf-8").read()

for name in ("axiom-logo.png", "axiom-logo-github.png"):
    with open(root + "/docs/assets/" + name, "rb") as handle:
        width, height = struct.unpack(">II", handle.read(24)[16:24])
    pattern = r'src="[^"]*%s"[^>]*width="(\d+)" height="(\d+)"' % re.escape(name)
    match = re.search(pattern, html)
    if match is None:
        raise SystemExit("no <img> declares dimensions for " + name)
    if (int(match.group(1)), int(match.group(2))) != (width, height):
        raise SystemExit(
            "%s declares %sx%s but the canonical asset is %dx%d"
            % (name, match.group(1), match.group(2), width, height)
        )
PYEOF
pass "declared image dimensions match the canonical assets in docs/assets/"

# --- Metadata -----------------------------------------------------------------

grep -Fq -- "<link rel=\"canonical\" href=\"$PUBLIC_URL\">" "$SITE/index.html" \
  || fail "canonical link does not point to $PUBLIC_URL"
grep -Fq -- '<meta name="twitter:card" content="summary_large_image">' "$SITE/index.html" \
  || fail "Twitter card metadata is missing"
grep -Fq -- '<meta property="og:type" content="website">' "$SITE/index.html" \
  || fail "Open Graph metadata is missing"
pass "canonical URL, Open Graph and Twitter metadata are present"

# --- Theme --------------------------------------------------------------------

grep -Fq -- "stored === 'light' ? 'light' : 'dark'" "$SITE/index.html" \
  || fail "first-paint bootstrap does not default to dark"
grep -Fq -- 'data-theme="dark"' "$SITE/index.html" \
  || fail "<html> does not carry the dark default"
if grep -q "prefers-color-scheme" "$SITE/index.html" "$SITE/app.js"; then
  fail "the initial theme still follows the OS preference; #86 requires dark by default"
fi
grep -Fq -- "localStorage.setItem(STORAGE_KEY, theme)" "$SITE/app.js" \
  || fail "the explicit theme choice is not persisted"
pass "dark is the default, light requires an explicit stored choice, and the choice persists"

grep -Fq -- '@media (prefers-reduced-motion: reduce)' "$SITE/styles.css" \
  || fail "styles.css has no prefers-reduced-motion treatment"
grep -Fq -- "prefers-reduced-motion: reduce" "$SITE/rain.js" \
  || fail "the background animation does not honour prefers-reduced-motion"
pass "prefers-reduced-motion is honoured by both the stylesheet and the animation"

# --- Documented URLs ----------------------------------------------------------

grep -Fq -- "$PUBLIC_URL" "$ROOT/README.md" \
  || fail "the public Pages URL is not documented in README.md"
grep -Fq -- "python3 -m http.server 8000 --directory site" "$ROOT/README.md" \
  || fail "local development instructions are not documented in README.md"
pass "README.md documents the public URL and the local development command"

# --- Local serving ------------------------------------------------------------

if ! command -v python3 >/dev/null 2>&1; then
  skip "python3 is unavailable; local serving check skipped"
else
  # A free ephemeral port keeps the check deterministic even when the port the
  # README suggests is already taken by another local service.
  PORT="${AXIOM_LANDING_PAGE_PORT:-$(python3 -c '
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
')}"
  python3 -m http.server "$PORT" --bind 127.0.0.1 --directory "$SITE" >/dev/null 2>&1 &
  server_pid=$!
  trap 'kill "$server_pid" >/dev/null 2>&1 || true' EXIT

  ready=0
  for _ in $(seq 1 40); do
    kill -0 "$server_pid" 2>/dev/null || fail "the local server exited; port $PORT is probably in use"
    if curl -fsS -o /dev/null "http://127.0.0.1:$PORT/index.html" 2>/dev/null; then
      ready=1
      break
    fi
    sleep 0.25
  done
  [[ "$ready" -eq 1 ]] || fail "the local server did not answer on port $PORT"

  for relative in "${site_files[@]}"; do
    curl -fsS -o /dev/null "http://127.0.0.1:$PORT/$relative" \
      || fail "local server returned a failure for /$relative"
  done
  pass "every published file answers over http://localhost:$PORT"

  kill "$server_pid" >/dev/null 2>&1 || true
  wait "$server_pid" 2>/dev/null || true
  trap - EXIT
fi

# --- Remote assets ------------------------------------------------------------

if [[ "${AXIOM_LANDING_PAGE_REMOTE:-0}" != "1" ]]; then
  skip "remote asset checks disabled; set AXIOM_LANDING_PAGE_REMOTE=1 to enable"
else
  for url in "$LOGO" "$LOGO_GITHUB"; do
    curl -fsSI -o /dev/null --max-time 20 "$url" \
      || fail "canonical asset URL did not answer successfully: $url"
  done
  pass "every canonical asset URL answers successfully"
fi

pass "landing page validation passed"
