#!/usr/bin/env bash
# Internal recovery protocol. Pins are explicit release.sh/workflow inputs,
# inherited by notes, delivery and publication; never a floating-ref fallback.
recovery_validate() {
  local recovery_tag=$1 recovery_source=$2 recovery_output=${3:-}
  [[ "$recovery_tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || fail 'correction recovery is stable-only'
  local recovery_version
  recovery_version=$(git -C "$repository_root" show "$recovery_source:.release-please-manifest.json" | jq -er '.["."]') || return 1
  [[ "$recovery_tag" == "v$recovery_version" ]] || fail 'correction recovery tag differs from source manifest'
  local recovery_args=(--source-revision "$recovery_source"
    --corrections-revision "${AXIOM_RELEASE_CORRECTIONS_REVISION:-}"
    --corrections-digest "${AXIOM_RELEASE_CORRECTIONS_DIGEST:-}")
  [[ -z "$recovery_output" ]] || recovery_args+=(--file-output "$recovery_output")
  "$repository_root/scripts/release-corrections.sh" "${recovery_args[@]}" || return 1
}
recovery_note_lines() {
  printf -- '- Recovery corrections revision: `%s`\n' "$AXIOM_RELEASE_CORRECTIONS_REVISION"
  printf -- '- Recovery corrections SHA-256: `%s`\n' "$AXIOM_RELEASE_CORRECTIONS_DIGEST"
}
recovery_check_notes() {
  local recovery_notes=$1
  [[ $(grep -c '^- Recovery corrections revision:' "$recovery_notes") == 1 \
    && $(grep -c '^- Recovery corrections SHA-256:' "$recovery_notes") == 1 ]] \
    || fail 'release notes lack unique recovery provenance'
  grep -Fxq -- "$(recovery_note_lines | head -n 1)" "$recovery_notes" \
    && grep -Fxq -- "$(recovery_note_lines | tail -n 1)" "$recovery_notes" \
    || fail 'release notes recovery pins differ'
}
