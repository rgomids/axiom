#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 5 ]]; then
  echo "usage: $0 WORKSPACE EVENTS RESULT TIME_FILE MODEL" >&2
  exit 64
fi

workspace=$1
events=$2
result=$3
time_file=$4
model=$5
codex_executable=${CODEX_EXECUTABLE:-/Applications/ChatGPT.app/Contents/Resources/codex}

if [[ ! -x "$codex_executable" ]]; then
  echo "adapter_error=capability_unavailable executable=$codex_executable" >&2
  exit 69
fi

mkdir -p "$(dirname "$events")" "$(dirname "$result")" "$(dirname "$time_file")"

if ! /usr/bin/time -p -o "$time_file" \
  "$codex_executable" exec -C "$workspace" --ignore-user-config \
    --sandbox workspace-write --ephemeral --json -m "$model" \
    -c 'model_reasoning_effort="high"' \
    --output-schema "$workspace/.experiment/protocol/findings.schema.json" \
    -o "$result" - \
    < "$workspace/.experiment/protocol/speckit-encapsulated-prompt.md" \
    > "$events"; then
  echo "adapter_error=capability_failure" >&2
  exit 70
fi

echo "adapter_status=capability_complete"
