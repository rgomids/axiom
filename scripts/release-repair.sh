#!/usr/bin/env bash
# Read-only validation of explicitly pinned control-code repair for an already
# published immutable recovery release. Original correction provenance stays fixed.
# Caller must gate required CI with trusted code before executing this file;
# the CI check here is defense in depth, not authorization to execute itself.
set -euo pipefail
umask 077
repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
repository= tag= revision= repair_revision=
while (($#)); do
  case "$1" in
    --repo) repository=${2:-}; shift 2 ;;
    --tag) tag=${2:-}; shift 2 ;;
    --revision) revision=${2:-}; shift 2 ;;
    --repair-revision) repair_revision=${2:-}; shift 2 ;;
    *) printf 'release_repair_error: invalid argument\n' >&2; exit 1 ;;
  esac
done
fail() { printf 'release_repair_error: %s\n' "$1" >&2; exit 1; }
[[ "$repository" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ && "$revision" =~ ^[0-9a-f]{40}$ && "$repair_revision" =~ ^[0-9a-f]{40}$ ]] || fail 'full source and repair revisions required'
[[ "${AXIOM_RELEASE_CORRECTIONS_REVISION:-}" =~ ^[0-9a-f]{40}$ && "${AXIOM_RELEASE_CORRECTIONS_DIGEST:-}" =~ ^[0-9a-f]{64}$ ]] || fail 'original correction pins required'
[[ $(git -C "$repository_root" rev-parse HEAD) == "$repair_revision" ]] || fail 'repair scripts do not match selected revision'
grep -Fxq "$repair_revision" < <(git -C "$repository_root" rev-list --first-parent origin/main) || fail 'repair revision is not on first-parent main'
git -C "$repository_root" merge-base --is-ancestor "$AXIOM_RELEASE_CORRECTIONS_REVISION" "$repair_revision" || fail 'repair revision does not descend from correction revision'
[[ "$repair_revision" != "$AXIOM_RELEASE_CORRECTIONS_REVISION" ]] || fail 'repair revision must be newer than correction revision'
temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT
# Validate metadata with the original clean control protocol as well as the
# selected repaired implementation. Neither path may replace the original pins.
grep -Fxq "$AXIOM_RELEASE_CORRECTIONS_REVISION" < <(git -C "$repository_root" rev-list --first-parent origin/main) || fail 'original correction revision is not on first-parent main'
git -C "$repository_root" merge-base --is-ancestor "$revision" "$AXIOM_RELEASE_CORRECTIONS_REVISION" || fail 'original correction revision does not descend from source'
git clone --quiet --no-hardlinks "$repository_root" "$temporary/original"
git -C "$temporary/original" checkout --quiet --detach "$AXIOM_RELEASE_CORRECTIONS_REVISION"
git -C "$temporary/original" fetch --quiet "$repository_root" '+refs/remotes/origin/main:refs/remotes/origin/main'
(
  repository_root=$temporary/original
  source "$repository_root/scripts/release-recovery.sh"
  recovery_validate "$tag" "$revision"
) >/dev/null || fail 'original recovery protocol does not validate pins'
source "$repository_root/scripts/release-recovery.sh"
recovery_validate "$tag" "$revision" >/dev/null || fail 'original recovery pins do not validate'
checks=$(gh api "repos/$repository/commits/$repair_revision/check-runs?per_page=100") || fail 'cannot read repair CI'
# A required check binds a context to the integration the ruleset names; a
# same-named Check Run from any other app is not that check. As in
# publish-release.yml, Check Run id orders attempts even while started_at is
# null, and a policy with no required check authorizes nothing.
required=0
while IFS=$'\t' read -r context integration; do
  [[ "$context" == delivery-metadata ]] && continue
  required=$((required + 1))
  [[ "$integration" =~ ^[0-9]+$ ]] || fail 'required check lacks an integration binding'
  jq -e --arg name "$context" --argjson app "$integration" '[.check_runs[] | select(.name == $name and .app.id == $app)] | sort_by(.id, .started_at) | last | .status == "completed" and .conclusion == "success"' <<<"$checks" >/dev/null || fail 'required CI on repair revision is not successful'
done < <(jq -r '.rules[] | select(.type == "required_status_checks") | .parameters.required_status_checks[] | [.context, (.integration_id // "" | tostring)] | @tsv' "$repository_root/.github/rulesets/main.json")
[[ "$required" != 0 ]] || fail 'required CI on repair revision is not successful'
record=$(gh api "repos/$repository/releases/tags/$tag") || fail 'repair requires an already published recovery release'
jq -e --arg tag "$tag" --arg source "$revision" '.tag_name == $tag and .target_commitish == $source and .draft == false and .prerelease == false and .immutable == true' <<<"$record" >/dev/null || fail 'repair requires an immutable published stable release at the original source'
jq -j '.body // ""' <<<"$record" >"$temporary/notes"
recovery_check_notes "$temporary/notes"
printf 'repair_revision=%s\n' "$repair_revision"
