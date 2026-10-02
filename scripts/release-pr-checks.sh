#!/usr/bin/env bash
# Read-only resolution followed only by required workflow dispatches.
# Release Please output is intentionally irrelevant, including unchanged PRs.
#   release-pr-checks.sh resolve  REPO [VERSION]
#   release-pr-checks.sh dispatch REPO VERSION
# dispatch requires the open Release PR to record VERSION, the version the
# release plan validated (ADR-0011): a Release PR with another version gets no
# required checks and so cannot be merged.
set -euo pipefail
fail() { printf 'release_pr_checks_error: %s\n' "$1" >&2; exit 1; }
mode=${1:-}
repository=${2:-}
expected=${3:-}
[[ "$mode" == resolve || "$mode" == dispatch ]] || fail 'expected resolve or dispatch'
[[ "$repository" == rgomids/axiom ]] || fail 'unexpected repository'
[[ -z "$expected" || "$expected" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || fail 'invalid planned version'
[[ "$mode" == resolve || -n "$expected" ]] || fail 'dispatch requires the planned version'
temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT
# Broad discovery catches malformed release identities instead of silently
# dropping them through author/base/head API filters. Pagination is mandatory.
gh api --paginate --slurp "repos/$repository/pulls?state=open&per_page=100" >"$temporary/pages" || fail 'cannot resolve Release PRs'
jq -e 'type == "array" and all(.[]; type == "array")' "$temporary/pages" >/dev/null || fail 'invalid pulls response'
jq '[.[][] | select((.head.ref // "" | startswith("release-please--")) or any(.labels[]?; .name == "autorelease: pending"))]' "$temporary/pages" >"$temporary/candidates"
count=$(jq 'length' "$temporary/candidates")
if [[ "$count" == 0 ]]; then
  # A planned dispatch must have produced a Release PR.
  [[ "$mode" == resolve ]] || fail 'no open Release PR for the planned version'
  printf 'release_pr_checks=no_open_release_pr\n'
  exit 0
fi
[[ "$count" == 1 ]] || fail 'ambiguous Release PR candidates'
jq '.[0]' "$temporary/candidates" >"$temporary/pr"
validate() {
  jq -e --arg repo "$repository" '
    .state == "open" and .user.login == "github-actions[bot]"
    and .base.repo.full_name == $repo and .head.repo.full_name == $repo
    and .base.ref == "main" and .head.ref == "release-please--branches--main"
    and any(.labels[]?; .name == "autorelease: pending")
    and (.number | type == "number" and . > 0 and floor == .)
    and (.head.sha | type == "string" and test("^[0-9a-f]{40}$"))
  ' "$1" >/dev/null || fail 'inconsistent Release PR identity'
  if [[ -n "$expected" ]]; then
    jq -e --arg title "chore(main): release $expected" '.title == $title' "$1" >/dev/null \
      || fail "Release PR does not record the planned version $expected"
  fi
}
validate "$temporary/pr"
number=$(jq -r '.number' "$temporary/pr")
RELEASE_BRANCH=$(jq -r '.head.ref' "$temporary/pr")
sha=$(jq -r '.head.sha' "$temporary/pr")
assert_current() {
  gh api "repos/$repository/pulls/$number" >"$temporary/current" || fail 'cannot read Release PR'
  validate "$temporary/current"
  [[ $(jq -r '[.number,.head.ref,.head.sha] | @tsv' "$temporary/current") == "$(jq -r '[.number,.head.ref,.head.sha] | @tsv' "$temporary/pr")" ]] || fail 'Release PR head changed'
  gh api "repos/$repository/git/ref/heads/$RELEASE_BRANCH" >"$temporary/ref" || fail 'cannot resolve head ref'
  jq -e --arg ref "refs/heads/$RELEASE_BRANCH" --arg sha "$sha" '.ref == $ref and .object.type == "commit" and .object.sha == $sha' "$temporary/ref" >/dev/null || fail 'inconsistent head ref SHA'
}
assert_current
printf 'release_pr=#%s branch=%s head=%s\n' "$number" "$RELEASE_BRANCH" "$sha"
if [[ "$mode" == dispatch ]]; then
  gh workflow run ci.yml --repo "$repository" --ref "$RELEASE_BRANCH"
  assert_current
  gh workflow run delivery-metadata.yml --repo "$repository" --ref "$RELEASE_BRANCH"
  assert_current
  printf 'release_pr_checks=dispatched\n'
fi
