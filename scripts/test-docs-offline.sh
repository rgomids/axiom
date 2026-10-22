#!/usr/bin/env bash
set -euo pipefail

# Installs archive generation 2 into a throwaway prefix; checks discovery,
# corpus links/hashes and refusal. Empty environment/CWD outside the checkout
# do not deny checkout or network access. This is development evidence, not V8.

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
archive=
checksums=
while (($#)); do
  case "$1" in
    --archive|--checksums)
      [[ $# -ge 2 && "$2" == /* && -f "$2" ]] || { printf 'docs_offline_error: absolute existing file required\n' >&2; exit 1; }
      if [[ "$1" == --archive ]]; then archive=$2; else checksums=$2; fi
      shift 2 ;;
    --help)
      printf 'Usage: test-docs-offline.sh [--archive /path/archive.tar.gz --checksums /path/SHA256SUMS]\nLocal checks only; no network/path isolation or Runtime qualification.\n'
      exit 0 ;;
    *) printf 'docs_offline_error: invalid argument\n' >&2; exit 1 ;;
  esac
done
[[ -z "$archive" && -z "$checksums" || -n "$archive" && -n "$checksums" ]] || { printf 'docs_offline_error: archive and checksums required together\n' >&2; exit 1; }
work=$(mktemp -d)
trap 'chmod -R u+w "$work" 2>/dev/null || true; rm -rf -- "$work"' EXIT

# Trust this checkout for the test verifier, independently of fixture archives.
helper_name="axiom-verifier-0.0.0-test-$(git -C "$repository_root" rev-parse HEAD)-$(go env GOHOSTOS)-$(go env GOHOSTARCH)"
export AXIOM_VERIFIER_PATH="$work/$helper_name"
(cd "$repository_root" && CGO_ENABLED=0 go build -trimpath -ldflags "-X main.identity=$helper_name" -o "$AXIOM_VERIFIER_PATH" ./cmd/axiom-archive-verify)
export AXIOM_VERIFIER_SHA256="$(shasum -a 256 "$AXIOM_VERIFIER_PATH" | awk '{print $1}')"

work=$(cd "$work" && pwd -P)

case "$(uname -s):$(uname -m)" in
  Darwin:arm64) row=macos-arm64 ;;
  Linux:x86_64) row=linux-amd64 ;;
  Linux:aarch64) row=linux-arm64 ;;
  *) printf 'docs_offline_skip: unsupported host\n'; exit 0 ;;
esac

fail() { printf 'docs_offline_error: %s\n' "$1" >&2; exit 1; }

if [[ -z "$archive" ]]; then
  version=0.0.0-docs.offline
  "$repository_root/scripts/build-release-archives.sh" --version "$version" --output "$work/release" --format 2 --development --only "$row" >/dev/null
  archive=$(ls "$work/release"/axiom-"$version"-*.tar.gz)
  checksums="$work/release/SHA256SUMS"
fi
mkdir -m 700 "$work/bin" "$work/receipts" "$work/home"

"$repository_root/scripts/install-release.sh" --archive "$archive" --checksums "$checksums" --bin-dir "$work/bin" --receipt-dir "$work/receipts" >/dev/null

inspect() { (cd / && env -i HOME="$work/home" PATH=/usr/bin:/bin "$work/bin/axiom" --json docs inspect "$@"); }

report=$(inspect) || fail 'verified installation did not exit 0'
field() { printf '%s' "$1" | python3 -I -c 'import json,sys; d=json.load(sys.stdin)["docs"]; v=d
for k in sys.argv[1].split("."): v=v[k]
print(v)' "$2"; }

[[ $(field "$report" state) == verified && $(field "$report" reason) == ok ]] || fail 'installed documentation is not verified'
index=$(field "$report" paths.index)
root=$(field "$report" paths.root)
case "$root" in "$repository_root"*) fail 'documentation path points into the checkout' ;; esac
[[ -f "$index" ]] || fail 'index is missing'

python3 -I - "$report" <<'PY'
import hashlib, json, os, re, sys
docs = json.loads(sys.argv[1])["docs"]
root = docs["paths"]["root"]
rows = [l.split("\t") for l in open(docs["paths"]["inventory"]).read().splitlines()[2:]]
assert rows, "empty inventory"
known = {r[0] for r in rows}
for path, kind, size, sha in rows:
    data = open(os.path.join(os.path.dirname(root), path), "rb").read()
    assert len(data) == int(size) and hashlib.sha256(data).hexdigest() == sha, path
for path in known:
    if not path.endswith(".md"):
        continue
    text = open(os.path.join(os.path.dirname(root), path)).read()
    for target in re.findall(r"\]\(([^)#\s]+)(?:#[^)]*)?\)", text):
        if re.match(r"^[a-z]+:", target):
            continue
        resolved = os.path.normpath(os.path.join(os.path.dirname(path), target))
        assert resolved in known, f"{path}: unresolved link {target}"
print("docs_offline_corpus: %d files, links closed" % len(rows))
PY

python3 -I - "$archive" "$report" "$row" <<'PY'
import hashlib, json, sys, tarfile
with open(sys.argv[1], "rb") as source:
    digest = hashlib.sha256()
    for block in iter(lambda: source.read(1024 * 1024), b""):
        digest.update(block)
    archive_hash = digest.hexdigest()
with tarfile.open(sys.argv[1], "r:gz") as source:
    metadata_entries = [m for m in source.getmembers() if m.name.endswith("/release-metadata.txt")]
    assert len(metadata_entries) == 1, "ambiguous metadata"
    metadata = dict(line.split("=", 1) for line in source.extractfile(metadata_entries[0]).read().decode().splitlines())
docs = json.loads(sys.argv[2])["docs"]
print("docs_offline_artifact: " + json.dumps({
    "hostRow": sys.argv[3], "archiveSha256": archive_hash,
    "metadata": {k: metadata[k] for k in ("version", "revision", "sourceState", "release", "platform", "architecture")},
    "binarySha256": docs["binary"]["sha256"], "sourceCommit": docs["binary"]["sourceCommit"],
    "installation": docs["installation"], "integrity": docs["integrity"],
    "networkDenial": "NOT_RUN", "sourceDenial": "NOT_RUN", "nativeRuntimes": "NOT_RUN"
}, sort_keys=True))
PY

if command -v man >/dev/null 2>&1; then
  selected=$(MANPATH="$(field "$report" paths.manRoot):" man -w axiom 2>/dev/null || true)
  [[ -z "$selected" || "$selected" == "$(field "$report" man.page)" ]] || printf 'docs_offline_note: man selected a different page\n'
fi

expect_refused() {
  local label=$1 out
  out=$(inspect) && fail "$label: inspection succeeded"
  [[ $(field "$out" state) != verified ]] || fail "$label: state verified"
  [[ $(field "$out" paths.index) == None ]] || fail "$label: path exposed on refusal"
  printf 'docs_offline_refused: %s -> %s\n' "$label" "$(field "$out" reason)"
}

pinned=4b9c4fb409b79d98e0c52e6a00384c7c8fa42bf8
if git -C "$repository_root" cat-file -e "$pinned:scripts/install-release.sh" 2>/dev/null; then
  git -C "$repository_root" show "$pinned:scripts/install-release.sh" >"$work/install-release.v1.sh"
  chmod 700 "$work/install-release.v1.sh"
  mkdir -m 700 "$work/v1bin" "$work/v1receipts"
  if refusal=$("$work/install-release.v1.sh" --archive "$archive" --checksums "$checksums" --bin-dir "$work/v1bin" --receipt-dir "$work/v1receipts" 2>&1); then
    fail 'pinned v1 installer accepted a v2 archive'
  fi
  [[ "$refusal" == *'release metadata schema mismatch'* && ! -e "$work/v1bin/axiom" ]] || fail 'pinned v1 installer did not refuse before any effect'
  printf 'docs_offline_refused: pinned v1 installer rejects v2 before execution\n'
else
  printf 'docs_offline_note: pinned v1 installer unavailable in this checkout (UNVERIFIED)\n'
fi

digest() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'; else shasum -a 256 "$1" | awk '{print $1}'; fi
}

# repack_refused extracts the archive, applies one mutation, re-seals the
# manifest so only the deeper documentary checks can catch it, and requires the
# installer to refuse before any effect.
repack_refused() {
  local label=$1 mutation=$2 expected=$3 dir="$work/repack-$1" bundle out
  mkdir -m 700 "$dir" "$dir/bin" "$dir/receipts"
  tar -xzf "$archive" -C "$dir"
  bundle=$(find "$dir" -mindepth 1 -maxdepth 1 -type d -not -name bin -not -name receipts)
  # Safe execution sentinel: documentary rejection must happen before even
  # the candidate's version probe, not merely before persistent publication.
  printf '#!/bin/sh\n: >"%s"\nexit 0\n' "$dir/candidate-executed" >"$bundle/axiom"
  sed "s/^binarySha256=.*/binarySha256=$(digest "$bundle/axiom")/" "$bundle/release-metadata.txt" >"$dir/metadata"
  mv "$dir/metadata" "$bundle/release-metadata.txt"
  (cd "$bundle" && eval "$mutation")
  (
    cd "$bundle"
    find . -type d -exec chmod 700 {} +
    find . -type f -exec chmod 600 {} +
    chmod 700 axiom install.sh
    find . -type f ! -name MANIFEST.sha256 -print | sed 's#^\./##' | LC_ALL=C sort | while IFS= read -r file; do
      printf '%s  %s\n' "$(digest "$file")" "$file"
    done >MANIFEST.sha256
  )
  COPYFILE_DISABLE=1 tar --format=ustar -C "$dir" -czf "$dir/$(basename "$bundle").tar.gz" "$(basename "$bundle")"
  printf '%s  %s\n' "$(digest "$dir/$(basename "$bundle").tar.gz")" "$(basename "$bundle").tar.gz" >"$dir/SHA256SUMS"
  if out=$("$repository_root/scripts/install-release.sh" --archive "$dir/$(basename "$bundle").tar.gz" --checksums "$dir/SHA256SUMS" --bin-dir "$dir/bin" --receipt-dir "$dir/receipts" 2>&1); then
    fail "$label: installer accepted a resealed bad archive"
  fi
  [[ "$out" == *"$expected"* && ! -e "$dir/bin/axiom" && ! -e "$dir/candidate-executed" ]] || fail "$label: unexpected refusal: $out"
  printf 'docs_offline_refused: %s -> %s\n' "$label" "$expected"
}

repack_refused extra-output 'printf "x\n" >docs/text/extra.md' 'bundle contents do not match documentary inventory'
repack_refused swapped-content 'printf "# changed\n" >docs/text/index.md' 'documentary inventory mismatch'
repack_refused extra-root-file 'printf "x\n" >notes.txt' 'bundle contents do not match documentary inventory'
repack_refused stale-metadata 'sed "s/^docsManMapSha256=.*/docsManMapSha256=$(printf "%064d" 0)/" release-metadata.txt >m && mv m release-metadata.txt' 'release metadata does not match bundle'

mkdir -m 700 "$work/copy"
cp -p "$work/bin/axiom" "$work/copy/axiom"
(cd / && env -i HOME="$work/home" PATH=/usr/bin:/bin "$work/copy/axiom" --json docs inspect >/dev/null) && fail 'copied binary inspected as verified'

chmod u+w "$index"
printf 'tampered\n' >>"$index"
expect_refused 'tampered content'

printf 'x' >>"$work/bin/axiom"
expect_refused 'modified binary'

printf 'docs_offline_success: %s\n' "$row"
printf 'docs_offline_qualification: PARTIAL (network/source denial and native Runtime observation NOT_RUN)\n'
