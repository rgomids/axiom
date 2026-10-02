#!/usr/bin/env bash
# Release plan: the preflight of a NEW stable release, before any Release PR.
# Read-only and Git-only (local history plus remote tags). Prints closed
# key=value Evidence ending in result=pass, or release_plan_error on stderr
# with exit 1. See ADR-0011 and CONTRIBUTING.md "Release flow".
#
#   release-plan.sh [--main-ref REF] [--remote NAME]
#
# Range: the first-parent commits of main after the last commit that changed
# .release-please-manifest.json (the previous release commit). Rules:
# - the previous release v<manifest> is published, its tag at that commit;
# - every commit in the range has a Conventional Commit subject (the squash
#   PR title, same grammar as delivery-issues.sh check-pr) and declared or
#   reviewed delivery metadata (delivery-issues.sh commits, corrections read
#   at main); one inconsistent commit stops the plan;
# - releasable when a commit has a visible changelog type (feat, fix,
#   security, perf, revert), a breaking change or a Release-As footer;
# - bump (release-please 17.6.0 default strategy, bump-minor-pre-major,
#   no bump-patch-for-minor-pre-major): breaking -> major (minor while 0.x),
#   feat -> minor, anything else -> patch; Release-As X.Y.Z forces X.Y.Z;
# - the planned tag does not exist and is newer than every stable tag.
# Release Please still computes the version; release.sh start and
# release-please.yml refuse a Release PR whose version differs from this plan.
set -euo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
main_ref=origin/main
remote=origin
while (($#)); do
  case "$1" in
    --main-ref) main_ref=${2:-}; shift 2 ;;
    --remote) remote=${2:-}; shift 2 ;;
    *) printf 'release_plan_error: invalid argument\n' >&2; exit 1 ;;
  esac
done

fail() {
  printf 'release_plan_error: %s\n' "$1" >&2
  exit 1
}

temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT

git_() { git -C "$repository_root" "$@"; }

main=$(git_ rev-parse --verify --quiet "$main_ref^{commit}") || fail "main ref not found: $main_ref"

numeric='(0|[1-9][0-9]*)'
manifest_at() {
  local content
  content=$(git_ show "$1:.release-please-manifest.json" 2>/dev/null) || { printf 'none'; return; }
  content=$(tr -d ' \t\r\n' <<<"$content")
  [[ "$content" =~ ^\{\"\.\":\"(${numeric}\.${numeric}\.${numeric})\"\}$ ]] || fail 'release manifest is malformed'
  printf '%s' "${BASH_REMATCH[1]}"
}

compare_core() {
  local -a a b
  local i
  IFS=. read -r -a a <<<"$1"
  IFS=. read -r -a b <<<"$2"
  for i in 0 1 2; do
    if ((10#${a[i]} < 10#${b[i]})); then printf -- '-1'; return; fi
    if ((10#${a[i]} > 10#${b[i]})); then printf '1'; return; fi
  done
  printf '0'
}

current=$(manifest_at "$main")
[[ "$current" != none ]] || fail 'main has no .release-please-manifest.json'
boundary=$(git_ log --first-parent --max-count=1 --format=%H "$main" -- .release-please-manifest.json)
[[ -n "$boundary" ]] || fail 'previous release commit not found'

# Remote tags, peeled to commits (same table as release-preflight.sh).
remote_tags=$(git_ ls-remote --tags "$remote" 2>/dev/null) || fail "cannot read tags from remote: $remote"
tag_table=$(awk '
  NF == 2 {
    name = $2; sub(/^refs\/tags\//, "", name)
    if (name ~ /\^\{\}$/) { sub(/\^\{\}$/, "", name); peeled[name] = $1 }
    else if (!(name in plain)) { plain[name] = $1 }
  }
  END { for (name in plain) print name, ((name in peeled) ? peeled[name] : plain[name]) }' <<<"$remote_tags")
tag_commit() { awk -v name="$1" '$1 == name {print $2}' <<<"$tag_table"; }

previous_tag_commit=$(tag_commit "v$current")
[[ -n "$previous_tag_commit" ]] \
  || fail "release v$current is recorded on main but not published; finish it with \$axiom-release first"
[[ "$previous_tag_commit" == "$boundary" ]] \
  || fail "tag v$current is not at the release commit that recorded it ($boundary)"

latest_stable=none
while read -r name _; do
  [[ -n "${name:-}" ]] || continue
  if [[ "$name" =~ ^v(${numeric}\.${numeric}\.${numeric})$ ]]; then
    candidate=${BASH_REMATCH[1]}
    if [[ "$latest_stable" == none || $(compare_core "$candidate" "$latest_stable") == 1 ]]; then
      latest_stable=$candidate
    fi
  fi
done <<<"$tag_table"

# Delivery metadata of every commit in the range: delivery-issues.sh is the
# single parser; it fails closed on malformed metadata.
"$repository_root/scripts/delivery-issues.sh" commits --from "$boundary" --to "$main" >"$temporary/delivery" \
  2>"$temporary/delivery-error" || fail "$(sed 's/^delivery_metadata_error: //' "$temporary/delivery-error" | head -n 1)"

# Same grammar as delivery-issues.sh check-pr (the squash subject is the PR
# title); test-release-flow.sh asserts both expressions stay identical.
conventional_re='^(feat|fix|docs|test|refactor|perf|build|ci|chore|security|revert)(\([^()[:space:]]+\))?!?: [^[:space:]].*$'
visible_re='^(feat|fix|security|perf|revert)$'
release_as_re='^release-as:[[:space:]]*v?((0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))[[:space:]]*$'

rank=0 # 0 none, 1 patch, 2 minor, 3 major
releasable=false
release_as=none
total=0
: >"$temporary/commits"
: >"$temporary/changes"
: >"$temporary/delivered"
while IFS= read -r line; do
  [[ -n "$line" ]] || continue
  total=$((total + 1))
  sha=$(sed -E 's/^commit=([0-9a-f]{40}) .*/\1/' <<<"$line")
  pr=$(sed -E 's/^commit=[0-9a-f]+ pr=([^ ]+) .*/\1/' <<<"$line")
  release=$(sed -E 's/.* release=([^ ]+) .*/\1/' <<<"$line")
  metadata=$(sed -E 's/.* metadata=([a-z]+) .*/\1/' <<<"$line")
  completes=$(sed -E 's/.* completes=([^ ]+)$/\1/' <<<"$line")
  [[ "$release" == none ]] || fail "commit $sha records release $release inside the release range"
  [[ "$metadata" != undeclared ]] \
    || fail "commit $sha has no delivery metadata; merge a reviewed correction to .github/delivery-corrections.txt"
  subject=$(git_ log -1 --format=%s "$sha")
  [[ "$subject" =~ $conventional_re ]] || fail "commit $sha subject is not a Conventional Commit"
  type=${BASH_REMATCH[1]}
  breaking=false
  [[ "$subject" =~ ^[a-z]+(\([^()[:space:]]+\))?!: ]] && breaking=true
  git_ log -1 --format=%B "$sha" >"$temporary/message"
  grep -Eq '^BREAKING[ -]CHANGE:' "$temporary/message" && breaking=true
  grep -Eq '^[[:space:]]*BEGIN_COMMIT_OVERRIDE[[:space:]]*$' "$temporary/message" \
    && fail "commit $sha uses a Release Please commit override, which the release plan does not model"
  forced=none
  shopt -s nocasematch
  while IFS= read -r body_line; do
    body_line=${body_line%$'\r'}
    if [[ "$body_line" =~ $release_as_re ]]; then
      [[ "$forced" == none || "$forced" == "${BASH_REMATCH[1]}" ]] || { shopt -u nocasematch; fail "commit $sha has conflicting Release-As footers"; }
      forced=${BASH_REMATCH[1]}
    fi
  done <"$temporary/message"
  shopt -u nocasematch
  if [[ "$forced" != none ]]; then
    [[ "$release_as" == none || "$release_as" == "$forced" ]] || fail 'the release range has conflicting Release-As footers'
    release_as=$forced
  fi
  visible=false
  [[ "$type" =~ $visible_re ]] && visible=true
  commit_releasable=false
  if [[ "$visible" == true || "$breaking" == true || "$forced" != none ]]; then
    releasable=true
    commit_releasable=true
    # Subjects are shown to the human: drop control and bidi override bytes.
    printf '%s\n' "$(printf '%s' "$subject" | tr -d '\000-\037\177' \
      | LC_ALL=C sed -e $'s/\xe2\x80[\xaa-\xae]//g' -e $'s/\xe2\x81[\xa6-\xa9]//g')" >>"$temporary/changes"
  fi
  if [[ "$breaking" == true ]]; then
    ((rank < 3)) && rank=3
  elif [[ "$type" == feat ]]; then
    ((rank < 2)) && rank=2
  else
    ((rank < 1)) && rank=1
  fi
  printf 'commit=%s pr=%s type=%s breaking=%s visible=%s releasable=%s metadata=%s completes=%s\n' \
    "$sha" "$pr" "$type" "$breaking" "$visible" "$commit_releasable" "$metadata" "$completes" >>"$temporary/commits"
  if [[ "$completes" != none ]]; then
    tr ',' '\n' <<<"$completes" >>"$temporary/delivered"
  fi
done <"$temporary/delivery"

IFS=. read -r major minor patch <<<"$current"
bump=none
planned=none
if [[ "$releasable" == true ]]; then
  case "$rank" in
    3) if ((major == 0)); then bump=minor; else bump=major; fi ;;
    2) bump=minor ;;
    *) bump=patch ;;
  esac
  case "$bump" in
    major) planned="$((major + 1)).0.0" ;;
    minor) planned="$major.$((minor + 1)).0" ;;
    patch) planned="$major.$minor.$((patch + 1))" ;;
  esac
  if [[ "$release_as" != none ]]; then
    bump=release_as
    planned=$release_as
  fi
  [[ $(compare_core "$planned" "$current") == 1 ]] || fail "planned version $planned is not newer than v$current"
  [[ -z $(tag_commit "v$planned") ]] || fail "tag v$planned already exists"
  if [[ "$latest_stable" != none ]]; then
    [[ $(compare_core "$planned" "$latest_stable") == 1 ]] || fail "planned version $planned is not newer than the latest stable release v$latest_stable"
  fi
fi

issues=$(LC_ALL=C sort -n -u "$temporary/delivered" | paste -sd, -)
printf 'planVersion=1\n'
printf 'main=%s\n' "$main"
printf 'current_version=%s\n' "$current"
printf 'previous_tag=v%s\n' "$current"
printf 'previous_release_commit=%s\n' "$boundary"
printf 'latest_stable=%s\n' "$latest_stable"
printf 'commits=%s\n' "$total"
printf 'releasable=%s\n' "$releasable"
printf 'bump=%s\n' "$bump"
printf 'release_as=%s\n' "$release_as"
printf 'planned_version=%s\n' "$planned"
printf 'planned_tag=%s\n' "$([[ "$planned" == none ]] && printf none || printf 'v%s' "$planned")"
printf 'issues=%s\n' "${issues:-none}"
cat "$temporary/commits"
n=0
while IFS= read -r change; do
  n=$((n + 1))
  printf 'change.%s=%s\n' "$n" "$change"
done <"$temporary/changes"
printf 'result=pass\n'
