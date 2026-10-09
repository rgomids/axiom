#!/usr/bin/env bash
# Offline bootstrap fixtures only: shell candidates exercise facade delegation,
# not the Go installer, native binary execution, or a real noexec mount.
set -euo pipefail
umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
temporary=$(cd "$temporary" && pwd -P)
trap 'rm -rf -- "$temporary"' EXIT
facade="$repository_root/scripts/install-release.sh"
tools="$temporary/tools"
mkdir "$tools"
failures=0

digest() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

# Select the fixture row using the actual OS, so stat retains its native ABI.
case "$(uname -s):$(uname -m)" in
  Darwin:arm64) platform=macos-27; goos=darwin; architecture=arm64; system=Darwin; machine=arm64 ;;
  Linux:x86_64) platform=linux; goos=linux; architecture=amd64; system=Linux; machine=x86_64 ;;
  Linux:aarch64) platform=linux; goos=linux; architecture=arm64; system=Linux; machine=aarch64 ;;
  *) printf 'result=blocked reason=unsupported-fixture-host\n'; exit 78 ;;
esac
cat >"$tools/uname" <<EOF
#!/bin/sh
case "\$1" in -s) printf '%s\n' '$system' ;; -m) printf '%s\n' '$machine' ;; *) exit 1 ;; esac
EOF
# Issue #183: the facade never consults the macOS version. This sw_vers shim
# reports a fixture version and records any invocation.
write_sw_vers() {
  printf '#!/bin/sh\n: >"%s/sw_vers.invoked"\nprintf "%%s\\n" %s\n' "$temporary" "$1" >"$tools/sw_vers"
  chmod 700 "$tools/sw_vers"
}
write_sw_vers 27.0
chmod 700 "$tools/uname"
real_mktemp=$(command -v mktemp)
cat >"$tools/mktemp" <<'EOF'
#!/bin/sh
# Darwin mktemp can replace unsafe TMPDIR with its safe system fallback.
# Force a real directory beneath the specified path only for policy fixtures.
if [ "${FIXTURE_FORCE_TMPDIR:-0}" = 1 ]; then
  exec "$FIXTURE_REAL_MKTEMP" -d "$TMPDIR/work.XXXXXX"
fi
exec "$FIXTURE_REAL_MKTEMP" "$@"
EOF
chmod 700 "$tools/mktemp"

bundle_name="axiom-1.0.0-$platform-$architecture"
bundle="$temporary/source/$bundle_name"
mkdir -p "$bundle/skills"
cat >"$bundle/axiom" <<'EOF'
#!/bin/sh
printf '%s\n' "$0" >"$FIXTURE_LEDGER.candidate"
pwd -P >"$FIXTURE_LEDGER.cwd"
case "$1" in
  version)
    printf 'version\n' >>"$FIXTURE_LEDGER"
    workspace=$(dirname "$(dirname "$0")")
    case "$(uname -s)" in
      Darwin) stat -f %Lp "$workspace" ;;
      Linux) stat -c %a "$workspace" ;;
    esac >"$FIXTURE_LEDGER.mode"
    if [ "$FIXTURE_REPLACE_INPUTS" = 1 ]; then
      printf 'replaced after verified candidate preflight\n' >"$FIXTURE_ORIGINAL_ARCHIVE"
      printf 'replaced original checksums\n' >"$FIXTURE_ORIGINAL_CHECKSUMS"
    fi
    exit "$FIXTURE_VERSION_STATUS"
    ;;
  install-release)
    printf '%s\n' "$@" >>"$FIXTURE_LEDGER"
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum "$3" | awk '{print $1}' >"$FIXTURE_LEDGER.archive-digest"
    else
      shasum -a 256 "$3" | awk '{print $1}' >"$FIXTURE_LEDGER.archive-digest"
    fi
    cat "$5" >"$FIXTURE_LEDGER.checksums"
    case "$(uname -s)" in
      Darwin) stat -f %Lp "$3" "$5" ;;
      Linux) stat -c %a "$3" "$5" ;;
    esac >"$FIXTURE_LEDGER.input-modes"
    exit "$FIXTURE_INSTALL_STATUS"
    ;;
  *) exit 2 ;;
esac
EOF
chmod 700 "$bundle/axiom"
cat >"$bundle/release-metadata.txt" <<EOF
formatVersion=1
product=Axiom
version=1.0.0
revision=0123456789ab
sourceState=clean
release=true
platform=$platform
goos=$goos
architecture=$architecture
skillSetVersion=1
EOF
printf 'formatVersion=1\nskillSetVersion=1\nbinaryCompatibility=1\n' >"$bundle/skills-manifest.txt"
for name in axiom-project axiom-work-item; do
  mkdir "$bundle/skills/$name"
  printf '# Offline %s fixture\n' "$name" >"$bundle/skills/$name/SKILL.md"
  printf 'skill.%s=%s\n' "$name" "$(digest "$bundle/skills/$name/SKILL.md")" >>"$bundle/skills-manifest.txt"
done
(
  cd "$bundle"
  find . -type f -print | sed 's#^./##' | LC_ALL=C sort | while IFS= read -r file; do
    printf '%s  %s\n' "$(digest "$file")" "$file"
  done
) >"$temporary/manifest"
mv "$temporary/manifest" "$bundle/MANIFEST.sha256"
archive="$temporary/$bundle_name.tar.gz"
tar -czf "$archive" -C "$temporary/source" "$bundle_name"
checksums="$temporary/SHA256SUMS"
printf '%s  %s.tar.gz\n' "$(digest "$archive")" "$bundle_name" >"$checksums"

run_facade() {
  local case_root=$1 temp_root=$2 version_status=${3:-0} install_status=${4:-0} checksum_file=${5:-$checksums}
  local candidate_archive=${7:-$archive}
  local temp_environment=()
  [[ "$temp_root" == unset ]] || temp_environment=("TMPDIR=$temp_root")
  mkdir -p "$case_root/home"
  : >"$case_root/ledger"
  digest "$candidate_archive" >"$case_root/original-digest"
  set +e
  env -i HOME="$case_root/home" PATH="$tools:/usr/bin:/bin" ${temp_environment[@]+"${temp_environment[@]}"} \
    FIXTURE_REAL_MKTEMP="$real_mktemp" FIXTURE_FORCE_TMPDIR="${6:-0}" \
    FIXTURE_LEDGER="$case_root/ledger" FIXTURE_VERSION_STATUS="$version_status" FIXTURE_INSTALL_STATUS="$install_status" \
    FIXTURE_REPLACE_INPUTS="${8:-0}" FIXTURE_ORIGINAL_ARCHIVE="$candidate_archive" FIXTURE_ORIGINAL_CHECKSUMS="$checksum_file" \
    bash "$facade" --archive "$candidate_archive" --checksums "$checksum_file" \
      --bin-dir "$case_root/bin with spaces" --receipt-dir "$case_root/state with spaces" \
      >"$case_root/stdout" 2>"$case_root/stderr"
  result=$?
  set -e
}

no_persistent_effects() {
  [[ ! -e "$1/bin with spaces" && ! -e "$1/state with spaces" ]]
}

delegation_matches() {
  local case_root=$1 workspace expected_digest
  workspace=$(dirname "$(dirname "$(cat "$case_root/ledger.candidate")")")
  expected_digest=$(cat "$case_root/original-digest")
  printf '%s\n' version install-release --archive "$workspace/release.tar.gz" --checksums "$workspace/SHA256SUMS" \
    --bin-dir "$case_root/bin with spaces" --receipt-dir "$case_root/state with spaces" >"$case_root/expected"
  cmp "$case_root/expected" "$case_root/ledger"
  [[ $(cat "$case_root/ledger.mode") == 700 && $(cat "$case_root/ledger.cwd") == / ]]
  [[ $(cat "$case_root/ledger.archive-digest") == "$expected_digest" ]]
  printf '%s  release.tar.gz\n' "$expected_digest" >"$case_root/expected-checksums"
  cmp "$case_root/expected-checksums" "$case_root/ledger.checksums"
  [[ $(cat "$case_root/ledger.input-modes") == $'600\n600' ]]
  [[ ! -e $(cat "$case_root/ledger.candidate") ]]
  [[ ! -e "$workspace" ]]
  no_persistent_effects "$case_root"
}

original_inputs_replaced() {
  local case_root="$temporary/replaced-inputs" replacement_archive
  mkdir -p "$case_root/tmp"
  replacement_archive="$case_root/$bundle_name.tar.gz"
  cp "$archive" "$replacement_archive"
  cp "$checksums" "$case_root/SHA256SUMS"
  run_facade "$case_root" "$case_root/tmp" 0 0 "$case_root/SHA256SUMS" 0 "$replacement_archive" 1
  [[ "$result" == 0 ]]
  delegation_matches "$case_root"
  [[ $(cat "$replacement_archive") == 'replaced after verified candidate preflight' ]]
  [[ $(cat "$case_root/SHA256SUMS") == 'replaced original checksums' ]]
  [[ -z $(find "$case_root/tmp" -mindepth 1 -print -quit) ]]
}

custom_tmpdir() {
  local case_root="$temporary/custom"
  mkdir -p "$case_root/tmp"
  run_facade "$case_root" "$case_root/tmp"
  [[ "$result" == 0 ]]
  delegation_matches "$case_root"
  [[ $(cat "$case_root/ledger.candidate") == "$case_root/tmp/"* ]]
  [[ -z $(find "$case_root/tmp" -mindepth 1 -print -quit) ]]
}

missing_tmpdir() {
  local case_root="$temporary/missing"
  run_facade "$case_root" unset
  [[ "$result" == 0 ]]
  delegation_matches "$case_root"
  [[ ! -e "$case_root/absent" ]]
}

execution_refusal() {
  local case_root="$temporary/refusal"
  mkdir -p "$case_root/tmp"
  run_facade "$case_root" "$case_root/tmp" 126
  [[ "$result" != 0 && $(cat "$case_root/ledger") == version ]]
  grep -Fq 'choose an executable TMPDIR' "$case_root/stderr"
  no_persistent_effects "$case_root"
  [[ -z $(find "$case_root/tmp" -mindepth 1 -print -quit) ]]
}

missing_interpreter_refusal() {
  local case_root="$temporary/missing-interpreter" candidate_archive candidate_bundle
  mkdir -p "$case_root/tmp" "$case_root/source"
  cp -R "$bundle" "$case_root/source/$bundle_name"
  candidate_bundle="$case_root/source/$bundle_name"
  printf '#!/nonexistent/axiom-fixture-interpreter\n' >"$candidate_bundle/axiom"
  (
    cd "$candidate_bundle"
    find . -type f ! -name MANIFEST.sha256 -print | sed 's#^./##' | LC_ALL=C sort | while IFS= read -r file; do
      printf '%s  %s\n' "$(digest "$file")" "$file"
    done
  ) >"$case_root/manifest"
  mv "$case_root/manifest" "$candidate_bundle/MANIFEST.sha256"
  candidate_archive="$case_root/$bundle_name.tar.gz"
  tar -czf "$candidate_archive" -C "$case_root/source" "$bundle_name"
  printf '%s  %s.tar.gz\n' "$(digest "$candidate_archive")" "$bundle_name" >"$case_root/SHA256SUMS"
  run_facade "$case_root" "$case_root/tmp" 0 0 "$case_root/SHA256SUMS" 0 "$candidate_archive"
  [[ "$result" != 0 && ! -s "$case_root/ledger" && ! -e "$case_root/ledger.candidate" ]]
  grep -Fq 'choose an executable TMPDIR' "$case_root/stderr"
  no_persistent_effects "$case_root"
  [[ -z $(find "$case_root/tmp" -mindepth 1 -print -quit) ]]
}

exit_propagation() {
  local case_root="$temporary/exit75"
  mkdir -p "$case_root/tmp"
  run_facade "$case_root" "$case_root/tmp" 0 75
  [[ "$result" == 75 ]]
  delegation_matches "$case_root"
  [[ -z $(find "$case_root/tmp" -mindepth 1 -print -quit) ]]
}

checksum_refusal() {
  local case_root="$temporary/checksum"
  mkdir -p "$case_root/tmp"
  printf '%064d  %s.tar.gz\n' 0 "$bundle_name" >"$case_root/bad-checksums"
  run_facade "$case_root" "$case_root/tmp" 0 0 "$case_root/bad-checksums"
  [[ "$result" != 0 && ! -s "$case_root/ledger" ]]
  grep -Fq 'archive checksum mismatch' "$case_root/stderr"
  no_persistent_effects "$case_root"
  [[ -z $(find "$case_root/tmp" -mindepth 1 -print -quit) ]]
}

unsafe_workspace() {
  local variant=$1 case_root="$temporary/unsafe-$1" temp_root
  mkdir -p "$case_root/tmp"
  temp_root="$case_root/tmp"
  case "$variant" in
    tmpdir) chmod 777 "$case_root/tmp" ;;
    ancestor) chmod 777 "$case_root" ;;
    symlink) ln -s tmp "$case_root/link"; temp_root="$case_root/link" ;;
  esac
  run_facade "$case_root" "$temp_root" 0 0 "$checksums" 1
  [[ "$result" != 0 && ! -s "$case_root/ledger" ]]
  grep -Fq 'unsafe temporary workspace' "$case_root/stderr"
  no_persistent_effects "$case_root"
  [[ -z $(find "$case_root/tmp" -mindepth 1 -print -quit) ]]
}

# Run each assertion set with errexit active; failures remain independently
# visible without allowing failed assertions to be masked by later commands.
step() {
  local label=$1 status
  shift
  set +e
  (set -e; "$@") >"$temporary/step.log" 2>&1
  status=$?
  set -e
  if [[ "$status" == 0 ]]; then
    printf 'case=%s result=pass\n' "$label"
  else
    printf 'case=%s result=fail\n' "$label"
    sed -n '1,12p' "$temporary/step.log"
    failures=$((failures + 1))
  fi
}

macos_version_agnostic() {
  local version=$1 case_root="$temporary/macos-version-$1"
  mkdir -p "$case_root/tmp"
  write_sw_vers "$version"
  run_facade "$case_root" "$case_root/tmp"
  [[ "$result" == 0 ]]
  delegation_matches "$case_root"
  [[ ! -e "$temporary/sw_vers.invoked" ]]
}

printf 'suite=posix-facade-offline-fixtures scope=bootstrap-only\n'
step custom-tmpdir-private-cleaned custom_tmpdir
step missing-tmpdir-fallback missing_tmpdir
step candidate-version-execution-refusal execution_refusal
step verified-candidate-missing-interpreter-refusal missing_interpreter_refusal
step exact-delegation-exit75 exit_propagation
step original-inputs-replaced-staged-bytes-retained original_inputs_replaced
step checksum-before-candidate checksum_refusal
step controlled-workspace-unsafe-tmpdir-refused unsafe_workspace tmpdir
step controlled-workspace-unsafe-ancestor-refused unsafe_workspace ancestor
if [[ $(id -u) == 0 ]]; then
  printf 'case=user-symlink-refused result=skip reason=root-owned-alias-exception\n'
else
  step controlled-workspace-user-symlink-refused unsafe_workspace symlink
fi
if [[ "$platform" == macos-27 ]]; then
  for macos_version in 27.0.1 27.1 28.0; do
    step "macos-version-agnostic:$macos_version" macos_version_agnostic "$macos_version"
  done
fi
step sw-vers-never-invoked test ! -e "$temporary/sw_vers.invoked"
printf 'failures=%d\n' "$failures"
[[ "$failures" == 0 ]]
