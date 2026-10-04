#!/usr/bin/env bash
set -euo pipefail

umask 077

archive=
checksums=
binary_root=
receipt_root=
while (($#)); do
  case "$1" in
    --archive) archive=${2:-}; shift 2 ;;
    --checksums) checksums=${2:-}; shift 2 ;;
    --bin-dir) binary_root=${2:-}; shift 2 ;;
    --receipt-dir) receipt_root=${2:-}; shift 2 ;;
    *) printf 'install_error: invalid argument\n' >&2; exit 1 ;;
  esac
done

case "$archive:$checksums:$binary_root:$receipt_root" in
  /*:/*:/*:/*) ;;
  *) printf 'install_error: absolute archive and destinations required\n' >&2; exit 1 ;;
esac
for value in "$archive" "$checksums" "$binary_root" "$receipt_root"; do
  case "$value" in *$'\n'*|*$'\r'*|*'='*) printf 'install_error: unsafe path\n' >&2; exit 1 ;; esac
done

digest() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

host_matches_release_row() {
  local release_platform=$1 release_goos=$2 release_architecture=$3
  local host_system host_architecture
  host_system=$(uname -s) || return 1
  host_architecture=$(uname -m) || return 1
  case "$release_platform:$release_goos:$release_architecture" in
    macos-27:darwin:arm64)
      # Any macOS version runs the darwin/arm64 build.
      [[ "$host_system:$host_architecture" == Darwin:arm64 ]]
      ;;
    linux:linux:amd64)
      [[ "$host_system:$host_architecture" == Linux:x86_64 ]]
      ;;
    linux:linux:arm64)
      [[ "$host_system:$host_architecture" == Linux:aarch64 ]]
      ;;
    *) return 1 ;;
  esac
}

archive_name=$(basename "$archive")
expected=$(awk -v name="$archive_name" '$2 == name {print $1}' "$checksums")
if [[ ! "$expected" =~ ^[0-9a-f]{64}$ || $(printf '%s\n' "$expected" | wc -l | tr -d ' ') != 1 ]]; then
  printf 'install_error: exact archive checksum unavailable\n' >&2
  exit 1
fi
actual=$(digest "$archive")
if [[ "$actual" != "$expected" ]]; then
  printf 'install_error: archive checksum mismatch\n' >&2
  exit 1
fi

file_owner() {
  if [[ $(uname -s) == Darwin ]]; then stat -f %u "$1"; else stat -c %u "$1"; fi
}

file_mode() {
  if [[ $(uname -s) == Darwin ]]; then stat -f %Lp "$1"; else stat -c %a "$1"; fi
}

private_acl() {
  local path=$1 mode
  case "$(uname -s)" in
    Darwin)
      [[ $(LC_ALL=C ls -lde "$path" | wc -l | tr -d ' ') == 1 ]]
      ;;
    Linux)
      mode=$(LC_ALL=C ls -ld "$path" | awk '{print $1}') || return 1
      [[ "$mode" =~ ^[-d][rwx-]{9}$ ]]
      ;;
    *) return 1 ;;
  esac
}

private_directory() {
  local directory=$1
  [[ -d "$directory" && ! -L "$directory" && $(file_owner "$directory") == "$(id -u)" && $(file_mode "$directory") == 700 ]] || return 1
  private_acl "$directory"
}

safe_components() {
  local current=$1
  while [[ "$current" != / ]]; do
    [[ ! -L "$current" || $(file_owner "$current") == 0 ]] || return 1
    current=$(dirname "$current")
  done
}

# has_sticky_bit reports whether $1 carries the sticky bit (mode &01000).
# file_mode/stat's %Lp on macOS omits it, so it is checked separately.
has_sticky_bit() {
  [[ -n "$(find "$1" -maxdepth 0 -perm -1000 2>/dev/null)" ]]
}

# Temporary workspace containers must prevent replacement by other principals.
# Root/current-user ownership and sticky-bit semantics match ADR-0005.
ancestor_safe() {
  local directory=$1 owner mode
  owner=$(file_owner "$directory") || return 1
  [[ "$owner" == 0 || "$owner" == "$(id -u)" ]] || return 1
  mode=$(file_mode "$directory") || return 1
  [[ "$mode" =~ ^[0-7]{3,4}$ ]] || return 1
  if (( (8#$mode & 8#022) != 0 )); then
    has_sticky_bit "$directory" || return 1
  fi
  return 0
}

# Inspect existing containers up to the filesystem root before using storage.
ancestors_safe() {
  local current
  current=$(dirname "$1")
  while true; do
    # A missing ancestor will be created by mkdir -p (owned by the current
    # user); only an existing ancestor's real ownership/mode is evaluated.
    if [[ -e "$current" ]]; then
      ancestor_safe "$current" || return 1
    fi
    [[ "$current" == / ]] && break
    current=$(dirname "$current")
  done
}

temporary=$(mktemp -d)
cleanup() { rm -rf -- "$temporary"; }
trap cleanup EXIT
safe_components "$temporary" || { printf 'install_error: unsafe temporary workspace path\n' >&2; exit 1; }
temporary=$(cd "$temporary" && pwd -P)
private_directory "$temporary" && ancestors_safe "$temporary" || {
  printf 'install_error: unsafe temporary workspace ownership, permissions, ACL, or type\n' >&2
  exit 1
}
staged_archive="$temporary/release.tar.gz"
cp -- "$archive" "$staged_archive"
chmod 600 "$staged_archive"
[[ $(digest "$staged_archive") == "$actual" ]] || { printf 'install_error: archive changed during staging\n' >&2; exit 1; }
# Bind delegation to these verified bytes, not to mutable caller inputs.
staged_checksums="$temporary/SHA256SUMS"
printf '%s  release.tar.gz\n' "$actual" >"$staged_checksums"

while IFS= read -r entry; do
  case "$entry" in /*|../*|*/../*|*/..|*\\*) printf 'install_error: unsafe archive path\n' >&2; exit 1 ;; esac
done < <(tar -tzf "$staged_archive")
if tar -tvzf "$staged_archive" | awk 'substr($1,1,1) != "-" && substr($1,1,1) != "d" {exit 1}'; then :; else
  printf 'install_error: archive links or special files refused\n' >&2
  exit 1
fi
tar -xzf "$staged_archive" -C "$temporary"
bundle=$(find "$temporary" -mindepth 1 -maxdepth 1 -type d)
if [[ -z "$bundle" || $(printf '%s\n' "$bundle" | wc -l | tr -d ' ') != 1 ]]; then
  printf 'install_error: invalid archive root\n' >&2
  exit 1
fi
(
  cd "$bundle"
  while read -r hash name extra; do
    [[ -z "${extra:-}" && "$hash" =~ ^[0-9a-f]{64}$ && -f "$name" && ! -L "$name" ]] || exit 1
    [[ $(digest "$name") == "$hash" ]] || exit 1
  done <MANIFEST.sha256
  find . -type f ! -path './MANIFEST.sha256' -print | sed 's#^./##' | LC_ALL=C sort >"$temporary/actual-files"
  awk '{print $2}' MANIFEST.sha256 | LC_ALL=C sort >"$temporary/manifest-files"
  cmp -s "$temporary/actual-files" "$temporary/manifest-files"
) || { printf 'install_error: bundle manifest mismatch\n' >&2; exit 1; }

[[ -f "$bundle/axiom" && ! -L "$bundle/axiom" ]] || { printf 'install_error: bundle axiom executable missing\n' >&2; exit 1; }

metadata="$bundle/release-metadata.txt"
metadata_fields='formatVersion product version revision sourceState release platform goos architecture skillSetVersion'
[[ $(wc -l <"$metadata" | tr -d ' ') == 10 ]] || { printf 'install_error: release metadata schema mismatch\n' >&2; exit 1; }
for field in $metadata_fields; do
  [[ $(grep -c "^${field}=" "$metadata") == 1 ]] || { printf 'install_error: release metadata schema mismatch\n' >&2; exit 1; }
done
awk -F= '$1 != "formatVersion" && $1 != "product" && $1 != "version" && $1 != "revision" && $1 != "sourceState" && $1 != "release" && $1 != "platform" && $1 != "goos" && $1 != "architecture" && $1 != "skillSetVersion" {exit 1}' "$metadata" || { printf 'install_error: release metadata schema mismatch\n' >&2; exit 1; }
[[ $(awk -F= '$1 == "formatVersion" {print $2}' "$metadata") == 1 ]] || { printf 'install_error: unsupported release metadata\n' >&2; exit 1; }
[[ $(awk -F= '$1 == "product" {print $2}' "$metadata") == Axiom ]] || { printf 'install_error: unsupported release metadata\n' >&2; exit 1; }
version=$(awk -F= '$1 == "version" {print $2}' "$metadata")
[[ "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$ ]] || { printf 'install_error: unsupported release metadata\n' >&2; exit 1; }
[[ $(awk -F= '$1 == "revision" {print $2}' "$metadata") =~ ^[0-9a-f]{12}$ ]] || { printf 'install_error: unsupported release metadata\n' >&2; exit 1; }
release=$(awk -F= '$1 == "release" {print $2}' "$metadata")
source_state=$(awk -F= '$1 == "sourceState" {print $2}' "$metadata")
[[ "$release" == false || "$release" == true && "$source_state" == clean ]] || { printf 'install_error: unsupported release metadata\n' >&2; exit 1; }

skills_manifest="$bundle/skills-manifest.txt"
[[ $(wc -l <"$skills_manifest" | tr -d ' ') == 9 ]] || { printf 'install_error: skill manifest schema mismatch\n' >&2; exit 1; }
for field in formatVersion skillSetVersion binaryCompatibility; do
  [[ $(grep -c "^${field}=" "$skills_manifest") == 1 ]] || { printf 'install_error: skill manifest schema mismatch\n' >&2; exit 1; }
done
[[ $(awk -F= '$1 == "formatVersion" {print $2}' "$skills_manifest") == 1 && $(awk -F= '$1 == "skillSetVersion" {print $2}' "$skills_manifest") == 1 && $(awk -F= '$1 == "binaryCompatibility" {print $2}' "$skills_manifest") == 1 ]] || { printf 'install_error: unsupported skill manifest\n' >&2; exit 1; }
[[ $(grep -c '^skill\.[a-z0-9-]*=[0-9a-f]\{64\}$' "$skills_manifest") == 6 ]] || { printf 'install_error: skill manifest schema mismatch\n' >&2; exit 1; }
for name in axiom-project-configure axiom-project-list axiom-project-show axiom-work-item-create axiom-work-item-run axiom-work-item-status; do
  [[ $(grep -c "^skill\.${name}=" "$skills_manifest") == 1 ]] || { printf 'install_error: skill manifest schema mismatch\n' >&2; exit 1; }
done
while IFS='=' read -r key hash; do
  case "$key" in
    skill.*)
      name=${key#skill.}
      [[ -f "$bundle/skills/$name/SKILL.md" && ! -L "$bundle/skills/$name/SKILL.md" && $(digest "$bundle/skills/$name/SKILL.md") == "$hash" ]] || { printf 'install_error: skill manifest mismatch\n' >&2; exit 1; }
      ;;
  esac
done <"$skills_manifest"

platform=$(awk -F= '$1 == "goos" {print $2}' "$metadata")
architecture=$(awk -F= '$1 == "architecture" {print $2}' "$metadata")
release_platform=$(awk -F= '$1 == "platform" {print $2}' "$metadata")
if ! host_matches_release_row "$release_platform" "$platform" "$architecture"; then
  printf 'install_error: unsupported host for release row %s/%s; supported OS family and architecture required\n' "$release_platform" "$architecture" >&2
  exit 1
fi

# Execute only the fully verified candidate, before any persistent effect.
chmod 700 "$bundle/axiom"
if ! (cd / && "$bundle/axiom" version) >/dev/null 2>&1; then
  printf 'install_error: temporary workspace does not permit execution; choose an executable TMPDIR\n' >&2
  exit 1
fi
set +e
(cd / && "$bundle/axiom" install-release --archive "$staged_archive" --checksums "$staged_checksums" --bin-dir "$binary_root" --receipt-dir "$receipt_root")
status=$?
set -e
exit "$status"
