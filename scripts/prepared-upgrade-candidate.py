#!/usr/bin/env python3
"""Pin and privately snapshot a complete prepared release set before upgrade use."""
import argparse
from contextlib import contextmanager
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import tarfile
import tempfile

ROWS = {"macos-27-arm64": ("macos-27", "darwin", "arm64"),
        "linux-amd64": ("linux", "linux", "amd64"),
        "linux-arm64": ("linux", "linux", "arm64"),
        "windows-amd64": ("windows", "windows", "amd64")}
VERSION = r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?"
IDENTITY_KEYS = {"format_version", "tag", "version", "revision", "row", "sha256sums_sha256",
                 "archives", "selected_archive", "artifacts_directory", "bundle_directory",
                 "binary", "metadata", "manifest_sha256", "verification_evidence_sha256"}
DIR_FLAGS = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW
FILE_FLAGS = os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK


class CandidateError(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise CandidateError(message)


def identity_object(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "identity contains duplicate keys")
        result[key] = value
    return result


@contextmanager
def open_directory(path):
    """Walk every absolute path component without following symbolic links."""
    path = os.fspath(path)
    require(path.startswith("/") and os.path.normpath(path) == path and not path.startswith("//"),
            "canonical absolute directory required")
    descriptor = os.open("/", DIR_FLAGS)
    try:
        for component in path.split("/")[1:]:
            if not component:
                continue
            child = os.open(component, DIR_FLAGS, dir_fd=descriptor)
            os.close(descriptor)
            descriptor = child
        yield descriptor
    finally:
        os.close(descriptor)


def private_directory(descriptor):
    info = os.fstat(descriptor)
    require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) & 0o077 == 0,
            "output directory must be owned by this user and private")


@contextmanager
def open_regular(directory, name, mode=None):
    descriptor = os.open(name, FILE_FLAGS, dir_fd=directory)
    try:
        info = os.fstat(descriptor)
        require(stat.S_ISREG(info.st_mode), "artifact is not a regular file")
        if mode is not None:
            require(stat.S_IMODE(info.st_mode) == mode, "materialized file mode changed")
        with os.fdopen(descriptor, "rb", closefd=False) as stream:
            yield stream
    finally:
        os.close(descriptor)


def digest_stream(stream):
    digest = hashlib.sha256()
    for block in iter(lambda: stream.read(1024 * 1024), b""):
        digest.update(block)
    return digest.hexdigest()


def read_regular(directory, name, mode=None):
    with open_regular(directory, name, mode) as stream:
        data = stream.read(1024 * 1024 + 1)
        require(len(data) <= 1024 * 1024, "metadata file exceeds size limit")
        return data


def hash_regular(directory, name, mode=None):
    with open_regular(directory, name, mode) as stream:
        return digest_stream(stream)


def archive_names(version):
    return {f"axiom-{version}-{platform}-{architecture}.tar.gz"
            for platform, _, architecture in ROWS.values()}


def parse_checksum_records(data):
    require(data.endswith(b"\n"), "SHA256SUMS must end with a newline")
    result = {}
    for line in data.split(b"\n")[:-1]:
        match = re.fullmatch(rb"([0-9a-f]{64})  (axiom-[0-9A-Za-z.-]+\.(?:tar\.gz|zip))", line)
        require(match is not None, "SHA256SUMS line malformed")
        digest, name = (value.decode("ascii") for value in match.groups())
        require(name not in result, "SHA256SUMS repeats an archive")
        result[name] = digest
    require(result, "SHA256SUMS is empty")
    return result


def parse_checksums(data, names):
    result = parse_checksum_records(data)
    require(len(result) == 4, "SHA256SUMS must have four lines")
    require(set(result) == names, "SHA256SUMS names do not match the closed set")
    return result


def snapshot_file(source, destination, name):
    with open_regular(source, name) as stream:
        before = os.fstat(stream.fileno())
        descriptor = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                             0o600, dir_fd=destination)
        try:
            with os.fdopen(descriptor, "wb", closefd=False) as target:
                for block in iter(lambda: stream.read(1024 * 1024), b""):
                    target.write(block)
                target.flush()
                os.fsync(descriptor)
            after = os.fstat(stream.fileno())
            require((before.st_dev, before.st_ino, before.st_size, before.st_mtime_ns, before.st_ctime_ns) ==
                    (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns),
                    "source artifact changed while being read")
            os.fchmod(descriptor, 0o400)
        finally:
            os.close(descriptor)


def metadata_bytes(identity):
    platform, goos, architecture = ROWS[identity["row"]]
    return (f"formatVersion=1\nproduct=Axiom\nversion={identity['version']}\n"
            f"revision={identity['revision'][:12]}\nsourceState=clean\nrelease=true\n"
            f"platform={platform}\ngoos={goos}\narchitecture={architecture}\nskillSetVersion=1\n").encode()


def selected_contents(artifacts, name):
    try:
        return archive_contents(artifacts, name)
    except (tarfile.TarError, EOFError) as error:
        raise CandidateError("archive cannot be read") from error


def archive_contents(artifacts, name):
    """Read validated archive files; never delegate archive extraction to the filesystem."""
    prefix = name.removesuffix(".tar.gz")
    files, directories = {}, {""}
    with open_regular(artifacts, name, 0o400) as stream, tarfile.open(fileobj=stream, mode="r:gz") as archive:
        seen = set()
        for member in archive.getmembers():
            path = member.name.rstrip("/")
            require(member.name == path or (member.isdir() and member.name == path + "/"), "unsafe archive entry")
            require(path not in seen, "duplicate archive entry")
            seen.add(path)
            require(path == prefix or path.startswith(prefix + "/"), "unsafe archive entry")
            relative = "" if path == prefix else path[len(prefix) + 1:]
            require(all(part not in ("", ".", "..") for part in relative.split("/")) or relative == "",
                    "unsafe archive entry")
            parts = relative.split("/")
            directories.update("/".join(parts[:index]) for index in range(1, len(parts)))
            if member.isdir():
                directories.add(relative)
                continue
            require(relative and member.type in (tarfile.REGTYPE, tarfile.AREGTYPE), "archive contains a link or special file")
            extracted = archive.extractfile(member)
            require(extracted is not None, "archive file unavailable")
            with extracted:
                files[relative] = (digest_stream(extracted), member)
        # The verifier is authoritative for the full release contract. This closed
        # extraction boundary independently rejects traversal, links and duplicates.
        require(not (set(files) & directories), "archive path is both a file and directory")
    return files, directories


def write_bundle(artifacts, bundle, name, preserve_executable=False):
    files, directories = selected_contents(artifacts, name)
    for directory in sorted(directories - {""}, key=lambda value: (value.count("/"), value)):
        os.mkdir(directory, 0o700, dir_fd=bundle)
    with open_regular(artifacts, name, 0o400) as stream, tarfile.open(fileobj=stream, mode="r:gz") as archive:
        for relative, (_, member) in files.items():
            descriptor = os.open(relative, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                                 0o600, dir_fd=bundle)
            try:
                with archive.extractfile(member) as source, os.fdopen(descriptor, "wb", closefd=False) as target:
                    for block in iter(lambda: source.read(1024 * 1024), b""):
                        target.write(block)
                    target.flush()
                    os.fsync(descriptor)
                executable = bool(member.mode & 0o111) if preserve_executable else relative in ("axiom", "axiom.exe")
                os.fchmod(descriptor, 0o500 if executable else 0o400)
            finally:
                os.close(descriptor)
    return files


def bundle_inventory(directory, prefix=""):
    files, directories = {}, {prefix}
    for name in os.listdir(directory):
        relative = f"{prefix}/{name}" if prefix else name
        info = os.stat(name, dir_fd=directory, follow_symlinks=False)
        if stat.S_ISDIR(info.st_mode):
            child = os.open(name, DIR_FLAGS, dir_fd=directory)
            try:
                private_directory(child)
                child_files, child_directories = bundle_inventory(child, relative)
                files.update(child_files)
                directories.update(child_directories)
            finally:
                os.close(child)
        else:
            mode = 0o500 if relative in ("axiom", "axiom.exe") else 0o400
            files[relative] = hash_regular(directory, name, mode)
    return files, directories


def validate_identity(identity, output, expected_row, expected_sums):
    require(isinstance(identity, dict) and set(identity) == IDENTITY_KEYS, "identity shape changed")
    require(identity["format_version"] == 1 and type(identity["format_version"]) is int, "identity version changed")
    require(isinstance(identity["version"], str) and re.fullmatch(VERSION, identity["version"]), "identity version malformed")
    require(identity["tag"] == "v" + identity["version"], "identity tag changed")
    require(isinstance(identity["revision"], str) and re.fullmatch(r"[0-9a-f]{40}", identity["revision"]), "identity revision malformed")
    require(isinstance(identity["row"], str) and identity["row"] in ROWS and
            identity["row"] == expected_row, "identity row changed")
    require(isinstance(expected_sums, str) and identity["sha256sums_sha256"] == expected_sums and
            re.fullmatch(r"[0-9a-f]{64}", expected_sums), "identity pin changed")
    names = archive_names(identity["version"])
    require(isinstance(identity["archives"], dict) and set(identity["archives"]) == names and
            all(isinstance(value, str) and re.fullmatch(r"[0-9a-f]{64}", value) for value in identity["archives"].values()),
            "identity archive set changed")
    platform, _, architecture = ROWS[identity["row"]]
    selected = f"axiom-{identity['version']}-{platform}-{architecture}.tar.gz"
    require(identity["selected_archive"] == {"name": selected, "sha256": identity["archives"][selected]}, "selected archive changed")
    require(identity["artifacts_directory"] == str(output / "artifacts") and
            identity["bundle_directory"] == str(output / "bundle"), "identity directories changed")
    binary = "axiom.exe" if identity["row"] == "windows-amd64" else "axiom"
    for key, filename in (("binary", binary), ("metadata", "release-metadata.txt")):
        value = identity[key]
        require(isinstance(value, dict) and set(value) == {"path", "sha256"} and
                value["path"] == str(output / "bundle" / filename) and
                isinstance(value["sha256"], str) and re.fullmatch(r"[0-9a-f]{64}", value["sha256"]), "identity file changed")
    for key in ("manifest_sha256", "verification_evidence_sha256"):
        require(isinstance(identity[key], str) and re.fullmatch(r"[0-9a-f]{64}", identity[key]), "identity digest malformed")
    return selected, binary


def verify_materialized(identity_path, expected_row, expected_sums):
    """Recheck the private snapshot against caller-held row and checksum identity."""
    identity_path = Path(identity_path)
    require(identity_path.name == "identity.json", "identity filename changed")
    output = identity_path.parent
    with open_directory(output) as root:
        private_directory(root)
        require(set(os.listdir(root)) == {"artifacts", "bundle", "identity.json"}, "output shape changed")
        identity = json.loads(read_regular(root, "identity.json", 0o400), object_pairs_hook=identity_object)
        selected, binary = validate_identity(identity, output, expected_row, expected_sums)
        with open_directory(output / "artifacts") as artifacts:
            private_directory(artifacts)
            require(set(os.listdir(artifacts)) == {"SHA256SUMS", *identity["archives"]}, "snapshot set changed")
            sums = read_regular(artifacts, "SHA256SUMS", 0o400)
            require(hashlib.sha256(sums).hexdigest() == expected_sums, "snapshot checksum pin changed")
            require(parse_checksums(sums, set(identity["archives"])) == identity["archives"], "snapshot checksums changed")
            for name, digest in identity["archives"].items():
                require(hash_regular(artifacts, name, 0o400) == digest, "snapshot archive changed")
            expected_files, expected_directories = selected_contents(artifacts, selected)
        with open_directory(output / "bundle") as bundle:
            private_directory(bundle)
            files, directories = bundle_inventory(bundle)
            require(files == {name: value[0] for name, value in expected_files.items()} and
                    directories == expected_directories, "materialized bundle changed")
            require({binary, "release-metadata.txt", "MANIFEST.sha256"} <= set(files), "materialized metadata missing")
            require(read_regular(bundle, "release-metadata.txt", 0o400) == metadata_bytes(identity), "materialized metadata changed")
            require(binary in files and files[binary] == identity["binary"]["sha256"] and
                    files["release-metadata.txt"] == identity["metadata"]["sha256"] and
                    files["MANIFEST.sha256"] == identity["manifest_sha256"], "identity file digest changed")
    return identity


def extract_source(args):
    """Snapshot and safely materialize one checksummed historical row archive."""
    require(args.row in ROWS, "unsupported row")
    output = Path(args.output)
    require(os.path.commonpath([str(output), args.extract_source]) not in (str(output), args.extract_source),
            "source and extraction output must be disjoint")
    with open_directory(args.extract_source) as source, open_directory(output) as root:
        private_directory(root)
        matches = [name for name in os.listdir(source)
                   if re.fullmatch(r"axiom-[0-9A-Za-z.-]+-" + re.escape(args.row) + r"\.tar\.gz", name)]
        require(len(matches) == 1, "source must contain exactly one archive for the row")
        name = matches[0]
        bundle_name = name.removesuffix(".tar.gz")
        require(bundle_name not in os.listdir(root), "source bundle already exists")
        with tempfile.TemporaryDirectory(prefix=".source-snapshot-", dir=output) as temporary:
            with open_directory(temporary) as snapshot:
                snapshot_file(source, snapshot, "SHA256SUMS")
                checksums = parse_checksum_records(read_regular(snapshot, "SHA256SUMS", 0o400))
                require(name in checksums, "source archive is absent from SHA256SUMS")
                snapshot_file(source, snapshot, name)
                require(hash_regular(snapshot, name, 0o400) == checksums[name], "source archive checksum mismatch")
                # Validate every entry before creating the destination bundle.
                selected_contents(snapshot, name)
                os.mkdir(bundle_name, 0o700, dir_fd=root)
                bundle = os.open(bundle_name, DIR_FLAGS, dir_fd=root)
                try:
                    write_bundle(snapshot, bundle, name, preserve_executable=True)
                finally:
                    os.close(bundle)
    return str(output / bundle_name)


def prepare(args):
    require(re.fullmatch("v" + VERSION, args.tag) is not None, "exact release tag required")
    require(re.fullmatch(r"[0-9a-f]{40}", args.revision) is not None, "full source revision required")
    require(re.fullmatch(r"[0-9a-f]{64}", args.sha256sums_sha256) is not None, "checksum identity pin required")
    require(args.row in ROWS, "unsupported row")
    version, output = args.tag[1:], Path(args.output)
    names = archive_names(version)
    with open_directory(args.source_root):
        pass
    with open_directory(args.prepared_set) as source, open_directory(output) as root:
        private_directory(root)
        require(not os.listdir(root), "output directory must be empty")
        require(set(os.listdir(source)) == {"SHA256SUMS", *names}, "prepared set is not closed")
        os.mkdir("artifacts", 0o700, dir_fd=root)
        os.mkdir("bundle", 0o700, dir_fd=root)
        with open_directory(output / "artifacts") as artifacts:
            snapshot_file(source, artifacts, "SHA256SUMS")
            sums = read_regular(artifacts, "SHA256SUMS", 0o400)
            require(hashlib.sha256(sums).hexdigest() == args.sha256sums_sha256, "prepared checksum identity mismatch")
            checksums = parse_checksums(sums, names)
            for name in sorted(names):
                snapshot_file(source, artifacts, name)
                require(hash_regular(artifacts, name, 0o400) == checksums[name], "prepared archive checksum mismatch")
    verifier = Path(__file__).resolve().with_name("verify-release-artifacts.sh")
    result = subprocess.run(["bash", str(verifier), "--dir", str(output / "artifacts"),
                             "--version", version, "--revision", args.revision,
                             "--source-root", args.source_root, "--skip-executable-smoke"],
                            check=True, capture_output=True, env=dict(os.environ, GOTOOLCHAIN="local"))
    require(b"result=structural_pass\n" in result.stdout and b"publication=none\n" in result.stdout,
            "release verifier did not confirm complete validation")
    platform, _, architecture = ROWS[args.row]
    selected = f"axiom-{version}-{platform}-{architecture}.tar.gz"
    binary = "axiom.exe" if args.row == "windows-amd64" else "axiom"
    with open_directory(output / "artifacts") as artifacts, open_directory(output / "bundle") as bundle:
        # Recheck all bytes after the external verifier and before materialization.
        require(hash_regular(artifacts, "SHA256SUMS", 0o400) == args.sha256sums_sha256, "snapshot checksum changed")
        for name, digest in checksums.items():
            require(hash_regular(artifacts, name, 0o400) == digest, "snapshot archive changed")
        files = write_bundle(artifacts, bundle, selected)
        identity = {"format_version": 1, "tag": args.tag, "version": version,
                    "revision": args.revision, "row": args.row,
                    "sha256sums_sha256": args.sha256sums_sha256, "archives": checksums,
                    "selected_archive": {"name": selected, "sha256": checksums[selected]},
                    "artifacts_directory": str(output / "artifacts"), "bundle_directory": str(output / "bundle"),
                    "binary": {"path": str(output / "bundle" / binary), "sha256": files[binary][0]},
                    "metadata": {"path": str(output / "bundle" / "release-metadata.txt"), "sha256": files["release-metadata.txt"][0]},
                    "manifest_sha256": files["MANIFEST.sha256"][0],
                    "verification_evidence_sha256": hashlib.sha256(result.stdout).hexdigest()}
    with open_directory(output) as root:
        descriptor = os.open("identity.json", os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o400, dir_fd=root)
        with os.fdopen(descriptor, "w") as stream:
            json.dump(identity, stream, sort_keys=True)
            stream.write("\n")
    return verify_materialized(output / "identity.json", args.row, args.sha256sums_sha256)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sources = parser.add_mutually_exclusive_group(required=True)
    sources.add_argument("--prepared-set")
    sources.add_argument("--extract-source")
    for name in ("output", "row"):
        parser.add_argument("--" + name, required=True)
    for name in ("tag", "revision", "sha256sums-sha256", "source-root"):
        parser.add_argument("--" + name)
    args = parser.parse_args()
    try:
        if args.extract_source:
            require(not any((args.tag, args.revision, args.sha256sums_sha256, args.source_root)), "prepared arguments are not source extraction arguments")
            print(extract_source(args))
            return 0
        require(all((args.tag, args.revision, args.sha256sums_sha256, args.source_root)), "prepared identity arguments required")
        identity = prepare(args)
    except (OSError, subprocess.CalledProcessError, tarfile.TarError, ValueError, KeyError, TypeError):
        # Candidate bytes and verifier output are untrusted; do not echo them.
        print("prepared_candidate_error: validation failed", file=__import__("sys").stderr)
        return 1
    print(json.dumps(identity, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
