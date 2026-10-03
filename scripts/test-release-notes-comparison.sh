#!/usr/bin/env bash
# Exercise the real recovery notes comparisons without GitHub or publication.
set -euo pipefail
repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT

# Markdown links and glob metacharacters must be literal, never Bash patterns.
printf '## Release\n\n* Fix [#158](https://example.invalid/158): [abc], *, ? and \\ backslash.\n' >"$temporary/notes"
cp "$temporary/notes" "$temporary/published-notes"
cp "$temporary/notes" "$temporary/recovery-notes"
notes=$temporary/notes
fail() { printf '%s\n' "$1" >&2; return 1; }
comparisons=$(sed -n '/published recovery notes differ from prepared notes/p; /published recovery notes differ from pinned inputs/p' "$repository_root/scripts/publish-release.sh")
[[ $(wc -l <<<"$comparisons" | tr -d ' ') == 2 ]]
# Lines come from the trusted repository script, never provider content.
eval "$comparisons"
printf 'ok: identical Markdown notes compare literally in both recovery paths\n'

printf 'changed\n' >>"$temporary/published-notes"
while IFS= read -r comparison; do
  if (eval "$comparison") >"$temporary/out" 2>"$temporary/err"; then
    printf 'FAIL: changed published notes accepted\n' >&2
    exit 1
  fi
  grep -Fq 'published recovery notes differ' "$temporary/err"
done <<<"$comparisons"
printf 'ok: changed notes refuse in both recovery paths\n'

# A different string matching the old unquoted RHS must also refuse.
printf '[abc]\n' >"$notes"
cp "$notes" "$temporary/recovery-notes"
printf 'a\n' >"$temporary/published-notes"
while IFS= read -r comparison; do
  if (eval "$comparison") >"$temporary/out" 2>"$temporary/err"; then
    printf 'FAIL: notes matching a glob pattern accepted\n' >&2
    exit 1
  fi
  grep -Fq 'published recovery notes differ' "$temporary/err"
done <<<"$comparisons"
printf 'ok: different notes matching the old glob pattern refuse\n'
