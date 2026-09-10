#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 VALIDATOR RAW_RESULT NORMALIZED_RESULT" >&2
  exit 64
fi

validator=$1
raw_result=$2
normalized_result=$3

if [[ -e "$normalized_result" ]]; then
  echo "adapter_error=output_exists" >&2
  exit 73
fi

if ! "$validator" "$raw_result" >/dev/null; then
  echo "adapter_error=incompatible_output" >&2
  exit 65
fi

output_dir=$(dirname "$normalized_result")
mkdir -p "$output_dir"
temporary_output=$(mktemp "$output_dir/.normalize.XXXXXX")
trap 'rm -f "$temporary_output"' EXIT

jq '.approach = "B-speckit-encapsulated" | .findings |= sort_by(.id)' \
  "$raw_result" > "$temporary_output"
mv "$temporary_output" "$normalized_result"
trap - EXIT

echo "adapter_status=normalized"
