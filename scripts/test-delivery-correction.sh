#!/usr/bin/env bash
# Contract tests for delivery-correction.sh and delivery-correction.yml: a
# squash commit that lost the metadata its PR declared gets exactly one
# proposed correction PR with the validated values, and the delivery-metadata
# dispatch accepts only that bot-authored correction head. A local fixture
# repository and a stateful fake `gh`; no network, no GitHub.
set -euo pipefail

umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
temporary=$(cd "$temporary" && pwd -P)
trap 'rm -rf -- "$temporary"' EXIT

failures=0
check() {
  local label=$1
  shift
  if "$@" >/dev/null 2>&1; then
    printf 'ok: %s\n' "$label"
  else
    printf 'FAIL: %s\n' "$label" >&2
    failures=$((failures + 1))
  fi
}
expect_failure() {
  local label=$1 pattern=$2
  shift 2
  if "$@" >"$temporary/out" 2>"$temporary/err"; then
    printf 'FAIL: %s unexpectedly succeeded\n' "$label" >&2
    failures=$((failures + 1))
    return
  fi
  if grep -Fq -- "$pattern" "$temporary/err"; then
    printf 'ok: %s\n' "$label"
  else
    printf 'FAIL: %s did not report "%s":\n' "$label" "$pattern" >&2
    cat "$temporary/err" >&2
    failures=$((failures + 1))
  fi
}

# --- Fake GitHub -----------------------------------------------------------------
tools=$temporary/tools
state=$temporary/github
mkdir -p "$tools" "$state"
export FAKE_GH_STATE=$state FAKE_GH_FIXTURE=$temporary/fixture
export PATH="$tools:$PATH"
cat >"$tools/gh" <<'EOF'
#!/usr/bin/env bash
# Stateful fake of the gh subset used by delivery-correction.sh.
set -euo pipefail
s=$FAKE_GH_STATE
repo=rgomids/axiom
log() { printf '%s\n' "$*" >>"$s/ledger"; }
not_found() { printf 'gh: Not Found (HTTP 404)\n' >&2; exit 1; }
case "${1:-}" in
  workflow) log "workflow $*"; exit 0 ;;
  pr)
    [[ "$2" == create ]] || exit 2
    shift 2
    head= title= body=
    while (($#)); do
      case "$1" in
        --head) head=$2; shift 2 ;;
        --title) title=$2; shift 2 ;;
        --body-file) body=$2; shift 2 ;;
        *) shift ;;
      esac
    done
    log "PR create $head $title"
    cp "$body" "$s/pr-body"
    printf 'https://github.com/%s/pull/900\n' "$repo"
    exit 0 ;;
  api) shift ;;
  *) printf 'fake gh: unsupported %s\n' "$*" >&2; exit 2 ;;
esac
method=GET endpoint= fields=()
while (($#)); do
  case "$1" in
    --method|-X) method=$2; shift 2 ;;
    -f|-F) fields+=("$2"); shift 2 ;;
    *) endpoint=$1; shift ;;
  esac
done
var() { local f; for f in "${fields[@]}"; do [[ "$f" == "$1="* ]] && { printf '%s' "${f#*=}"; return; }; done; }
if [[ "$endpoint" == graphql ]]; then
  jq -n --argjson n "$(cat "$s/closing-refs" 2>/dev/null || printf '[]')" \
    '{data: {repository: {pullRequest: {closingIssuesReferences: {nodes: ($n | map({number: .}))}}}}}'
  exit 0
fi
path=${endpoint#repos/$repo/}
case "$method $path" in
  "GET commits/"*"/pulls") sha=${path#commits/}; sha=${sha%%/*}; cat "$s/pulls-$sha.json" 2>/dev/null || printf '[]\n' ;;
  "GET commits/"*"/check-runs?per_page=100") sha=${path#commits/}; sha=${sha%%/*}; cat "$s/checks-$sha.json" 2>/dev/null || printf '{"check_runs":[]}\n' ;;
  "GET commits/"*) [[ -f "$s/commit-${path#commits/}.json" ]] || not_found; cat "$s/commit-${path#commits/}.json" ;;
  "GET compare/main..."*) cat "$s/compare.json" 2>/dev/null || printf '{"status":"behind"}\n' ;;
  "GET contents/.github/delivery-corrections.txt?ref="*)
    ref=${path##*ref=}
    content=$(git -C "$FAKE_GH_FIXTURE" show "$ref:.github/delivery-corrections.txt" | base64 | tr -d '\n')
    jq -n --arg c "$content" '{sha: "blob0", content: $c}' ;;
  "GET git/ref/heads/"*) [[ -f "$s/branch-${path#git/ref/heads/delivery-correction/}" ]] || not_found; printf '{}\n' ;;
  "POST git/refs")
    ref=$(var ref)
    log "POST ref ${ref#refs/heads/} $(var sha)"
    : >"$s/branch-${ref#refs/heads/delivery-correction/}"
    printf '{}\n' ;;
  "PUT contents/.github/delivery-corrections.txt")
    log "PUT corrections $(var branch) $(var message)"
    printf '%s' "$(var content)" | base64 --decode >"$s/committed-corrections"
    printf '{}\n' ;;
  "GET pulls?"*) cat "$s/open-pulls.json" 2>/dev/null || printf '[]\n' ;;
  *) printf 'fake gh: unsupported api %s %s\n' "$method" "$path" >&2; exit 2 ;;
esac
EOF
chmod 700 "$tools/gh"

# --- Fixture: a release commit, then squash commits on main ---------------------
fixture=$temporary/fixture
git init -q -b main "$fixture"
git -C "$fixture" config user.email delivery-test@example.invalid
git -C "$fixture" config user.name 'Delivery Test'
git -C "$fixture" config commit.gpgsign false
mkdir -p "$fixture/scripts" "$fixture/.github"
cp "$repository_root/scripts/delivery-issues.sh" "$repository_root/scripts/delivery-correction.sh" "$fixture/scripts/"
printf '# Reviewed delivery metadata.\n' >"$fixture/.github/delivery-corrections.txt"
commit() {
  git -C "$fixture" add -A
  printf '%b' "$1" | git -C "$fixture" commit -q --allow-empty -F -
  git -C "$fixture" rev-parse HEAD
}
printf '{".":"0.4.0"}\n' >"$fixture/.release-please-manifest.json"
commit 'chore(main): release 0.4.0 (#10)\n' >/dev/null
declared=$(commit 'feat(cli): declared (#11)\n\nRelated-Issues: #5\nCompletes-Issues: #5\n')
lost=$(commit 'feat(cli): lost metadata (#12)\n\nA custom squash summary.\n')
head=$(git -C "$fixture" rev-parse HEAD)
pr_head=$(printf 'c%.0s' {1..40})
dc() { "$fixture/scripts/delivery-correction.sh" "$@"; }
stage_pr() {
  jq -n --arg sha "$lost" --arg head "$pr_head" --arg body "${1:-$'## Delivered Issues\n\nRelated-Issues: #7, #8\nCompletes-Issues: #7\n'}" \
    '[{number: 12, merged_at: "2026-10-25T00:00:00Z", merge_commit_sha: $sha, base: {ref: "main"},
       head: {sha: $head}, title: "feat(cli): lost metadata", body: $body}]' >"$state/pulls-$lost.json"
  printf '{"check_runs":[{"id":2,"name":"delivery-metadata","status":"completed","conclusion":"%s"},{"id":1,"name":"PR Checks","status":"completed","conclusion":"success"}]}\n' \
    "${2:-success}" >"$state/checks-$pr_head.json"
}

# --- 1. Plan ---------------------------------------------------------------------------
stage_pr
dc plan --repo rgomids/axiom --to "$head" >"$temporary/plan"
check 'plan proposes only the undeclared commit, with the validated PR values' bash -c "
  grep -Fxq 'correction.1=$lost pr=12 head=$pr_head related=7,8 completes=7' '$temporary/plan' &&
  grep -Fxq corrections=1 '$temporary/plan' && ! grep -Fq '$declared' '$temporary/plan'"
check 'plan has no remote effect' test ! -s "$state/ledger"
stage_pr $'A summary without delivery metadata.\n' success
expect_failure 'a PR description without metadata is never guessed' 'no longer validates' dc plan --repo rgomids/axiom --to "$head"
stage_pr $'Closes #7\n\nRelated-Issues: #7\nCompletes-Issues: #7\n' success
expect_failure 'a description with a closing keyword is refused' 'no longer validates' dc plan --repo rgomids/axiom --to "$head"
stage_pr $'Related-Issues: #7\nCompletes-Issues: #7\n' failure
expect_failure 'a failed delivery-metadata check at the PR head fails closed' 'delivery-metadata check of PR #12 is failure' dc plan --repo rgomids/axiom --to "$head"
rm "$state/checks-$pr_head.json"
expect_failure 'a missing delivery-metadata check fails closed' 'is missing at its head' dc plan --repo rgomids/axiom --to "$head"
printf '[]\n' >"$state/pulls-$lost.json"
expect_failure 'a commit without its merged PR fails closed' 'is not the merge commit of exactly one merged PR' dc plan --repo rgomids/axiom --to "$head"
stage_pr
printf '\n# reviewed by a human\n%s related=7,8 completes=7\n' "$lost" >>"$fixture/.github/delivery-corrections.txt"
corrected=$(commit 'fix(delivery): restore squash metadata of PR #12 (#13)\n\nRelated-Issues: none\nCompletes-Issues: none\n')
dc plan --repo rgomids/axiom --to "$corrected" >"$temporary/plan"
check 'a commit with a reviewed correction is not proposed again' grep -Fxq corrections=0 "$temporary/plan"
expect_failure 'plan requires a full revision' 'requires a full --to' dc plan --repo rgomids/axiom --to HEAD

# --- 2. Propose ------------------------------------------------------------------------
dc propose --repo rgomids/axiom --to "$head" >"$temporary/propose"
branch=delivery-correction/$lost
check 'propose creates the branch at the pushed main and commits only the appended row' bash -c "
  grep -Fxq 'POST ref $branch $head' '$state/ledger' &&
  grep -Fxq 'PUT corrections $branch fix(delivery): restore squash metadata of PR #12' '$state/ledger' &&
  [[ \$(tail -n 1 '$state/committed-corrections') == '$lost related=7,8 completes=7' ]] &&
  head -n 1 '$state/committed-corrections' | grep -Fxq '# Reviewed delivery metadata.' &&
  grep -Fq '# PR #12 declared Related-Issues: 7,8 and Completes-Issues: 7; its' '$state/committed-corrections'"
check 'propose opens one correction PR that declares no Issue and dispatches its required checks' bash -c "
  grep -Fxq 'PR create $branch fix(delivery): restore squash metadata of PR #12' '$state/ledger' &&
  grep -Fxq 'Related-Issues: none' '$state/pr-body' && grep -Fxq 'Completes-Issues: none' '$state/pr-body' &&
  grep -Fq 'workflow workflow run ci.yml --repo rgomids/axiom --ref $branch' '$state/ledger' &&
  grep -Fq 'workflow workflow run delivery-metadata.yml --repo rgomids/axiom --ref $branch' '$state/ledger' &&
  grep -Fxq 'effect=pr_opened branch=$branch url=https://github.com/rgomids/axiom/pull/900' '$temporary/propose'"
printf '%s\n' "fix(delivery): restore squash metadata of PR #12" >"$temporary/title"
check 'the proposed PR passes the delivery-metadata validator' "$repository_root/scripts/delivery-issues.sh" check-pr --title "$temporary/title" --body "$state/pr-body"
: >"$state/ledger"
dc propose --repo rgomids/axiom --to "$head" >"$temporary/propose"
check 'propose is idempotent: an existing correction branch is left alone' bash -c "
  grep -Fxq 'effect=none branch=$branch reason=already_proposed' '$temporary/propose' && ! grep -Eq '^(POST|PUT|PR|workflow)' '$state/ledger'"

# --- 3. Dispatch path of the delivery-metadata check -----------------------------------
correction_head=$(printf 'd%.0s' {1..40})
stage_head() {
  jq -n --arg sha "$correction_head" --arg branch "$branch" --arg body "$(cat "$state/pr-body")" \
    '[{number: 900, user: {login: "github-actions[bot]"}, base: {ref: "main"}, title: "fix(delivery): restore squash metadata of PR #12",
       body: $body, head: {ref: $branch, sha: $sha, repo: {full_name: "rgomids/axiom"}}}]' >"$state/open-pulls.json"
  jq -n --arg parent "$head" '{parents: [{sha: $parent}],
    files: [{filename: ".github/delivery-corrections.txt", status: "modified", additions: 5, deletions: 0}]}' >"$state/commit-$correction_head.json"
}
ph() { dc pr-head --repo rgomids/axiom "$@"; }
stage_head
ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]' >"$temporary/ph"
check 'the bot-dispatched check passes for the correction PR head' bash -c "
  grep -Fxq 'correction_pr=#900 head=$correction_head' '$temporary/ph' && grep -Fxq delivery_metadata=pass '$temporary/ph'"
expect_failure 'a human dispatch never produces a passing check' 'only github-actions[bot] dispatches' \
  ph --branch "$branch" --sha "$correction_head" --actor rgomids
expect_failure 'a dispatch on another branch fails' 'runs only on a delivery-correction/<sha> branch' \
  ph --branch feature/x --sha "$correction_head" --actor 'github-actions[bot]'
expect_failure 'a dispatch for another commit than the PR head fails' 'is not the head of one open bot-authored correction PR' \
  ph --branch "$branch" --sha "$(printf 'e%.0s' {1..40})" --actor 'github-actions[bot]'
jq '.[0].user.login = "someone"' "$state/open-pulls.json" >"$state/p" && mv "$state/p" "$state/open-pulls.json"
expect_failure 'a correction branch PR not authored by the bot fails' 'is not the head of one open bot-authored correction PR' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
stage_head
jq '.[0].head.repo.full_name = "fork/axiom"' "$state/open-pulls.json" >"$state/p" && mv "$state/p" "$state/open-pulls.json"
expect_failure 'a fork head with the same branch name fails' 'is not the head of one open bot-authored correction PR' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
stage_head
jq '.files += [{filename: "scripts/release.sh", status: "modified", additions: 1, deletions: 0}]' "$state/commit-$correction_head.json" >"$state/c" \
  && mv "$state/c" "$state/commit-$correction_head.json"
expect_failure 'a head that changes another file fails' 'must only append lines to .github/delivery-corrections.txt' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
stage_head
jq '.files[0].deletions = 1' "$state/commit-$correction_head.json" >"$state/c" && mv "$state/c" "$state/commit-$correction_head.json"
expect_failure 'a head that removes a reviewed row fails' 'must only append lines' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
stage_head
jq '.parents += [{sha: "x"}]' "$state/commit-$correction_head.json" >"$state/c" && mv "$state/c" "$state/commit-$correction_head.json"
expect_failure 'a head that is not one commit fails' 'is not one commit' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
stage_head
printf '{"status":"diverged"}\n' >"$state/compare.json"
expect_failure 'a head not built on main history fails' 'is not built on main history' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
rm "$state/compare.json"
jq '.[0].body = "Related-Issues: #7\nCompletes-Issues: #7\n"' "$state/open-pulls.json" >"$state/p" && mv "$state/p" "$state/open-pulls.json"
expect_failure 'a correction PR that declares an Issue fails' 'declares no Issue' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
stage_head
printf '[7]\n' >"$state/closing-refs"
expect_failure 'a correction PR that would close an Issue at merge fails' 'would close #7 at merge' \
  ph --branch "$branch" --sha "$correction_head" --actor 'github-actions[bot]'
rm "$state/closing-refs"

# --- 4. Workflow shape -----------------------------------------------------------------
workflows=$repository_root/.github/workflows
w=$workflows/delivery-correction.yml
check 'delivery-correction.yml runs trusted main code on push only, with job-scoped write permissions and no secret' bash -c "
  [[ \$(sed -n '/^on:/,/^[a-z]/p' '$w' | grep -E '^  [a-z_]+:' | tr -d ' :') == push ]] &&
  sed -n '/^  push:/,/^[a-z]/p' '$w' | grep -Fxq '      - main' &&
  grep -Fxq 'permissions: {}' '$w' && ! grep -Ev '^ *#' '$w' | grep -Eq 'secrets\\.|pull_request|github\\.event\\.(pull_request|head_commit|commits)' &&
  grep -Fxq '      contents: write' '$w' && grep -Fxq '      pull-requests: write' '$w' && grep -Fxq '      actions: write' '$w' &&
  grep -Fq './scripts/delivery-correction.sh propose --repo \"\$GITHUB_REPOSITORY\" --to \"\$PUSH_AFTER\"' '$w'"
check 'delivery-metadata.yml routes only delivery-correction/ dispatches to the correction head check' bash -c "
  grep -Fq 'if [[ \"\$DISPATCH_BRANCH\" == delivery-correction/* ]]; then' '$workflows/delivery-metadata.yml' &&
  grep -Fq './scripts/delivery-correction.sh pr-head' '$workflows/delivery-metadata.yml' &&
  grep -Fq './scripts/delivery-github.sh release-pr-head' '$workflows/delivery-metadata.yml'"

if ((failures > 0)); then
  printf 'FAIL: %s delivery correction check(s) failed\n' "$failures" >&2
  exit 1
fi
printf 'PASS: delivery correction plan, proposal and correction PR dispatch path\n'
