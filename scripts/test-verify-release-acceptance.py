#!/usr/bin/env python3
"""Offline tests for scripts/verify-release-acceptance.py (Issue #238): the
publication-side verification of release-candidate acceptance Evidence of
every release row, automated and manual, and the manual Evidence transport.

    scripts/test-verify-release-acceptance.py
    scripts/test-verify-release-acceptance.py write-evidence --output ABS_FILE \\
      --row ROW --artifacts ABS_DIR --tag TAG --revision SHA --run ID [--status pass|fail]

write-evidence writes one valid candidate-acceptance axiom-gate-evidence/v1
document for a synthetic prepared set: for an automated row as the prepared
run's `accept` job retains it, for a manual row as a maintainer records it
(Linux arm64 native, Windows amd64 bounded proxy); scripts/test-release-flow.sh
uses it to stage Evidence. Nothing here touches the network.
"""

import argparse
import base64
import copy
import gzip
import hashlib
import importlib.util
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
import uuid

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
VERIFIER = os.path.join(ROOT, "scripts", "verify-release-acceptance.py")
TEMPLATE = os.path.join(ROOT, "scripts", "testdata", "gate-evidence", "pass-prepared.json")
WORKFLOW = os.path.join(ROOT, ".github", "workflows", "release-artifacts.yml")
REPOSITORY = "rgomids/axiom"

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("verify_release_acceptance", VERIFIER)
acceptance = importlib.util.module_from_spec(spec)
spec.loader.exec_module(acceptance)
gate = acceptance.gate
HOSTS = {"linux-amd64": ("linux", "amd64", "ubuntu", "24.04"), "macos-27-arm64": ("darwin", "arm64", "macos", "27.0"),
         "linux-arm64": ("linux", "arm64", "ubuntu", "24.04"), "windows-amd64": ("windows", "amd64", "windows", "2025")}
NOT_APPLICABLE = "Windows Server is the bounded proxy, never a supported client host"


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def subject_of(artifacts):
    with open(os.path.join(artifacts, "SHA256SUMS"), "rb") as handle:
        sums = handle.read()
    archives = {}
    for line in sums.decode().splitlines():
        digest, name = line.split("  ")
        archives[name] = digest
    return sha256(sums), archives


def proxy_journey(template):
    """The FR-072 bounded proxy: identity, provenance, direct CLI and the
    installers' Server refusal pass; install, upgrade and reinstall are
    not_applicable, and the installer reports no install outcome."""
    journey = copy.deepcopy(template)
    na = [{"id": step, "result": "not_applicable", "failure_category": None, "reason": NOT_APPLICABLE,
           "governing_reference": "FR-072", "duration_ms": None} for step in acceptance.PROXY_NOT_APPLICABLE_STEPS]
    ok = [{"id": step, "result": "pass", "failure_category": None, "reason": None, "governing_reference": None,
           "duration_ms": 1000} for step in acceptance.PROXY_PASS_STEPS]
    journey.update(id="windows-proxy", installer={"upgrade": None, "rerun": None},
                   classification={"before": None, "after": None}, steps=na + ok)
    return journey


def recount(document):
    steps = [step["result"] for journey in document["journeys"] for step in journey["steps"]]
    document["result"]["counts"] = {"journeys": len(document["journeys"]), "steps": len(steps),
                                    "passed": steps.count("pass"), "failed": steps.count("fail"),
                                    "not_applicable": steps.count("not_applicable")}


def evidence_document(*, row, artifacts, tag, revision, run, status="pass", repository=REPOSITORY, job="accept"):
    """A valid candidate-acceptance document about the prepared set at artifacts."""
    with open(TEMPLATE, encoding="utf-8") as handle:
        document = json.load(handle)
    sums, archives = subject_of(artifacts)
    family, architecture, distribution, version = HOSTS[row]
    document["gate"] = gate.ACCEPTANCE_GATE
    document["row"] = row
    document["subject"].update(kind="prepared", tag=tag, version=tag[1:], revision=revision, sha256sums_sha256=sums,
                               artifacts=[{"name": name, "sha256": digest} for name, digest in sorted(archives.items())])
    # Deterministic per input, so restaging the same run yields the same bytes.
    seed = hashlib.sha256(f"{row} {tag} {revision} {run} {sums} {status} {repository} {job}".encode()).digest()
    document["run"] = {"attempt_id": str(uuid.UUID(bytes=seed[:16], version=4)), "ci": {
        "provider": "github-actions", "repository": repository,
        "workflow_ref": f"{repository}/.github/workflows/release-artifacts.yml@refs/heads/main",
        "run_id": str(run), "run_attempt": 1, "job": job}}
    if row in acceptance.MANUAL_ROWS:
        # Recorded by a maintainer on the downloaded prepared set.
        document["run"]["ci"] = None
    document["environment"].update(mode="bounded_proxy" if row == "windows-amd64" else "native", architecture=architecture)
    document["environment"]["os"].update(family=family, distribution=distribution, version=version)
    for source in document["inputs"]["upgrade_sources"]:
        for item in source["artifacts"]:
            item["name"] = item["name"].replace("macos-27-arm64", row)
    command = document["inputs"]["command"]
    command.insert(command.index("{subject-sha256sums}") + 1, "--candidate-acceptance")
    if row == "windows-amd64":
        document["suite"] = acceptance.PROXY_SUITE
        document["journeys"] = [proxy_journey(document["journeys"][0])]
    if status == "fail":
        journey = document["journeys"][0]
        journey["result"] = "fail"
        journey["steps"][-1]["result"] = "fail"
        document["result"].update(status="fail", exit_code=1)
    recount(document)
    return document


def encode(document):
    return (json.dumps(document, indent=2, ensure_ascii=True) + "\n").encode()


def write_evidence(path, **arguments):
    data = encode(evidence_document(**arguments))
    gate.validate_bytes(data)
    with open(path, "xb") as handle:
        handle.write(data)
    return data


def make_set(directory, version, salt="first"):
    os.makedirs(directory)
    lines = []
    for row in ("linux-amd64", "linux-arm64", "macos-27-arm64", "windows-amd64"):
        name = f"axiom-{version}-{row}.tar.gz"
        data = f"{version} {row} {salt}\n".encode()
        with open(os.path.join(directory, name), "wb") as handle:
            handle.write(data)
        lines.append(f"{sha256(data)}  {name}\n")
    with open(os.path.join(directory, "SHA256SUMS"), "w", encoding="ascii") as handle:
        handle.write("".join(lines))


def inventory(*directories):
    found = {}
    for base in directories:
        for directory, subdirectories, files in os.walk(base):
            for name in subdirectories + files:
                path = os.path.join(directory, name)
                if os.path.isdir(path):
                    found[path] = None
                    continue
                with open(path, "rb") as handle:
                    found[path] = (os.lstat(path).st_mode, handle.read())
    return found


class Verifier(unittest.TestCase):
    TAG = "v1.2.3"
    REVISION = "c" * 40
    RUN = "4242"

    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = os.path.realpath(temporary.name)
        self.artifacts = os.path.join(self.root, "prepared", "artifacts")
        make_set(self.artifacts, self.TAG[1:])
        self.acceptance = os.path.join(self.root, "acceptance")
        self.data = {}
        for row in acceptance.RELEASE_ROWS:
            os.makedirs(os.path.join(self.acceptance, row))
            self.data[row] = write_evidence(self.evidence(row), row=row, artifacts=self.artifacts, tag=self.TAG,
                                            revision=self.REVISION, run=self.RUN)

    def evidence(self, row):
        return os.path.join(self.acceptance, row, "gate-evidence.json")

    def run_verifier(self, *, tag=None, revision=None, run=None, repo=REPOSITORY, acceptance_dir=None, artifacts=None,
                     extra=()):
        return subprocess.run([sys.executable, VERIFIER, "verify", "--acceptance", acceptance_dir or self.acceptance,
                               "--artifacts", artifacts or self.artifacts, "--tag", tag or self.TAG,
                               "--revision", revision or self.REVISION, "--repo", repo, "--prepared-run", run or self.RUN,
                               *extra],
                              capture_output=True, text=True, check=False)

    def replace(self, row, document=None, data=None):
        os.unlink(self.evidence(row))
        with open(self.evidence(row), "wb") as handle:
            handle.write(encode(document) if data is None else data)

    def document(self, row="linux-amd64", **overrides):
        arguments = dict(row=row, artifacts=self.artifacts, tag=self.TAG, revision=self.REVISION, run=self.RUN)
        arguments.update(overrides)
        return evidence_document(**arguments)

    def refused(self, pattern, **arguments):
        completed = self.run_verifier(**arguments)
        self.assertEqual(completed.returncode, 1, completed.stdout + completed.stderr)
        self.assertEqual(completed.stdout, "")
        self.assertRegex(completed.stderr, r"^release_acceptance_error: ")
        self.assertIn(pattern, completed.stderr)
        return completed

    # --- happy path -------------------------------------------------------

    def test_passing_evidence_of_every_row_is_bound_by_digest(self):
        before = inventory(self.artifacts, self.acceptance)
        completed = self.run_verifier()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        sums, _ = subject_of(self.artifacts)
        self.assertEqual(completed.stdout.splitlines(), [
            f"acceptance.linux-amd64={sha256(self.data['linux-amd64'])}",
            f"acceptance.linux-arm64={sha256(self.data['linux-arm64'])}",
            f"acceptance.macos-27-arm64={sha256(self.data['macos-27-arm64'])}",
            f"acceptance.windows-amd64={sha256(self.data['windows-amd64'])}",
            "acceptance_manual_transition=linux-arm64,windows-amd64",
            f"acceptance_sha256sums={sums}"])
        self.assertEqual(inventory(self.artifacts, self.acceptance), before, "verification never modifies its inputs")

    def test_binding_is_deterministic_and_follows_the_exact_evidence_bytes(self):
        first = self.run_verifier().stdout
        self.assertEqual(first, self.run_verifier().stdout)
        # Same meaning, different bytes: a different binding the envelope must re-authorize.
        self.replace("linux-amd64", data=self.data["linux-amd64"].rstrip(b"\n") + b"\n\n")
        second = self.run_verifier()
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertNotEqual(first, second.stdout)

    def test_blocking_rows_are_exactly_the_hosted_native_rows_of_the_workflow(self):
        completed = subprocess.run([sys.executable, VERIFIER, "rows"], capture_output=True, text=True, check=True)
        self.assertEqual(completed.stdout.splitlines(), ["linux-amd64", "macos-27-arm64"])
        completed = subprocess.run([sys.executable, VERIFIER, "manual-rows"], capture_output=True, text=True, check=True)
        self.assertEqual(completed.stdout.splitlines(), ["linux-arm64", "windows-amd64"])
        self.assertEqual(acceptance.RELEASE_ROWS, tuple(sorted(gate.Schema.load().root["properties"]["row"]["enum"])),
                         "every release row of the Evidence contract needs Evidence")
        with open(WORKFLOW, encoding="utf-8") as handle:
            workflow = handle.read()
        accept = workflow[workflow.index("\n  accept:\n"):workflow.index("\n  acceptance-evidence:\n")]
        self.assertEqual(re.findall(r"^            row: (\S+)$", accept, re.MULTILINE), ["linux-amd64", "macos-27-arm64"])
        self.assertEqual(re.findall(r"^          - gate: (\S+)$", accept, re.MULTILINE), ["linux-amd64", "macos-arm64"])

    # --- missing, failing, malformed --------------------------------------

    def test_missing_row_evidence_is_refused(self):
        shutil.rmtree(os.path.join(self.acceptance, "linux-amd64"))
        self.refused("missing acceptance Evidence for row linux-amd64")

    def test_empty_row_directory_is_refused(self):
        os.unlink(self.evidence("macos-27-arm64"))
        self.refused("must be exactly gate-evidence.json")

    def test_missing_acceptance_directory_is_refused(self):
        shutil.rmtree(self.acceptance)
        self.refused("cannot be read")

    def test_failing_evidence_is_refused(self):
        self.replace("macos-27-arm64", self.document("macos-27-arm64", status="fail"))
        self.refused("acceptance Evidence of row macos-27-arm64 does not pass")

    def test_malformed_evidence_is_refused(self):
        for data in (b"", b"{", b"\xff\xfe", b"[]", b'{"schema": "axiom-gate-evidence/v1", "schema": "x"}'):
            with self.subTest(data=data):
                self.replace("linux-amd64", data=data)
                self.refused("is not valid axiom-gate-evidence/v1")

    def test_schema_violations_are_refused(self):
        document = self.document()
        document["raw_log"] = "anything"
        self.replace("linux-amd64", document)
        self.refused("is not valid axiom-gate-evidence/v1")
        document = self.document()
        document["result"]["status"] = "pass"
        document["result"]["termination"] = "interrupted"
        self.replace("linux-amd64", document)
        self.refused("is not valid axiom-gate-evidence/v1")

    def test_oversized_evidence_is_refused(self):
        self.replace("linux-amd64", data=b" " * (gate.MAX_DOCUMENT_BYTES + 1))
        self.refused("exceeds its size limit")

    # --- other candidate ----------------------------------------------------

    def test_evidence_of_another_row_is_refused(self):
        self.replace("macos-27-arm64", data=self.data["linux-amd64"])
        self.refused("acceptance Evidence of row macos-27-arm64 names row linux-amd64")

    def test_evidence_of_another_tag_or_version_is_refused(self):
        document = self.document()
        document["subject"].update(tag="v1.2.4", version="1.2.4")
        self.replace("linux-amd64", document)
        self.refused("is about another tag or version")
        self.replace("linux-amd64", data=self.data["linux-amd64"])
        self.refused("SHA256SUMS names another version", tag="v1.2.3-rc.1")
        shutil.rmtree(self.artifacts)
        make_set(self.artifacts, "1.2.3-rc.1")
        self.refused("is about another tag or version", tag="v1.2.3-rc.1")

    def test_evidence_of_another_revision_is_refused(self):
        self.replace("linux-amd64", self.document(revision="d" * 40))
        self.refused("is about another revision")
        self.replace("linux-amd64", data=self.data["linux-amd64"])
        self.refused("is about another revision", revision="e" * 40)

    def test_divergent_sha256sums_is_refused(self):
        document = self.document()
        document["subject"]["sha256sums_sha256"] = "0" * 64
        self.replace("linux-amd64", document)
        self.refused("binds another SHA256SUMS")

    def test_divergent_artifact_digest_is_refused(self):
        document = self.document()
        document["subject"]["artifacts"][0]["sha256"] = "0" * 64
        self.replace("linux-amd64", document)
        self.refused("binds other artifact digests")
        document = self.document()
        document["subject"]["artifacts"].pop()
        self.replace("linux-amd64", document)
        self.refused("binds other artifact digests")

    def test_prepared_artifact_replaced_after_acceptance_is_refused(self):
        archive = os.path.join(self.artifacts, "axiom-1.2.3-linux-amd64.tar.gz")
        with open(archive, "ab") as handle:
            handle.write(b"substituted")
        self.refused("archive differs from SHA256SUMS: axiom-1.2.3-linux-amd64.tar.gz")
        # A self-consistent substituted set: SHA256SUMS rewritten to match.
        shutil.rmtree(self.artifacts)
        make_set(self.artifacts, "1.2.3", salt="rebuilt")
        self.refused("binds another SHA256SUMS")

    def test_rebuilt_candidate_evidence_is_refused(self):
        document = self.document()
        document["subject"].update(kind="rebuilt")
        document["inputs"]["command"] = ["scripts/test-upgrade-journeys.sh", "--candidate", "{candidate}",
                                         "--previous", "{previous-1}", "--previous", "{previous-2}",
                                         "--poc-binary", "{poc-binary}", "--evidence", "{evidence}"]
        self.replace("linux-amd64", document)
        self.refused("is not valid axiom-gate-evidence/v1")
        document["gate"] = "merge-regression"
        self.replace("linux-amd64", document)
        self.refused("is not candidate-acceptance Evidence")

    def test_merge_regression_evidence_of_the_prepared_set_is_refused(self):
        document = self.document()
        document["gate"] = "merge-regression"
        self.replace("linux-amd64", document)
        self.refused("is not valid axiom-gate-evidence/v1")
        document["inputs"]["command"].remove("--candidate-acceptance")
        self.replace("linux-amd64", document)
        self.refused("is not candidate-acceptance Evidence")

    def test_evidence_of_another_run_repository_workflow_or_job_is_refused(self):
        message = "was not produced by the acceptance job of prepared run 4242"
        self.replace("linux-amd64", self.document(run="4243"))
        self.refused(message)
        self.replace("linux-amd64", self.document(repository="fork/axiom"))
        self.refused(message)
        self.replace("linux-amd64", self.document(job="prepare"))
        self.refused(message)
        for workflow_ref in (f"{REPOSITORY}/.github/workflows/ci.yml@refs/heads/main",
                             f"{REPOSITORY}/.github/workflows/release-artifacts.yml@refs/heads/feature"):
            document = self.document()
            document["run"]["ci"]["workflow_ref"] = workflow_ref
            self.replace("linux-amd64", document)
            self.refused(message)
        document = self.document()
        document["run"]["ci"] = None
        self.replace("linux-amd64", document)
        self.refused(message)
        self.replace("linux-amd64", data=self.data["linux-amd64"])
        self.refused("was not produced by the acceptance job of prepared run 1", run="1")

    def test_evidence_not_observed_on_a_native_row_host_is_refused(self):
        for change in (dict(mode="bounded_proxy"), dict(architecture="arm64")):
            document = self.document()
            document["environment"].update(change)
            self.replace("linux-amd64", document)
            self.refused("was not observed on a native linux-amd64 host")

    def test_a_reduced_upgrade_matrix_is_refused(self):
        document = self.document()
        poc = next(journey for journey in document["journeys"] if journey["source"]["role"] == "historical_format")
        document["journeys"].remove(poc)
        counts = document["result"]["counts"]
        counts["journeys"] -= 1
        counts["steps"] -= len(poc["steps"])
        counts["passed"] -= len(poc["steps"])
        self.replace("linux-amd64", document)
        self.refused("does not cover the required upgrade matrix")

    def test_rows_sharing_one_attempt_are_refused(self):
        document = self.document("macos-27-arm64")
        document["run"]["attempt_id"] = json.loads(self.data["linux-amd64"])["run"]["attempt_id"]
        self.replace("macos-27-arm64", document)
        self.refused("shares one attempt")

    # --- closed inputs ------------------------------------------------------

    def test_incomplete_or_extended_candidate_is_refused(self):
        os.unlink(os.path.join(self.artifacts, "axiom-1.2.3-windows-amd64.tar.gz"))
        self.refused("artifact directory is not exactly the prepared set")
        shutil.rmtree(self.artifacts)
        make_set(self.artifacts, "1.2.3")
        with open(os.path.join(self.artifacts, "notes.txt"), "w", encoding="ascii") as handle:
            handle.write("extra\n")
        self.refused("artifact directory is not exactly the prepared set")

    def test_unexpected_acceptance_entries_are_refused(self):
        os.mkdir(os.path.join(self.acceptance, "windows-arm64"))
        self.refused("holds an unexpected entry")
        os.rmdir(os.path.join(self.acceptance, "windows-arm64"))
        with open(os.path.join(self.acceptance, "linux-amd64", "log.txt"), "w", encoding="ascii") as handle:
            handle.write("raw log\n")
        self.refused("must be exactly gate-evidence.json")

    def test_links_are_refused(self):
        outside = os.path.join(self.root, "outside.json")
        shutil.move(self.evidence("linux-amd64"), outside)
        os.symlink(outside, self.evidence("linux-amd64"))
        self.refused("cannot be read")
        os.unlink(self.evidence("linux-amd64"))
        shutil.move(outside, self.evidence("linux-amd64"))
        moved = os.path.join(self.root, "macos-row")
        shutil.move(os.path.join(self.acceptance, "macos-27-arm64"), moved)
        os.symlink(moved, os.path.join(self.acceptance, "macos-27-arm64"))
        self.refused("is not a directory")
        os.unlink(os.path.join(self.acceptance, "macos-27-arm64"))
        shutil.move(moved, os.path.join(self.acceptance, "macos-27-arm64"))
        linked = os.path.join(self.root, "linked-acceptance")
        os.symlink(self.acceptance, linked)
        self.refused("cannot be read", acceptance_dir=linked)
        self.refused("absolute canonical directory required", acceptance_dir=self.acceptance + "/../acceptance")
        archive = os.path.join(self.artifacts, "axiom-1.2.3-linux-amd64.tar.gz")
        shutil.move(archive, os.path.join(self.root, "archive"))
        os.symlink(os.path.join(self.root, "archive"), archive)
        self.refused("cannot be read")

    # --- manual-transition rows (CR-001) -----------------------------------

    def test_missing_manual_row_evidence_is_refused(self):
        for rows in (["linux-arm64"], ["windows-amd64"], ["linux-arm64", "windows-amd64"]):
            with self.subTest(rows=rows):
                for row in rows:
                    shutil.move(os.path.join(self.acceptance, row), os.path.join(self.root, row))
                self.refused(f"missing manual acceptance Evidence for row {', '.join(rows)}")
                for row in rows:
                    shutil.move(os.path.join(self.root, row), os.path.join(self.acceptance, row))
        self.assertEqual(self.run_verifier().returncode, 0)

    def test_automated_refusal_is_reported_before_pending_manual_evidence(self):
        for row in acceptance.MANUAL_ROWS:
            shutil.rmtree(os.path.join(self.acceptance, row))
        self.replace("linux-amd64", self.document(status="fail"))
        self.refused("acceptance Evidence of row linux-amd64 does not pass")
        shutil.rmtree(os.path.join(self.acceptance, "macos-27-arm64"))
        self.refused("missing acceptance Evidence for row macos-27-arm64")

    def test_automated_only_never_yields_an_envelope(self):
        for row in acceptance.MANUAL_ROWS:
            shutil.rmtree(os.path.join(self.acceptance, row))
        completed = self.run_verifier(extra=["--automated-only"])
        self.assertEqual(completed.returncode, 0, completed.stderr)
        lines = completed.stdout.splitlines()
        self.assertEqual([line.split("=")[0] for line in lines], [
            "acceptance.linux-amd64", "acceptance.macos-27-arm64", "acceptance_manual_pending", "acceptance_sha256sums"])
        self.assertNotIn("acceptance_manual_transition", completed.stdout)
        self.refused("missing manual acceptance Evidence for row linux-arm64, windows-amd64")

    def test_failing_manual_evidence_is_refused(self):
        for row in acceptance.MANUAL_ROWS:
            with self.subTest(row=row):
                self.replace(row, self.document(row, status="fail"))
                self.refused(f"acceptance Evidence of row {row} does not pass")
                self.replace(row, data=self.data[row])

    def test_malformed_manual_evidence_is_refused(self):
        for row in acceptance.MANUAL_ROWS:
            for data in (b"", b"{", b"[]", b"manual acceptance passed\n", b'{"result": "pass"}'):
                with self.subTest(row=row, data=data):
                    self.replace(row, data=data)
                    self.refused(f"acceptance Evidence of row {row} is not valid axiom-gate-evidence/v1")
            self.replace(row, data=self.data[row])

    def test_manual_evidence_of_another_row_is_refused(self):
        self.replace("windows-amd64", data=self.data["linux-arm64"])
        self.refused("acceptance Evidence of row windows-amd64 names row linux-arm64")
        self.replace("windows-amd64", data=self.data["windows-amd64"])
        self.replace("linux-arm64", data=self.data["linux-amd64"])
        self.refused("acceptance Evidence of row linux-arm64 names row linux-amd64")

    def test_manual_evidence_bound_to_another_candidate_is_refused(self):
        for row in acceptance.MANUAL_ROWS:
            with self.subTest(row=row):
                document = self.document(row)
                document["subject"].update(tag="v1.2.4", version="1.2.4")
                self.replace(row, document)
                self.refused(f"acceptance Evidence of row {row} is about another tag or version")
                self.replace(row, self.document(row, revision="d" * 40))
                self.refused(f"acceptance Evidence of row {row} is about another revision")
                document = self.document(row)
                document["subject"]["sha256sums_sha256"] = "0" * 64
                self.replace(row, document)
                self.refused(f"acceptance Evidence of row {row} binds another SHA256SUMS")
                document = self.document(row)
                document["subject"]["artifacts"][-1]["sha256"] = "0" * 64
                self.replace(row, document)
                self.refused(f"acceptance Evidence of row {row} binds other artifact digests")
                document = self.document(row)
                document["subject"].update(kind="rebuilt")
                self.replace(row, document)
                self.refused(f"acceptance Evidence of row {row} is not valid axiom-gate-evidence/v1")
                self.replace(row, data=self.data[row])
        # A whole other prepared candidate: valid Evidence of other bytes.
        other = os.path.join(self.root, "other", "artifacts")
        make_set(other, self.TAG[1:], salt="other")
        for row in acceptance.MANUAL_ROWS:
            self.replace(row, self.document(row, artifacts=other))
            self.refused(f"acceptance Evidence of row {row} binds another SHA256SUMS")
            self.replace(row, data=self.data[row])

    def test_manual_rows_hold_their_environment_contract(self):
        document = self.document("windows-amd64")
        document["environment"]["mode"] = "native"
        self.replace("windows-amd64", document)
        self.refused("acceptance Evidence of row windows-amd64 was not observed on a bounded proxy windows-amd64 host")
        document = self.document("windows-amd64")
        document["environment"]["os"]["family"] = "linux"
        self.replace("windows-amd64", document)
        self.refused("was not observed on a bounded proxy windows-amd64 host")
        self.replace("windows-amd64", data=self.data["windows-amd64"])
        for change in (dict(mode="bounded_proxy"), dict(architecture="amd64")):
            document = self.document("linux-arm64")
            document["environment"].update(change)
            self.replace("linux-arm64", document)
            self.refused("acceptance Evidence of row linux-arm64 was not observed on a native linux-arm64 host")

    def test_windows_proxy_never_claims_a_client_install(self):
        for field, outcome in (("upgrade", "upgraded"), ("upgrade", "installed"), ("rerun", "unchanged")):
            with self.subTest(field=field, outcome=outcome):
                document = self.document("windows-amd64")
                document["journeys"][0]["installer"][field] = outcome
                self.replace("windows-amd64", document)
                self.refused("acceptance Evidence of row windows-amd64 claims a Windows client install, upgrade or reinstall")

    def test_windows_proxy_holds_exactly_the_fr072_scope(self):
        claim = "acceptance Evidence of row windows-amd64 claims a Windows client install, upgrade or reinstall"
        # The reviewed attack: native upgrade-journeys Evidence relabelled as
        # the proxy, installer outcomes nulled.
        document = self.document("linux-arm64")
        document.update(row="windows-amd64", suite=acceptance.PROXY_SUITE)
        document["environment"].update(mode="bounded_proxy", architecture="amd64")
        document["environment"]["os"].update(family="windows", distribution="windows")
        for journey in document["journeys"]:
            journey["installer"] = {"upgrade": None, "rerun": None}
        for source in document["inputs"]["upgrade_sources"]:
            for item in source["artifacts"]:
                item["name"] = item["name"].replace("linux-arm64", "windows-amd64")
        self.replace("windows-amd64", document)
        self.refused(claim)
        document = self.document("windows-amd64")
        document["suite"] = gate.SUITE
        self.replace("windows-amd64", document)
        self.refused("is not the windows-bounded-proxy suite")
        for step, result in (("owned-upgrade", "pass"), ("reinstall", "pass")):
            document = self.document("windows-amd64")
            target = next(item for item in document["journeys"][0]["steps"] if item["id"] == step)
            target.update(result=result, reason=None, governing_reference=None, duration_ms=1000)
            recount(document)
            self.replace("windows-amd64", document)
            self.refused(claim)
        document = self.document("windows-amd64")
        document["journeys"][0]["steps"].append({"id": "upgrade", "result": "pass", "failure_category": None,
                                                  "reason": None, "governing_reference": None, "duration_ms": 1000})
        recount(document)
        self.replace("windows-amd64", document)
        self.refused(claim)
        document = self.document("windows-amd64")
        refusal = next(item for item in document["journeys"][0]["steps"] if item["id"] == "installer-server-refusal")
        refusal.update(result="not_applicable", reason=NOT_APPLICABLE, governing_reference="FR-072", duration_ms=None)
        recount(document)
        self.replace("windows-amd64", document)
        self.refused("does not cover the bounded proxy observations")
        document = self.document("windows-amd64")
        document["journeys"][0]["steps"] = [item for item in document["journeys"][0]["steps"] if item["id"] != "reinstall"]
        recount(document)
        self.replace("windows-amd64", document)
        self.refused(claim)

    def test_manual_evidence_never_claims_automated_or_foreign_provenance(self):
        ci = {"provider": "github-actions", "repository": REPOSITORY, "run_id": self.RUN, "run_attempt": 1,
              "workflow_ref": f"{REPOSITORY}/.github/workflows/release-artifacts.yml@refs/heads/main", "job": "accept"}
        for row in acceptance.MANUAL_ROWS:
            with self.subTest(row=row):
                for change in ({}, {"repository": "fork/axiom", "workflow_ref": "fork/axiom/.github/workflows/x.yml@refs/heads/main",
                                    "job": "manual"}):
                    document = self.document(row)
                    document["run"]["ci"] = {**ci, **change}
                    self.replace(row, document)
                    self.refused(f"acceptance Evidence of row {row} claims a provenance that manual Evidence cannot have")
                document = self.document(row)
                document["run"]["ci"] = {**ci, "workflow_ref": f"{REPOSITORY}/.github/workflows/manual-acceptance.yml@refs/heads/main",
                                         "job": "windows-proxy"}
                self.replace(row, document)
                self.assertEqual(self.run_verifier().returncode, 0, "a run of this repository may record manual Evidence")
                self.replace(row, data=self.data[row])

    def test_linux_arm64_needs_the_upgrade_matrix(self):
        document = self.document("linux-arm64")
        document["journeys"] = [journey for journey in document["journeys"] if journey["source"]["role"] != "earlier_release"]
        recount(document)
        self.replace("linux-arm64", document)
        self.refused("acceptance Evidence of row linux-arm64 does not cover the required upgrade matrix")

    def test_manual_evidence_reusing_another_attempt_is_refused(self):
        document = self.document("windows-amd64")
        document["run"]["attempt_id"] = json.loads(self.data["linux-arm64"])["run"]["attempt_id"]
        self.replace("windows-amd64", document)
        self.refused("shares one attempt")

    def test_manual_links_are_refused(self):
        outside = os.path.join(self.root, "outside.json")
        shutil.move(self.evidence("windows-amd64"), outside)
        os.symlink(outside, self.evidence("windows-amd64"))
        self.refused("cannot be read")
        os.unlink(self.evidence("windows-amd64"))
        moved = os.path.join(self.root, "arm-row")
        shutil.move(os.path.join(self.acceptance, "linux-arm64"), moved)
        os.symlink(moved, os.path.join(self.acceptance, "linux-arm64"))
        self.refused("is not a directory")

    def test_publication_envelope_requires_every_release_row(self):
        """The CR-001 regression: automated Evidence alone is never enough."""
        for row in acceptance.MANUAL_ROWS:
            shutil.rmtree(os.path.join(self.acceptance, row))
        completed = self.run_verifier()
        self.assertEqual(completed.returncode, 1)
        self.assertEqual(completed.stdout, "", "a refusal never prints a partial binding")

    def test_identity_arguments_are_strict(self):
        self.refused("exact release tag required", tag="v1.2.3-beta.1")
        self.refused("full source revision required", revision="c" * 12)
        self.refused("numeric prepared run id required", run="12a")
        self.refused("repository must be OWNER/NAME", repo="axiom")


class Transport(unittest.TestCase):
    """pack/unpack carry the manual Evidence bytes, never their meaning."""

    TAG = "v1.2.3"
    REVISION = "c" * 40

    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = os.path.realpath(temporary.name)
        self.artifacts = os.path.join(self.root, "artifacts")
        make_set(self.artifacts, self.TAG[1:])
        self.manual = os.path.join(self.root, "manual")
        self.data = {}
        for row in acceptance.MANUAL_ROWS:
            os.makedirs(os.path.join(self.manual, row))
            self.data[row] = write_evidence(os.path.join(self.manual, row, "gate-evidence.json"), row=row,
                                            artifacts=self.artifacts, tag=self.TAG, revision=self.REVISION, run="1")
        self.bundle = os.path.join(self.root, "bundle")
        self.into = os.path.join(self.root, "into")
        os.mkdir(self.into)

    def invoke(self, *arguments):
        return subprocess.run([sys.executable, VERIFIER, *arguments], capture_output=True, text=True, check=False)

    def pack(self, manual=None):
        completed = self.invoke("pack", "--manual", manual or self.manual)
        if completed.returncode == 0:
            with open(self.bundle, "w", encoding="ascii") as handle:
                handle.write(completed.stdout)
        return completed

    def write_line(self, line):
        with open(self.bundle, "w", encoding="ascii") as handle:
            handle.write(line)

    def refused(self, completed, pattern):
        self.assertEqual(completed.returncode, 1, completed.stdout + completed.stderr)
        self.assertEqual(completed.stdout, "")
        self.assertIn(pattern, completed.stderr)

    def unpack(self):
        return self.invoke("unpack", "--bundle", self.bundle, "--into", self.into)

    def framed(self, parts, *, prefix=acceptance.BUNDLE_PREFIX, raw=None):
        payload = raw if raw is not None else b"".join(f"{row} {len(data)}\n".encode() + data for row, data in parts)
        return prefix + base64.b64encode(gzip.compress(payload, mtime=0)).decode()

    def test_round_trip_is_byte_exact_and_deterministic(self):
        first = self.pack()
        self.assertEqual(first.returncode, 0, first.stderr)
        self.assertEqual(first.stdout.count("\n"), 1)
        self.assertTrue(first.stdout.startswith(acceptance.BUNDLE_PREFIX))
        self.assertLessEqual(len(first.stdout.strip()), acceptance.MAX_BUNDLE_CHARS)
        self.assertEqual(first.stdout, self.pack().stdout)
        completed = self.unpack()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertEqual(completed.stdout.splitlines(),
                         [f"manual_acceptance.{row}={sha256(self.data[row])}" for row in sorted(acceptance.MANUAL_ROWS)])
        for row in acceptance.MANUAL_ROWS:
            with open(os.path.join(self.into, row, "gate-evidence.json"), "rb") as handle:
                self.assertEqual(handle.read(), self.data[row])
            self.assertEqual(sorted(os.listdir(os.path.join(self.into, row))), ["gate-evidence.json"])
        # The dispatch input arrives without the trailing newline.
        shutil.rmtree(self.into)
        os.mkdir(self.into)
        self.write_line(first.stdout.strip())
        self.assertEqual(self.unpack().returncode, 0)

    def test_pack_needs_exactly_the_manual_rows(self):
        shutil.rmtree(os.path.join(self.manual, "windows-amd64"))
        self.refused(self.pack(), "missing manual acceptance Evidence for row windows-amd64")
        os.makedirs(os.path.join(self.manual, "windows-amd64"))
        self.refused(self.pack(), "must be exactly gate-evidence.json")
        shutil.rmtree(os.path.join(self.manual, "windows-amd64"))
        os.makedirs(os.path.join(self.manual, "linux-amd64"))
        self.refused(self.pack(), "holds an unexpected entry")

    def test_pack_refuses_a_line_the_dispatch_cannot_carry(self):
        for row in acceptance.MANUAL_ROWS:
            path = os.path.join(self.manual, row, "gate-evidence.json")
            os.unlink(path)
            with open(path, "wb") as handle:
                handle.write(os.urandom(64 * 1024))
        self.refused(self.pack(), "too large for the publication dispatch")

    def test_pack_refuses_links(self):
        target = os.path.join(self.manual, "windows-amd64", "gate-evidence.json")
        shutil.move(target, os.path.join(self.root, "outside.json"))
        os.symlink(os.path.join(self.root, "outside.json"), target)
        self.refused(self.pack(), "cannot be read")
        linked = os.path.join(self.root, "linked")
        os.symlink(self.manual, linked)
        self.refused(self.pack(linked), "cannot be read")

    def test_malformed_bundles_are_refused(self):
        arm, windows = self.data["linux-arm64"], self.data["windows-amd64"]
        cases = {
            "wrong prefix": self.framed([("linux-arm64", arm), ("windows-amd64", windows)], prefix="axiom-manual-acceptance-v2:"),
            "not base64": acceptance.BUNDLE_PREFIX + "!!!",
            "not gzip": acceptance.BUNDLE_PREFIX + "aGVsbG8=",
            "missing row": self.framed([("linux-arm64", arm)]),
            "reordered rows": self.framed([("windows-amd64", windows), ("linux-arm64", arm)]),
            "foreign row": self.framed([("linux-amd64", arm), ("windows-amd64", windows)]),
            "extra bytes": self.framed([("linux-arm64", arm), ("windows-amd64", windows), ("linux-amd64", arm)]),
            "truncated": self.framed([], raw=f"linux-arm64 {len(arm)}\n".encode() + arm[:-1]),
            "oversized frame": self.framed([], raw=f"linux-arm64 {gate.MAX_DOCUMENT_BYTES + 1}\n".encode()),
            "inflation bomb": self.framed([], raw=b"\0" * (3 * gate.MAX_DOCUMENT_BYTES)),
            "empty": "",
        }
        for name, line in cases.items():
            with self.subTest(name):
                self.write_line(line)
                self.refused(self.unpack(), "release_acceptance_error:")
                self.assertEqual(os.listdir(self.into), [], "a refused bundle writes nothing")
        self.write_line("x" * (acceptance.MAX_BUNDLE_CHARS + 2))
        self.refused(self.unpack(), "exceeds its size limit")

    def test_unpack_never_reuses_or_follows_existing_entries(self):
        self.pack()
        os.mkdir(os.path.join(self.into, "windows-amd64"))
        self.refused(self.unpack(), "already holds manual acceptance Evidence")
        self.assertEqual(os.listdir(self.into), ["windows-amd64"], "a refused unpack writes nothing")
        shutil.rmtree(self.into)
        os.symlink(self.manual, self.into)
        self.refused(self.unpack(), "cannot be read")
        os.unlink(self.into)
        os.mkdir(self.into)
        shutil.move(self.bundle, self.bundle + ".real")
        os.symlink(self.bundle + ".real", self.bundle)
        self.refused(self.unpack(), "cannot be read")
        self.refused(self.invoke("unpack", "--bundle", "bundle", "--into", self.into), "absolute bundle file required")

    def test_transport_is_untrusted_verify_decides(self):
        """A forged line unpacks, but verification still refuses it."""
        forged = copy.deepcopy(json.loads(self.data["windows-amd64"]))
        forged["result"]["status"] = "fail"
        self.write_line(self.framed([("linux-arm64", self.data["linux-arm64"]),
                                     ("windows-amd64", encode(forged))]))
        self.assertEqual(self.unpack().returncode, 0)
        for row in acceptance.ROWS:
            os.makedirs(os.path.join(self.into, row))
            write_evidence(os.path.join(self.into, row, "gate-evidence.json"), row=row, artifacts=self.artifacts,
                           tag=self.TAG, revision=self.REVISION, run="1")
        completed = self.invoke("verify", "--acceptance", self.into, "--artifacts", self.artifacts, "--tag", self.TAG,
                                "--revision", self.REVISION, "--repo", REPOSITORY, "--prepared-run", "1")
        self.refused(completed, "acceptance Evidence of row windows-amd64 is not valid axiom-gate-evidence/v1")


def write_evidence_command(argv):
    parser = argparse.ArgumentParser(prog="test-verify-release-acceptance.py write-evidence")
    for name in ("output", "row", "artifacts", "tag", "revision", "run"):
        parser.add_argument("--" + name, required=True)
    parser.add_argument("--status", default="pass", choices=["pass", "fail"])
    arguments = parser.parse_args(argv)
    data = write_evidence(arguments.output, row=arguments.row, artifacts=arguments.artifacts, tag=arguments.tag,
                          revision=arguments.revision, run=arguments.run, status=arguments.status)
    print(f"evidence_sha256={sha256(data)}")
    return 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "write-evidence":
        sys.exit(write_evidence_command(sys.argv[2:]))
    unittest.main()
