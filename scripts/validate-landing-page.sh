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
LOGO_HERO="$RAW_BASE/axiom-logo-black.png"      # Open Graph and Twitter card
LOGO_BRAND="$RAW_BASE/axiom-logo-white.png"     # hero logo, transparent
LOGO_HEADER="$RAW_BASE/axiom-logo-github.png"   # header mark, rounded app tile
PHOTO_HOW="$RAW_BASE/generic-server.jpeg"       # How it works section background
LOGO_ICON="$RAW_BASE/axiom-logo-app-black.png"  # favicon and Apple Touch icon
PUBLIC_URL="https://rgomids.github.io/axiom/"

# --- Published tree -----------------------------------------------------------

site_files=(
  "index.html"
  "styles.css"
  "lang.js"
  "diagram.js"
  "skills.js"
)

# Retired files: app.js carried the theme toggle, rain.js the Matrix background,
# ocean.js the scroll-driven tentacles. None may come back unnoticed.
retired_files=(
  "app.js"
  "rain.js"
  "ocean.js"
)

for relative in "${site_files[@]}"; do
  [[ -s "$SITE/$relative" ]] || fail "landing page file is missing or empty: site/$relative"
done
pass "site/ contains every file the Pages artifact publishes"

for relative in "${retired_files[@]}"; do
  [[ ! -e "$SITE/$relative" ]] || fail "retired landing page file is back: site/$relative"
done
pass "no retired landing page file is published"

# --- Canonical assets ---------------------------------------------------------

canonical_assets=(
  "docs/assets/axiom-logo-black.png"
  "docs/assets/axiom-logo-white.png"
  "docs/assets/axiom-logo-app-black.png"
  "docs/assets/axiom-logo-github.png"
  "docs/assets/generic-server.jpeg"
)

for relative in "${canonical_assets[@]}"; do
  [[ -s "$ROOT/$relative" ]] || fail "canonical asset is missing: $relative"
done
pass "canonical identity assets exist under docs/assets/"

if find "$SITE" -type f \( -name 'axiom-logo*' -o -name 'generic-server*' \) -print -quit | grep -q .; then
  fail "canonical asset is duplicated inside site/"
fi
if [[ -d "$SITE/assets" ]]; then
  fail "site/assets/ exists; identity assets must stay only in docs/assets/"
fi
pass "no canonical asset is duplicated inside site/"

# --- Asset reference hygiene --------------------------------------------------

# This checker and its Evidence are excluded: both necessarily spell the legacy
# name in order to describe the check itself. evidence-s3.md is excluded for the
# same reason — it quotes a historical stash message verbatim, and rewriting an
# accepted record to satisfy a lint would falsify it.
# Only versioned content counts: an untracked scratch file in a working copy is
# not something the repository publishes.
if git -C "$ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  residual() {
    git -C "$ROOT" grep -q "landing_page" -- . \
      ':!scripts/validate-landing-page.sh' \
      ':!docs/product/evidence-landing-page.md' \
      ':!docs/specifications/004-mvp-v1-baseline/evidence-s3.md'
  }
else
  residual() {
    grep -rq "landing_page" "$ROOT" \
      --exclude=validate-landing-page.sh --exclude=evidence-landing-page.md \
      --exclude=evidence-s3.md
  }
fi
if residual; then
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
  "<meta property=\"og:image\" content=\"$LOGO_HERO\">"
  "<meta name=\"twitter:image\" content=\"$LOGO_HERO\">"
  "<link rel=\"icon\" href=\"$LOGO_ICON\" type=\"image/png\">"
  "<link rel=\"apple-touch-icon\" href=\"$LOGO_ICON\">"
  "src=\"$LOGO_BRAND\""
  "src=\"$LOGO_HEADER\""
  "src=\"$PHOTO_HOW\""
)

for reference in "${expected_refs[@]}"; do
  grep -Fq -- "$reference" "$SITE/index.html" \
    || fail "canonical asset reference is missing from site/index.html: $reference"
done
pass "favicon, Apple Touch icon, Open Graph, Twitter, header, hero and section photo use canonical raw URLs"

# Declared width/height must match the canonical PNGs, or the reserved box has
# the wrong aspect ratio and the hero shifts once the remote image arrives.
python3 - "$ROOT" <<'PYEOF' || fail "declared image dimensions do not match the canonical assets"
import re
import struct
import sys

root = sys.argv[1]
html = open(root + "/site/index.html", encoding="utf-8").read()

def png_size(data):
    return struct.unpack(">II", data[16:24])

def jpeg_size(data):
    i = 2
    while i < len(data):
        marker, length = data[i + 1], struct.unpack(">H", data[i + 2:i + 4])[0]
        if marker in (0xC0, 0xC1, 0xC2):
            height, width = struct.unpack(">HH", data[i + 5:i + 9])
            return width, height
        i += 2 + length
    raise SystemExit("no JPEG frame header")

for name in ("axiom-logo-white.png", "axiom-logo-github.png", "generic-server.jpeg"):
    with open(root + "/docs/assets/" + name, "rb") as handle:
        data = handle.read()
    width, height = jpeg_size(data) if data[:2] == b"\xff\xd8" else png_size(data)
    pattern = r'src="[^"]*%s"[^>]*width="(\d+)" height="(\d+)"' % re.escape(name)
    matches = re.findall(pattern, html)
    if not matches:
        raise SystemExit("no <img> declares dimensions for " + name)
    for declared in matches:
        if (int(declared[0]), int(declared[1])) != (width, height):
            raise SystemExit(
                "%s declares %sx%s but the canonical asset is %dx%d"
                % (name, declared[0], declared[1], width, height)
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

# --- Installer parity ---------------------------------------------------------

posix_install='curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh'
windows_install='&amp; ([scriptblock]::Create((Invoke-RestMethod https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.ps1)))'

grep -Fq -- "$posix_install" "$SITE/index.html" \
  || fail "landing page does not publish the POSIX installer command"
grep -Fq -- "$windows_install" "$SITE/index.html" \
  || fail "landing page does not publish the native PowerShell installer command"
grep -Fq -- 'Windows setup updates PATH for the current terminal and persistent user PATH' "$SITE/index.html" \
  || fail "landing page does not explain Windows user PATH setup"
grep -Fq -- 'Fresh Windows installations use private profile storage' "$SITE/index.html" \
  || fail "landing page does not explain private fresh Windows storage"
grep -Fq -- 'asks for approval before changing only the necessary directories' "$SITE/index.html" \
  || fail "landing page does not explain bounded permission-repair authority"
grep -Fq -- 'Windows client edition, amd64: 64-bit PowerShell 5.1+' "$SITE/index.html" \
  || fail "landing page does not state the supported Windows host row"
pass "landing page presents equivalent POSIX and native PowerShell installation paths"

# --- Theme: dark only ---------------------------------------------------------

# The landing page has exactly one palette. These are closed checks: the page
# must declare dark, and nothing may reintroduce a second theme by any route —
# an OS branch, a stored preference, a toggle control, or a theme attribute.
grep -Fq -- 'color-scheme: dark;' "$SITE/styles.css" \
  || fail "styles.css does not declare color-scheme: dark"

if grep -rq "prefers-color-scheme" "$SITE"; then
  fail "site/ still branches on the OS colour scheme; the landing page is dark only"
fi
if grep -rq "data-theme" "$SITE"; then
  fail "site/ still carries a data-theme hook; the landing page is dark only"
fi
if grep -rqi "theme-toggle\|axiom-theme" "$SITE"; then
  fail "site/ still ships a theme switch; the landing page is dark only"
fi
# Every localStorage access in site/ goes through one key, and that key is the
# language. So no palette preference can be stored, by construction.
if grep -rhoE 'localStorage\.(get|set)Item\([^,)]+' "$SITE" | grep -qv 'STORAGE_KEY'; then
  fail "site/ reads or writes browser storage outside the single declared key"
fi
grep -Fq -- "var STORAGE_KEY = 'axiom-lang';" "$SITE/lang.js" \
  || fail "the only stored preference is no longer the language"
pass "dark is the only theme: no OS branch, no stored preference, no toggle"

# --- Design system: palette contrast ------------------------------------------

# Text is only ever set on the page background or on one of its surfaces, so
# every text token must reach WCAG AA (4.5:1) against every one of them, and the
# primary-button ink must reach it against the accent it sits on.
python3 - "$SITE/styles.css" <<'PYEOF' || fail "the palette tokens do not reach WCAG AA contrast"
import re
import sys

css = open(sys.argv[1], encoding="utf-8").read()
root = re.search(r":root\s*\{(.*?)\n\}", css, re.S)
if root is None:
    raise SystemExit("styles.css has no :root token block")
tokens = dict(re.findall(r"--([\w-]+):\s*(#[0-9a-fA-F]{6})\s*;", root.group(1)))

def luminance(hex_value):
    channels = [int(hex_value[i:i + 2], 16) / 255 for i in (1, 3, 5)]
    linear = [c / 12.92 if c <= 0.03928 else ((c + 0.055) / 1.055) ** 2.4 for c in channels]
    return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2]

def contrast(a, b):
    high, low = sorted((luminance(a), luminance(b)), reverse=True)
    return (high + 0.05) / (low + 0.05)

pairs = [
    (fg, bg)
    for fg in ("text", "text-muted", "text-faint", "accent", "gold", "gold-strong")
    for bg in ("bg", "bg-raised", "surface", "surface-hover")
] + [("accent-ink", "accent"), ("accent-ink", "gold"), ("accent-ink", "gold-strong")]

for fg, bg in pairs:
    if fg not in tokens or bg not in tokens:
        raise SystemExit("palette token is missing or not a 6-digit hex: --%s / --%s" % (fg, bg))
    ratio = contrast(tokens[fg], tokens[bg])
    if ratio < 4.5:
        raise SystemExit("--%s on --%s is %.2f:1, below 4.5:1" % (fg, bg, ratio))
PYEOF
pass "every text token reaches WCAG AA contrast on every background and surface token"

# --- Motion -------------------------------------------------------------------

# Motion is confined to two places: the flow diagram in "How it works" (pulsing
# rings in CSS, signals moved by diagram.js) and the spinning globe on the
# language button. Everything else is static. Both stop under
# prefers-reduced-motion.
grep -Fq -- '@media (prefers-reduced-motion: reduce)' "$SITE/styles.css" \
  || fail "styles.css has no prefers-reduced-motion treatment"

python3 - "$SITE" <<'PYEOF' || fail "animation escapes the flow diagram or ignores reduced motion"
import re
import sys

site = sys.argv[1]
css = open(site + "/styles.css", encoding="utf-8").read()
css = re.sub(r"/\*.*?\*/", "", css, flags=re.S)

for name in re.findall(r"@keyframes\s+([\w-]+)", css):
    if not name.startswith(("flow-", "globe-")):
        raise SystemExit("@keyframes %s is outside the flow diagram and the globe" % name)

body = re.sub(r"@keyframes\s+[\w-]+\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}", "", css)
for selector, block in re.findall(r"([^{}]+)\{([^{}]*)\}", body):
    if re.search(r"(^|;|\s)animation(-name)?\s*:", block):
        value = re.search(r"animation(?:-name)?\s*:\s*([^;]+)", block).group(1).strip()
        if value != "none" and not all(part.strip().startswith((".flow", ".globe")) for part in selector.split(",")):
            raise SystemExit("animation declared outside the flow diagram and the globe: " + selector.strip())

reduced = re.search(r"@media \(prefers-reduced-motion: reduce\)\s*\{(.*?)\n\}", css, re.S)
if not reduced or not re.search(r"\.flow-ring\s*\{\s*animation:\s*none;", reduced.group(1)):
    raise SystemExit("the flow rings keep pulsing under prefers-reduced-motion")
if not re.search(r"\.globe-meridian\s*\{\s*animation:\s*none;", reduced.group(1)):
    raise SystemExit("the language globe keeps spinning under prefers-reduced-motion")
PYEOF

for file in "$SITE"/*.js; do
  [[ "$(basename "$file")" == "diagram.js" ]] && continue
  if grep -qE 'requestAnimationFrame|setInterval|\.animate\(' "$file"; then
    fail "$(basename "$file") drives motion; only diagram.js may"
  fi
done
if grep -qE 'setInterval|\.animate\(' "$SITE/diagram.js"; then
  fail "diagram.js uses setInterval or the Web Animations API"
fi
grep -Fq -- "matchMedia('(prefers-reduced-motion: reduce)')" "$SITE/diagram.js" \
  || fail "diagram.js does not honour prefers-reduced-motion"
pass "motion is confined to the flow diagram and the language globe, off under reduced motion"

# --- Retired background effects ----------------------------------------------

# The Matrix rain, the bubbles and the scroll-driven tentacles were all retired;
# the background is static light only.
if grep -rqi "matrix\|rain-drop\|rain-glyph\|bubble\|tentacle" "$SITE"; then
  fail "site/ still carries a retired background effect"
fi
pass "no retired background effect: the background is static light only"

# --- Language switch ----------------------------------------------------------

# The page is authored in English and carries the Portuguese version inline, so
# a translation can never go missing without this check noticing.
grep -Fq -- 'id="lang-toggle"' "$SITE/index.html" \
  || fail "the language switch is missing from index.html"

python3 - "$SITE" <<'PYEOF' || fail "the English and Portuguese versions are not in step"
import re
import sys

html = open(sys.argv[1] + "/index.html", encoding="utf-8").read()

plain = len(re.findall(r'\sdata-pt="', html))
if plain < 40:
    raise SystemExit("only %d translated strings found; expected the whole page" % plain)

# Translation data must stay text. Markup belongs to the static document.
if re.search(r'\sdata-(pt|en)-html=', html):
    raise SystemExit("HTML translation attributes reintroduce the unsafe translation boundary")
script = open(sys.argv[1] + "/lang.js", encoding="utf-8").read()
if re.search(r'innerHTML|outerHTML|insertAdjacentHTML|document\.write', script):
    raise SystemExit("language switching must not parse translation strings as HTML")

# data-en is filled in by lang.js at runtime. Authoring one by
# hand would silently win over the page's own English text.
if re.search(r'\sdata-en(-html)?="', html):
    raise SystemExit("index.html hardcodes data-en; the English text is the page itself")

# Every element that translates must actually have English text to go back to.
for match in re.finditer(r'<(\w+)[^>]*\sdata-pt="[^"]*"[^>]*>\s*(.{0,3})', html):
    if not match.group(2).strip():
        raise SystemExit("<%s> carries data-pt but has no English text" % match.group(1))
PYEOF
pass "every visible string has an English and a Portuguese version"

# --- Documented URLs ----------------------------------------------------------

grep -Fq -- "$PUBLIC_URL" "$ROOT/README.md" \
  || fail "the public Pages URL is not documented in README.md"
grep -Fq -- "python3 -m http.server 8000 --directory site" "$ROOT/docs/development/getting-started.md" \
  || fail "local development instructions are not documented in docs/development/getting-started.md"
pass "README.md documents the public URL; the development guide documents local serving"

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
  for url in "$LOGO_HERO" "$LOGO_BRAND" "$LOGO_HEADER" "$LOGO_ICON" "$PHOTO_HOW"; do
    curl -fsSI -o /dev/null --max-time 20 "$url" \
      || fail "canonical asset URL did not answer successfully: $url"
  done
  pass "every canonical asset URL answers successfully"
fi

pass "landing page validation passed"
