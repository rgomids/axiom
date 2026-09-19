#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
binary_root=${AXIOM_BIN_DIR:-"${HOME}/.local/bin"}
state_root=${AXIOM_INSTALL_STATE_ROOT:-"${XDG_STATE_HOME:-${HOME}/.local/state}/axiom/install"}
source_url=https://github.com/rgomids/axiom

case "$binary_root:$state_root" in
  /*:/*) ;;
  *) printf '%s\n' 'install_error: absolute destinations required' >&2; exit 1 ;;
esac
case "$binary_root$state_root" in
  *$'\n'*|*$'\r'*) printf '%s\n' 'install_error: invalid destination' >&2; exit 1 ;;
esac

ensure_private_directory() {
  local directory=$1
  if [[ -L "$directory" ]]; then
    printf '%s\n' 'install_error: symlink destination refused' >&2
    exit 1
  fi
  mkdir -p -- "$directory"
  chmod 700 "$directory"
}

ensure_private_directory "$binary_root"
ensure_private_directory "$state_root"

destination="$binary_root/lingo"
receipt="$state_root/lingo.receipt"
stage=$(mktemp "$binary_root/.lingo-axiom-stage.XXXXXX")
receipt_stage=$(mktemp "$state_root/.lingo-receipt-stage.XXXXXX")
cleanup() {
  rm -f -- "$stage" "$receipt_stage"
}
trap cleanup EXIT

commit=$(git -C "$repository_root" rev-parse --verify HEAD)
version="poc-${commit:0:12}"
go build -trimpath \
  -ldflags "-X main.buildVersion=$version -X main.buildCommit=$commit -X main.buildSource=$source_url" \
  -o "$stage" "$repository_root/cmd/lingo"
chmod 700 "$stage"
new_checksum=$(shasum -a 256 "$stage" | awk '{print $1}')

if [[ -e "$destination" || -L "$destination" ]]; then
  if [[ -L "$destination" || ! -f "$destination" ]]; then
    printf '%s\n' 'install_error: existing destination is not a regular file' >&2
    exit 1
  fi
  if cmp -s -- "$stage" "$destination"; then
    printf '{"event":"axiom_install","result":"unchanged","version":"%s","commit":"%s"}\n' "$version" "$commit"
    exit 0
  fi
  if [[ ! -f "$receipt" || -L "$receipt" ]]; then
    printf '%s\n' 'install_error: unowned destination refused' >&2
    exit 1
  fi
  recorded_destination=$(awk -F= '$1 == "destination" {sub(/^[^=]*=/, ""); print; exit}' "$receipt")
  recorded_checksum=$(awk -F= '$1 == "sha256" {print $2; exit}' "$receipt")
  current_checksum=$(shasum -a 256 "$destination" | awk '{print $1}')
  if [[ "$recorded_destination" != "$destination" || "$recorded_checksum" != "$current_checksum" ]]; then
    printf '%s\n' 'install_error: existing destination differs from owned install' >&2
    exit 1
  fi
fi

mv -f -- "$stage" "$destination"
stage=
{
  printf 'formatVersion=1\n'
  printf 'destination=%s\n' "$destination"
  printf 'sha256=%s\n' "$new_checksum"
  printf 'source=%s\n' "$source_url"
  printf 'commit=%s\n' "$commit"
} >"$receipt_stage"
chmod 600 "$receipt_stage"
mv -f -- "$receipt_stage" "$receipt"
receipt_stage=

printf '{"event":"axiom_install","result":"installed","version":"%s","commit":"%s"}\n' "$version" "$commit"
