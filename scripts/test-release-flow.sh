#!/usr/bin/env bash
# Release flow contract tests: release-preflight.sh, release-notes.sh,
# publish-release.sh, the release.sh authority boundary used by the
# $axiom-release skill, and the static no-publication/least-privilege shape of
# the CI, Release PR and publication workflows.
#
# GitHub is a stateful fake `gh` on PATH backed by local JSON files; Git
# history and remote tags come from a local fixture repository and bare
# remote. No network access, tag, release or repository mutation happens.
set -euo pipefail

umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
temporary=$(cd "$temporary" && pwd -P)
trap 'rm -rf -- "$temporary"' EXIT
export AXIOM_RELEASE_RETRY_DELAY=0

digest() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

failures=0
check() {
  local label=$1
  shift
  if "$@" >/dev/null; then
    printf 'ok: %s\n' "$label"
  else
    printf 'FAIL: %s\n' "$label" >&2
    failures=$((failures + 1))
  fi
}

# expect_failure LABEL PATTERN COMMAND... requires a non-zero exit whose
# stderr contains PATTERN.
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

# --- Fake GitHub --------------------------------------------------------------
tools=$temporary/tools
state=$temporary/github
mkdir -p "$tools"
export FAKE_GH_STATE=$state
export PATH="$tools:$PATH"
cat >"$tools/gh" <<'EOF'
#!/usr/bin/env bash
# Stateful fake of the gh subset used by the release scripts.
set -euo pipefail
s=$FAKE_GH_STATE
repo=rgomids/axiom
log() { printf '%s\n' "$*" >>"$s/ledger"; }
not_found() { printf 'gh: Not Found (HTTP 404)\n' >&2; exit 1; }
maybe_fail() {
  if [[ -n "${FAKE_GH_FAIL_ON:-}" && "$*" == *"$FAKE_GH_FAIL_ON"* ]]; then
    printf 'gh: injected failure (HTTP 502)\n' >&2
    exit 1
  fi
}
digest() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'; else shasum -a 256 "$1" | awk '{print $1}'; fi; }
release_file() { printf '%s/releases/%s.json' "$s" "$1"; }
all_releases() {
  if compgen -G "$s/releases/*.json" >/dev/null; then jq -s 'sort_by(.id) | reverse' "$s"/releases/*.json; else printf '[]'; fi
}
tag_sha() { awk -v t="$1" '$1 == t {print $2}' "$s/tags" 2>/dev/null; }

case "${1:-}" in
  repo) printf '%s\n' "$repo"; exit 0 ;;
  pr)
    shift 2
    kind=open
    while (($#)); do case "$1" in --state) kind=$2; shift 2 ;; *) shift ;; esac; done
    cat "$s/pr-$kind.json" 2>/dev/null || printf '[]\n'
    exit 0 ;;
  workflow)
    log "workflow ${*:2}"
    # A dispatched workflow's remote outcome (for example Release Please
    # opening the Release PR) is staged by the test as an effect script.
    if [[ -f "$s/dispatch-effect-$3" ]]; then bash "$s/dispatch-effect-$3"; fi
    exit 0 ;;
  run)
    sub=$2
    shift 2
    workflow= name= dir= id=
    while (($#)); do
      case "$1" in
        --workflow) workflow=$2; shift 2 ;;
        --name) name=$2; shift 2 ;;
        --dir) dir=$2; shift 2 ;;
        --repo|--event|--limit|--json) shift 2 ;;
        --*) shift ;;
        *) id=$1; shift ;;
      esac
    done
    case "$sub" in
      list)
        # inflight-WORKFLOW.json stages a run that has not completed.
        if [[ -f "$s/inflight-$workflow.json" ]]; then cat "$s/inflight-$workflow.json"; exit 0; fi
        id=4242
        [[ "$workflow" == release-artifacts.yml ]] && id=5151
        [[ "$workflow" == release-please.yml ]] && id=3131
        printf '[{"databaseId":%s,"url":"https://github.com/%s/actions/runs/%s","createdAt":"%s","status":"completed"}]\n' "$id" "$repo" "$id" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" ;;
      watch) log "run watch $id" ;;
      download)
        [[ -d "$s/runs/$id/$name" ]] || not_found
        cp -R "$s/runs/$id/$name/." "$dir/" ;;
      *) exit 2 ;;
    esac
    exit 0 ;;
  release)
    # gh release download TAG --repo R --dir DIR
    tag=$3 dir=
    shift 3
    while (($#)); do case "$1" in --dir) dir=$2; shift 2 ;; *) shift ;; esac; done
    id=$(all_releases | jq -r --arg t "$tag" '.[] | select(.tag_name == $t and .draft == false) | .id')
    [[ -n "$id" ]] || not_found
    jq -r '.assets[] | "\(.id) \(.name)"' "$(release_file "$id")" | while read -r aid name; do cp "$s/assets/$aid" "$dir/$name"; done
    exit 0 ;;
  api) shift ;;
  *) printf 'fake gh: unsupported command %s\n' "$*" >&2; exit 2 ;;
esac

method=GET input= endpoint= fields=()
while (($#)); do
  case "$1" in
    --method|-X) method=$2; shift 2 ;;
    --input) input=$2; shift 2 ;;
    -H|--jq) shift 2 ;;
    -f|-F) fields+=("$2"); shift 2 ;;
    --paginate) shift ;;
    *) endpoint=$1; shift ;;
  esac
done
maybe_fail "$method $endpoint"
# GraphQL subset for the delivery Project (Projects v2). The Project, its
# fields and its items live in project.json / items.json; every mutation is
# logged as PROJECT.
if [[ "$endpoint" == graphql ]]; then
  var() { local f; for f in "${fields[@]}"; do [[ "$f" == "$1="* ]] && { printf '%s' "${f#*=}"; return; }; done; }
  query=$(var query)
  op=query
  for name in timelineItems closingIssuesReferences projectItems addProjectV2ItemById updateProjectV2ItemFieldValue clearProjectV2ItemFieldValue projectV2 node; do
    [[ "$query" == *"$name("* ]] && { op=$name; break; }
  done
  maybe_fail "graphql $op"
  # The close event of an Issue is read with the default (Issues) credential,
  # never the Project credential; closed/N.json stages its last ClosedEvent.
  if [[ "$op" == closingIssuesReferences ]]; then
    [[ "${GH_TOKEN:-}" != "${FAKE_PROJECT_TOKEN:-unset}" ]] || { printf 'gh: Project credential used for a PR read\n' >&2; exit 1; }
    jq -n --argjson n "$(cat "$s/closing-refs" 2>/dev/null || printf '[]')" \
      '{data: {repository: {pullRequest: {closingIssuesReferences: {nodes: ($n | map({number: .}))}}}}}'
    exit 0
  fi
  if [[ "$op" == timelineItems ]]; then
    [[ "${GH_TOKEN:-}" != "${FAKE_PROJECT_TOKEN:-unset}" ]] || { printf 'gh: Project credential used for an Issue read\n' >&2; exit 1; }
    cat "$s/closed/$(var number).json" 2>/dev/null || printf '{"data":{"repository":{"issue":{"timelineItems":{"nodes":[]}}}}}\n'
    exit 0
  fi
  [[ -n "${GH_TOKEN:-}" && "$GH_TOKEN" == "${FAKE_PROJECT_TOKEN:-}" ]] || { printf 'gh: Bad credentials (HTTP 401)\n' >&2; exit 1; }
  items=$s/items.json
  [[ -f "$items" ]] || printf '{}' >"$items"
  case "$op" in
    projectV2)
      jq --arg owner "$(var owner)" --argjson number "$(var number)" \
        '{data: {user: {projectV2: (if .owner == $owner and .number == $number
          then {id, title, closed, fields: {nodes: .fields}} else null end)}}}' "$s/project.json" ;;
    addProjectV2ItemById)
      content=$(var content)
      log "PROJECT add $content"
      jq --arg c "$content" 'if has($c) then . else .[$c] = {id: "PVTI_\($c)", status: "", target: ""} end' "$items" >"$items.new" && mv "$items.new" "$items"
      jq --arg c "$content" '{data: {addProjectV2ItemById: {item: {id: .[$c].id}}}}' "$items" ;;
    updateProjectV2ItemFieldValue)
      item=$(var item) field=$(var field)
      if [[ -n "$(var option)" ]]; then
        name=$(jq -r --arg f "$field" --arg o "$(var option)" '.fields[] | select(.id == $f) | .options[] | select(.id == $o) | .name' "$s/project.json")
        [[ -n "$name" ]] || { printf 'gh: unknown option\n' >&2; exit 1; }
        log "PROJECT status $item $field $name"
        jq --arg i "$item" --arg v "$name" '(.[] | select(.id == $i) | .status) = $v' "$items" >"$items.new" && mv "$items.new" "$items"
      else
        log "PROJECT target $item $field $(var text)"
        jq --arg i "$item" --arg v "$(var text)" '(.[] | select(.id == $i) | .target) = $v' "$items" >"$items.new" && mv "$items.new" "$items"
      fi
      printf '{"data":{"updateProjectV2ItemFieldValue":{"projectV2Item":{"id":"%s"}}}}\n' "$item" ;;
    clearProjectV2ItemFieldValue)
      item=$(var item)
      log "PROJECT clear $item"
      jq --arg i "$item" '(.[] | select(.id == $i) | .target) = ""' "$items" >"$items.new" && mv "$items.new" "$items"
      printf '{"data":{"clearProjectV2ItemFieldValue":{"projectV2Item":{"id":"%s"}}}}\n' "$item" ;;
    projectItems)
      jq --arg c "$(var issue)" --arg p "$(jq -r .id "$s/project.json")" '.[$c] as $it
        | {data: {node: {projectItems: {nodes: (if $it == null then [] else [{id: $it.id, project: {id: $p},
            status: (if $it.status == "" then null else {name: $it.status} end),
            target: (if $it.target == "" then null else {text: $it.target} end)}] end)}}}}' "$items" ;;
    node)
      jq --arg i "$(var item)" '[.[] | select(.id == $i)][0] as $it
        | {data: {node: {status: (if $it.status == "" then null else {name: $it.status} end),
                         target: (if $it.target == "" then null else {text: $it.target} end)}}}' "$items" ;;
    *) printf 'fake gh: unsupported graphql\n' >&2; exit 2 ;;
  esac
  exit 0
fi
path=${endpoint#https://uploads.github.com/}
path=${path#repos/$repo/}

case "$method $path" in
  "GET releases?per_page=100") all_releases ;;
  "GET releases/latest")
    [[ -s "$s/latest" ]] || not_found
    all_releases | jq --arg t "$(cat "$s/latest")" '.[] | select(.tag_name == $t)' ;;
  "GET releases/assets/"*)
    id=${path#releases/assets/}
    [[ -f "$s/assets/$id" ]] || not_found
    cat "$s/assets/$id" ;;
  "DELETE releases/assets/"*)
    id=${path#releases/assets/}
    log "DELETE asset $id"
    for f in "$s"/releases/*.json; do
      jq --argjson id "$id" '.assets |= map(select(.id != $id))' "$f" >"$f.new" && mv "$f.new" "$f"
    done
    rm -f "$s/assets/$id" ;;
  "GET releases/tags/"*)
    requested_tag=${path#releases/tags/}
    matched=$(all_releases | jq -c --arg tag "$requested_tag" '[.[] | select(.tag_name == $tag)] | .[0] // empty')
    [[ -n "$matched" ]] || not_found
    printf '%s\n' "$matched" ;;
  "GET releases/"*)
    id=${path#releases/}
    [[ -f $(release_file "$id") ]] || not_found
    cat "$(release_file "$id")" ;;
  "POST releases")
    id=$(( $(cat "$s/next-id" 2>/dev/null || echo 100) + 1 ))
    echo "$id" >"$s/next-id"
    log "POST release $(jq -r .tag_name "$input") draft=$(jq -r .draft "$input") prerelease=$(jq -r .prerelease "$input")"
    jq --argjson id "$id" --arg repo "$repo" '. + {id: $id, assets: [],
      upload_url: "https://uploads.github.com/repos/\($repo)/releases/\($id)/assets{?name,label}",
      html_url: "https://github.com/\($repo)/releases/tag/\(.tag_name)", immutable: false}' "$input" >"$(release_file "$id")"
    cat "$(release_file "$id")" ;;
  "PATCH releases/"*)
    id=${path#releases/}
    f=$(release_file "$id")
    [[ -f "$f" ]] || not_found
    log "PATCH release $id $(jq -c 'del(.body)' "$input")"
    jq -s '.[0] * (.[1] | del(.make_latest))' "$f" "$input" >"$f.new" && mv "$f.new" "$f"
    # Model of GitHub detaching a release whose tag does not exist yet into
    # untagged-*: a PATCH that omits tag_name and leaves the release a draft
    # (the v0.1.2-rc.2 incident hypothesis; v0.1.2-rc.1 shows a publishing
    # PATCH without tag_name on a fresh draft kept it), or an injected detach
    # on a draft or publishing PATCH.
    detach=false
    [[ -z $(tag_sha "$(jq -r .tag_name "$f")") && $(jq -r .draft "$f") == true ]] \
      && ! jq -e 'has("tag_name")' "$input" >/dev/null && detach=true
    [[ "${FAKE_GH_UNTAG_ON_PATCH:-}" == draft && $(jq -r .draft "$f") == true ]] && detach=true
    [[ "${FAKE_GH_UNTAG_ON_PATCH:-}" == publish && $(jq -r .draft "$f") == false ]] && detach=true
    if [[ "$detach" == true ]]; then
      jq '.tag_name = "untagged-898fac51a51187009dea" | .html_url = "https://github.com/rgomids/axiom/releases/tag/untagged-898fac51a51187009dea"' \
        "$f" >"$f.new" && mv "$f.new" "$f"
    fi
    if [[ $(jq -r .draft "$f") == false ]]; then
      t=$(jq -r .tag_name "$f")
      [[ "$t" == untagged-* || -n $(tag_sha "$t") ]] || printf '%s %s\n' "$t" "$(jq -r .target_commitish "$f")" >>"$s/tags"
      [[ $(jq -r '.make_latest // empty' "$input") == true ]] && printf '%s' "$t" >"$s/latest"
      [[ "${FAKE_GH_IMMUTABLE:-}" == 1 ]] && { jq '.immutable = true' "$f" >"$f.new" && mv "$f.new" "$f"; }
    fi
    cat "$f" ;;
  "POST releases/"*"/assets?name="*)
    rest=${path#releases/}
    id=${rest%%/*}
    name=${path##*name=}
    f=$(release_file "$id")
    aid=$(( $(cat "$s/next-asset" 2>/dev/null || echo 500) + 1 ))
    echo "$aid" >"$s/next-asset"
    mkdir -p "$s/assets"
    cp "$input" "$s/assets/$aid"
    log "UPLOAD $name"
    jq --argjson aid "$aid" --arg name "$name" --argjson size "$(wc -c <"$input" | tr -d ' ')" --arg d "sha256:$(digest "$input")" \
      '.assets += [{id: $aid, name: $name, size: $size, state: "uploaded", digest: $d}]' "$f" >"$f.new" && mv "$f.new" "$f"
    if [[ "${FAKE_GH_UNTAG_ON_UPLOAD:-}" == 1 ]]; then
      jq '.tag_name = "untagged-898fac51a51187009dea"' "$f" >"$f.new" && mv "$f.new" "$f"
    fi
    printf '{"id":%s}\n' "$aid" ;;
  "GET git/matching-refs/tags/"*)
    prefix=${path#git/matching-refs/tags/}
    awk -v p="$prefix" 'index($1, p) == 1 {printf "%s{\"ref\":\"refs/tags/%s\",\"object\":{\"type\":\"commit\",\"sha\":\"%s\"}}", (n++ ? "," : ""), $1, $2} BEGIN {printf "["} END {print "]"}' "$s/tags" 2>/dev/null || printf '[]\n' ;;
  "GET commits/"*"/check-runs?per_page=100")
    sha=${path#commits/}
    sha=${sha%%/*}
    cat "$s/checks-$sha.json" 2>/dev/null || printf '{"check_runs":[]}\n' ;;
  "GET commits/"*"/pulls") cat "$s/pulls.json" 2>/dev/null || printf '[]\n' ;;
  "GET commits/"*)
    [[ -f "$s/commit-${path#commits/}.json" ]] || not_found
    cat "$s/commit-${path#commits/}.json" ;;
  "GET pulls?"*)
    head=${path#*head=}
    head=${head%%&*}
    jq --arg h "$head" '[.[] | select((.head.repo.owner.login + ":" + .head.ref) == $h)]' "$s/open-pulls.json" 2>/dev/null || printf '[]\n' ;;
  "GET issues/"*"/comments?per_page=100")
    number=${path#issues/}
    number=${number%%/*}
    # Replace the latest closure between planning and the reopen effect.
    if [[ -n "${FAKE_GH_CLOSURE_ON_READ:-}" && "${FAKE_GH_CLOSURE_ON_READ%%:*}" == "$number" ]]; then
      reads=$(( $(cat "$s/closure-reads-$number" 2>/dev/null || echo 0) + 1 ))
      echo "$reads" >"$s/closure-reads-$number"
      if ((reads == ${FAKE_GH_CLOSURE_ON_READ##*:})); then
        cp "$s/closure-after-$number.json" "$s/closed/$number.json"
        cp "$s/issue-after-$number.json" "$s/issues/$number.json"
      fi
    fi
    # Test hook: a publication records FAKE_GH_RELEASE_ON_READ=N:TAG:K on the
    # K-th comments read of Issue N (between sync planning and its effect).
    if [[ -n "${FAKE_GH_RELEASE_ON_READ:-}" && "${FAKE_GH_RELEASE_ON_READ%%:*}" == "$number" ]]; then
      reads=$(( $(cat "$s/reads-$number" 2>/dev/null || echo 0) + 1 ))
      echo "$reads" >"$s/reads-$number"
      if ((reads == ${FAKE_GH_RELEASE_ON_READ##*:})); then
        [[ -f "$s/comments/$number.json" ]] || printf '[]' >"$s/comments/$number.json"
        tag=${FAKE_GH_RELEASE_ON_READ#*:}
        jq --arg tag "${tag%%:*}" '. + [{id: (length + 1), user: {login: "github-actions[bot]"},
          created_at: "2026-10-01T12:00:00Z", body: "<!-- axiom-delivery:released tag=\($tag) -->\npublished"}]' \
          "$s/comments/$number.json" >"$s/comments/$number.new" && mv "$s/comments/$number.new" "$s/comments/$number.json"
      fi
    fi
    cat "$s/comments/$number.json" 2>/dev/null || printf '[]\n' ;;
  "POST issues/"*"/comments")
    number=${path#issues/}
    number=${number%%/*}
    log "COMMENT $number $(jq -r '.body | split("\n")[0]' "$input")"
    mkdir -p "$s/comments"
    [[ -f "$s/comments/$number.json" ]] || printf '[]' >"$s/comments/$number.json"
    jq --slurpfile c "$input" --arg at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '. + [{id: (length + 1), user: {login: "github-actions[bot]"}, created_at: $at, body: $c[0].body}]' \
      "$s/comments/$number.json" >"$s/comments/$number.new" && mv "$s/comments/$number.new" "$s/comments/$number.json"
    printf '{"id":1}\n' ;;
  "PATCH issues/"*)
    number=${path#issues/}
    [[ -f "$s/issues/$number.json" ]] || not_found
    log "ISSUE $number ${fields[*]}"
    for f in "${fields[@]}"; do
      jq --arg k "${f%%=*}" --arg v "${f#*=}" '.[$k] = $v' "$s/issues/$number.json" >"$s/issues/$number.new" && mv "$s/issues/$number.new" "$s/issues/$number.json"
    done
    cat "$s/issues/$number.json" ;;
  "GET issues/"*)
    number=${path#issues/}
    [[ -f "$s/issues/$number.json" ]] || not_found
    cat "$s/issues/$number.json" ;;
  "POST issues/"*"/labels")
    number=${path#issues/}
    number=${number%%/*}
    log "LABEL add $number ${fields[*]}"
    jq --argjson n "$number" '(.[] | select(.number == $n) | .labels) += [{name: "autorelease: tagged"}]' "$s/pulls.json" >"$s/pulls.new" && mv "$s/pulls.new" "$s/pulls.json" ;;
  "DELETE issues/"*"/labels/"*)
    number=${path#issues/}
    number=${number%%/*}
    log "LABEL remove $number ${path##*/}"
    jq --argjson n "$number" '(.[] | select(.number == $n) | .labels) |= map(select(.name != "autorelease: pending"))' "$s/pulls.json" >"$s/pulls.new" && mv "$s/pulls.new" "$s/pulls.json" ;;
  "GET actions/artifacts?name="*)
    name=${path#actions/artifacts?name=}
    name=${name%%&*}
    jq --arg n "$name" '{artifacts: [(.artifacts // [])[] | select(.name == $n)]}' "$s/artifacts.json" 2>/dev/null \
      || printf '{"artifacts":[]}\n' ;;
  "GET actions/runs/"*)
    [[ -f "$s/runs/${path#actions/runs/}.json" ]] || not_found
    cat "$s/runs/${path#actions/runs/}.json" ;;
  "GET environments/release")
    [[ -f "$s/environment.json" ]] || not_found
    cat "$s/environment.json" ;;
  *) printf 'fake gh: unsupported api %s %s\n' "$method" "$path" >&2; exit 2 ;;
esac
EOF
chmod 700 "$tools/gh"

reset_github() {
  rm -rf -- "$state"
  mkdir -p "$state/releases" "$state/assets" "$state/issues" "$state/comments"
  : >"$state/ledger"
  : >"$state/tags"
}
mutations() {
  grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL|COMMENT|ISSUE|PROJECT|workflow)' "$state/ledger" || true
}
# release_effects counts effects on tags, releases, assets, labels, or a
# publication dispatch; a preparation dispatch is not one.
release_effects() {
  grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL|COMMENT|ISSUE|PROJECT|workflow run publish-release)' "$state/ledger" || true
}

# --- Fixture repository -----------------------------------------------------------
fixture=$temporary/fixture
remote=$temporary/remote.git
git init -q --bare "$remote"
git init -q -b main "$fixture"
git -C "$fixture" config user.email release-test@example.invalid
git -C "$fixture" config user.name 'Release Test'
git -C "$fixture" config commit.gpgsign false
mkdir -p "$fixture/scripts" "$fixture/.github/rulesets"
for script in release-tag-version.sh release-preflight.sh release-plan.sh release-notes.sh publish-release.sh release.sh verify-prepared-release.sh delivery-issues.sh delivery-github.sh release-corrections.sh release-recovery.sh; do
  cp "$repository_root/scripts/$script" "$fixture/scripts/$script"
done
cp "$repository_root/.github/rulesets/main.json" "$fixture/.github/rulesets/main.json"
# Stub of verify-release-artifacts.sh for synthetic sets: same Evidence shape,
# same revision and checksum refusals. The real verifier is covered by
# test-release-pipeline.sh.
cat >"$fixture/scripts/verify-release-artifacts.sh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
sha() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'; else shasum -a 256 "$1" | awk '{print $1}'; fi; }
while (($#)); do case "$1" in --dir) d=$2 ;; --version) v=$2 ;; --revision) r=$2 ;; esac; shift 2; done
[[ $(git -C "$root" rev-parse HEAD) == "$r" ]] || { echo 'release_verify_error: checkout is not the artifact source revision' >&2; exit 1; }
printf 'evidenceVersion=1\nproduct=Axiom\nversion=%s\nrevision=%s\n' "$v" "$r"
printf 'sha256sums=%s\n' "$(sha "$d/SHA256SUMS")"
while read -r h n; do
  [[ $(sha "$d/$n") == "$h" ]] || { echo "release_verify_error: checksum mismatch: $n" >&2; exit 1; }
  printf 'archive=%s sha256=%s axiom_sha256=x manifest_sha256=x version_smoke=not_host_architecture\n' "$n" "$h"
done <"$d/SHA256SUMS"
printf 'publication=none\nresult=pass\n'
STUB
chmod 700 "$fixture/scripts/verify-release-artifacts.sh"
printf '# Changelog\n\n## [2026-09-28]\n\n- curated history\n' >"$fixture/CHANGELOG.md"
commit() { git -C "$fixture" add -A && git -C "$fixture" commit -q -m "$1" && git -C "$fixture" rev-parse HEAD; }
c0=$(commit 'chore: base')
printf '{\n  ".": "0.0.0"\n}\n' >"$fixture/.release-please-manifest.json"
c1=$(commit 'ci(release): adopt release please')
printf 'feature\n' >"$fixture/feature.txt"
c2=$(commit 'feat(cli): add feature')
printf '{\n  ".": "0.1.0"\n}\n' >"$fixture/.release-please-manifest.json"
printf '# Changelog\n\n## [0.1.0](https://github.com/rgomids/axiom/compare/v0.0.0...v0.1.0) (2026-10-01)\n\n\n### Features\n\n* **cli:** add feature\n\n## [2026-09-28]\n\n- curated history\n' >"$fixture/CHANGELOG.md"
c3=$(commit 'chore(main): release 0.1.0')
printf 'fix\n' >"$fixture/fix.txt"
c4=$(commit 'fix(cli): repair feature')
git -C "$fixture" checkout -q -b side "$c2"
printf 'side\n' >"$fixture/side.txt"
side=$(commit 'feat: side branch')
git -C "$fixture" checkout -q main
git -C "$fixture" remote add origin "$remote"
git -C "$fixture" push -q origin main
git -C "$fixture" fetch -q origin

preflight() { "$fixture/scripts/release-preflight.sh" --main-ref origin/main "$@"; }
remote_tag() { git -C "$fixture" push -q -f origin "$2:refs/tags/$1"; }
remote_untag() { git -C "$fixture" push -q origin ":refs/tags/$1"; }

# --- 1. Preflight: SemVer, revision and Release PR binding ---------------------------
preflight --tag v0.1.0 --revision "$c3" >"$temporary/pf"
check 'stable at release commit passes' grep -Fxq 'result=pass' "$temporary/pf"
check 'stable is latest and not prerelease' bash -c "grep -Fxq make_latest=true '$temporary/pf' && grep -Fxq prerelease=false '$temporary/pf' && grep -Fxq release_commit=$c3 '$temporary/pf'"
preflight --tag v0.1.0-rc.1 --revision "$c2" >"$temporary/pf"
check 'RC on main passes as prerelease, never latest' bash -c "grep -Fxq prerelease=true '$temporary/pf' && grep -Fxq make_latest=false '$temporary/pf' && grep -Fxq channel=rc '$temporary/pf'"
check 'RC at the release commit passes' preflight --tag v0.1.0-rc.1 --revision "$c3"
for invalid in 0.1.0 v0.1 v01.0.0 v0.1.0-beta.1 v0.1.0-rc.01 v0.1.0-poc.1 'v0.1.0;id'; do
  expect_failure "invalid tag $invalid" 'release_tag_error' preflight --tag "$invalid" --revision "$c3"
done
expect_failure 'stable before its Release PR' 'merge the Release PR first' preflight --tag v0.1.0 --revision "$c2"
expect_failure 'stable after the release commit' 'not the release commit' preflight --tag v0.1.0 --revision "$c4"
expect_failure 'stable version not recorded' 'does not record version 0.2.0' preflight --tag v0.2.0 --revision "$c3"
expect_failure 'short revision' 'full source revision required' preflight --tag v0.1.0 --revision "${c3:0:12}"
expect_failure 'unknown revision' 'not a known commit' preflight --tag v0.1.0 --revision "$(printf 'a%.0s' {1..40})"
expect_failure 'revision before the release flow' 'predates the release flow' preflight --tag v0.1.0-rc.1 --revision "$c0"
expect_failure 'revision off main' 'not on the first-parent history of main' preflight --tag v0.1.0-rc.1 --revision "$side"
expect_failure 'RC older than recorded version' 'already records a newer version 0.1.0' preflight --tag v0.0.9-rc.1 --revision "$c3"
remote_tag v0.1.0-rc.1 "$c1"
preflight --tag v0.1.0-rc.1 --revision "$c1" >"$temporary/pf"
check 'rerun with existing tag at the same revision' grep -Fxq 'tag_state=present' "$temporary/pf"
expect_failure 'existing tag at another revision' 'tag already exists at another revision' preflight --tag v0.1.0-rc.1 --revision "$c2"
check 'next RC number passes' preflight --tag v0.1.0-rc.2 --revision "$c2"
remote_tag v0.1.0-rc.3 "$c2"
expect_failure 'RC number not increasing' 'greater than existing rc.3' preflight --tag v0.1.0-rc.2 --revision "$c2"
remote_tag v0.1.0-poc.1 "$c0"
check 'non-policy tags are ignored' preflight --tag v0.1.0-rc.4 --revision "$c2"
remote_tag v0.1.0 "$c3"
preflight --tag v0.1.0 --revision "$c3" >"$temporary/pf"
check 'stable rerun stays latest' bash -c "grep -Fxq tag_state=present '$temporary/pf' && grep -Fxq make_latest=true '$temporary/pf'"
expect_failure 'RC after its stable' 'stable v0.1.0 already exists' preflight --tag v0.1.0-rc.5 --revision "$c3"
expect_failure 'version not newer than stable' 'not newer than the latest stable release v0.1.0' preflight --tag v0.0.9-rc.1 --revision "$c1"
for name in v0.1.0 v0.1.0-rc.1 v0.1.0-rc.3 v0.1.0-poc.1; do remote_untag "$name"; done
printf '{".": "0.1"}\n' >"$fixture/.release-please-manifest.json"
malformed=$(commit 'chore: malformed manifest')
git -C "$fixture" push -q origin main && git -C "$fixture" fetch -q origin
expect_failure 'malformed manifest' 'release manifest is malformed' preflight --tag v0.2.0-rc.1 --revision "$malformed"
git -C "$fixture" reset -q --hard "$c4"
git -C "$fixture" push -q -f origin main && git -C "$fixture" fetch -q origin

# --- 2. Deterministic release notes ------------------------------------------------------
notes() { "$fixture/scripts/release-notes.sh" --repo rgomids/axiom "$@"; }
notes --tag v0.1.0 --revision "$c3" >"$temporary/notes-stable"
check 'stable notes use the Release Please section only' bash -c "grep -Fq '* **cli:** add feature' '$temporary/notes-stable' && ! grep -Fq 'curated history' '$temporary/notes-stable'"
check 'stable notes bind revision and exact install' bash -c "grep -Fq '$c3' '$temporary/notes-stable' && grep -Fq -- '--version v0.1.0' '$temporary/notes-stable'"
notes --tag v0.1.0-rc.1 --revision "$c2" >"$temporary/notes-rc"
check 'RC notes state prerelease and exact-tag install' bash -c "grep -Fq 'prerelease' '$temporary/notes-rc' && grep -Fq -- '--version v0.1.0-rc.1' '$temporary/notes-rc'"
cmp -s "$temporary/notes-rc" <(notes --tag v0.1.0-rc.1 --revision "$c2") && check 'notes are deterministic' true || check 'notes are deterministic' false
expect_failure 'stable notes without a changelog section' 'no Release Please section' notes --tag v0.1.0 --revision "$c2"

# --- 3. Publication state machine ------------------------------------------------------------
# make_set DIR VERSION REVISION SALT writes a synthetic artifact set and the
# verification Evidence publish-release.sh requires.
make_set() {
  local dir=$1 version=$2 rev=$3 salt=$4 row
  rm -rf -- "$dir"
  mkdir -p "$dir/artifacts"
  : >"$dir/sums"
  {
    printf 'evidenceVersion=1\nproduct=Axiom\nversion=%s\nrevision=%s\n' "$version" "$rev"
    for row in macos-27-arm64 linux-amd64 linux-arm64; do
      printf '%s %s %s\n' "$version" "$row" "$salt" >"$dir/artifacts/axiom-$version-$row.tar.gz"
      printf '%s  %s\n' "$(digest "$dir/artifacts/axiom-$version-$row.tar.gz")" "axiom-$version-$row.tar.gz" >>"$dir/sums"
    done
  } >"$dir/evidence.head"
  cp "$dir/sums" "$dir/artifacts/SHA256SUMS"
  {
    cat "$dir/evidence.head"
    printf 'sha256sums=%s\n' "$(digest "$dir/artifacts/SHA256SUMS")"
    awk '{printf "archive=%s sha256=%s axiom_sha256=x manifest_sha256=x version_smoke=not_host_architecture\n", $2, $1}' "$dir/sums"
    printf 'publication=none\nresult=pass\n'
  } >"$dir/evidence.txt"
  printf 'notes for %s\n' "$version" >"$dir/notes.md"
}
# envelope DIR ARGS prints the publication envelope of a set (read-only).
envelope() {
  local dir=$1
  shift
  "$fixture/scripts/publish-release.sh" --envelope --repo rgomids/axiom --prepared-run 11 --dir "$dir/artifacts" \
    --evidence "$dir/evidence.txt" --notes "$dir/notes.md" "$@"
}
envelope_digest() { envelope "$@" | awk -F= '$1 == "preview_digest" {print $2}'; }
# publish_raw DIR ARGS publishes with whatever --authorized-digest ARGS carry.
publish_raw() {
  local dir=$1
  shift
  "$fixture/scripts/publish-release.sh" --repo rgomids/axiom --prepared-run 11 --dir "$dir/artifacts" \
    --evidence "$dir/evidence.txt" --notes "$dir/notes.md" "$@"
}
# publish DIR ARGS authorizes exactly the current envelope, then publishes.
publish() {
  local dir=$1
  shift
  envelope "$dir" "$@" >"$temporary/envelope" || return 1
  publish_raw "$dir" "$@" --authorized-digest "$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/envelope")"
}
release_json() { jq -s --arg t "$1" '[.[] | select(.tag_name == $t)] | .[0]' "$state"/releases/*.json; }

# Publication envelope: complete, deterministic, and the only authority.
reset_github
make_set "$temporary/env-a" 0.1.0-rc.1 "$c2" first
envelope "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false >"$temporary/env"
for line in tag=v0.1.0-rc.1 "revision=$c2" channel=rc prerelease=true make_latest=false prepared_run=11 \
  "release_notes_sha256=$(digest "$temporary/env-a/notes.md")" "sha256sums_sha256=$(digest "$temporary/env-a/artifacts/SHA256SUMS")" \
  publication_state=absent effect.tag=create_at_revision effect.publish=prerelease effect.latest=unchanged \
  release_pr=not_applicable release_pr_label_state=not_applicable effect.release_pr_label=none; do
  check "envelope states $line" grep -Fxq -- "$line" "$temporary/env"
done
for f in "$temporary/env-a/artifacts"/axiom-*; do
  check "envelope states $(basename "$f") and its SHA-256" grep -Fxq "artifact.$(basename "$f")=$(digest "$f")" "$temporary/env"
done
check 'envelope lists exactly three artifacts' test "$(grep -c '^artifact\.' "$temporary/env")" == 3
grep -v '^preview_digest=' "$temporary/env" >"$temporary/env-body"
check 'preview_digest is the SHA-256 of the envelope' grep -Fxq "preview_digest=$(digest "$temporary/env-body")" "$temporary/env"
digest_a=$(envelope_digest "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false)
check 'envelope digest is deterministic' test "$digest_a" == "$(envelope_digest "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false)"
make_set "$temporary/env-b" 0.1.0-rc.1 "$c2" second
digest_b=$(envelope_digest "$temporary/env-b" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false)
check 'a changed artifact changes the digest' test "$digest_a" != "$digest_b"
cp -R "$temporary/env-a" "$temporary/env-notes"
printf 'edited\n' >>"$temporary/env-notes/notes.md"
check 'changed release notes change the digest' test "$digest_a" != "$(envelope_digest "$temporary/env-notes" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false)"
make_set "$temporary/env-c" 0.1.0-rc.1 "$c4" first
check 'another revision changes the digest' test "$digest_a" != "$(envelope_digest "$temporary/env-c" --tag v0.1.0-rc.1 --revision "$c4" --make-latest false)"
other_run=$("$fixture/scripts/publish-release.sh" --envelope --repo rgomids/axiom --prepared-run 12 --dir "$temporary/env-a/artifacts" \
  --evidence "$temporary/env-a/evidence.txt" --notes "$temporary/env-a/notes.md" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false \
  | awk -F= '$1 == "preview_digest" {print $2}')
check 'another prepared run changes the digest' test "$digest_a" != "$other_run"
check 'envelope made no GitHub effect' test "$(mutations)" == 0
expect_failure 'publication without an authorized digest' 'requires --authorized-digest' \
  publish_raw "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false
expect_failure 'authority for other bytes is stale' 'preview changed; review and authorize again' \
  publish_raw "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false --authorized-digest "$digest_b"
check 'stale or missing authority made no GitHub effect' test "$(mutations)" == 0
publish_raw "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false --authorized-digest "$digest_a" >"$temporary/pub"
check 'exact authority publishes' grep -Fxq publication=published "$temporary/pub"
envelope_assets=$(grep -E '^(artifact\.|sha256sums_sha256=)' "$temporary/env" | sed -E 's/^artifact\.//; s/^sha256sums_sha256=/SHA256SUMS=/' | LC_ALL=C sort | paste -sd, -)
published_assets=$(jq -r '.assets[] | "\(.name)=\(.digest | sub("^sha256:"; ""))"' "$state"/releases/*.json | LC_ALL=C sort | paste -sd, -)
check 'published assets are exactly the envelope artifacts' test "$envelope_assets" == "$published_assets"
before=$(mutations)
expect_failure 'old authority after the state changed is stale' 'preview changed' \
  publish_raw "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false --authorized-digest "$digest_a"
envelope "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false >"$temporary/env"
check 'published envelope has no effects' bash -c "grep -Fxq publication_state=published '$temporary/env' && grep -Fxq effect=none '$temporary/env'"
publish "$temporary/env-a" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false >"$temporary/pub"
check 'authorized rerun of a published release converges' bash -c "grep -Fxq publication=already_published '$temporary/pub' && [[ $(mutations) == $before ]]"

reset_github
make_set "$temporary/rc" 0.1.0-rc.1 "$c2" first
"$fixture/scripts/publish-release.sh" --check --repo rgomids/axiom --tag v0.1.0-rc.1 --revision "$c2" --make-latest false >"$temporary/check"
check 'check reports absent with zero effects' bash -c "grep -Fxq publication_state=absent '$temporary/check' && [[ \$(grep -Ec '^(POST|PATCH|DELETE|UPLOAD)' '$state/ledger' || true) == 0 ]]"
publish "$temporary/rc" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false >"$temporary/pub"
check 'RC publishes as prerelease' bash -c "grep -Fxq publication=published '$temporary/pub' && [[ \$(jq -s '.[0] | .draft == false and .prerelease == true' $state/releases/*.json) == true ]]"
check 'RC is not latest' bash -c "[[ ! -s '$state/latest' ]] && grep -Fxq latest=none '$temporary/pub'"
check 'tag created only at publication, at the revision' grep -Fxq "v0.1.0-rc.1 $c2" "$state/tags"
check 'draft created before any upload, published last' bash -c "head -n 1 '$state/ledger' | grep -q '^POST release v0.1.0-rc.1 draft=true prerelease=true' && grep -E '^(PATCH|UPLOAD)' '$state/ledger' | tail -n 1 | grep -q '\"draft\":false'"
check 'exactly four assets uploaded' bash -c "[[ \$(grep -c '^UPLOAD' '$state/ledger') == 4 ]]"
check 'RC does not touch Release PR labels' bash -c "grep -Fxq release_pr=not_applicable '$temporary/pub' && ! grep -q '^LABEL' '$state/ledger'"

before=$(mutations)
make_set "$temporary/rc-rebuild" 0.1.0-rc.1 "$c2" rebuilt
publish "$temporary/rc-rebuild" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false >"$temporary/pub"
check 'rerun after publication converges without effects' bash -c "grep -Fxq publication=already_published '$temporary/pub' && [[ $(mutations) == $before ]]"
check 'no duplicate release after rerun' bash -c "[[ \$(jq -s '[.[] | select(.tag_name == \"v0.1.0-rc.1\")] | length' $state/releases/*.json) == 1 ]]"

asset=$(release_json v0.1.0-rc.1 | jq -r '.assets[] | select(.name | endswith("amd64.tar.gz")) | .id')
printf 'tampered\n' >>"$state/assets/$asset"
f=$(grep -l '"v0.1.0-rc.1"' "$state"/releases/*.json)
jq --argjson id "$asset" '(.assets[] | select(.id == $id) | .digest) = null' "$f" >"$f.new" && mv "$f.new" "$f"
before=$(mutations)
expect_failure 'inconsistent published release is a conflict' 'published asset differs from SHA256SUMS' \
  publish "$temporary/rc" --tag v0.1.0-rc.1 --revision "$c2" --make-latest false
check 'published release is never modified' bash -c "[[ $(mutations) == $before ]]"

expect_failure 'RC can never be latest' 'never latest' publish "$temporary/rc" --tag v0.1.0-rc.2 --revision "$c2" --make-latest true
expect_failure 'invalid tag before any effect' 'release_tag_error' publish "$temporary/rc" --tag 0.1.0 --revision "$c2" --make-latest false

# Local set: incomplete, checksum and revision mismatches fail before GitHub.
reset_github
make_set "$temporary/bad" 0.1.0-rc.2 "$c2" x
rm "$temporary/bad/artifacts/axiom-0.1.0-rc.2-linux-arm64.tar.gz"
expect_failure 'incomplete artifact set' 'not exactly the verified set' publish "$temporary/bad" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
make_set "$temporary/bad" 0.1.0-rc.2 "$c2" x
printf 'x\n' >>"$temporary/bad/artifacts/axiom-0.1.0-rc.2-macos-27-arm64.tar.gz"
expect_failure 'invalid checksum' 'checksum mismatch' publish "$temporary/bad" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
make_set "$temporary/bad" 0.1.0-rc.2 "$c2" x
printf '0000  extra\n' >>"$temporary/bad/artifacts/SHA256SUMS"
expect_failure 'SHA256SUMS not the verified one' 'SHA256SUMS differs from verified evidence' publish "$temporary/bad" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
make_set "$temporary/bad" 0.1.0-rc.2 "$c2" x
expect_failure 'evidence for another revision' 'evidence does not name this revision' publish "$temporary/bad" --tag v0.1.0-rc.2 --revision "$c3" --make-latest false
make_set "$temporary/bad" 0.1.0-rc.2 "$c2" x
sed -i.bak 's/^result=pass$/result=fail/' "$temporary/bad/evidence.txt"
expect_failure 'failed verification evidence' 'not a passing verification' publish "$temporary/bad" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
check 'local refusals made no GitHub effect' bash -c "[[ $(mutations) == 0 ]]"

# Remote conflicts fail closed before any effect.
make_set "$temporary/rc2" 0.1.0-rc.2 "$c2" x
printf 'v0.1.0-rc.2 %s\n' "$c1" >"$state/tags"
expect_failure 'tag at another revision' 'exists at another revision' publish "$temporary/rc2" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
: >"$state/tags"
for id in 1 2; do
  printf '{"id":%s,"tag_name":"v0.1.0-rc.2","name":"v0.1.0-rc.2","target_commitish":"%s","draft":true,"prerelease":true,"assets":[]}\n' "$id" "$c2" >"$state/releases/$id.json"
done
expect_failure 'duplicate releases' 'more than one release uses v0.1.0-rc.2' publish "$temporary/rc2" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
rm "$state/releases/2.json"
jq '.target_commitish = "main"' "$state/releases/1.json" >"$state/r" && mv "$state/r" "$state/releases/1.json"
expect_failure 'draft for another revision' 'draft release targets another revision' publish "$temporary/rc2" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
jq --arg c "$c2" '.target_commitish = $c | .prerelease = false' "$state/releases/1.json" >"$state/r" && mv "$state/r" "$state/releases/1.json"
expect_failure 'draft with the wrong channel' 'wrong prerelease flag' publish "$temporary/rc2" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
jq '.prerelease = true | .assets = [{"id":9,"name":"notes.txt","state":"uploaded","size":1,"digest":null}]' "$state/releases/1.json" >"$state/r" && mv "$state/r" "$state/releases/1.json"
expect_failure 'draft with a foreign asset' 'asset outside the verified set' publish "$temporary/rc2" --tag v0.1.0-rc.2 --revision "$c2" --make-latest false
check 'remote refusals made no GitHub effect' bash -c "[[ $(mutations) == 0 ]]"

# Stable: interrupted upload leaves an unpublished draft; the rerun with a
# rebuilt set reconciles it and publishes once as latest.
reset_github
printf '[{"number":7,"merged_at":"2026-10-01T00:00:00Z","labels":[{"name":"autorelease: pending"}]}]\n' >"$state/pulls.json"
make_set "$temporary/stable" 0.1.0 "$c3" first
# The envelope binds the exact Release PR and its label state; a changed PR
# or label is a new preview and stale authority fails before any effect.
envelope "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true >"$temporary/env"
for line in release_pr=7 release_pr_label_state=pending effect.release_pr_label=pending_to_tagged; do
  check "stable envelope states $line" grep -Fxq -- "$line" "$temporary/env"
done
pr_digest=$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/env")
cp "$state/pulls.json" "$temporary/pulls.saved"
printf '[{"number":8,"merged_at":"2026-10-01T00:00:00Z","labels":[{"name":"autorelease: pending"}]}]\n' >"$state/pulls.json"
check 'another Release PR changes the digest' test "$pr_digest" != "$(envelope_digest "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true)"
expect_failure 'authority for another Release PR is stale' 'preview changed; review and authorize again' \
  publish_raw "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true --authorized-digest "$pr_digest"
printf '[{"number":7,"merged_at":"2026-10-01T00:00:00Z","labels":[{"name":"autorelease: tagged"}]}]\n' >"$state/pulls.json"
envelope "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true >"$temporary/env"
check 'a changed label state changes the digest' bash -c "grep -Fxq release_pr_label_state=tagged '$temporary/env' && grep -Fxq effect.release_pr_label=none '$temporary/env' && ! grep -Fxq preview_digest=$pr_digest '$temporary/env'"
expect_failure 'authority for another label state is stale' 'preview changed; review and authorize again' \
  publish_raw "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true --authorized-digest "$pr_digest"
check 'divergent Release PR state made no GitHub effect' test "$(mutations)" == 0
mv "$temporary/pulls.saved" "$state/pulls.json"
FAKE_GH_FAIL_ON='assets?name=axiom-0.1.0-linux-arm64' expect_failure 'interrupted upload' 'draft left for a rerun' \
  publish "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true
check 'partial state is an unpublished draft without tag' bash -c "[[ \$(jq -s '.[0].draft' $state/releases/*.json) == true && ! -s '$state/tags' && ! -s '$state/latest' ]]"
"$fixture/scripts/publish-release.sh" --check --repo rgomids/axiom --tag v0.1.0 --revision "$c3" --make-latest true >"$temporary/check"
check 'check reports the partial draft' grep -Fxq publication_state=draft "$temporary/check"
interrupted_digest=$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/envelope")
check 'the partial draft is a new preview' test "$interrupted_digest" != "$(envelope_digest "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true)"
expect_failure 'authority of the interrupted run is stale' 'preview changed' \
  publish_raw "$temporary/stable" --tag v0.1.0 --revision "$c3" --make-latest true --authorized-digest "$interrupted_digest"
make_set "$temporary/stable-rebuild" 0.1.0 "$c3" rebuilt
cp "$temporary/stable/artifacts/axiom-0.1.0-linux-amd64.tar.gz" "$temporary/stable-rebuild/artifacts/"
( cd "$temporary/stable-rebuild/artifacts" && for f in axiom-*.tar.gz; do printf '%s  %s\n' "$(digest "$f")" "$f"; done ) >"$temporary/stable-rebuild/sums"
cp "$temporary/stable-rebuild/sums" "$temporary/stable-rebuild/artifacts/SHA256SUMS"
{
  cat "$temporary/stable-rebuild/evidence.head"
  printf 'sha256sums=%s\n' "$(digest "$temporary/stable-rebuild/artifacts/SHA256SUMS")"
  awk '{printf "archive=%s sha256=%s\n", $2, $1}' "$temporary/stable-rebuild/sums"
  printf 'publication=none\nresult=pass\n'
} >"$temporary/stable-rebuild/evidence.txt"
: >"$state/ledger"
FAKE_GH_IMMUTABLE=1 publish "$temporary/stable-rebuild" --tag v0.1.0 --revision "$c3" --make-latest true >"$temporary/pub"
check 'rerun keeps identical draft assets' grep -Fxq 'draft_asset_kept=axiom-0.1.0-linux-amd64.tar.gz' "$temporary/pub"
check 'rerun replaces mismatched draft assets' grep -Fq 'effect=draft_asset_deleted name=SHA256SUMS' "$temporary/pub"
check 'stable published once, not prerelease, latest' bash -c "grep -Fxq publication=published '$temporary/pub' && grep -Fxq latest=v0.1.0 '$temporary/pub' && [[ \$(jq -s 'length == 1 and (.[0] | .draft == false and .prerelease == false)' $state/releases/*.json) == true ]]"
expected_digests=$(for f in "$temporary/stable-rebuild/artifacts"/*; do printf 'sha256:%s\n' "$(digest "$f")"; done | LC_ALL=C sort | paste -sd, -)
published_digests=$(jq -r '.assets[].digest' "$state"/releases/*.json | LC_ALL=C sort | paste -sd, -)
check 'published assets are exactly the rebuilt set' test "$expected_digests" == "$published_digests"
check 'immutable release reported' grep -Fxq immutable=true "$temporary/pub"
check 'Release PR handed to tagged' bash -c "grep -Fxq release_pr=7 '$temporary/pub' && [[ \$(jq -r '[.[0].labels[].name] | join(\",\")' '$state/pulls.json') == 'autorelease: tagged' ]]"
before=$(mutations)
publish "$temporary/stable-rebuild" --tag v0.1.0 --revision "$c3" --make-latest true >"$temporary/pub"
check 'stable rerun converges without effects' bash -c "grep -Fxq publication=already_published '$temporary/pub' && [[ $(mutations) == $before ]]"
rm "$state/latest"
expect_failure 'published stable that is not latest is reported' 'expected v0.1.0' publish "$temporary/stable-rebuild" --tag v0.1.0 --revision "$c3" --make-latest true

# --- 3b. Draft identity and orphan releases (v0.1.2-rc.2 incident) ------------------------------
# Run 36599923652 published release 399339376 as untagged-898fac51a51187009dea
# instead of v0.1.2-rc.2. The fake detaches a release whose tag does not exist
# when a PATCH omits tag_name, and can inject a detach on any draft PATCH,
# publishing PATCH or upload. Every case must fail closed before a second
# release, and an orphan must never read as absent.
publishing_patches() { grep -c '^PATCH release .*"draft":false' "$state/ledger" || true; }
release_count() { jq -s 'length' "$state"/releases/*.json 2>/dev/null || printf 0; }
# orphan_release ID TAG REVISION DIR [DRAFT] writes a release named TAG with an
# untagged-* tag_name holding the set in DIR, like release 399339376.
orphan_release() {
  local id=$1 tag=$2 rev=$3 dir=$4 draft=${5:-false} name aid=900
  jq -n --argjson id "$id" --arg tag "$tag" --arg rev "$rev" --argjson draft "$draft" \
    '{id: $id, tag_name: "untagged-898fac51a51187009dea", name: $tag, target_commitish: $rev, draft: $draft,
      prerelease: true, immutable: true, assets: [],
      html_url: "https://github.com/rgomids/axiom/releases/tag/untagged-898fac51a51187009dea"}' >"$state/releases/$id.json"
  for name in $(cd "$dir/artifacts" && ls); do
    aid=$((aid + 1))
    cp "$dir/artifacts/$name" "$state/assets/$aid"
    jq --argjson aid "$aid" --arg name "$name" --argjson size "$(wc -c <"$dir/artifacts/$name" | tr -d ' ')" \
      --arg d "sha256:$(digest "$dir/artifacts/$name")" \
      '.assets += [{id: $aid, name: $name, size: $size, state: "uploaded", digest: $d}]' \
      "$state/releases/$id.json" >"$state/r" && mv "$state/r" "$state/releases/$id.json"
  done
}
rc2=(--tag v0.1.0-rc.2 --revision "$c2" --make-latest false)

# 1 + 6. A normal publication creates the draft with the full identity and
# tags the revision only when it publishes.
reset_github
make_set "$temporary/id" 0.1.0-rc.2 "$c2" identity
publish "$temporary/id" "${rc2[@]}" >"$temporary/pub"
check 'incident: new draft is created with tag, name, revision and channel' bash -c "grep -q '^POST release v0.1.0-rc.2 draft=true prerelease=true' '$state/ledger' && [[ \$(jq -s '.[0] | .name' $state/releases/*.json) == '\"v0.1.0-rc.2\"' ]]"
check 'incident: publishing PATCH carries the identity explicitly' bash -c "grep '^PATCH release .*\"draft\":false' '$state/ledger' | grep -Fq '\"tag_name\":\"v0.1.0-rc.2\"' && grep '^PATCH release .*\"draft\":false' '$state/ledger' | grep -Fq '\"target_commitish\":\"$c2\"'"
check 'incident: valid publication is bound to the tag at the revision' bash -c "grep -Fxq publication=published '$temporary/pub' && grep -Fxq 'v0.1.0-rc.2 $c2' '$state/tags' && [[ \$(jq -s '.[0].tag_name' $state/releases/*.json) == '\"v0.1.0-rc.2\"' ]]"

# 2 + 3. Reconciling an interrupted draft keeps tag_name and target_commitish
# (a PATCH without tag_name is what detaches the draft in the fake).
reset_github
FAKE_GH_FAIL_ON='assets?name=axiom-0.1.0-rc.2-linux-arm64' expect_failure 'incident: interrupted upload leaves a draft' 'draft left for a rerun' \
  publish "$temporary/id" "${rc2[@]}"
publish "$temporary/id" "${rc2[@]}" >"$temporary/pub"
check 'incident: reconcile PATCH sends tag_name, name and target_commitish' bash -c "grep '^PATCH release .*\"draft\":true' '$state/ledger' | grep -Fq '\"tag_name\":\"v0.1.0-rc.2\"' && grep '^PATCH release .*\"draft\":true' '$state/ledger' | grep -Fq '\"name\":\"v0.1.0-rc.2\"' && grep '^PATCH release .*\"draft\":true' '$state/ledger' | grep -Fq '\"target_commitish\":\"$c2\"'"
check 'incident: reconciled draft publishes with its tag, never untagged-*' bash -c "grep -Fxq publication=published '$temporary/pub' && [[ \$(jq -sc '[.[] | .tag_name]' $state/releases/*.json) == '[\"v0.1.0-rc.2\"]' ]] && grep -Fxq 'v0.1.0-rc.2 $c2' '$state/tags'"

# 4. A draft PATCH response that comes back untagged-* stops before publish.
reset_github
FAKE_GH_FAIL_ON='assets?name=axiom-0.1.0-rc.2-linux-arm64' expect_failure 'incident: interrupted upload leaves a draft again' 'draft left for a rerun' \
  publish "$temporary/id" "${rc2[@]}"
FAKE_GH_UNTAG_ON_PATCH=draft expect_failure 'incident: untagged-* PATCH response fails before publish' \
  "refreshed draft: release tag_name is 'untagged-898fac51a51187009dea'" publish "$temporary/id" "${rc2[@]}"
check 'incident: nothing was published or tagged after the detached draft' bash -c "[[ \$(grep -c '^PATCH release .*\"draft\":false' '$state/ledger' || true) == 0 ]] && ! grep -q '^v0.1.0-rc.2 ' '$state/tags' && [[ \$(jq -s '.[0].draft' $state/releases/*.json) == true ]]"

# 5. A draft whose identity changes before publication fails closed.
reset_github
FAKE_GH_UNTAG_ON_UPLOAD=1 expect_failure 'incident: identity change before publish fails closed' \
  'draft read-back before publication: release tag_name' publish "$temporary/id" "${rc2[@]}"
check 'incident: changed identity is never published' bash -c "[[ \$(grep -c '^PATCH release .*\"draft\":false' '$state/ledger' || true) == 0 ]] && ! grep -q '^v0.1.0-rc.2 ' '$state/tags'"

# The exact incident: the publishing PATCH itself returns a public untagged-*
# release. Stop at once: no read-back retries, no tag, no second release.
reset_github
FAKE_GH_UNTAG_ON_PATCH=publish expect_failure 'incident: public untagged-* release is reported as orphan at once' \
  'orphan_conflict: release 101 is public' publish "$temporary/id" "${rc2[@]}"
check 'incident: one publishing PATCH, no tag, one release' bash -c "[[ \$(grep -c '^PATCH release .*\"draft\":false' '$state/ledger') == 1 ]] && ! grep -q '^v0.1.0-rc.2 ' '$state/tags' && [[ \$(jq -s length $state/releases/*.json) == 1 ]]"
before=$(mutations)
expect_failure 'incident: rerun after the orphan publish refuses' 'orphan_conflict' publish "$temporary/id" "${rc2[@]}"
check 'incident: rerun after the orphan publish made no effect' test "$(mutations)" == "$before"

# 7 + 8 + 9. The observed remote state of release 399339376 is an orphan
# conflict, never absent; nothing is created, and reruns are idempotent.
reset_github
orphan_release 399339376 v0.1.0-rc.2 "$c2" "$temporary/id"
for attempt in 1 2; do
  if "$fixture/scripts/publish-release.sh" --check --repo rgomids/axiom "${rc2[@]}" >"$temporary/check-$attempt" 2>"$temporary/check-err"; then
    check "incident: check $attempt refuses the orphan" false
  fi
done
check 'incident: check classifies the orphan, never absent' bash -c "grep -Fxq publication_state=orphan_conflict '$temporary/check-1' && ! grep -Fxq publication_state=absent '$temporary/check-1' && grep -Fxq release_id=399339376 '$temporary/check-1' && grep -Fxq orphan_tag_name=untagged-898fac51a51187009dea '$temporary/check-1' && grep -Fxq result=conflict '$temporary/check-1'"
check 'incident: orphan Evidence names its revision, flags and four asset digests' bash -c "grep -Fxq orphan_target=$c2 '$temporary/check-1' && grep -Fxq orphan_draft=false '$temporary/check-1' && grep -Fxq orphan_immutable=true '$temporary/check-1' && [[ \$(grep -c '^orphan_asset\\.' '$temporary/check-1') == 4 ]] && grep -Fxq 'orphan_asset.SHA256SUMS=$(digest "$temporary/id/artifacts/SHA256SUMS")' '$temporary/check-1'"
check 'incident: orphan classification is deterministic' cmp -s "$temporary/check-1" "$temporary/check-2"
expect_failure 'incident: no envelope while the orphan exists' 'orphan_conflict: release 399339376' envelope "$temporary/id" "${rc2[@]}"
expect_failure 'incident: no publication while the orphan exists' 'orphan_conflict: release 399339376' \
  publish_raw "$temporary/id" "${rc2[@]}" --authorized-digest "$(printf 'a%.0s' {1..64})"
expect_failure 'incident: publication rerun still refuses' 'orphan_conflict' \
  publish_raw "$temporary/id" "${rc2[@]}" --authorized-digest "$(printf 'a%.0s' {1..64})"
check 'incident: no second draft or release, no tag, no effect' bash -c "[[ \$(grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL|workflow)' '$state/ledger' || true) == 0 && \$(jq -s length $state/releases/*.json) == 1 ]] && ! grep -q '^v0.1.0-rc.2 ' '$state/tags'"
orphan_release 399339376 v0.1.0-rc.2 "$c2" "$temporary/id" true
expect_failure 'incident: an untagged draft named for the candidate is an orphan too' 'orphan_conflict' envelope "$temporary/id" "${rc2[@]}"

# 10. Detection is bounded, and the existing fail-closed checks still hold.
reset_github
make_set "$temporary/other" 0.0.9 "$c1" other
orphan_release 50 v0.0.9 "$c1" "$temporary/other"
envelope "$temporary/id" "${rc2[@]}" >"$temporary/env"
check 'incident: an unrelated untagged-* release is not this candidate' grep -Fxq publication_state=absent "$temporary/env"
reset_github
orphan_release 51 v0.1.0-rc.2 "$c1" "$temporary/other"
expect_failure 'incident: a release named for the tag at another revision is a conflict' 'orphan_conflict: release 51' \
  envelope "$temporary/id" "${rc2[@]}"
reset_github
printf '{"id":60,"tag_name":"v0.1.0-rc.2","name":"other","target_commitish":"%s","draft":true,"prerelease":true,"assets":[]}\n' "$c2" >"$state/releases/60.json"
expect_failure 'incident: a draft with a foreign name fails closed' "existing draft: release name is 'other'" envelope "$temporary/id" "${rc2[@]}"
check 'incident: refusals made no effect' test "$(mutations)" == 0

# A published release found by its tag converges only with its full identity:
# right tag, tag at the revision and right assets are not enough.
reset_github
publish "$temporary/id" "${rc2[@]}" >/dev/null
published=$(grep -l '"v0.1.0-rc.2"' "$state"/releases/*.json)
cp "$published" "$temporary/published.json"
before=$(mutations)
jq '.name = "other"' "$temporary/published.json" >"$published"
expect_failure 'published release with a foreign name fails closed' "existing published release: release name is 'other'" \
  publish "$temporary/id" "${rc2[@]}"
expect_failure 'check refuses a published release with a foreign name' "existing published release: release name is 'other'" \
  "$fixture/scripts/publish-release.sh" --check --repo rgomids/axiom "${rc2[@]}"
jq --arg c "$c1" '.target_commitish = $c' "$temporary/published.json" >"$published"
expect_failure 'published release targeting another revision fails closed' "existing published release: release targets '$c1'" \
  publish "$temporary/id" "${rc2[@]}"
check 'published identity refusals made no effect' bash -c "[[ \$(grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL|workflow)' '$state/ledger' || true) == $before ]] && grep -Fxq 'v0.1.0-rc.2 $c2' '$state/tags'"
cp "$temporary/published.json" "$published"
publish "$temporary/id" "${rc2[@]}" >"$temporary/pub"
check 'a fully consistent published release still converges' bash -c "grep -Fxq publication=already_published '$temporary/pub' && grep -Fxq result=pass '$temporary/pub' && [[ \$(grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL|workflow)' '$state/ledger' || true) == $before ]]"

# --- 3c. Issue delivery: merge projection and release closure ------------------------------------
# A separate fixture history with explicit delivery metadata. Issues close only
# inside an authorized stable publication, after the release reads back.
dfix=$temporary/delivery-fixture
git init -q -b main "$dfix"
git -C "$dfix" config user.email release-test@example.invalid
git -C "$dfix" config user.name 'Release Test'
git -C "$dfix" config commit.gpgsign false
mkdir -p "$dfix/scripts" "$dfix/.github"
cp "$fixture"/scripts/*.sh "$dfix/scripts/"
dcommit() {
  [[ -z "${3:-}" ]] || printf '{\n  ".": "%s"\n}\n' "$3" >"$dfix/.release-please-manifest.json"
  git -C "$dfix" add -A
  printf '%s\n\n%b' "$1" "$2" | git -C "$dfix" commit -q --allow-empty -F -
  git -C "$dfix" rev-parse HEAD
}
printf '# Changelog\n' >"$dfix/CHANGELOG.md"
d0=$(dcommit 'chore(main): release 0.1.0 (#9)' '' 0.1.0)
d1=$(dcommit 'feat(project): EDIT preview (I20-T01) (#11)' 'Related-Issues: #20\nCompletes-Issues: none\n')
d2=$(dcommit 'feat(project): CREATE and EDIT complete (#12)' 'Related-Issues: #20, #21\nCompletes-Issues: #20, #21\n')
d3=$(dcommit 'fix(cli): help (#13)' 'Related-Issues: #22\nCompletes-Issues: #22\n')
printf '# Changelog\n\n## [0.2.0](https://github.com/rgomids/axiom/compare/v0.1.0...v0.2.0) (2026-10-01)\n\n\n### Features\n\n* **project:** create and edit\n' >"$dfix/CHANGELOG.md"
drel=$(dcommit 'chore(main): release 0.2.0 (#14)' 'Release PR body.\n' 0.2.0)
dgh() { "$dfix/scripts/delivery-github.sh" "$@" --repo rgomids/axiom; }
issue() {
  jq -n --argjson n "$1" --arg title "$2" --arg state "${3:-open}" --arg reason "${4:-}" \
    '{number: $n, node_id: "I_\($n)", title: $title, state: $state, state_reason: (if $reason == "" then null else $reason end)}' \
    >"$state/issues/$1.json"
}
stage_issues() {
  issue 20 'Support create and edit in project-configure'
  issue 21 'Title with @someone, `code`, <b>bold</b> and [link](x)'
  issue 22 'Help text'
}
# project_config disabled|enabled binds the fixture Project #7 like
# .github/delivery-project.json binds Project #5.
project_config() {
  jq -n --arg projection "$1" '{schemaVersion: 2, owner: "rgomids", number: 7, projectId: "PVT_7",
    title: "Axiom Delivery", projection: $projection, migrationStatuses: ["Legacy Done"]}' >"$dfix/.github/delivery-project.json"
}
# stage_project [EXTRA_STATUS...] stages the migrated Project: the five
# operational Status options, any extra option, preserved Priority and
# Workstream fields, and Target Release.
stage_project() {
  jq -n --args '{owner: "rgomids", number: 7, title: "Axiom Delivery", closed: false, id: "PVT_7",
    fields: [{id: "F_TITLE", name: "Title", dataType: "TITLE"},
      {id: "F_STATUS", name: "Status", dataType: "SINGLE_SELECT",
       options: (([["Planned","O1"],["In Progress","O2"],["In Review","O3"],["Awaiting Release","O4"],["Released","O5"]]
         + ($ARGS.positional | to_entries | map([.value, "O9\(.key)"]))) | map({name: .[0], id: .[1]}))},
      {id: "F_PRIORITY", name: "Priority", dataType: "SINGLE_SELECT", options: [{id: "P1", name: "P1"}]},
      {id: "F_WORKSTREAM", name: "Workstream", dataType: "SINGLE_SELECT", options: [{id: "W1", name: "Delivery"}]},
      {id: "F_TARGET", name: "Target Release", dataType: "TEXT"}]}' "$@" >"$state/project.json"
}
item() { jq -r --arg c "I_$1" '.[$c] // {} | "\(.status // "")|\(.target // "")"' "$FAKE_GH_STATE/items.json" 2>/dev/null; }
export FAKE_PROJECT_TOKEN=project-token-for-tests

# Merge-time projection: partial PR no effect; completing PR -> Awaiting
# Release, Issue open; reruns idempotent; closed Issue fails before effects.
reset_github
stage_issues
project_config disabled
dgh sync --from "$d0" --to "$d1" >"$temporary/sync"
check 'partial PR merge: no Issue effect' bash -c "grep -Fxq planned=0 '$temporary/sync' && [[ \$(grep -Ec '^(COMMENT|ISSUE|PROJECT)' '$state/ledger' || true) == 0 ]]"
dgh sync --from "$d1" --to "$d2" >"$temporary/sync"
check 'completing PR merge comments on each completed Issue once' bash -c "[[ \$(grep -c '^COMMENT 20 <!-- axiom-delivery:completed commit=$d2 -->' '$state/ledger') == 1 && \$(grep -c '^COMMENT 21 ' '$state/ledger') == 1 ]] && grep -Fxq 'delivery_project=users/rgomids/projects/7 projection=disabled' '$temporary/sync'"
check 'merge never closes the Issue' bash -c "[[ \$(jq -r .state '$state/issues/20.json') == open ]] && ! grep -q '^ISSUE' '$state/ledger'"
check 'merge comment is bounded and names the PR' bash -c "jq -e '.[0].body | contains(\"#12\") and contains(\"Awaiting Release\") and (length < 400)' '$state/comments/20.json'"
dgh sync --from "$d1" --to "$d2" >"$temporary/sync"
check 'merge projection rerun is idempotent' bash -c "[[ \$(jq length '$state/comments/20.json') == 1 ]] && grep -Fxq 'issue_comment_present=20 commit=$d2' '$temporary/sync'"
issue 22 'Help text' closed completed
before=$(mutations)
dgh sync --from "$d1" --to "$d3" >"$temporary/sync"
check 'a closed Issue is reported and never moved' bash -c "grep -Fxq 'issue_skipped=22 reason=closed commit=$d3' '$temporary/sync' && ! grep -Eq '^(COMMENT|ISSUE|PROJECT).* 22( |$)' <(sed -n '$((before + 1)),\$p' '$state/ledger')"
issue 22 'Help text'
project_config enabled
stage_project 'Legacy Done'
expect_failure 'configured Project without its credential fails before effects' 'AXIOM_DELIVERY_PROJECT_TOKEN is unavailable' \
  env -u AXIOM_DELIVERY_PROJECT_TOKEN "$dfix/scripts/delivery-github.sh" sync --repo rgomids/axiom --from "$d1" --to "$d3"
check 'missing credential made no effect' test "$(mutations)" == "$before"
export AXIOM_DELIVERY_PROJECT_TOKEN=$FAKE_PROJECT_TOKEN
ledger_before=$(wc -l <"$state/ledger")
dgh sync --from "$d1" --to "$d3" >"$temporary/sync"
check 'Target Release is never assigned at a non-release merge' bash -c "! sed -n '$((ledger_before + 1)),\$p' '$state/ledger' | grep -q '^PROJECT target'"
check 'completed Issues are Awaiting Release with no Target Release yet' bash -c "[[ '$(item 20)' == 'Awaiting Release|' && '$(item 22)' == 'Awaiting Release|' ]]"
dgh sync --from "$d3" --to "$drel" >"$temporary/sync"
check 'release commit records Target Release for its Issue set only' bash -c "[[ '$(item 20)' == 'Awaiting Release|v0.2.0' && '$(item 21)' == 'Awaiting Release|v0.2.0' && '$(item 22)' == 'Awaiting Release|v0.2.0' ]] && [[ \$(jq -r .state '$state/issues/20.json') == open ]]"
dgh sync --to "$drel" >"$temporary/sync"
check 'sync without --from re-scans from the latest release range and converges' bash -c "grep -Fxq 'range_from=$d0' '$temporary/sync' && [[ '$(item 20)' == 'Awaiting Release|v0.2.0' ]] && [[ \$(jq length '$state/comments/20.json') == 1 ]]"

# Release notes list the delivered Issues, sanitized; RC lists none.
"$dfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.2.0 --revision "$drel" >"$temporary/dnotes"
check 'stable notes have an Issues delivered section' bash -c "grep -Fxq '### Issues delivered' '$temporary/dnotes' && grep -Fxq -- '- #20 \`Support create and edit in project-configure\`' '$temporary/dnotes' && grep -Fxq -- '- #22 \`Help text\`' '$temporary/dnotes'"
check 'untrusted Issue titles render as one code span: no mention, reference, link or HTML' grep -Fxq -- "- #21 \`Title with @someone, 'code', <b>bold</b> and [link](x)\`" "$temporary/dnotes"
cmp -s "$temporary/dnotes" <("$dfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.2.0 --revision "$drel") \
  && check 'notes with delivered Issues are deterministic' true || check 'notes with delivered Issues are deterministic' false
"$dfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.2.0-rc.1 --revision "$d3" >"$temporary/dnotes-rc"
check 'RC notes deliver no Issue' bash -c "! grep -q 'Issues delivered' '$temporary/dnotes-rc'"

# Publication: the envelope binds the Issue set, states and effects.
denvelope() {
  local dir=$1
  shift
  "$dfix/scripts/publish-release.sh" --envelope --repo rgomids/axiom --prepared-run 11 --dir "$dir/artifacts" \
    --evidence "$dir/evidence.txt" --notes "$dir/notes.md" "$@"
}
dpublish_raw() {
  local dir=$1
  shift
  "$dfix/scripts/publish-release.sh" --repo rgomids/axiom --prepared-run 11 --dir "$dir/artifacts" \
    --evidence "$dir/evidence.txt" --notes "$dir/notes.md" "$@"
}
dpublish() {
  local dir=$1
  shift
  denvelope "$dir" "$@" >"$temporary/denv" || return 1
  dpublish_raw "$dir" "$@" --authorized-digest "$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/denv")"
}
stable=(--tag v0.2.0 --revision "$drel" --make-latest true)
reset_github
stage_issues
stage_project 'Legacy Done'
make_set "$temporary/dset" 0.2.0 "$drel" delivery
cp "$temporary/dnotes" "$temporary/dset/notes.md"
denvelope "$temporary/dset" "${stable[@]}" >"$temporary/denv"
for line in envelopeVersion=2 'delivery_project=users/rgomids/projects/7 projection=enabled' delivery_issues=20,21,22 "delivery_range_base=$d0" \
  'delivery_issue.20=open delivered_by=#12' 'delivery_issue.22=open delivered_by=#13' \
  effect.issue.20=comment,project,close effect.issue.21=comment,project,close; do
  check "stable envelope states $line" grep -Fxq -- "$line" "$temporary/denv"
done
env -u AXIOM_DELIVERY_PROJECT_TOKEN "$dfix/scripts/publish-release.sh" --envelope --repo rgomids/axiom --prepared-run 11 \
  --dir "$temporary/dset/artifacts" --evidence "$temporary/dset/evidence.txt" --notes "$temporary/dset/notes.md" "${stable[@]}" >"$temporary/denv-notoken"
check 'computing the envelope needs no Project credential and makes no effect' cmp -s "$temporary/denv" "$temporary/denv-notoken"
check 'the envelope made no effect' bash -c "[[ $(mutations) == 0 ]] && ! grep -q '^PROJECT' '$state/ledger'"
ddigest=$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/denv")
issue 22 'Help text' closed completed
check 'a changed Issue state changes the preview digest' bash -c "! denvelope_out=\$('$dfix/scripts/publish-release.sh' --envelope --repo rgomids/axiom --prepared-run 11 --dir '$temporary/dset/artifacts' --evidence '$temporary/dset/evidence.txt' --notes '$temporary/dset/notes.md' ${stable[*]} | grep -Fx preview_digest=$ddigest)"
expect_failure 'authority over another Issue state is stale' 'preview changed; review and authorize again' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
check 'stale delivery authority made no effect' test "$(mutations)" == 0
issue 22 'Help text'
expect_failure 'configured Project without credential fails before any publication effect' 'AXIOM_DELIVERY_PROJECT_TOKEN is unavailable' \
  env -u AXIOM_DELIVERY_PROJECT_TOKEN "$dfix/scripts/publish-release.sh" --repo rgomids/axiom --prepared-run 11 --dir "$temporary/dset/artifacts" \
  --evidence "$temporary/dset/evidence.txt" --notes "$temporary/dset/notes.md" "${stable[@]}" --authorized-digest "$ddigest"
stage_project Blocked
expect_failure 'Project Status drift fails before any publication effect' 'Project Status must contain' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
stage_project 'Legacy Done' Ready
expect_failure 'a legacy Ready option is not accepted as a delivery state' 'Project Status must contain' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
stage_project 'Legacy Done'
jq '(.fields[] | select(.name == "Status") | .options) |= map(select(.name != "Awaiting Release"))' "$state/project.json" >"$state/p" && mv "$state/p" "$state/project.json"
expect_failure 'a Project missing an operational status fails closed' 'Project Status must contain' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
stage_project 'Legacy Done'
jq '.id = "PVT_other"' "$state/project.json" >"$state/p" && mv "$state/p" "$state/project.json"
expect_failure 'another Project node under the same number fails closed' 'is not the configured node PVT_7' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
stage_project 'Legacy Done'
jq '.title = "Other"' "$state/project.json" >"$state/p" && mv "$state/p" "$state/project.json"
expect_failure 'wrong Project fails before any publication effect' "is not titled 'Axiom Delivery'" \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
FAKE_GH_FAIL_ON='graphql projectV2' expect_failure 'Project API failure fails before any publication effect' 'cannot read Project' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
check 'delivery preflight refusals created no draft, tag or Issue effect' bash -c "[[ $(mutations) == 0 && ! -s '$state/tags' ]]"
stage_project 'Legacy Done'

# Partial: the release publishes, #20 is delivered, closing #21 fails.
FAKE_GH_FAIL_ON='PATCH repos/rgomids/axiom/issues/21' expect_failure 'Issue effect failure after publication is reported' 'cannot close Issue #21' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
check 'release was published and read back before any Issue effect' bash -c "
  first_issue=\$(grep -nE '^(COMMENT|ISSUE|PROJECT)' '$state/ledger' | head -n 1 | cut -d: -f1)
  published=\$(grep -n '^PATCH release .*\"draft\":false' '$state/ledger' | cut -d: -f1)
  [[ -n \"\$published\" && \"\$first_issue\" -gt \"\$published\" ]]"
check 'delivered Issue: release record, Released vX.Y.Z, closed completed' bash -c "
  [[ \$(jq -r '.state + \"/\" + .state_reason' '$state/issues/20.json') == closed/completed ]] && [[ '$(item 20)' == 'Released|v0.2.0' ]] &&
  jq -e '.[-1].body | startswith(\"<!-- axiom-delivery:released tag=v0.2.0 -->\") and contains(\"releases/tag/v0.2.0\")' '$state/comments/20.json'"
check 'partial state: #21 recorded but open, #22 untouched' bash -c "[[ \$(jq -r .state '$state/issues/21.json') == open && \$(jq -r .state '$state/issues/22.json') == open && ! -f '$state/comments/22.json' ]]"
before=$(mutations)
expect_failure 'the interrupted authority does not complete the partial state' 'preview changed; review and authorize again' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$ddigest"
check 'stale authority after partial delivery made no effect' test "$(mutations)" == "$before"
denvelope "$temporary/dset" "${stable[@]}" >"$temporary/denv"
for line in publication_state=published 'delivery_issue.20=closed_released delivered_by=#12' effect.issue.20=none \
  'delivery_issue.21=open_recorded delivered_by=#12' effect.issue.21=project,close effect.issue.22=comment,project,close; do
  check "recovery envelope states $line" grep -Fxq -- "$line" "$temporary/denv"
done
check 'recovery envelope has no release effect' bash -c "! grep -Eq '^effect\\.(release|assets|publish|tag)=' '$temporary/denv'"
dpublish "$temporary/dset" "${stable[@]}" >"$temporary/dpub"
check 'new authority completes delivery without duplicates' bash -c "
  grep -Fxq publication=already_published '$temporary/dpub' && grep -Fxq 'delivery=released issues=20,21,22' '$temporary/dpub' &&
  [[ \$(jq -r .state '$state/issues/21.json') == closed && \$(jq -r .state '$state/issues/22.json') == closed ]] &&
  [[ \$(jq '[.[] | select(.body | startswith(\"<!-- axiom-delivery:released\"))] | length' '$state/comments/21.json') == 1 ]] &&
  [[ '$(item 21)' == 'Released|v0.2.0' && '$(item 22)' == 'Released|v0.2.0' ]]"
before=$(mutations)
dpublish "$temporary/dset" "${stable[@]}" >"$temporary/dpub"
check 'rerun after full success is a no-op' bash -c "grep -Fxq effect=none '$temporary/denv' && [[ $(mutations) == $before ]] && grep -Fxq publication=already_published '$temporary/dpub'"
dgh verify --tag v0.2.0 --revision "$drel" >"$temporary/dverify"
check 'verify confirms every delivered Issue' grep -Fxq delivery=verified "$temporary/dverify"
before=$(mutations)
dgh sync --to "$drel" >"$temporary/sync"
check 'a sync re-run after publication never moves released Issues back' bash -c "[[ $(mutations) == $before || \$(grep -Ec '^(COMMENT|ISSUE|PROJECT)' <(sed -n '$((before + 1)),\$p' '$state/ledger') || true) == 0 ]] && grep -Fxq 'issue_skipped=20 reason=closed commit=$d2' '$temporary/sync' && [[ '$(item 20)' == 'Released|v0.2.0' ]]"

# Fresh stable publication applies every effect once, after read-back.
reset_github
stage_issues
stage_project 'Legacy Done'
dpublish "$temporary/dset" "${stable[@]}" >"$temporary/dpub"
check 'stable publication closes exactly the delivered Issues' bash -c "
  grep -Fxq publication=published '$temporary/dpub' && grep -Fxq 'delivery=released issues=20,21,22' '$temporary/dpub' &&
  [[ \$(grep -c '^ISSUE .*state=closed' '$state/ledger') == 3 && \$(grep -c '^COMMENT' '$state/ledger') == 3 ]]"
before=$(mutations)
dpublish "$temporary/dset" "${stable[@]}" >/dev/null
check 'stable delivery rerun converges' test "$(mutations)" == "$before"
check 'Project effects touch only Status and Target Release, never Legacy Done' bash -c "
  grep -q '^PROJECT status' '$state/ledger' && ! grep -E '^PROJECT (status|target) ' '$state/ledger' | grep -Ev '^PROJECT (status [^ ]+ F_STATUS|target [^ ]+ F_TARGET) ' | grep -q . &&
  ! grep -q 'Legacy Done' '$state/ledger'"

# Disabled projection (the committed state until the migration is
# authorized): Project #N is identified, but no Project read, credential or
# effect is needed; Issues are still recorded and closed.
reset_github
stage_issues
project_config disabled
denvelope "$temporary/dset" "${stable[@]}" >"$temporary/denv"
check 'disabled projection: envelope names the Project and omits Project effects' bash -c "
  grep -Fxq 'delivery_project=users/rgomids/projects/7 projection=disabled' '$temporary/denv' && grep -Fxq effect.issue.20=comment,close '$temporary/denv'"
env -u AXIOM_DELIVERY_PROJECT_TOKEN "$dfix/scripts/publish-release.sh" --repo rgomids/axiom --prepared-run 11 --dir "$temporary/dset/artifacts" \
  --evidence "$temporary/dset/evidence.txt" --notes "$temporary/dset/notes.md" "${stable[@]}" \
  --authorized-digest "$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/denv")" >"$temporary/dpub"
check 'disabled projection: Issues released and closed without any Project call' bash -c "
  grep -Fxq 'delivery=released issues=20,21,22' '$temporary/dpub' && ! grep -q '^PROJECT' '$state/ledger' && [[ \$(jq -r .state '$state/issues/20.json') == closed ]]"
dgh sync --to "$drel" >"$temporary/sync"
check 'disabled projection: sync does not reconcile the Project' bash -c "
  grep -Fxq reconcile=projection_disabled '$temporary/sync' && ! grep -q '^PROJECT' '$state/ledger'"

# Enabling projection later repairs the released Issues: the Issue and its
# release record are canonical, the Project is reconstructed from them.
project_config enabled
stage_project 'Legacy Done'
before=$(wc -l <"$state/ledger")
dgh sync --to "$drel" >"$temporary/sync"
check 'reconciliation after enabling projection: every released Issue becomes Released vX.Y.Z' bash -c "
  grep -Fxq reconcile=repaired '$temporary/sync' && grep -Fxq 'reconcile_repair=20 tag=v0.2.0 observed=absent' '$temporary/sync' &&
  [[ '$(item 20)' == 'Released|v0.2.0' && '$(item 21)' == 'Released|v0.2.0' && '$(item 22)' == 'Released|v0.2.0' ]] &&
  ! sed -n '$((before + 1)),\$p' '$state/ledger' | grep -Eq '^(COMMENT|ISSUE)'"
[[ "$(item 20)|$(item 21)|$(item 22)" == 'Released|v0.2.0|Released|v0.2.0|Released|v0.2.0' ]] \
  && check 'reconciled items read Released with Target Release v0.2.0' true || check 'reconciled items read Released with Target Release v0.2.0' false
before=$(mutations)
dgh sync --to "$drel" >"$temporary/sync"
check 'reconciliation rerun is idempotent: no Project write when consistent' bash -c "
  [[ $(mutations) == $before ]] && grep -Fxq reconcile=consistent '$temporary/sync' && grep -Fxq 'reconcile_consistent=21 tag=v0.2.0' '$temporary/sync'"
jq '.I_21.status = "Awaiting Release" | .I_21.target = "v0.1.9"' "$state/items.json" >"$state/i" && mv "$state/i" "$state/items.json"
before=$(wc -l <"$state/ledger")
dgh sync --to "$drel" >"$temporary/sync"
check 'reconciliation repairs a wrong Status/Target Release and only that item' bash -c "
  [[ '$(item 21)' == 'Released|v0.2.0' ]] && grep -Fxq 'reconcile_repair=21 tag=v0.2.0 observed=Awaiting_Release_v0.1.9' '$temporary/sync' &&
  ! sed -n '$((before + 1)),\$p' '$state/ledger' | grep -Eq 'I_(20|22)'"
[[ "$(item 21)" == 'Released|v0.2.0' ]] && check 'repaired item reads back Released v0.2.0' true || check 'repaired item reads back Released v0.2.0' false
# No repair without an unambiguous release record from the workflow bot.
jq '.I_20.status = "Awaiting Release" | .I_21.status = "Awaiting Release" | .I_22.status = "Awaiting Release"' "$state/items.json" >"$state/i" && mv "$state/i" "$state/items.json"
printf '[]\n' >"$state/comments/20.json"
printf '[{"id":1,"user":{"login":"someone"},"body":"<!-- axiom-delivery:released tag=v0.2.0 -->"}]\n' >"$state/comments/21.json"
jq '. + [{id: 9, user: {login: "github-actions[bot]"}, body: "<!-- axiom-delivery:released tag=v0.3.0 -->\\nlater"}]' "$state/comments/22.json" >"$state/c" && mv "$state/c" "$state/comments/22.json"
before=$(mutations)
dgh sync --to "$drel" >"$temporary/sync"
check 'no repair without a release record, with a foreign record, or when a later release record names another tag' bash -c "
  [[ $(mutations) == $before ]] && grep -Fxq reconcile=consistent '$temporary/sync' &&
  grep -Fxq 'reconcile_skipped=20 tag=v0.2.0 reason=no_release_record' '$temporary/sync' &&
  grep -Fxq 'reconcile_skipped=21 tag=v0.2.0 reason=no_release_record' '$temporary/sync' &&
  grep -Fxq 'reconcile_skipped=22 tag=v0.2.0 reason=latest_record_v0.3.0' '$temporary/sync'"
check 'unrepaired items keep their observed Status' bash -c "[[ \$(jq -r '.I_20.status + .I_21.status + .I_22.status' '$state/items.json') == 'Awaiting ReleaseAwaiting ReleaseAwaiting Release' ]]"
# An unreadable Issue or an unresolvable later release is reported and
# skipped; it never stops every later sync.
mv "$state/issues/21.json" "$state/issue-21.moved"
dgh sync --from "$drel" --to "$drel" >"$temporary/sync"
check 'reconciliation skips an unreadable (transferred or deleted) Issue and continues' bash -c "
  grep -Fxq 'reconcile_skipped=21 tag=v0.2.0 reason=issue_unreadable' '$temporary/sync' && grep -Fxq 'reconcile_skipped=22 tag=v0.2.0 reason=latest_record_v0.3.0' '$temporary/sync'"
mv "$state/issue-21.moved" "$state/issues/21.json"
dbad=$(dcommit 'fix: merged without metadata (#15)' 'Closes #23\n')
dr021=$(dcommit 'chore(main): release 0.2.1 (#16)' '' 0.2.1)
dgh sync --from "$dr021" --to "$dr021" >"$temporary/sync"
check 'reconciliation skips an unresolvable release and still reconciles the others' bash -c "
  grep -Fxq 'reconcile_skipped_release=v0.2.1 reason=unresolvable' '$temporary/sync' && grep -Fxq 'reconcile_skipped=20 tag=v0.2.0 reason=no_release_record' '$temporary/sync'"
git -C "$dfix" reset -q --hard "$drel"
mkdir -p "$dfix/.github"
project_config enabled
issue 20 'Reopened after release' open
printf '[{"id":1,"user":{"login":"github-actions[bot]"},"body":"<!-- axiom-delivery:released tag=v0.2.0 -->"}]\n' >"$state/comments/20.json"
dgh sync --to "$drel" >"$temporary/sync"
check 'an open Issue is never reconciled as released' grep -Fxq 'reconcile_skipped=20 tag=v0.2.0 reason=not_closed_completed' "$temporary/sync"

# A delivery state that changes after authorization stops the Issue effects.
reset_github
stage_issues
dgh state --tag v0.2.0 --revision "$drel" >"$temporary/dstate"
issue 21 'Changed title' closed completed
expect_failure 'changed delivery state stops before any Issue effect' 'delivery state changed since authorization' \
  dgh release --tag v0.2.0 --revision "$drel" --expect "$temporary/dstate"
check 'changed delivery state made no effect' test "$(mutations)" == 0

# Refused Issue states.
issue 21 'Not wanted' closed not_planned
expect_failure 'an Issue closed as not planned cannot be delivered' 'closed as not_planned' dgh state --tag v0.2.0 --revision "$drel"
printf '{"number":21,"node_id":"PR_21","title":"a PR","state":"open","pull_request":{}}\n' >"$state/issues/21.json"
expect_failure 'a pull request is not an Issue' '#21 is a pull request' dgh state --tag v0.2.0 --revision "$drel"
rm "$state/issues/21.json"
expect_failure 'a missing Issue fails closed' 'Issue #21 cannot be read' dgh state --tag v0.2.0 --revision "$drel"
issue 21 'Released before' closed completed
printf '[{"id":1,"user":{"login":"github-actions[bot]"},"body":"<!-- axiom-delivery:released tag=v0.1.0 -->\\nold"}]\n' >"$state/comments/21.json"
expect_failure 'an Issue already released by another tag is not released again' 'already released by another tag' dgh state --tag v0.2.0 --revision "$drel"
printf '[{"id":1,"user":{"login":"someone"},"body":"<!-- axiom-delivery:released tag=v0.2.0 -->"}]\n' >"$state/comments/21.json"
dgh state --tag v0.2.0 --revision "$drel" >"$temporary/dstate"
check 'release records from other authors are ignored' grep -Fxq 'delivery_issue.21=closed_unrecorded delivered_by=#12' "$temporary/dstate"
check 'a closed unrecorded Issue is recorded, never reopened or re-closed' grep -Fxq effect.issue.21=comment,project "$temporary/dstate"

# The release record is written only after the Project reads back: a Project
# failure leaves no marker, so the next envelope still carries the effect,
# including for an Issue that was already closed (legacy reconciliation).
reset_github
stage_issues
stage_project 'Legacy Done'
issue 21 'Closed by a legacy keyword' closed completed
denvelope "$temporary/dset" "${stable[@]}" >"$temporary/denv"
check 'a closed unrecorded Issue is recorded, not re-closed' grep -Fxq effect.issue.21=comment,project "$temporary/denv"
FAKE_GH_FAIL_ON='graphql updateProjectV2ItemFieldValue' expect_failure 'Project failure after publication is reported' 'cannot set Project Status' \
  dpublish_raw "$temporary/dset" "${stable[@]}" --authorized-digest "$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/denv")"
check 'no release record without its Project read-back' bash -c "! grep -q '^COMMENT' '$state/ledger' && [[ \$(jq -r .state '$state/issues/20.json') == open ]]"
dpublish "$temporary/dset" "${stable[@]}" >"$temporary/dpub"
check 'the recovery envelope completes the closed Issue Project effect' bash -c "
  grep -Fxq effect.issue.21=comment,project '$temporary/denv' && [[ '$(item 21)' == 'Released|v0.2.0' ]] &&
  jq -e '.[-1].body | contains(\"already closed before this release\")' '$state/comments/21.json' && ! grep -q '^ISSUE 21 ' '$state/ledger'"
# A re-opened released Issue is not moved back by its released delivery.
issue 20 'Support create and edit in project-configure'
before=$(mutations)
dgh sync --to "$drel" >"$temporary/sync"
check 'sync never moves a re-opened released Issue back to Awaiting Release' bash -c "
  grep -Fxq 'issue_skipped=20 reason=already_released commit=$d2' '$temporary/sync' && [[ '$(item 20)' == 'Released|v0.2.0' ]] &&
  ! sed -n '$((before + 1)),\$p' '$state/ledger' | grep -Eq '^(COMMENT 20|PROJECT .*I_20)'"

# Development-sidebar race fail-safe: an Issue GitHub closed at the merge
# that completes it is reopened only when its closer is exactly that merged
# PR/commit and no stable release recorded it; anything else is untouched.
closer() {
  mkdir -p "$state/closed"
  jq -n --arg kind "$2" --arg number "${3:-0}" --arg sha "${4:-}" --arg reason "${5:-COMPLETED}" '
    {data: {repository: {issue: {timelineItems: {nodes: [{createdAt: "2026-10-01T06:00:00Z", stateReason: $reason, closer:
      (if $kind == "pr" then {__typename: "PullRequest", number: ($number | tonumber), merged: true, mergeCommit: {oid: $sha}}
       elif $kind == "commit" then {__typename: "Commit", oid: $sha} else null end)}]}}}}}' >"$state/closed/$1.json"
}
reset_github
stage_issues
stage_project 'Legacy Done'
issue 20 'Closed by the sidebar link at merge' closed completed
closer 20 pr 12 "$d2"
issue 21 'Closed manually by a human' closed completed
closer 21 none
dgh sync --from "$d1" --to "$d2" >"$temporary/sync"
check 'fail-safe: a closure attributable to the exact merged PR is reopened with a bounded record' bash -c "
  [[ \$(jq -r .state '$state/issues/20.json') == open ]] && grep -Fxq 'effect=issue_reopened issue=20 commit=$d2' '$temporary/sync' &&
  jq -e '.[0].body | startswith(\"<!-- axiom-delivery:reopened commit=$d2 -->\") and contains(\"#12\") and (length < 400)' '$state/comments/20.json' >/dev/null"
check 'fail-safe: the record precedes the reopen, then the normal Awaiting Release projection continues' bash -c "
  r=\$(grep -n '^COMMENT 20 <!-- axiom-delivery:reopened' '$state/ledger' | cut -d: -f1)
  o=\$(grep -n '^ISSUE 20 state=open state_reason=reopened' '$state/ledger' | cut -d: -f1)
  c=\$(grep -n '^COMMENT 20 <!-- axiom-delivery:completed commit=$d2' '$state/ledger' | cut -d: -f1)
  [[ -n \"\$r\" && -n \"\$o\" && -n \"\$c\" && \$r -lt \$o && \$o -lt \$c ]] && [[ '$(item 20)' == 'Awaiting Release|' ]]"
check 'fail-safe: a closure not attributable to the merge is never reopened' bash -c "
  [[ \$(jq -r .state '$state/issues/21.json') == closed ]] && grep -Fxq 'issue_skipped=21 reason=closed commit=$d2' '$temporary/sync' &&
  ! grep -Eq '^(COMMENT|ISSUE) 21 ' '$state/ledger'"
[[ "$(item 20)" == 'Awaiting Release|' ]] && check 'fail-safe: reopened Issue is Awaiting Release' true || check 'fail-safe: reopened Issue is Awaiting Release' false
before=$(grep -Ec '^(COMMENT|ISSUE)' "$state/ledger" || true)
dgh sync --from "$d1" --to "$d2" >"$temporary/sync"
check 'fail-safe rerun is idempotent: no second record, reopen or comment' bash -c "
  [[ \$(grep -Ec '^(COMMENT|ISSUE)' '$state/ledger') == $before ]] && ! grep -q '^effect=issue_reopened' '$temporary/sync'"
for case in 'pr 99 '"$d3"' COMPLETED|another merged PR' 'pr 13 '"$d2"' COMPLETED|the PR number with another merge commit' \
  'pr 13 '"$d3"' NOT_PLANNED|a not-planned closure' 'commit 0 '"$d2"' COMPLETED|another commit' 'none 0 x COMPLETED|an unknown closer'; do
  read -r kind number sha reason <<<"${case%%|*}"
  issue 22 'Help text' closed "$([[ "$reason" == NOT_PLANNED ]] && printf not_planned || printf completed)"
  closer 22 "$kind" "$number" "$sha" "$reason"
  before=$(grep -Ec '^(COMMENT|ISSUE) 22 ' "$state/ledger" || true)
  dgh sync --from "$d2" --to "$d3" >"$temporary/sync"
  check "fail-safe: ${case#*|} is never reopened" bash -c "
    [[ \$(jq -r .state '$state/issues/22.json') == closed && \$(grep -Ec '^(COMMENT|ISSUE) 22 ' '$state/ledger' || true) == $before ]]"
done
issue 22 'Help text' closed completed
closer 22 pr 13 "$d3"
printf '[{"id":1,"user":{"login":"github-actions[bot]"},"created_at":"2026-10-01T09:00:00Z","body":"<!-- axiom-delivery:released tag=v0.2.0 -->\\npublished"}]\n' >"$state/comments/22.json"
dgh sync --from "$d2" --to "$drel" >"$temporary/sync"
check 'fail-safe: an Issue the delivering stable release recorded is never reopened' bash -c "
  [[ \$(jq -r .state '$state/issues/22.json') == closed ]] && ! grep -q '^ISSUE 22 ' '$state/ledger'"
dgh sync --from "$d1" --to "$d3" >"$temporary/sync"
check 'fail-safe: a stale rerun whose window ends before the release commit never reopens a published Issue' bash -c "
  [[ \$(jq -r .state '$state/issues/22.json') == closed ]] && ! grep -q '^ISSUE 22 ' '$state/ledger' && grep -Fxq 'issue_skipped=22 reason=closed commit=$d3' '$temporary/sync'"
printf '[{"id":1,"user":{"login":"github-actions[bot]"},"created_at":"2026-09-01T00:00:00Z","body":"<!-- axiom-delivery:released tag=v0.1.0 -->\\nold"}]\n' >"$state/comments/22.json"
dgh sync --from "$d2" --to "$d3" >"$temporary/sync"
check 'fail-safe: a record of an earlier release (earlier delivery of a reopened Issue) does not block' bash -c "
  [[ \$(jq -r .state '$state/issues/22.json') == open ]] && grep -Fxq 'effect=issue_reopened issue=22 commit=$d3' '$temporary/sync'"
issue 22 'Help text' closed completed
rm "$state/comments/22.json"
before=$(wc -l <"$state/ledger")
FAKE_GH_RELEASE_ON_READ=22:v0.2.0:3 dgh sync --from "$d2" --to "$drel" >"$temporary/sync"
check 'fail-safe: a release recorded between planning and the reopen wins: no reopen, comment or Awaiting Release' bash -c "
  [[ \$(jq -r .state '$state/issues/22.json') == closed ]] && grep -Fxq 'issue_skipped=22 reason=already_released commit=$d3' '$temporary/sync' &&
  ! sed -n '$((before + 1)),\$p' '$state/ledger' | grep -Eq '^(ISSUE 22|COMMENT 22|PROJECT status PVTI_I_22 F_STATUS Awaiting Release)' &&
  [[ '$(item 22)' == 'Released|v0.2.0' ]]"
rm -f "$state/comments/22.json" "$state/reads-22"
for replacement in manual other_merge changed_event state_open not_planned; do
  issue 22 'Help text' closed completed
  closer 22 pr 13 "$d3"
  cp "$state/closed/22.json" "$state/closure-after-22.json"
  cp "$state/issues/22.json" "$state/issue-after-22.json"
  expected_state=closed
  case "$replacement" in
    manual) expression='.data.repository.issue.timelineItems.nodes[0].closer = null' ;;
    other_merge) expression=".data.repository.issue.timelineItems.nodes[0].closer.mergeCommit.oid = \"$d2\"" ;;
    changed_event) expression='.data.repository.issue.timelineItems.nodes[0].createdAt = "2026-10-01T07:00:00Z"' ;;
    state_open)
      expression='.'
      expected_state=open
      jq '.state = "open" | .state_reason = "reopened"' "$state/issues/22.json" >"$state/issue-after-22.json"
      ;;
    not_planned)
      expression='.'
      jq '.state_reason = "not_planned"' "$state/issues/22.json" >"$state/issue-after-22.json"
      ;;
  esac
  jq "$expression" "$state/closure-after-22.json" >"$state/closure-after-22.new"
  mv "$state/closure-after-22.new" "$state/closure-after-22.json"
  rm -f "$state/closure-reads-22" "$state/comments/22.json"
  before=$(wc -l <"$state/ledger")
  FAKE_GH_CLOSURE_ON_READ=22:2 dgh sync --from "$d2" --to "$d3" >"$temporary/sync"
  check "fail-safe: $replacement closure replacing the planned event prevents all reopen effects" bash -c "
    [[ \$(jq -r .state '$state/issues/22.json') == '$expected_state' ]] &&
    grep -Fxq 'issue_skipped=22 reason=closure_changed commit=$d3' '$temporary/sync' &&
    ! sed -n '$((before + 1)),\$p' '$state/ledger' | grep -Eq '^(ISSUE 22|COMMENT 22|PROJECT .*PVTI_I_22)'"
done
issue 22 'Help text' closed completed
rm -f "$state/comments/22.json"
closer 22 pr 13 "$d3"
FAKE_GH_FAIL_ON='graphql timelineItems' dgh sync --from "$d2" --to "$d3" >"$temporary/sync" 2>"$temporary/err"
check 'fail-safe: an unreadable closer is not attributable and never stops the projection' bash -c "
  [[ \$(jq -r .state '$state/issues/22.json') == closed ]] && grep -Fxq 'closer_unreadable=22 commit=$d3' '$temporary/sync' &&
  grep -Fxq 'issue_skipped=22 reason=closed commit=$d3' '$temporary/sync'"
rm -f "$state/comments/22.json"
closer 22 commit 0 "$d3"
FAKE_GH_FAIL_ON='PATCH repos/rgomids/axiom/issues/22' expect_failure 'fail-safe: a failed reopen is reported' 'cannot reopen Issue #22' \
  dgh sync --from "$d2" --to "$d3"
check 'fail-safe: the bounded record exists, the Issue is still closed' bash -c "
  [[ \$(jq -r .state '$state/issues/22.json') == closed ]] && [[ \$(jq length '$state/comments/22.json') == 1 ]]"
dgh sync --from "$d2" --to "$d3" >"$temporary/sync"
check 'fail-safe: a rerun after a failed reopen converges without a duplicate record (commit closer)' bash -c "
  [[ \$(jq -r .state '$state/issues/22.json') == open ]] &&
  [[ \$(jq '[.[] | select(.body | startswith(\"<!-- axiom-delivery:reopened\"))] | length' '$state/comments/22.json') == 1 ]]"
# Premature closure and the Release PR merge in one sync window: the
# reopened Issue also receives the release's Target Release.
issue 20 'Closed by the sidebar link at merge' closed completed
closer 20 pr 12 "$d2"
rm -f "$state/comments/20.json"
dgh sync --from "$d1" --to "$drel" >"$temporary/sync"
check 'fail-safe in the release window: reopened and Awaiting Release with Target Release' bash -c "
  [[ \$(jq -r .state '$state/issues/20.json') == open ]] && [[ '$(item 20)' == 'Awaiting Release|v0.2.0' ]]"
[[ "$(item 20)" == 'Awaiting Release|v0.2.0' ]] && check 'fail-safe in the release window: Target Release reads back' true || check 'fail-safe in the release window: Target Release reads back' false
rm -rf "$state/closed"
check 'reconciliation and undeclared-commit enforcement share the v0.2.0 boundary' bash -c "
  grep -Fxq 'legacy_boundary=0.2.0' '$repository_root/scripts/delivery-issues.sh' &&
  grep -Fq 'exit !(a[1] > 0 || a[2] >= 2)' '$repository_root/scripts/delivery-github.sh'"

# v0.2.0 migration boundary: two reviewed legacy records for Issues already
# closed by keywords. The notes list both; publication records them and never
# closes them again; #132 stays out.
lfix=$temporary/legacy-fixture
git init -q -b main "$lfix"
git -C "$lfix" config user.email release-test@example.invalid
git -C "$lfix" config user.name 'Release Test'
git -C "$lfix" config commit.gpgsign false
mkdir -p "$lfix/scripts" "$lfix/.github"
cp "$dfix"/scripts/*.sh "$lfix/scripts/"
lcommit() {
  [[ -z "${3:-}" ]] || printf '{\n  ".": "%s"\n}\n' "$3" >"$lfix/.release-please-manifest.json"
  git -C "$lfix" add -A
  printf '%s\n\n%b' "$1" "$2" | git -C "$lfix" commit -q --allow-empty -F -
  git -C "$lfix" rev-parse HEAD
}
printf '# Changelog\n' >"$lfix/CHANGELOG.md"
lcommit 'chore(main): release 0.1.2 (#121)' '' 0.1.2 >/dev/null
lcommit 'feat(project): EDIT preview (I132-T01) (#145)' 'Partial work for #132.\n' >/dev/null
l143=$(lcommit 'feat(project): list configured projects (#143)' 'Implements issue #129\n\nCloses #129\n')
l148=$(lcommit 'fix(work-items): resolve portable Project from recorded SourceLocation (#147) (#148)' '- Closes #147.\n')
printf '%s related=129 completes=129\n%s related=147 completes=147\n' "$l143" "$l148" >"$lfix/.github/delivery-corrections.txt"
jq -n '{schemaVersion: 2, owner: "rgomids", number: 7, projectId: "PVT_7", title: "Axiom Delivery",
  projection: "enabled", migrationStatuses: ["Legacy Done"]}' >"$lfix/.github/delivery-project.json"
lcommit 'ci(delivery): adopt the delivery contract (#160)' 'Related-Issues: none\nCompletes-Issues: none\n' >/dev/null
printf '# Changelog\n\n## [0.2.0](https://github.com/rgomids/axiom/compare/v0.1.2...v0.2.0) (2026-10-01)\n\n\n### Features\n\n* **project:** list configured projects\n' >"$lfix/CHANGELOG.md"
lrel=$(lcommit 'chore(main): release 0.2.0 (#146)' 'Release PR body.\n' 0.2.0)
reset_github
stage_project 'Legacy Done'
issue 129 'List configured Projects' closed completed
issue 147 'resolve portable Project from recorded SourceLocation' closed completed
issue 132 'Support create and edit in project-configure'
"$lfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.2.0 --revision "$lrel" >"$temporary/lnotes"
check 'boundary notes list exactly #129 and #147' bash -c "
  grep -Fxq -- '- #129 \`List configured Projects\`' '$temporary/lnotes' &&
  grep -Fxq -- '- #147 \`resolve portable Project from recorded SourceLocation\`' '$temporary/lnotes' &&
  [[ \$(sed -n '/^### Issues delivered/,/^---/p' '$temporary/lnotes' | grep -c '^- #') == 2 ]] && ! grep -q '#132' '$temporary/lnotes'"
make_set "$temporary/lset" 0.2.0 "$lrel" legacy
cp "$temporary/lnotes" "$temporary/lset/notes.md"
lenvelope() {
  "$lfix/scripts/publish-release.sh" --envelope --repo rgomids/axiom --prepared-run 11 --dir "$temporary/lset/artifacts" \
    --evidence "$temporary/lset/evidence.txt" --notes "$temporary/lset/notes.md" --tag v0.2.0 --revision "$lrel" --make-latest true
}
lenvelope >"$temporary/lenv"
for line in delivery_issues=129,147 'delivery_issue.129=closed_unrecorded delivered_by=#143' effect.issue.129=comment,project \
  'delivery_issue.147=closed_unrecorded delivered_by=#148' effect.issue.147=comment,project; do
  check "boundary envelope states $line" grep -Fxq -- "$line" "$temporary/lenv"
done
"$lfix/scripts/publish-release.sh" --repo rgomids/axiom --prepared-run 11 --dir "$temporary/lset/artifacts" \
  --evidence "$temporary/lset/evidence.txt" --notes "$temporary/lset/notes.md" --tag v0.2.0 --revision "$lrel" --make-latest true \
  --authorized-digest "$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/lenv")" >"$temporary/lpub"
check 'boundary publication records #129 and #147 once, Released vX.Y.Z, never re-closes' bash -c "
  grep -Fxq 'delivery=released issues=129,147' '$temporary/lpub' &&
  [[ \$(grep -c '^COMMENT 129 ' '$state/ledger') == 1 && \$(grep -c '^COMMENT 147 ' '$state/ledger') == 1 ]] &&
  ! grep -q '^ISSUE ' '$state/ledger' && [[ '$(item 129)' == 'Released|v0.2.0' && '$(item 147)' == 'Released|v0.2.0' ]] &&
  ! grep -Eq '^(COMMENT|PROJECT .*I_)132' '$state/ledger' && [[ \$(jq -r .state '$state/issues/132.json') == open ]]"
before=$(mutations)
lenvelope >"$temporary/lenv"
check 'boundary rerun converges: both recorded, no effect' bash -c "
  grep -Fxq effect.issue.129=none '$temporary/lenv' && grep -Fxq effect.issue.147=none '$temporary/lenv' && [[ $(mutations) == $before ]]"

# Release PR path of the required delivery-metadata check: Release Please
# PRs start no pull_request run, so release-please.yml dispatches the check;
# it passes only for the bot-authored Release PR head.
reset_github
rp_head=$(printf 'a%.0s' {1..40})
jq -n --arg sha "$rp_head" '[{number: 146, user: {login: "github-actions[bot]"}, base: {ref: "main"},
  head: {ref: "release-please--branches--main", sha: $sha, repo: {full_name: "rgomids/axiom", owner: {login: "rgomids"}}}}]' >"$state/open-pulls.json"
rph() { "$dfix/scripts/delivery-github.sh" release-pr-head --repo rgomids/axiom "$@"; }
rph --branch release-please--branches--main --sha "$rp_head" --actor 'github-actions[bot]' >"$temporary/rph"
check 'Release PR head dispatched by the bot passes without validating metadata' bash -c "
  grep -Fxq 'release_pr=#146 head=$rp_head' '$temporary/rph' && grep -Fxq delivery_metadata=not_applicable '$temporary/rph' && [[ $(mutations) == 0 ]]"
printf '[129]\n' >"$state/closing-refs"
expect_failure 'a Release PR that would close an Issue at merge fails' 'the Release PR would close #129 at merge' \
  rph --branch release-please--branches--main --sha "$rp_head" --actor 'github-actions[bot]'
rm -f "$state/closing-refs"
expect_failure 'a human dispatch never produces a passing check' 'only github-actions[bot] dispatches' \
  rph --branch release-please--branches--main --sha "$rp_head" --actor rgomids
expect_failure 'a dispatch on another branch fails' 'runs only on a release-please-- branch' \
  rph --branch feature/x --sha "$rp_head" --actor 'github-actions[bot]'
expect_failure 'a dispatch for another commit than the Release PR head fails' 'is not the head of one open bot-authored Release PR' \
  rph --branch release-please--branches--main --sha "$(printf 'b%.0s' {1..40})" --actor 'github-actions[bot]'
jq '.[0].user.login = "someone"' "$state/open-pulls.json" >"$state/p" && mv "$state/p" "$state/open-pulls.json"
expect_failure 'a release-please-- branch PR not authored by the bot fails' 'is not the head of one open bot-authored Release PR' \
  rph --branch release-please--branches--main --sha "$rp_head" --actor 'github-actions[bot]'
jq '.[0].user.login = "github-actions[bot]" | .[0].head.repo = {full_name: "fork/axiom", owner: {login: "fork"}}' "$state/open-pulls.json" >"$state/p" && mv "$state/p" "$state/open-pulls.json"
expect_failure 'a fork head with the same branch name fails' 'is not the head of one open bot-authored Release PR' \
  rph --branch release-please--branches--main --sha "$rp_head" --actor 'github-actions[bot]'
rm -f "$state/open-pulls.json"
expect_failure 'no open Release PR fails' 'is not the head of one open bot-authored Release PR' \
  rph --branch release-please--branches--main --sha "$rp_head" --actor 'github-actions[bot]'

# Release candidates never touch Issues.
reset_github
stage_issues
make_set "$temporary/drc" 0.2.0-rc.1 "$d3" rc
"$dfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.2.0-rc.1 --revision "$d3" >"$temporary/drc/notes.md"
denvelope "$temporary/drc" --tag v0.2.0-rc.1 --revision "$d3" --make-latest false >"$temporary/denv"
check 'RC envelope delivers no Issue' grep -Fxq delivery_issues=not_applicable "$temporary/denv"
dpublish "$temporary/drc" --tag v0.2.0-rc.1 --revision "$d3" --make-latest false >"$temporary/dpub"
check 'RC publication leaves every Issue open and untouched' bash -c "
  grep -Fxq publication=published '$temporary/dpub' && grep -Fxq delivery=not_applicable '$temporary/dpub' &&
  ! grep -Eq '^(COMMENT|ISSUE|PROJECT)' '$state/ledger' && [[ \$(jq -r .state '$state/issues/20.json') == open ]]"
project_config disabled
unset AXIOM_DELIVERY_PROJECT_TOKEN

# --- 4. release.sh: prepare, envelope and authority boundary ---------------------------------------
release() { (cd "$fixture" && "$fixture/scripts/release.sh" "$@"); }
green() {
  printf '{"check_runs":[{"name":"verify (linux)","status":"completed","conclusion":"success","started_at":"1"},{"name":"verify (macos)","status":"completed","conclusion":"success","started_at":"1"},{"name":"release-contract","status":"completed","conclusion":"success","started_at":"1"}]}\n' >"$state/checks-$1.json"
}
# stage_prepared_run ID TAG REVISION SALT [WORKFLOW] places a prepared set, as
# release-artifacts.yml retains it, behind the fake `gh run download`.
stage_prepared_run() {
  local id=$1 tag=$2 rev=$3 salt=$4 path=${5:-.github/workflows/release-artifacts.yml} root
  root=$state/runs/$id/axiom-release-$tag
  rm -rf -- "$state/runs/$id"
  make_set "$temporary/run-$id" "${tag#v}" "$rev" "$salt"
  mkdir -p "$root"
  cp -R "$temporary/run-$id/artifacts" "$root/artifacts"
  cp "$temporary/run-$id/evidence.txt" "$root/release-evidence.txt"
  "$fixture/scripts/release-notes.sh" --tag "$tag" --revision "$rev" --repo rgomids/axiom >"$root/release-notes.md"
  jq -n --arg path "$path" '{path: $path, event: "workflow_dispatch", head_branch: "main", status: "completed", conclusion: "success"}' \
    >"$state/runs/$id.json"
}
reset_github
printf '{"protection_rules":[{"type":"required_reviewers"}]}\n' >"$state/environment.json"
green "$c2"
green "$c3"
green "$c4"

release status >"$temporary/status"
check 'no Release PR while main records an unpublished release: no new release starts' bash -c "grep -Fxq state=no_release_in_progress '$temporary/status' && grep -Fxq next_action=blocked '$temporary/status' && grep -Fq 'release v0.1.0 is recorded on main but not published' '$temporary/status'"
printf '[{"number":8,"url":"https://github.com/rgomids/axiom/pull/8","title":"chore(main): release 0.1.0","mergeCommit":{"oid":"%s"}}]\n' "$c3" >"$state/pr-merged.json"
release status >"$temporary/status"
check 'merged Release PR resolves the stable tag and asks to prepare its release commit' bash -c "grep -Fxq tag=v0.1.0 '$temporary/status' && grep -Fxq revision=$c3 '$temporary/status' && grep -Fxq next_action=prepare '$temporary/status'"
check 'status reports repository, CI and environment facts' bash -c "grep -Fxq main_ci=success '$temporary/status' && grep -Fxq release_environment=protected '$temporary/status' && grep -Fxq worktree=clean '$temporary/status'"
rm "$state/pr-merged.json"

release status --tag v0.1.0-rc.1 >"$temporary/status"
check 'RC status asks to prepare main first' bash -c "grep -Fxq next_action=prepare '$temporary/status' && grep -Fxq revision=$c4 '$temporary/status' && ! grep -q '^preview_digest=' '$temporary/status'"
check 'status made no remote effect' test "$(mutations)" == 0

stage_prepared_run 5151 v0.1.0-rc.1 "$c4" first
release prepare --tag v0.1.0-rc.1 >"$temporary/prepare"
check 'prepare dispatches only the preparation workflow' bash -c "[[ \$(grep -c '^workflow run release-artifacts.yml' '$state/ledger') == 1 ]] && grep -Fq 'revision=$c4' '$state/ledger'"
check 'prepare makes no release, tag, asset or publication effect' test "$(release_effects)" == 0
check 'prepare stops at the authority boundary' bash -c "grep -Fxq next_action=authorize_publication '$temporary/prepare' && grep -Fxq prepared_run=5151 '$temporary/prepare'"
for key in tag=v0.1.0-rc.1 "revision=$c4" channel=rc prerelease=true make_latest=false prepared_run=5151 publication_state=absent; do
  check "prepared envelope states $key" grep -Fxq "preview.$key" "$temporary/prepare"
done
check 'prepared envelope states notes, SHA256SUMS and every artifact digest' bash -c "grep -Eq '^preview\\.release_notes_sha256=[0-9a-f]{64}$' '$temporary/prepare' && grep -Eq '^preview\\.sha256sums_sha256=[0-9a-f]{64}$' '$temporary/prepare' && [[ \$(grep -Ec '^preview\\.artifact\\.axiom-0\\.1\\.0-rc\\.1-[a-z0-9.-]+\\.tar\\.gz=[0-9a-f]{64}$' '$temporary/prepare') == 3 ]]"
digest_value=$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/prepare")
release status --tag v0.1.0-rc.1 --prepared-run 5151 >"$temporary/status"
check 'the envelope of a prepared run is deterministic' grep -Fxq "preview_digest=$digest_value" "$temporary/status"

: >"$state/ledger"
expect_failure 'publish without authorization is refused' 'requires explicit human authorization' \
  release publish --tag v0.1.0-rc.1 --revision "$c4" --prepared-run 5151 --preview-digest "$digest_value"
expect_failure 'publish without a given or discoverable prepared run is refused' 'publication is not the next step' \
  release publish --tag v0.1.0-rc.1 --revision "$c4" --preview-digest "$digest_value" --authorize-publication
expect_failure 'publish without the preview digest is refused' 'preview-digest' \
  release publish --tag v0.1.0-rc.1 --revision "$c4" --prepared-run 5151 --authorize-publication
expect_failure 'publish with a stale preview is refused' 'preview changed; review and authorize again' \
  release publish --tag v0.1.0-rc.1 --revision "$c4" --prepared-run 5151 --preview-digest "$(printf '0%.0s' {1..64})" --authorize-publication
stage_prepared_run 5151 v0.1.0-rc.1 "$c4" second
expect_failure 'authority does not follow changed prepared bytes' 'preview changed; review and authorize again' \
  release publish --tag v0.1.0-rc.1 --revision "$c4" --prepared-run 5151 --preview-digest "$digest_value" --authorize-publication
stage_prepared_run 5151 v0.1.0-rc.1 "$c4" first
stage_prepared_run 6161 v0.1.0-rc.1 "$c4" first .github/workflows/other.yml
release status --tag v0.1.0-rc.1 --prepared-run 6161 >"$temporary/status"
check 'a set from another workflow is refused' bash -c "grep -Fxq next_action=blocked '$temporary/status' && grep -Fq 'not a successful release-artifacts.yml' '$temporary/status'"
stage_prepared_run 7171 v0.1.0-rc.1 "$c4" first
printf 'x\n' >>"$state/runs/7171/axiom-release-v0.1.0-rc.1/artifacts/axiom-0.1.0-rc.1-linux-amd64.tar.gz"
release status --tag v0.1.0-rc.1 --prepared-run 7171 >"$temporary/status"
check 'a tampered prepared set is refused' bash -c "grep -Fxq next_action=blocked '$temporary/status' && grep -Fq 'checksum mismatch' '$temporary/status'"
stage_prepared_run 8181 v0.1.0-rc.1 "$c4" first
printf 'edited\n' >>"$state/runs/8181/axiom-release-v0.1.0-rc.1/release-notes.md"
release status --tag v0.1.0-rc.1 --prepared-run 8181 >"$temporary/status"
check 'prepared notes must be the revision notes' grep -Fq 'release notes differ' "$temporary/status"
rm "$state/environment.json"
expect_failure 'unprotected release environment blocks publication' 'publication is not the next step' \
  release publish --tag v0.1.0-rc.1 --revision "$c4" --prepared-run 5151 --preview-digest "$digest_value" --authorize-publication
printf '{"protection_rules":[{"type":"required_reviewers"}]}\n' >"$state/environment.json"
rm "$state/checks-$c4.json"
release status --tag v0.1.0-rc.1 >"$temporary/status"
check 'missing CI blocks publication' bash -c "grep -Fxq next_action=blocked '$temporary/status' && grep -Fq 'required CI' '$temporary/status'"
green "$c4"
check 'refusals caused no remote effect' test "$(mutations)" == 0
release publish --tag v0.1.0-rc.1 --revision "$c4" --prepared-run 5151 --preview-digest "$digest_value" --authorize-publication >"$temporary/dispatch"
check 'the exact authorized envelope dispatches publication once' bash -c "[[ \$(grep -c '^workflow run publish-release.yml' '$state/ledger') == 1 ]] && grep -Fq 'prepared_run=5151' '$state/ledger' && grep -Fq 'preview_digest=$digest_value' '$state/ledger' && grep -Fq 'revision=$c4' '$state/ledger' && grep -Fxq run_id=4242 '$temporary/dispatch'"
check 'release.sh itself never touches releases, tags or assets' test "$(grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL)' "$state/ledger" || true)" == 0
release status --tag v0.2.0 >"$temporary/status"
check 'stable without a Release PR is blocked' bash -c "grep -Fxq next_action=blocked '$temporary/status' && grep -Fq 'no Release PR prepares 0.2.0' '$temporary/status'"
release status --tag v0.1.0-rc.1 --revision "$side" >"$temporary/status"
check 'preflight refusal is reported as blocked' grep -Fq 'reason=revision is not on the first-parent history of main' "$temporary/status"

# Published: verify reports the release instead of publishing again.
make_set "$temporary/rc" 0.1.0-rc.1 "$c4" first
publish "$temporary/rc" --tag v0.1.0-rc.1 --revision "$c4" --make-latest false >/dev/null
remote_tag v0.1.0-rc.1 "$c4"
release status --tag v0.1.0-rc.1 >"$temporary/status"
check 'published release routes to verification' grep -Fxq next_action=verify_published "$temporary/status"
release verify --tag v0.1.0-rc.1 >"$temporary/verify"
check 'verify confirms the published prerelease' bash -c "grep -Fxq publication_state=published '$temporary/verify' && grep -Fxq latest=none '$temporary/verify' && grep -Fxq result=pass '$temporary/verify'"
remote_untag v0.1.0-rc.1
make_set "$temporary/orphan" 0.1.0-rc.2 "$c4" orphan
orphan_release 399339376 v0.1.0-rc.2 "$c4" "$temporary/orphan"
: >"$state/ledger"
release status --tag v0.1.0-rc.2 >"$temporary/status"
check 'status reports the orphan release as a blocking conflict, never absent' bash -c "grep -Fxq next_action=blocked '$temporary/status' && grep -Fxq publication_state=orphan_conflict '$temporary/status' && grep -Fxq release_id=399339376 '$temporary/status' && grep -Fq 'reason=orphan_conflict: release 399339376' '$temporary/status' && ! grep -Fxq publication_state=absent '$temporary/status'"
stage_prepared_run 9191 v0.1.0-rc.2 "$c4" orphan
release status --tag v0.1.0-rc.2 --prepared-run 9191 >"$temporary/status"
check 'status offers no envelope while the orphan exists' bash -c "grep -Fxq next_action=blocked '$temporary/status' && ! grep -q '^preview_digest=' '$temporary/status'"
expect_failure 'publish is refused while the orphan exists' 'publication is not the next step' \
  release publish --tag v0.1.0-rc.2 --revision "$c4" --prepared-run 9191 --preview-digest "$(printf 'a%.0s' {1..64})" --authorize-publication
check 'orphan status and refusal made no remote effect' test "$(mutations)" == 0
rm "$state/releases/399339376.json"
printf 'dirty\n' >"$fixture/dirty.txt"
release status >"$temporary/status"
check 'dirty worktree is reported' grep -Fxq worktree=dirty "$temporary/status"
rm "$fixture/dirty.txt"

# --- 5. Workflow and skill static contracts ------------------------------------------------------
workflows=$repository_root/.github/workflows
triggers() { sed -n '/^on:/,/^[a-z]/p' "$1" | grep -E '^  [a-z_]+:' | tr -d ' :' | LC_ALL=C sort | paste -sd, -; }
check 'CI runs on pull requests, pushes to main and dispatch only' test "$(triggers "$workflows/ci.yml")" == pull_request,push,workflow_dispatch
check 'CI push trigger is main only' bash -c "sed -n '/^  push:/,/^  [a-z]/p' '$workflows/ci.yml' | grep -Fxq '      - main'"
check 'CI has a read-only token and no publication path' bash -c "grep -A1 '^permissions:' '$workflows/ci.yml' | tail -n 1 | grep -Fxq '  contents: read' && ! grep -Eiq 'contents: write|gh release|git tag|git push|publish-release|secrets\\.|id-token' '$workflows/ci.yml'"
check 'Release PR workflow never creates tags or releases' bash -c "grep -Fxq '          skip-github-release: true' '$workflows/release-please.yml' && [[ \$(jq -r '.\"skip-github-release\"' '$repository_root/release-please-config.json') == true ]] && ! grep -Eiq 'gh release|git tag|git push|softprops|secrets\\.' '$workflows/release-please.yml'"
config=$repository_root/release-please-config.json
check 'Release Please uses its default versioning with pre-1.0 bumps as documented' bash -c "jq -e '(.packages[\".\"] | .\"bump-minor-pre-major\" == true and .\"bump-patch-for-minor-pre-major\" == false and (has(\"versioning\") | not) and .\"initial-version\" == \"0.1.0\") and (has(\"versioning\") | not)' '$config' >/dev/null"
check 'visible changelog types are exactly the documented release triggers' test "$(jq -r '.packages["."]["changelog-sections"][] | select(.hidden != true) | .type' "$config" | paste -sd, -)" == feat,fix,security,perf,revert
check 'hidden changelog types are exactly the documented non-triggers' test "$(jq -r '.packages["."]["changelog-sections"][] | select(.hidden == true) | .type' "$config" | paste -sd, -)" == docs,test,refactor,build,ci,chore
check 'CONTRIBUTING documents the pinned Release Please semantics' bash -c "grep -Fq 'release-please 17.6.0' '$repository_root/CONTRIBUTING.md' && grep -Fq '| \`feat\` | minor | minor |' '$repository_root/CONTRIBUTING.md' && grep -Fq 'hidden in the changelog, not non-releasable' '$repository_root/CONTRIBUTING.md' && grep -Fq 'googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0' '$workflows/release-please.yml'"
check 'merges never start a release: the Release PR workflow runs on explicit dispatch only' test "$(triggers "$workflows/release-please.yml")" == workflow_dispatch
check 'publication runs only on explicit dispatch' test "$(triggers "$workflows/publish-release.yml")" == workflow_dispatch
check 'publication token is scoped to the gated publish job' bash -c "grep -Fxq 'permissions: {}' '$workflows/publish-release.yml' && [[ \$(grep -c 'contents: write' '$workflows/publish-release.yml') == 1 ]] && sed -n '/^  publish:/,\$p' '$workflows/publish-release.yml' | grep -Fxq '    environment: release' && sed -n '/^  publish:/,\$p' '$workflows/publish-release.yml' | grep -Fq 'contents: write'"
check 'publication requires dispatch from main and a verified preflight' bash -c "grep -Fq 'refs/heads/main' '$workflows/publish-release.yml' && grep -Fxq '    needs: preflight' '$workflows/publish-release.yml'"
check 'publication never rebuilds: it consumes the prepared run artifact' bash -c "! grep -Eq 'build-release-archives|upload-artifact' '$workflows/publish-release.yml' && [[ \$(grep -c 'run-id: \${{ inputs.prepared_run }}' '$workflows/publish-release.yml') == 2 ]] && [[ \$(grep -c 'verify-prepared-release.sh' '$workflows/publish-release.yml') == 2 ]]"
check 'publication is bound to the authorized envelope digest' bash -c "grep -Fq -- '--authorized-digest \"\$PREVIEW_DIGEST\"' '$workflows/publish-release.yml' && grep -Fq 'PREVIEW_DIGEST: \${{ inputs.preview_digest }}' '$workflows/publish-release.yml'"
check 'preparation builds, verifies and retains the exact set without publishing' bash -c "grep -Fq build-release-archives.sh '$workflows/release-artifacts.yml' && grep -Fq verify-release-artifacts.sh '$workflows/release-artifacts.yml' && grep -Fq release-notes.sh '$workflows/release-artifacts.yml' && grep -Fq 'name: axiom-release-\${{ env.RELEASE_TAG }}' '$workflows/release-artifacts.yml' && ! grep -Eiq 'contents: write|gh release|git tag|git push|publish-release' '$workflows/release-artifacts.yml'"
# The only secret is the delivery Project credential, and only inside jobs
# gated by an environment: the approved publish job and the Project sync job.
check 'only the delivery Project secret is used, only in environment-gated jobs' bash -c "
  [[ \$(grep -ho 'secrets\.[A-Za-z0-9_]*' $workflows/*.yml | LC_ALL=C sort -u) == secrets.AXIOM_DELIVERY_PROJECT_TOKEN ]] &&
  [[ \$(grep -l 'secrets\.' $workflows/*.yml | xargs -n1 basename | LC_ALL=C sort | paste -sd, -) == delivery-sync.yml,publish-release.yml ]] &&
  [[ \$(grep -c 'secrets\.' '$workflows/publish-release.yml') == 1 && \$(grep -c 'secrets\.' '$workflows/delivery-sync.yml') == 1 ]] &&
  sed -n '/^  publish:/,\$p' '$workflows/publish-release.yml' | grep -Fq 'secrets.AXIOM_DELIVERY_PROJECT_TOKEN' &&
  sed -n '/^  sync-project:/,\$p' '$workflows/delivery-sync.yml' | grep -Fxq '    environment: delivery' &&
  sed -n '/^  sync-project:/,\$p' '$workflows/delivery-sync.yml' | grep -Fq 'secrets.AXIOM_DELIVERY_PROJECT_TOKEN' &&
  ! sed -n '/^  sync:/,/^  sync-project:/p' '$workflows/delivery-sync.yml' | grep -Fq 'secrets.'"
check 'delivery workflows: PR metadata check has no token or secret, sync runs only on main pushes' bash -c "
  [[ \$(sed -n '/^on:/,/^[a-z]/p' '$workflows/delivery-metadata.yml' | grep -E '^  [a-z_]+:' | tr -d ' :' | paste -sd, -) == pull_request,workflow_dispatch ]] &&
  grep -Fxq 'permissions: {}' '$workflows/delivery-metadata.yml' && ! grep -Eq 'write|secrets\.|pull_request_target' '$workflows/delivery-metadata.yml' &&
  grep -Fq 'ref: \${{ github.event.pull_request.base.sha }}' '$workflows/delivery-metadata.yml' &&
  grep -Fq 'PR_BODY: \${{ github.event.pull_request.body }}' '$workflows/delivery-metadata.yml' &&
  [[ \$(grep -c 'github.event.pull_request.body' '$workflows/delivery-metadata.yml') == 1 ]] &&
  [[ \$(sed -n '/^on:/,/^[a-z]/p' '$workflows/delivery-sync.yml' | grep -E '^  [a-z_]+:' | tr -d ' :') == push ]] &&
  sed -n '/^  push:/,/^[a-z]/p' '$workflows/delivery-sync.yml' | grep -Fxq '      - main' &&
  grep -Fxq 'permissions: {}' '$workflows/delivery-sync.yml' && ! grep -Eq 'contents: write|pull_request' '$workflows/delivery-sync.yml'"
check 'Issue closure is a publication effect: no workflow closes Issues on merge or release events' bash -c "
  for w in $workflows/*.yml; do sed -n '/^on:/,/^[a-z]/p' \"\$w\"; done | grep -E '^  [a-z_]+:' | tr -d ' :' | grep -Eqv '^(pull_request|push|workflow_dispatch)$' && exit 1;
  ! grep -Eq 'state=closed|state_reason' $workflows/*.yml"
pinned=true
while IFS= read -r line; do
  [[ "$line" =~ uses:\ [A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}\ \#\ v[0-9.]+$ ]] || { printf 'unpinned: %s\n' "$line" >&2; pinned=false; }
done < <(grep -hE '^\s+(- )?uses:' "$workflows"/*.yml)
check 'every action is pinned by SHA' "$pinned"
check 'checkouts never persist credentials' bash -c "[[ \$(grep -c 'actions/checkout@' $workflows/*.yml | awk -F: '{s+=\$2} END {print s}') == \$(grep -c 'persist-credentials: false' $workflows/*.yml | awk -F: '{s+=\$2} END {print s}') ]]"
ci_contexts=$(printf 'delivery-metadata\nrelease-contract\nverify (linux)\nverify (macos)\n')
check 'ruleset requires exactly the CI job checks and the PR delivery-metadata check' bash -c "[[ \$(jq -r '.rules[] | select(.type == \"required_status_checks\") | .parameters.required_status_checks[].context' '$repository_root/.github/rulesets/main.json' | LC_ALL=C sort) == '$ci_contexts' ]] && grep -Fq 'name: verify (\${{ matrix.platform }})' '$workflows/ci.yml' && grep -Fxq '          - platform: linux' '$workflows/ci.yml' && grep -Fxq '          - platform: macos' '$workflows/ci.yml' && grep -Fxq '    name: release-contract' '$workflows/ci.yml' && grep -Fxq '    name: delivery-metadata' '$workflows/delivery-metadata.yml'"
# Every required context must exist on the Release PR head, where Release
# Please events start no pull_request run: its workflows are dispatched there.
check 'every required context is produced on the Release PR path' bash -c "
  grep -Fq 'gh workflow run ci.yml --repo \"\$repository\" --ref \"\$RELEASE_BRANCH\"' '$repository_root/scripts/release-pr-checks.sh' &&
  grep -Fq 'gh workflow run delivery-metadata.yml --repo \"\$repository\" --ref \"\$RELEASE_BRANCH\"' '$repository_root/scripts/release-pr-checks.sh' &&
  sed -n '/^on:/,/^[a-z]/p' '$workflows/ci.yml' | grep -Fxq '  workflow_dispatch:' &&
  sed -n '/^on:/,/^[a-z]/p' '$workflows/delivery-metadata.yml' | grep -Fxq '  workflow_dispatch:'"
dispatch_path_is_guarded() {
  local w=$workflows/delivery-metadata.yml
  grep -Fq "\${{ github.event_name == 'workflow_dispatch'" "$w" &&
    grep -Fq 'ref: ${{ github.event.repository.default_branch }}' "$w" &&
    grep -Fq './scripts/delivery-github.sh release-pr-head' "$w" &&
    [[ $(grep -c "if: github.event_name == 'pull_request'" "$w") == 3 ]] &&
    [[ $(grep -c "if: github.event_name == 'workflow_dispatch'" "$w") == 2 ]]
}
check 'the delivery-metadata dispatch path always runs trusted default-branch code and is never skipped' dispatch_path_is_guarded
check 'release CI state ignores only the PR-only delivery-metadata check' bash -c "
  grep -Fxq '    [[ \"\$name\" == delivery-metadata ]] && continue' '$repository_root/scripts/release.sh' &&
  [[ \$(grep -c 'delivery-metadata ]] && continue' '$repository_root/scripts/release.sh') == 1 ]]"
check 'ruleset: squash only, reviews, code owners, threads, no direct bypass' bash -c "jq -e '(.rules[] | select(.type == \"pull_request\") | .parameters | .allowed_merge_methods == [\"squash\"] and .required_approving_review_count >= 1 and .require_code_owner_review and .required_review_thread_resolution) and ([.rules[].type] | index(\"deletion\") and index(\"non_fast_forward\")) and all(.bypass_actors[]; .bypass_mode != \"always\")' '$repository_root/.github/rulesets/main.json' >/dev/null"
check 'CODEOWNERS covers the repository' grep -Eq '^\* @[A-Za-z0-9-]+$' "$repository_root/.github/CODEOWNERS"
skill=$repository_root/.agents/skills/axiom-release/SKILL.md
check 'axiom-release skill exists and is routed' bash -c "grep -Fxq 'name: axiom-release' '$skill' && grep -Fq '.agents/skills/axiom-release/SKILL.md' '$repository_root/AGENTS.md'"
check 'skill publishes only through release.sh after human authorization' bash -c "grep -Fq 'scripts/release.sh publish' '$skill' && grep -Fq -- '--authorize-publication' '$skill' && ! grep -Eiq 'gh release (create|upload|edit|delete)|git tag|git push .*--tags|pending_deployments|--force' '$skill'"
check 'skill is not a product Runtime skill' bash -c "[[ ! -e '$repository_root/internal/codexruntime/skills/axiom-release' ]]"

# --- Pinned recovery: original artifacts, independent append-only metadata ---
# Everything here stays inside the local Git/fake GitHub fixtures.
check 'recovery workflows keep source checkout and pin separate control checkout' bash -c "
  for w in '$workflows/release-artifacts.yml' '$workflows/publish-release.yml'; do
    grep -Fq 'ref: \${{ inputs.corrections_revision }}' \"\$w\" &&
      grep -Fxq '          path: source' \"\$w\" &&
      grep -Fxq '          path: control' \"\$w\" &&
      grep -Fq 'rev-list --first-parent origin/main' \"\$w\" &&
      grep -Fq 'AXIOM_RELEASE_CORRECTIONS_DIGEST=' \"\$w\" || exit 1;
  done"
rfix=$temporary/recovery-fixture
rremote=$temporary/recovery-remote.git
git init -q --bare "$rremote"
git init -q -b main "$rfix"
git -C "$rfix" config user.email release-test@example.invalid
git -C "$rfix" config user.name 'Release Test'
git -C "$rfix" config commit.gpgsign false
mkdir -p "$rfix/scripts" "$rfix/.github/rulesets"
cp "$fixture"/scripts/*.sh "$rfix/scripts/"
cp "$repository_root/.github/rulesets/main.json" "$rfix/.github/rulesets/main.json"
rcommit() {
  git -C "$rfix" add -A
  printf '%s\n\n%b' "$1" "${2:-}" | git -C "$rfix" commit -q --allow-empty -F -
  git -C "$rfix" rev-parse HEAD
}
printf '{".":"0.2.1"}\n' >"$rfix/.release-please-manifest.json"
printf '# Changelog\n' >"$rfix/CHANGELOG.md"
jq -n '{schemaVersion: 2, owner: "rgomids", number: 7, projectId: "PVT_7", title: "Axiom Delivery", projection: "disabled", migrationStatuses: ["Legacy Done"]}' >"$rfix/.github/delivery-project.json"
rbase=$(rcommit 'chore(main): release 0.2.1')
rfeat=$(rcommit 'feat(site): add landing page' 'Related-Issues: #86\nCompletes-Issues: #86\n')
rdocs=$(rcommit 'docs: reconcile forward compatibility contract for #153')
printf '{".":"0.3.0"}\n' >"$rfix/.release-please-manifest.json"
printf '# Changelog\n\n## [0.3.0](https://github.com/rgomids/axiom/compare/v0.2.1...v0.3.0) (2026-10-02)\n\n### Features\n\n* **site:** add landing page\n' >"$rfix/CHANGELOG.md"
rsource=$(rcommit 'chore(main): release 0.3.0 (#160)')
printf '%s related=153 completes=none\n' "$rdocs" >"$rfix/.github/delivery-corrections.txt"
# Deliberately make the control artifact verifier unusable: verification must
# select the original source verifier. Control Project changes must be ignored.
printf '#!/usr/bin/env bash\necho "wrong control artifact verifier" >&2\nexit 91\n' >"$rfix/scripts/verify-release-artifacts.sh"
jq '.projection = "enabled"' "$rfix/.github/delivery-project.json" >"$temporary/rproject"
cp "$temporary/rproject" "$rfix/.github/delivery-project.json"
rcontrol=$(rcommit 'fix(release): reviewed metadata recovery' 'Related-Issues: #153\nCompletes-Issues: none\n')
git -C "$rfix" remote add origin "$rremote"
git -C "$rfix" push -q origin main
git -C "$rfix" fetch -q origin
export AXIOM_RELEASE_CORRECTIONS_REVISION=$rcontrol
export AXIOM_RELEASE_CORRECTIONS_DIGEST=$(digest "$rfix/.github/delivery-corrections.txt")
reset_github
issue 86 'Landing page'
printf '{"protection_rules":[{"type":"required_reviewers"}]}\n' >"$state/environment.json"
green "$rsource"
green "$rcontrol"
printf '[{"number":160,"url":"https://github.com/rgomids/axiom/pull/160","title":"chore(main): release 0.3.0","mergeCommit":{"oid":"%s"}}]\n' "$rsource" >"$state/pr-merged.json"
"$rfix/scripts/delivery-issues.sh" release --tag v0.3.0 --revision "$rsource" >"$temporary/rissues"
check 'recovery metadata preserves exactly the original completed Issue set' bash -c "grep -Fxq issues=86 '$temporary/rissues' && ! grep -Fxq issues=153 '$temporary/rissues'"
"$rfix/scripts/delivery-github.sh" state --repo rgomids/axiom --tag v0.3.0 --revision "$rsource" >"$temporary/rstate"
check 'recovery keeps original Project authority despite changed control config' grep -Fq 'projection=disabled' "$temporary/rstate"
make_set "$temporary/rset" 0.3.0 "$rsource" original-source-bytes
"$rfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.3.0 --revision "$rsource" >"$temporary/rset/notes.md"
rprepared=$state/runs/6161/axiom-release-v0.3.0
mkdir -p "$rprepared"
cp -R "$temporary/rset/artifacts" "$rprepared/artifacts"
cp "$temporary/rset/evidence.txt" "$rprepared/release-evidence.txt"
cp "$temporary/rset/notes.md" "$rprepared/release-notes.md"
"$rfix/scripts/release-corrections.sh" --source-revision "$rsource" --corrections-revision "$rcontrol" \
  --corrections-digest "$AXIOM_RELEASE_CORRECTIONS_DIGEST" >"$rprepared/release-recovery.txt"
printf '{"path":".github/workflows/release-artifacts.yml","event":"workflow_dispatch","head_branch":"main","status":"completed","conclusion":"success"}\n' >"$state/runs/6161.json"
check 'prepared recovery verifies original source artifacts using original verifier' \
  "$rfix/scripts/verify-prepared-release.sh" --tag v0.3.0 --revision "$rsource" --prepared "$rprepared" --repo rgomids/axiom --run 6161
printf 'corrupt\n' >>"$rprepared/release-recovery.txt"
expect_failure 'prepared recovery metadata drift refuses' 'metadata differs' \
  "$rfix/scripts/verify-prepared-release.sh" --tag v0.3.0 --revision "$rsource" --prepared "$rprepared" --repo rgomids/axiom --run 6161
sed '$d' "$rprepared/release-recovery.txt" >"$temporary/rmetadata"
cp "$temporary/rmetadata" "$rprepared/release-recovery.txt"
printf 'changed notes\n' >>"$rprepared/release-notes.md"
expect_failure 'prepared recovery notes drift refuses' 'release notes differ' \
  "$rfix/scripts/verify-prepared-release.sh" --tag v0.3.0 --revision "$rsource" --prepared "$rprepared" --repo rgomids/axiom --run 6161
cp "$temporary/rset/notes.md" "$rprepared/release-notes.md"
expect_failure 'recovery cannot apply to release candidates' 'stable-only' \
  "$rfix/scripts/release-notes.sh" --tag v0.3.0-rc.1 --revision "$rsource" --repo rgomids/axiom
saved_fixture=$fixture
fixture=$rfix
envelope "$temporary/rset" --tag v0.3.0 --revision "$rsource" --make-latest true >"$temporary/renvelope"
check 'recovery envelope binds correction revision and committed-file digest' bash -c "grep -Fxq corrections_revision=$rcontrol '$temporary/renvelope' && grep -Fxq corrections_sha256=$AXIOM_RELEASE_CORRECTIONS_DIGEST '$temporary/renvelope' && grep -Fxq revision=$rsource '$temporary/renvelope'"
release status --tag v0.3.0 --revision "$rsource" --prepared-run 6161 \
  --corrections-revision "$rcontrol" --corrections-digest "$AXIOM_RELEASE_CORRECTIONS_DIGEST" >"$temporary/rstatus"
check 'release status selects pinned control while retaining original release source' bash -c "grep -Fxq next_action=authorize_publication '$temporary/rstatus' && grep -Fxq preview.corrections_revision=$rcontrol '$temporary/rstatus' && grep -Fxq preview.revision=$rsource '$temporary/rstatus'"
# Same correction bytes at a different reviewed commit remain different
# authority: source archive bytes stay fixed, while control pin changes digest.
ralt=$(rcommit 'docs: another reviewed control revision')
git -C "$rfix" push -q origin main
git -C "$rfix" fetch -q origin
green "$ralt"
export AXIOM_RELEASE_CORRECTIONS_REVISION=$ralt
expect_failure 'a different valid control SHA rejects earlier prepared notes' 'pins differ' \
  envelope "$temporary/rset" --tag v0.3.0 --revision "$rsource" --make-latest true
"$rfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.3.0 --revision "$rsource" >"$temporary/rset/notes.md"
envelope "$temporary/rset" --tag v0.3.0 --revision "$rsource" --make-latest true >"$temporary/raltenvelope"
check 'different control SHA changes authority despite identical correction file and archives' test \
  "$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/renvelope")" != \
  "$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/raltenvelope")"
export AXIOM_RELEASE_CORRECTIONS_REVISION=$rcontrol
"$rfix/scripts/release-notes.sh" --repo rgomids/axiom --tag v0.3.0 --revision "$rsource" >"$temporary/rset/notes.md"
mkdir -p "$state/runs/5151"
cp -R "$rprepared" "$state/runs/5151/axiom-release-v0.3.0"
cp "$state/runs/6161.json" "$state/runs/5151.json"
release prepare --tag v0.3.0 --revision "$rsource" --corrections-revision "$rcontrol" \
  --corrections-digest "$AXIOM_RELEASE_CORRECTIONS_DIGEST" >"$temporary/rprepare"
check 'prepare dispatch propagates exact recovery pins with no publication' bash -c "grep '^workflow run release-artifacts.yml' '$state/ledger' | grep -Fq 'corrections_revision=$rcontrol' && grep '^workflow run release-artifacts.yml' '$state/ledger' | grep -Fq 'corrections_digest=$AXIOM_RELEASE_CORRECTIONS_DIGEST' && ! grep -q '^POST release' '$state/ledger'"
rpreview=$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/rstatus")
release publish --tag v0.3.0 --revision "$rsource" --prepared-run 6161 \
  --corrections-revision "$rcontrol" --corrections-digest "$AXIOM_RELEASE_CORRECTIONS_DIGEST" \
  --preview-digest "$rpreview" --authorize-publication >"$temporary/rdispatch"
check 'authorized publish dispatch propagates exact recovery pins' bash -c "grep '^workflow run publish-release.yml' '$state/ledger' | grep -Fq 'corrections_revision=$rcontrol' && grep '^workflow run publish-release.yml' '$state/ledger' | grep -Fq 'corrections_digest=$AXIOM_RELEASE_CORRECTIONS_DIGEST'"
publish "$temporary/rset" --tag v0.3.0 --revision "$rsource" --make-latest true >"$temporary/rpublished"
check 'recovery publication delivers only feature Issue in fake GitHub' bash -c "grep -Fxq result=pass '$temporary/rpublished' && [[ \$(jq -r .state '$state/issues/86.json') == closed ]] && ! grep -Eq '^(ISSUE|COMMENT).*153' '$state/ledger'"
"$rfix/scripts/delivery-github.sh" verify --repo rgomids/axiom --tag v0.3.0 --revision "$rsource" >"$temporary/rverified"
check 'recovery delivered Issue verification passes against pinned inputs' grep -Fxq delivery=verified "$temporary/rverified"
# Real tag operations below are confined to the synthetic local bare remote.
git -C "$rfix" push -q origin "$rsource:refs/tags/v0.3.0"
release verify --tag v0.3.0 --download >"$temporary/rdownload"
check 'download verification discovers published pins and verifies original source bytes' bash -c "grep -Fxq artifacts_verified=pass '$temporary/rdownload' && grep -Fxq result=pass '$temporary/rdownload' && grep -Fxq corrections_revision=$rcontrol '$temporary/rdownload'"
export AXIOM_RELEASE_CORRECTIONS_REVISION=$ralt
expect_failure 'published recovery refuses changed correction pins' 'pins differ' \
  "$rfix/scripts/publish-release.sh" --check --repo rgomids/axiom --tag v0.3.0 --revision "$rsource" --make-latest true
unset AXIOM_RELEASE_CORRECTIONS_REVISION AXIOM_RELEASE_CORRECTIONS_DIGEST
# Ordinary post-release sync must resolve the immutable release commit using
# published pins, without adopting those pins for unrelated commit parsing.
export AXIOM_DELIVERY_PROJECT_TOKEN=$FAKE_PROJECT_TOKEN
stage_project 'Legacy Done'
rrecord=$state/releases/$(release_json v0.3.0 | jq -r .id).json
cp "$rrecord" "$temporary/rrecord-original.json"
reject_historical_sync() {
  local label=$1 filter=$2 before
  jq "$filter" "$temporary/rrecord-original.json" >"$rrecord"
  before=$(mutations)
  if "$rfix/scripts/delivery-github.sh" sync --repo rgomids/axiom --from "$rbase" --to "$ralt" \
    >"$temporary/historical-out" 2>"$temporary/historical-error"; then
    check "$label" false
  else
    check "$label" test "$(mutations)" == "$before"
  fi
  cp "$temporary/rrecord-original.json" "$rrecord"
}
reject_historical_sync 'historical recovery refuses draft provenance before any effects' '.draft = true'
reject_historical_sync 'historical recovery refuses prerelease provenance before any effects' '.prerelease = true'
reject_historical_sync 'historical recovery refuses another source revision before any effects' '.target_commitish = "0000000000000000000000000000000000000000"'
reject_historical_sync 'historical recovery refuses duplicate pins before any effects' '.body += "\n- Recovery corrections revision: `0000000000000000000000000000000000000000`"'
reject_historical_sync 'historical recovery refuses malformed pins before any effects' '.body |= sub("Recovery corrections SHA-256: `[0-9a-f]{64}`"; "Recovery corrections SHA-256: `bad`")'
check 'ordinary sync resolves published recovery and repairs released Project projection' \
  "$rfix/scripts/delivery-github.sh" sync --repo rgomids/axiom --from "$rbase" --to "$ralt"
check 'historical recovery reconciliation projects only the published completed Issue' bash -c "
  [[ '$(item 86)' == 'Released|v0.3.0' ]] && [[ \$(jq -r .state '$state/issues/86.json') == closed ]] &&
  ! grep -Eq '^(ISSUE|COMMENT|PROJECT).*153' '$state/ledger'"
unset AXIOM_DELIVERY_PROJECT_TOKEN
# A release published through the recovery is a valid previous release: the
# next start checks it with the pins it records, as verify does.
rm -f "$state/pr-merged.json"
printf '%s related=none completes=none\n' "$ralt" >>"$rfix/.github/delivery-corrections.txt"
rcommit 'docs(delivery): record reviewed correction (#171)' 'Related-Issues: none\nCompletes-Issues: none\n' >/dev/null
# Notes generated later differ; the recovery release keeps its control notes.
printf "printf 'Later notes format.\\\\n'\n" >>"$rfix/scripts/release-notes.sh"
rcommit 'docs(release): change the notes format (#173)' 'Related-Issues: none\nCompletes-Issues: none\n' >/dev/null
rnext=$(rcommit 'fix(cli): repair after recovery (#172)' 'Related-Issues: none\nCompletes-Issues: none\n')
git -C "$rfix" push -q origin main
green "$rnext"
release status >"$temporary/rnext"
check 'a recovery-published previous release allows the next start' bash -c "grep -Fxq next_action=start_release '$temporary/rnext' && grep -Fxq previous_release_state=published '$temporary/rnext' && grep -Fxq planned_tag=v0.3.1 '$temporary/rnext'"
jq '.body |= sub("Recovery corrections SHA-256: `[0-9a-f]{64}`"; "Recovery corrections SHA-256: `bad`")' "$temporary/rrecord-original.json" >"$rrecord"
release status >"$temporary/rnext"
check 'malformed recovery pins on the previous release block the next start' bash -c "grep -Fxq next_action=blocked '$temporary/rnext' && grep -Fq 'previous release v0.3.0: malformed recovery provenance' '$temporary/rnext'"
jq --arg r "$(printf '1%.0s' {1..40})" '.body |= sub("Recovery corrections revision: `[0-9a-f]{40}`"; "Recovery corrections revision: `\($r)`")' "$temporary/rrecord-original.json" >"$rrecord"
release status >"$temporary/rnext"
check 'a previous release pinning a control revision off main runs none of its scripts and blocks the start' bash -c "grep -Fxq next_action=blocked '$temporary/rnext' && grep -Fq 'published correction revision is not on first-parent main' '$temporary/rnext'"
jq --arg r "$rbase" '.body |= sub("Recovery corrections revision: `[0-9a-f]{40}`"; "Recovery corrections revision: `\($r)`")' "$temporary/rrecord-original.json" >"$rrecord"
release status >"$temporary/rnext"
check 'a previous release pinning a control revision older than its source blocks the start' bash -c "grep -Fxq next_action=blocked '$temporary/rnext' && grep -Fq 'correction revision does not descend from source' '$temporary/rnext'"
jq --arg r "$rsource" '.body |= sub("Recovery corrections revision: `[0-9a-f]{40}`"; "Recovery corrections revision: `\($r)`")' "$temporary/rrecord-original.json" >"$rrecord"
release status >"$temporary/rnext"
check 'a previous release pinning its own release commit as control blocks the start' bash -c "grep -Fxq next_action=blocked '$temporary/rnext' && grep -Fq 'correction pins do not validate at the correction revision' '$temporary/rnext'"
jq '.body |= sub("- Recovery corrections revision: `[0-9a-f]{40}`\n"; "")' "$temporary/rrecord-original.json" >"$rrecord"
release status >"$temporary/rnext"
check 'a recovery previous release without its revision pin blocks the next start' bash -c "grep -Fxq next_action=blocked '$temporary/rnext' && grep -Fq 'previous release v0.3.0' '$temporary/rnext'"
cp "$temporary/rrecord-original.json" "$rrecord"
fixture=$saved_fixture
unset AXIOM_RELEASE_CORRECTIONS_REVISION AXIOM_RELEASE_CORRECTIONS_DIGEST

# --- 6. Release Flow v2: merges integrate code, `start` starts releases (ADR-0011) ---
# A fixture main with delivery metadata and a published v0.1.0. Release Please
# itself is the fake dispatch effect that opens the Release PR.
sfix=$temporary/start-fixture
sremote=$temporary/start-remote.git
git init -q --bare "$sremote"
git init -q -b main "$sfix"
git -C "$sfix" config user.email release-test@example.invalid
git -C "$sfix" config user.name 'Release Test'
git -C "$sfix" config commit.gpgsign false
mkdir -p "$sfix/scripts" "$sfix/.github/rulesets"
cp "$saved_fixture"/scripts/*.sh "$sfix/scripts/"
cp "$repository_root/.github/rulesets/main.json" "$sfix/.github/rulesets/main.json"
git -C "$sfix" remote add origin "$sremote"
scommit() {
  [[ -z "${3:-}" ]] || printf '{\n  ".": "%s"\n}\n' "$3" >"$sfix/.release-please-manifest.json"
  git -C "$sfix" add -A
  printf '%s\n\n%b' "$1" "${2:-}" | git -C "$sfix" commit -q --allow-empty -F -
  git -C "$sfix" rev-parse HEAD
}
meta='Related-Issues: none\nCompletes-Issues: none\n'
printf '# Changelog\n\n## [0.1.0](https://github.com/rgomids/axiom/compare/v0.0.0...v0.1.0) (2026-10-01)\n\n* first\n' >"$sfix/CHANGELOG.md"
s0=$(scommit 'chore(main): release 0.1.0 (#1)' '' 0.1.0)
git -C "$sfix" push -q origin main
git -C "$sfix" push -q origin "$s0:refs/tags/v0.1.0"
# stage_published ID TAG REVISION stages, in the fake GitHub, the published
# release of TAG at REVISION (non-draft, SHA256SUMS plus one archive) and
# its tag, as publish-release.sh leaves them.
stage_published() {
  local id=$1 tag=$2 rev=$3 archive=$(($1 * 10)) sums=$(($1 * 10 + 1))
  printf 'archive of %s\n' "$tag" >"$state/assets/$archive"
  printf '%s  axiom-%s-linux-amd64.tar.gz\n' "$(digest "$state/assets/$archive")" "${tag#v}" >"$state/assets/$sums"
  jq -n --argjson id "$id" --arg tag "$tag" --arg rev "$rev" --argjson a "$archive" --argjson s "$sums" \
    --arg ad "sha256:$(digest "$state/assets/$archive")" --arg sd "sha256:$(digest "$state/assets/$sums")" --arg an "axiom-${tag#v}-linux-amd64.tar.gz" \
    '{id: $id, tag_name: $tag, name: $tag, target_commitish: $rev, draft: false, prerelease: false, immutable: true, body: "notes",
      assets: [{id: $s, name: "SHA256SUMS", state: "uploaded", digest: $sd}, {id: $a, name: $an, state: "uploaded", digest: $ad}]}' \
    >"$state/releases/$id.json"
  printf '%s %s\n' "$tag" "$rev" >>"$state/tags"
}
splan() { (cd "$sfix" && "$sfix/scripts/release-plan.sh" --main-ref HEAD); }
# plan_case NAME SUBJECT BODY: one commit on top of s0, then the plan of it.
plan_case() {
  git -C "$sfix" checkout -q -B "case-$1" "$s0"
  scommit "$2" "$3" >/dev/null
  splan
}
planned() { awk -F= '$1 == "planned_version" {print $2}' <<<"$1"; }

# Versioning (0.x, bump-minor-pre-major).
check 'plan: hidden types alone are not releasable' bash -c "[[ \$(git -C '$sfix' checkout -q -B case-docs '$s0' && cd '$sfix' && printf x >d && git add -A && printf 'docs: explain\n\n$meta' | git commit -q -F - && ./scripts/release-plan.sh --main-ref HEAD | grep -E '^(releasable|planned_version)=' | paste -sd, -) == releasable=false,planned_version=none ]]"
check 'plan: fix -> patch' test "$(planned "$(plan_case fix 'fix(cli): repair (#2)' "$meta")")" == 0.1.1
check 'plan: feat -> minor' test "$(planned "$(plan_case feat 'feat(cli): add (#2)' "$meta")")" == 0.2.0
check 'plan: breaking feat! -> minor while 0.x' test "$(planned "$(plan_case bang 'feat(cli)!: rename (#2)' "$meta")")" == 0.2.0
out=$(plan_case footer 'refactor(core): split (#2)' "BREAKING CHANGE: layout moved\n\n$meta")
check 'plan: a BREAKING CHANGE footer makes a hidden type releasable (minor while 0.x)' bash -c "[[ '$(planned "$out")' == 0.2.0 ]] && grep -Fxq releasable=true <<<'$out'"
check 'plan: Release-As forces the version' test "$(planned "$(plan_case as 'chore: prepare 0.5.0 (#2)' "Release-As: 0.5.0\n\n$meta")")" == 0.5.0
# Versioning from 1.0.0: breaking -> major.
git -C "$sfix" checkout -q -B case-one "$s0"
s1=$(scommit 'chore(main): release 1.2.3 (#5)' '' 1.2.3)
git -C "$sfix" push -q origin "$s1:refs/tags/v1.2.3"
one_case() { git -C "$sfix" checkout -q -B "one-$1" "$s1"; scommit "$2" "$meta" >/dev/null; splan; }
check 'plan from 1.x: breaking -> major' test "$(planned "$(one_case bang 'fix(cli)!: drop flag (#6)')")" == 2.0.0
check 'plan from 1.x: feat -> minor' test "$(planned "$(one_case feat 'feat(cli): add (#6)')")" == 1.3.0
check 'plan from 1.x: fix -> patch' test "$(planned "$(one_case fix 'fix(cli): repair (#6)')")" == 1.2.4
git -C "$sfix" push -q origin :refs/tags/v1.2.3
# Refusals: one inconsistent commit stops the plan.
git -C "$sfix" checkout -q -B case-undeclared "$s0"
bad=$(scommit 'fix(cli): repair without metadata (#2)')
expect_failure 'plan: a commit without delivery metadata stops the release' "commit $bad has no delivery metadata" splan
printf '# reviewed: PR #2 predates its metadata\n%s related=none completes=none\n' "$bad" >"$sfix/.github/delivery-corrections.txt"
scommit 'docs(delivery): record reviewed correction (#3)' "$meta" >/dev/null
check 'plan: a reviewed correction merged to main repairs it before any Release PR' test "$(planned "$(splan)")" == 0.1.1
git -C "$sfix" checkout -q -B case-nonconventional "$s0"
odd=$(scommit 'Update stuff (#2)' "$meta")
expect_failure 'plan: a nonconventional subject stops the release' "commit $odd subject is not a Conventional Commit" splan
git -C "$sfix" checkout -q -B case-malformed "$s0"
scommit 'fix(cli): repair (#2)' 'Related-Issues: #4\nCompletes-Issues: #5\n' >/dev/null
expect_failure 'plan: malformed delivery metadata stops the release' 'must also be listed in Related-Issues' splan
git -C "$sfix" checkout -q -B case-unpublished "$s0"
scommit 'chore(main): release 0.2.0 (#2)' '' 0.2.0 >/dev/null
scommit 'fix(cli): repair (#3)' "$meta" >/dev/null
expect_failure 'plan: the previous release must be published first' 'release v0.2.0 is recorded on main but not published' splan
git -C "$sfix" checkout -q -B case-tag "$s0"
scommit 'fix(cli): repair (#2)' "$meta" >/dev/null
git -C "$sfix" push -q origin "$s0:refs/tags/v0.1.1"
expect_failure 'plan: an existing planned tag is a conflict' 'tag v0.1.1 already exists' splan
git -C "$sfix" push -q origin :refs/tags/v0.1.1
grammar() { grep -E "^ *conventional_re='" "$repository_root/scripts/$1" | sed 's/^ *//'; }
same_grammar() { [[ $(grammar release-plan.sh | wc -l | tr -d ' ') == 1 && "$(grammar release-plan.sh)" == "$(grammar delivery-issues.sh)" ]]; }
check 'the plan and the PR title check share one Conventional Commit grammar' same_grammar

# Start: status -> preflight -> dispatch Release Please -> Release PR -> stop.
git -C "$sfix" checkout -q main
printf 'feature\n' >"$sfix/feature.txt"
sfeat=$(scommit 'feat(cli): add feature (#2)' 'Related-Issues: #5\nCompletes-Issues: #5\n')
git -C "$sfix" push -q origin main
fixture=$sfix
reset_github
issue 5 'Feature'
printf '{"protection_rules":[{"type":"required_reviewers"}]}\n' >"$state/environment.json"
green "$sfeat"
# The previous release must be published on GitHub, not only tagged: the plan
# is Git-only, release.sh checks the remote publication state. Any dispatch
# would open a Release PR through this effect.
printf 'printf %s >"%s/pr-open.json"\n' "'[{\"number\":7,\"url\":\"https://github.com/rgomids/axiom/pull/7\",\"title\":\"chore(main): release 0.2.0\",\"headRefOid\":\"x\"}]'" "$state" \
  >"$state/dispatch-effect-release-please.yml"
release status >"$temporary/sstatus"
check 'previous tag without its GitHub Release: start_release is not offered' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fxq previous_release_state=absent '$temporary/sstatus' && grep -Fq 'previous release v0.1.0 has its tag but its GitHub Release is missing' '$temporary/sstatus'"
expect_failure 'previous tag without its GitHub Release: start is refused' 'GitHub Release is missing' release start
check 'previous tag without its GitHub Release: no dispatch, no Release PR' bash -c "! grep -q '^workflow' '$state/ledger' && [[ ! -e '$state/pr-open.json' ]]"
stage_published 50 v0.1.0 "$s0"
jq '.draft = true' "$state/releases/50.json" >"$temporary/srel50" && cp "$temporary/srel50" "$state/releases/50.json"
release status >"$temporary/sstatus"
check 'previous GitHub Release still a draft: start_release is not offered' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fxq previous_release_state=draft '$temporary/sstatus' && grep -Fq 'GitHub Release is draft' '$temporary/sstatus'"
expect_failure 'previous GitHub Release still a draft: start is refused' 'GitHub Release is draft' release start
jq --arg c "$sfeat" '.draft = false | .target_commitish = $c' "$state/releases/50.json" >"$temporary/srel50" && cp "$temporary/srel50" "$state/releases/50.json"
release status >"$temporary/sstatus"
check 'previous GitHub Release at another revision: start_release is not offered' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'previous release v0.1.0 is not consistently published' '$temporary/sstatus' && grep -Fq \"release targets '$sfeat'\" '$temporary/sstatus'"
expect_failure 'previous GitHub Release at another revision: start is refused' 'not consistently published' release start
jq '.target_commitish = "'"$s0"'" | .tag_name = "v0.1.0-other"' "$state/releases/50.json" >"$temporary/srel50" && cp "$temporary/srel50" "$state/releases/50.json"
release status >"$temporary/sstatus"
check 'previous GitHub Release bound to another tag: start_release is not offered' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'previous release v0.1.0 is not consistently published' '$temporary/sstatus'"
check 'refused previous-release states dispatched nothing and opened no Release PR' bash -c "! grep -q '^workflow' '$state/ledger' && [[ ! -e '$state/pr-open.json' ]]"
rm "$state/dispatch-effect-release-please.yml" "$state/releases/50.json"
: >"$state/tags"
stage_published 50 v0.1.0 "$s0"
release status >"$temporary/sstatus"
check 'no release in progress: status plans the next release without effects' bash -c "grep -Fxq state=no_release_in_progress '$temporary/sstatus' && grep -Fxq next_action=start_release '$temporary/sstatus' && grep -Fxq planned_version=0.2.0 '$temporary/sstatus' && grep -Fxq planned_tag=v0.2.0 '$temporary/sstatus' && grep -Fxq plan.issues=5 '$temporary/sstatus' && [[ \$(grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL|workflow)' '$state/ledger' || true) == 0 ]]"
check 'start_release is offered only over a published previous GitHub Release' grep -Fxq previous_release_state=published "$temporary/sstatus"
rm "$state/checks-$sfeat.json"
release status >"$temporary/sstatus"
check 'red or missing CI on main blocks the start' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'required CI on main' '$temporary/sstatus'"
green "$sfeat"
printf '{"id":900,"tag_name":"v0.2.0","name":"v0.2.0","draft":true,"prerelease":false,"target_commitish":"%s","assets":[]}\n' "$sfeat" >"$state/releases/900.json"
release status >"$temporary/sstatus"
check 'a GitHub Release already using the planned tag blocks the start' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'already uses v0.2.0' '$temporary/sstatus'"
rm "$state/releases/900.json"
printf '[{"databaseId":3030,"url":"https://github.com/rgomids/axiom/actions/runs/3030","createdAt":"2026-10-02T00:00:00Z","status":"in_progress"}]\n' >"$state/inflight-release-please.yml.json"
release status >"$temporary/sstatus"
check 'an in-flight Release Please run is reported, never dispatched again' bash -c "grep -Fxq state=release_pr_starting '$temporary/sstatus' && grep -Fxq next_action=await_run '$temporary/sstatus' && grep -Fxq run_id=3030 '$temporary/sstatus'"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'status --tag also waits for an in-flight Release Please run' bash -c "grep -Fxq state=release_pr_starting '$temporary/sstatus' && grep -Fxq next_action=await_run '$temporary/sstatus' && grep -Fxq run_id=3030 '$temporary/sstatus'"
expect_failure 'start is refused while Release Please runs' 'starting a release is not the next step' release start
rm "$state/inflight-release-please.yml.json"
check 'refused starts dispatched nothing' test "$(grep -c '^workflow' "$state/ledger" || true)" == 0
expect_failure 'start takes no version or revision from the operator' 'start takes no options' release start --tag v0.9.0
spr_head=$(printf 'e%.0s' {1..40})
release_pr_head() { printf '{"sha":"%s","parents":[{"sha":"%s"}]}\n' "$spr_head" "$1" >"$state/commit-$spr_head.json"; }
cat >"$state/dispatch-effect-release-please.yml" <<EFFECT
printf '[{"number":7,"url":"https://github.com/rgomids/axiom/pull/7","title":"chore(main): release 0.2.0","headRefOid":"$spr_head"}]\n' >"$state/pr-open.json"
printf '{"sha":"$spr_head","parents":[{"sha":"%s"}]}\n' "\$(git -C '$sfix' rev-parse HEAD)" >"$state/commit-$spr_head.json"
EFFECT
cp "$state/dispatch-effect-release-please.yml" "$temporary/release-please-effect"
green "$spr_head"
release start >"$temporary/sstart"
check 'start dispatches Release Please once with the planned version and exact main' bash -c "[[ \$(grep -c '^workflow run release-please.yml' '$state/ledger') == 1 ]] && grep -Fq 'planned_version=0.2.0' '$state/ledger' && grep -Fq 'main=$sfeat' '$state/ledger' && ! grep -q '^workflow run release-artifacts' '$state/ledger'"
check 'start stops at the Release PR with its URL, planned version and tag' bash -c "grep -Fxq release_pr=https://github.com/rgomids/axiom/pull/7 '$temporary/sstart' && grep -Fxq planned_version=0.2.0 '$temporary/sstart' && grep -Fxq planned_tag=v0.2.0 '$temporary/sstart' && grep -Fxq state=release_pr_open '$temporary/sstart' && grep -Fxq next_action=review_release_pr '$temporary/sstart'"
check 'start creates no tag, release, asset or label' bash -c "[[ $(release_effects) == 0 ]] && [[ -z \$(git -C '$sremote' tag -l v0.2.0) ]]"
rm "$state/dispatch-effect-release-please.yml"
expect_failure 'start again with an open Release PR is refused (idempotent)' 'starting a release is not the next step' release start
check 'repeated start never duplicates the dispatch' test "$(grep -c '^workflow run release-please.yml' "$state/ledger")" == 1
# Gate 1: an open Release PR is human authority; nothing is prepared.
expect_failure 'open Release PR: prepare is refused' 'preparation is not the next step' release prepare
release status --tag v0.2.0 >"$temporary/sstatus"
check 'open Release PR: a stable status asks for review, not preparation' bash -c "grep -Fxq next_action=review_release_pr '$temporary/sstatus' && ! grep -q '^preview_digest=' '$temporary/sstatus'"
check 'open Release PR: no preparation was dispatched' bash -c "! grep -q '^workflow run release-artifacts' '$state/ledger'"
# status and status --tag judge one Release PR by one path (release_pr_facts).
decision() { grep -E '^(state|next_action|reason|release_pr|release_pr_version|release_pr_behind_main)=' "$1"; }
same_decision() {
  release status >"$temporary/sstatus"
  release status --tag "$1" >"$temporary/sstatus-tag"
  [[ "$(decision "$temporary/sstatus")" == "$(decision "$temporary/sstatus-tag")" ]]
}
check 'open Release PR: status --tag reaches the same decision as status' same_decision v0.2.0
cp "$state/pr-open.json" "$temporary/spr-open"
jq '.[0].title = "chore(main): release 0.2.10"' "$temporary/spr-open" >"$state/pr-open.json"
release status --tag v0.2.1 >"$temporary/sstatus"
check 'status --tag v0.2.1 never accepts the Release PR of 0.2.10' bash -c "grep -Fxq state=no_release_in_progress '$temporary/sstatus' && grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'no Release PR prepares 0.2.1' '$temporary/sstatus' && ! grep -q '^release_pr=' '$temporary/sstatus'"
jq '.[0].title = "chore(main): release 0.2.1"' "$temporary/spr-open" >"$state/pr-open.json"
release status --tag v0.2.10 >"$temporary/sstatus"
check 'status --tag v0.2.10 never accepts the Release PR of 0.2.1' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'no Release PR prepares 0.2.10' '$temporary/sstatus'"
jq '.[0].title = "chore(main): release 0.2.0 (extra)"' "$temporary/spr-open" >"$state/pr-open.json"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'status --tag needs the exact Release PR title, not a substring' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'no Release PR prepares 0.2.0' '$temporary/sstatus'"
jq '. + [.[0] | .number = 9 | .url = "https://github.com/rgomids/axiom/pull/9" | .title = "chore(main): release 0.2.10"]' "$temporary/spr-open" >"$state/pr-open.json"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'status --tag is blocked like status when more than one Release PR is open' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'more than one open Release PR' '$temporary/sstatus'"
cp "$temporary/spr-open" "$state/pr-open.json"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'status --tag checks the open Release PR against the plan of main' bash -c "grep -Fxq next_action=review_release_pr '$temporary/sstatus' && grep -Fxq release_pr_version=0.2.0 '$temporary/sstatus' && grep -Fxq planned_version=0.2.0 '$temporary/sstatus' && grep -Fxq release_pr_base=$sfeat '$temporary/sstatus' && grep -Fxq previous_release_state=published '$temporary/sstatus'"
cp "$state/releases/50.json" "$temporary/srel50"
rm "$state/releases/50.json"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'an open Release PR is not offered for review while the previous release is unpublished' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'GitHub Release is missing' '$temporary/sstatus'"
check 'the unpublished previous release blocks status and status --tag alike' same_decision v0.2.0
cp "$temporary/srel50" "$state/releases/50.json"
sdocs=$(scommit 'docs(cli): explain feature (#3)' "$meta")
git -C "$sfix" push -q origin main
green "$sdocs"
release status >"$temporary/sstatus"
check 'a Release PR behind main by validated hidden commits stays reviewable (branch update only)' bash -c "grep -Fxq next_action=review_release_pr '$temporary/sstatus' && grep -Fxq release_pr_behind_main=1 '$temporary/sstatus' && grep -Fq 'update its branch' '$temporary/sstatus'"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'status --tag: hidden-only commits after the Release PR keep it reviewable (branch update only)' bash -c "grep -Fxq next_action=review_release_pr '$temporary/sstatus' && grep -Fxq release_pr_behind_main=1 '$temporary/sstatus' && grep -Fq 'update its branch' '$temporary/sstatus'"
check 'status --tag and status agree on hidden-only commits' same_decision v0.2.0
sfix2=$(scommit 'fix(cli): repair feature (#4)' "$meta")
git -C "$sfix" push -q origin main
green "$sfix2"
release status >"$temporary/sstatus"
check 'a releasable commit merged after the Release PR requires a refresh' bash -c "grep -Fxq next_action=refresh_release_pr '$temporary/sstatus' && grep -Fq 'main changed the release' '$temporary/sstatus'"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'status --tag: a releasable commit after the Release PR requires a refresh, never a review' bash -c "grep -Fxq next_action=refresh_release_pr '$temporary/sstatus' && ! grep -Fxq next_action=review_release_pr '$temporary/sstatus'"
check 'status --tag and status agree on a stale Release PR' same_decision v0.2.0
rm "$state/checks-$sfix2.json"
release status >"$temporary/sstatus"
check 'a refresh is not offered on a red main' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'must be refreshed once it is green' '$temporary/sstatus'"
release status --tag v0.2.0 >"$temporary/sstatus"
check 'status --tag: a stale Release PR on a red main is blocked, never reviewed' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'must be refreshed once it is green' '$temporary/sstatus'"
green "$sfix2"
cp "$temporary/release-please-effect" "$state/dispatch-effect-release-please.yml"
release start >"$temporary/sstart" 2>&1 || true
rm "$state/dispatch-effect-release-please.yml"
check 'start refreshes a stale Release PR after a new preflight' bash -c "[[ \$(grep -c '^workflow run release-please.yml' '$state/ledger') == 2 ]] && grep -Fq 'main=$sfix2' '$state/ledger'"
release status >"$temporary/sstatus"
check 'the refreshed Release PR is reviewable again' grep -Fxq next_action=review_release_pr "$temporary/sstatus"
git -C "$sfix" reset -q --hard "$sfeat"
git -C "$sfix" push -q -f origin main
release_pr_head "$sfeat"
printf '[{"number":7,"url":"https://github.com/rgomids/axiom/pull/7","title":"chore(main): release 0.9.0","headRefOid":"%s"}]\n' "$spr_head" >"$state/pr-open.json"
release status >"$temporary/sstatus"
check 'a Release PR version different from the plan is never offered for review' bash -c "grep -Fxq next_action=refresh_release_pr '$temporary/sstatus' && grep -Fq 'it records 0.9.0, main plans 0.2.0' '$temporary/sstatus'"
check 'status --tag of that Release PR version agrees with status' same_decision v0.9.0
printf '[{"number":7,"url":"https://github.com/rgomids/axiom/pull/7","title":"chore(main): release 0.2.0","headRefOid":"%s"}]\n' "$spr_head" >"$state/pr-open.json"
sbad=$(scommit 'fix(cli): merged without metadata (#4)')
git -C "$sfix" push -q origin main
release status >"$temporary/sstatus"
check 'an inconsistent commit merged after the Release PR blocks its review' bash -c "grep -Fxq next_action=blocked '$temporary/sstatus' && grep -Fq 'commit $sbad has no delivery metadata' '$temporary/sstatus'"
rm "$state/pr-open.json"
: >"$state/ledger"
release status >"$temporary/sstatus"
expect_failure 'invalid metadata: start fails in preflight' "commit $sbad has no delivery metadata" release start
check 'invalid metadata: no Release PR, artifact, tag or release' bash -c "[[ \$(grep -Ec '^(workflow|POST|PATCH|DELETE|UPLOAD|LABEL)' '$state/ledger' || true) == 0 ]] && [[ ! -e '$state/pr-open.json' ]] && [[ -z \$(git -C '$sremote' tag -l 'v0.2*') ]] && [[ \$(ls '$state/releases') == 50.json ]]"
git -C "$sfix" reset -q --hard "$sfeat"
git -C "$sfix" push -q -f origin main
cat >"$state/dispatch-effect-release-please.yml" <<EFFECT
printf '[{"number":7,"url":"https://github.com/rgomids/axiom/pull/7","title":"chore(main): release 0.3.0","headRefOid":"$spr_head"}]\n' >"$state/pr-open.json"
EFFECT
expect_failure 'a Release PR whose version differs from the plan fails start closed' 'the Release PR is not ready for review' release start
rm "$state/dispatch-effect-release-please.yml" "$state/pr-open.json"

# Human merge: the merged Release PR is discovered; prepare uses its commit.
printf '# Changelog\n\n## [0.2.0](https://github.com/rgomids/axiom/compare/v0.1.0...v0.2.0) (2026-10-02)\n\n### Features\n\n* **cli:** add feature ([#2](https://github.com/rgomids/axiom/issues/2)) [0.2.0]\n\n## [0.1.0](https://github.com/rgomids/axiom/compare/v0.0.0...v0.1.0) (2026-10-01)\n\n* first\n' >"$sfix/CHANGELOG.md"
srel=$(scommit 'chore(main): release 0.2.0 (#7)' '' 0.2.0)
git -C "$sfix" push -q origin main
green "$srel"
printf '[{"number":7,"url":"https://github.com/rgomids/axiom/pull/7","title":"chore(main): release 0.2.0","mergeCommit":{"oid":"%s"}}]\n' "$srel" >"$state/pr-merged.json"
printf '[{"number":7,"merged_at":"2026-10-02T00:00:00Z","labels":[{"name":"autorelease: pending"}]}]\n' >"$state/pulls.json"
: >"$state/ledger"
release status >"$temporary/sstatus"
check 'merged Release PR: the release commit is discovered without operator SHAs' bash -c "grep -Fxq state=release_pr_merged '$temporary/sstatus' && grep -Fxq tag=v0.2.0 '$temporary/sstatus' && grep -Fxq revision=$srel '$temporary/sstatus' && grep -Fxq next_action=prepare '$temporary/sstatus'"
check 'a new start is refused while a merged release awaits publication' bash -c "! (cd '$sfix' && ./scripts/release.sh start) 2>/dev/null"
stage_prepared_run 5151 v0.2.0 "$srel" first
release prepare >"$temporary/sprepare"
check 'prepare dispatches the exact release commit and nothing else' bash -c "[[ \$(grep -c '^workflow run release-artifacts.yml' '$state/ledger') == 1 ]] && grep -Fq 'tag=v0.2.0 -f revision=$srel' '$state/ledger' && [[ \$(grep -c '^workflow' '$state/ledger') == 1 ]]"
check 'prepare creates no tag or release' bash -c "[[ $(release_effects) == 0 ]] && [[ -z \$(git -C '$sremote' tag -l v0.2.0) ]] && [[ \$(ls '$state/releases') == 50.json ]]"
check 'prepare stops at the authority boundary with the complete envelope' bash -c "grep -Fxq state=awaiting_publication_authority '$temporary/sprepare' && grep -Fxq next_action=authorize_publication '$temporary/sprepare' && grep -Fxq preview.revision=$srel '$temporary/sprepare' && grep -Fxq preview.tag=v0.2.0 '$temporary/sprepare' && grep -Fxq preview.make_latest=true '$temporary/sprepare' && grep -Fxq preview.prepared_run=5151 '$temporary/sprepare' && grep -Fxq preview.delivery_issues=5 '$temporary/sprepare' && grep -q '^preview.effect.tag=create_at_revision' '$temporary/sprepare' && grep -Eq '^preview_digest=[0-9a-f]{64}$' '$temporary/sprepare'"
sdigest=$(awk -F= '$1 == "preview_digest" {print $2}' "$temporary/sprepare")
# Continuation: a later $axiom-release finds the prepared run by itself.
printf '{"artifacts":[{"name":"axiom-release-v0.2.0","expired":false,"created_at":"2026-10-02T01:00:00Z","workflow_run":{"id":5151}}]}\n' >"$state/artifacts.json"
release status >"$temporary/sstatus"
check 'status rediscovers the verified prepared run and the same envelope' bash -c "grep -Fxq prepared_run_source=discovered '$temporary/sstatus' && grep -Fxq preview_digest=$sdigest '$temporary/sstatus'"
expect_failure 'prepare again is refused once a verified set exists (no duplicate build)' 'preparation is not the next step' release prepare
: >"$state/ledger"
expect_failure 'no authorization: zero effects' 'requires explicit human authorization' release publish --preview-digest "$sdigest"
expect_failure 'stale authority: an old digest is refused' 'preview changed; review and authorize again' \
  release publish --preview-digest "$(printf 'b%.0s' {1..64})" --authorize-publication
stage_prepared_run 5151 v0.2.0 "$srel" drifted
expect_failure 'drift: a changed prepared set invalidates the authorized digest' 'preview changed; review and authorize again' \
  release publish --preview-digest "$sdigest" --authorize-publication
stage_prepared_run 5151 v0.2.0 "$srel" first
stage_prepared_run 6262 v0.2.0 "$srel" first
expect_failure 'the digest binds the prepared run: another run with the same bytes is refused' 'preview changed; review and authorize again' \
  release publish --preview-digest "$sdigest" --prepared-run 6262 --authorize-publication
expect_failure 'a given revision must equal the envelope one' 'differs from the envelope' \
  release publish --preview-digest "$sdigest" --revision "$sfeat" --authorize-publication
check 'refused publications had zero effects' test "$(mutations)" == 0
printf '[{"databaseId":4040,"url":"https://github.com/rgomids/axiom/actions/runs/4040","createdAt":"2026-10-02T00:00:00Z","status":"waiting"}]\n' >"$state/inflight-publish-release.yml.json"
release status >"$temporary/sstatus"
check 'a publication waiting for environment approval is reported, never redispatched' bash -c "grep -Fxq state=publishing '$temporary/sstatus' && grep -Fxq next_action=await_run '$temporary/sstatus' && grep -Fxq run_status=waiting '$temporary/sstatus'"
expect_failure 'publish is refused while a publication run is in flight' 'publication is not the next step' \
  release publish --preview-digest "$sdigest" --authorize-publication
rm "$state/inflight-publish-release.yml.json"
release publish --preview-digest "$sdigest" --authorize-publication >"$temporary/sdispatch"
check 'the authorized digest alone dispatches the discovered exact envelope once' bash -c "[[ \$(grep -c '^workflow run publish-release.yml' '$state/ledger') == 1 ]] && grep -Fq 'tag=v0.2.0 -f revision=$srel -f prepared_run=5151 -f preview_digest=$sdigest' '$state/ledger' && [[ \$(grep -Ec '^(POST|PATCH|DELETE|UPLOAD|LABEL)' '$state/ledger' || true) == 0 ]]"

# Publication (the workflow's publish-release.sh on the prepared bytes), then verify.
sprep=$state/runs/5151/axiom-release-v0.2.0
spub=("$sfix/scripts/publish-release.sh" --repo rgomids/axiom --tag v0.2.0 --revision "$srel" --make-latest true --prepared-run 5151
  --dir "$sprep/artifacts" --evidence "$sprep/release-evidence.txt" --notes "$sprep/release-notes.md")
"${spub[@]}" --authorized-digest "$("${spub[@]}" --envelope | awk -F= '$1 == "preview_digest" {print $2}')" >/dev/null
git -C "$sfix" push -q origin "$srel:refs/tags/v0.2.0"
release status >"$temporary/sstatus"
check 'published: status routes to verification' bash -c "grep -Fxq state=published '$temporary/sstatus' && grep -Fxq next_action=verify_published '$temporary/sstatus'"
release verify --download >"$temporary/sverify"
check 'verify reads back tag, revision, assets, latest and the prepared bytes' bash -c "grep -Fxq publication_state=published '$temporary/sverify' && grep -Fxq latest=v0.2.0 '$temporary/sverify' && grep -Fxq artifacts_verified=pass '$temporary/sverify' && grep -Fxq prepared_match=pass '$temporary/sverify' && grep -Fxq result=pass '$temporary/sverify'"
stage_prepared_run 5151 v0.2.0 "$srel" other-bytes
expect_failure 'verify fails when the published assets differ from the prepared set' 'published assets differ from the prepared set' release verify
stage_prepared_run 5151 v0.2.0 "$srel" first
sasset=$(jq -r 'select(.tag_name == "v0.2.0") | .assets[] | select(.name | endswith("linux-amd64.tar.gz")) | .id' "$state"/releases/*.json)
cp "$state/assets/$sasset" "$temporary/sasset"
printf 'tampered\n' >>"$state/assets/$sasset"
expect_failure 'verify fails when a published asset differs' 'checksum mismatch' release verify --download
cp "$temporary/sasset" "$state/assets/$sasset"
srelease_file=$(grep -l '"v0.2.0"' "$state"/releases/*.json)
cp "$srelease_file" "$temporary/srelease.json"
jq --arg c "$sfeat" '.target_commitish = $c' "$temporary/srelease.json" >"$srelease_file"
expect_failure 'verify fails when the release revision differs' "release targets '$sfeat'" release verify
cp "$temporary/srelease.json" "$srelease_file"
git -C "$sfix" push -q -f origin "$sfeat:refs/tags/v0.2.0"
expect_failure 'verify fails when the tag points elsewhere' 'release_preflight_error' release verify
git -C "$sfix" push -q -f origin "$srel:refs/tags/v0.2.0"
check 'verify passes again on the consistent release' bash -c "cd '$sfix' && ./scripts/release.sh verify | grep -Fxq result=pass"
fixture=$saved_fixture

# Static: the skill orchestrates the new contract.
check 'skill starts releases through release.sh start and never merges' bash -c "grep -Fq 'scripts/release.sh start' '$skill' && grep -Fq 'Merges integrate code' '$skill' && ! grep -Eiq 'gh pr (merge|review)' '$skill'"
check 'release-please.yml re-runs the plan on the planned main before Release Please' bash -c "
  w='$workflows/release-please.yml'
  [[ \$(grep -n 'release-plan.sh --main-ref HEAD' \"\$w\" | cut -d: -f1) -lt \$(grep -n 'googleapis/release-please-action@' \"\$w\" | cut -d: -f1) ]] &&
  grep -Fq 'ref: \${{ inputs.main }}' \"\$w\" && grep -Fq '[[ \"\$DISPATCH_REF\" == refs/heads/main ]]' \"\$w\" &&
  grep -Fq 'PLANNED_VERSION: \${{ inputs.planned_version }}' \"\$w\" && ! grep -Fq '\${{ inputs.planned_version }}\"' \"\$w\""

if ((failures > 0)); then
  printf 'FAIL: %s release flow check(s) failed\n' "$failures" >&2
  exit 1
fi
printf 'PASS: release preflight, notes, publication state machine, authority boundary and workflow contracts\n'
