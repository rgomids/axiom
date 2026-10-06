#!/usr/bin/env bash
# Release upgrade acceptance (Issues #153 and #186): installs real earlier
# releases into isolated homes, creates non-empty persisted state with those
# releases' own commands, upgrades through the candidate archive's own bundle
# facade, and asserts the resulting version, state, Runtime integration and
# idempotent rerun.
#
#   scripts/test-upgrade-journeys.sh --candidate ABS_DIR --previous ABS_DIR \
#     [--previous ABS_DIR ...] [--poc-binary ABS_FILE] [--evidence ABS_FILE]
#
# --candidate and each --previous hold one release's SHA256SUMS and archives
# (as build-release-archives.sh or `gh release download` produce them). The
# first --previous is release N and runs the full v1 journey; every further
# --previous runs the skill-convergence journey from that release. With
# --poc-binary (a lingo built from tag v0.1.0-poc.1), the RecognizedPOC
# journey upgrades historical POC state through N to the candidate.
# Nothing touches the real home, Providers, or the network.
#
# With --evidence (Issue #234), the run also writes an axiom-gate-evidence/v1
# document (scripts/schemas/axiom-gate-evidence-v1.schema.json) for the
# rebuilt candidate, built by scripts/gate-evidence.py from the step records
# below and validated before it is written. It is written on pass and on
# failure, aborts and interrupts whenever the candidate can be identified and
# every step record was captured; the console only gains its digest. The
# original exit status is kept; a passing run whose Evidence cannot be written
# exits 70. The --evidence path must not exist yet, so no earlier document can
# stand in for this attempt.
set -euo pipefail
umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
candidate=
poc_binary=
evidence=
previous=()
while (($#)); do
  case "$1" in
    --candidate) candidate=${2:-}; shift 2 ;;
    --previous) previous+=("${2:-}"); shift 2 ;;
    --poc-binary) poc_binary=${2:-}; shift 2 ;;
    --evidence) evidence=${2:-}; shift 2 ;;
    *) printf 'upgrade_journey_error: invalid argument\n' >&2; exit 2 ;;
  esac
done
[[ "$candidate" == /* && ${#previous[@]} -ge 1 ]] || { printf 'upgrade_journey_error: --candidate and --previous absolute directories required\n' >&2; exit 2; }
[[ -z "$poc_binary" || "$poc_binary" == /* ]] || { printf 'upgrade_journey_error: --poc-binary must be absolute\n' >&2; exit 2; }
if [[ -n "$evidence" ]]; then
  # Evidence never lands inside the subject, a source or the repository.
  evidence_parent=$(cd "$(dirname "$evidence")" 2>/dev/null && pwd -P) || evidence_parent=
  [[ "$evidence" == /* && -n "$evidence_parent" && ! -d "$evidence" ]] || { printf 'upgrade_journey_error: --evidence must be an absolute file in an existing directory\n' >&2; exit 2; }
  [[ ! -e "$evidence" && ! -L "$evidence" ]] || { printf 'upgrade_journey_error: --evidence must not exist yet\n' >&2; exit 2; }
  for input in "$candidate" "${previous[@]}" "$repository_root"; do
    input=$(cd "$input" 2>/dev/null && pwd -P) || continue
    case "$evidence_parent/" in "$input"/*) printf 'upgrade_journey_error: --evidence must be outside the candidate, previous releases and repository\n' >&2; exit 2 ;; esac
  done
  evidence_python=$(command -v python3) || { printf 'upgrade_journey_error: --evidence requires python3\n' >&2; exit 2; }
fi
stable_corpus=internal/compatibility/testdata/stable-v1/snapshots/v0.4.0

case "$(uname -s):$(uname -m)" in
  Darwin:arm64) row=macos-27-arm64 ;;
  Linux:x86_64) row=linux-amd64 ;;
  Linux:aarch64) row=linux-arm64 ;;
  *) printf 'upgrade_journey_row=unsupported\nresult=blocked\n'; exit 78 ;;
esac

original_home=$HOME original_path=$PATH
# --- evidence capture: begin (exercised directly by scripts/test-gate-evidence.py)
utc_now() { date -u +%Y-%m-%dT%H:%M:%SZ; }
cleanup() {
  local status=$?
  trap - EXIT
  # A signal deferred inside check() runs here with its redirections still
  # active; the Evidence summary goes to the original streams.
  [[ -z "$evidence" ]] || { exec 1>&8 2>&9; emit_evidence "$status"; } || [[ $status -ne 0 ]] || status=70
  chmod -R u+w "$work" 2>/dev/null; rm -rf -- "$work"
  exit "$status"
}
now_ms() {
  if [[ $clock_resolution_ms -eq 1 ]]; then local now=${EPOCHREALTIME/[.,]/}; printf '%s\n' $((10#$now / 1000)); else printf '%s000\n' "$(date +%s)"; fi
}
# Step records for the Evidence emitter: numbered, tab-separated, fixed
# vocabulary, never command output. A failed append never changes a journey's
# outcome, but it compromises the capture: the attempt then has no Evidence.
record() {
  [[ -n "$evidence" ]] || return 0
  record_count=$((record_count + 1))
  (IFS=$'\t'; printf '%s\t%s\n' "$record_count" "$*") >>"$work/evidence.records" 2>/dev/null || capture_lost=$((capture_lost + 1))
}
observe() { local value=${2%%$'\n'*}; record observe "$journey_id" "$1" "${value//$'\t'/ }"; }
end_journey() { [[ -z "$journey_id" ]] || record journey_end "$journey_id" "$(utc_now)"; journey_id=; }
# begin_journey ID ROLE SOURCES GENERATION: closes the previous journey.
begin_journey() { end_journey; journey_id=$1; record journey "$1" "$2" "$3" "$4" "$(utc_now)"; }
pass() { printf 'journey=%s step=%s result=pass\n' "$journey" "$1"; record step "$journey_id" "$1" pass $(($(now_ms) - $2)) -; }
# fail STEP STARTED CATEGORY: CATEGORY is the observed failure category, or -.
fail() { printf 'journey=%s step=%s result=fail\n' "$journey" "$1"; failures=$((failures + 1)); record step "$journey_id" "$1" fail $(($(now_ms) - $2)) "$3"; }
run_check() { local category=$1 step=$2 started; shift 2; started=$(now_ms); if "$@" >"$work/last.out" 2>&1; then pass "$step" "$started"; else fail "$step" "$started" "$category"; sed 's/^/  /' "$work/last.out" | head -20; fi; }
# check STEP CMD...: the cause of a failure is not observable here, so its
# failure category stays undetermined (null in the Evidence).
check() { run_check - "$@"; }
# check_observed OBSERVED STEP CMD...: STEP compares OBSERVED, a value the
# candidate itself produced, with its contract. A definite OBSERVED that
# contradicts it is a `product` failure; an empty one stays undetermined.
check_observed() { local category=-; [[ -z "$1" ]] || category=product; shift; run_check "$category" "$@"; }

# emit_evidence STATUS: builds, validates and writes the Evidence document with
# the caller's own HOME and PATH (never the isolated homes or their shims).
emit_evidence() {
  local status=$1 state=aborted arguments=() directory
  if [[ $capture_lost -ne 0 ]]; then
    printf 'evidence=unavailable\n  capture incomplete: %s of %s Evidence records not written\n' "$capture_lost" "$record_count"
    return 1
  fi
  if [[ -n "$signal" ]]; then state=interrupted; elif [[ "$termination" == completed ]]; then state=completed; fi
  for directory in "${previous[@]}"; do arguments+=(--previous "$directory"); done
  [[ -z "$poc_binary" ]] || arguments+=(--poc-binary "$poc_binary")
  [[ -n "$signal" ]] && arguments+=(--signal "$signal")
  touch "$work/evidence.records"
  if HOME=$original_home PATH=$original_path "$evidence_python" "$repository_root/scripts/gate-evidence.py" upgrade-journeys \
    --records "$work/evidence.records" --output "$evidence" --row "$row" --candidate "$candidate" \
    --fixture "stable-v1-v0.4.0=$stable_corpus" --started-at "$started_at" --finished-at "$(utc_now)" \
    --exit-code "$status" --termination "$state" --bash-version "$BASH_VERSION" \
    --attempt-id "$attempt_id" --record-count "$record_count" \
    --clock-resolution-ms "$clock_resolution_ms" --work-dir "$work" --repository "$repository_root" \
    "${arguments[@]}" >"$work/evidence.out" 2>"$work/evidence.err"; then
    cat "$work/evidence.out" || true
  else
    printf 'evidence=unavailable\n'
    sed 's/^/  /' "$work/evidence.err" | head -5
    return 1
  fi
}
# --- evidence capture: end

termination= signal= record_count=0 capture_lost=0 attempt_id=
# The attempt identity exists before anything runs; every record and any
# re-emission of this attempt carry it.
if [[ -n "$evidence" ]]; then
  attempt_id=$("$evidence_python" -c 'import uuid; print(uuid.uuid4())') || { printf 'upgrade_journey_error: cannot create the Evidence attempt identity\n' >&2; exit 2; }
fi
started_at=$(utc_now)
work=$(mktemp -d)
work=$(cd "$work" && pwd -P)
trap cleanup EXIT
if [[ -n "$evidence" ]]; then
  # Original streams for the Evidence summary; they stay open in check() because
  # a deferred signal runs cleanup inside its redirections.
  exec 8>&1 9>&2
  trap 'signal=HUP; exit 129' HUP
  trap 'signal=INT; exit 130' INT
  trap 'signal=TERM; exit 143' TERM
fi
failures=0
journey= journey_id=
if [[ -n "${EPOCHREALTIME:-}" ]]; then clock_resolution_ms=1; else clock_resolution_ms=1000; fi

digest() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'; else shasum -a 256 "$1" | awk '{print $1}'; fi
}

# release_bundle DIR: extracts the host row archive once and prints the bundle.
release_bundle() {
  local dir=$1 archive name
  archive=$(find "$dir" -maxdepth 1 -name "axiom-*-$row.tar.gz" | head -1)
  [[ -n "$archive" ]] || { printf 'upgrade_journey_error: no %s archive in %s\n' "$row" "$dir" >&2; return 1; }
  name=$(basename "$archive" .tar.gz)
  if [[ ! -d "$work/bundles/$name" ]]; then
    mkdir -p "$work/bundles"
    tar -xzf "$archive" -C "$work/bundles"
  fi
  printf '%s\n' "$work/bundles/$name"
}

# new_home NAME: an isolated HOME with Runtime stand-ins and fake Providers.
new_home() {
  H="$work/home-$1"
  mkdir -p "$H/fakebin" "$H/repo" "$H/cwd"
  chmod 700 "$H"
  for runtime in codex claude; do printf '#!/bin/sh\nexit 0\n' >"$H/fakebin/$runtime"; done
  # Bundles older than ADR-0015 compared the macOS product version with 27.0.
  printf '#!/bin/sh\ncase "$1" in -productVersion) echo 27.0 ;; *) exec /usr/bin/sw_vers "$@" ;; esac\n' >"$H/fakebin/sw_vers"
  printf '%s\n' '#!/bin/sh' "printf '%s\\n' 'git@github.com:owner/repo.git'" >"$H/fakebin/git-provider"
  printf '%s\n' '#!/bin/sh' 'case "$*" in' \
    '  *search/issues*) echo "{\"total_count\":0,\"items\":[]}" ;;' \
    '  *POST*repos/owner/repo/issues*) cat >/dev/null; echo "{\"number\":7,\"html_url\":\"https://github.com/owner/repo/issues/7\",\"state\":\"open\",\"labels\":[]}" ;;' \
    '  *repos/owner/repo/labels*) echo "[]" ;;' \
    '  *issues/7/comments*) echo "[]" ;;' \
    '  *issues/42*) echo "{\"number\":42,\"html_url\":\"https://github.com/owner/repo/issues/42\",\"state\":\"open\",\"labels\":[]}" ;;' \
    '  *issues/7*) echo "{\"number\":7,\"html_url\":\"https://github.com/owner/repo/issues/7\",\"state\":\"open\",\"labels\":[]}" ;;' \
    '  "issue create"*) echo https://github.com/owner/repo/issues/7 ;;' \
    '  "issue view"*) echo "{\"Number\":7,\"URL\":\"https://github.com/owner/repo/issues/7\",\"State\":\"OPEN\"}" ;;' \
    '  *) exit 1 ;;' 'esac' >"$H/fakebin/gh-provider"
  chmod 755 "$H"/fakebin/*
  export HOME=$H PATH="$H/.local/bin:$H/fakebin:/usr/bin:/bin:/usr/sbin:/sbin"
  export AXIOM_GIT_BIN="$H/fakebin/git-provider" AXIOM_GH_BIN="$H/fakebin/gh-provider"
  unset XDG_STATE_HOME XDG_CONFIG_HOME XDG_DATA_HOME CLAUDE_CONFIG_DIR AXIOM_CODEX_SKILLS_ROOT LINGO_PROJECTS_ROOT LINGO_STATE_ROOT AXIOM_ARCHIVE_ROOT
  STATE=$(cd "$H/cwd" && "$candidate_bundle/axiom" --json compatibility inspect 2>/dev/null | sed -n 's/.*"stateRoot":"\([^"]*\)".*/\1/p')
  if [[ -z "$STATE" ]]; then
    case "$row" in
      macos-*) STATE="$H/Library/Application Support/Lingo" ;;
      *) STATE="$H/.local/state/lingo" ;;
    esac
  fi
  PROJECTS="$H/.axiom/projects"
}

# install_release DIR: the release's own bundle facade into ~/.local.
install_release() {
  local bundle archive
  bundle=$(release_bundle "$1")
  archive=$(find "$1" -maxdepth 1 -name "axiom-*-$row.tar.gz" | head -1)
  (cd / && bash "$bundle/install.sh" --archive "$archive" --checksums "$1/SHA256SUMS" \
    --bin-dir "$H/.local/bin" --receipt-dir "$H/.local/state/axiom/install") >"$work/install.out" 2>"$work/install.err"
}

tree_digest() {
  (cd "$1" && find . -type f -print | LC_ALL=C sort | while IFS= read -r file; do printf '%s %s\n' "$file" "$(digest "$file")"; done) | digest /dev/stdin
}

version_of() { "$H/.local/bin/axiom" version | sed -n 's/^provenance: Axiom \([^ ]*\) .*/\1/p'; }
installer_status() { sed -n 's/^install_status=//p' "$work/install.out" | head -1; }

candidate_bundle=$(release_bundle "$candidate")
candidate_version=$(sed -n 's/^version=//p' "$candidate_bundle/release-metadata.txt")
printf 'suite=upgrade-journeys\nrow=%s\ncandidate=%s\n' "$row" "$candidate_version"

# Seeds non-empty v1 state with the installed release's own CLI: a Project,
# a GitHub Work Item (with its create attempt), an Execution advanced through
# two gates; then adds the frozen stable-v1 corpus, written by the real v1
# stores, for the stores that have no offline CLI (graph, coordination,
# runtime profile, detail artifacts and their cleanup/retirement records).
seed_v1_state() {
  local repo="$H/repo" preview digest_value
  git -C "$repo" init -q 2>/dev/null || true
  (cd "$H/cwd" && axiom --json project configure --slug journey --name Journey --repository "main=$repo" --work-item-provider github) >"$work/project-preview.json"
  project_id=$(sed -n 's/.*"projectId":"\([^"]*\)".*/\1/p' "$work/project-preview.json")
  digest_value=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$work/project-preview.json")
  (cd "$H/cwd" && axiom --json project configure --project-id "$project_id" --slug journey --name Journey --repository "main=$repo" --work-item-provider github --preview-digest "$digest_value" --authorize-local) >"$work/project.json"
  grep -q '"status":"success"' "$work/project.json"
  local draft=(work-item create --project journey --repository main --provider-repository owner/repo --intent "Upgrade journey" --desired-outcome "State survives upgrades" --context "Acceptance" --scope "Bounded" --constraints "Exact authority" --non-goals "None" --acceptance "Upgrade keeps state")
  (cd "$H/cwd" && axiom --json "${draft[@]}") >"$work/work-item-preview.json"
  digest_value=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$work/work-item-preview.json")
  (cd "$H/cwd" && axiom --json "${draft[@]}" --preview-digest "$digest_value" --authorize-external) >"$work/work-item.json"
  grep -q '"status":"success"' "$work/work-item.json"
  (cd "$H/cwd" && axiom --json workflow start --project journey --repository main --number 7) >"$work/workflow.json"
  execution_id=$(sed -n 's/.*"executionId":"\([^"]*\)".*/\1/p' "$work/workflow.json")
  [[ -n "$execution_id" ]]
  local revision=1 gate reference
  for gate in intake specification; do
    printf '%s\n' "$gate" >"$repo/$gate.md"
    reference="evidence:$gate.md:$(digest "$repo/$gate.md")"
    (cd "$H/cwd" && axiom --json workflow advance --project journey --repository main --number 7 --expected-revision "$revision" --gate "$gate" --outcome pass --reference "$reference") >"$work/advance.json"
    grep -q '"status":"success"' "$work/advance.json"
    revision=$((revision + 1))
  done
  local corpus="$repository_root/$stable_corpus"
  (cd "$corpus/state" && find . -type d -print) | while IFS= read -r directory; do mkdir -p "$STATE/$directory"; chmod 700 "$STATE/$directory"; done
  (cd "$corpus/state" && find . -type f -print) | while IFS= read -r file; do cp "$corpus/state/$file" "$STATE/$file"; chmod 600 "$STATE/$file"; done
  mkdir -p "$PROJECTS/sample" && chmod 700 "$PROJECTS/sample"
  cp "$corpus/projects/sample/axiom.yaml" "$PROJECTS/sample/axiom.yaml" && chmod 600 "$PROJECTS/sample/axiom.yaml"
}

classification() { (cd "$H/cwd" && axiom --json compatibility inspect) | sed -n 's/.*"classification":"\([^"]*\)".*/\1/p' | head -1; }

# Journey 1: release N with non-empty v1 state -> candidate (direct).
journey=v1-state-$(basename "${previous[0]}")
begin_journey v1-state n previous-1 -
new_home v1
check install-previous install_release "${previous[0]}"
previous_version=$(version_of)
check first-run-previous bash -c 'cd "$HOME/cwd" && axiom first-run'
check seed-state-with-previous seed_v1_state
classification_before=$(classification) || true
check previous-reads-seeded-state-as-v1 test "$classification_before" = valid_v1
observe classification_before "$classification_before"
state_before=$(tree_digest "$STATE")
portable_before=$(tree_digest "$PROJECTS")
check upgrade install_release "$candidate"
upgrade_status=$(installer_status) || true
observe installer_upgrade "$upgrade_status"
check_observed "$upgrade_status" upgrade-status grep -qx 'install_status=upgraded' "$work/install.out"
version_after=$(version_of) || true
check_observed "$version_after" version test "$version_after" = "$candidate_version"
state_after=$(tree_digest "$STATE") || true
check_observed "$state_after" state-bytes-unchanged test "$state_after" = "$state_before"
portable_after=$(tree_digest "$PROJECTS") || true
check_observed "$portable_after" portable-bytes-unchanged test "$portable_after" = "$portable_before"
classification_after=$(classification) || true
check_observed "$classification_after" state-valid-v1 test "$classification_after" = valid_v1
observe classification_after "$classification_after"
check project-readable bash -c 'cd "$HOME/cwd" && axiom --json project show --selector journey | grep -q "\"status\":\"success\""'
check execution-readable bash -c "cd \"\$HOME/cwd\" && axiom --json workflow status --project journey --repository main --number 7 | grep -q '$execution_id'"
check codex-ready bash -c 'cd "$HOME/cwd" && axiom runtime codex status'
check first-run bash -c 'cd "$HOME/cwd" && axiom first-run'
check first-run-no-op bash -c 'cd "$HOME/cwd" && axiom first-run | grep -q "runtime: claude present=true state=already_configured" && axiom first-run | grep -q "runtime: codex present=true state=already_configured"'
check rerun-installer install_release "$candidate"
rerun_status=$(installer_status) || true
observe installer_rerun "$rerun_status"
check_observed "$rerun_status" rerun-unchanged grep -qx 'install_status=unchanged' "$work/install.out"
printf 'journey=%s from=%s to=%s\n' "$journey" "$previous_version" "$candidate_version"

# Journey 2: every further earlier release -> candidate (skill and receipt
# convergence, issue #186), with a configured Project.
for index in "${!previous[@]}"; do
  [[ $index -eq 0 ]] && continue
  journey=skills-$(basename "${previous[$index]}")
  begin_journey "skills-$index" earlier_release "previous-$((index + 1))" -
  new_home "skills-$index"
  check install-previous install_release "${previous[$index]}"
  check first-run-previous bash -c 'cd "$HOME/cwd" && axiom first-run'
  check upgrade install_release "$candidate"
  upgrade_status=$(installer_status) || true
  observe installer_upgrade "$upgrade_status"
  check_observed "$upgrade_status" upgrade-status grep -qx 'install_status=upgraded' "$work/install.out"
  check codex-ready-without-first-run bash -c 'cd "$HOME/cwd" && axiom runtime codex status'
  check first-run bash -c 'cd "$HOME/cwd" && axiom first-run'
  check claude-ready bash -c 'cd "$HOME/cwd" && axiom runtime claude status'
  check rerun-unchanged bash -c "$(declare -f install_release release_bundle); row=$row work=$work H=$H; install_release '$candidate' && grep -qx install_status=unchanged '$work/install.out'"
  observe installer_rerun "$(installer_status)"
done

# Journey 3: historical POC state -> N -> candidate (RecognizedPOC
# preserve -> clean rebuild -> supported reconfiguration).
if [[ -n "$poc_binary" ]]; then
  journey=recognized-poc
  begin_journey recognized-poc historical_format poc-binary,previous-1 recognized-poc
  new_home poc
  poc() { (cd "$H/cwd" && "$poc_binary" --json "$@") >/dev/null; }
  seed_poc() {
    poc runtime codex install
    poc project configure --slug poc-project --name "POC Project" --repository "main=$H/repo"
    poc work-item create --project poc-project --repository main --title "POC item" --body historical --authorize-external
    poc workflow start --project poc-project --repository main --number 7
    for gate in specification clarification; do
      printf '%s\n' "$gate" >"$H/repo/$gate.md"
      poc workflow advance --project poc-project --repository main --number 7 --gate "$gate" --outcome pass --reference "$gate.md"
    done
  }
  check poc-creates-state seed_poc
  check install-previous install_release "${previous[0]}"
  check previous-classifies-recognized-poc test "$(classification)" = recognized_poc
  # N cannot publish edits to the capability-free historical Project.
  # Configure a separate v1 Project in the same state root through N.
  configure_v1_provider() {
    local configuration_digest configuration_id
    (cd "$H/cwd" && axiom --json project configure --slug v1-selected --name "V1 Selected" --repository "main=$H/repo" --work-item-provider github) >"$work/provider-preview.json"
    configuration_id=$(sed -n 's/.*"projectId":"\([^"]*\)".*/\1/p' "$work/provider-preview.json")
    configuration_digest=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$work/provider-preview.json")
    (cd "$H/cwd" && axiom --json project configure --project-id "$configuration_id" --slug v1-selected --name "V1 Selected" --repository "main=$H/repo" --work-item-provider github --preview-digest "$configuration_digest" --authorize-local) >"$work/provider.json"
    grep -q '"status":"success"' "$work/provider.json" || { cat "$work/provider-preview.json" "$work/provider.json"; return 1; }
  }
  check previous-configures-v1-provider configure_v1_provider
  # CR-005: select writes a legitimate v1 link without a create attempt,
  # leaving the POC classification intact. Use the installed release's CLI.
  select_v1_link() {
    local selection_digest
    (cd "$H/cwd" && axiom --json work-item select --project v1-selected --repository main --provider-repository owner/repo --number 42) >"$work/selection-preview.json"
    selection_digest=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$work/selection-preview.json")
    (cd "$H/cwd" && axiom --json work-item select --project v1-selected --repository main --provider-repository owner/repo --number 42 --preview-digest "$selection_digest" --authorize-local) >"$work/selection.json"
    grep -q '"status":"success"' "$work/selection.json"
  }
  check previous-selects-v1-link select_v1_link
  classification_before=$(classification) || true
  check selected-link-still-recognized-poc test "$classification_before" = recognized_poc
  observe classification_before "$classification_before"
  selected_file=$(python3 - "$STATE" <<'PYTHON'
import json, pathlib, sys
matches = [p for p in pathlib.Path(sys.argv[1], "work-items").rglob("*.json")
           if json.loads(p.read_text()).get("number") == 42]
assert len(matches) == 1
print(matches[0])
PYTHON
  )
  selected_digest=$(digest "$selected_file")
  inventory="$work/poc-inventory"
  for category in state projects; do
    root=$STATE; [[ $category == projects ]] && root=$PROJECTS
    (cd "$root" && find . -type f -print | LC_ALL=C sort | while IFS= read -r file; do printf '%s/%s %s %s\n' "$category" "${file#./}" "$(digest "$file")" "$(wc -c <"$file" | tr -d ' ')"; done)
  done | LC_ALL=C sort >"$inventory"
  workflow_file=$(cd "$STATE" && find workflows -type f | head -1)
  workflow_digest=$(digest "$STATE/$workflow_file")
  check upgrade install_release "$candidate"
  upgrade_status=$(installer_status) || true
  observe installer_upgrade "$upgrade_status"
  check_observed "$upgrade_status" upgrade-status grep -qx 'install_status=upgraded' "$work/install.out"
  archive=$(sed -n 's/^install_preserved=//p' "$work/install.out")
  check archive-reported test -n "$archive"
  if [[ -n "$evidence" && -n "$archive" && -f "$archive/manifest.json" ]]; then observe preservation_manifest_sha256 "$(digest "$archive/manifest.json")"; fi
  check archive-outside-active-roots bash -c "case '$archive' in '$STATE'*|'$PROJECTS'*) exit 1 ;; esac"
  manifest_inventory() {
    python3 - "$archive/manifest.json" <<'PY'
import json, sys
document = json.load(open(sys.argv[1]))
for item in sorted(document["objects"], key=lambda o: (o["category"], o["relative"])):
    print(f'{item["category"]}/{item["relative"]} {item["sha256"]} {item["bytes"]}')
PY
  }
  check manifest-corresponds-to-independent-inventory bash -c "diff <($(declare -f manifest_inventory); archive='$archive' manifest_inventory | LC_ALL=C sort) '$inventory'"
  check archived-objects-verify bash -c "while read -r path sha bytes; do test \"\$($(declare -f digest); digest '$archive/objects/'\$sha)\" = \"\$sha\" || exit 1; done <'$inventory'"
  check workflow-history-preserved test -f "$archive/objects/$workflow_digest"
  check workflow-history-not-active test ! -e "$STATE/workflows"
  classification_after=$(classification) || true
  check_observed "$classification_after" rebuilt-state-valid-v1 test "$classification_after" = valid_v1
  observe classification_after "$classification_after"
  selected_after=$(digest "$selected_file") || true
  check_observed "$selected_after" selected-link-byte-identical test "$selected_after" = "$selected_digest"
  selected_link_loads() {
    (cd "$H/cwd" && axiom --json work-item show --project v1-selected --repository main --provider-repository owner/repo --number 42) >"$work/selected-link.json"
    grep -q '"status":"success"' "$work/selected-link.json"
  }
  check selected-link-loads selected_link_loads
  check selected-link-manifest-kept python3 - "$archive/manifest.json" "$STATE" "$selected_file" <<'PYTHON'
import json, pathlib, sys
manifest = json.load(open(sys.argv[1]))
key = "state/" + pathlib.Path(sys.argv[3]).relative_to(sys.argv[2]).as_posix()
assert key in manifest["kept"] and key not in manifest["retired"]
assert len(manifest["retired"]) == 2
PYTHON
  check project-reconfigured bash -c 'cd "$HOME/cwd" && axiom project list | grep -q "project: poc-project"'
  check first-run bash -c 'cd "$HOME/cwd" && axiom first-run'
  archive_before=$(tree_digest "$archive")
  check rerun-installer install_release "$candidate"
  rerun_status=$(installer_status) || true
  observe installer_rerun "$rerun_status"
  check_observed "$rerun_status" rerun-unchanged grep -qx 'install_status=unchanged' "$work/install.out"
  archive_after=$(tree_digest "$archive") || true
  check_observed "$archive_after" archive-retained-unchanged test "$archive_after" = "$archive_before"
fi

end_journey
termination=completed
printf 'failures=%s\n' "$failures"
if [[ $failures -ne 0 ]]; then
  printf 'result=fail\n'
  exit 1
fi
printf 'result=pass\n'
