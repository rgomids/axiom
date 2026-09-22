#!/usr/bin/env bash
set -euo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
temporary=$(mktemp -d)
temporary=$(cd "$temporary" && pwd -P)
trap 'rm -rf -- "$temporary"' EXIT

for invalid_version in 01.2.3 1.2.3. 1.2.3-01; do
  if "$repository_root/scripts/build-release-archives.sh" --version "$invalid_version" --output "$temporary/invalid-$invalid_version" --development >/dev/null 2>&1; then
    exit 1
  fi
done

"$repository_root/scripts/build-release-archives.sh" --version 0.0.0-s2-test --output "$temporary/release" --development >/dev/null

[[ $(find "$temporary/release" -name '*.tar.gz' -type f | wc -l | tr -d ' ') == 3 ]]
[[ $(wc -l <"$temporary/release/SHA256SUMS" | tr -d ' ') == 3 ]]
while read -r hash archive; do
  [[ "$hash" =~ ^[0-9a-f]{64}$ ]]
  [[ -f "$temporary/release/$archive" ]]
  actual=$(shasum -a 256 "$temporary/release/$archive" | awk '{print $1}')
  [[ "$actual" == "$hash" ]]
  listing=$(tar -tzf "$temporary/release/$archive")
  for expected in /lingo /LICENSE /install.sh /release-metadata.txt /skills-manifest.txt /MANIFEST.sha256; do
    grep -Fq "$expected" <<<"$listing"
  done
  [[ $(grep -c '/skills/.*/SKILL.md' <<<"$listing") == 5 ]]
done <"$temporary/release/SHA256SUMS"

case "$(uname -s):$(uname -m)" in
  Darwin:arm64) native="$temporary/release/axiom-0.0.0-s2-test-macos-27-arm64.tar.gz" ;;
  Linux:x86_64) native="$temporary/release/axiom-0.0.0-s2-test-ubuntu-26.04-amd64.tar.gz" ;;
  Linux:aarch64) native="$temporary/release/axiom-0.0.0-s2-test-ubuntu-26.04-arm64.tar.gz" ;;
  *) native= ;;
esac

if [[ -n "$native" ]]; then
  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/bin" --receipt-dir "$temporary/receipt" | grep -Fq 'install_status=installed'
  "$temporary/bin/lingo" --json version | grep -Fq '"version":"development"'
  if AXIOM_CODEX_SKILLS_ROOT="$temporary/runtime-skills" "$temporary/bin/lingo" --json first-run >"$temporary/first-run.json"; then
    exit 1
  fi
  mkdir -p "$temporary/native-bundle"
  tar -xzf "$native" -C "$temporary/native-bundle"
  archived_skills_manifest=$(find "$temporary/native-bundle" -name skills-manifest.txt -type f)
  while IFS='=' read -r key hash; do
    case "$key" in
      skill.*)
        name=${key#skill.}
        grep -Fq '"name":"'"$name"'","sha256":"'"$hash"'","state":"missing"' "$temporary/first-run.json"
        ;;
    esac
  done <"$archived_skills_manifest"
  before=$(shasum -a 256 "$temporary/bin/lingo" | awk '{print $1}')
  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/bin" --receipt-dir "$temporary/receipt" | grep -Fq 'install_status=unchanged'
  [[ $(shasum -a 256 "$temporary/bin/lingo" | awk '{print $1}') == "$before" ]]

  mkdir -p "$temporary/foreign-bin"
  printf 'foreign\n' >"$temporary/foreign-bin/lingo"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/foreign-bin" --receipt-dir "$temporary/foreign-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ $(cat "$temporary/foreign-bin/lingo") == foreign ]]

  cp "$native" "$temporary/tampered.tar.gz"
  printf 'tamper\n' >>"$temporary/tampered.tar.gz"
  if "$repository_root/scripts/install-release.sh" --archive "$temporary/tampered.tar.gz" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/tampered-bin" --receipt-dir "$temporary/tampered-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ ! -e "$temporary/tampered-bin/lingo" ]]

  wrong_platform=$(find "$temporary/release" -name '*.tar.gz' -type f ! -path "$native" | LC_ALL=C sort | head -n 1)
  if "$repository_root/scripts/install-release.sh" --archive "$wrong_platform" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/wrong-bin" --receipt-dir "$temporary/wrong-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ ! -e "$temporary/wrong-bin/lingo" ]]

  mkdir -p "$temporary/real-bin"
  ln -s "$temporary/real-bin" "$temporary/symlink-bin"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/symlink-bin" --receipt-dir "$temporary/symlink-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ ! -e "$temporary/real-bin/lingo" ]]

  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/hardlink-bin" --receipt-dir "$temporary/hardlink-receipt" >/dev/null
  ln "$temporary/hardlink-bin/lingo" "$temporary/hardlink-copy"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/hardlink-bin" --receipt-dir "$temporary/hardlink-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ -f "$temporary/hardlink-bin/lingo" && -f "$temporary/hardlink-copy" ]]

  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/receipt-link-bin" --receipt-dir "$temporary/receipt-link-state" >/dev/null
  ln "$temporary/receipt-link-state/installation.receipt" "$temporary/receipt-link-copy"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/receipt-link-bin" --receipt-dir "$temporary/receipt-link-state" >/dev/null 2>&1; then
    exit 1
  fi
  [[ -f "$temporary/receipt-link-state/installation.receipt" && -f "$temporary/receipt-link-copy" ]]

  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/mode-bin" --receipt-dir "$temporary/mode-receipt" >/dev/null
  chmod 755 "$temporary/mode-bin/lingo"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/mode-bin" --receipt-dir "$temporary/mode-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ $(stat -f %Lp "$temporary/mode-bin/lingo" 2>/dev/null || stat -c %a "$temporary/mode-bin/lingo") == 755 ]]

  if AXIOM_INSTALL_TEST_FAIL_STAGE=before_binary "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/pre-bin" --receipt-dir "$temporary/pre-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ ! -e "$temporary/pre-bin/lingo" ]]
  [[ ! -e "$temporary/pre-receipt/installation.receipt" ]]
  [[ ! -e "$temporary/pre-receipt/.axiom-install-operation" ]]

  if AXIOM_INSTALL_TEST_FAIL_STAGE=after_binary "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/post-bin" --receipt-dir "$temporary/post-receipt" >/dev/null 2>&1; then
    exit 1
  fi
  [[ -f "$temporary/post-bin/lingo" ]]
  [[ ! -e "$temporary/post-receipt/installation.receipt" ]]
  grep -Fq 'stage=binary_committed' "$temporary/post-receipt/.axiom-install-operation"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/post-bin" --receipt-dir "$temporary/post-receipt" >/dev/null 2>"$temporary/post-retry.stderr"; then
    exit 1
  fi
  grep -Fq 'recovery_required' "$temporary/post-retry.stderr"

  mkdir -p "$temporary/locked-receipt/.axiom-install.lock"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/locked-bin" --receipt-dir "$temporary/locked-receipt" >/dev/null 2>"$temporary/locked.stderr"; then
    exit 1
  fi
  grep -Fq 'concurrent installation refused' "$temporary/locked.stderr"
  [[ ! -e "$temporary/locked-bin/lingo" ]]

  for field in formatVersion destination sha256 version revision sourceState release platform goos architecture skillSetVersion archiveSha256 skillManifestSha256; do
    [[ $(grep -c "^${field}=" "$temporary/receipt/installation.receipt") == 1 ]]
  done
fi

printf '%s\n' 'PASS: S2 release archives, ownership, interruption, recovery, and conflict preservation'
