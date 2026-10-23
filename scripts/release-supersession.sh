#!/usr/bin/env bash
# Explicit supersession of an unpublished stable release (ADR-0035). The
# reviewed record .github/release-supersessions.txt at a revision lists, one
# row per version, a stable release whose release commit exists on main but
# that is never published; the next version ships its changes instead:
#
#   <MAJOR.MINOR.PATCH> <full release commit SHA> <reason>
#
# Blank lines and lines starting with '#' are ignored; any other line, or two
# rows for one version, fails closed. Sourced, not executed: the caller
# defines repository_root and fail().

# superseded_commit REV VERSION prints the release commit recorded as
# superseded for VERSION in the record at REV, or nothing.
superseded_commit() {
  local supersession_rev=$1 supersession_version=$2 supersession_record supersession_line found=
  supersession_record=$(git -C "$repository_root" show "$supersession_rev:.github/release-supersessions.txt" 2>/dev/null) || return 0
  while IFS= read -r supersession_line; do
    supersession_line=${supersession_line%$'\r'}
    [[ -z "$supersession_line" || "$supersession_line" == \#* ]] && continue
    [[ "$supersession_line" =~ ^((0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))\ ([0-9a-f]{40})\ ([[:print:]]+)$ ]] \
      || fail 'release supersession record is malformed'
    [[ "${BASH_REMATCH[1]}" == "$supersession_version" ]] || continue
    [[ -z "$found" ]] || fail "release supersession record has two rows for $supersession_version"
    found=${BASH_REMATCH[5]}
  done <<<"$supersession_record"
  printf '%s' "$found"
}

# last_release_sha REV prints the top-level "last-release-sha" of
# release-please-config.json at REV, or nothing.
last_release_sha() {
  local supersession_config
  supersession_config=$(git -C "$repository_root" show "$1:release-please-config.json" 2>/dev/null) || return 0
  jq -r '.["last-release-sha"] // empty' <<<"$supersession_config" || fail 'release-please-config.json is malformed'
}
