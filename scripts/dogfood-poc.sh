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

if lingo --json first-run >"$temporary/first-run-missing.json"; then
  exit 1
fi
assert_canonical "$temporary/first-run-missing.json" validation_failure "Codex skill compatibility is not ready"
run_success codex_configured "$temporary/runtime-install.json" runtime codex install
lingo --json first-run >"$temporary/runtime-status.json"
assert_canonical "$temporary/runtime-status.json" success "Lingo and Codex skills are compatible"
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
if lingo --json runtime codex status >"$temporary/runtime-missing.json"; then
  exit 1
fi
assert_canonical "$temporary/runtime-missing.json" validation_failure "Codex skill compatibility is not ready"
run_success codex_configured "$temporary/runtime-repair.json" runtime codex install

gh_binary="$temporary/gh"
provider_state="$temporary/provider-state"
printf '%s\n' OPEN >"$provider_state"
printf '%s\n' '#!/bin/sh' \
  'case "$*" in' \
  '  *search/issues*) printf "%s\n" "{\"total_count\":0,\"items\":[]}" ;;' \
  '  *issues/7/comments*) printf "%s\n" "{\"id\":1}" ;;' \
  '  *PATCH*issues/7*) printf "%s\n" CLOSED >"$AXIOM_FAKE_PROVIDER_STATE"; printf "%s\n" "{\"number\":7,\"html_url\":\"https://github.com/owner/repo/issues/7\",\"state\":\"closed\"}" ;;' \
  '  *issues/7*) value=$(sed -n "1p" "$AXIOM_FAKE_PROVIDER_STATE" | tr "[:upper:]" "[:lower:]"); printf "{\"number\":7,\"html_url\":\"https://github.com/owner/repo/issues/7\",\"state\":\"%s\"}\n" "$value" ;;' \
  '  *POST*repos/owner/repo/issues*) cat >/dev/null; printf "%s\n" "{\"number\":7,\"html_url\":\"https://github.com/owner/repo/issues/7\",\"state\":\"open\"}" ;;' \
  '  *) exit 1 ;;' \
  'esac' >"$gh_binary"
chmod 700 "$gh_binary"
export AXIOM_GH_BIN="$gh_binary"
export AXIOM_FAKE_PROVIDER_STATE="$provider_state"

lingo --json project configure --slug dogfood-project --name "Dogfood Project" \
  --repository "main=$repository" --work-item-provider github >"$temporary/project-preview.json"
assert_canonical "$temporary/project-preview.json" success "Project setup preview ready"
project_id=$(sed -n 's/.*"projectId":"\([^"]*\)".*/\1/p' "$temporary/project-preview.json")
preview_digest=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$temporary/project-preview.json")
[[ -n "$project_id" && -n "$preview_digest" ]]
lingo --json project configure --project-id "$project_id" --slug dogfood-project \
  --name "Dogfood Project" --repository "main=$repository" \
  --work-item-provider github --preview-digest "$preview_digest" --authorize-local \
  >"$temporary/project-configure.json"
assert_canonical "$temporary/project-configure.json" success "Project setup published"
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

draft_args=(work-item create --project dogfood-project --repository main \
  --provider-repository owner/repo --intent "Dogfood delivery is blocked" \
  --desired-outcome "Dogfood delivery proceeds safely" \
  --context "Synthetic deterministic E2E" --scope "Bounded Work Item change" \
  --constraints "Preserve exact authority" --non-goals "No implicit workflow" \
  --acceptance "Deterministic dogfood passes")
lingo --json "${draft_args[@]}" >"$temporary/work-item-preview.json"
assert_canonical "$temporary/work-item-preview.json" success "Work Item draft ready for review"
work_item_digest=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$temporary/work-item-preview.json")
[[ -n "$work_item_digest" ]]
run_canonical_failure work-item-stale denied_authority "Work Item authority denied" \
  "Review the exact preview and grant only the required authority" \
  "${draft_args[@]}" --preview-digest stale --authorize-external
lingo --json "${draft_args[@]}" --preview-digest "$work_item_digest" \
  --authorize-external >"$temporary/work-item.json"
assert_canonical "$temporary/work-item.json" success "GitHub Work Item linked"
grep -Fq '"externalId":"7"' "$temporary/work-item.json"
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
