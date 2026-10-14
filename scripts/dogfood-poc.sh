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
if [[ "$skill_count" != 4 ]]; then
  exit 1
fi
axiom help >"$temporary/help.txt"
for skill in axiom-project axiom-work-item axiom-workflow axiom-runtime; do
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
# It is authored as canonical v3 through the public reviewed lifecycle
# (Specification 007 PM-1/PM-11): fresh state is initialized directly, nothing
# is migrated and no local configuration file is written by hand. Binding
# creation only looks the executables up on an isolated PATH holding
# never-executed stubs; the host's real Codex or Claude is never visible.
authoring_bin="$temporary/authoring-bin"
mkdir -p "$authoring_bin"
for stub in codex claude; do
  printf '#!/bin/sh\nexit 99\n' >"$authoring_bin/$stub"
  chmod 700 "$authoring_bin/$stub"
done
authoring_axiom() {
  env -u CLAUDE_CONFIG_DIR HOME="$runtime_home" PATH="$authoring_bin:$binary_root" axiom --json "$@"
}
author_runtime() {
  local label=$1
  shift
  authoring_axiom "$@" >"$temporary/runtime-$label-preview.json"
  grep -q '"category":"preview"' "$temporary/runtime-$label-preview.json"
  read -r local_token local_preview < <(python3 -c 'import json,sys; w=json.load(open(sys.argv[1]))["runtimeAuthoring"]; print(w["expectedRevision"], w["previewDigest"])' "$temporary/runtime-$label-preview.json")
  authoring_axiom "$@" --expected-revision "$local_token" --preview-digest "$local_preview" --authorize-local >"$temporary/runtime-$label-applied.json"
  grep -q '"status":"success"' "$temporary/runtime-$label-applied.json"
  grep -q '"category":"applied"' "$temporary/runtime-$label-applied.json"
}
axiom --json runtime profile validate >"$temporary/runtime-profile-missing.json" && exit 1
author_runtime claude runtime binding create --runtime claude --profile careful --model approved-claude-model \
  --effort-mode runtime-default --effort default --capabilities axiom-skills --complexities high
author_runtime codex runtime binding create --runtime codex --profile worker --model approved-codex-model \
  --effort-mode runtime-default --effort default --capabilities axiom-skills --complexities high
python3 - "$state_root/runtime-profiles/v1/configuration.json" <<'PYJSON'
import json,sys
c=json.load(open(sys.argv[1]))
assert c["formatVersion"] == 3 and c["revision"] == 2
assert [r["id"] for r in c["runtimes"]] == ["claude", "codex"]
assert all("effortDeclarations" not in r for r in c["runtimes"])
assert all(p["effort"] == {"mode":"runtime-default","value":"default"} for p in c["modelProfiles"])
PYJSON
axiom --json runtime profile validate >"$temporary/runtime-profile-validate.json"
assert_canonical "$temporary/runtime-profile-validate.json" success "Runtime profile configuration is valid"
# The removed migration surface is not callable.
if axiom --json runtime profile migrate --profile-effort worker:runtime-default:default >"$temporary/runtime-migrate.json"; then
  exit 1
fi
grep -q '"category":"invalid_command"' "$temporary/runtime-migrate.json"

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

# Two configured, allowed Runtimes are never resolved by machine availability.
assert_runtime_blocked implicit ambiguous_match \
  workflow start --project dogfood-project --repository main --number 7 "${policy_inputs[@]}"

# The observed codex stub and its verified skill integration are not proof:
# the production composition injects no Profile proof and the embedded
# version registry has no production entry, so even the explicitly selected,
# structurally valid v3 Profile fails closed before any Execution exists.
assert_runtime_blocked historical-proof effort_unproven "${start_args[@]}"

# The new builtin selection has its own exact confirmation. Missing G-3 proof
# still blocks use after explicit selection.
"$temporary/dogfoodfixture" definition-current >"$temporary/definition-current.json"
read -r current_id current_revision current_digest < <(python3 -c 'import json,sys; w=json.load(open(sys.argv[1])); print(w["workflowId"],w["revision"],w["digest"])' "$temporary/definition-current.json")
current_selection=(project workflow select --project dogfood-project --workflow "$current_id" --revision "$current_revision" --digest "$current_digest" --source builtin)
run_success previewed "$temporary/current-selection-preview.json" "${current_selection[@]}"
read -r project_revision selection_preview < <(python3 -c 'import json,sys; w=json.load(open(sys.argv[1]))["workflowAuthoring"]; print(w["projectRevision"],w["previewDigest"])' "$temporary/current-selection-preview.json")
run_success applied "$temporary/current-selected.json" "${current_selection[@]}" --expected-revision "$project_revision" --preview-digest "$selection_preview" --authorize-local
assert_runtime_blocked current-proof effort_unproven "${start_args[@]}"
run_success inspected "$temporary/current-profiles.json" runtime profile list
python3 - "$temporary/current-profiles.json" <<'PYJSON'
import json,sys
r=json.load(open(sys.argv[1]))["runtimeAuthoring"]
assert r["state"] == "active-v3" and r["revision"] == 2
assert r["authentication"] == "unknown" and r["modelSupport"] == "unproven" and r["effortSupport"] == "unproven"
assert all(p["effort"] == {"mode":"runtime-default","value":"default"} for p in r["profiles"])
PYJSON

# Positive current lifecycle and historical inspection are synthetic integration
# receipts with injected evidence, separate from this real CLI refusal journal.
# Synthetic canonical-v3 identity/applicability receipts use only the
# test-only registry decoder; the embedded production registry stays empty.
(cd "$repository_root" && env -u OPENAI_API_KEY -u CODEX_API_KEY -u CLAUDE_CODE_EFFORT_LEVEL go test ./cmd/lingo -run 'TestExecutableMinimalLifecycleAndFailurePaths|TestExecutableDeclarationAuthoringWithoutMigration|TestCurrentWorkflowStartFreezesReviewedV2ProfileWithoutModelRun|TestWorkflowStagePlanPublicJourneySingleAndMixedRuntimeGraph' -count=1)
(cd "$repository_root" && env -u CLAUDE_CODE_EFFORT_LEVEL go test ./internal/runtimeadapter -run 'TestDogfoodV3Fixture' -count=1)
printf '%s\n' 'PASS: bounded production canonical-v3 authoring/refusal smoke; current positive lifecycle and v3 identity SYNTHETIC; vendor model/auth, production Runtime identity and integrated E2E UNVERIFIED'
