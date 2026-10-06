#!/usr/bin/env python3
"""Release-candidate acceptance Evidence of one prepared set (Issue #238;
Specification 004 FR-068/FR-069/FR-075, AC-50/AC-51/AC-56; ADR-0019).

    scripts/verify-release-acceptance.py rows
    scripts/verify-release-acceptance.py verify --acceptance ABS_DIR \\
      --artifacts ABS_DIR --tag vVERSION --revision FULL_SHA \\
      --repo OWNER/NAME --prepared-run RUN_ID

`rows` prints the release rows whose acceptance blocks publication, one per
line, each retained by release-artifacts.yml as the workflow artifact
`axiom-acceptance-<tag>-<row>` holding one `gate-evidence.json`.

`verify` is read-only and offline. --acceptance holds exactly
`<row>/gate-evidence.json` for every blocking row (no other entry, no link).
Each document is read once; those bytes are validated as
axiom-gate-evidence/v1 and must be passing candidate-acceptance Evidence of
the native row, whose prepared subject is exactly this tag, version, full
revision, SHA256SUMS digest and archive set, and which the `accept` job of
this repository's release-artifacts.yml dispatch from main wrote in run
--prepared-run. --artifacts must be exactly SHA256SUMS plus the archives it
lists, each matching its digest. Prints, for the publication envelope:

    acceptance.<row>=<SHA-256 of the exact Evidence bytes>   (sorted by row)
    acceptance_manual_transition=<rows still under the manual transition>
    acceptance_sha256sums=<SHA-256 of the SHA256SUMS the Evidence binds>

Any missing, failing, malformed, foreign or mismatched input prints
release_acceptance_error on stderr and exits 1. Exit 2 is usage.
"""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import os
import re
import stat
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
# Native rows with hosted release-blocking acceptance today (Issue #238).
# Linux arm64 (Slice 5) and the Windows bounded proxy (FR-072) join this set
# only through their own Slices; never by an absent row being skipped.
ROWS = {"linux-amd64": ("linux", "amd64"), "macos-27-arm64": ("darwin", "arm64")}
EVIDENCE = "gate-evidence.json"
JOB = "accept"
# The upgrade matrix every blocking row must have passed (FR-070.7, FR-074):
# N, an earlier release baseline and the declared historical format. The
# generation registry (Slice 6) replaces this minimum with declared baselines.
REQUIRED_ROLES = {"n", "earlier_release", "historical_format"}
# Rows that FR-069 covers but that have no automated acceptance yet; they stay
# under the Specification 004 manual transition and are named in the envelope.
MANUAL_ROWS = ("linux-arm64", "windows-amd64")
WORKFLOW = ".github/workflows/release-artifacts.yml@refs/heads/main"
TAG = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.(0|[1-9][0-9]*))?")
DIR_FLAGS = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW
FILE_FLAGS = os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK

sys.dont_write_bytecode = True
_spec = importlib.util.spec_from_file_location("gate_evidence", os.path.join(ROOT, "scripts", "gate-evidence.py"))
gate = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(gate)


class Refused(Exception):
    pass


def require(condition, message):
    if not condition:
        raise Refused(message)


def open_directory(path):
    """Opens an absolute directory without following a link at any component."""
    require(path.startswith("/") and os.path.normpath(path) == path and not path.startswith("//"),
            "absolute canonical directory required")
    descriptor = os.open("/", DIR_FLAGS)
    try:
        for component in path.split("/")[1:]:
            if component:
                child = os.open(component, DIR_FLAGS, dir_fd=descriptor)
                os.close(descriptor)
                descriptor = child
    except OSError:
        os.close(descriptor)
        raise
    return descriptor


def read_regular(directory, name, limit):
    """The bytes of one regular, non-link file, read once and bounded."""
    descriptor = os.open(name, FILE_FLAGS, dir_fd=directory)
    try:
        require(stat.S_ISREG(os.fstat(descriptor).st_mode), f"{name} is not a regular file")
        chunks, size = [], 0
        while True:
            block = os.read(descriptor, 1 << 20)
            if not block:
                break
            size += len(block)
            require(size <= limit, f"{name} exceeds its size limit")
            chunks.append(block)
        return b"".join(chunks)
    finally:
        os.close(descriptor)


def digest_regular(directory, name):
    descriptor = os.open(name, FILE_FLAGS, dir_fd=directory)
    try:
        require(stat.S_ISREG(os.fstat(descriptor).st_mode), f"{name} is not a regular file")
        digest = hashlib.sha256()
        for block in iter(lambda: os.read(descriptor, 1 << 20), b""):
            digest.update(block)
        return digest.hexdigest()
    finally:
        os.close(descriptor)


def prepared_set(path, version):
    """SHA256SUMS digest and archive digests of the closed prepared set."""
    directory = open_directory(path)
    try:
        sums = read_regular(directory, "SHA256SUMS", 64 * 1024)
        require(sums.endswith(b"\n"), "SHA256SUMS must end with a newline")
        archives = {}
        for line in sums.decode("ascii", "replace").split("\n")[:-1]:
            match = re.fullmatch(r"([0-9a-f]{64})  (axiom-[0-9A-Za-z.-]+\.tar\.gz)", line)
            require(match is not None, "SHA256SUMS line malformed")
            digest, name = match.groups()
            require(name.startswith(f"axiom-{version}-") and name not in archives, "SHA256SUMS names another version or repeats an archive")
            archives[name] = digest
        require(archives, "SHA256SUMS lists no archive")
        require(sorted(os.listdir(directory)) == sorted(["SHA256SUMS", *archives]), "artifact directory is not exactly the prepared set")
        for name, digest in archives.items():
            require(digest_regular(directory, name) == digest, f"archive differs from SHA256SUMS: {name}")
        return hashlib.sha256(sums).hexdigest(), archives
    finally:
        os.close(directory)


def evidence_bytes(path):
    """The exact Evidence bytes of every blocking row, from a closed layout."""
    directory = open_directory(path)
    found = {}
    try:
        entries = sorted(os.listdir(directory))
        missing = sorted(set(ROWS) - set(entries))
        require(not missing, f"missing acceptance Evidence for row {', '.join(missing)}")
        require(entries == sorted(ROWS), "acceptance directory holds an entry that is not a blocking row")
        for row in ROWS:
            try:
                child = os.open(row, DIR_FLAGS, dir_fd=directory)
            except OSError as error:
                raise Refused(f"acceptance Evidence of row {row} is not a directory") from error
            try:
                require(os.listdir(child) == [EVIDENCE], f"acceptance Evidence of row {row} must be exactly {EVIDENCE}")
                found[row] = read_regular(child, EVIDENCE, gate.MAX_DOCUMENT_BYTES)
            finally:
                os.close(child)
    finally:
        os.close(directory)
    return found


def check_document(row, data, expected):
    try:
        document = gate.validate_bytes(data)
    except (gate.Invalid, gate.SchemaError) as error:
        raise Refused(f"acceptance Evidence of row {row} is not valid axiom-gate-evidence/v1") from error
    where = f"acceptance Evidence of row {row}"
    require(document["gate"] == gate.ACCEPTANCE_GATE, f"{where} is not candidate-acceptance Evidence")
    require(document["suite"] == gate.SUITE, f"{where} is not the upgrade-journeys suite")
    require(document["row"] == row, f"{where} names row {document['row']}")
    result = document["result"]
    require(result["status"] == "pass" and result["termination"] == "completed" and result["exit_code"] == 0
            and result["counts"]["failed"] == 0, f"{where} does not pass")
    passed = {journey["source"]["role"] for journey in document["journeys"] if journey["result"] == "pass"}
    require(REQUIRED_ROLES <= passed, f"{where} does not cover the required upgrade matrix")
    subject = document["subject"]
    require(subject["kind"] == "prepared", f"{where} is about a {subject['kind']} candidate, not the prepared set")
    require(subject["tag"] == expected["tag"] and subject["version"] == expected["version"],
            f"{where} is about another tag or version")
    require(subject["revision"] == expected["revision"], f"{where} is about another revision")
    require(subject["sha256sums_sha256"] == expected["sha256sums"], f"{where} binds another SHA256SUMS")
    require({item["name"]: item["sha256"] for item in subject["artifacts"]} == expected["archives"]
            and len(subject["artifacts"]) == len(expected["archives"]), f"{where} binds other artifact digests")
    family, architecture = ROWS[row]
    environment = document["environment"]
    require(environment["mode"] == "native" and environment["os"]["family"] == family
            and environment["architecture"] == architecture, f"{where} was not observed on a native {row} host")
    ci = document["run"]["ci"]
    require(ci is not None and ci["repository"] == expected["repository"]
            and ci["workflow_ref"] == f"{expected['repository']}/{WORKFLOW}"
            and ci["run_id"] == expected["run"] and ci["job"] == JOB and ci["run_attempt"] is not None,
            f"{where} was not produced by the acceptance job of prepared run {expected['run']}")
    return document["run"]["attempt_id"]


def verify(arguments):
    require(TAG.fullmatch(arguments.tag) is not None, "exact release tag required")
    require(re.fullmatch(r"[0-9a-f]{40}", arguments.revision) is not None, "full source revision required")
    require(re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", arguments.repo) is not None, "repository must be OWNER/NAME")
    require(re.fullmatch(r"[0-9]{1,20}", arguments.prepared_run) is not None, "numeric prepared run id required")
    version = arguments.tag[1:]
    sums, archives = prepared_set(arguments.artifacts, version)
    expected = {"tag": arguments.tag, "version": version, "revision": arguments.revision, "sha256sums": sums,
                "archives": archives, "repository": arguments.repo, "run": arguments.prepared_run}
    documents = evidence_bytes(arguments.acceptance)
    attempts = [check_document(row, data, expected) for row, data in sorted(documents.items())]
    require(len(set(attempts)) == len(attempts), "acceptance Evidence of two rows shares one attempt")
    # Recheck the subject after reading the Evidence: the bytes bound now are
    # the bytes the Evidence names.
    require(prepared_set(arguments.artifacts, version) == (sums, archives), "prepared artifacts changed during verification")
    for row, data in sorted(documents.items()):
        print(f"acceptance.{row}={hashlib.sha256(data).hexdigest()}")
    print(f"acceptance_manual_transition={','.join(MANUAL_ROWS)}")
    print(f"acceptance_sha256sums={sums}")


def main(argv=None):
    parser = argparse.ArgumentParser(prog="verify-release-acceptance.py")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("rows")
    check = commands.add_parser("verify")
    for name in ("acceptance", "artifacts", "tag", "revision", "repo", "prepared-run"):
        check.add_argument("--" + name, required=True)
    arguments = parser.parse_args(sys.argv[1:] if argv is None else argv)
    if arguments.command == "rows":
        print("\n".join(ROWS))
        return 0
    try:
        verify(arguments)
    except Refused as error:
        print(f"release_acceptance_error: {error}", file=sys.stderr)
        return 1
    except (OSError, UnicodeDecodeError, KeyError, TypeError):
        # Paths and Evidence are untrusted input; never echo them.
        print("release_acceptance_error: acceptance Evidence or prepared artifacts cannot be read", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
