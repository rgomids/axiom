#!/usr/bin/env bash
set -euo pipefail

umask 077

archive=
checksums=
binary_root=
receipt_root=
verify_only=false
while (($#)); do
  case "$1" in
    --verify-only) verify_only=true; shift ;;
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
    macos:darwin:arm64|macos-27:darwin:arm64)
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

fail_v2() { printf 'install_error: %s\n' "$1" >&2; exit 1; }

# verify_v2 validates archive generation 2 completely, in shell, before any
# candidate execution. It mirrors internal/install.loadCandidateV2.
verify_v2() {
  local m="$bundle/release-metadata.txt" key value keys expected_keys hash
  expected_keys='formatVersion product version revision sourceState release platform goos architecture skillSetVersion sourceCommit archiveGeneration docsInventorySha256 docsProvenanceSha256 docsManMapSha256 binarySha256'
  [[ $(wc -c <"$m" | tr -d ' ') -le 16384 && $(wc -l <"$m" | tr -d ' ') == 16 && -z $(tail -c1 "$m" | tr -d '\n') ]] || fail_v2 'release metadata schema mismatch'
  LC_ALL=C grep -q $'[\x01-\x08\x0b-\x1f\x7f]' "$m" && fail_v2 'release metadata schema mismatch'
  [[ $(LC_ALL=C grep -c '^[A-Za-z0-9]*=[^=]\{1,\}$' "$m") == 16 ]] || fail_v2 'release metadata schema mismatch'
  keys=$(awk -F= '{printf "%s ", $1}' "$m")
  [[ "$keys" == "$expected_keys " ]] || fail_v2 'release metadata schema mismatch'
  meta() { awk -F= -v k="$1" '$1 == k {print $2}' "$m"; }
  local semver='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'
  version=$(meta version); source_state=$(meta sourceState); release=$(meta release)
  local commit revision
  commit=$(meta sourceCommit); revision=$(meta revision)
  platform_name=$(meta platform); goos_name=$(meta goos); arch_name=$(meta architecture)
  [[ $(meta formatVersion) == 2 && $(meta product) == Axiom && "$version" =~ $semver && ${#version} -le 128 && "$commit" =~ ^[0-9a-f]{40}$ && "$revision" == "${commit:0:12}" ]] || fail_v2 'unsupported release metadata'
  [[ ( "$source_state" == clean || "$source_state" == dirty ) && ( "$release" == false || "$release" == true && "$source_state" == clean ) ]] || fail_v2 'unsupported release metadata'
  [[ $(meta skillSetVersion) == 1 && $(meta archiveGeneration) == 2 ]] || fail_v2 'unsupported release metadata'
  case "$platform_name:$goos_name:$arch_name" in macos:darwin:arm64|linux:linux:amd64|linux:linux:arm64) ;; *) fail_v2 'unsupported release metadata' ;; esac
  for key in docsInventorySha256 docsProvenanceSha256 docsManMapSha256 binarySha256; do
    [[ $(meta "$key") =~ ^[0-9a-f]{64}$ ]] || fail_v2 'unsupported release metadata'
  done
  [[ $(basename "$bundle") == "axiom-$version-$platform_name-$arch_name" ]] || fail_v2 'archive root does not match release metadata'
  [[ $(digest "$bundle/axiom") == "$(meta binarySha256)" && $(digest "$bundle/docs/inventory.tsv") == "$(meta docsInventorySha256)" && $(digest "$bundle/docs/provenance.tsv") == "$(meta docsProvenanceSha256)" && $(digest "$bundle/docs/man-map.tsv") == "$(meta docsManMapSha256)" ]] || fail_v2 'release metadata does not match bundle'

  local control
  for control in inventory provenance man-map; do
    local file="$bundle/docs/$control.tsv"
    [[ $(wc -c <"$file" | tr -d ' ') -le 262144 && -z $(tail -c1 "$file" | tr -d '\n') ]] || fail_v2 'documentary record schema mismatch'
    LC_ALL=C grep -q $'[\x01-\x08\x0b-\x0c\x0e-\x1f\x7f\r]' "$file" && fail_v2 'documentary record schema mismatch'
    [[ $(sed -n 1p "$file") == formatVersion=1 ]] || fail_v2 'unsupported documentary record'
  done
  [[ $(sed -n 2p "$bundle/docs/inventory.tsv") == $'path\tkind\tbytes\tsha256' && $(sed -n 2p "$bundle/docs/provenance.tsv") == $'output\tsource\tsourceCommit\tsourceSha256\ttransform' && $(sed -n 2p "$bundle/docs/man-map.tsv") == $'command\tpage\tanchor' ]] || fail_v2 'unsupported documentary record'
  for control in inventory provenance man-map; do
    tail -n +3 "$bundle/docs/$control.tsv" >"$temporary/$control.rows"
    [[ -s "$temporary/$control.rows" ]] || fail_v2 'documentary record schema mismatch'
    LC_ALL=C sort -c -u "$temporary/$control.rows" 2>/dev/null || fail_v2 'documentary record not strictly sorted'
  done
  [[ $(wc -l <"$temporary/inventory.rows" | tr -d ' ') -le 64 && $(wc -l <"$temporary/provenance.rows" | tr -d ' ') -le 1024 && $(wc -l <"$temporary/man-map.rows" | tr -d ' ') -le 256 ]] || fail_v2 'documentary quota exceeded'

  local path kind size sha extra total=0 expected_files="$temporary/expected-v2"
  : >"$expected_files"
  for key in axiom LICENSE install.sh release-metadata.txt skills-manifest.txt docs/inventory.tsv docs/provenance.tsv docs/man-map.tsv; do printf '%s\n' "$key" >>"$expected_files"; done
  for key in axiom-project axiom-runtime axiom-work-item axiom-workflow; do printf 'skills/%s/SKILL.md\n' "$key" >>"$expected_files"; done
  while IFS=$'\t' read -r path kind size sha extra; do
    [[ -z "${extra:-}" && "$path" =~ ^docs/(text/[a-z0-9][a-z0-9._-]*\.md|man/man1/[a-z0-9][a-z0-9._-]*\.1)$ && ${#path} -le 240 && "$sha" =~ ^[0-9a-f]{64}$ && "$size" =~ ^[1-9][0-9]*$ && "$size" -le 1048576 ]] || fail_v2 'documentary inventory mismatch'
    case "$kind:$path" in markdown:docs/text/*.md|man1:docs/man/man1/*.1) ;; *) fail_v2 'documentary inventory mismatch' ;; esac
    [[ -f "$bundle/$path" && ! -L "$bundle/$path" && $(wc -c <"$bundle/$path" | tr -d ' ') == "$size" && $(digest "$bundle/$path") == "$sha" ]] || fail_v2 'documentary inventory mismatch'
    total=$((total + size))
    printf '%s\n' "$path" >>"$expected_files"
  done <"$temporary/inventory.rows"
  [[ $total -le 8388608 ]] || fail_v2 'documentary quota exceeded'
  for key in index concepts installation project runtime workflow work-item configuration troubleshooting cli notices; do
    grep -q "^docs/text/$key.md	" "$temporary/inventory.rows" || fail_v2 'documentary inventory mismatch'
  done
  LC_ALL=C sort "$expected_files" >"$temporary/expected-v2.sorted"
  find "$bundle" -type f ! -name MANIFEST.sha256 -print | sed "s#^$bundle/##" | LC_ALL=C sort >"$temporary/actual-v2"
  cmp -s "$temporary/expected-v2.sorted" "$temporary/actual-v2" || fail_v2 'bundle contents do not match documentary inventory'

  local output source source_commit source_sha transform previous=
  while IFS=$'\t' read -r output source source_commit source_sha transform extra; do
    [[ -z "${extra:-}" && "$source" =~ ^[a-zA-Z0-9][a-zA-Z0-9._/-]*$ && ${#source} -le 240 && "$source" != *..* && "$source_commit" == "$commit" && "$source_sha" =~ ^[0-9a-f]{64}$ ]] || fail_v2 'documentary provenance mismatch'
    case "$transform" in copy-v1|operational-synthesis-v1|cli-render-v1) ;; *) fail_v2 'documentary provenance mismatch' ;; esac
    cut -f1 "$temporary/inventory.rows" | grep -qxF -- "$output" || fail_v2 'documentary provenance mismatch'
    printf '%s\n' "$output" >>"$temporary/provenance.outputs"
  done <"$temporary/provenance.rows"
  cut -f1 "$temporary/inventory.rows" | LC_ALL=C sort >"$temporary/inventory.outputs"
  LC_ALL=C sort -u "$temporary/provenance.outputs" | cmp -s - "$temporary/inventory.outputs" || fail_v2 'documentary provenance mismatch'

  local command page anchor have_root=false
  while IFS=$'\t' read -r command page anchor extra; do
    [[ -z "${extra:-}" && "$command" =~ ^axiom(\ [a-z0-9][a-z0-9-]*)*$ && ${#command} -le 256 && "$page" == docs/man/man1/* && "$anchor" =~ ^[A-Z][A-Z0-9_-]{0,63}$ ]] || fail_v2 'documentary man map mismatch'
    cut -f1 "$temporary/inventory.rows" | grep -qxF -- "$page" && grep -qx "\.SH $anchor" "$bundle/$page" || fail_v2 'documentary man map mismatch'
    [[ "$command" == axiom ]] && have_root=true
  done <"$temporary/man-map.rows"
  cut -f1 "$temporary/man-map.rows" | LC_ALL=C sort -c -u 2>/dev/null || fail_v2 'documentary man map mismatch'
  [[ "$have_root" == true ]] || fail_v2 'documentary man map mismatch'

  # Strict manifest: exactly every other regular file, sorted, two-space syntax.
  find "$bundle" -type f ! -name MANIFEST.sha256 -print | sed "s#^$bundle/##" | LC_ALL=C sort | while IFS= read -r path; do
    printf '%s  %s\n' "$(digest "$bundle/$path")" "$path"
  done >"$temporary/manifest-v2"
  cmp -s "$temporary/manifest-v2" "$bundle/MANIFEST.sha256" || fail_v2 'bundle manifest mismatch'
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
archive_format=1
metadata_member=$(tar -tzf "$staged_archive" | grep -E '^axiom-[^/]+/release-metadata\.txt$' | head -n1 || true)
if [[ -n "$metadata_member" ]] && tar -xzOf "$staged_archive" "$metadata_member" 2>/dev/null | head -n1 | grep -qx 'formatVersion=2'; then archive_format=2; fi
if [[ "$archive_format" == 2 ]]; then
  [[ $(tar -tzf "$staged_archive" | wc -l | tr -d ' ') -le 192 ]] || { printf 'install_error: archive exceeds entry limit\n' >&2; exit 1; }
  while IFS= read -r entry; do
    entry=${entry%/}
    [[ "$entry" =~ ^axiom-[^/]+(/[A-Za-z0-9][A-Za-z0-9._-]*)*$ && ${#entry} -le 280 ]] || { printf 'install_error: unsafe archive path\n' >&2; exit 1; }
  done < <(tar -tzf "$staged_archive")
  tar -tvzf "$staged_archive" | awk '{ name = $NF; sub(/\/$/, "", name); n = split(name, parts, "/"); leaf = parts[n]; want = "-rw-------"; if (substr($1, 1, 1) == "d") want = "drwx------"; else if (n == 2 && (leaf == "axiom" || leaf == "install.sh")) want = "-rwx------"; if ($1 != want) exit 1 }' || { printf 'install_error: archive modes refused\n' >&2; exit 1; }
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

if [[ "$archive_format" == 2 ]]; then
  verify_v2
  platform=$goos_name
  architecture=$arch_name
  release_platform=$platform_name
else
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
  
fi

skills_manifest="$bundle/skills-manifest.txt"
[[ $(wc -l <"$skills_manifest" | tr -d ' ') == 7 ]] || { printf 'install_error: skill manifest schema mismatch\n' >&2; exit 1; }
for field in formatVersion skillSetVersion binaryCompatibility; do
  [[ $(grep -c "^${field}=" "$skills_manifest") == 1 ]] || { printf 'install_error: skill manifest schema mismatch\n' >&2; exit 1; }
done
[[ $(awk -F= '$1 == "formatVersion" {print $2}' "$skills_manifest") == 1 && $(awk -F= '$1 == "skillSetVersion" {print $2}' "$skills_manifest") == 1 && $(awk -F= '$1 == "binaryCompatibility" {print $2}' "$skills_manifest") == 1 ]] || { printf 'install_error: unsupported skill manifest\n' >&2; exit 1; }
[[ $(grep -c '^skill\.[a-z0-9-]*=[0-9a-f]\{64\}$' "$skills_manifest") == 4 ]] || { printf 'install_error: skill manifest schema mismatch\n' >&2; exit 1; }
for name in axiom-project axiom-work-item axiom-workflow axiom-runtime; do
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

if [[ "$archive_format" != 2 ]]; then
  platform=$(awk -F= '$1 == "goos" {print $2}' "$metadata")
  architecture=$(awk -F= '$1 == "architecture" {print $2}' "$metadata")
  release_platform=$(awk -F= '$1 == "platform" {print $2}' "$metadata")
fi
# Release-set verification validates all rows without executing candidates or
# publishing installation state. Destination arguments remain mandatory.
if [[ "$verify_only" == true ]]; then
  printf 'archive_verify_success: format=%s platform=%s architecture=%s\n' "$archive_format" "$release_platform" "$architecture"
  exit 0
fi
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
