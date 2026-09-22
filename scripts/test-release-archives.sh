#!/usr/bin/env bash
set -euo pipefail

umask 077

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

native=
host_candidate=
case "$(uname -s):$(uname -m)" in
  Darwin:arm64)
    host_candidate="$temporary/release/axiom-0.0.0-s2-test-macos-27-arm64.tar.gz"
    [[ $(sw_vers -productVersion 2>/dev/null) == 27.0 ]] && native=$host_candidate
    ;;
  Linux:x86_64)
    host_candidate="$temporary/release/axiom-0.0.0-s2-test-ubuntu-26.04-amd64.tar.gz"
    if [[ -r /etc/os-release ]] && grep -Eq '^ID=(ubuntu|"ubuntu")$' /etc/os-release && grep -Eq '^VERSION_ID=(26\.04|"26\.04")$' /etc/os-release; then native=$host_candidate; fi
    ;;
  Linux:aarch64)
    host_candidate="$temporary/release/axiom-0.0.0-s2-test-ubuntu-26.04-arm64.tar.gz"
    if [[ -r /etc/os-release ]] && grep -Eq '^ID=(ubuntu|"ubuntu")$' /etc/os-release && grep -Eq '^VERSION_ID=(26\.04|"26\.04")$' /etc/os-release; then native=$host_candidate; fi
    ;;
esac

if [[ -n "$host_candidate" && -z "$native" ]]; then
  if "$repository_root/scripts/install-release.sh" --archive "$host_candidate" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/unsupported-bin" --receipt-dir "$temporary/unsupported-receipt" >/dev/null 2>"$temporary/unsupported.stderr"; then
    exit 1
  fi
  grep -Fq 'exact approved OS, version, distribution, and architecture required' "$temporary/unsupported.stderr"
  [[ ! -e "$temporary/unsupported-bin/lingo" ]]
fi

if [[ -n "$native" ]]; then
  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/bin" --receipt-dir "$temporary/receipt" | grep -Fq 'install_status=installed'
  installed_at=$(awk -F= '$1 == "installedAt" {print $2}' "$temporary/receipt/installation.receipt")
  [[ "$installed_at" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]
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
  sleep 1
  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/bin" --receipt-dir "$temporary/receipt" | grep -Fq 'install_status=unchanged'
  [[ $(shasum -a 256 "$temporary/bin/lingo" | awk '{print $1}') == "$before" ]]
  [[ $(awk -F= '$1 == "installedAt" {print $2}' "$temporary/receipt/installation.receipt") == "$installed_at" ]]

  "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/schema-bin" --receipt-dir "$temporary/schema-receipt" >/dev/null
  printf 'unknown=value\n' >>"$temporary/schema-receipt/installation.receipt"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/schema-bin" --receipt-dir "$temporary/schema-receipt" >/dev/null 2>"$temporary/schema.stderr"; then
    exit 1
  fi
  grep -Fq 'invalid receipt schema preserved' "$temporary/schema.stderr"
  grep -Fxq 'unknown=value' "$temporary/schema-receipt/installation.receipt"

  if [[ $(uname -s) == Darwin ]]; then
    mkdir -p "$temporary/wrong-version-tools"
    printf '%s\n' '#!/bin/sh' "printf '%s\\n' '27.1'" >"$temporary/wrong-version-tools/sw_vers"
    chmod 700 "$temporary/wrong-version-tools/sw_vers"
    if PATH="$temporary/wrong-version-tools:$PATH" "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/wrong-version-bin" --receipt-dir "$temporary/wrong-version-receipt" >/dev/null 2>"$temporary/wrong-version.stderr"; then
      exit 1
    fi
    grep -Fq 'exact approved OS, version, distribution, and architecture required' "$temporary/wrong-version.stderr"
    [[ ! -e "$temporary/wrong-version-bin/lingo" ]]
  fi

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

  mkdir -p "$temporary/permissive-bin" "$temporary/permissive-receipt"
  chmod 770 "$temporary/permissive-bin"
  if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/permissive-bin" --receipt-dir "$temporary/permissive-receipt" >/dev/null 2>"$temporary/permissive.stderr"; then
    exit 1
  fi
  grep -Fq 'unsafe destination ownership, permissions, ACL, or type' "$temporary/permissive.stderr"
  [[ $(stat -f %Lp "$temporary/permissive-bin" 2>/dev/null || stat -c %a "$temporary/permissive-bin") == 770 ]]
  [[ ! -e "$temporary/permissive-bin/lingo" ]]

  if [[ $(uname -s) == Darwin ]]; then
    mkdir -p "$temporary/acl-bin" "$temporary/acl-receipt"
    chmod 700 "$temporary/acl-bin" "$temporary/acl-receipt"
    chmod +a 'everyone allow read,search' "$temporary/acl-bin"
    if "$repository_root/scripts/install-release.sh" --archive "$native" --checksums "$temporary/release/SHA256SUMS" --bin-dir "$temporary/acl-bin" --receipt-dir "$temporary/acl-receipt" >/dev/null 2>"$temporary/acl.stderr"; then
      exit 1
    fi
    grep -Fq 'unsafe destination ownership, permissions, ACL, or type' "$temporary/acl.stderr"
    [[ ! -e "$temporary/acl-bin/lingo" ]]
  fi

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

  [[ $(wc -l <"$temporary/receipt/installation.receipt" | tr -d ' ') == 15 ]]
  for field in formatVersion destination sha256 product version revision sourceState release platform goos architecture skillSetVersion archiveSha256 skillManifestSha256 installedAt; do
    [[ $(grep -c "^${field}=" "$temporary/receipt/installation.receipt") == 1 ]]
  done
fi

printf '%s\n' 'PASS: S2 release archives, ownership, interruption, recovery, and conflict preservation'
