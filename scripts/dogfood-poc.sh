#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
if [[ -z "$temporary" || ! -d "$temporary" ]]; then
  exit 1
fi
trap 'rm -rf -- "$temporary"' EXIT

binary_root="$temporary/bin"
install_state="$temporary/install-state"
portable_root="$temporary/projects"
state_root="$temporary/state"
skills_root="$temporary/global-skills"
unrelated="$temporary/unrelated-cwd"
repository="$temporary/project-repository"
mkdir -p "$unrelated" "$repository"

AXIOM_BIN_DIR="$binary_root" AXIOM_INSTALL_STATE_ROOT="$install_state" \
  "$repository_root/scripts/install-axiom.sh" >"$temporary/install.json"
export PATH="$binary_root:$PATH"
export LINGO_PROJECTS_ROOT="$portable_root"
export LINGO_STATE_ROOT="$state_root"
export AXIOM_CODEX_SKILLS_ROOT="$skills_root"

run_success() {
  local category=$1
  local destination=$2
  shift 2
  lingo --json "$@" >"$destination"
  grep -q '"status":"success"' "$destination"
  grep -q '"category":"'"$category"'"' "$destination"
}

run_failure() {
  local category=$1
  shift
  local output="$temporary/failure-$category.json"
  if lingo --json "$@" >"$output"; then
    exit 1
  fi
  grep -q '"status":"error"' "$output"
  grep -q '"category":"'"$category"'"' "$output"
}

assert_canonical() {
  local output=$1
  local status=$2
  local result=$3
  grep -Fq '"status":"'"$status"'"' "$output"
  grep -Fq '"result":"'"$result"'"' "$output"
  grep -Fq '"provenance":{"product":"Axiom","version":"development","revision":"' "$output"
  grep -Fq '"sourceState":"' "$output"
  if grep -Fq '"category":' "$output"; then
    exit 1
  fi
}

run_canonical_failure() {
  local label=$1
  local status=$2
  local result=$3
  local next=$4
  shift 4
  local output="$temporary/canonical-failure-$label.json"
  if lingo --json "$@" >"$output"; then
    exit 1
  fi
  assert_canonical "$output" "$status" "$result"
  grep -Fq '"next":"'"$next"'"' "$output"
}

cd "$unrelated"
resolved_binary=$(command -v lingo)
if [[ "$resolved_binary" != "$binary_root/lingo" ]]; then
  exit 1
fi
lingo --json version >"$temporary/version.json"
assert_canonical "$temporary/version.json" success "Axiom build information"

run_failure codex_not_configured runtime codex status
run_success codex_configured "$temporary/runtime-install.json" runtime codex install
run_success codex_ready "$temporary/runtime-status.json" runtime codex status
skill_count=$(find "$skills_root" -name SKILL.md -type f | wc -l | tr -d ' ')
if [[ "$skill_count" != 5 ]]; then
  exit 1
fi
lingo help >"$temporary/help.txt"
for skill in axiom-project-configure axiom-project-show axiom-work-item-create axiom-work-item-run axiom-work-item-status; do
  grep -q "\$${skill}" "$temporary/help.txt"
done
rm -- "$skills_root/axiom-work-item-status/SKILL.md"
rmdir -- "$skills_root/axiom-work-item-status"
run_failure codex_skills_missing_or_changed runtime codex status
run_success codex_configured "$temporary/runtime-repair.json" runtime codex install

git_binary="$temporary/git"
gh_binary="$temporary/gh"
provider_state="$temporary/provider-state"
printf '%s\n' OPEN >"$provider_state"
printf '%s\n' '#!/bin/sh' "printf '%s\\n' 'git@github.com:owner/repo.git'" >"$git_binary"
printf '%s\n' '#!/bin/sh' \
  'if [ "$1" = issue ] && [ "$2" = create ]; then printf "%s\n" "https://github.com/owner/repo/issues/7"; exit 0; fi' \
  'if [ "$1" = issue ] && [ "$2" = view ]; then value=$(sed -n "1p" "$AXIOM_FAKE_PROVIDER_STATE"); printf "{\"Number\":7,\"URL\":\"https://github.com/owner/repo/issues/7\",\"State\":\"%s\"}\n" "$value"; exit 0; fi' \
  'if [ "$1" = issue ] && [ "$2" = comment ]; then printf "%s\n" ok; exit 0; fi' \
  'if [ "$1" = issue ] && [ "$2" = close ]; then printf "%s\n" CLOSED >"$AXIOM_FAKE_PROVIDER_STATE"; printf "%s\n" ok; exit 0; fi' \
  'exit 1' >"$gh_binary"
chmod 700 "$git_binary" "$gh_binary"
export AXIOM_GIT_BIN="$git_binary"
export AXIOM_GH_BIN="$gh_binary"
export AXIOM_FAKE_PROVIDER_STATE="$provider_state"

run_success project_configured "$temporary/project-configure.json" project configure \
  --slug dogfood-project --name "Dogfood Project" --repository "main=$repository"
lingo --json project show --selector dogfood-project >"$temporary/project-show.json"
assert_canonical "$temporary/project-show.json" success "Project resolved"
grep -Fq '"references":["project:' "$temporary/project-show.json"
grep -Fq '"repository:main"' "$temporary/project-show.json"
if grep -Fq '"path":' "$temporary/project-show.json"; then
  exit 1
fi
run_canonical_failure project-not-found validation_failure "Project was not found" \
  "Provide an existing Project UUID or slug" project show --selector missing-project
mv -- "$repository" "$temporary/moved-repository"
run_canonical_failure repository-unavailable retryable_failure \
  "Project repository is unavailable" \
  "Restore the configured repository binding and retry inspection" \
  project show --selector dogfood-project
mv -- "$temporary/moved-repository" "$repository"

run_failure external_mutation_denied work-item create --project dogfood-project \
  --repository main --title "POC dogfood"
run_success work_item_linked "$temporary/work-item.json" work-item create \
  --project dogfood-project --repository main --title "POC dogfood" \
  --body "Synthetic deterministic E2E" --authorize-external
run_success workflow_started "$temporary/workflow-start.json" workflow start \
  --project dogfood-project --repository main --number 7

for gate in specification clarification plan tasks; do
  printf '%s\n' "$gate" >"$repository/$gate.md"
  run_success workflow_advanced "$temporary/workflow-$gate.json" workflow advance \
    --project dogfood-project --repository main --number 7 --gate "$gate" \
    --outcome pass --reference "$gate.md"
done

printf '%s\n' implementation >"$repository/implementation.md"
run_failure workflow_interrupted workflow advance --project dogfood-project \
  --repository main --number 7 --gate implementation --outcome fail \
  --reference implementation.md
run_success workflow_interrupted "$temporary/workflow-interrupted.json" workflow status \
  --project dogfood-project --repository main --number 7
run_failure workflow_resume_required workflow advance --project dogfood-project \
  --repository main --number 7 --gate implementation --outcome pass \
  --reference implementation.md
run_success workflow_resumed "$temporary/workflow-resume.json" workflow resume \
  --project dogfood-project --repository main --number 7

for gate in implementation review evidence reconciliation; do
  printf '%s\n' "$gate" >"$repository/$gate.md"
  run_success workflow_advanced "$temporary/workflow-$gate.json" workflow advance \
    --project dogfood-project --repository main --number 7 --gate "$gate" \
    --outcome pass --reference "$gate.md"
done

run_success workflow_evidence_ready "$temporary/workflow-evidence.json" workflow evidence \
  --project dogfood-project --repository main --number 7
grep -q '"currentGate":"completion"' "$temporary/workflow-evidence.json"
run_failure external_mutation_denied workflow advance --project dogfood-project \
  --repository main --number 7 --gate completion --outcome pass
run_success workflow_completed "$temporary/workflow-completed.json" workflow advance \
  --project dogfood-project --repository main --number 7 --gate completion \
  --outcome pass --authorize-external
if [[ $(sed -n '1p' "$provider_state") != CLOSED ]]; then
  exit 1
fi

binary_sha=$(shasum -a 256 "$resolved_binary" | awk '{print $1}')
workflow_record=$(find "$state_root/workflows" -name '*.json' -type f -print)
workflow_count=$(printf '%s\n' "$workflow_record" | grep -c .)
if [[ "$workflow_count" != 1 ]]; then
  exit 1
fi
workflow_sha=$(shasum -a 256 "$workflow_record" | awk '{print $1}')
printf '{"evidenceVersion":1,"evidence":"axiom_e2e_dogfood","cwdIndependent":true,"globalSkillCount":%s,"workItem":7,"workflow":"completed","binarySha256":"%s","workflowSha256":"%s","result":"pass"}\n' \
  "$skill_count" "$binary_sha" "$workflow_sha"
