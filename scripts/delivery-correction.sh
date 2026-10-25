#!/usr/bin/env bash
# Proposes reviewed delivery corrections for squash commits on main that lost
# the Related-Issues/Completes-Issues lines their PR declared (a custom squash
# message at merge). It never merges, never edits history and never touches an
# Issue: a human reviews and merges each correction PR like any other.
#
#   delivery-correction.sh plan    --repo R --to SHA
#       Read-only. One `correction.N=<sha> pr=<n> head=<sha> related=<list|none>
#       completes=<list|none>` line per ordinary first-parent commit since the
#       latest release commit up to TO that is undeclared and has no reviewed
#       correction, then `corrections=N`. The values are those of the merged
#       PR's description, re-validated with delivery-issues.sh check-pr, and
#       only when its latest delivery-metadata check on the PR head passed.
#       Anything else (no single merged PR, a failed or missing check, an
#       invalid description) fails closed: the correction is then written by
#       hand as before.
#   delivery-correction.sh propose --repo R --to SHA
#       For each planned correction without a branch yet: creates branch
#       delivery-correction/<sha> at TO, appends the reason and the row to
#       .github/delivery-corrections.txt through the contents API, opens the
#       PR (Related-Issues: none, Completes-Issues: none) and dispatches its
#       required checks (ci.yml, delivery-metadata.yml), because events
#       created with GITHUB_TOKEN start no pull_request run. Every effect is
#       printed as an effect= line. An existing branch is left alone.
#   delivery-correction.sh pr-head --repo R --branch B --sha SHA --actor A
#       Read-only. The dispatch path of the delivery-metadata check for a
#       correction PR: passes only when the workflow bot dispatched it for the
#       head of the one open bot-authored PR from delivery-correction/<sha> of
#       this repository, whose head is one commit on main history that only
#       appends lines to .github/delivery-corrections.txt, whose description
#       declares no Issue and which GitHub would not close any Issue with.
#
# The PR description is untrusted: it reaches the validator only as a file,
# and only the normalized lists the validator prints are written.
set -euo pipefail

umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
bot_login='github-actions[bot]'
corrections_path=.github/delivery-corrections.txt

fail() {
  printf 'delivery_correction_error: %s\n' "$1" >&2
  exit 1
}

mode=${1:-}
[[ -n "$mode" ]] && shift
repository=
to=
branch=
head_sha=
actor=
while (($#)); do
  case "$1" in
    --repo) repository=${2:-}; shift 2 ;;
    --to) to=${2:-}; shift 2 ;;
    --branch) branch=${2:-}; shift 2 ;;
    --sha) head_sha=${2:-}; shift 2 ;;
    --actor) actor=${2:-}; shift 2 ;;
    *) fail 'invalid argument' ;;
  esac
done
[[ "$repository" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || fail 'repository must be OWNER/NAME'
command -v gh >/dev/null 2>&1 || fail 'gh is required'
command -v jq >/dev/null 2>&1 || fail 'jq is required'

temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT

field() {
  sed -nE "s/.* $1=([^ ]+)( .*)?$/\\1/p" <<<"$2"
}

# closing_refs PR prints the Issues GitHub would close when PR merges.
closing_refs() {
  # shellcheck disable=SC2016
  gh api graphql -f owner="${repository%%/*}" -f name="${repository#*/}" -F number="$1" -f query='
    query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) {
      pullRequest(number: $number) { closingIssuesReferences(first: 20) { nodes { number } } } } }' \
    | jq -r '[.data.repository.pullRequest.closingIssuesReferences.nodes[].number] | map("#\(.)") | join(", ")'
}

# plan_corrections writes the planned correction lines to $temporary/plan.
plan_corrections() {
  local boundary line sha pr head n=0 conclusion result
  [[ "$to" =~ ^[0-9a-f]{40}$ ]] || fail "$mode requires a full --to"
  boundary=$(git -C "$repository_root" log --first-parent --max-count=1 --format=%H "$to" -- .release-please-manifest.json)
  [[ -n "$boundary" ]] || fail 'no release commit precedes the requested revision'
  "$repository_root/scripts/delivery-issues.sh" commits --from "$boundary" --to "$to" >"$temporary/commits" || exit 1
  : >"$temporary/plan"
  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    [[ "$(field release "$line")" == none && "$(field metadata "$line")" == undeclared ]] || continue
    sha=$(sed -E 's/^commit=([0-9a-f]{40}) .*/\1/' <<<"$line")
    gh api "repos/$repository/commits/$sha/pulls" >"$temporary/pulls" || fail "cannot read the PR merged as $sha"
    jq -c --arg sha "$sha" '[.[] | select(.merged_at != null and .merge_commit_sha == $sha and .base.ref == "main")]
      | if length == 1 then .[0] else empty end' "$temporary/pulls" >"$temporary/pr"
    [[ -s "$temporary/pr" ]] || fail "$sha is not the merge commit of exactly one merged PR; write its correction by hand"
    pr=$(jq -r '.number' "$temporary/pr")
    head=$(jq -r '.head.sha' "$temporary/pr")
    [[ "$pr" =~ ^[1-9][0-9]*$ && "$head" =~ ^[0-9a-f]{40}$ ]] || fail "PR merged as $sha has no readable number or head"
    gh api "repos/$repository/commits/$head/check-runs?per_page=100" >"$temporary/checks" || fail "cannot read the checks of PR #$pr"
    conclusion=$(jq -r '[.check_runs[] | select(.name == "delivery-metadata")] | sort_by(.id) | last
      | if . == null then "missing" elif .status != "completed" then "pending" else .conclusion end' "$temporary/checks")
    [[ "$conclusion" == success ]] || fail "the delivery-metadata check of PR #$pr is $conclusion at its head; write its correction by hand"
    jq -r '.title // ""' "$temporary/pr" >"$temporary/title"
    jq -r '.body // ""' "$temporary/pr" >"$temporary/body"
    result=$("$repository_root/scripts/delivery-issues.sh" check-pr --title "$temporary/title" --body "$temporary/body" 2>"$temporary/check-error") \
      || fail "the description of PR #$pr no longer validates ($(head -n 1 "$temporary/check-error")); write its correction by hand"
    n=$((n + 1))
    printf 'correction.%s=%s pr=%s head=%s related=%s completes=%s\n' "$n" "$sha" "$pr" "$head" \
      "$(sed -n 's/^related=//p' <<<"$result")" "$(sed -n 's/^completes=//p' <<<"$result")" >>"$temporary/plan"
  done <"$temporary/commits"
  printf 'corrections=%s\n' "$n" >>"$temporary/plan"
}

render_body() {
  cat <<EOF
## Context

PR #$2 declared \`Related-Issues: $4\` and \`Completes-Issues: $5\` in its
description, and its \`delivery-metadata\` check passed on head \`$3\`. Its
squash commit \`$1\` omitted both lines, so the release plan refuses the
undeclared commit.

## Scope

Append one reviewed row to \`$corrections_path\` restoring exactly the
validated relationship. Proposed automatically by
\`.github/workflows/delivery-correction.yml\`; merging this reviewed
correction records it.

## Non-goals

No history rewrite, no Issue relationship beyond the validated one, no Issue
completion and no release.

## Related issue / Specification / ADR

The delivery tracking and correction contract in CONTRIBUTING.md.

## Delivered Issues

Related-Issues: none
Completes-Issues: none

## Work Item lifecycle / authority

Not applicable. Human review and merge of this PR are the authority.

## Changes

- \`$corrections_path\`: \`$1 related=$4 completes=$5\`.

## Validation

\`delivery-correction.sh plan\` re-validated the PR description with
\`delivery-issues.sh check-pr\` and required its passing \`delivery-metadata\`
check. Required checks were dispatched on this branch.

## Documentation impact

None.

## Security impact

Only normalized Issue lists reach the file; the PR description is untrusted
input validated as a file.

## Limitations

Review the values against PR #$2 before merging.
EOF
}

case "$mode" in
  plan)
    plan_corrections
    cat "$temporary/plan"
    ;;
  propose)
    plan_corrections
    cat "$temporary/plan"
    gh api "repos/$repository/contents/$corrections_path?ref=$to" >"$temporary/file" || fail 'cannot read the corrections file at the requested revision'
    while IFS= read -r line; do
      [[ "$line" == correction.* ]] || continue
      sha=$(sed -E 's/^correction\.[0-9]+=([0-9a-f]{40}) .*/\1/' <<<"$line")
      pr=$(field pr "$line") head=$(field head "$line") related=$(field related "$line") completes=$(field completes "$line")
      correction_branch=delivery-correction/$sha
      if gh api "repos/$repository/git/ref/heads/$correction_branch" >/dev/null 2>&1; then
        printf 'effect=none branch=%s reason=already_proposed\n' "$correction_branch"
        continue
      fi
      jq -r '.content' "$temporary/file" | base64 --decode >"$temporary/corrections"
      {
        printf '\n# PR #%s declared Related-Issues: %s and Completes-Issues: %s; its\n' "$pr" "$related" "$completes"
        printf '# delivery-metadata check passed on head %s. Its squash\n' "$head"
        printf '# %s omitted the metadata. Proposed by\n' "$sha"
        printf '# delivery-correction.yml; merging this reviewed correction records it.\n'
        printf '%s related=%s completes=%s\n' "$sha" "$related" "$completes"
      } >>"$temporary/corrections"
      gh api --method POST "repos/$repository/git/refs" -f "ref=refs/heads/$correction_branch" -f "sha=$to" >/dev/null \
        || fail "cannot create $correction_branch"
      printf 'effect=branch_created branch=%s base=%s\n' "$correction_branch" "$to"
      gh api --method PUT "repos/$repository/contents/$corrections_path" -f "branch=$correction_branch" \
        -f "message=fix(delivery): restore squash metadata of PR #$pr" \
        -f "sha=$(jq -r '.sha' "$temporary/file")" -f "content=$(base64 <"$temporary/corrections" | tr -d '\n')" >/dev/null \
        || fail "cannot commit the correction on $correction_branch"
      printf 'effect=correction_committed branch=%s commit=%s related=%s completes=%s\n' "$correction_branch" "$sha" "$related" "$completes"
      render_body "$sha" "$pr" "$head" "$related" "$completes" >"$temporary/pr-body"
      gh pr create --repo "$repository" --base main --head "$correction_branch" \
        --title "fix(delivery): restore squash metadata of PR #$pr" --body-file "$temporary/pr-body" >"$temporary/pr-url" \
        || fail "cannot open the correction PR for $correction_branch"
      printf 'effect=pr_opened branch=%s url=%s\n' "$correction_branch" "$(head -n 1 "$temporary/pr-url")"
      gh workflow run ci.yml --repo "$repository" --ref "$correction_branch" >/dev/null || fail 'cannot dispatch CI on the correction PR'
      gh workflow run delivery-metadata.yml --repo "$repository" --ref "$correction_branch" >/dev/null \
        || fail 'cannot dispatch delivery-metadata on the correction PR'
      printf 'effect=checks_dispatched branch=%s workflows=ci.yml,delivery-metadata.yml\n' "$correction_branch"
    done <"$temporary/plan"
    ;;
  pr-head)
    [[ "$actor" == "$bot_login" ]] || fail "only $bot_login dispatches the correction PR delivery check (actor: ${actor:-none})"
    [[ "$branch" =~ ^delivery-correction/([0-9a-f]{40})$ ]] || fail 'the correction PR delivery check runs only on a delivery-correction/<sha> branch'
    [[ "$head_sha" =~ ^[0-9a-f]{40}$ ]] || fail 'pr-head requires a full --sha'
    gh api "repos/$repository/pulls?state=open&head=${repository%%/*}:$branch&per_page=100" >"$temporary/prs" \
      || fail 'cannot read the open correction PR'
    jq -e --arg bot "$bot_login" --arg repo "$repository" --arg branch "$branch" --arg sha "$head_sha" '
      length == 1 and (.[0] | .user.login == $bot and .head.repo.full_name == $repo and .head.ref == $branch
        and .head.sha == $sha and .base.ref == "main")' "$temporary/prs" >/dev/null \
      || fail "$head_sha is not the head of one open bot-authored correction PR from $branch"
    number=$(jq -r '.[0].number' "$temporary/prs")
    gh api "repos/$repository/commits/$head_sha" >"$temporary/head" || fail 'cannot read the correction PR head commit'
    parent=$(jq -r 'if (.parents | length) == 1 then .parents[0].sha else empty end' "$temporary/head")
    [[ "$parent" =~ ^[0-9a-f]{40}$ ]] || fail 'the correction PR head is not one commit'
    gh api "repos/$repository/compare/main...$parent" >"$temporary/compare" || fail 'cannot compare the correction PR base with main'
    jq -e '.status == "identical" or .status == "behind"' "$temporary/compare" >/dev/null \
      || fail 'the correction PR is not built on main history'
    jq -e --arg path "$corrections_path" '.files | length == 1 and .[0].filename == $path and .[0].status == "modified"
      and .[0].deletions == 0 and .[0].additions > 0' "$temporary/head" >/dev/null \
      || fail "the correction PR head must only append lines to $corrections_path"
    jq -r '.[0].title // ""' "$temporary/prs" >"$temporary/title"
    jq -r '.[0].body // ""' "$temporary/prs" >"$temporary/body"
    result=$("$repository_root/scripts/delivery-issues.sh" check-pr --title "$temporary/title" --body "$temporary/body") || exit 1
    grep -Fxq 'related=none' <<<"$result" && grep -Fxq 'completes=none' <<<"$result" \
      || fail 'a correction PR declares no Issue: Related-Issues: none and Completes-Issues: none'
    linked=$(closing_refs "$number") || fail 'cannot read the Issues the correction PR would close'
    [[ -z "$linked" ]] || fail "the correction PR would close $linked at merge; remove the closing keyword or Development link"
    printf 'correction_pr=#%s head=%s\ndelivery_metadata=pass\n' "$number" "$head_sha"
    ;;
  *)
    fail 'usage: delivery-correction.sh plan|propose|pr-head [options]'
    ;;
esac
