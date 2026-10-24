#!/usr/bin/env bash
# Sourced library: names the archive generation a release inventory declares.
# format 1 names macos-27-arm64 (historical); format 2 names macos-arm64. An
# unreadable inventory, one naming both macOS rows, or one naming neither fails
# closed: the format is never guessed, and the strict verifier still enforces
# the selected generation's closed inventory. Defines release_archive_format.

# release_archive_format SHA256SUMS: prints 1 or 2; returns 1 with a message
# on stderr when the generation cannot be named unambiguously.
release_archive_format() {
  local sums=$1 v1 v2
  [[ -f "$sums" && ! -L "$sums" ]] || { printf 'release_archive_format_error: inventory unreadable\n' >&2; return 1; }
  v1=$(grep -Ec '^[0-9a-f]{64}  axiom-[^ ]+-macos-27-arm64\.tar\.gz$' "$sums" || true)
  v2=$(grep -Ec '^[0-9a-f]{64}  axiom-[^ ]+-macos-arm64\.tar\.gz$' "$sums" || true)
  if [[ "$v1" == 1 && "$v2" == 0 ]]; then printf '1\n'
  elif [[ "$v1" == 0 && "$v2" == 1 ]]; then printf '2\n'
  else
    printf 'release_archive_format_error: inventory does not name exactly one macOS row (v1=%s v2=%s)\n' "$v1" "$v2" >&2
    return 1
  fi
}
