#!/usr/bin/env bash
set -Eeuo pipefail

# Print script source only: never expand arguments or dump temporary outputs.
trap 'failure_code=$?; printf "dogfood failure: line=%s command=%q exit_code=%s\n" "$LINENO" "$BASH_COMMAND" "$failure_code" >&2' ERR

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
(cd "$repository_root" && go build -o "$temporary/dogfoodfixture" ./scripts/dogfoodfixture)
export PATH="$binary_root:$PATH"
export LINGO_PROJECTS_ROOT="$portable_root"
export LINGO_STATE_ROOT="$state_root"
export AXIOM_CODEX_SKILLS_ROOT="$skills_root"

run_success() {
  local category=$1
  local destination=$2
  shift 2
  axiom --json "$@" >"$destination"
  grep -q '"status":"success"' "$destination"
  grep -q '"category":"'"$category"'"' "$destination"
}

run_failure() {
  local category=$1
  shift
  local output="$temporary/failure-$category.json"
  if axiom --json "$@" >"$output"; then
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

# Parse repository fields structurally, including paths requiring JSON escaping.
assert_project_repository() {
  python3 - "$1" "$project_id" "$configured_repository" "$2" <<'PYJSON'
import json
import sys

output, project_id, path, availability = sys.argv[1:]
with open(output, encoding="utf-8") as source:
    document = json.load(source)
assert document["status"] == "success"
assert document["project"]["id"] == project_id
assert document["project"]["repositories"] == [
    {"key": "main", "path": path, "availability": availability}
]
assert "project:" + project_id in document["references"]
assert "repository:main" in document["references"]
PYJSON
}

run_canonical_failure() {
  local label=$1
  local status=$2
  local result=$3
  local next=$4
  shift 4
  local output="$temporary/canonical-failure-$label.json"
  if axiom --json "$@" >"$output"; then
    exit 1
  fi
  assert_canonical "$output" "$status" "$result"
  grep -Fq '"next":"'"$next"'"' "$output"
}

cd "$unrelated"
resolved_binary=$(command -v axiom)
if [[ "$resolved_binary" != "$binary_root/axiom" ]]; then
  exit 1
fi
axiom --json version >"$temporary/version.json"
assert_canonical "$temporary/version.json" success "Axiom build information"

# first-run discovers Runtimes from PATH, HOME and CLAUDE_CONFIG_DIR; isolate
# them so the host's real Codex or Claude is never seen or touched.
runtime_bin="$temporary/runtime-bin"
runtime_home="$temporary/runtime-home"
mkdir -p "$runtime_bin" "$runtime_home"
first_run() {
  env -u CLAUDE_CONFIG_DIR HOME="$runtime_home" PATH="$runtime_bin:$binary_root" axiom --json first-run
}
first_run >"$temporary/first-run-none.json"
assert_canonical "$temporary/first-run-none.json" success "No supported Runtime is currently available"
if axiom --json runtime codex status >"$temporary/first-run-missing.json"; then
  exit 1
fi
assert_canonical "$temporary/first-run-missing.json" validation_failure "Codex skill compatibility is not ready"
printf '#!/bin/sh\nexit 99\n' >"$runtime_bin/codex"
chmod 700 "$runtime_bin/codex"
first_run >"$temporary/runtime-install.json"
assert_canonical "$temporary/runtime-install.json" success "Axiom integration is configured for every detected Runtime"
grep -q '"runtime":"codex","executable":"codex","present":true,"configurationWithoutExecutable":false,"state":"configured","reason":"codex_configured"' "$temporary/runtime-install.json"
axiom --json runtime codex status >"$temporary/runtime-status.json"
assert_canonical "$temporary/runtime-status.json" success "Lingo and Codex skills are compatible"
if [[ -n $(find "$runtime_home" -mindepth 1 -print -quit) ]]; then
  exit 1
fi
skill_count=$(find "$skills_root" -name SKILL.md -type f | wc -l | tr -d ' ')
if [[ "$skill_count" != 2 ]]; then
  exit 1
fi
axiom help >"$temporary/help.txt"
for skill in axiom-project axiom-work-item; do
  grep -q "\$${skill}" "$temporary/help.txt"
done
rm -- "$skills_root/axiom-work-item/SKILL.md"
rmdir -- "$skills_root/axiom-work-item"
if axiom --json runtime codex status >"$temporary/runtime-missing.json"; then
  exit 1
fi
assert_canonical "$temporary/runtime-missing.json" validation_failure "Codex skill compatibility is not ready"
run_success codex_configured "$temporary/runtime-repair.json" runtime codex install

gh_binary="$temporary/gh"
provider_label="$temporary/provider-label"
provider_comment="$temporary/provider-comment.json"
: >"$provider_label"
: >"$provider_comment"
printf '%s\n' '#!/bin/sh' \
  'case "$*" in' \
  '  *search/issues*) printf "%s\n" "{\"total_count\":0,\"items\":[]}" ;;' \
  '*POST*repos/owner/repo/labels*) input=$(cat); value=$(printf "%s" "$input" | sed -n "s/.*\"name\":\"\([^\"]*\)\".*/\1/p"); printf "%s\n" "$value" >"$AXIOM_FAKE_PROVIDER_LABEL"; printf "%s\n" "$input" ;;' \
  '*repos/owner/repo/labels?per_page=100*) if [ -s "$AXIOM_FAKE_PROVIDER_LABEL" ]; then value=$(sed -n "1p" "$AXIOM_FAKE_PROVIDER_LABEL"); printf "[{\"name\":\"%s\"}]\n" "$value"; else printf "%s\n" "[]"; fi ;;' \
  '*POST*issues/7/comments*) cat >"$AXIOM_FAKE_PROVIDER_COMMENT"; printf "%s\n" "{\"id\":1}" ;;' \
  '*issues/7/comments?per_page=100*) if [ -s "$AXIOM_FAKE_PROVIDER_COMMENT" ]; then printf "["; cat "$AXIOM_FAKE_PROVIDER_COMMENT"; printf "]\n"; else printf "%s\n" "[]"; fi ;;' \
  '*POST*issues/7/labels*) input=$(cat); value=$(printf "%s" "$input" | sed -n "s/.*\"labels\":\[\"\([^\"]*\)\"\].*/\1/p"); printf "%s\n" "$value" >"$AXIOM_FAKE_PROVIDER_LABEL"; printf "%s\n" "$input" ;;' \
  '*DELETE*issues/7/labels/*) : >"$AXIOM_FAKE_PROVIDER_LABEL"; printf "%s\n" "{}" ;;' \
  '*issues/7*) if [ -s "$AXIOM_FAKE_PROVIDER_LABEL" ]; then value=$(sed -n "1p" "$AXIOM_FAKE_PROVIDER_LABEL"); labels="[{\"name\":\"$value\"}]"; else labels="[]"; fi; printf "{\"number\":7,\"html_url\":\"https://github.com/owner/repo/issues/7\",\"state\":\"open\",\"labels\":%s}\n" "$labels" ;;' \
  '  *POST*repos/owner/repo/issues*) cat >/dev/null; printf "%s\n" "{\"number\":7,\"html_url\":\"https://github.com/owner/repo/issues/7\",\"state\":\"open\"}" ;;' \
  '  *) exit 1 ;;' \
  'esac' >"$gh_binary"
chmod 700 "$gh_binary"
export AXIOM_GH_BIN="$gh_binary"
export AXIOM_FAKE_PROVIDER_LABEL="$provider_label"
export AXIOM_FAKE_PROVIDER_COMMENT="$provider_comment"

# project configure publishes Projects without a Runtime/Profile policy; this
# one proves the guided setup and, later, that workflow start never falls back.
configured_repository="$temporary/configured-repository"
mkdir -p "$configured_repository"
axiom --json project configure --slug dogfood-configured --name "Dogfood Configured" \
  --repository "main=$configured_repository" --work-item-provider github >"$temporary/project-preview.json"
assert_canonical "$temporary/project-preview.json" success "Project setup preview ready"
project_id=$(sed -n 's/.*"projectId":"\([^"]*\)".*/\1/p' "$temporary/project-preview.json")
preview_digest=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$temporary/project-preview.json")
[[ -n "$project_id" && -n "$preview_digest" ]]
axiom --json project configure --project-id "$project_id" --slug dogfood-configured \
  --name "Dogfood Configured" --repository "main=$configured_repository" \
  --work-item-provider github --preview-digest "$preview_digest" --authorize-local \
  >"$temporary/project-configure.json"
assert_canonical "$temporary/project-configure.json" success "Project setup published"
axiom --json project show --selector dogfood-configured >"$temporary/project-show.json"
assert_canonical "$temporary/project-show.json" success "Project resolved"
assert_project_repository "$temporary/project-show.json" available
run_canonical_failure project-not-found validation_failure "Project was not found" \
  "Provide an existing Project UUID or slug" project show --selector missing-project
mv -- "$configured_repository" "$temporary/moved-repository"
axiom --json project show --selector dogfood-configured >"$temporary/project-unavailable.json"
assert_canonical "$temporary/project-unavailable.json" success "Project resolved"
assert_project_repository "$temporary/project-unavailable.json" unavailable
mv -- "$temporary/moved-repository" "$configured_repository"
axiom --json project show --selector dogfood-configured >"$temporary/project-restored.json"
assert_canonical "$temporary/project-restored.json" success "Project resolved"
assert_project_repository "$temporary/project-restored.json" available

# The delivery Project carries an operator-authored portable Runtime/Profile
# policy: both concrete Runtimes are allowed and nothing is a default. Install
# records it only with an exact binding for every declared Repository.
authored="$portable_root/dogfood-project"
mkdir -p "$authored"
cat >"$authored/axiom.yaml" <<'MANIFEST'
schemaVersion: 2
project:
  id: 5f0c7a52-1b7e-4c1d-9a3e-0d6f2b8c4e19
  slug: dogfood-project
  name: Dogfood Project
repositories:
  - key: main
providers:
  - key: work-items
    id: github
integrations:
  - key: work-items
    providerRef: work-items
    capabilities:
      - work-item
runtimes:
  - id: claude
  - id: codex
modelProfiles:
  - key: careful
    runtimeRef: claude
    model: approved-claude-model
  - key: worker
    runtimeRef: codex
    model: approved-codex-model
MANIFEST
run_failure invalid_repository_bindings project install --source "$authored"
run_success installed "$temporary/project-install.json" project install --source "$authored" \
  --repository "main=$repository"
axiom --json project show --selector dogfood-project >"$temporary/project-authored.json"
assert_canonical "$temporary/project-authored.json" success "Project resolved"

# Machine-local Runtime Profile configuration stays outside portable intent.
# Profiles may only need what Lingo can prove itself: axiom-skills.
mkdir -p "$state_root/runtime-profiles/v1"
printf '%s\n' '{"formatVersion":1,"revision":1,"runtimes":[{"id":"claude","adapter":"claude","enabled":true,"allowlistedProfileIds":["careful"]},{"id":"codex","adapter":"codex","enabled":true,"allowlistedProfileIds":["worker"]}],"modelProfiles":[{"id":"careful","runtimeId":"claude","model":"approved-claude-model","capabilities":["axiom-skills"],"complexities":["high"]},{"id":"worker","runtimeId":"codex","model":"approved-codex-model","capabilities":["axiom-skills"],"complexities":["high"]}]}' \
  >"$state_root/runtime-profiles/v1/configuration.json"
axiom --json runtime profile validate >"$temporary/runtime-profile-validate.json"
assert_canonical "$temporary/runtime-profile-validate.json" success "Runtime profile configuration is valid"

draft_args=(work-item create --project dogfood-project --repository main \
  --provider-repository owner/repo --intent "Dogfood delivery is blocked" \
  --desired-outcome "Dogfood delivery proceeds safely" \
  --context "Synthetic deterministic E2E" --scope "Bounded Work Item change" \
  --constraints "Preserve exact authority" --non-goals "No implicit workflow" \
  --acceptance "Deterministic dogfood passes")
axiom --json "${draft_args[@]}" >"$temporary/work-item-preview.json"
assert_canonical "$temporary/work-item-preview.json" success "Work Item draft ready for review"
work_item_digest=$(sed -n 's/.*"digest":"\([^"]*\)".*/\1/p' "$temporary/work-item-preview.json")
[[ -n "$work_item_digest" ]]
run_canonical_failure work-item-stale denied_authority "Work Item authority denied" \
  "Review the exact preview and grant only the required authority" \
  "${draft_args[@]}" --preview-digest stale --authorize-external
axiom --json "${draft_args[@]}" --preview-digest "$work_item_digest" \
  --authorize-external >"$temporary/work-item.json"
assert_canonical "$temporary/work-item.json" success "GitHub Work Item linked"
grep -Fq '"externalId":"7"' "$temporary/work-item.json"
# Lingo observes Runtimes itself, so start/resume use the isolated Runtime PATH:
# only the codex stub (never executed) and the Codex skills installed above are
# visible; the host's real Codex or Claude never is. The fake Provider needs
# only sed and cat.
tool_bin="$temporary/tool-bin"
mkdir -p "$tool_bin"
ln -s "$(command -v sed)" "$tool_bin/sed"
ln -s "$(command -v cat)" "$tool_bin/cat"
observed_axiom() {
  env -u CLAUDE_CONFIG_DIR HOME="$runtime_home" PATH="$runtime_bin:$binary_root:$tool_bin" axiom --json "$@"
}
assert_no_execution() {
  if [[ -d "$state_root/executions" && -n $(find "$state_root/executions" -name '*.json' -type f -print -quit) ]]; then
    exit 1
  fi
}
assert_runtime_blocked() {
  local label=$1
  local code=$2
  shift 2
  local output="$temporary/runtime-blocked-$label.json"
  if observed_axiom "$@" >"$output"; then
    exit 1
  fi
  assert_canonical "$output" validation_failure "Project Runtime policy resolution blocked"
  grep -Fq '"blocker":{"code":"'"$code"'"' "$output"
  if grep -Fq '"choice":' "$output"; then
    exit 1
  fi
  assert_no_execution
}
policy_inputs=(--role implementation --complexity high --capabilities axiom-skills)
# As the axiom-work-item skill prescribes, --runtime names the Runtime that
# conducts the workflow; it narrows the policy and never widens it.
start_args=(workflow start --project dogfood-project --repository main --number 7 "${policy_inputs[@]}" --runtime codex)

# Target validation now precedes Runtime policy review (#137). Link the explicit
# synthetic issue into this second Project before testing its absent policy;
# otherwise the test would stop at work_item_not_linked instead of policy.
selection_args=(work-item select --project dogfood-configured --repository main --provider-repository owner/repo --number 7)
axiom --json "${selection_args[@]}" >"$temporary/unconfigured-selection-preview.json"
assert_canonical "$temporary/unconfigured-selection-preview.json" success "GitHub Work Item selection ready for review"
selection_digest=$(sed -n 's/.*"digest":"\([0-9a-f]*\)".*/\1/p' "$temporary/unconfigured-selection-preview.json")
[[ ${#selection_digest} == 64 ]]
axiom --json "${selection_args[@]}" --preview-digest "$selection_digest" --authorize-local >"$temporary/unconfigured-selection.json"
assert_canonical "$temporary/unconfigured-selection.json" success "GitHub Work Item linked"

# Select the builtin revision explicitly before reviewing Runtime policy.
"$temporary/dogfoodfixture" definition >"$temporary/definition.json"
read -r workflow_id workflow_digest < <(python3 -c 'import json,sys; w=json.load(open(sys.argv[1])); print(w["workflowId"],w["digest"])' "$temporary/definition.json")
for selected_project in dogfood-project dogfood-configured; do
  workflow_selection=(project workflow select --project "$selected_project" --workflow "$workflow_id" --revision 1 --digest "$workflow_digest" --source builtin)
  run_success previewed "$temporary/workflow-selection.json" "${workflow_selection[@]}"
  read -r project_revision selection_preview < <(python3 -c 'import json,sys; w=json.load(open(sys.argv[1]))["workflowAuthoring"]; print(w["projectRevision"], w["previewDigest"])' "$temporary/workflow-selection.json")
  run_success applied "$temporary/workflow-selected.json" "${workflow_selection[@]}" --expected-revision "$project_revision" --preview-digest "$selection_preview" --authorize-local
done

# No configured policy, no explicitly allowed-and-observed Runtime and no
# capability Lingo cannot prove ever falls back to Codex.
assert_runtime_blocked unconfigured policy_unconfigured \
  workflow start --project dogfood-configured --repository main --number 7 "${policy_inputs[@]}"
assert_runtime_blocked claude-unobserved no_allowed_match \
  workflow start --project dogfood-project --repository main --number 7 "${policy_inputs[@]}" --runtime claude
assert_runtime_blocked unprovable no_allowed_match \
  workflow start --project dogfood-project --repository main --number 7 \
  --role implementation --complexity high --capabilities go

# 1. Preview: the first start only reviews the decision.
observed_axiom "${start_args[@]}" >"$temporary/workflow-runtime-preview.json"
assert_canonical "$temporary/workflow-runtime-preview.json" success "Project Runtime resolution preview ready"
grep -Fq '"choice":{"runtimeId":"codex","adapter":"codex","modelProfileId":"worker","model":"approved-codex-model","capabilities":["axiom-skills"],"executableDigest":"' "$temporary/workflow-runtime-preview.json"
grep -Fq '"projectId":"5f0c7a52-1b7e-4c1d-9a3e-0d6f2b8c4e19"' "$temporary/workflow-runtime-preview.json"
if grep -Fq "$temporary" "$temporary/workflow-runtime-preview.json" || grep -Fq '"workflow":' "$temporary/workflow-runtime-preview.json"; then
  exit 1
fi
runtime_preview=$(sed -n 's/.*"previewDigest":"\([0-9a-f]*\)".*/\1/p' "$temporary/workflow-runtime-preview.json")
[[ ${#runtime_preview} == 64 ]]
assert_no_execution

# 2. A Runtime executable replaced after review no longer matches the digest.
cp -- "$runtime_bin/codex" "$temporary/codex-reviewed"
printf '%s\n' '# replaced after review' >>"$runtime_bin/codex"
assert_runtime_blocked replaced stale_preview "${start_args[@]}" --runtime-preview "$runtime_preview"
cp -- "$temporary/codex-reviewed" "$runtime_bin/codex"
# A blocker means a fresh preview; the restored executable reproduces the
# reviewed identity, so the decision and its digest are the same.
observed_axiom "${start_args[@]}" >"$temporary/workflow-runtime-repreview.json"
assert_canonical "$temporary/workflow-runtime-repreview.json" success "Project Runtime resolution preview ready"
fresh_preview=$(sed -n 's/.*"previewDigest":"\([0-9a-f]*\)".*/\1/p' "$temporary/workflow-runtime-repreview.json")
[[ "$fresh_preview" == "$runtime_preview" ]]

# 3. The exact reviewed decision, revalidated now, creates the Execution.
observed_axiom "${start_args[@]}" --runtime-preview "$fresh_preview" >"$temporary/workflow-start.json"
assert_canonical "$temporary/workflow-start.json" success "Execution workflow operation completed"
grep -Fq '"runtimeId":"codex"' "$temporary/workflow-start.json"
execution_id=$(sed -n 's/.*"executionId":"\([^"]*\)".*/\1/p' "$temporary/workflow-start.json")
[[ -n "$execution_id" ]]
revision=1

# Typed synthetic outputs use the production artifact store, without editing
# canonical Execution records or manufacturing validator results.
advance_configured() {
  local gate=$1
  axiom --json workflow status --project dogfood-project --repository main --number 7 >"$temporary/current-stage.json"
  "$temporary/dogfoodfixture" result "$state_root" <"$temporary/current-stage.json" >"$temporary/stage-result.json"
  axiom --json workflow advance --project dogfood-project --repository main --number 7 \
    --expected-revision "$revision" --gate "$gate" --outcome pass \
    --stage-result "$temporary/stage-result.json" >"$temporary/workflow-$gate.json"
  assert_canonical "$temporary/workflow-$gate.json" success "Execution workflow operation completed"
}

for gate in intake specification clarification; do
  printf '%s\n' "$gate" >"$repository/$gate.md"
  reference_digest=$(shasum -a 256 "$repository/$gate.md" | awk '{print $1}')
  advance_configured "$gate"
  revision=$((revision + 1))
done

specification_digest=$(shasum -a 256 "$repository/specification.md" | awk '{print $1}')
axiom --json workflow fact --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" --fact planning-authority --active --actor maintainer \
  --reference "specification:specification.md:$specification_digest" --authorize-local \
  >"$temporary/workflow-planning-authority.json"
assert_canonical "$temporary/workflow-planning-authority.json" success "Execution workflow operation completed"
revision=$((revision + 1))

for gate in plan tasks; do
  printf '%s\n' "$gate" >"$repository/$gate.md"
  reference_digest=$(shasum -a 256 "$repository/$gate.md" | awk '{print $1}')
  advance_configured "$gate"
  revision=$((revision + 1))
done

plan_digest=$(shasum -a 256 "$repository/plan.md" | awk '{print $1}')
axiom --json workflow fact --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" --fact implementation-authority --active --actor maintainer \
  --reference "plan:plan.md:$plan_digest" --authorize-local \
  >"$temporary/workflow-implementation-authority.json"
assert_canonical "$temporary/workflow-implementation-authority.json" success "Execution workflow operation completed"
revision=$((revision + 1))

# S6 treats zero lifecycle markers as Provider drift. Seed the bounded fake with
# the exact already-aligned observation; deterministic adapter tests cover label
# creation/replacement effects separately.
printf '%s\n' 'axiom:stage:implementing' >"$provider_label"
axiom --json workflow reconcile --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" >"$temporary/workflow-projection-preview.json"
assert_canonical "$temporary/workflow-projection-preview.json" success "Execution workflow operation completed"
projection_digest=$(sed -n 's/.*"projection":.*"digest":"\([^"]*\)".*/\1/p' "$temporary/workflow-projection-preview.json")
projection_key=$(sed -n 's/.*"projectionKey":"\([^"]*\)".*/\1/p' "$temporary/workflow-projection-preview.json")
[[ -n "$projection_digest" && -n "$projection_key" ]]
axiom --json workflow reconcile --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" --preview-digest "$projection_digest" --authorize-external \
  >"$temporary/workflow-projected.json"
assert_canonical "$temporary/workflow-projected.json" success "Execution workflow operation completed"
grep -Fq 'axiom:stage:implementing' "$provider_label"
grep -Fq 'axiom:workflow-projection:' "$provider_comment"
axiom --json workflow reconcile --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" >"$temporary/workflow-projection-replay.json"
assert_canonical "$temporary/workflow-projection-replay.json" success "Execution workflow operation completed"
grep -Fq '"effects":[]' "$temporary/workflow-projection-replay.json"

printf '%s\n' implementation >"$repository/implementation.md"
implementation_digest=$(shasum -a 256 "$repository/implementation.md" | awk '{print $1}')
if axiom --json workflow advance --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" --gate implementation --outcome fail \
  --reference "evidence:implementation.md:$implementation_digest" >"$temporary/workflow-interrupted.json"; then
  exit 1
fi
assert_canonical "$temporary/workflow-interrupted.json" interrupted "Execution remains at the current workflow stage"
revision=$((revision + 1))
observed_axiom workflow resume --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" >"$temporary/workflow-resume.json"
assert_canonical "$temporary/workflow-resume.json" success "Execution workflow operation completed"
revision=$((revision + 1))

printf '%s\n' implementation >"$repository/implementation.md"
implementation_digest=$(shasum -a 256 "$repository/implementation.md" | awk '{print $1}')
advance_configured implementation
revision=$((revision + 1))

axiom --json workflow fact --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" --fact review-started --active --actor maintainer \
  --reference "evidence:implementation.md:$implementation_digest" --authorize-local \
  >"$temporary/workflow-review-started.json"
assert_canonical "$temporary/workflow-review-started.json" success "Execution workflow operation completed"
revision=$((revision + 1))

for gate in review evidence reconciliation completion; do
  printf '%s\n' "$gate" >"$repository/$gate.md"
  reference_digest=$(shasum -a 256 "$repository/$gate.md" | awk '{print $1}')
  advance_configured "$gate"
  revision=$((revision + 1))
done

evidence_digest=$(shasum -a 256 "$repository/evidence.md" | awk '{print $1}')
if axiom --json workflow fact --project dogfood-project --repository main --number 7 \
  --expected-revision "$revision" --fact human-acceptance --active --actor maintainer \
  --reference "evidence:evidence.md:$evidence_digest" --authorize-local \
  >"$temporary/workflow-human-acceptance.json"; then
  exit 1
fi
assert_canonical "$temporary/workflow-human-acceptance.json" denied_authority "Execution authority is stale or incomplete"
python3 - "$temporary/workflow-human-acceptance.json" "$revision" <<'PYJSON'
import json, sys
w = json.load(open(sys.argv[1]))["workflow"]
assert w["revision"] == int(sys.argv[2])
assert w["lifecycleStage"] == "reviewed"
assert w["blockers"] == ["delivery_packet_required"]
assert not w.get("gateAction") and not w.get("gateCommand")
PYJSON

axiom --json workflow evidence --project dogfood-project --repository main --number 7 \
  >"$temporary/workflow-evidence.json"
assert_canonical "$temporary/workflow-evidence.json" success "Execution workflow operation completed"
grep -q '"currentGate":"completion"' "$temporary/workflow-evidence.json"
grep -q '"status":"completed"' "$temporary/workflow-evidence.json"
grep -q '"lifecycleStage":"reviewed"' "$temporary/workflow-evidence.json"

binary_sha=$(shasum -a 256 "$resolved_binary" | awk '{print $1}')
workflow_record=$(find "$state_root/executions/v1" -name '*.json' -type f -print)
workflow_count=$(printf '%s\n' "$workflow_record" | grep -c .)
if [[ "$workflow_count" != 1 ]]; then
  exit 1
fi
workflow_sha=$(shasum -a 256 "$workflow_record" | awk '{print $1}')
printf '{"evidenceVersion":1,"evidence":"axiom_e2e_dogfood","cwdIndependent":true,"globalSkillCount":%s,"workItem":7,"executionId":"%s","runtimeId":"codex","runtimePreviewDigest":"%s","revision":%s,"workflow":"completed","projectionKey":"%s","projectionDigest":"%s","binarySha256":"%s","workflowSha256":"%s","result":"pass"}\n' \
  "$skill_count" "$execution_id" "$runtime_preview" "$revision" "$projection_key" "$projection_digest" "$binary_sha" "$workflow_sha"
