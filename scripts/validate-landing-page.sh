#!/usr/bin/env bash
# Compatibility entrypoint used by repository and GitHub Pages checks.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# Some developer hosts expose a nonfunctional python3 application alias.
if python3 -c 'import sys; assert sys.version_info >= (3, 9)' >/dev/null 2>&1; then
  exec python3 "$SCRIPT_DIR/validate-landing-page.py" "${1:-.}"
fi
exec python "$SCRIPT_DIR/validate-landing-page.py" "${1:-.}"
