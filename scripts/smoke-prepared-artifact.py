#!/usr/bin/env python3
"""Bounded, offline execution of one exact prepared release row (Issue #244).

No build/toolchain, installer, provider or Runtime is invoked. JSON on stdout
is retained separately from the publishable inputs, including on failure.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import tarfile
import tempfile


def digest(path):
    with path.open("rb") as stream:
        hasher = hashlib.sha256()
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            hasher.update(chunk)
        return hasher.hexdigest()


def inventory(root):
    result = {}
    for path in sorted(root.rglob("*")):
        if path.is_symlink() or not (path.is_dir() or path.is_file()):
            raise ValueError("prepared inputs contain a link or special file")
        result[str(path.relative_to(root))] = digest(path) if path.is_file() else "directory"
    return result


def smoke(args, summary):
    if not re.fullmatch(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?", args.version):
        raise ValueError("exact semantic version required")
    if not re.fullmatch(r"[0-9a-f]{40}", args.revision):
        raise ValueError("full source revision required")
    host = (platform.system(), platform.machine())
    expected = {"linux-amd64": ("Linux", "x86_64"), "linux-arm64": ("Linux", "aarch64"),
                "macos-27-arm64": ("Darwin", "arm64")}
    if host != expected[args.row]:
        raise ValueError("native host does not match requested row")
    root = Path(args.directory)
    if not root.is_absolute() or root.is_symlink() or not root.is_dir():
        raise ValueError("absolute regular artifact directory required")
    root = root.resolve()
    before = inventory(root)
    bundle = f"axiom-{args.version}-{args.row}"
    archive = root / f"{bundle}.tar.gz"
    summary["archive"] = archive.name
    summary["archiveSha256"] = digest(archive)
    checksums = (root / "SHA256SUMS").read_text().splitlines()
    if checksums.count(f"{summary['archiveSha256']}  {archive.name}") != 1:
        raise ValueError("prepared archive checksum mismatch")
    try:
        with tempfile.TemporaryDirectory(prefix="axiom-release-smoke-") as temporary:
            work = Path(temporary).resolve()
            extracted = work / "extract"
            extracted.mkdir()
            with tarfile.open(archive, "r:gz") as source:
                members = source.getmembers()
                if len(members) > 256 or sum(member.size for member in members) > 128 * 1024 * 1024:
                    raise ValueError("prepared archive exceeds smoke bound")
                names = set()
                for member in members:
                    parts = member.name.split("/")
                    if (parts[0] != bundle or any(part in ("", ".", "..") for part in parts)
                            or "\\" in member.name or not (member.isfile() or member.isdir())
                            or member.name in names):
                        raise ValueError("unsafe prepared archive entry")
                    names.add(member.name)
                # Explicit validation above also works on hosted Python versions
                # predating tarfile's extraction filter API.
                source.extractall(extracted, members=members)
            binary = extracted / bundle / "axiom"
            if not binary.is_file() or not os.access(binary, os.X_OK):
                raise ValueError("prepared binary is not executable")
            summary["binarySha256"] = digest(binary)
            private = work / "private"
            private.mkdir()
            for name in ("home", "cwd", "repository", "bin", "tmp"):
                (private / name).mkdir()
            # Closed environment: no credentials, host Runtime discovery, local
            # Axiom roots or provider executable can leak into this lifecycle.
            environment = {
                "HOME": str(private / "home"), "USERPROFILE": str(private / "home"),
                "PATH": str(private / "bin"), "TMPDIR": str(private / "tmp"),
                "LINGO_PROJECTS_ROOT": str(private / "projects"),
                "LINGO_STATE_ROOT": str(private / "state"),
                "AXIOM_CODEX_SKILLS_ROOT": str(private / "skills"),
                "AXIOM_GH_BIN": str(private / "bin" / "unavailable-gh"),
                "LANG": "C", "LC_ALL": "C",
            }

            def run(*command):
                # File capture avoids unbounded memory and keeps raw CLI output
                # diagnostic-only; each subprocess has a fixed 20 second limit.
                with (private / "stdout").open("wb") as out, (private / "stderr").open("wb") as err:
                    completed = subprocess.run([str(binary), "--json", *command],
                                               cwd=private / "cwd", env=environment,
                                               stdin=subprocess.DEVNULL, stdout=out, stderr=err,
                                               timeout=20, check=False)
                if completed.returncode:
                    raise ValueError(f"CLI command failed: {' '.join(command[:2])}")
                if (private / "stdout").stat().st_size > 65536:
                    raise ValueError("CLI output exceeds smoke bound")
                event = json.loads((private / "stdout").read_text())
                if event.get("status") != "success":
                    raise ValueError("CLI did not report success")
                return event

            version = run("version")
            if version.get("provenance") != {
                    "product": "Axiom", "version": args.version,
                    "revision": args.revision[:12], "sourceState": "clean"}:
                raise ValueError("prepared binary version/revision mismatch")
            configuration = ("project", "configure", "--slug", "release-smoke", "--name", "Release Smoke",
                             "--repository", f"main={private / 'repository'}", "--work-item-provider", "none")
            preview = run(*configuration)
            intent = preview["setup"]
            run(*configuration, "--project-id", intent["projectId"],
                "--preview-digest", intent["digest"], "--authorize-local")
            shown = run("project", "show", "--selector", "release-smoke")
            if shown.get("project", {}).get("slug") != "release-smoke":
                raise ValueError("prepared binary failed to read published Project")
            listed = run("project", "list")
            if not any(project.get("slug") == "release-smoke" for project in listed.get("projects", [])):
                raise ValueError("prepared binary failed to list published Project")
            if not (private / "projects").is_dir() or not (private / "state").is_dir():
                raise ValueError("CLI lifecycle did not persist isolated state")
            if digest(binary) != summary["binarySha256"]:
                raise ValueError("extracted binary changed during smoke")
            summary["checks"] = ["version-revision", "project-configure-show-list", "inputs-unchanged"]
    finally:
        summary["inputsUnchanged"] = inventory(root) == before
        if not summary["inputsUnchanged"]:
            raise ValueError("prepared input digests changed during smoke")


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dir", dest="directory", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--revision", required=True)
    parser.add_argument("--row", choices=("linux-amd64", "linux-arm64", "macos-27-arm64"), required=True)
    args = parser.parse_args()
    summary = {
        "schema": "axiom-prepared-artifact-smoke/v1", "version": args.version,
        "revision": args.revision, "row": args.row,
        "environment": {"os": platform.system(), "release": platform.release(),
                        "architecture": platform.machine(), "platform": platform.platform(),
                        "runnerImage": os.getenv("ImageOS", "local"),
                        "runnerImageVersion": os.getenv("ImageVersion", "unknown")},
        "result": "fail",
    }
    try:
        smoke(args, summary)
        summary["result"] = "pass"
    except (ValueError, OSError, KeyError, TypeError, AttributeError, tarfile.TarError,
            subprocess.TimeoutExpired) as error:
        # No raw output, paths or inherited environment in retained Evidence.
        summary["error"] = str(error) if isinstance(error, ValueError) else type(error).__name__
    print(json.dumps(summary, sort_keys=True))
    return 0 if summary["result"] == "pass" else 1


if __name__ == "__main__":
    raise SystemExit(main())
