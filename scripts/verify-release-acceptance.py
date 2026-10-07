#!/usr/bin/env python3
"""Release-candidate acceptance Evidence of one prepared set (Issue #238;
Specification 004 FR-068/FR-069/FR-072/FR-075, AC-50/AC-51/AC-55/AC-56;
ADR-0019).

    scripts/verify-release-acceptance.py rows
    scripts/verify-release-acceptance.py manual-rows
    scripts/verify-release-acceptance.py verify --acceptance ABS_DIR \\
      --artifacts ABS_DIR --tag vVERSION --revision FULL_SHA \\
      --repo OWNER/NAME --prepared-run RUN_ID [--automated-only]
    scripts/verify-release-acceptance.py pack --manual ABS_DIR
    scripts/verify-release-acceptance.py unpack --bundle ABS_FILE --into ABS_DIR

A stable release has four release rows, and every one needs passing
acceptance Evidence before publication (FR-069). `rows` prints the automated
rows, whose Evidence release-artifacts.yml retains as the workflow artifact
`axiom-acceptance-<tag>-<row>` of the prepared run; `manual-rows` prints the
rows still under the Specification 004 manual transition, whose Evidence the
maintainer produces on the downloaded prepared set.

`verify` is read-only and offline. --acceptance holds exactly
`<row>/gate-evidence.json` for every release row (no other entry, no link).
Each document is read once; those bytes are validated as
axiom-gate-evidence/v1 and must be passing candidate-acceptance Evidence of
its row whose prepared subject is exactly this tag, version, full revision,
SHA256SUMS digest and archive set. An automated row must also be native and
written by the `accept` job of this repository's release-artifacts.yml
dispatch from main in run --prepared-run. A manual row's run.ci is null or a
run of this repository other than that accept job. Linux arm64 must be native
and cover the required upgrade matrix. Windows amd64 must be the FR-072
bounded proxy (suite windows-bounded-proxy): one journey whose only steps are
the passing artifact-identity, provenance, direct-cli and
installer-server-refusal and the FR-072 not_applicable fresh-install,
owned-upgrade and reinstall, with no installer outcome. --artifacts must
be exactly SHA256SUMS plus the archives it lists, each matching its digest.
Prints, for the publication envelope:

    acceptance.<row>=<SHA-256 of the exact Evidence bytes>   (every row, sorted)
    acceptance_manual_transition=<rows whose Evidence is manual>
    acceptance_sha256sums=<SHA-256 of the SHA256SUMS the Evidence binds>

--automated-only (release-artifacts.yml, inside the prepared run) checks only
the automated rows and prints `acceptance_manual_pending=` instead; publication
never uses it, so its output is never an envelope.

`pack` reads the manual rows' Evidence from the same closed layout and prints
it as one bounded line, the `manual_acceptance` input of publish-release.yml;
`unpack` writes that line back, byte for byte, as `<row>/gate-evidence.json`
of each manual row into an existing directory that holds none of them. The
envelope binds the Evidence digests, never the line, so the transport is not
trusted: `verify` decides.

Any missing, failing, malformed, foreign or mismatched input prints
release_acceptance_error on stderr and exits 1. Exit 2 is usage.
"""

import argparse
import base64
import binascii
import gzip
import hashlib
import importlib.util
import os
import re
import stat
import sys
import zlib

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
# Native rows with hosted release-blocking acceptance today (Issue #238).
ROWS = {"linux-amd64": ("linux", "amd64"), "macos-27-arm64": ("darwin", "arm64")}
# Rows that FR-069 covers but that have no automated acceptance yet: the
# Specification 004 manual transition. Their Evidence is still required and
# envelope-bound; only who produces it differs. Each joins ROWS only through
# its own Slice (Linux arm64: Slice 5; the FR-072 Windows bounded proxy).
MANUAL_ROWS = {"linux-arm64": ("linux", "arm64", "native"), "windows-amd64": ("windows", "amd64", "bounded_proxy")}
RELEASE_ROWS = tuple(sorted({*ROWS, *MANUAL_ROWS}))
EVIDENCE = "gate-evidence.json"
JOB = "accept"
# The upgrade matrix every native row must have passed (FR-070.7, FR-074):
# N, an earlier release baseline and the declared historical format. The
# generation registry (Slice 6) replaces this minimum with declared baselines.
# The Windows proxy is exempt: FR-072 makes its upgrade steps not_applicable.
REQUIRED_ROLES = {"n", "earlier_release", "historical_format"}
# FR-072/AC-55 bounded proxy Evidence: one journey holding exactly these
# steps. The proxy observations pass; install, owned upgrade and reinstall are
# not_applicable under FR-072. No other step, hence no install or upgrade
# claim, is accepted. The Windows proxy Slice's emitter must produce this shape.
PROXY_SUITE = "windows-bounded-proxy"
PROXY_PASS_STEPS = ("artifact-identity", "provenance", "direct-cli", "installer-server-refusal")
PROXY_NOT_APPLICABLE_STEPS = ("fresh-install", "owned-upgrade", "reinstall")
BUNDLE_PREFIX = "axiom-manual-acceptance-v1:"
# One workflow_dispatch payload holds at most 65535 characters in all.
MAX_BUNDLE_CHARS = 60000
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


def evidence_bytes(path, rows):
    """The exact Evidence bytes of the rows present in a closed layout.

    The layout may hold only the given rows, each as a directory holding
    exactly the Evidence document; which rows are missing is for the caller."""
    directory = open_directory(path)
    found = {}
    try:
        entries = sorted(os.listdir(directory))
        require(set(entries) <= set(rows), "acceptance directory holds an unexpected entry")
        for row in entries:
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
    require(document["row"] == row, f"{where} names row {document['row']}")
    result = document["result"]
    require(result["status"] == "pass" and result["termination"] == "completed" and result["exit_code"] == 0
            and result["counts"]["failed"] == 0, f"{where} does not pass")
    subject = document["subject"]
    require(subject["kind"] == "prepared", f"{where} is about a {subject['kind']} candidate, not the prepared set")
    require(subject["tag"] == expected["tag"] and subject["version"] == expected["version"],
            f"{where} is about another tag or version")
    require(subject["revision"] == expected["revision"], f"{where} is about another revision")
    require(subject["sha256sums_sha256"] == expected["sha256sums"], f"{where} binds another SHA256SUMS")
    require({item["name"]: item["sha256"] for item in subject["artifacts"]} == expected["archives"]
            and len(subject["artifacts"]) == len(expected["archives"]), f"{where} binds other artifact digests")
    environment = document["environment"]
    if row in ROWS:
        family, architecture = ROWS[row]
        mode = "native"
    else:
        family, architecture, mode = MANUAL_ROWS[row]
    require(environment["mode"] == mode and environment["os"]["family"] == family
            and environment["architecture"] == architecture,
            f"{where} was not observed on a {'native ' + row if mode == 'native' else 'bounded proxy ' + row} host")
    if mode == "native":
        require(document["suite"] == gate.SUITE, f"{where} is not the upgrade-journeys suite")
        passed = {journey["source"]["role"] for journey in document["journeys"] if journey["result"] == "pass"}
        require(REQUIRED_ROLES <= passed, f"{where} does not cover the required upgrade matrix")
    else:
        # FR-072/AC-55: the proxy is an Evidence environment, never a supported
        # client host. A reported install_status or any step outside the
        # closed proxy set is an install, upgrade or reinstall claim.
        require(document["suite"] == PROXY_SUITE, f"{where} is not the {PROXY_SUITE} suite")
        journeys = document["journeys"]
        require(all(journey["installer"]["upgrade"] is None and journey["installer"]["rerun"] is None
                    for journey in journeys), f"{where} claims a Windows client install, upgrade or reinstall")
        steps = {step["id"]: step for journey in journeys for step in journey["steps"]}
        require(len(journeys) == 1 and len(steps) == len(journeys[0]["steps"])
                and set(steps) == {*PROXY_PASS_STEPS, *PROXY_NOT_APPLICABLE_STEPS},
                f"{where} claims a Windows client install, upgrade or reinstall")
        require(all(steps[name]["result"] == "pass" for name in PROXY_PASS_STEPS),
                f"{where} does not cover the bounded proxy observations")
        require(all(steps[name]["result"] == "not_applicable" and steps[name]["governing_reference"] == "FR-072"
                    for name in PROXY_NOT_APPLICABLE_STEPS),
                f"{where} claims a Windows client install, upgrade or reinstall")
    ci = document["run"]["ci"]
    if row in MANUAL_ROWS:
        # Manual Evidence never claims this repository's automation or another
        # repository's: local (null) or a run of this repository that is not
        # the prepared run's accept job.
        require(ci is None or (ci["repository"] == expected["repository"]
                               and not (ci["workflow_ref"] == f"{expected['repository']}/{WORKFLOW}" and ci["job"] == JOB)),
                f"{where} claims a provenance that manual Evidence cannot have")
    if row in ROWS:
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
    rows = tuple(sorted(ROWS)) if arguments.automated_only else RELEASE_ROWS
    documents = evidence_bytes(arguments.acceptance, rows)
    # Automated rows first: their failure is a failed preparation, not a
    # pending manual acceptance.
    missing = sorted(set(ROWS) - set(documents))
    require(not missing, f"missing acceptance Evidence for row {', '.join(missing)}")
    attempts = [check_document(row, documents[row], expected) for row in sorted(ROWS)]
    if not arguments.automated_only:
        missing = sorted(set(MANUAL_ROWS) - set(documents))
        require(not missing, f"missing manual acceptance Evidence for row {', '.join(missing)}")
        attempts += [check_document(row, documents[row], expected) for row in sorted(MANUAL_ROWS)]
    require(len(set(attempts)) == len(attempts), "acceptance Evidence of two rows shares one attempt")
    # Recheck the subject after reading the Evidence: the bytes bound now are
    # the bytes the Evidence names.
    require(prepared_set(arguments.artifacts, version) == (sums, archives), "prepared artifacts changed during verification")
    for row in rows:
        print(f"acceptance.{row}={hashlib.sha256(documents[row]).hexdigest()}")
    if arguments.automated_only:
        print(f"acceptance_manual_pending={','.join(sorted(MANUAL_ROWS))}")
    else:
        print(f"acceptance_manual_transition={','.join(sorted(MANUAL_ROWS))}")
    print(f"acceptance_sha256sums={sums}")


def pack(arguments):
    """One bounded line carrying the exact manual Evidence bytes."""
    documents = evidence_bytes(arguments.manual, tuple(sorted(MANUAL_ROWS)))
    missing = sorted(set(MANUAL_ROWS) - set(documents))
    require(not missing, f"missing manual acceptance Evidence for row {', '.join(missing)}")
    framed = b"".join(f"{row} {len(documents[row])}\n".encode("ascii") + documents[row] for row in sorted(MANUAL_ROWS))
    line = BUNDLE_PREFIX + base64.b64encode(gzip.compress(framed, compresslevel=9, mtime=0)).decode("ascii")
    require(len(line) <= MAX_BUNDLE_CHARS, "manual acceptance Evidence is too large for the publication dispatch")
    print(line)


def unframe(line):
    """The Evidence bytes of every manual row, from a strictly parsed line."""
    require(line.startswith(BUNDLE_PREFIX), "manual acceptance bundle is malformed")
    try:
        compressed = base64.b64decode(line[len(BUNDLE_PREFIX):], validate=True)
    except (binascii.Error, ValueError) as error:
        raise Refused("manual acceptance bundle is malformed") from error
    limit = len(MANUAL_ROWS) * (gate.MAX_DOCUMENT_BYTES + 64)
    inflater = zlib.decompressobj(16 + zlib.MAX_WBITS)
    try:
        framed = inflater.decompress(compressed, limit + 1)
    except zlib.error as error:
        raise Refused("manual acceptance bundle is malformed") from error
    require(len(framed) <= limit and not inflater.unconsumed_tail, "manual acceptance bundle exceeds its size limit")
    require(inflater.eof and not inflater.unused_data, "manual acceptance bundle is malformed")
    documents, offset = {}, 0
    for row in sorted(MANUAL_ROWS):
        header = re.compile(rb"([a-z0-9-]+) (0|[1-9][0-9]{0,6})\n").match(framed, offset)
        require(header is not None and header.group(1).decode("ascii") == row,
                f"manual acceptance bundle does not carry row {row}")
        size = int(header.group(2))
        require(size <= gate.MAX_DOCUMENT_BYTES, f"acceptance Evidence of row {row} exceeds its size limit")
        start = header.end()
        require(start + size <= len(framed), "manual acceptance bundle is truncated")
        documents[row] = framed[start:start + size]
        offset = start + size
    require(offset == len(framed), "manual acceptance bundle carries more than the manual rows")
    return documents


def unpack(arguments):
    require(arguments.bundle.startswith("/"), "absolute bundle file required")
    directory = open_directory(os.path.dirname(arguments.bundle) or "/")
    try:
        data = read_regular(directory, os.path.basename(arguments.bundle), MAX_BUNDLE_CHARS + 1)
    finally:
        os.close(directory)
    line = data.decode("ascii")
    documents = unframe(line[:-1] if line.endswith("\n") else line)
    target = open_directory(arguments.into)
    try:
        require(not set(os.listdir(target)) & set(documents), "the target already holds manual acceptance Evidence")
        for row, evidence in documents.items():
            # mkdir and O_EXCL never reuse or follow an existing entry.
            os.mkdir(row, 0o700, dir_fd=target)
            child = os.open(row, DIR_FLAGS, dir_fd=target)
            try:
                handle = os.open(EVIDENCE, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600, dir_fd=child)
                try:
                    view = memoryview(evidence)
                    while view:
                        view = view[os.write(handle, view):]
                finally:
                    os.close(handle)
            finally:
                os.close(child)
    finally:
        os.close(target)
    for row, evidence in sorted(documents.items()):
        print(f"manual_acceptance.{row}={hashlib.sha256(evidence).hexdigest()}")


def main(argv=None):
    parser = argparse.ArgumentParser(prog="verify-release-acceptance.py")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("rows")
    commands.add_parser("manual-rows")
    check = commands.add_parser("verify")
    for name in ("acceptance", "artifacts", "tag", "revision", "repo", "prepared-run"):
        check.add_argument("--" + name, required=True)
    check.add_argument("--automated-only", action="store_true")
    packer = commands.add_parser("pack")
    packer.add_argument("--manual", required=True)
    unpacker = commands.add_parser("unpack")
    unpacker.add_argument("--bundle", required=True)
    unpacker.add_argument("--into", required=True)
    arguments = parser.parse_args(sys.argv[1:] if argv is None else argv)
    if arguments.command == "rows":
        print("\n".join(ROWS))
        return 0
    if arguments.command == "manual-rows":
        print("\n".join(sorted(MANUAL_ROWS)))
        return 0
    try:
        {"verify": verify, "pack": pack, "unpack": unpack}[arguments.command](arguments)
    except Refused as error:
        print(f"release_acceptance_error: {error}", file=sys.stderr)
        return 1
    except (OSError, UnicodeDecodeError, KeyError, TypeError):
        # Paths, bundles and Evidence are untrusted input; never echo them.
        print("release_acceptance_error: acceptance Evidence or prepared artifacts cannot be read", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
