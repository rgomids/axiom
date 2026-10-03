#!/usr/bin/env bash
# Delivery metadata: the explicit Pull Request -> Issue relationship and the
# Issue set a stable release delivers. Local and read-only (Git only, no
# network). Modes:
#
#   delivery-issues.sh parse [--file FILE]
#       Parse one squash commit message or PR body (stdin by default).
#       Prints metadata=declared|undeclared, related=, completes=.
#   delivery-issues.sh check-pr --body FILE --title FILE
#       A PR must have a Conventional Commit title, declare valid metadata,
#       and avoid GitHub closing keywords (they close Issues at merge time,
#       before release).
#   delivery-issues.sh commits --from SHA --to SHA
#       One line per first-parent commit in FROM..TO (merge-time sync).
#   delivery-issues.sh release --tag vX.Y.Z --revision SHA
#       The Issue set delivered by the stable release committed at SHA:
#       Completes-Issues of every first-parent commit since the previous
#       release commit, deduplicated. A release candidate delivers none.
#       After the legacy boundary (stable versions above v0.2.0) every
#       ordinary commit in the range must declare metadata or have a reviewed
#       correction; an undeclared one fails closed.
#
# Metadata grammar (one line per key, at column 0, outside fenced code):
#
#   Related-Issues: none | #N[, #N...]
#   Completes-Issues: none | #N[, #N...]
#
# Text inside HTML comments is ignored like fenced code. Key lines must fit
# in 72 columns: GitHub wraps the squash commit body near that width.
# Both keys exactly once; same-repository Issue numbers only; no duplicates;
# every completed Issue is also related. A message with neither key is
# undeclared (history before this contract, Release PRs). Anything else fails
# closed. A reviewed correction for an immutable commit can be recorded in
# .github/delivery-corrections.txt (read at the resolved revision) as
# "<full-sha> related=<list|none> completes=<list|none>".
#
# Legacy boundary: v0.2.0 is the last release whose range may contain
# undeclared commits (history merged before this contract). For every later
# stable release, only the Release Please release commit itself may be
# undeclared; old closing keywords are never interpreted either way.
set -euo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
max_issues=20
legacy_boundary=0.2.0

fail() {
  printf 'delivery_metadata_error: %s\n' "$1" >&2
  exit 1
}

# normalize_list VALUE KEY prints the sorted list ("132,140") or "none".
normalize_list() {
  local value=$1 key=$2 list_re item count sorted
  list_re='^#[1-9][0-9]{0,8}([ ]*,[ ]*#[1-9][0-9]{0,8})*$'
  if [[ "$value" == none ]]; then
    printf 'none'
    return
  fi
  [[ "$value" =~ $list_re ]] || fail "$key must be 'none' or a list of same-repository Issues like '#12, #34'"
  sorted=$(printf '%s\n' "$value" | tr ',' '\n' | tr -d ' #' | LC_ALL=C sort -n)
  count=$(printf '%s\n' "$sorted" | wc -l | tr -d ' ')
  ((count <= max_issues)) || fail "$key lists more than $max_issues Issues"
  [[ $(printf '%s\n' "$sorted" | uniq -d) == "" ]] || fail "$key lists an Issue more than once"
  item=$(printf '%s\n' "$sorted" | paste -sd, -)
  printf '%s' "$item"
}

# parse_message FILE prints metadata=, related=, completes= for one message.
parse_message() {
  local file=$1 line fence=false comment=false related= completes= related_seen=0 completes_seen=0
  local key_re near_re fence_re value item rest
  key_re='^(Related-Issues|Completes-Issues):[ ]*(.*)$'
  near_re='^[ >]*[*_`]*(related|complete[sd]?|closes?|fixe?s?|resolves?)[-_ ]*issues?[*_`]*[ ]*:'
  fence_re='^ {0,3}(```|~~~)'
  while IFS= read -r line || [[ -n "$line" ]]; do
    line=${line%$'\r'}
    if [[ "$line" =~ $fence_re ]]; then
      [[ "$fence" == true ]] && fence=false || fence=true
      continue
    fi
    [[ "$fence" == false ]] || continue
    # Text inside HTML comments is invisible in the rendered PR, so it can
    # never carry metadata a reviewer did not see.
    if [[ "$comment" == true ]]; then
      [[ "$line" == *'-->'* ]] && comment=false
      continue
    fi
    if [[ "$line" == *'<!--'* ]]; then
      rest=${line##*<!--}
      [[ "$rest" == *'-->'* ]] || comment=true
      [[ "$line" == '<!--'* ]] && continue
    fi
    if [[ "$line" =~ $key_re ]]; then
      value=${BASH_REMATCH[2]}
      value=${value%"${value##*[![:space:]]}"}
      item=$(normalize_list "$value" "${BASH_REMATCH[1]}") || exit 1
      if [[ "${BASH_REMATCH[1]}" == Related-Issues ]]; then
        related_seen=$((related_seen + 1))
        related=$item
      else
        completes_seen=$((completes_seen + 1))
        completes=$item
      fi
      continue
    fi
    shopt -s nocasematch
    if [[ "$line" =~ $near_re ]]; then
      shopt -u nocasematch
      fail "malformed delivery metadata line: use exactly 'Related-Issues:' and 'Completes-Issues:' at the start of a line"
    fi
    shopt -u nocasematch
  done <"$file"
  ((related_seen <= 1)) || fail 'Related-Issues appears more than once'
  ((completes_seen <= 1)) || fail 'Completes-Issues appears more than once'
  if ((related_seen == 0 && completes_seen == 0)); then
    printf 'metadata=undeclared\nrelated=none\ncompletes=none\n'
    return
  fi
  ((related_seen == 1)) || fail 'Completes-Issues requires Related-Issues'
  ((completes_seen == 1)) || fail 'Related-Issues requires Completes-Issues (use "Completes-Issues: none" for a partial PR)'
  if [[ "$completes" != none ]]; then
    [[ "$related" != none ]] || fail 'every completed Issue must also be listed in Related-Issues'
    for item in ${completes//,/ }; do
      [[ ",$related," == *",$item,"* ]] || fail "completed Issue #$item must also be listed in Related-Issues"
    done
  fi
  printf 'metadata=declared\nrelated=%s\ncompletes=%s\n' "$related" "$completes"
}

# commit_metadata SHA prints the metadata of one commit, using a reviewed
# correction when one is recorded for it in the corrections file of the
# revision being resolved (load_corrections), never the working tree.
commit_metadata() {
  local sha=$1 corrections=$temporary/corrections line correction_re
  correction_re='^([0-9a-f]{40}) related=([0-9,]+|none) completes=([0-9,]+|none)$'
  if [[ -s "$corrections" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
      [[ -z "$line" || "$line" == '#'* ]] && continue
      [[ "$line" =~ $correction_re ]] || fail 'malformed line in .github/delivery-corrections.txt'
      if [[ "${BASH_REMATCH[1]}" == "$sha" ]]; then
        printf 'Related-Issues: %s\nCompletes-Issues: %s\n' \
          "$(sed 's/\([0-9][0-9]*\)/#\1/g; s/,/, /g' <<<"${BASH_REMATCH[2]}")" \
          "$(sed 's/\([0-9][0-9]*\)/#\1/g; s/,/, /g' <<<"${BASH_REMATCH[3]}")" >"$temporary/message"
        parse_message "$temporary/message" | sed 's/^metadata=declared$/metadata=corrected/'
        return
      fi
    done <"$corrections"
  fi
  git -C "$repository_root" log -1 --format=%B "$sha" >"$temporary/message"
  (parse_message "$temporary/message") || fail "commit $sha has malformed delivery metadata"
}

# commit_line SHA prints one normalized line for a commit. release= names
# the stable version a release commit records (the Release PR squash), else
# none.
commit_line() {
  local sha=$1 subject pr=none metadata pr_re release=none manifest parent
  pr_re=' \(#([1-9][0-9]{0,8})\)$'
  subject=$(git -C "$repository_root" log -1 --format=%s "$sha")
  [[ "$subject" =~ $pr_re ]] && pr=${BASH_REMATCH[1]}
  manifest=$(manifest_at "$sha") || exit 1
  parent=none
  git -C "$repository_root" cat-file -e "$sha^1" 2>/dev/null && { parent=$(manifest_at "$sha^1") || exit 1; }
  # Adopting the manifest is not a release; a later manifest change is.
  [[ "$manifest" != none && "$parent" != none && "$manifest" != "$parent" ]] && release=$manifest
  metadata=$(commit_metadata "$sha") || exit 1
  printf 'commit=%s pr=%s release=%s %s\n' "$sha" "$pr" "$release" "$(tr '\n' ' ' <<<"$metadata" | sed 's/ $//')"
}

# load_corrections REV reads .github/delivery-corrections.txt as committed at
# REV, so an Issue set depends only on that revision.
load_corrections() {
  if [[ -n "${AXIOM_RELEASE_CORRECTIONS_REVISION:-}${AXIOM_RELEASE_CORRECTIONS_DIGEST:-}" ]]; then
    [[ "$mode" == release ]] || fail 'recovery corrections are only valid for release resolution'
    source "$repository_root/scripts/release-recovery.sh"
    recovery_validate "$tag" "$1" "$temporary/corrections" >/dev/null || exit 1
  else
  git -C "$repository_root" show "$1:.github/delivery-corrections.txt" >"$temporary/corrections" 2>/dev/null \
    || : >"$temporary/corrections"
  fi
}

full_sha() {
  local value=$1
  [[ "$value" =~ ^[0-9a-f]{40}$ ]] || fail 'full 40-character revision required'
  git -C "$repository_root" cat-file -e "$value^{commit}" 2>/dev/null || fail "not a known commit: $value"
}

# after_boundary VERSION succeeds when VERSION is a stable version above the
# legacy boundary.
after_boundary() {
  local a b
  IFS=. read -r -a a <<<"$1"
  IFS=. read -r -a b <<<"$legacy_boundary"
  ((a[0] != b[0])) && { ((a[0] > b[0])); return; }
  ((a[1] != b[1])) && { ((a[1] > b[1])); return; }
  ((a[2] > b[2]))
}

# manifest_at SHA prints the Release Please root version at SHA, or none.
manifest_at() {
  local content
  content=$(git -C "$repository_root" show "$1:.release-please-manifest.json" 2>/dev/null) || { printf 'none'; return; }
  content=$(tr -d ' \t\r\n' <<<"$content")
  [[ "$content" =~ ^\{\"\.\":\"([0-9]+\.[0-9]+\.[0-9]+)\"\}$ ]] || fail 'release manifest is malformed'
  printf '%s' "${BASH_REMATCH[1]}"
}

temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT

mode=${1:-}
[[ -n "$mode" ]] && shift
file=
body=
title=
from=
to=
tag=
revision=
while (($#)); do
  case "$1" in
    --file) file=${2:-}; shift 2 ;;
    --body) body=${2:-}; shift 2 ;;
    --title) title=${2:-}; shift 2 ;;
    --from) from=${2:-}; shift 2 ;;
    --to) to=${2:-}; shift 2 ;;
    --tag) tag=${2:-}; shift 2 ;;
    --revision) revision=${2:-}; shift 2 ;;
    *) fail 'invalid argument' ;;
  esac
done

case "$mode" in
  parse)
    if [[ -n "$file" ]]; then
      [[ -f "$file" ]] || fail 'message file not found'
      parse_message "$file"
    else
      cat >"$temporary/message"
      parse_message "$temporary/message"
    fi
    ;;
  check-pr)
    [[ -f "$body" && -f "$title" ]] || fail 'check-pr requires --body FILE and --title FILE'
    # The title becomes the squash subject. Reject unsupported types and
    # malformed subjects before Release Please can silently omit a change.
    subject=$(cat "$title")
    conventional_re='^(feat|fix|docs|test|refactor|perf|build|ci|chore|security|revert)(\([^()[:space:]]+\))?!?: [^[:space:]].*$'
    [[ "$subject" != *$'\n'* && "$subject" != *$'\r'* && "$subject" =~ $conventional_re ]] \
      || fail 'PR title must follow Conventional Commits: type(scope): description (see CONTRIBUTING.md)'
    # GitHub closes an Issue when a closing keyword reaches the default
    # branch (PR description or squash commit message). Delivery closes
    # Issues only at release publication, so these keywords are refused
    # anywhere in the title or body, code blocks included.
    closing_re='(^|[^[:alnum:]_])(close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved)([[:space:]]*:[[:space:]]*|[[:space:]]+)[*_[(]*(#[0-9]+|[[:alnum:]_.-]+/[[:alnum:]_.-]+#[0-9]+|https?://github\.com/[^[:space:]]+/issues/[0-9]+)'
    shopt -s nocasematch
    while IFS= read -r line || [[ -n "$line" ]]; do
      if [[ "$line" =~ $closing_re ]]; then
        shopt -u nocasematch
        fail "GitHub closing keyword found ('${BASH_REMATCH[2]} ${BASH_REMATCH[4]}'); use Related-Issues/Completes-Issues instead"
      fi
    done < <(cat "$title" "$body")
    shopt -u nocasematch
    result=$(parse_message "$body") || exit 1
    grep -Fxq metadata=declared <<<"$result" \
      || fail 'PR description must declare "Related-Issues:" and "Completes-Issues:" (see the PR template)'
    # GitHub wraps the squash commit body near 72 columns: a longer key line
    # would split, and prose could wrap a key to the start of a line. The
    # metadata must parse identically after such a wrap.
    long=$(awk '/^(Related|Completes)-Issues:/ && length > 72 {print; exit}' "$body")
    [[ -z "$long" ]] || fail 'a Related-Issues/Completes-Issues line is longer than 72 characters; split the PR or shorten the list'
    fold -s -w 72 "$body" >"$temporary/wrapped"
    wrapped=$(parse_message "$temporary/wrapped") || fail 'metadata does not survive the 72-column squash wrap; keep each key on its own short line'
    [[ "$wrapped" == "$result" ]] || fail 'metadata changes under the 72-column squash wrap; keep "Related-Issues:"/"Completes-Issues:" text out of long prose lines'
    printf '%s\n' "$result"
    ;;
  commits)
    full_sha "$from"
    full_sha "$to"
    git -C "$repository_root" merge-base --is-ancestor "$from" "$to" || fail 'FROM is not an ancestor of TO'
    load_corrections "$to"
    while IFS= read -r sha; do
      commit_line "$sha"
    done < <(git -C "$repository_root" rev-list --first-parent --reverse "$from..$to")
    ;;
  release)
    facts=$("$repository_root/scripts/release-tag-version.sh" "$tag") || exit 1
    version=$(awk -F= '$1 == "version" {print $2}' <<<"$facts")
    channel=$(awk -F= '$1 == "channel" {print $2}' <<<"$facts")
    full_sha "$revision"
    printf 'deliveryVersion=1\ntag=%s\nrevision=%s\n' "$tag" "$revision"
    load_corrections "$revision"
    if [[ "$channel" != stable ]]; then
      # A release candidate is not a delivery: it never closes or releases
      # an Issue, so it carries no Issue set.
      printf 'issues=not_applicable\n'
      exit 0
    fi
    [[ $(manifest_at "$revision") == "$version" ]] \
      || fail "revision does not record version $version (stable Issues are resolved at the release commit)"
    parent_manifest=$(manifest_at "$revision^1") || exit 1
    [[ "$parent_manifest" != "$version" ]] || fail "revision is not the release commit of $version"
    # The previous release boundary is the previous first-parent commit that
    # changed the manifest; the ranges of consecutive releases are disjoint.
    base=$(git -C "$repository_root" log --first-parent --max-count=1 --format=%H "$revision^1" -- .release-please-manifest.json)
    printf 'range_base=%s\n' "${base:-none}"
    if [[ -n "$base" ]]; then
      git -C "$repository_root" rev-list --first-parent --reverse "$base..$revision" >"$temporary/range"
    else
      git -C "$repository_root" rev-list --first-parent --reverse "$revision" >"$temporary/range"
    fi
    : >"$temporary/delivered"
    undeclared=0
    total=0
    enforce=false
    after_boundary "$version" && enforce=true
    printf 'legacy_boundary=v%s\nundeclared_policy=%s\n' "$legacy_boundary" \
      "$([[ "$enforce" == true ]] && printf fail_closed || printf legacy_allowed)"
    release_subject_re="^chore\\(main\\): release ${version//./\\.}( \\(#[1-9][0-9]{0,8}\\))?\$"
    while IFS= read -r sha; do
      total=$((total + 1))
      line=$(commit_line "$sha") || exit 1
      if [[ "$line" == *' metadata=undeclared '* ]]; then
        undeclared=$((undeclared + 1))
        # Only this release's own Release Please commit may stay undeclared.
        if [[ "$enforce" == true ]] && ! { [[ "$sha" == "$revision" ]] \
          && [[ $(git -C "$repository_root" log -1 --format=%s "$sha") =~ $release_subject_re ]]; }; then
          fail "commit $sha in the v$version range has no delivery metadata; record a reviewed correction in .github/delivery-corrections.txt"
        fi
      fi
      completes=$(sed -E 's/.* completes=([^ ]+)$/\1/' <<<"$line")
      pr=$(sed -E 's/^commit=[0-9a-f]+ pr=([^ ]+) .*/\1/' <<<"$line")
      [[ "$completes" == none ]] && continue
      for item in ${completes//,/ }; do
        printf '%s %s %s\n' "$item" "$pr" "$sha" >>"$temporary/delivered"
      done
    done <"$temporary/range"
    printf 'commits=%s\nundeclared_commits=%s\n' "$total" "$undeclared"
    issues=$(awk '{print $1}' "$temporary/delivered" | LC_ALL=C sort -n -u | paste -sd, -)
    printf 'issues=%s\n' "${issues:-none}"
    # One line per delivered Issue with the PRs that completed it, in range
    # order; an Issue completed by several PRs appears once.
    for item in ${issues//,/ }; do
      prs=$(awk -v n="$item" '$1 == n && $2 != "none" {print "#" $2}' "$temporary/delivered" | awk '!seen[$0]++' | paste -sd, -)
      printf 'issue.%s=%s\n' "$item" "${prs:-none}"
    done
    ;;
  *)
    fail 'usage: delivery-issues.sh parse|check-pr|commits|release [options]'
    ;;
esac
