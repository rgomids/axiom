#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 EXPERIMENT_ROOT EMPTY_TARGET SPEC_KIT_REF" >&2
  exit 64
fi

experiment_root=$1
target=$2
spec_kit_ref=$3
launcher=${SPECIFY_LAUNCHER:-uvx}

if [[ ! -d "$experiment_root/scenario" || ! -f "$experiment_root/scenario/checksums.sha256" ]]; then
  echo "adapter_error=input_missing" >&2
  exit 66
fi

if [[ -z "$target" || "$target" == "/" ]]; then
  echo "adapter_error=unsafe_target" >&2
  exit 64
fi

if ! command -v "$launcher" >/dev/null 2>&1; then
  echo "adapter_error=spec_kit_unavailable launcher=$launcher" >&2
  exit 69
fi

mkdir -p "$target"
if find "$target" -mindepth 1 -maxdepth 1 -print -quit | grep -q .; then
  echo "adapter_error=target_not_empty" >&2
  exit 73
fi

(
  cd "$experiment_root"
  shasum -a 256 -c scenario/checksums.sha256 >/dev/null
)

git -C "$target" init -q
(
  cd "$target"
  "$launcher" --from "git+https://github.com/github/spec-kit.git@$spec_kit_ref" \
    specify init --here --force --integration codex --script sh \
    --ignore-agent-tools
)

feature_dir="$target/specs/002-analyze-converge-reuse"
mkdir -p "$feature_dir" "$target/src" "$target/tests" \
  "$target/.experiment/protocol"

cp "$experiment_root/scenario/specification.md" "$feature_dir/spec.md"
cp "$experiment_root/scenario/plan.md" "$feature_dir/plan.md"
cp "$experiment_root/scenario/tasks.md" "$feature_dir/tasks.md"
cp "$experiment_root/scenario/decisions.md" "$target/.experiment/decisions.md"
cp -R "$experiment_root/scenario/implementation/src/." "$target/src/"
cp -R "$experiment_root/scenario/implementation/tests/." "$target/tests/"
cp "$experiment_root/protocol/capability-contract.md" "$target/.experiment/protocol/"
cp "$experiment_root/protocol/findings.schema.json" "$target/.experiment/protocol/"
cp "$experiment_root/protocol/speckit-encapsulated-prompt.md" "$target/.experiment/protocol/"

jq -n --arg feature_directory "specs/002-analyze-converge-reuse" \
  '{feature_directory:$feature_directory}' > "$target/.specify/feature.json"

jq -n \
  --arg spec_kit_ref "$spec_kit_ref" \
  --arg feature_directory "specs/002-analyze-converge-reuse" \
  '{
    spec_kit_ref:$spec_kit_ref,
    feature_directory:$feature_directory,
    exact_copies:[
      "scenario/specification.md -> specs/002-analyze-converge-reuse/spec.md",
      "scenario/plan.md -> specs/002-analyze-converge-reuse/plan.md",
      "scenario/tasks.md -> specs/002-analyze-converge-reuse/tasks.md",
      "scenario/implementation/src -> src",
      "scenario/implementation/tests -> tests",
      "scenario/decisions.md -> .experiment/decisions.md"
    ],
    translation_loss:[
      "Official analyze/converge does not consume the neutral decision artifact."
    ],
    invented_adapter_state:[
      "Disposable feature directory and feature.json",
      "Generated .specify scaffold and default unfilled constitution"
    ]
  }' > "$target/.experiment/translation-manifest.json"

git -C "$target" add .
git -C "$target" -c user.name='Axiom Experiment' \
  -c user.email='experiment@example.invalid' \
  commit -qm 'freeze disposable adapter workspace'

echo "adapter_status=prepared"
echo "workspace=$target"
echo "feature_dir=$feature_dir"
