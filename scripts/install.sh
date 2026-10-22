#!/bin/sh
# Axiom remote installer bootstrap.
#
#   curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh
#   ... | sh -s -- --channel stable
#   ... | sh -s -- --version v0.1.0-rc.2
#
# Resolves one published Axiom release, downloads the archive for this exact
# supported host row plus the release SHA256SUMS, verifies the archive digest
# before anything is extracted, and hands the verified archive to the release
# installer it contains (install.sh in the bundle). That installer owns every
# installation effect: ownership, receipts, no-op reinstall, protected owned
# upgrade, downgrade refusal and recovery. This bootstrap only reads the
# network and writes a private temporary directory that it removes.
#
# It never uses sudo, edits shell profiles or PATH, installs Runtimes, or
# touches credentials.
set -eu
umask 077

repository_url=https://github.com/rgomids/axiom
supported_rows='macOS/arm64, Linux/amd64, Linux/arm64'
banner_printed=false

if [ -t 2 ] && [ -z "${NO_COLOR+x}" ] && [ -n "${TERM:-}" ] && [ "${TERM:-}" != dumb ]; then
  style_bold=$(printf '\033[1m')
  style_dim=$(printf '\033[2m')
  style_blue=$(printf '\033[34m')
  style_cyan=$(printf '\033[36m')
  style_green=$(printf '\033[32m')
  style_red=$(printf '\033[31m')
  style_underline=$(printf '\033[4m')
  style_reset=$(printf '\033[0m')
else
  style_bold=
  style_dim=
  style_blue=
  style_cyan=
  style_green=
  style_red=
  style_underline=
  style_reset=
fi

logo_standard() {
  printf '%s' "$style_bold"
  cat <<'EOF'
                                     +
                                   +++++
                                 +++++++++
                                +++++++++++
                              ++++++  .+++++*
                               ++++     ++++
                                ++++  :++++
                         ++++++   +++ +++   ++++++
                             +++   ++ ++   +++
                             +++ + ++ ++ + +++
                      +++++++++ ++ ++ ++ ++ +++++++++
                   ++++++++    +++ ++ ++ +++    ++++++++
                  *++       ++++  +++ +++ *++++       ++
                  ++      ++++   +++   +++   +++       ++
                   ++     ++   ++++     ++++   ++     ++
                     +    ++  +++         +++  ++   **
                           +  ++           ++  +
                               +           +
             +        +++    ++     ++        +++       +         +
            +++         ++ +++      ++     +++   ++     +++     +++
          ++  ++          ++        ++    ++      ++    +++++ +++++
         ++     ++      ++ +++      ++     ++    +++    ++  +++  ++
        ++       ++   +++    ++     ++      ++++++      ++       ++
EOF
  printf '%s' "$style_reset"
}

banner() {
  [ "$banner_printed" = false ] || return 0
  banner_printed=true
  {
    printf '\n'
    logo_standard
    printf '\n'
    printf '%s----- Keep intent, decisions, code, and evidence connected.%s\n' "$style_dim" "$style_reset"
    printf '\n'
  } >&2
}

info() {
  banner
  printf '%sinstall_step:%s %s\n' "$style_blue" "$style_reset" "$1" >&2
}

success() {
  banner
  printf '%sinstall_ok:%s %s\n' "$style_green" "$style_reset" "$1" >&2
}

display_path() {
  case "$1" in
    "$HOME") printf '~\n' ;;
    "$HOME"/*) printf '~/%s\n' "${1#"$HOME"/}" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

shell_export_path() {
  if [ "$binary_root" = "$HOME/.local/bin" ]; then
    printf 'export PATH="$HOME/.local/bin:$PATH"\n'
    return 0
  fi
  quoted=$(printf '%s' "$binary_root" | sed "s/'/'\\\\''/g")
  printf "export PATH='%s':\"\$PATH\"\n" "$quoted"
}

human_summary() {
  install_status=$1
  path_required=$2
  case "$install_status" in
    installed) summary_title='Installation complete'; summary_status='Installed' ;;
    upgraded) summary_title='Installation complete'; summary_status='Upgraded' ;;
    unchanged) summary_title='Axiom is already up to date'; summary_status='Unchanged' ;;
    *) summary_title='Installation complete'; summary_status=$install_status ;;
  esac
  docs_url=$repository_url/tree/main/docs
  location=$(display_path "$binary_root/axiom")

  printf '\n' >&2
  printf '%s%s%s\n\n' "$style_bold$style_green" "$summary_title" "$style_reset" >&2
  printf '  %s%-13s%s %s%s%s\n' "$style_bold" Status "$style_reset" "$style_green" "$summary_status" "$style_reset" >&2
  printf '  %s%-13s%s %s%s%s\n' "$style_bold" Version "$style_reset" "$style_cyan" "$tag" "$style_reset" >&2
  printf '  %s%-13s%s %s%s%s\n\n' "$style_bold" Location "$style_reset" "$style_cyan" "$location" "$style_reset" >&2
  printf '  %sDocumentation%s\n' "$style_bold" "$style_reset" >&2
  printf '  %s%s%s%s\n\n' "$style_blue" "$style_underline" "$docs_url" "$style_reset" >&2
  printf '  %sNext step%s\n' "$style_bold" "$style_reset" >&2
  printf '  %s%saxiom first-run%s\n\n' "$style_bold" "$style_cyan" "$style_reset" >&2
  if [ "$path_required" = true ]; then
    printf '  %sPATH setup required%s\n\n' "$style_bold" "$style_reset" >&2
    printf '  Add Axiom to your current shell:\n\n' >&2
    printf '  %s%s%s\n\n' "$style_cyan" "$(shell_export_path)" "$style_reset" >&2
  fi
}

fail() {
  banner
  printf '%sinstall_error:%s %s\n' "$style_red" "$style_reset" "$1" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage: install.sh [--channel stable | --version vMAJOR.MINOR.PATCH[-rc.N]]
                  [--bin-dir <absolute-dir>] [--receipt-dir <absolute-dir>]
                  [--helper <absolute-file> --helper-sha256 <trusted-pin>]

Without a selector, or with --channel stable, the latest published stable
release is installed. Release candidates are installed only by exact version
(--version vMAJOR.MINOR.PATCH-rc.N); there is no release-candidate channel.
--channel and
--version are mutually exclusive. Defaults: --bin-dir $HOME/.local/bin and
--receipt-dir ${XDG_STATE_HOME:-$HOME/.local/state}/axiom/install.
EOF
}

newline='
'
channel=
version_tag=
binary_root=
receipt_root=
helper=${AXIOM_VERIFIER_PATH:-}
helper_sha256=${AXIOM_VERIFIER_SHA256:-}
channel_seen=false
version_seen=false
bin_seen=false
receipt_seen=false
while [ $# -gt 0 ]; do
  case "$1" in
    --channel|--version|--bin-dir|--receipt-dir|--helper|--helper-sha256)
      [ $# -ge 2 ] || fail "$1 requires a value"
      case "$1" in
        --helper) helper=$2 ;;
        --helper-sha256) helper_sha256=$2 ;;
        --channel) [ "$channel_seen" = false ] || fail '--channel given more than once'; channel_seen=true; channel=$2 ;;
        --version) [ "$version_seen" = false ] || fail '--version given more than once'; version_seen=true; version_tag=$2 ;;
        --bin-dir) [ "$bin_seen" = false ] || fail '--bin-dir given more than once'; bin_seen=true; binary_root=$2 ;;
        --receipt-dir) [ "$receipt_seen" = false ] || fail '--receipt-dir given more than once'; receipt_seen=true; receipt_root=$2 ;;
      esac
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; fail 'invalid argument' ;;
  esac
done

if [ "$channel_seen" = true ] && [ "$version_seen" = true ]; then
  fail '--channel and --version are mutually exclusive; choose one selector'
fi
if [ "$version_seen" = true ]; then
  case "$version_tag" in *"$newline"*) fail 'invalid --version' ;; esac
  printf '%s\n' "$version_tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.(0|[1-9][0-9]*))?$' \
    || fail '--version must be an exact published tag vMAJOR.MINOR.PATCH or vMAJOR.MINOR.PATCH-rc.N'
  selector="version $version_tag"
else
  [ "$channel_seen" = true ] || channel=stable
  case "$channel" in
    stable) selector='channel stable' ;;
    rc)
      # Release candidates are exact-version only (FR-064, Issue #81): there
      # is no floating RC selector, so this is an unsupported selector with a
      # pointer to the exact-version form.
      fail 'release candidates require an explicit version: use --version vMAJOR.MINOR.PATCH-rc.N (--channel rc does not select a release candidate)'
      ;;
    *) fail '--channel must be stable; release candidates use --version vMAJOR.MINOR.PATCH-rc.N' ;;
  esac
fi

valid_directory() {
  case "$1" in
    /) return 1 ;;
    /*) ;;
    *) return 1 ;;
  esac
  case "$1" in
    *"$newline"*|*'='*|*//*|*/./*|*/../*|*/.|*/..|*/) return 1 ;;
  esac
  case "$1" in *"$(printf '\r')"*) return 1 ;; esac
  return 0
}

if [ -z "$binary_root" ] || [ -z "$receipt_root" ]; then
  [ -n "${HOME:-}" ] && valid_directory "$HOME" || fail 'HOME must be an absolute path when --bin-dir or --receipt-dir is omitted'
fi
[ -n "$binary_root" ] || binary_root=$HOME/.local/bin
if [ -z "$receipt_root" ]; then
  state_home=${XDG_STATE_HOME:-}
  valid_directory "$state_home" || state_home=$HOME/.local/state
  receipt_root=$state_home/axiom/install
fi
valid_directory "$binary_root" || fail '--bin-dir must be an absolute canonical path without newlines or ='
valid_directory "$receipt_root" || fail '--receipt-dir must be an absolute canonical path without newlines or ='

# Supported OS family and architecture; anything else stops before any
# download. The OS version is never an eligibility filter.
system=$(uname -s)
machine=$(uname -m)
row=
case "$system:$machine" in
  Darwin:arm64)
    # Any macOS version runs the darwin/arm64 build.
    row=macos-27-arm64
    ;;
  Linux:x86_64|Linux:aarch64)
    # Any Linux distribution runs the static linux build.
    case "$machine" in
      x86_64) row=linux-amd64 ;;
      aarch64) row=linux-arm64 ;;
    esac
    ;;
esac
[ -n "$row" ] || fail "unsupported host $system/$machine; supported hosts are $supported_rows"
info "checking required tools for $row"

for tool in curl tar gzip bash awk grep mktemp jq od cut cp chmod wc tr sort cmp stat id ls dirname find; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool not found: $tool"
done
if command -v sha256sum >/dev/null 2>&1; then
  digest() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
  digest() { shasum -a 256 "$1" | awk '{print $1}'; }
else
  fail 'required tool not found: sha256sum or shasum'
fi
success "host $row is supported"

# fetch <url> <output> <max-bytes>: HTTPS only, including redirects.
fetch() {
  curl --proto '=https' --tlsv1.2 --fail --silent --show-error --location \
    --retry 2 --connect-timeout 20 --max-time 300 --max-filesize "$3" \
    --output "$2" "$1"
}

# Resolve the selector to one exact tag.
if [ "$version_seen" = true ]; then
  tag=$version_tag
else
  info "resolving $selector"
  # GitHub redirects /releases/latest to the latest published non-prerelease,
  # non-draft release, or to /releases when there is none. Only the redirect
  # target is read; the page itself is not fetched or parsed.
  set +e
  location=$(curl --proto '=https' --tlsv1.2 --fail --silent --show-error \
    --retry 2 --connect-timeout 20 --max-time 60 \
    --output /dev/null --write-out '%{redirect_url}' "$repository_url/releases/latest")
  status=$?
  set -e
  [ "$status" -eq 0 ] || fail "network failure while resolving the latest stable release (curl exit $status); nothing was installed"
  case "$location" in
    "$repository_url/releases")
      fail 'no published stable Axiom release exists yet; release candidates are opt-in: use --version vMAJOR.MINOR.PATCH-rc.N'
      ;;
    "$repository_url/releases/tag/"*)
      tag=${location#"$repository_url/releases/tag/"}
      ;;
    *)
      fail 'the latest stable release could not be resolved unambiguously; use --version vMAJOR.MINOR.PATCH'
      ;;
  esac
  case "$tag" in *"$newline"*) fail 'invalid latest stable release tag' ;; esac
  printf '%s\n' "$tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' \
    || fail 'the latest release tag is not a stable vMAJOR.MINOR.PATCH tag; use --version'
fi
version=${tag#v}
success "resolved $tag"

work=$(mktemp -d)
[ -n "$work" ] && [ -d "$work" ] || fail 'private temporary directory unavailable'
cleanup() { rm -rf -- "$work"; }
trap cleanup EXIT
trap 'exit 130' HUP INT TERM
chmod 700 "$work"

# Refuse unsafe staging ancestors before authenticating/executing any helper.
stat_style=linux
if stat -f %u "$work" >/dev/null 2>&1; then stat_style=darwin; fi
owner_of() { if [ "$stat_style" = darwin ]; then stat -f %u "$1"; else stat -c %u "$1"; fi; }
mode_of() { if [ "$stat_style" = darwin ]; then stat -f %Lp "$1"; else stat -c %a "$1"; fi; }
[ ! -h "$work" ] && [ "$(owner_of "$work")" = "$(id -u)" ] && [ "$(mode_of "$work")" = 700 ] || fail 'unsafe private workspace'
current=$work
while :; do
  owner=$(owner_of "$current") || fail 'workspace ancestor unavailable'
  [ "$owner" = 0 ] || [ "$owner" = "$(id -u)" ] || fail 'unsafe workspace ancestor owner'
  [ ! -h "$current" ] || [ "$owner" = 0 ] || fail 'unsafe workspace ancestor symlink'
  mode=$(mode_of "$current")
  case "$mode" in *[!0-7]*|'') fail 'invalid workspace ancestor mode' ;; esac
  if [ "$((0$mode & 0022))" -ne 0 ]; then
    [ -n "$(find "$current" -maxdepth 0 -perm -1000 -print 2>/dev/null)" ] || fail 'unsafe workspace ancestor permissions'
  fi
  if [ "$stat_style" = darwin ]; then
    [ "$(LC_ALL=C ls -lde "$current" | wc -l | tr -d ' ')" = 1 ] || fail 'unsafe workspace ACL'
  else
    LC_ALL=C ls -ld "$current" | awk 'length($1) != 10 {exit 1}' || fail 'unsafe workspace ACL'
  fi
  [ "$current" != / ] || break
  current=$(dirname "$current")
done

download_url=$repository_url/releases/download/$tag
info "downloading checksums for $tag"
set +e
fetch "$download_url/SHA256SUMS" "$work/SHA256SUMS" 65536
status=$?
set -e
case "$status" in
  0) ;;
  22) fail "release $tag has no published SHA256SUMS; nothing was installed" ;;
  *) fail "network failure while downloading SHA256SUMS for $tag (curl exit $status); nothing was installed" ;;
esac
# GitHub API metadata is a separate authenticated control-plane channel.
# Never trust a checksum asset merely because it accompanied the helper.
fetch "https://api.github.com/repos/rgomids/axiom/releases/tags/$tag" "$work/release.json" 1048576 \
  || fail 'trusted GitHub release metadata unavailable; nothing was installed'
metadata_digest=$(jq -er --arg tag "$tag" '
  select(.tag_name == $tag and .draft == false) |
  [.assets[] | select(.name == "SHA256SUMS" and .state == "uploaded")] |
  select(length == 1) | .[0].digest | select(test("^sha256:[0-9a-f]{64}$")) | sub("^sha256:"; "")
' "$work/release.json") || fail 'authenticated checksum digest absent; nothing was installed'
[ "$(digest "$work/SHA256SUMS")" = "$metadata_digest" ] || fail 'authenticated checksum inventory mismatch; nothing was installed'
# The complete checksum set selects one generation, for every host. Linux
# names are shared; the macOS token disambiguates v1/v2 without probing URLs.
archive_format=$(awk -v v="$version" '
  NF != 2 || length($1) != 64 || $1 ~ /[^0-9a-f]/ { bad=1; next }
  { if (seen[$2]++) bad=1 }
  $2 == "axiom-" v "-macos-27-arm64.tar.gz" { legacy++; next }
  $2 == "axiom-" v "-macos-arm64.tar.gz" { current++; next }
  $2 == "axiom-" v "-linux-amd64.tar.gz" { amd++; next }
  $2 == "axiom-" v "-linux-arm64.tar.gz" { arm++; next }
  $2 ~ ("^axiom-verifier-" v "-[0-9a-f]+-(darwin-arm64|linux-amd64|linux-arm64)$") {
    n=split($2, p, "-"); commit=p[n-2]; if (length(commit) != 40 || index($2, "axiom-verifier-" v "-" commit "-") != 1) bad=1;
    if (helpercommit != "" && helpercommit != commit) bad=1; helpercommit=commit;
    host=p[n-1] "-" p[n]; if (helpers[host]++) bad=1; helpercount++; next
  }
  { bad=1 }
  END { if (bad || (NR != 3 && NR != 6) || (helpercount != 0 && helpercount != 3) || amd != 1 || arm != 1 || legacy+current != 1) exit 1; print current ? 2 : 1 }
' "$work/SHA256SUMS") || fail "release $tag checksum asset set is ambiguous or incomplete; nothing was installed"
# Reject foreign or missing release assets, including duplicate asset names.
{ printf 'SHA256SUMS\n'; awk '{print $2}' "$work/SHA256SUMS"; } | LC_ALL=C sort >"$work/expected-assets"
jq -er '.assets[].name' "$work/release.json" | LC_ALL=C sort >"$work/actual-assets"
cmp -s "$work/expected-assets" "$work/actual-assets" || fail 'trusted release asset inventory mismatch'
if [ "$(wc -l <"$work/SHA256SUMS" | tr -d ' ')" = 6 ]; then
  fetch "https://api.github.com/repos/rgomids/axiom/commits/$tag" "$work/commit.json" 1048576 || fail 'trusted release commit unavailable'
  release_commit=$(jq -er '.sha | select(test("^[0-9a-f]{40}$"))' "$work/commit.json") || fail 'trusted release commit invalid'
  awk -v commit="$release_commit" '$2 ~ /^axiom-verifier-/ {n=split($2,p,"-"); if (p[n-2] != commit) exit 1}' "$work/SHA256SUMS" || fail 'helper release commit mismatch'
fi
if [ "$archive_format" = 2 ] && [ "$row" = macos-27-arm64 ]; then row=macos-arm64; fi
asset=axiom-$version-$row.tar.gz
bundle=axiom-$version-$row
expected=$(awk -v name="$asset" '$2 == name && NF == 2 {print $1}' "$work/SHA256SUMS")
[ "$(printf '%s\n' "$expected" | grep -c .)" = 1 ] && printf '%s\n' "$expected" | grep -Eq '^[0-9a-f]{64}$' \
  || fail "release $tag publishes no single checksum for $asset; nothing was installed"

info "downloading $asset"
set +e
fetch "$download_url/$asset" "$work/$asset" 268435456
status=$?
set -e
case "$status" in
  0) ;;
  22) fail "release $tag has no published asset $asset; nothing was installed" ;;
  *) fail "network failure while downloading $asset (curl exit $status); nothing was installed" ;;
esac
actual=$(digest "$work/$asset")
[ "$actual" = "$expected" ] || fail "checksum mismatch for $asset; nothing was installed"
success "verified $asset"

# Resolve a verifier from the authenticated closed inventory, or an explicitly
# pinned local verifier for historical releases which predate helper assets.
case "$row" in macos*) helper_host=darwin-arm64 ;; *) helper_host=$row ;; esac
if [ -z "$helper" ]; then
  helper_name=$(awk -v host="$helper_host" '$2 ~ /^axiom-verifier-/ && $2 ~ ("-" host "$") {print $2}' "$work/SHA256SUMS")
  [ "$(printf '%s\n' "$helper_name" | grep -c .)" = 1 ] || fail 'trusted release helper absent; supply independently pinned --helper for legacy release'
  helper_sha256=$(awk -v name="$helper_name" '$2 == name {print $1}' "$work/SHA256SUMS")
  fetch "$download_url/$helper_name" "$work/$helper_name" 16777216 || fail 'helper download failed'
  helper=$work/$helper_name
else
  case "$helper" in /*) ;; *) fail 'absolute local helper required' ;; esac
  current=$helper
  while [ "$current" != / ]; do
    [ ! -h "$current" ] || [ "$(owner_of "$current")" = 0 ] || fail 'unsafe helper ancestor symlink'
    current=$(dirname "$current")
  done
  [ -f "$helper" ] && [ ! -L "$helper" ] && [ "$(wc -c <"$helper")" -le 16777216 ] || fail 'unsafe local helper'
  helper_name=$(basename "$helper")
  cp "$helper" "$work/$helper_name"
  helper=$work/$helper_name
fi
printf '%s\n' "$helper_sha256" | grep -Eq '^[0-9a-f]{64}$' || fail 'independent helper checksum required'
[ "$(digest "$helper")" = "$helper_sha256" ] || fail 'helper checksum mismatch before execution'
case "$helper_name" in *-"$helper_host") ;; *) fail 'helper platform identity mismatch' ;; esac
printf '%s\n' "$helper_name" | grep -Eq '^axiom-verifier-[0-9A-Za-z.+-]+-[0-9a-f]{40}-(darwin|linux)-(amd64|arm64)$' || fail 'helper release identity invalid'
header=$(od -An -tx1 -N20 "$helper" | tr -d ' \n')
case "$helper_host:$header" in
  darwin-arm64:cffaedfe0c000001*) ;;
  linux-amd64:7f454c46020101*) [ "$(printf '%s' "$header" | cut -c37-40)" = 3e00 ] || fail 'helper platform mismatch' ;;
  linux-arm64:7f454c46020101*) [ "$(printf '%s' "$header" | cut -c37-40)" = b700 ] || fail 'helper platform mismatch' ;;
  *) fail 'helper platform mismatch before execution' ;;
esac
chmod 700 "$helper"
[ "$("$helper" identity)" = "$helper_name" ] || fail 'helper release linkage mismatch'
"$helper" "$work/$asset" "$work/archive.tar" || fail 'strict archive integrity failed before installer execution'
# Only now read the verified archive: take the bundle's release installer and
# release metadata, regular files whose digests must match the bundle
# manifest, and require the metadata to be the resolved version and host row
# of a clean release build.
# Fixed member names without whitespace; expanded unquoted on purpose.
members="$bundle/install.sh $bundle/release-metadata.txt $bundle/MANIFEST.sha256"
info "checking verified bundle metadata"
listing=$(tar -tvf "$work/archive.tar" $members 2>/dev/null) \
  || fail "verified archive lacks its release installer or metadata; nothing was installed"
[ "$(printf '%s\n' "$listing" | awk 'substr($1,1,1) == "-"' | grep -c .)" = 3 ] \
  || fail 'release installer or metadata entry is not a regular file; nothing was installed'
mkdir "$work/bundle"
tar -xf "$work/archive.tar" -C "$work/bundle" $members
for member in install.sh release-metadata.txt; do
  path=$work/bundle/$bundle/$member
  [ -f "$path" ] && [ ! -h "$path" ] || fail "bundle $member is not a regular file; nothing was installed"
  [ "$(awk -v name="$member" '$2 == name && NF == 2 {print $1}' "$work/bundle/$bundle/MANIFEST.sha256")" = "$(digest "$path")" ] \
    || fail "bundle $member does not match its manifest; nothing was installed"
done
metadata=$work/bundle/$bundle/release-metadata.txt
for key in formatVersion product version sourceState release platform architecture; do
  [ "$(awk -F= -v key="$key" '$1 == key {n++} END {print n+0}' "$metadata")" = 1 ] \
    || fail "release metadata is ambiguous; nothing was installed"
done
platform=${row%-*}
architecture=${row##*-}
for expected_line in "formatVersion=$archive_format" "product=Axiom" "version=$version" "sourceState=clean" "release=true" "platform=$platform" "architecture=$architecture"; do
  grep -Fxq "$expected_line" "$metadata" || fail "release metadata does not match $tag for $row; nothing was installed"
done
if [ -n "${release_commit:-}" ]; then
  [ "$(awk -F= '$1 == "revision" {print $2}' "$metadata")" = "$(printf '%s' "$release_commit" | cut -c1-12)" ] || fail 'archive release commit mismatch'
fi
installer=$work/bundle/$bundle/install.sh
success "bundle metadata matches $tag for $row"

printf 'install_selector=%s\n' "$selector"
printf 'install_tag=%s\n' "$tag"
printf 'install_version=%s\n' "$version"
printf 'install_asset=%s\n' "$asset"
printf 'install_asset_sha256=%s\n' "$actual"
printf 'install_row=%s\n' "$row"

info "running verified release installer"
release_stdout=$work/release-installer.stdout
set +e
bash "$installer" --helper "$helper" --helper-sha256 "$helper_sha256" --archive "$work/$asset" --checksums "$work/SHA256SUMS" --bin-dir "$binary_root" --receipt-dir "$receipt_root" >"$release_stdout"
status=$?
set -e
cat "$release_stdout"
[ "$status" -eq 0 ] || exit "$status"
install_status=$(awk -F= '$1 == "install_status" {print $2; exit}' "$release_stdout")
[ -n "$install_status" ] || install_status=installed

receipt=$receipt_root/installation.receipt
printf 'install_revision=%s\n' "$(awk -F= '$1 == "revision" {print $2}' "$receipt")"
printf 'install_receipt=%s\n' "$receipt"
printf 'install_binary=%s/axiom\n' "$binary_root"
path_required=false
case ":${PATH:-}:" in
  *":$binary_root:"*) ;;
  *) path_required=true ;;
esac
success "installation complete"
human_summary "$install_status" "$path_required"
