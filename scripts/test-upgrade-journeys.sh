#!/usr/bin/env bash
# Release upgrade acceptance (Issues #153 and #186): installs real earlier
# releases into isolated homes, creates non-empty persisted state with those
# releases' own commands, upgrades through the candidate archive's own bundle
# facade, and asserts the resulting version, state, Runtime integration and
# idempotent rerun.
#
#   scripts/test-upgrade-journeys.sh --candidate ABS_DIR --previous ABS_DIR \
#     [--previous ABS_DIR ...] [--poc-binary ABS_FILE]
#
# --candidate and each --previous hold one release's SHA256SUMS and archives
# (as build-release-archives.sh or `gh release download` produce them). The
# first --previous is release N and runs the full v1 journey; every further
# --previous runs the skill-convergence journey from that release. With
# --poc-binary (a lingo built from tag v0.1.0-poc.1), the RecognizedPOC
# journey upgrades historical POC state through N to the candidate.
# Nothing touches the real home, Providers, or the network.
set -euo pipefail
umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
candidate=
poc_binary=
previous=()
while (($#)); do
  case "$1" in
    --candidate) candidate=${2:-}; shift 2 ;;
    --previous) previous+=("${2:-}"); shift 2 ;;
    --poc-binary) poc_binary=${2:-}; shift 2 ;;
    *) printf 'upgrade_journey_error: invalid argument\n' >&2; exit 2 ;;
  esac
done
[[ "$candidate" == /* && ${#previous[@]} -ge 1 ]] || { printf 'upgrade_journey_error: --candidate and --previous absolute directories required\n' >&2; exit 2; }
[[ -z "$poc_binary" || "$poc_binary" == /* ]] || { printf 'upgrade_journey_error: --poc-binary must be absolute\n' >&2; exit 2; }

case "$(uname -s):$(uname -m)" in
  Darwin:arm64) row=macos-27-arm64 ;;
  Linux:x86_64) row=linux-amd64 ;;
  Linux:aarch64) row=linux-arm64 ;;
  *) printf 'upgrade_journey_row=unsupported\nresult=blocked\n'; exit 78 ;;
esac

work=$(mktemp -d)
work=$(cd "$work" && pwd -P)
trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf -- "$work"' EXIT
failures=0
pass() { printf 'journey=%s step=%s result=pass\n' "$journey" "$1"; }
fail() { printf 'journey=%s step=%s result=fail\n' "$journey" "$1"; failures=$((failures + 1)); }
check() { local step=$1; shift; if "$@" >"$work/last.out" 2>&1; then pass "$step"; else fail "$step"; sed 's/^/  /' "$work/last.out" | head -20; fi; }

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
  local corpus="$repository_root/internal/compatibility/testdata/stable-v1/snapshots/v0.4.0"
  (cd "$corpus/state" && find . -type d -print) | while IFS= read -r directory; do mkdir -p "$STATE/$directory"; chmod 700 "$STATE/$directory"; done
  (cd "$corpus/state" && find . -type f -print) | while IFS= read -r file; do cp "$corpus/state/$file" "$STATE/$file"; chmod 600 "$STATE/$file"; done
  mkdir -p "$PROJECTS/sample" && chmod 700 "$PROJECTS/sample"
  cp "$corpus/projects/sample/axiom.yaml" "$PROJECTS/sample/axiom.yaml" && chmod 600 "$PROJECTS/sample/axiom.yaml"
}

classification() { (cd "$H/cwd" && axiom --json compatibility inspect) | sed -n 's/.*"classification":"\([^"]*\)".*/\1/p' | head -1; }

# Journey 1: release N with non-empty v1 state -> candidate (direct).
journey=v1-state-$(basename "${previous[0]}")
new_home v1
check install-previous install_release "${previous[0]}"
previous_version=$(version_of)
check first-run-previous bash -c 'cd "$HOME/cwd" && axiom first-run'
check seed-state-with-previous seed_v1_state
check previous-reads-seeded-state-as-v1 test "$(classification)" = valid_v1
state_before=$(tree_digest "$STATE")
portable_before=$(tree_digest "$PROJECTS")
check upgrade install_release "$candidate"
check upgrade-status grep -qx 'install_status=upgraded' "$work/install.out"
check version test "$(version_of)" = "$candidate_version"
check state-bytes-unchanged test "$(tree_digest "$STATE")" = "$state_before"
check portable-bytes-unchanged test "$(tree_digest "$PROJECTS")" = "$portable_before"
check state-valid-v1 test "$(classification)" = valid_v1
check project-readable bash -c 'cd "$HOME/cwd" && axiom --json project show --selector journey | grep -q "\"status\":\"success\""'
check execution-readable bash -c "cd \"\$HOME/cwd\" && axiom --json workflow status --project journey --repository main --number 7 | grep -q '$execution_id'"
check codex-ready bash -c 'cd "$HOME/cwd" && axiom runtime codex status'
check first-run bash -c 'cd "$HOME/cwd" && axiom first-run'
check first-run-no-op bash -c 'cd "$HOME/cwd" && axiom first-run | grep -q "runtime: claude present=true state=already_configured" && axiom first-run | grep -q "runtime: codex present=true state=already_configured"'
check rerun-installer install_release "$candidate"
check rerun-unchanged grep -qx 'install_status=unchanged' "$work/install.out"
printf 'journey=%s from=%s to=%s\n' "$journey" "$previous_version" "$candidate_version"

# Journey 2: every further earlier release -> candidate (skill and receipt
# convergence, issue #186), with a configured Project.
for index in "${!previous[@]}"; do
  [[ $index -eq 0 ]] && continue
  journey=skills-$(basename "${previous[$index]}")
  new_home "skills-$index"
  check install-previous install_release "${previous[$index]}"
  check first-run-previous bash -c 'cd "$HOME/cwd" && axiom first-run'
  check upgrade install_release "$candidate"
  check upgrade-status grep -qx 'install_status=upgraded' "$work/install.out"
  check codex-ready-without-first-run bash -c 'cd "$HOME/cwd" && axiom runtime codex status'
  check first-run bash -c 'cd "$HOME/cwd" && axiom first-run'
  check claude-ready bash -c 'cd "$HOME/cwd" && axiom runtime claude status'
  check rerun-unchanged bash -c "$(declare -f install_release release_bundle); row=$row work=$work H=$H; install_release '$candidate' && grep -qx install_status=unchanged '$work/install.out'"
done

# Journey 3: historical POC state -> N -> candidate (RecognizedPOC
# preserve -> clean rebuild -> supported reconfiguration).
if [[ -n "$poc_binary" ]]; then
  journey=recognized-poc
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
  inventory="$work/poc-inventory"
  for category in state projects; do
    root=$STATE; [[ $category == projects ]] && root=$PROJECTS
    (cd "$root" && find . -type f -print | LC_ALL=C sort | while IFS= read -r file; do printf '%s/%s %s %s\n' "$category" "${file#./}" "$(digest "$file")" "$(wc -c <"$file" | tr -d ' ')"; done)
  done | LC_ALL=C sort >"$inventory"
  workflow_file=$(cd "$STATE" && find workflows -type f | head -1)
  workflow_digest=$(digest "$STATE/$workflow_file")
  check upgrade install_release "$candidate"
  check upgrade-status grep -qx 'install_status=upgraded' "$work/install.out"
  archive=$(sed -n 's/^install_preserved=//p' "$work/install.out")
  check archive-reported test -n "$archive"
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
  check rebuilt-state-valid-v1 test "$(classification)" = valid_v1
  check project-reconfigured bash -c 'cd "$HOME/cwd" && axiom project list | grep -q "project: poc-project"'
  check first-run bash -c 'cd "$HOME/cwd" && axiom first-run'
  archive_before=$(tree_digest "$archive")
  check rerun-installer install_release "$candidate"
  check rerun-unchanged grep -qx 'install_status=unchanged' "$work/install.out"
  check archive-retained-unchanged test "$(tree_digest "$archive")" = "$archive_before"
fi

printf 'failures=%s\n' "$failures"
if [[ $failures -ne 0 ]]; then
  printf 'result=fail\n'
  exit 1
fi
printf 'result=pass\n'
