#!/usr/bin/env python3
"""Offline tests for scripts/verify-release-acceptance.py (Issue #238): the
publication-side verification of release-candidate acceptance Evidence.

    scripts/test-verify-release-acceptance.py
    scripts/test-verify-release-acceptance.py write-evidence --output ABS_FILE \\
      --row ROW --artifacts ABS_DIR --tag TAG --revision SHA --run ID [--status pass|fail]

write-evidence writes one valid candidate-acceptance axiom-gate-evidence/v1
document for a synthetic prepared set; scripts/test-release-flow.sh uses it to
stage the Evidence a prepared run retains. Nothing here touches the network.
"""

import argparse
import copy
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
HOSTS = {"linux-amd64": ("linux", "amd64", "ubuntu", "24.04"), "macos-27-arm64": ("darwin", "arm64", "macos", "27.0")}


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
    document["environment"].update(mode="native", architecture=architecture)
    document["environment"]["os"].update(family=family, distribution=distribution, version=version)
    for source in document["inputs"]["upgrade_sources"]:
        for item in source["artifacts"]:
            item["name"] = item["name"].replace("macos-27-arm64", row)
    command = document["inputs"]["command"]
    command.insert(command.index("{subject-sha256sums}") + 1, "--candidate-acceptance")
    if status == "fail":
        journey = document["journeys"][0]
        journey["result"] = "fail"
        journey["steps"][-1]["result"] = "fail"
        counts = document["result"]["counts"]
        counts["passed"] -= 1
        counts["failed"] += 1
        document["result"].update(status="fail", exit_code=1)
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
        for row in acceptance.ROWS:
            os.makedirs(os.path.join(self.acceptance, row))
            self.data[row] = write_evidence(self.evidence(row), row=row, artifacts=self.artifacts, tag=self.TAG,
                                            revision=self.REVISION, run=self.RUN)

    def evidence(self, row):
        return os.path.join(self.acceptance, row, "gate-evidence.json")

    def run_verifier(self, *, tag=None, revision=None, run=None, repo=REPOSITORY, acceptance_dir=None, artifacts=None):
        return subprocess.run([sys.executable, VERIFIER, "verify", "--acceptance", acceptance_dir or self.acceptance,
                               "--artifacts", artifacts or self.artifacts, "--tag", tag or self.TAG,
                               "--revision", revision or self.REVISION, "--repo", repo, "--prepared-run", run or self.RUN],
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
            f"acceptance.macos-27-arm64={sha256(self.data['macos-27-arm64'])}",
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
        os.mkdir(os.path.join(self.acceptance, "linux-arm64"))
        self.refused("holds an entry that is not a blocking row")
        os.rmdir(os.path.join(self.acceptance, "linux-arm64"))
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

    def test_identity_arguments_are_strict(self):
        self.refused("exact release tag required", tag="v1.2.3-beta.1")
        self.refused("full source revision required", revision="c" * 12)
        self.refused("numeric prepared run id required", run="12a")
        self.refused("repository must be OWNER/NAME", repo="axiom")


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
