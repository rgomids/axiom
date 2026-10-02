#!/usr/bin/env bash
# Resolve an explicitly pinned, append-only delivery correction input.
# Git reads only; source artifacts and release-range boundaries stay unchanged.
set -euo pipefail
umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
source_revision=
corrections_revision=
corrections_digest=
main_ref=origin/main
file_output=
fail() { printf 'release_corrections_error: %s\n' "$1" >&2; exit 1; }
while (($#)); do
  case "$1" in
    --source-revision) source_revision=${2:-}; shift 2 ;;
    --corrections-revision) corrections_revision=${2:-}; shift 2 ;;
    --corrections-digest) corrections_digest=${2:-}; shift 2 ;;
    --main-ref) main_ref=${2:-}; shift 2 ;;
    --file-output) file_output=${2:-}; shift 2 ;;
    *) fail 'invalid argument' ;;
  esac
done
[[ "$source_revision" =~ ^[0-9a-f]{40}$ && "$corrections_revision" =~ ^[0-9a-f]{40}$ ]] || fail 'full source and corrections revisions required'
[[ "$corrections_digest" =~ ^[0-9a-f]{64}$ ]] || fail 'full corrections SHA-256 required'
git -C "$repository_root" cat-file -e "$source_revision^{commit}" 2>/dev/null || fail 'unknown source revision'
git -C "$repository_root" cat-file -e "$corrections_revision^{commit}" 2>/dev/null || fail 'unknown corrections revision'
main_sha=$(git -C "$repository_root" rev-parse --verify "$main_ref^{commit}" 2>/dev/null) || fail 'unknown main ref'
temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT
git -C "$repository_root" rev-list --first-parent "$main_sha" >"$temporary/main"
grep -Fxq "$source_revision" "$temporary/main" || fail 'source revision is not on first-parent main'
grep -Fxq "$corrections_revision" "$temporary/main" || fail 'corrections revision is not on first-parent main'
git -C "$repository_root" merge-base --is-ancestor "$source_revision" "$corrections_revision" || fail 'corrections revision does not descend from source'
command -v jq >/dev/null 2>&1 || fail 'jq is required'
manifest_at() {
  git -C "$repository_root" show "$1:.release-please-manifest.json" 2>/dev/null \
    | jq -er 'select(type == "object" and keys == ["."]) | .["."] | select(type == "string" and test("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"))'
}
source_version=$(manifest_at "$source_revision") || fail 'source must record a stable release manifest'
source_parent=$(git -C "$repository_root" rev-parse --verify "$source_revision^1" 2>/dev/null) || fail 'source release commit must have a parent'
parent_version=$(manifest_at "$source_parent") || fail 'source parent must record a stable manifest'
[[ "$parent_version" != "$source_version" ]] || fail 'source is not the commit introducing the release version'
base=$(git -C "$repository_root" log --first-parent --max-count=1 --format=%H "$source_parent" -- .release-please-manifest.json)
[[ -n "$base" ]] || fail 'previous release boundary is missing'
git -C "$repository_root" rev-list --first-parent "$base..$source_revision" >"$temporary/range"
corrections_mode=$(git -C "$repository_root" ls-tree "$corrections_revision" -- .github/delivery-corrections.txt | awk '{print $1}')
[[ "$corrections_mode" == 100644 || "$corrections_mode" == 100755 ]] || fail 'corrections file missing or not regular at pinned revision'
git -C "$repository_root" show "$corrections_revision:.github/delivery-corrections.txt" >"$temporary/corrections" 2>/dev/null || fail 'corrections file missing at pinned revision'
if command -v sha256sum >/dev/null 2>&1; then
  actual_digest=$(sha256sum "$temporary/corrections" | awk '{print $1}')
else
  actual_digest=$(shasum -a 256 "$temporary/corrections" | awk '{print $1}')
fi
[[ "$actual_digest" == "$corrections_digest" ]] || fail 'corrections digest mismatch'
if git -C "$repository_root" cat-file -e "$source_revision:.github/delivery-corrections.txt" 2>/dev/null; then
  source_mode=$(git -C "$repository_root" ls-tree "$source_revision" -- .github/delivery-corrections.txt | awk '{print $1}')
  [[ "$source_mode" == 100644 || "$source_mode" == 100755 ]] || fail 'source corrections file is not regular'
  git -C "$repository_root" show "$source_revision:.github/delivery-corrections.txt" >"$temporary/source"
else
  : >"$temporary/source"
fi

validate_list() {
  local list=$1 item seen=, count=0
  [[ "$list" == none ]] && return
  [[ "$list" =~ ^[1-9][0-9]{0,8}(,[1-9][0-9]{0,8})*$ ]] || fail 'invalid Issue list in corrections file'
  for item in ${list//,/ }; do
    [[ "$seen" != *",$item,"* ]] || fail 'duplicate Issue in corrections file'
    seen="$seen$item,"
    count=$((count + 1))
  done
  ((count <= 20)) || fail 'correction lists more than 20 Issues'
}
validate_file() {
  local input=$1 rows=$2 line sha related completes item
  local row_re='^([0-9a-f]{40}) related=([^ ]+) completes=([^ ]+)$'
  : >"$rows"
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == '#'* ]] && continue
    [[ "$line" =~ $row_re ]] || fail 'malformed corrections row'
    sha=${BASH_REMATCH[1]}; related=${BASH_REMATCH[2]}; completes=${BASH_REMATCH[3]}
    validate_list "$related"
    validate_list "$completes"
    if [[ "$completes" != none ]]; then
      for item in ${completes//,/ }; do
        [[ ",$related," == *",$item,"* ]] || fail 'completed Issue missing from related Issues'
      done
    fi
    ! awk -v sha="$sha" '$1 == sha {found=1} END {exit !found}' "$rows" || fail 'duplicate commit in corrections file'
    printf '%s\n' "$line" >>"$rows"
  done <"$input"
}
validate_file "$temporary/source" "$temporary/source-rows"
validate_file "$temporary/corrections" "$temporary/rows"
while IFS= read -r line; do
  grep -Fxq "$line" "$temporary/rows" || fail 'existing source correction changed or removed'
done <"$temporary/source-rows"
while IFS= read -r line; do
  grep -Fxq "$line" "$temporary/source-rows" && continue
  sha=${line%% *}
  git -C "$repository_root" cat-file -e "$sha^{commit}" 2>/dev/null || fail 'new correction names unknown commit'
  [[ "$sha" != "$source_revision" ]] && grep -Fxq "$sha" "$temporary/range" || fail 'new correction target is outside source release range'
done <"$temporary/rows"
if [[ -n "$file_output" ]]; then
  [[ "$file_output" == /* && ! -L "$file_output" && ( ! -e "$file_output" || -f "$file_output" ) ]] || fail 'absolute regular output file required'
  cp "$temporary/corrections" "$file_output"
fi
printf 'corrections_revision=%s\ncorrections_sha256=%s\n' "$corrections_revision" "$actual_digest"
