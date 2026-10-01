#!/usr/bin/env bash
# GitHub delivery projection of the explicit Issue metadata resolved by
# delivery-issues.sh. It is a human-visibility projection only: it never
# reads or writes Axiom Execution truth or `axiom:stage:*` markers.
#
#   delivery-github.sh state     --repo R --tag T --revision SHA
#       Read-only. Delivery lines of the publication envelope: the release
#       Issue set, each Issue's remote state and the exact effects.
#   delivery-github.sh preflight --repo R --tag T --revision SHA
#       Read-only. Resolves the configured Project and its fields; fails
#       closed before any publication effect when they are unusable.
#   delivery-github.sh release   --repo R --tag T --revision SHA --expect FILE
#       After the release is published and read back: applies exactly the
#       effects of the authorized delivery lines in FILE, then reads back.
#   delivery-github.sh verify    --repo R --tag T --revision SHA
#       Read-only. Every delivered Issue is closed with its release record.
#   delivery-github.sh titles    --repo R --tag T --revision SHA
#       Read-only. "N<TAB>title" for the release notes, titles sanitized.
#   delivery-github.sh sync      --repo R --from SHA --to SHA
#       Merge-time projection for the commits FROM..TO on main: a completing
#       PR moves its Issues to Awaiting Release (Issue stays open); a release
#       commit records Target Release for its Issue set.
#
# Issue comments and closure use the caller's `gh` credentials (GITHUB_TOKEN
# with issues: write in Actions). Project fields use only
# AXIOM_DELIVERY_PROJECT_TOKEN and only when .github/delivery-project.json at
# the checked-out revision names a Project. Every remote effect is printed as
# an effect= line; tokens are never printed.
set -euo pipefail

umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
bot_login='github-actions[bot]'
status_options='Planned|In Progress|In Review|Awaiting Release|Released'

fail() {
  printf 'delivery_github_error: %s\n' "$1" >&2
  exit 1
}

mode=${1:-}
[[ -n "$mode" ]] && shift
repository=
tag=
revision=
expect=
from=
to=
while (($#)); do
  case "$1" in
    --repo) repository=${2:-}; shift 2 ;;
    --tag) tag=${2:-}; shift 2 ;;
    --revision) revision=${2:-}; shift 2 ;;
    --expect) expect=${2:-}; shift 2 ;;
    --from) from=${2:-}; shift 2 ;;
    --to) to=${2:-}; shift 2 ;;
    *) fail 'invalid argument' ;;
  esac
done
[[ "$repository" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || fail 'repository must be OWNER/NAME'
command -v gh >/dev/null 2>&1 || fail 'gh is required'
command -v jq >/dev/null 2>&1 || fail 'jq is required'

temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT

value() {
  awk -F= -v key="$1" '$1 == key {sub(/^[^=]*=/, ""); print; exit}' "$2"
}

# --- Project configuration (versioned, no credentials) ----------------------
# .github/delivery-project.json binds the existing Project by owner, number
# and node id. "projection": "disabled" records that identity without any
# Project read or effect (no credential needed); only "enabled" turns Project
# effects on, after the separately authorized migration. "migrationStatuses"
# names temporary Status options (Legacy Done) that may exist during
# historical reconciliation; scripts never set or read them as a delivery
# state, and they never mean Released.
project_owner=
project_number=
project_title=
project_node=
project_enabled=false
migration_statuses=
read_project_config() {
  local config=$repository_root/.github/delivery-project.json
  if [[ ! -f "$config" ]]; then
    return
  fi
  jq -e --arg operational "$status_options" '
    (keys == ["migrationStatuses", "number", "owner", "projectId", "projection", "schemaVersion", "title"])
    and .schemaVersion == 2
    and (.owner | type == "string" and test("^[A-Za-z0-9-]{1,39}$"))
    and (.title | type == "string" and length > 0 and length <= 100)
    and (.number | type == "number" and . >= 1 and . == floor)
    and (.projectId | type == "string" and test("^PVT_[A-Za-z0-9_-]{1,64}$"))
    and (.projection == "enabled" or .projection == "disabled")
    and (.migrationStatuses | type == "array" and length <= 5
      and all(.[]; type == "string" and test("^[A-Za-z][A-Za-z ]{0,30}$")
        and (. as $m | ($operational | split("|") | index($m)) == null)))' "$config" >/dev/null \
    || fail '.github/delivery-project.json is malformed'
  project_owner=$(jq -r '.owner' "$config")
  project_number=$(jq -r '.number' "$config")
  project_title=$(jq -r '.title' "$config")
  project_node=$(jq -r '.projectId' "$config")
  migration_statuses=$(jq -r '.migrationStatuses | join("|")' "$config")
  [[ $(jq -r '.projection' "$config") == enabled ]] && project_enabled=true
  return 0
}
project_label() {
  if [[ -z "$project_number" ]]; then
    printf 'not_configured'
  else
    printf 'users/%s/projects/%s projection=%s' "$project_owner" "$project_number" \
      "$([[ "$project_enabled" == true ]] && printf enabled || printf disabled)"
  fi
}

# --- Project API (classic PAT with the project scope) -----------------------
project_gh() {
  [[ -n "${AXIOM_DELIVERY_PROJECT_TOKEN:-}" ]] \
    || fail 'a delivery Project is configured but AXIOM_DELIVERY_PROJECT_TOKEN is unavailable'
  GH_TOKEN=$AXIOM_DELIVERY_PROJECT_TOKEN gh api graphql "$@"
}

project_id=
status_field=
target_field=
# resolve_project validates the Project and its two fields; any drift from
# the documented shape fails closed before effects.
resolve_project() {
  local project
  [[ "$project_enabled" == true ]] || return 0
  [[ -z "$project_id" ]] || return 0
  # shellcheck disable=SC2016
  project=$(project_gh -f owner="$project_owner" -F number="$project_number" -f query='
    query($owner: String!, $number: Int!) {
      user(login: $owner) { projectV2(number: $number) { id title closed
        fields(first: 50) { nodes {
          ... on ProjectV2Field { id name dataType }
          ... on ProjectV2SingleSelectField { id name dataType options { id name } }
        } } } } }') || fail "cannot read Project $(project_label)"
  jq -e '.data.user.projectV2 != null' <<<"$project" >/dev/null || fail "Project $(project_label) not found"
  [[ $(jq -r '.data.user.projectV2.title' <<<"$project") == "$project_title" ]] \
    || fail "Project $(project_label) is not titled '$project_title'"
  [[ $(jq -r '.data.user.projectV2.closed' <<<"$project") == false ]] || fail "Project $(project_label) is closed"
  project_id=$(jq -r '.data.user.projectV2.id' <<<"$project")
  [[ "$project_id" == "$project_node" ]] || fail "Project $(project_label) is not the configured node $project_node"
  # The operational options must all exist; the only other options allowed
  # are the configured migration statuses. Other fields (Priority,
  # Workstream, ...) are neither read nor written.
  jq -c '[.data.user.projectV2.fields.nodes[] | select(.name == "Status")]' <<<"$project" >"$temporary/status-field"
  jq -e --arg options "$status_options" --arg migration "$migration_statuses" '
    ($options | split("|")) as $operational | ($migration | split("|") | map(select(. != ""))) as $legacy
    | length == 1 and .[0].dataType == "SINGLE_SELECT"
    and ([.[0].options[].name] as $names
      | all($operational[]; . as $o | $names | index($o) != null)
      and all($names[]; . as $n | ($operational + $legacy) | index($n) != null))' "$temporary/status-field" >/dev/null \
    || fail "Project Status must contain ${status_options//|/, } and no other option than: ${migration_statuses:-none}"
  status_field=$(jq -r '.[0].id' "$temporary/status-field")
  jq -e '[.data.user.projectV2.fields.nodes[] | select(.name == "Target Release")]
    | length == 1 and .[0].dataType == "TEXT"' <<<"$project" >/dev/null \
    || fail 'Project needs exactly one text field named Target Release'
  target_field=$(jq -r '.data.user.projectV2.fields.nodes[] | select(.name == "Target Release") | .id' <<<"$project")
}

status_option() {
  jq -r --arg name "$1" '.[0].options[] | select(.name == $name) | .id' "$temporary/status-field"
}

# project_item NODE_ID prints the Project item of an Issue, adding it when it
# is absent (GitHub returns the existing item for an Issue already present).
project_item() {
  local item
  # shellcheck disable=SC2016
  item=$(project_gh -f project="$project_id" -f content="$1" -f query='
    mutation($project: ID!, $content: ID!) {
      addProjectV2ItemById(input: {projectId: $project, contentId: $content}) { item { id } } }') \
    || fail 'cannot add the Issue to the Project'
  jq -er '.data.addProjectV2ItemById.item.id' <<<"$item" || fail 'Project item has no id'
}

set_status() {
  # shellcheck disable=SC2016
  project_gh -f project="$project_id" -f item="$1" -f field="$status_field" -f option="$(status_option "$2")" -f query='
    mutation($project: ID!, $item: ID!, $field: ID!, $option: String!) {
      updateProjectV2ItemFieldValue(input: {projectId: $project, itemId: $item, fieldId: $field,
        value: {singleSelectOptionId: $option}}) { projectV2Item { id } } }' >/dev/null \
    || fail "cannot set Project Status to $2"
}

set_target() {
  if [[ -z "$2" ]]; then
    # shellcheck disable=SC2016
    project_gh -f project="$project_id" -f item="$1" -f field="$target_field" -f query='
      mutation($project: ID!, $item: ID!, $field: ID!) {
        clearProjectV2ItemFieldValue(input: {projectId: $project, itemId: $item, fieldId: $field}) { projectV2Item { id } } }' \
      >/dev/null || fail 'cannot clear Project Target Release'
  else
    # shellcheck disable=SC2016
    project_gh -f project="$project_id" -f item="$1" -f field="$target_field" -f text="$2" -f query='
      mutation($project: ID!, $item: ID!, $field: ID!, $text: String!) {
        updateProjectV2ItemFieldValue(input: {projectId: $project, itemId: $item, fieldId: $field,
          value: {text: $text}}) { projectV2Item { id } } }' >/dev/null || fail "cannot set Project Target Release to $2"
  fi
}

# read_item ITEM prints "status<TAB>target" as stored by GitHub.
read_item() {
  # shellcheck disable=SC2016
  project_gh -f item="$1" -f query='
    query($item: ID!) { node(id: $item) { ... on ProjectV2Item {
      status: fieldValueByName(name: "Status") { ... on ProjectV2ItemFieldSingleSelectValue { name } }
      target: fieldValueByName(name: "Target Release") { ... on ProjectV2ItemFieldTextValue { text } } } } }' \
    | jq -r '"\(.data.node.status.name // "")\t\(.data.node.target.text // "")"' || fail 'cannot read the Project item back'
}

# project_apply NODE STATUS TARGET sets and reads back one Issue's projection.
# TARGET "-" leaves Target Release unchanged; "" clears it.
project_apply() {
  local node=$1 status=$2 target=$3 issue=$4 item observed
  [[ "$project_enabled" == true ]] || return 0
  resolve_project
  item=$(project_item "$node")
  set_status "$item" "$status"
  [[ "$target" == - ]] || set_target "$item" "$target"
  observed=$(read_item "$item")
  [[ "${observed%%$'\t'*}" == "$status" ]] || fail "Project Status of #$issue did not read back as $status"
  [[ "$target" == - || "${observed#*$'\t'}" == "$target" ]] || fail "Project Target Release of #$issue did not read back"
  printf 'effect=project_updated issue=%s status=%s target_release=%s\n' "$issue" "${status// /_}" \
    "$([[ "$target" == - ]] && printf unchanged || printf '%s' "${target:-cleared}")"
}

# --- Issues ---------------------------------------------------------------------
# issue_json N reads an Issue of this repository; a pull request or a missing
# Issue fails closed.
issue_json() {
  local issue
  issue=$(gh api "repos/$repository/issues/$1") || fail "Issue #$1 cannot be read (missing or inaccessible)"
  jq -e 'has("pull_request") | not' <<<"$issue" >/dev/null || fail "#$1 is a pull request, not an Issue"
  [[ $(jq -r '.number' <<<"$issue") == "$1" ]] || fail "Issue #$1 resolved to another number (transferred?)"
  printf '%s' "$issue"
}

issue_comments() {
  gh api --paginate "repos/$repository/issues/$1/comments?per_page=100" | jq -s 'add // []' \
    || fail "cannot read comments of Issue #$1"
}

# has_marker COMMENTS MARKER: a comment by the workflow bot starting with MARKER.
has_marker() {
  jq -e --arg bot "$bot_login" --arg marker "$2" \
    'any(.[]; .user.login == $bot and (.body | startswith($marker)))' <<<"$1" >/dev/null
}

released_marker() { printf '<!-- axiom-delivery:released tag=%s -->' "$1"; }
completed_marker() { printf '<!-- axiom-delivery:completed commit=%s -->' "$1"; }

comment() {
  jq -n --arg body "$2" '{body: $body}' >"$temporary/comment.json"
  gh api --method POST "repos/$repository/issues/$1/comments" --input "$temporary/comment.json" >/dev/null \
    || fail "cannot comment on Issue #$1"
}

release_facts() {
  "$repository_root/scripts/delivery-issues.sh" release --tag "$tag" --revision "$revision" >"$temporary/release" || exit 1
}

# write_state prints the delivery lines bound by the publication envelope.
write_state() {
  local issues issue state reason comments effects
  release_facts
  printf 'delivery_project=%s\n' "$(project_label)"
  issues=$(value issues "$temporary/release")
  printf 'delivery_issues=%s\n' "$issues"
  [[ "$issues" == none || "$issues" == not_applicable ]] && return 0
  printf 'delivery_range_base=%s\n' "$(value range_base "$temporary/release")"
  for issue in ${issues//,/ }; do
    issue_json "$issue" >"$temporary/issue-$issue.json"
    comments=$(issue_comments "$issue")
    state=$(jq -r '.state' "$temporary/issue-$issue.json")
    reason=$(jq -r '.state_reason // "completed"' "$temporary/issue-$issue.json")
    if [[ "$state" == open ]]; then
      if has_marker "$comments" "$(released_marker "$tag")"; then
        state=open_recorded; effects=project,close
      else
        state=open; effects=comment,project,close
      fi
    elif [[ "$reason" != completed ]]; then
      fail "Issue #$issue is closed as $reason; a release cannot deliver it (record a human decision)"
    elif has_marker "$comments" "$(released_marker "$tag")"; then
      state=closed_released; effects=none
    elif has_marker "$comments" '<!-- axiom-delivery:released tag='; then
      fail "Issue #$issue was already released by another tag; reopen it before delivering it again"
    else
      state=closed_unrecorded; effects=comment,project
    fi
    [[ "$project_enabled" == true ]] || effects=$(sed -E 's/(^|,)project(,|$)/\1/; s/,$//; s/^,//' <<<"$effects")
    [[ -n "$effects" ]] || effects=none
    printf 'delivery_issue.%s=%s delivered_by=%s\n' "$issue" "$state" "$(value "issue.$issue" "$temporary/release")"
    printf 'effect.issue.%s=%s\n' "$issue" "$effects"
  done
}

need_release_args() {
  [[ -n "$tag" ]] || fail "$mode requires --tag"
  [[ "$revision" =~ ^[0-9a-f]{40}$ ]] || fail "$mode requires a full --revision"
}

read_project_config

case "$mode" in
  state)
    need_release_args
    write_state
    ;;
  preflight)
    need_release_args
    # Only a stable release with delivered Issues has Project effects; a
    # release candidate never depends on the delivery Project.
    release_facts
    issues=$(value issues "$temporary/release")
    [[ "$issues" == none || "$issues" == not_applicable ]] || resolve_project
    printf 'delivery_project=%s\ndelivery_preflight=pass\n' "$(project_label)"
    ;;
  titles)
    need_release_args
    release_facts
    issues=$(value issues "$temporary/release")
    [[ "$issues" == none || "$issues" == not_applicable ]] && exit 0
    for issue in ${issues//,/ }; do
      # Untrusted title: control characters dropped, whitespace collapsed,
      # bounded, and rendered as a code span, where GitHub creates no
      # mention, reference, link or formatting.
      issue_json "$issue" | jq -r --arg n "$issue" '.title
        | gsub("[\u0000-\u001f\u007f]"; " ") | gsub("\\s+"; " ") | ltrimstr(" ") | rtrimstr(" ")
        | if length > 120 then .[0:119] + "…" else . end
        | gsub("`"; "\u0027")
        | "\($n)\t`\(.)`"'
    done
    ;;
  release)
    need_release_args
    [[ -f "$expect" ]] || fail 'release requires --expect with the authorized delivery lines'
    write_state >"$temporary/current"
    cmp -s "$expect" "$temporary/current" \
      || fail 'delivery state changed since authorization; review and authorize again (no Issue effect applied)'
    issues=$(value delivery_issues "$temporary/current")
    if [[ "$issues" == none || "$issues" == not_applicable ]]; then
      printf 'delivery=%s\n' "$([[ "$issues" == none ]] && printf no_issues || printf not_applicable)"
      exit 0
    fi
    resolve_project
    release_url="https://github.com/$repository/releases/tag/$tag"
    for issue in ${issues//,/ }; do
      effects=$(value "effect.issue.$issue" "$temporary/current")
      node=$(jq -r '.node_id' "$temporary/issue-$issue.json")
      prs=$(value "issue.$issue" "$temporary/release")
      [[ "$prs" == none ]] && prs='commits of this release' || prs=${prs//,/, }
      # Order project -> comment -> close: the release record is written only
      # after the Project reads back, so a recorded Issue never lacks it.
      if [[ ",$effects," == *,project,* ]]; then
        project_apply "$node" Released "$tag" "$issue"
      fi
      if [[ ",$effects," == *,comment,* ]]; then
        closing='Closing as completed.'
        [[ ",$effects," == *,close,* ]] || closing='It was already closed before this release and is recorded here.'
        comment "$issue" "$(released_marker "$tag")
Released in [$tag]($release_url) (revision \`${revision:0:12}\`), delivered by $prs. $closing"
        printf 'effect=issue_commented issue=%s marker=released tag=%s\n' "$issue" "$tag"
      fi
      if [[ ",$effects," == *,close,* ]]; then
        gh api --method PATCH "repos/$repository/issues/$issue" -f state=closed -f state_reason=completed >/dev/null \
          || fail "cannot close Issue #$issue"
        printf 'effect=issue_closed issue=%s state_reason=completed\n' "$issue"
      fi
      closed=$(issue_json "$issue")
      [[ $(jq -r '.state' <<<"$closed") == closed ]] || fail "Issue #$issue did not read back closed"
      has_marker "$(issue_comments "$issue")" "$(released_marker "$tag")" || fail "Issue #$issue has no release record after effects"
    done
    printf 'delivery=released issues=%s\n' "$issues"
    ;;
  verify)
    need_release_args
    write_state >"$temporary/current"
    issues=$(value delivery_issues "$temporary/current")
    if awk '/^effect\.issue\./ && !/=none$/ {found = 1} END {exit !found}' "$temporary/current"; then
      grep '^delivery_issue\.' "$temporary/current"
      fail "delivered Issues are not all closed with the $tag release record"
    fi
    printf 'delivery_issues=%s\ndelivery=verified\n' "$issues"
    ;;
  sync)
    [[ "$to" =~ ^[0-9a-f]{40}$ ]] || fail 'sync requires a full --to revision'
    if [[ -z "$from" ]]; then
      # Re-scan from the start of the latest release's range: every run
      # converges on the whole open window, so a push whose own run was
      # skipped or superseded is still projected. Effects are idempotent.
      from=$(git -C "$repository_root" log --first-parent --format=%H "$to" -- .release-please-manifest.json | sed -n '2p;1p' | tail -n 1)
      [[ -n "$from" ]] || fail 'no .release-please-manifest.json history to anchor the sync range'
    fi
    [[ "$from" =~ ^[0-9a-f]{40}$ ]] || fail 'sync requires a full --from revision'
    "$repository_root/scripts/delivery-issues.sh" commits --from "$from" --to "$to" >"$temporary/commits" || exit 1
    # cover: each commit and the release version whose release commit (at or
    # after it in this window) delivers it, or none.
    awk '{sha[NR] = substr($1, 8); rel[NR] = substr($3, 9)} END {
      c = "none"; for (i = NR; i >= 1; i--) { if (rel[i] != "none") c = rel[i]; print sha[i], c } }' \
      "$temporary/commits" >"$temporary/cover"
    # Plan every effect and validate every Issue before the first effect. A
    # closed Issue is never moved back: it is reported and skipped.
    : >"$temporary/plan"
    while IFS= read -r line; do
      sha=$(sed -E 's/^commit=([0-9a-f]+) .*/\1/' <<<"$line")
      pr=$(sed -E 's/.* pr=([^ ]+) .*/\1/' <<<"$line")
      release=$(sed -E 's/.* release=([^ ]+) .*/\1/' <<<"$line")
      completes=$(sed -E 's/.* completes=([^ ]+)$/\1/' <<<"$line")
      targets=none
      if [[ "$release" != none ]]; then
        "$repository_root/scripts/delivery-issues.sh" release --tag "v$release" --revision "$sha" >"$temporary/release-$sha" || exit 1
        targets=$(value issues "$temporary/release-$sha")
      fi
      : >"$temporary/entries"
      [[ "$completes" == none ]] || printf 'awaiting %s\n' ${completes//,/ } >>"$temporary/entries"
      [[ "$targets" == none ]] || printf 'target %s\n' ${targets//,/ } >>"$temporary/entries"
      while read -r kind issue; do
        state=$(issue_json "$issue" | jq -r '.state') || exit 1
        if [[ "$state" != open ]]; then
          printf 'issue_skipped=%s reason=closed commit=%s\n' "$issue" "$sha"
          printf '::warning::Issue #%s is closed; delivery sync does not move it (commit %s)\n' "$issue" "${sha:0:12}"
          continue
        fi
        # A re-opened Issue is not moved back by a delivery that a release
        # already recorded: the release commit covering this commit (or this
        # release commit itself) left its release record on the Issue.
        comments=$(issue_comments "$issue") || exit 1
        covering=$(awk -v sha="$sha" '$1 == sha {print $2}' "$temporary/cover")
        if [[ "$kind" == awaiting && "$covering" != none ]] && has_marker "$comments" "$(released_marker "v$covering")"; then
          printf 'issue_skipped=%s reason=already_released commit=%s\n' "$issue" "$sha"
          continue
        fi
        if [[ "$kind" == target ]] && has_marker "$comments" "$(released_marker "v$release")"; then
          printf 'issue_skipped=%s reason=already_released commit=%s\n' "$issue" "$sha"
          continue
        fi
        if [[ "$kind" == awaiting ]]; then
          printf 'awaiting %s %s %s\n' "$issue" "$sha" "$pr" >>"$temporary/plan"
        else
          printf 'target %s %s v%s\n' "$issue" "$sha" "$release" >>"$temporary/plan"
        fi
      done <"$temporary/entries"
    done <"$temporary/commits"
    printf 'delivery_project=%s\nrange_from=%s\n' "$(project_label)" "$from"
    printf 'commits=%s\nplanned=%s\n' "$(wc -l <"$temporary/commits" | tr -d ' ')" "$(wc -l <"$temporary/plan" | tr -d ' ')"
    [[ -s "$temporary/plan" ]] || { printf 'delivery=no_effects\n'; exit 0; }
    resolve_project
    while read -r kind issue sha extra; do
      [[ "$kind" == awaiting ]] || continue
      if has_marker "$(issue_comments "$issue")" "$(completed_marker "$sha")"; then
        printf 'issue_comment_present=%s commit=%s\n' "$issue" "$sha"
        continue
      fi
      by="commit \`${sha:0:12}\`"
      [[ "$extra" != none ]] && by="#$extra ($by)"
      comment "$issue" "$(completed_marker "$sha")
Implementation completed by $by. This Issue stays open in **Awaiting Release** until a stable release that contains it is published; the release version is recorded when the Release PR merges."
      printf 'effect=issue_commented issue=%s marker=completed commit=%s\n' "$issue" "$sha"
    done <"$temporary/plan"
    # One Project update per Issue with its final state in commit order: a
    # completion clears Target Release, a later release commit sets it.
    awk '{final[$2] = ($1 == "target" ? $4 : "-none-")} END {for (i in final) print i, final[i]}' "$temporary/plan" \
      | LC_ALL=C sort -n >"$temporary/final"
    while read -r issue target; do
      [[ "$target" == -none- ]] && target=
      node=$(issue_json "$issue" | jq -r '.node_id') || exit 1
      project_apply "$node" 'Awaiting Release' "$target" "$issue"
    done <"$temporary/final"
    printf 'delivery=synced\n'
    ;;
  *)
    fail 'usage: delivery-github.sh state|preflight|release|verify|titles|sync [options]'
    ;;
esac
