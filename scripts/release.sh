#!/usr/bin/env bash
# Maintainer release orchestration used by the $axiom-release skill.
# Merges integrate code; `start` starts releases (ADR-0011).
#
#   release.sh status  [--tag vX.Y.Z[-rc.N]] [--revision SHA] [--prepared-run ID] [--manual-acceptance ABS_DIR]
#   release.sh start [--repair-revision SHA]
#   release.sh prepare [--tag TAG] [--revision SHA]
#   release.sh publish --preview-digest DIGEST --authorize-publication [--tag TAG] [--revision SHA] [--prepared-run ID]
#                      [--manual-acceptance ABS_DIR]
#   release.sh verify  [--tag TAG] [--prepared-run ID] [--download]
#
# STATUS is a state machine: it prints state= and next_action=, discovering
# the release in progress (open or merged Release PR, in-flight runs, the
# newest verified prepared run) so no SHA, run id or intermediate digest has
# to be supplied by hand.
# START runs the release plan (release-plan.sh: every commit since the last
# release has a Conventional Commit subject and delivery metadata; SemVer
# plan) and the remote facts (among them: the previous release the plan
# builds on is a published GitHub Release at its tag and release commit),
# then dispatches release-please.yml with the
# planned version and the exact main SHA, waits, and reports the Release PR,
# whose version must equal the plan. It never approves or merges it.
# PREPARE: `prepare` dispatches release-artifacts.yml (read-only token; it
# builds, verifies and retains the exact artifact set, and publishes nothing),
# waits for it, then prints the publication envelope of that prepared set.
# status with --prepared-run downloads the prepared set and its per-row
# release-candidate acceptance Evidence, re-verifies them in a clean clone of
# the revision and prints the envelope and its preview_digest.
# MANUAL ACCEPTANCE: a stable envelope binds the acceptance Evidence of every
# release row. The rows still under the manual transition (Linux arm64, the
# Windows bounded proxy) have no Evidence in the prepared run: until it is
# given, status stops at next_action=accept_manually. --manual-acceptance
# names the directory holding `<row>/gate-evidence.json` of each of them; it
# is packed once (verify-release-acceptance.py pack), the envelope is computed
# from those exact bytes, and `publish` dispatches that same packed line as
# the manual_acceptance input, so the workflow verifies the bytes authorized.
# PUBLISH: `publish` dispatches publish-release.yml only with
# --authorize-publication and a DIGEST equal to the envelope recomputed now;
# the workflow recomputes it again from the same prepared bytes before the
# first effect. status and verify are read-only apart from `git fetch` of main
# and temporary files. This script never creates tags or releases itself,
# never approves or merges a Release PR and never approves the `release`
# environment.
# Release rules live in release-plan.sh, release-preflight.sh,
# publish-release.sh and the workflows; this script only gathers facts and
# chooses the next step.
set -euo pipefail

umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
scripts=$repository_root/scripts
workflow=publish-release.yml
prepare_workflow=release-artifacts.yml
start_workflow=release-please.yml
command=${1:-}
[[ -n "$command" ]] && shift
tag=
revision=
preview_digest=
prepared_run=
corrections_revision=
corrections_digest=
repair_revision=
repair_applied=false
manual_acceptance=
manual_pending=false
manual_rejected=false
authorized=false
download=false
while (($#)); do
  case "$1" in
    --tag) tag=${2:-}; shift 2 ;;
    --revision) revision=${2:-}; shift 2 ;;
    --preview-digest) preview_digest=${2:-}; shift 2 ;;
    --prepared-run) prepared_run=${2:-}; shift 2 ;;
    --corrections-revision) corrections_revision=${2:-}; shift 2 ;;
    --corrections-digest) corrections_digest=${2:-}; shift 2 ;;
    --repair-revision) repair_revision=${2:-}; shift 2 ;;
    --manual-acceptance) manual_acceptance=${2:-}; shift 2 ;;
    --authorize-publication) authorized=true; shift ;;
    --download) download=true; shift ;;
    *) printf 'release_error: invalid argument\n' >&2; exit 1 ;;
  esac
done

fail() {
  printf 'release_error: %s\n' "$1" >&2
  exit 1
}

export AXIOM_RELEASE_CORRECTIONS_REVISION=$corrections_revision
export AXIOM_RELEASE_CORRECTIONS_DIGEST=$corrections_digest
if [[ -n "$corrections_revision$corrections_digest" ]]; then
  [[ "$corrections_revision" =~ ^[0-9a-f]{40}$ && "$corrections_digest" =~ ^[0-9a-f]{64}$ ]] || fail 'recovery requires full --corrections-revision and --corrections-digest'
fi

export AXIOM_RELEASE_REPAIR_REVISION=$repair_revision
if [[ -n "$repair_revision" ]]; then
  [[ "$repair_revision" =~ ^[0-9a-f]{40}$ ]] || fail 'repair requires full --repair-revision'
  [[ "$command" == status || "$command" == publish || "$command" == verify || "$command" == start ]] || fail 'repair is not allowed during preparation'
  if [[ "$command" == publish || ( "$command" == status && -n "$tag" ) ]]; then
    [[ -n "$tag" && -n "$corrections_revision" ]] || fail 'repair status/publish requires explicit tag and original correction pins'
  fi
fi

temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT

# The manual Evidence is read once, here; every envelope of this invocation
# and the publication dispatch use exactly these packed bytes.
: >"$temporary/manual-acceptance"
if [[ -n "$manual_acceptance" ]]; then
  [[ "$command" == status || "$command" == publish ]] || fail '--manual-acceptance applies to status and publish only'
  [[ "$manual_acceptance" == /* && -d "$manual_acceptance" && ! -L "$manual_acceptance" ]] \
    || fail '--manual-acceptance must be an absolute directory'
  python3 "$scripts/verify-release-acceptance.py" pack --manual "$(cd "$manual_acceptance" && pwd -P)" >"$temporary/manual-acceptance" \
    2>"$temporary/manual-error" || fail "$(sed -E 's/^[a-z_]+_error: //' "$temporary/manual-error" | head -n 1)"
fi

digest_stdin() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
  else
    shasum -a 256 | awk '{print $1}'
  fi
}

value() {
  awk -F= -v key="$1" '$1 == key {sub(/^[^=]*=/, ""); print; exit}' "$2"
}

command -v gh >/dev/null 2>&1 || fail 'gh is required'
command -v jq >/dev/null 2>&1 || fail 'jq is required'
repository=$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null) || fail 'cannot resolve the GitHub repository'
[[ "$repository" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || fail 'unexpected repository name'

# ci_state SHA prints success, pending, failure or missing for the required
# checks named by the versioned main ruleset. delivery-metadata is a Pull
# Request-only check (it validates the PR description before merge), so it
# never runs on a main commit and is not a CI state of the release revision.
# ci_state SHA bound also requires each Check Run to come from the integration
# the ruleset names (ADR-0013 repair); a check without that binding fails, as
# does a policy with no required check. Like publish-release.yml, it orders
# attempts by Check Run id, which exists even while started_at is null.
# The optional third argument is an internal policy file; only published repair
# uses it, to validate immutable correction metadata against its pinned policy.
ci_state() {
  local runs name integration conclusion result=success required=0
  runs=$(gh api "repos/$repository/commits/$1/check-runs?per_page=100") || { printf 'unknown'; return; }
  while IFS=$'\t' read -r name integration; do
    [[ "$name" == delivery-metadata ]] && continue
    required=$((required + 1))
    if [[ "${2:-}" == bound && ! "$integration" =~ ^[0-9]+$ ]]; then result=failure; continue; fi
    conclusion=$(jq -r --arg name "$name" --arg app "${2:+$integration}" \
      '[.check_runs[] | select(.name == $name and ($app == "" or (.app.id | tostring) == $app))] | sort_by(if $app == "" then .started_at else [.id, .started_at] end) | last | if . == null then "missing" elif .status != "completed" then "pending" else .conclusion end' <<<"$runs")
    case "$conclusion" in
      success) ;;
      pending) [[ "$result" == success ]] && result=pending ;;
      missing) [[ "$result" == success || "$result" == pending ]] && result=missing ;;
      *) result=failure ;;
    esac
  done < <(jq -r '.rules[] | select(.type == "required_status_checks") | .parameters.required_status_checks[] | [.context, (.integration_id // "" | tostring)] | @tsv' \
    "${3:-$repository_root/.github/rulesets/main.json}")
  [[ "${2:-}" == bound && "$required" == 0 ]] && result=failure
  printf '%s' "$result"
}

# release_environment prints protected only when the `release` environment
# that gates publish-release.yml requires a reviewer.
release_environment() {
  local environment
  environment=$(gh api "repos/$repository/environments/release" 2>/dev/null) || { printf 'missing'; return; }
  if jq -e 'any(.protection_rules[]?; .type == "required_reviewers")' <<<"$environment" >/dev/null; then
    printf 'protected'
  else
    printf 'unprotected'
  fi
}

# release_commit_for VERSION prints the first-parent main commit whose
# Release Please manifest introduced VERSION, or nothing.
release_commit_for() {
  local sha
  while IFS= read -r sha; do
    if [[ $(git -C "$repository_root" show "$sha:.release-please-manifest.json" 2>/dev/null | tr -d ' \t\r\n') == "{\".\":\"$1\"}" ]] \
      && [[ $(git -C "$repository_root" show "$sha^1:.release-please-manifest.json" 2>/dev/null | tr -d ' \t\r\n') != "{\".\":\"$1\"}" ]]; then
      printf '%s' "$sha"
      return
    fi
  done < <(git -C "$repository_root" log --first-parent --format=%H origin/main -- .release-please-manifest.json)
}

# prepared_envelope downloads the prepared set of $prepared_run and its
# release-candidate acceptance Evidence, verifies them in a clean clone of
# $revision with that revision's scripts, and writes the publication envelope
# to $temporary/envelope. On refusal it sets reason.
prepared_envelope() {
  local dir=$temporary/prepared clone=$temporary/source acceptance=$temporary/acceptance origin_url row
  manual_pending=false
  manual_rejected=false
  [[ "$prepared_run" =~ ^[0-9]+$ ]] || { reason='prepared run id must be numeric'; return 1; }
  rm -rf -- "$dir" "$clone" "$acceptance"
  mkdir "$dir" "$acceptance"
  if ! gh run download "$prepared_run" --repo "$repository" --name "axiom-release-$tag" --dir "$dir" >/dev/null 2>&1; then
    reason="cannot download axiom-release-$tag from run $prepared_run"
    return 1
  fi
  origin_url=$(git -C "$repository_root" remote get-url origin)
  git clone --quiet --no-hardlinks "$repository_root" "$clone"
  git -C "$clone" checkout --quiet --detach "${repair_revision:-${corrections_revision:-$revision}}"
  git -C "$clone" fetch --quiet "$repository_root" "+refs/remotes/origin/main:refs/remotes/origin/main"
  if ! "$clone/scripts/verify-prepared-release.sh" --tag "$tag" --revision "$revision" --prepared "$dir" \
    --repo "$repository" --run "$prepared_run" --remote "$origin_url" >"$temporary/prepared-facts" 2>"$temporary/prepared-error"; then
    reason=$(sed -E 's/^[a-z_]+_error: //' "$temporary/prepared-error" | head -n 1)
    return 1
  fi
  # One Evidence artifact per blocking row; an absent one stays absent and
  # publish-release.sh refuses it unless the release is already published.
  if ! "$clone/scripts/verify-release-acceptance.py" rows >"$temporary/acceptance-rows" 2>/dev/null; then
    reason='the release scripts of this revision predate release-candidate acceptance'
    return 1
  fi
  while IFS= read -r row; do
    mkdir "$acceptance/$row"
    gh run download "$prepared_run" --repo "$repository" --name "axiom-acceptance-$tag-$row" --dir "$acceptance/$row" >/dev/null 2>&1 \
      || rm -rf -- "$acceptance/$row"
  done <"$temporary/acceptance-rows"
  # The manual-transition rows' Evidence, written back byte for byte by the
  # revision's own verifier; the envelope then binds its digests.
  if [[ -s "$temporary/manual-acceptance" ]] && ! "$clone/scripts/verify-release-acceptance.py" unpack \
    --bundle "$(cd "$temporary" && pwd -P)/manual-acceptance" --into "$(cd "$acceptance" && pwd -P)" \
    >/dev/null 2>"$temporary/manual-error"; then
    reason=$(sed -E 's/^[a-z_]+_error: //' "$temporary/manual-error" | head -n 1)
    manual_rejected=true
    return 1
  fi
  if ! "$clone/scripts/publish-release.sh" --envelope --repo "$repository" --tag "$tag" --revision "$revision" \
    --make-latest "$(value make_latest "$temporary/prepared-facts")" --prepared-run "$prepared_run" --dir "$dir/artifacts" \
    --evidence "$dir/release-evidence.txt" --notes "$dir/release-notes.md" --acceptance "$acceptance" \
    >"$temporary/envelope" 2>"$temporary/envelope-error"; then
    reason=$(sed -E 's/^[a-z_]+_error: //' "$temporary/envelope-error" | head -n 1)
    # Everything else verified; only the manual rows' Evidence is absent.
    [[ "$reason" == 'missing manual acceptance Evidence for row '* ]] && manual_pending=true
    # The automated rows verified first; the given manual Evidence is what
    # failed. That is the operator's to fix, never a reason to re-prepare.
    [[ -s "$temporary/manual-acceptance" && "$reason" =~ ^acceptance\ Evidence\ of\ row\ (linux-arm64|windows-amd64)\  ]] \
      && manual_rejected=true
    return 1
  fi
}

# awaiting_manual_acceptance reports a verified prepared run whose envelope
# waits only for the manual-transition rows' Evidence. It is not authorizable.
awaiting_manual_acceptance() {
  state=awaiting_manual_acceptance; next=accept_manually
  reason="$reason; accept the downloaded prepared set of run $prepared_run on those rows, then rerun status with --manual-acceptance DIR"
}

# find_run WORKFLOW SINCE prints the newest dispatch run of WORKFLOW created
# at or after SINCE, as JSON, or nothing.
find_run() {
  local run=
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    run=$(gh run list --repo "$repository" --workflow "$1" --event workflow_dispatch --limit 10 \
      --json databaseId,url,createdAt | jq -c --arg since "$2" '[.[] | select(.createdAt >= $since)] | sort_by(.createdAt) | last // empty')
    [[ -n "$run" ]] && break
    sleep "${AXIOM_RELEASE_RETRY_DELAY:-3}"
  done
  printf '%s' "$run"
}

# inflight_run WORKFLOW prints the newest run of WORKFLOW that has not
# completed (queued, waiting for an environment approval, running), as JSON,
# or nothing. A run that cannot be listed fails closed.
inflight_run() {
  local runs
  runs=$(gh run list --repo "$repository" --workflow "$1" --limit 20 --json databaseId,status,url,createdAt) \
    || fail "cannot list $1 runs"
  jq -c '[.[] | select(.status != "completed")] | sort_by(.createdAt) | last // empty' <<<"$runs"
}

# await_run STATE WORKFLOW sets state, next and reason when WORKFLOW has a run
# in flight; re-running status then only reports it, never dispatches again.
await_run() {
  local run
  run=$(inflight_run "$2")
  [[ -n "$run" ]] || return 1
  state=$1
  next=await_run
  printf 'run_id=%s\nrun_url=%s\nrun_status=%s\n' "$(jq -r '.databaseId' <<<"$run")" "$(jq -r '.url' <<<"$run")" \
    "$(jq -r '.status' <<<"$run")" >>"$temporary/status"
  reason="$2 run $(jq -r '.databaseId' <<<"$run") is $(jq -r '.status' <<<"$run"); wait for it (a publish run waits for the human release environment approval), then run status again"
}

# github_release_exists TAG succeeds when any GitHub Release (draft included)
# uses TAG as its tag or name.
github_release_exists() {
  local releases
  releases=$(gh api --paginate "repos/$repository/releases?per_page=100" | jq -s 'add // []') || fail 'cannot list releases'
  jq -e --arg tag "$1" 'any(.[]; .tag_name == $tag or .name == $tag)' <<<"$releases" >/dev/null
}

# plan_facts runs release-plan.sh on origin/main into $temporary/plan and
# copies its facts into the status. On refusal it sets reason and fails.
plan_facts() {
  if ! "$scripts/release-plan.sh" --main-ref origin/main >"$temporary/plan" 2>"$temporary/plan-error"; then
    reason=$(sed 's/^release_plan_error: //' "$temporary/plan-error" | head -n 1)
    return 1
  fi
  grep -E '^(planned_version|planned_tag)=' "$temporary/plan" >>"$temporary/status"
  grep -Ev '^(planVersion|result|planned_version|planned_tag|main)=' "$temporary/plan" | sed 's/^/plan./' >>"$temporary/status"
}

# pinned_control DIR TAG SOURCE REVISION DIGEST is the one entry to the
# ADR-0010 control revision of a published recovery release. In this script
# it requires REVISION on first-parent main and descending from SOURCE (the
# release commit of TAG); only then it checks REVISION out into DIR (with
# origin/main) and validates the pins with that revision's recovery protocol,
# printing its facts. On refusal it sets reason and fails; no script of an
# unchecked revision runs.
pinned_control() {
  local dir=$1 control_tag=$2 source=$3 control=$4 control_digest=$5
  grep -Fxq "$control" < <(git -C "$repository_root" rev-list --first-parent origin/main) \
    || { reason='correction revision is not on first-parent main'; return 1; }
  git -C "$repository_root" merge-base --is-ancestor "$source" "$control" 2>/dev/null \
    || { reason='correction revision does not descend from source'; return 1; }
  rm -rf -- "$dir"
  { git clone --quiet --no-hardlinks "$repository_root" "$dir" \
    && git -C "$dir" checkout --quiet --detach "$control" \
    && git -C "$dir" fetch --quiet "$repository_root" '+refs/remotes/origin/main:refs/remotes/origin/main'; } 2>/dev/null \
    && [[ $(git -C "$dir" rev-parse HEAD) == "$control" && -f "$dir/scripts/release-recovery.sh" ]] \
    || { reason='correction revision has no recovery protocol to check out'; return 1; }
  (
    repository_root=$dir
    export AXIOM_RELEASE_CORRECTIONS_REVISION=$control AXIOM_RELEASE_CORRECTIONS_DIGEST=$control_digest
    source "$dir/scripts/release-recovery.sh"
    recovery_validate "$control_tag" "$source"
  ) || { reason='correction pins do not validate at the correction revision'; return 1; }
}

# Validate original provenance first, then the independently authorized repair
# code. No scripts from an unchecked revision execute.
pinned_repair() {
  local dir=$1 repair_tag=$2 source=$3 control=$4 control_digest=$5 repair=$6
  pinned_control "$dir-original" "$repair_tag" "$source" "$control" "$control_digest" >/dev/null || return 1
  grep -Fxq "$repair" < <(git -C "$repository_root" rev-list --first-parent origin/main) \
    || { reason='repair revision is not on first-parent main'; return 1; }
  git -C "$repository_root" merge-base --is-ancestor "$control" "$repair" \
    && [[ "$repair" != "$control" && $(ci_state "$repair" bound) == success ]] \
    || { reason='repair revision must descend from correction revision and have successful CI'; return 1; }
  rm -rf -- "$dir"
  { git clone --quiet --no-hardlinks "$repository_root" "$dir" \
    && git -C "$dir" checkout --quiet --detach "$repair" \
    && git -C "$dir" fetch --quiet "$repository_root" '+refs/remotes/origin/main:refs/remotes/origin/main'; } 2>/dev/null \
    || { reason='cannot check out repair revision'; return 1; }
  AXIOM_RELEASE_CORRECTIONS_REVISION=$control AXIOM_RELEASE_CORRECTIONS_DIGEST=$control_digest \
    "$dir/scripts/release-repair.sh" --repo "$repository" --tag "$repair_tag" --revision "$source" --repair-revision "$repair" \
    || { reason='repair provenance or published release does not validate'; return 1; }
}

# published_recovery_pins BODY sets pins_revision and pins_digest to the
# ADR-0010 correction pins a published release records in its notes, or
# empties both when it records none. Ambiguous or malformed pins set reason
# and fail.
published_recovery_pins() {
  pins_revision=
  pins_digest=
  grep -q '^- Recovery corrections ' <<<"$1" || return 0
  [[ $(grep -c '^- Recovery corrections revision:' <<<"$1") == 1 && $(grep -c '^- Recovery corrections SHA-256:' <<<"$1") == 1 ]] \
    || { reason='ambiguous recovery provenance'; return 1; }
  pins_revision=$(sed -nE 's/^- Recovery corrections revision: `([0-9a-f]{40})`$/\1/p' <<<"$1")
  pins_digest=$(sed -nE 's/^- Recovery corrections SHA-256: `([0-9a-f]{64})`$/\1/p' <<<"$1")
  [[ "$pins_revision" =~ ^[0-9a-f]{40}$ && "$pins_digest" =~ ^[0-9a-f]{64}$ ]] || { reason='malformed recovery provenance'; return 1; }
}

# previous_release_facts proves on GitHub what release-plan.sh, being
# Git-only, proves only for the tag: the release the plan builds on
# (plan.previous_tag at plan.previous_release_commit) is published as a
# non-draft GitHub Release bound to that tag and commit. It is the remote
# publication state of publish-release.sh --check, the one status and verify
# use. A recovery release (ADR-0010) is checked like verify checks it: its
# recorded pins through pinned_control, then the scripts of that control
# revision. Latest is not part of it. On refusal it sets reason and fails.
previous_release_facts() {
  local previous_tag previous_commit previous_state body check_scripts=$scripts
  previous_tag=$(value plan.previous_tag "$temporary/status")
  previous_commit=$(value plan.previous_release_commit "$temporary/status")
  # A missing or unreadable release leaves no pins; the check below then
  # classifies it (absent, or a recovery release without its pins).
  body=$(gh api "repos/$repository/releases/tags/$previous_tag" 2>/dev/null | jq -r '.body // ""') || body=
  published_recovery_pins "$body" || { reason="previous release $previous_tag: $reason"; return 1; }
  [[ -z "$repair_revision" || -n "$pins_revision" ]] || { reason='repair revision requires a published recovery release'; return 1; }
  if [[ -n "$pins_revision" ]]; then
    pinned_control "$temporary/previous-control" "$previous_tag" "$previous_commit" "$pins_revision" "$pins_digest" \
      >/dev/null 2>"$temporary/previous-error" || { reason="previous release $previous_tag: published $reason"; return 1; }
    if [[ -n "$repair_revision" ]]; then
      pinned_repair "$temporary/previous-repair" "$previous_tag" "$previous_commit" "$pins_revision" "$pins_digest" "$repair_revision" \
        >/dev/null 2>"$temporary/previous-error" || { reason="previous release $previous_tag: $reason"; return 1; }
      check_scripts=$temporary/previous-repair/scripts
      repair_applied=true
    else
      check_scripts=$temporary/previous-control/scripts
    fi
  fi
  if ! AXIOM_RELEASE_CORRECTIONS_REVISION=$pins_revision AXIOM_RELEASE_CORRECTIONS_DIGEST=$pins_digest \
    "$check_scripts/publish-release.sh" --check --repo "$repository" --tag "$previous_tag" --revision "$previous_commit" \
    --make-latest false >"$temporary/previous" 2>"$temporary/previous-error"; then
    printf 'previous_release_state=%s\n' "$(value publication_state "$temporary/previous")" >>"$temporary/status"
    reason="previous release $previous_tag is not consistently published: $(sed 's/^[a-z_]*_error: //' "$temporary/previous-error" | head -n 1); resolve it with \$axiom-release (release.sh status --tag $previous_tag) first"
    return 1
  fi
  previous_state=$(value publication_state "$temporary/previous")
  printf 'previous_release_state=%s\n' "$previous_state" >>"$temporary/status"
  if [[ "$previous_state" != published ]]; then
    reason="previous release $previous_tag has its tag but its GitHub Release is ${previous_state/absent/missing}; finish it with \$axiom-release (release.sh status --tag $previous_tag) first"
    return 1
  fi
}

# discover_prepared_run sets prepared_run to the newest unexpired prepared
# artifact of $tag whose set verifies at $revision and yields an envelope.
discover_prepared_run() {
  local listing id rejected=0
  listing=$(gh api "repos/$repository/actions/artifacts?name=axiom-release-$tag&per_page=100") || fail 'cannot list prepared artifacts'
  # Candidates from main only; verify-prepared-release.sh then requires a
  # successful release-artifacts.yml dispatch and the exact revision.
  while IFS= read -r id; do
    [[ "$id" =~ ^[0-9]+$ ]] || continue
    prepared_run=$id
    if prepared_envelope || [[ "$manual_pending" == true || "$manual_rejected" == true ]]; then
      printf 'prepared_run=%s\nprepared_run_source=discovered\n' "$prepared_run" >>"$temporary/status"
      return 0
    fi
    rejected=$((rejected + 1))
    printf 'prepared_run_rejected.%s=%s\n' "$id" "$reason" >>"$temporary/status"
  done < <(jq -r --arg name "axiom-release-$tag" '[.artifacts[]? | select(.name == $name and .expired == false
      and (.workflow_run.head_branch // "main") == "main")]
    | sort_by(.created_at) | reverse | .[:3][] | .workflow_run.id' <<<"$listing")
  prepared_run=
  reason=
  ((rejected == 0)) || reason="$rejected prepared run(s) rejected (prepared_run_rejected.*); "
  return 1
}

# release_pr_facts reports the one open Release PR against the plan of the
# current main: it is ready for review only when main still validates, plans
# the same version and the PR is not behind main.
release_pr_facts() {
  local pr url version head behind base refresh sha
  pr=$(jq -c '.[0]' <<<"$open")
  url=$(jq -r '.url' <<<"$pr")
  head=$(jq -r '.headRefOid // empty' <<<"$pr")
  version=$(jq -r '.title' <<<"$pr" | sed -nE 's/^chore\(main\): release ((0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))$/\1/p')
  printf 'release_pr=%s\nrelease_pr_version=%s\nrelease_pr_head=%s\n' "$url" "${version:-unknown}" "${head:-unknown}" >>"$temporary/status"
  if [[ -z "$version" || ! "$head" =~ ^[0-9a-f]{40}$ ]]; then
    next=blocked; reason="the open Release PR $url has no readable version or head"
    return
  fi
  if ! plan_facts; then
    next=blocked; reason="$reason; fix it on main, then run release.sh start to refresh the Release PR"
    return
  fi
  if ! previous_release_facts; then
    next=blocked
    return
  fi
  # Release Please builds the Release PR as one commit on top of main.
  base=$(gh api "repos/$repository/commits/$head" | jq -r 'if (.parents | length) == 1 then .parents[0].sha else empty end') \
    || fail 'cannot read the Release PR head commit'
  printf 'release_pr_base=%s\n' "${base:-unknown}" >>"$temporary/status"
  if [[ ! "$base" =~ ^[0-9a-f]{40}$ ]] || ! git -C "$repository_root" merge-base --is-ancestor "$base" "$main" 2>/dev/null; then
    next=blocked; reason="the Release PR $url is not one commit on top of main history; close it and start again"
    return
  fi
  # Commits merged after the Release PR was built. The plan above already
  # validated them; a releasable one changes the changelog, so Release
  # Please must refresh the PR. Hidden ones leave it unchanged (Release
  # Please keeps an unchanged PR as is), so the branch is only updated.
  behind=$(git -C "$repository_root" rev-list --first-parent --count "$base..$main")
  printf 'release_pr_behind_main=%s\n' "$behind" >>"$temporary/status"
  refresh=false
  [[ $(value planned_version "$temporary/status") == "$version" ]] || refresh=true
  while IFS= read -r sha; do
    grep -Eq "^plan\.commit=$sha .* releasable=true " "$temporary/status" && refresh=true
  done < <(git -C "$repository_root" rev-list --first-parent "$base..$main")
  if [[ "$refresh" == true ]]; then
    if [[ $(value main_ci "$temporary/status") != success ]]; then
      next=blocked; reason="required CI on main $main is $(value main_ci "$temporary/status"); the Release PR must be refreshed once it is green"
    elif github_release_exists "$(value planned_tag "$temporary/status")"; then
      next=blocked; reason="a GitHub Release already uses $(value planned_tag "$temporary/status"); resolve it before refreshing the Release PR"
    else
      next=refresh_release_pr
      reason="main changed the release since the Release PR was built (it records $version, main plans $(value planned_version "$temporary/status")); run release.sh start to re-validate and refresh it"
    fi
    return
  fi
  printf 'release_pr_ci=%s\n' "$(ci_state "$head")" >>"$temporary/status"
  next=review_release_pr
  reason="human review, approval and squash merge of $url"
  if [[ "$behind" != 0 ]]; then
    reason="$reason; it is $behind validated non-releasable commit(s) behind main: update its branch in GitHub before merging"
  fi
}

# open_release_pr_status judges the open Release PR(s) for `status` and for
# `status --tag` of a stable release not merged yet: more than one is
# ambiguous, exactly one is checked by release_pr_facts. One path, so the two
# never decide differently about the same Release PR.
open_release_pr_status() {
  state=release_pr_open
  if (($(jq 'length' <<<"$open") > 1)); then
    next=blocked; reason='more than one open Release PR'
  else
    release_pr_facts
  fi
}

# status writes key=value facts and the next step to $temporary/status.
status() {
  local out=$temporary/status reason='' head branch worktree merged
  next=
  state=
  : >"$out"
  git -C "$repository_root" fetch --quiet origin main || fail 'cannot fetch origin main'
  main=$(git -C "$repository_root" rev-parse --verify origin/main)
  head=$(git -C "$repository_root" rev-parse --verify HEAD)
  branch=$(git -C "$repository_root" branch --show-current)
  worktree=clean
  [[ -z $(git -C "$repository_root" status --porcelain --untracked-files=normal) ]] || worktree=dirty
  open=$(gh pr list --repo "$repository" --state open --label 'autorelease: pending' --json number,url,title,headRefOid --limit 10) \
    || fail 'cannot list open Release PRs'
  merged=$(gh pr list --repo "$repository" --state merged --label 'autorelease: pending' --json number,url,title,mergeCommit --limit 10) \
    || fail 'cannot list merged Release PRs'
  {
    printf 'statusVersion=2\n'
    printf 'repository=%s\n' "$repository"
    printf 'root=%s\n' "$repository_root"
    printf 'branch=%s\n' "${branch:-detached}"
    printf 'head=%s\n' "$head"
    printf 'worktree=%s\n' "$worktree"
    printf 'main=%s\n' "$main"
    printf 'main_ci=%s\n' "$(ci_state "$main")"
    printf 'release_environment=%s\n' "$(release_environment)"
    printf 'open_release_prs=%s\n' "$(jq -r 'map("#\(.number)") | join(",") | if . == "" then "none" else . end' <<<"$open")"
    printf 'unpublished_release_prs=%s\n' "$(jq -r 'map("#\(.number)") | join(",") | if . == "" then "none" else . end' <<<"$merged")"
  } >>"$out"

  if [[ -z "$tag" ]]; then
    if (($(jq 'length' <<<"$merged") > 1)); then
      state=release_pr_merged; next=blocked; reason='more than one merged Release PR awaits publication'
    elif (($(jq 'length' <<<"$merged") == 1)); then
      state=release_pr_merged
      revision=$(jq -r '.[0].mergeCommit.oid' <<<"$merged")
      local manifest
      manifest=$(git -C "$repository_root" show "$revision:.release-please-manifest.json" 2>/dev/null | tr -d ' \t\r\n' | sed -n 's/^{"\.":"\([0-9.]*\)"}$/\1/p')
      [[ -n "$manifest" ]] && tag=v$manifest
      printf 'release_pr=%s\n' "$(jq -r '.[0].url' <<<"$merged")" >>"$out"
      [[ -n "$tag" ]] || { next=blocked; reason='merged Release PR has no readable manifest version'; }
    elif await_run release_pr_starting "$start_workflow"; then
      :
    elif (($(jq 'length' <<<"$open") > 0)); then
      open_release_pr_status
    else
      state=no_release_in_progress
      if ! plan_facts; then
        next=blocked
      elif ! previous_release_facts; then
        next=blocked
      elif [[ $(value plan.releasable "$out") != true ]]; then
        next=none; reason="no releasable commit since $(value plan.previous_tag "$out"); merges integrate code, nothing to release"
      elif [[ $(value main_ci "$out") != success ]]; then
        next=blocked; reason="required CI on main $main is $(value main_ci "$out")"
      elif github_release_exists "$(value planned_tag "$out")"; then
        next=blocked; reason="a GitHub Release already uses $(value planned_tag "$out"); resolve it before starting"
      else
        next=start_release
        reason="preflight passed for $(value planned_tag "$out"); start it with release.sh start (dispatches Release Please; no tag, release or artifact)"
      fi
    fi
  fi

  if [[ -n "$tag" && -z "${next:-}" ]]; then
    local facts channel version
    facts=$("$scripts/release-tag-version.sh" "$tag") || fail "invalid tag: $tag"
    version=$(awk -F= '$1 == "version" {print $2}' <<<"$facts")
    channel=$(awk -F= '$1 == "channel" {print $2}' <<<"$facts")
    if [[ -z "$state" ]]; then
      if [[ "$channel" == stable ]]; then state=release_pr_merged; else state=release_candidate; fi
    fi
    if [[ -z "$revision" ]]; then
      if [[ "$channel" == stable ]]; then
        revision=$(release_commit_for "$version")
        if [[ -z "$revision" ]]; then
          # Not merged yet: only an open Release PR of exactly this version
          # counts, judged by the same path as `status`.
          if await_run release_pr_starting "$start_workflow"; then
            :
          elif jq -e --arg title "chore(main): release $version" 'any(.[]; .title == $title)' <<<"$open" >/dev/null; then
            open_release_pr_status
          else
            state=no_release_in_progress
            next=blocked; reason="no Release PR prepares $version; start a release with release.sh start"
          fi
        fi
      else
        revision=$main
      fi
    fi
  fi

  if [[ -n "$tag" && -z "${next:-}" ]]; then
    printf 'tag=%s\nrevision=%s\n' "$tag" "$revision" >>"$out"
    if [[ -n "$repair_revision" ]]; then
      if pinned_repair "$temporary/repair" "$tag" "$revision" "$corrections_revision" "$corrections_digest" "$repair_revision" >>"$out"; then
        scripts=$temporary/repair/scripts
        repair_applied=true
      else
        printf 'state=blocked\nnext_action=blocked\nreason=%s\n' "$reason"
        return
      fi
    fi
    if ! "$scripts/release-preflight.sh" --tag "$tag" --revision "$revision" --main-ref origin/main >"$temporary/preflight" 2>"$temporary/preflight-error"; then
      next=blocked; reason=$(sed 's/^release_preflight_error: //' "$temporary/preflight-error" | head -n 1)
    else
      if [[ -n "$corrections_revision" ]]; then
        source "$scripts/release-recovery.sh"
        recovery_validate "$tag" "$revision" >>"$out" || fail 'invalid pinned correction recovery'
        local correction_ci_args=("$corrections_revision")
        if [[ "$repair_applied" == true ]]; then
          # Repair code already passed the current execution policy. Original
          # corrections remain immutable metadata: later CI additions cannot
          # manufacture historical runs, nor may current worktree policy stand
          # in for the rules committed with that metadata pin.
          git -C "$repository_root" show "$corrections_revision:.github/rulesets/main.json" >"$temporary/correction-policy.json" \
            || fail 'pinned correction CI policy is unavailable'
          # Validate the complete document before ci_state reads any rows;
          # process substitution alone cannot propagate a later jq failure.
          jq -se 'length == 1 and (.[0] |
            type == "object" and (.rules | type == "array") and
            all(.rules[]; type == "object" and
              if .type == "required_status_checks" then
                (.parameters | type == "object") and
                (.parameters.required_status_checks | type == "array" and
                  all(.[]; type == "object" and
                    (.context | type == "string" and length > 0) and
                    (.integration_id | type == "number" and . > 0 and . == floor)))
              else true end))' "$temporary/correction-policy.json" >/dev/null \
            || fail 'pinned correction CI policy is malformed'
          correction_ci_args+=(bound "$temporary/correction-policy.json")
        fi
        [[ $(ci_state "${correction_ci_args[@]}") == success ]] || fail 'required CI on correction revision is not successful'
      fi
      local ci
      ci=$(ci_state "$revision")
      printf 'revision_ci=%s\n' "$ci" >>"$out"
      grep -E '^(channel|prerelease|manifest_version|latest_stable|tag_state|make_latest)=' "$temporary/preflight" >>"$out"
      if ! "$scripts/publish-release.sh" --check --repo "$repository" --tag "$tag" --revision "$revision" \
        --make-latest "$(value make_latest "$temporary/preflight")" >"$temporary/remote" 2>"$temporary/remote-error"; then
        # A refused check still reports the remote state it classified (for
        # example orphan_conflict), so status never shows it as absent.
        grep -E '^(publication_state|release_id|orphan_[a-z_]+|orphan_asset\.[^=]+)=' "$temporary/remote" >>"$out" || true
        next=blocked; reason=$(sed 's/^release_publish_error: //' "$temporary/remote-error" | head -n 1)
      else
        grep -E '^(publication_state|release_id)=' "$temporary/remote" >>"$out"
        if [[ $(value publication_state "$temporary/remote") == published ]]; then
          state=published; next=verify_published; reason='already published; run release.sh verify'
          if [[ -n "$repair_revision" ]]; then
            printf 'prepared_run=%s\n' "$prepared_run" >>"$out"
            if prepared_envelope; then
              grep -v '^preview_digest=' "$temporary/envelope" | sed 's/^/preview./' >>"$out"
              grep '^preview_digest=' "$temporary/envelope" >>"$out"
              if grep -Eq '^effect\.(issue\.[0-9]+|release_pr_label)=' "$temporary/envelope" \
                && ! grep -Fxq 'effect=none' "$temporary/envelope"; then
                next=authorize_publication; reason='authorize only remaining delivery/label effects of the published release'
              fi
            else
              next=blocked
            fi
          fi
        elif await_run publishing "$workflow"; then
          :
        elif [[ "$ci" != success ]]; then
          next=blocked; reason="required CI on $revision is $ci"
        elif [[ $(release_environment) != protected ]]; then
          next=blocked; reason='the release environment is missing or has no required reviewers (docs/security/repository-security.md)'
        elif [[ "$channel" == stable ]] && ! "$scripts/delivery-issues.sh" release --tag "$tag" --revision "$revision" \
          >/dev/null 2>"$temporary/delivery-error"; then
          # The Issue set is resolved again at preparation; refusing here
          # avoids dispatching a preparation that must fail.
          next=blocked; reason=$(sed 's/^delivery_metadata_error: //' "$temporary/delivery-error" | head -n 1)
          # The release commit already exists, so a correction on main cannot
          # enter it: only the explicit ADR-0010 recovery applies.
          [[ -n "$corrections_revision" ]] || reason="$reason; this release commit already exists: see docs/development/release-recovery.md"
        elif await_run preparing "$prepare_workflow"; then
          :
        elif [[ -n "$prepared_run" ]]; then
          printf 'prepared_run=%s\n' "$prepared_run" >>"$out"
          if prepared_envelope; then
            state=awaiting_publication_authority; next=authorize_publication
            reason='human authorization required for the exact publication envelope below'
            grep -v '^preview_digest=' "$temporary/envelope" | sed 's/^/preview./' >>"$out"
            grep '^preview_digest=' "$temporary/envelope" >>"$out"
          elif [[ "$manual_pending" == true ]]; then
            awaiting_manual_acceptance
          else
            next=blocked
          fi
        elif discover_prepared_run; then
          if [[ "$manual_pending" == true ]]; then
            awaiting_manual_acceptance
          elif [[ "$manual_rejected" == true ]]; then
            next=blocked; reason="the given manual acceptance Evidence is refused: $reason"
          else
            state=awaiting_publication_authority; next=authorize_publication
            reason='human authorization required for the exact publication envelope below'
            grep -v '^preview_digest=' "$temporary/envelope" | sed 's/^/preview./' >>"$out"
            grep '^preview_digest=' "$temporary/envelope" >>"$out"
          fi
        else
          next=prepare
          reason="${reason}build and verify the exact set first: release.sh prepare (no publication)"
        fi
      fi
    fi
  fi
  # An explicit repair revision is never ignored: a path that did not validate
  # a published recovery release through it fails closed.
  if [[ -n "$repair_revision" && "$repair_applied" != true && "$next" != blocked ]]; then
    next=blocked; reason='repair revision requires a published recovery release'
  fi
  printf 'state=%s\n' "${state:-unknown}"
  printf 'next_action=%s\n' "$next"
  [[ -n "$reason" ]] && printf 'reason=%s\n' "$reason"
  true
}

case "$command" in
  status)
    status_line=$(status)
    cat "$temporary/status"
    printf '%s\n' "$status_line"
    ;;
  start)
    [[ -z "$tag$revision$prepared_run$preview_digest$corrections_revision" && "$authorized" == false && "$download" == false ]] \
      || fail 'start takes only optional --repair-revision: the release plan decides the version'
    status_line=$(status)
    grep -Eq '^next_action=(start_release|refresh_release_pr)$' <<<"$status_line" \
      || { cat "$temporary/status"; fail "starting a release is not the next step: $(tr '\n' ' ' <<<"$status_line")"; }
    planned=$(value planned_version "$temporary/status")
    main=$(value main "$temporary/status")
    [[ "$planned" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ && "$main" =~ ^[0-9a-f]{40}$ ]] || fail 'release plan has no planned version'
    dispatched_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    gh workflow run "$start_workflow" --repo "$repository" --ref main -f "planned_version=$planned" -f "main=$main" >/dev/null \
      || fail 'Release Please dispatch failed'
    printf 'effect=workflow_dispatched workflow=%s planned_version=%s main=%s tag=none release=none\n' "$start_workflow" "$planned" "$main"
    run=$(find_run "$start_workflow" "$dispatched_at")
    [[ -n "$run" ]] || fail 'cannot find the Release Please run'
    printf 'release_pr_run_url=%s\n' "$(jq -r '.url' <<<"$run")"
    gh run watch "$(jq -r '.databaseId' <<<"$run")" --repo "$repository" --exit-status >/dev/null \
      || fail "Release Please run $(jq -r '.databaseId' <<<"$run") failed; nothing was merged, tagged or published"
    status_line=$(status)
    cat "$temporary/status"
    printf '%s\n' "$status_line"
    grep -Fxq 'next_action=review_release_pr' <<<"$status_line" \
      || fail "the Release PR is not ready for review: $(tr '\n' ' ' <<<"$status_line")"
    [[ $(value release_pr_version "$temporary/status") == "$planned" ]] \
      || fail "the Release PR version differs from the planned $planned"
    ;;
  prepare)
    [[ -z "$prepared_run" ]] || fail 'prepare starts a new preparation; use status --prepared-run for an existing one'
    status_line=$(status)
    grep -Fxq 'next_action=prepare' <<<"$status_line" \
      || { cat "$temporary/status"; fail "preparation is not the next step: $(tr '\n' ' ' <<<"$status_line")"; }
    tag=$(value tag "$temporary/status")
    revision=$(value revision "$temporary/status")
    dispatched_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    dispatch_args=(workflow run "$prepare_workflow" --repo "$repository" --ref main -f "tag=$tag" -f "revision=$revision")
    if [[ -n "$corrections_revision" ]]; then dispatch_args+=(-f "corrections_revision=$corrections_revision" -f "corrections_digest=$corrections_digest"); fi
    gh "${dispatch_args[@]}" >/dev/null || fail 'preparation dispatch failed'
    printf 'effect=workflow_dispatched workflow=%s tag=%s revision=%s publication=none\n' "$prepare_workflow" "$tag" "$revision"
    run=$(find_run "$prepare_workflow" "$dispatched_at")
    [[ -n "$run" ]] || fail 'cannot find the preparation run'
    prepared_run=$(jq -r '.databaseId' <<<"$run")
    printf 'prepare_run_url=%s\n' "$(jq -r '.url' <<<"$run")"
    gh run watch "$prepared_run" --repo "$repository" --exit-status >/dev/null || fail "preparation run $prepared_run failed"
    status_line=$(status)
    cat "$temporary/status"
    printf '%s\n' "$status_line"
    ;;
  publish)
    [[ "$authorized" == true ]] \
      || fail 'publication requires explicit human authorization (--authorize-publication) for the reviewed envelope; nothing was dispatched'
    [[ "$preview_digest" =~ ^[0-9a-f]{64}$ ]] || fail 'publish requires the --preview-digest of the authorized envelope'
    [[ -z "$revision" || "$revision" =~ ^[0-9a-f]{40}$ ]] || fail 'publish requires a full --revision when one is given'
    [[ -z "$prepared_run" || "$prepared_run" =~ ^[0-9]+$ ]] || fail 'publish requires a numeric --prepared-run when one is given'
    status_line=$(status)
    grep -Fxq 'next_action=authorize_publication' <<<"$status_line" \
      || fail "publication is not the next step: $(tr '\n' ' ' <<<"$status_line")"
    # The digest binds tag, revision and prepared run; given values must also
    # equal the ones the envelope was computed from.
    for field in tag revision prepared_run; do
      given=${!field}
      current=$(value "$field" "$temporary/status")
      [[ -z "$given" || "$given" == "$current" ]] || fail "publish --${field//_/-} $given differs from the envelope ($current)"
      printf -v "$field" '%s' "$current"
    done
    grep -Fxq "preview_digest=$preview_digest" "$temporary/status" \
      || fail "preview changed; review and authorize again (current $(grep '^preview_digest=' "$temporary/status"))"
    dispatched_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    dispatch_args=(workflow run "$workflow" --repo "$repository" --ref main -f "tag=$tag" -f "revision=$revision"
      -f "prepared_run=$prepared_run" -f "preview_digest=$preview_digest")
    if [[ -n "$corrections_revision" ]]; then dispatch_args+=(-f "corrections_revision=$corrections_revision" -f "corrections_digest=$corrections_digest"); fi
    if [[ -n "$repair_revision" ]]; then dispatch_args+=(-f "repair_revision=$repair_revision"); fi
    # The exact packed manual Evidence whose digests the authorized envelope
    # binds; only an envelope that binds manual rows carries it.
    if grep -q '^preview\.acceptance_manual_transition=' "$temporary/status"; then
      [[ -s "$temporary/manual-acceptance" ]] || fail 'the authorized envelope binds manual acceptance Evidence that was not given'
      dispatch_args+=(-f "manual_acceptance=$(cat "$temporary/manual-acceptance")")
    fi
    gh "${dispatch_args[@]}" >/dev/null || fail 'workflow dispatch failed'
    printf 'effect=workflow_dispatched workflow=%s tag=%s revision=%s prepared_run=%s preview_digest=%s\n' \
      "$workflow" "$tag" "$revision" "$prepared_run" "$preview_digest"
    run=$(find_run "$workflow" "$dispatched_at")
    if [[ -n "$run" ]]; then
      printf 'run_id=%s\nrun_url=%s\n' "$(jq -r '.databaseId' <<<"$run")" "$(jq -r '.url' <<<"$run")"
    else
      printf 'run_id=unknown\n'
    fi
    printf 'next_action=human_approves_release_environment_then_watch\n'
    ;;
  verify)
    git -C "$repository_root" fetch --quiet origin main || fail 'cannot fetch origin main'
    if [[ -z "$tag" ]]; then
      # Default: the stable version main records (the latest release commit).
      manifest=$(git -C "$repository_root" show origin/main:.release-please-manifest.json | tr -d ' \t\r\n' | sed -n 's/^{"\.":"\([0-9.]*\)"}$/\1/p')
      [[ -n "$manifest" ]] || fail 'verify cannot resolve the release version from main'
      tag=v$manifest
    fi
    tag_sha=$(git -C "$repository_root" ls-remote --tags origin "refs/tags/$tag^{}" "refs/tags/$tag" | awk 'NR == 1 {print $1}')
    [[ -n "$tag_sha" ]] || fail "tag $tag is not published"
    peeled=$(git -C "$repository_root" ls-remote --tags origin "refs/tags/$tag^{}" | awk '{print $1}')
    [[ -n "$peeled" ]] && tag_sha=$peeled
    published_body=$(gh api "repos/$repository/releases/tags/$tag" | jq -r '.body // ""') || fail 'cannot read published recovery provenance'
    if [[ -z "$corrections_revision" ]]; then
      published_recovery_pins "$published_body" || fail "$reason"
      if [[ -n "$pins_revision" ]]; then
        corrections_revision=$pins_revision
        corrections_digest=$pins_digest
        export AXIOM_RELEASE_CORRECTIONS_REVISION=$corrections_revision AXIOM_RELEASE_CORRECTIONS_DIGEST=$corrections_digest
      fi
    fi
    [[ -z "$repair_revision" || -n "$corrections_revision" ]] || fail 'repair revision requires a published recovery release'
    if [[ -n "$corrections_revision" ]]; then
      pinned_control "$temporary/control" "$tag" "$tag_sha" "$corrections_revision" "$corrections_digest" || fail "published $reason"
      if [[ -n "$repair_revision" ]]; then
        pinned_repair "$temporary/repair" "$tag" "$tag_sha" "$corrections_revision" "$corrections_digest" "$repair_revision" || fail "published $reason"
        scripts=$temporary/repair/scripts
        repository_root=$temporary/repair
      else
        scripts=$temporary/control/scripts
        repository_root=$temporary/control
      fi
    fi
    "$scripts/release-preflight.sh" --tag "$tag" --revision "$tag_sha" --main-ref origin/main >"$temporary/preflight"
    "$scripts/publish-release.sh" --check --repo "$repository" --tag "$tag" --revision "$tag_sha" \
      --make-latest "$(value make_latest "$temporary/preflight")" >"$temporary/remote"
    [[ $(value publication_state "$temporary/remote") == published ]] || fail "release $tag is not published"
    latest=$(gh api "repos/$repository/releases/latest" 2>/dev/null | jq -r '.tag_name // empty') || latest=
    if [[ $(value make_latest "$temporary/preflight") == true ]]; then
      [[ "$latest" == "$tag" ]] || fail "latest release is '${latest:-none}', expected $tag"
    else
      [[ "$latest" != "$tag" ]] || fail 'release must not be latest'
    fi
    grep -v '^result=' "$temporary/remote"
    printf 'latest=%s\n' "${latest:-none}"
    # Bind the published bytes to the prepared set the envelope authorized:
    # --check already matched every asset to the published SHA256SUMS, so an
    # identical SHA256SUMS and identical notes mean identical published bytes.
    if [[ -z "$prepared_run" ]]; then
      prepared_run=$(gh api "repos/$repository/actions/artifacts?name=axiom-release-$tag&per_page=100" \
        | jq -r --arg name "axiom-release-$tag" '[.artifacts[]? | select(.name == $name and .expired == false
            and (.workflow_run.head_branch // "main") == "main")]
          | sort_by(.created_at) | last | .workflow_run.id // empty') || fail 'cannot list prepared artifacts'
    fi
    if [[ -n "$prepared_run" ]]; then
      [[ "$prepared_run" =~ ^[0-9]+$ ]] || fail 'prepared run id must be numeric'
      gh api "repos/$repository/actions/runs/$prepared_run" | jq -e '.path == ".github/workflows/release-artifacts.yml"
        and .event == "workflow_dispatch" and .head_branch == "main" and .conclusion == "success"' >/dev/null \
        || fail "run $prepared_run is not a successful release-artifacts.yml dispatch from main"
      mkdir "$temporary/prepared-set" "$temporary/published-sums"
      gh run download "$prepared_run" --repo "$repository" --name "axiom-release-$tag" --dir "$temporary/prepared-set" >/dev/null \
        || fail "cannot download axiom-release-$tag from run $prepared_run"
      gh release download "$tag" --repo "$repository" --pattern SHA256SUMS --dir "$temporary/published-sums" >/dev/null \
        || fail 'cannot download the published SHA256SUMS'
      cmp -s "$temporary/prepared-set/artifacts/SHA256SUMS" "$temporary/published-sums/SHA256SUMS" \
        || fail "published assets differ from the prepared set of run $prepared_run"
      [[ "$published_body" == "$(cat "$temporary/prepared-set/release-notes.md")" ]] \
        || fail "published release notes differ from the prepared set of run $prepared_run"
      printf 'prepared_run=%s\nprepared_match=pass\n' "$prepared_run"
    else
      printf 'prepared_match=not_checked\n'
    fi
    if [[ "$download" == true ]]; then
      mkdir "$temporary/artifacts"
      gh release download "$tag" --repo "$repository" --dir "$temporary/artifacts" >/dev/null || fail 'cannot download release assets'
      git clone --quiet --no-hardlinks "$repository_root" "$temporary/source"
      git -C "$temporary/source" checkout --quiet --detach "$tag_sha"
      "$temporary/source/scripts/verify-release-artifacts.sh" --dir "$temporary/artifacts" \
        --version "$(value version "$temporary/preflight")" --revision "$tag_sha" >"$temporary/verified"
      grep -Ev '^(result|publication)=' "$temporary/verified"
      printf 'artifacts_verified=pass\n'
    fi
    # A stable release must also have closed every Issue it delivers with its
    # release record; a release candidate delivers none.
    "$scripts/delivery-github.sh" verify --repo "$repository" --tag "$tag" --revision "$tag_sha"
    printf 'result=pass\n'
    ;;
  *)
    fail 'usage: release.sh status|start|prepare|publish|verify [options]'
    ;;
esac
