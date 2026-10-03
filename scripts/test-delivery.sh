#!/usr/bin/env bash
# Delivery metadata contract tests for delivery-issues.sh: the explicit
# Related-Issues/Completes-Issues grammar, the PR check that refuses GitHub
# closing keywords, merge-time commit lines and the Issue set of a stable
# release resolved from Git history. Local only: a fixture repository, no
# network, no GitHub. The GitHub effects of delivery-github.sh are covered by
# test-release-flow.sh with its fake GitHub.
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

issues=$repository_root/scripts/delivery-issues.sh
# parse TEXT prints the normalized result of one message.
parse() { printf '%b' "$1" | "$issues" parse | paste -sd' ' -; }
parse_is() {
  local label=$1 text=$2 expected=$3 actual
  actual=$(parse "$text" 2>&1) || true
  if [[ "$actual" == "$expected" ]]; then
    printf 'ok: %s\n' "$label"
  else
    printf 'FAIL: %s: got "%s", expected "%s"\n' "$label" "$actual" "$expected" >&2
    failures=$((failures + 1))
  fi
}
parse_fails() { expect_failure "$1" "$3" bash -c "printf '%b' \"\$1\" | '$issues' parse" _ "$2"; }

# --- 1. Grammar --------------------------------------------------------------------
parse_is 'one related Issue, partial PR' 'feat: x\n\nRelated-Issues: #132\nCompletes-Issues: none\n' \
  'metadata=declared related=132 completes=none'
parse_is 'several related Issues are sorted and normalized' 'Related-Issues: #140, #132,#7\nCompletes-Issues: none\n' \
  'metadata=declared related=7,132,140 completes=none'
parse_is 'completing PR' 'Related-Issues: #132\nCompletes-Issues: #132\n' \
  'metadata=declared related=132 completes=132'
parse_is 'completing several Issues' 'Related-Issues: #5, #9, #12\nCompletes-Issues: #12, #5\n' \
  'metadata=declared related=5,9,12 completes=5,12'
parse_is 'related but not completed' 'Related-Issues: #5, #9\nCompletes-Issues: #9\n' \
  'metadata=declared related=5,9 completes=9'
parse_is 'none and none' 'Related-Issues: none\nCompletes-Issues: none\n' \
  'metadata=declared related=none completes=none'
parse_is 'no metadata is undeclared, never completion' 'feat: x\n\nmentions #132 and Refs #140\n' \
  'metadata=undeclared related=none completes=none'
parse_is 'order of the keys does not matter' 'Completes-Issues: none\nRelated-Issues: #3\n' \
  'metadata=declared related=3 completes=none'
parse_is 'CRLF and trailing spaces are tolerated' 'Related-Issues: #2  \r\nCompletes-Issues: none\r\n' \
  'metadata=declared related=2 completes=none'
parse_is 'examples inside fenced code are ignored' '```\nRelated-Issues: #1\nCompletes-Issues: #1\n```\nRelated-Issues: #2\nCompletes-Issues: #2\n' \
  'metadata=declared related=2 completes=2'
parse_is 'squash body with co-author trailers' '## Summary\n\nText.\n\nRelated-Issues: #132\nCompletes-Issues: none\n\n---------\n\nCo-authored-by: A <a@example.invalid>\n' \
  'metadata=declared related=132 completes=none'

parse_is 'metadata hidden in an HTML comment is ignored' '<!--\nRelated-Issues: #5\nCompletes-Issues: #5\n-->\n' \
  'metadata=undeclared related=none completes=none'
parse_is 'visible metadata wins over commented metadata' '<!-- template hint -->\n<!--\nCompletes-Issues: #5\n-->\nRelated-Issues: #6\nCompletes-Issues: none\n' \
  'metadata=declared related=6 completes=none'
parse_fails 'completed Issue must be related' 'Related-Issues: #132\nCompletes-Issues: #133\n' 'must also be listed in Related-Issues'
parse_fails 'completed Issue with no related Issues' 'Related-Issues: none\nCompletes-Issues: #133\n' 'must also be listed in Related-Issues'
parse_fails 'duplicated related Issue' 'Related-Issues: #132, #132\nCompletes-Issues: none\n' 'more than once'
parse_fails 'duplicated completed Issue' 'Related-Issues: #132\nCompletes-Issues: #132,#132\n' 'more than once'
parse_fails 'key repeated' 'Related-Issues: #2\nRelated-Issues: #2\nCompletes-Issues: #2\n' 'appears more than once'
parse_fails 'related without completes' 'Related-Issues: #132\n' 'requires Completes-Issues'
parse_fails 'completes without related' 'Completes-Issues: none\n' 'requires Related-Issues'
parse_fails 'number without #' 'Related-Issues: 132\nCompletes-Issues: none\n' "must be 'none' or a list"
parse_fails 'cross-repository reference' 'Related-Issues: owner/repo#1\nCompletes-Issues: none\n' "must be 'none' or a list"
parse_fails 'URL reference' 'Related-Issues: https://github.com/rgomids/axiom/issues/1\nCompletes-Issues: none\n' "must be 'none' or a list"
parse_fails 'zero is not an Issue' 'Related-Issues: #0\nCompletes-Issues: none\n' "must be 'none' or a list"
parse_fails 'empty value' 'Related-Issues:\nCompletes-Issues: none\n' "must be 'none' or a list"
parse_fails 'placeholder left from the template' 'Related-Issues: #<issue> or none\nCompletes-Issues: #<issue> or none\n' "must be 'none' or a list"
parse_fails 'None is case sensitive' 'Related-Issues: None\nCompletes-Issues: none\n' "must be 'none' or a list"
parse_fails 'space separated list' 'Related-Issues: #1 #2\nCompletes-Issues: none\n' "must be 'none' or a list"
parse_fails 'lowercase key' 'related-issues: #132\nCompletes-Issues: none\n' 'malformed delivery metadata line'
parse_fails 'singular key' 'Related-Issues: #1\nCompletes-Issue: #1\n' 'malformed delivery metadata line'
parse_fails 'decorated key' 'Related-Issues: #1\n**Completes-Issues:** #1\n' 'malformed delivery metadata line'
parse_fails 'indented key in a list' 'Related-Issues: #1\n Completes-Issues: none\n' 'malformed delivery metadata line'
parse_fails 'more than twenty Issues' "Related-Issues: $(seq 1 21 | sed 's/^/#/' | paste -sd, -)\nCompletes-Issues: none\n" 'more than 20'
[[ ! -e "$temporary/pwned" ]]
parse_fails 'shell syntax is data, never executed' "Related-Issues: \$(touch $temporary/pwned)\nCompletes-Issues: none\n" "must be 'none' or a list"
check 'no command from metadata ran' test ! -e "$temporary/pwned"

# --- 2. Pull Request check -------------------------------------------------------------
pr_check() {
  printf '%b' "$1" >"$temporary/title"
  printf '%b' "$2" >"$temporary/body"
  "$issues" check-pr --title "$temporary/title" --body "$temporary/body"
}
check 'a complete PR description passes' pr_check 'feat(project): add create' '## Context\n\nx\n\nRelated-Issues: #132\nCompletes-Issues: #132\n'
pr_check 'feat: x' 'Related-Issues: #132\nCompletes-Issues: none\n' >"$temporary/pr"
check 'PR check reports the normalized relation' grep -Fxq completes=none "$temporary/pr"
expect_failure 'PR without metadata' 'must declare' pr_check 'feat: x' '## Context\n\nmentions #132\n'
expect_failure 'PR with malformed metadata' "must be 'none' or a list" pr_check 'feat: x' 'Related-Issues: 132\nCompletes-Issues: none\n'
for keyword in 'Closes #132' 'closes #132' 'Fixes #1' 'fixed #1' 'Resolves #9' 'resolved: #9' 'CLOSE #4' \
  'Fixes rgomids/axiom#12' 'closes https://github.com/rgomids/axiom/issues/12' 'see it; fixes #3.'; do
  expect_failure "closing keyword '$keyword' in the body" 'GitHub closing keyword' \
    pr_check 'feat: x' "Related-Issues: #1\nCompletes-Issues: #1\n\n$keyword\n"
done
for keyword in 'Closes:#5' 'Closes\t#5' 'Closes **#5**' 'Closes [#5](https://github.com/rgomids/axiom/issues/5)'; do
  expect_failure "closing keyword variant '$keyword'" 'GitHub closing keyword' \
    pr_check 'feat: x' "Related-Issues: #5\nCompletes-Issues: #5\n\n$keyword\n"
done
expect_failure 'a key line the squash wrap would split' 'longer than 72 characters' \
  pr_check 'feat: x' "Related-Issues: $(seq 101 112 | sed 's/^/#/' | paste -sd, - | sed 's/,/, /g')\nCompletes-Issues: none\n"
expect_failure 'prose that the squash wrap turns into a key line' 'squash wrap' \
  pr_check 'feat: x' 'Related-Issues: #5\nCompletes-Issues: none\n\nThis long explanatory sentence is written so that GitHub wraps it at Completes-Issues: #5 here.\n'
expect_failure 'closing keyword in the title (squash subject)' 'GitHub closing keyword' \
  pr_check 'fix: thing, closes #4' 'Related-Issues: #4\nCompletes-Issues: #4\n'
expect_failure 'closing keyword inside code is refused too' 'GitHub closing keyword' \
  pr_check 'feat: x' 'Related-Issues: #1\nCompletes-Issues: #1\n\n```\nCloses #1\n```\n'
check 'words containing keywords are not keywords' pr_check 'feat: prefixes and closeness' \
  'Related-Issues: #1\nCompletes-Issues: none\n\nThe prefix #1 and enclosed #2 and unresolved #3 stay open.\n'

# PR titles become squash subjects and must be parseable by Release Please.
for title in 'feat: add page' 'feat(site): add page' 'fix!: reject old state' 'security(fs)!: tighten boundary' 'chore(main): release 0.3.0' 'revert: restore page'; do
  check "conventional title '$title' passes" pr_check "$title" 'Related-Issues: none\nCompletes-Issues: none\n'
done
for title in 'Docs/86 added landing page' 'Feat(site): add page' 'feature: add page' 'feat: ' 'feat(site) add page' 'feat(): add page' 'feat: add page\nfix: another subject'; do
  expect_failure "nonconventional title '$title' refused" 'Conventional Commits' \
    pr_check "$title" 'Related-Issues: none\nCompletes-Issues: none\n'
done

# --- 3. Fixture history: partial and completing PRs across releases ---------------------
fixture=$temporary/fixture
git init -q -b main "$fixture"
git -C "$fixture" config user.email delivery-test@example.invalid
git -C "$fixture" config user.name 'Delivery Test'
git -C "$fixture" config commit.gpgsign false
mkdir -p "$fixture/scripts" "$fixture/.github"
cp "$repository_root/scripts/delivery-issues.sh" "$repository_root/scripts/release-tag-version.sh" "$fixture/scripts/"
# commit SUBJECT BODY [MANIFEST_VERSION] commits a squash-shaped change.
commit() {
  [[ -z "${3:-}" ]] || printf '{\n  ".": "%s"\n}\n' "$3" >"$fixture/.release-please-manifest.json"
  git -C "$fixture" add -A
  printf '%s\n\n%b' "$1" "$2" | git -C "$fixture" commit -q --allow-empty -F -
  git -C "$fixture" rev-parse HEAD
}
d=$fixture/scripts/delivery-issues.sh
base=$(commit 'chore: base' 'History before the delivery contract.\n')
adopt=$(commit 'ci(release): adopt release please (#1)' 'Closes #99 in the old style.\n' 0.0.0)
t01=$(commit 'feat(project): EDIT preview (I132-T01) (#145)' 'Related-Issues: #132\nCompletes-Issues: none\n')
other=$(commit 'fix(cli): repair help (#150)' 'Related-Issues: #140, #141\nCompletes-Issues: #140\n')
t02=$(commit 'feat(project): CREATE (I132-T02) (#151)' 'Related-Issues: #132\nCompletes-Issues: none\n')
t03=$(commit 'feat(project): finish create/edit (I132-T03) (#152)' 'Related-Issues: #132, #141\nCompletes-Issues: #132, #141\n')
again=$(commit 'test(project): extra regression for #140 (#153)' 'Related-Issues: #140\nCompletes-Issues: #140\n')
docs=$(commit 'docs: tidy (#160)' 'Context only.\n\nRelated-Issues: #133\nCompletes-Issues: none\n')
r1=$(commit 'chore(main): release 0.2.0 (#154)' 'Release PR body without metadata.\n' 0.2.0)
after=$(commit 'feat: next (#155)' 'Related-Issues: #161\nCompletes-Issues: #161\n')
reopened=$(commit 'fix: regression of #132 (#156)' 'Related-Issues: #132\nCompletes-Issues: #132\n')
r2=$(commit 'chore(main): release 0.2.1 (#157)' '' 0.2.1)

"$d" commits --from "$adopt" --to "$t03" >"$temporary/commits"
check 'merge-time lines cover exactly FROM..TO in order' test "$(awk '{print $1}' "$temporary/commits" | paste -sd' ' -)" == "commit=$t01 commit=$other commit=$t02 commit=$t03"
check 'partial PR line: related only, no completion' grep -Fxq "commit=$t01 pr=145 release=none metadata=declared related=132 completes=none" "$temporary/commits"
check 'completing PR line names its PR and Issues' grep -Fxq "commit=$t03 pr=152 release=none metadata=declared related=132,141 completes=132,141" "$temporary/commits"
"$d" commits --from "$docs" --to "$r1" >"$temporary/commits"
check 'a release commit is recognized from the manifest' grep -Fxq "commit=$r1 pr=154 release=0.2.0 metadata=undeclared related=none completes=none" "$temporary/commits"
"$d" commits --from "$base" --to "$adopt" >"$temporary/commits"
check 'adopting the manifest is not a release' grep -Fq ' release=none ' "$temporary/commits"
expect_failure 'FROM must be an ancestor of TO' 'not an ancestor' "$d" commits --from "$r1" --to "$t01"
expect_failure 'merge-time range needs full revisions' 'full 40-character revision' "$d" commits --from "${t01:0:12}" --to "$r1"

# --- 4. Release Issue set -------------------------------------------------------------------
"$d" release --tag v0.2.0 --revision "$r1" >"$temporary/r1"
check 'release range starts after the previous manifest change' grep -Fxq "range_base=$adopt" "$temporary/r1"
check 'release Issue set: completed Issues only, deduplicated, sorted' grep -Fxq issues=132,140,141 "$temporary/r1"
check 'related-only Issues are not delivered' bash -c "! grep -Eq '(^issues=|,)133(,|$)|^issue\\.133=' '$temporary/r1'"
check 'an Issue completed by several PRs is listed once with each PR' grep -Fxq 'issue.140=#150,#153' "$temporary/r1"
check 'partial PRs do not deliver the parent Issue: only the completing PR is named' grep -Fxq 'issue.132=#152' "$temporary/r1"
check 'commits and undeclared commits are counted' bash -c "grep -Fxq commits=7 '$temporary/r1' && grep -Fxq undeclared_commits=1 '$temporary/r1'"
"$d" release --tag v0.2.1 --revision "$r2" >"$temporary/r2"
check 'next release range is disjoint: starts at the previous release commit' grep -Fxq "range_base=$r1" "$temporary/r2"
check 'an Issue appears only in the release whose commits complete it' grep -Fxq issues=132,161 "$temporary/r2"
check 'a reopened Issue completed again belongs to the later release by its new PR' grep -Fxq 'issue.132=#156' "$temporary/r2"
"$d" release --tag v0.2.0-rc.1 --revision "$t03" >"$temporary/rc"
check 'a release candidate delivers no Issue' grep -Fxq issues=not_applicable "$temporary/rc"
expect_failure 'stable Issues resolve only at the release commit' 'does not record version 0.2.0' "$d" release --tag v0.2.0 --revision "$t03"
expect_failure 'a later commit is not the release commit' 'not the release commit' "$d" release --tag v0.2.0 --revision "$after"
expect_failure 'invalid tag' 'release_tag_error' "$d" release --tag 0.2.0 --revision "$r1"
cmp -s "$temporary/r1" <("$d" release --tag v0.2.0 --revision "$r1") && check 'release Issue set is deterministic' true || check 'release Issue set is deterministic' false

# A malformed commit inside a release range fails closed; a reviewed
# correction committed before the release commit resolves it
# deterministically. Corrections are read at the resolved revision only.
bad=$(commit 'feat: bad metadata edited at merge (#158)' 'Related-Issues: 170\nCompletes-Issues: #170\n')
r3=$(commit 'chore(main): release 0.3.0 (#159)' '' 0.3.0)
expect_failure 'malformed metadata in a release range fails closed' "commit $bad has malformed delivery metadata" "$d" release --tag v0.3.0 --revision "$r3"
printf '# reviewed correction for an immutable commit\n%s related=170 completes=170\n' "$bad" >"$fixture/.github/delivery-corrections.txt"
expect_failure 'an uncommitted correction has no effect' "commit $bad has malformed delivery metadata" "$d" release --tag v0.3.0 --revision "$r3"
git -C "$fixture" reset -q --hard "$bad"
printf '# reviewed correction for an immutable commit\n%s related=170 completes=170\n' "$bad" >"$fixture/.github/delivery-corrections.txt"
fixcommit=$(commit 'docs(delivery): record reviewed correction (#160)' 'Related-Issues: none\nCompletes-Issues: none\n')
r3=$(commit 'chore(main): release 0.3.0 (#161)' '' 0.3.0)
"$d" release --tag v0.3.0 --revision "$r3" >"$temporary/r3"
check 'a committed correction replaces the malformed metadata' bash -c "grep -Fxq issues=170 '$temporary/r3' && grep -Fxq 'issue.170=#158' '$temporary/r3'"
git -C "$fixture" reset -q --hard "$bad"
mkdir -p "$fixture/.github"
printf '%s related=#170 completes=170\n' "$bad" >"$fixture/.github/delivery-corrections.txt"
commit 'docs(delivery): malformed correction (#162)' 'Related-Issues: none\nCompletes-Issues: none\n' >/dev/null
r3=$(commit 'chore(main): release 0.3.0 (#163)' '' 0.3.0)
expect_failure 'a malformed correction fails closed' 'malformed line in .github/delivery-corrections.txt' "$d" release --tag v0.3.0 --revision "$r3"

# --- 4b. v0.2.0 migration boundary ----------------------------------------------------------
# Legacy commits closed Issues with keywords. Only the reviewed corrections
# committed before the release commit deliver an Issue; other keywords and
# partial work stay undeclared, and a corrected Issue is listed once.
lfix=$temporary/legacy-repo
git init -q -b main "$lfix"
git -C "$lfix" config user.email delivery-test@example.invalid
git -C "$lfix" config user.name 'Delivery Test'
git -C "$lfix" config commit.gpgsign false
mkdir -p "$lfix/scripts" "$lfix/.github"
cp "$repository_root/scripts/delivery-issues.sh" "$repository_root/scripts/release-tag-version.sh" "$lfix/scripts/"
lcommit() {
  [[ -z "${3:-}" ]] || printf '{\n  ".": "%s"\n}\n' "$3" >"$lfix/.release-please-manifest.json"
  git -C "$lfix" add -A
  printf '%s\n\n%b' "$1" "$2" | git -C "$lfix" commit -q --allow-empty -F -
  git -C "$lfix" rev-parse HEAD
}
l0=$(lcommit 'chore(main): release 0.1.2 (#121)' '' 0.1.2)
lcommit 'feat(project): EDIT preview (I132-T01) (#145)' 'Partial work for #132; Closes nothing yet.\n' >/dev/null
l143=$(lcommit 'feat(project): list configured projects (#143)' 'Implements issue #129\n\nCloses #129\n')
lcommit 'chore: unrelated legacy (#150)' 'Closes #200\nFixes #132\n' >/dev/null
l148=$(lcommit 'fix(work-items): resolve portable Project from recorded SourceLocation (#147) (#148)' '- Closes #147.\n')
"$lfix/scripts/delivery-issues.sh" commits --from "$l0" --to "$l148" >"$temporary/legacy-commits"
check 'boundary: old closing keywords are never interpreted without a correction' bash -c "
  [[ \$(grep -c ' metadata=undeclared related=none completes=none' '$temporary/legacy-commits') == 4 ]]"
printf '# reviewed v0.2.0 legacy records\n%s related=129 completes=129\n%s related=147 completes=147\n' "$l143" "$l148" \
  >"$lfix/.github/delivery-corrections.txt"
lcommit 'ci(delivery): adopt the delivery contract (#160)' 'Related-Issues: none\nCompletes-Issues: none\n' >/dev/null
lcommit 'test(project): regression for #129 (#161)' 'Related-Issues: #129, #132\nCompletes-Issues: #129\n' >/dev/null
lrel=$(lcommit 'chore(main): release 0.2.0 (#146)' 'Release PR body.\n' 0.2.0)
"$lfix/scripts/delivery-issues.sh" release --tag v0.2.0 --revision "$lrel" >"$temporary/legacy"
check 'boundary: the v0.2.0 Issue set is exactly #129 and #147' grep -Fxq issues=129,147 "$temporary/legacy"
check 'boundary: #129 is listed once, by its corrected PR and the later completing PR' bash -c "
  [[ \$(grep -c '^issue\.129=' '$temporary/legacy') == 1 ]] && grep -Fxq 'issue.129=#143,#161' '$temporary/legacy'"
check 'boundary: #147 is listed once, by PR #148' bash -c "[[ \$(grep -c '^issue\.147=' '$temporary/legacy') == 1 ]] && grep -Fxq 'issue.147=#148' '$temporary/legacy'"
check 'boundary: #132 (partial, related only) and keyword-only #200 are excluded' bash -c "
  ! grep -Eq '(^issues=|,)(132|200)(,|$)|^issue\.(132|200)=' '$temporary/legacy'"
check 'boundary: corrected commits are reported as corrected, not declared' bash -c "
  [[ \$('$lfix/scripts/delivery-issues.sh' commits --from '$l0' --to '$lrel' | grep -c ' metadata=corrected ') == 2 ]]"

check 'boundary: the v0.2.0 range keeps legacy undeclared commits allowed' bash -c "
  grep -Fxq legacy_boundary=v0.2.0 '$temporary/legacy' && grep -Fxq undeclared_policy=legacy_allowed '$temporary/legacy'"

# --- 4c. After the boundary undeclared commits fail closed ------------------------------------
# A stable release above v0.2.0 accepts only declared or corrected commits;
# its own Release Please commit is the single undeclared exception.
lpost=$(lcommit 'feat: declared after the boundary (#170)' 'Related-Issues: #171\nCompletes-Issues: #171\n')
lr021=$(lcommit 'chore(main): release 0.2.1 (#172)' '' 0.2.1)
"$lfix/scripts/delivery-issues.sh" release --tag v0.2.1 --revision "$lr021" >"$temporary/post"
check 'after the boundary: a fully declared range resolves, the release commit may be undeclared' bash -c "
  grep -Fxq undeclared_policy=fail_closed '$temporary/post' && grep -Fxq issues=171 '$temporary/post' && grep -Fxq undeclared_commits=1 '$temporary/post'"
lsneak=$(lcommit 'fix: merged without metadata (#173)' 'Closes #174\n')
lr022=$(lcommit 'chore(main): release 0.2.2 (#175)' '' 0.2.2)
expect_failure 'after the boundary: an undeclared ordinary commit fails release resolution' \
  "commit $lsneak in the v0.2.2 range has no delivery metadata" "$lfix/scripts/delivery-issues.sh" release --tag v0.2.2 --revision "$lr022"
check 'after the boundary: the old closing keyword was not interpreted' bash -c "
  '$lfix/scripts/delivery-issues.sh' commits --from '$lr021' --to '$lr022' | grep -Fxq 'commit=$lsneak pr=173 release=none metadata=undeclared related=none completes=none'"
git -C "$lfix" reset -q --hard "$lsneak"
printf '%s related=174 completes=none\n' "$lsneak" >>"$lfix/.github/delivery-corrections.txt"
lcommit 'docs(delivery): reviewed correction for #173 (#176)' 'Related-Issues: none\nCompletes-Issues: none\n' >/dev/null
lr022=$(lcommit 'chore(main): release 0.2.2 (#177)' '' 0.2.2)
"$lfix/scripts/delivery-issues.sh" release --tag v0.2.2 --revision "$lr022" >"$temporary/post"
check 'after the boundary: an explicit reviewed correction resolves the commit without inferring completion' bash -c "
  grep -Fxq issues=none '$temporary/post' && grep -Fxq undeclared_commits=1 '$temporary/post'"
lsmuggle=$(lcommit 'feat: bump the manifest outside Release Please (#178)' '' 0.3.0)
expect_failure 'after the boundary: a manifest change that is not a Release Please commit stays undeclared and fails' \
  "commit $lsmuggle in the v0.3.0 range has no delivery metadata" "$lfix/scripts/delivery-issues.sh" release --tag v0.3.0 --revision "$lsmuggle"
git -C "$lfix" reset -q --hard "$lr022"
lcommit 'feat: declared (#179)' 'Related-Issues: none\nCompletes-Issues: none\n' >/dev/null
lr100=$(lcommit 'chore(main): release 0.10.0 (#180)' '' 0.10.0)
check 'after the boundary: versions compare numerically (0.10.0 enforces)' bash -c "
  '$lfix/scripts/delivery-issues.sh' release --tag v0.10.0 --revision '$lr100' | grep -Fxq undeclared_policy=fail_closed"

# --- 5. The repository's own history parses ------------------------------------------------
first=$(git -C "$repository_root" rev-list --max-parents=0 HEAD | tail -n 1)
"$issues" commits --from "$first" --to "$(git -C "$repository_root" rev-parse HEAD)" >"$temporary/history"
check 'every commit on this history parses (declared, corrected or undeclared)' bash -c "
  [[ \$(wc -l <'$temporary/history') -eq \$(git -C '$repository_root' rev-list --first-parent HEAD ^$first | wc -l) ]] &&
  ! grep -Ev ' metadata=(declared|corrected|undeclared) ' '$temporary/history' | grep -q ."

# --- 6. Contract surfaces ---------------------------------------------------------------------
template=$repository_root/.github/PULL_REQUEST_TEMPLATE.md
check 'PR template asks for both keys with a placeholder that fails until filled' bash -c "
  grep -Fxq 'Related-Issues: #<issue> or none' '$template' && grep -Fxq 'Completes-Issues: #<issue> or none' '$template' &&
  ! printf 'Related-Issues: #<issue> or none\nCompletes-Issues: #<issue> or none\n' | '$issues' parse"
check 'PR template and CONTRIBUTING never suggest closing keywords' bash -c "
  ! grep -Eiq '(^|[^[:alnum:]])(close[sd]?|fix(e[sd])?|resolve[sd]?) #[0-9]' '$template' '$repository_root/CONTRIBUTING.md'"
check 'delivery configuration binds the existing Project #5, disabled until migration' bash -c "
  jq -e '.schemaVersion == 2 and .owner == \"rgomids\" and .number == 5 and .projectId == \"PVT_kwHOAFN8-M4Bj3iz\"
    and .title == \"Axiom Delivery\" and .projection == \"disabled\" and .migrationStatuses == [\"Legacy Done\"]' '$repository_root/.github/delivery-project.json' &&
  ! grep -Eiq 'token|secret|gh[pousr]_' '$repository_root/.github/delivery-project.json'"
check 'no instruction or script creates a Project' bash -c "
  ! grep -rEn 'gh project create|createProjectV2[^A-Za-z]|createProjectV2\\(' '$repository_root/CONTRIBUTING.md' '$repository_root/docs' '$repository_root/scripts' '$repository_root/.github' '$repository_root/.agents' --exclude=test-delivery.sh"
check 'Legacy Done is never a delivery state the scripts set' bash -c "
  ! grep -n 'Legacy Done' '$repository_root/scripts/delivery-github.sh' '$repository_root/scripts/publish-release.sh' '$repository_root/scripts/delivery-issues.sh' | grep -v '^[^:]*:[0-9]*:#'"
# Reviewed corrections are bounded: each names a commit on this history.
corrections=$repository_root/.github/delivery-corrections.txt
check 'every reviewed correction names a commit of main history' bash -c "
  grep -Ev '^(#|$)' '$corrections' | while read -r sha _; do git -C '$repository_root' cat-file -e \"\$sha^{commit}\" || exit 1; done"
check 'the v0.2.0 legacy reconciliations are exactly #143 -> #129 and #148 -> #147; #132 is not completed' bash -c "
  [[ \$(grep -Evc '^(#|$)' '$corrections') == 4 ]] &&
  grep -Fxq 'f04dbf38db4cb609256ede47e5fbcbe7a16c6487 related=129 completes=129' '$corrections' &&
  grep -Fxq 'fc6cdf4749943df93ec4d91c473fe5152f4bc186 related=147 completes=147' '$corrections' &&
  ! grep -Ev '^#' '$corrections' | grep -q '132'"

check 'PR #157 restores related #153 without claiming completion' grep -Fxq \
  '684b5aca88b1176d317fa86aa223b6bc654e4519 related=153 completes=none' "$corrections"

check 'PR #149 supersedes PR #128 without claiming an Issue relationship' grep -Fxq \
  '4dbdba2a43844aa615f340e79b829956ee5880f4 related=none completes=none' "$corrections"

if ((failures > 0)); then
  printf 'FAIL: %s delivery check(s) failed\n' "$failures" >&2
  exit 1
fi
printf 'PASS: delivery metadata grammar, PR check, merge-time lines and release Issue sets\n'
