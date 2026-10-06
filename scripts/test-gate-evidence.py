#!/usr/bin/env python3
"""Offline tests for axiom-gate-evidence/v1 (Issue #234): the closed schema,
scripts/gate-evidence.py validation and its upgrade-journey emitter, and the
--evidence argument contract of scripts/test-upgrade-journeys.sh."""

import copy
import importlib.util
import io
import json
import os
import subprocess
import sys
import tarfile
import tempfile
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCRIPT = os.path.join(ROOT, "scripts", "gate-evidence.py")
HARNESS = os.path.join(ROOT, "scripts", "test-upgrade-journeys.sh")
FIXTURES = os.path.join(ROOT, "scripts", "testdata", "gate-evidence")

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("gate_evidence", SCRIPT)
evidence = importlib.util.module_from_spec(spec)
spec.loader.exec_module(evidence)
SCHEMA = evidence.Schema.load()
SHA = "a" * 64
ATTEMPT = "3c06716f-6202-475b-9390-b20ca00e4877"


def fixture(name):
    with open(os.path.join(FIXTURES, name), encoding="utf-8") as handle:
        return json.load(handle)


def errors(document):
    """Every schema or cross-field error of document ([] when valid)."""
    found = SCHEMA.errors(document)
    return found or evidence.semantic_errors(document)


def not_applicable_step():
    return {"id": "windows-server-install", "result": "not_applicable", "failure_category": None,
            "reason": "Production installers refuse Windows Server hosts", "governing_reference": "FR-072",
            "duration_ms": None}


class Fixtures(unittest.TestCase):
    def test_representative_documents_are_valid(self):
        for name in ("pass.json", "fail-assertion.json", "fail-aborted.json", "fail-interrupted.json"):
            with self.subTest(name=name), open(os.path.join(FIXTURES, name), "rb") as handle:
                evidence.validate_bytes(handle.read(), SCHEMA)

    def test_fixtures_cover_each_outcome(self):
        self.assertEqual(fixture("pass.json")["result"]["status"], "pass")
        self.assertEqual(fixture("pass.json")["result"]["counts"]["failed"], 0)
        failed = fixture("fail-assertion.json")["result"]
        self.assertEqual((failed["status"], failed["termination"], failed["exit_code"], failed["failure_categories"]),
                         ("fail", "completed", 1, ["product"]))
        aborted = fixture("fail-aborted.json")
        self.assertEqual(aborted["result"]["termination"], "aborted")
        self.assertFalse(aborted["journeys"][-1]["completed"])
        interrupted = fixture("fail-interrupted.json")["result"]
        self.assertEqual((interrupted["termination"], interrupted["signal"], interrupted["exit_code"]),
                         ("interrupted", "TERM", 143))

    def test_fixtures_are_local_rebuilt_candidates(self):
        for name in ("pass.json", "fail-assertion.json"):
            document = fixture(name)
            self.assertIsNone(document["run"]["ci"])
            self.assertIsNone(document["environment"]["runner"])
            self.assertEqual(document["subject"]["kind"], "rebuilt")
            self.assertIsNone(document["subject"]["tag"])


class Schema(unittest.TestCase):
    def setUp(self):
        self.document = fixture("pass.json")

    def rejected(self, mutate, expected=None):
        document = copy.deepcopy(self.document)
        mutate(document)
        found = errors(document)
        self.assertTrue(found, "document was accepted")
        if expected:
            self.assertTrue(any(expected in error for error in found), found)

    def test_schema_identifier_is_required_and_exact(self):
        self.rejected(lambda d: d.pop("schema"), "'schema'")
        self.rejected(lambda d: d.update(schema="axiom-gate-evidence/v2"))

    def test_identity_fields_cannot_disappear(self):
        for path in (("gate",), ("suite",), ("row",), ("subject",), ("run",), ("environment",), ("inputs",),
                     ("subject", "kind"), ("subject", "version"), ("subject", "revision"),
                     ("subject", "sha256sums_sha256"), ("subject", "artifacts"), ("run", "attempt_id"),
                     ("environment", "os"), ("inputs", "upgrade_sources"), ("started_at",), ("finished_at",)):
            with self.subTest(path=path):
                def drop(d, path=path):
                    target = d
                    for key in path[:-1]:
                        target = target[key]
                    target.pop(path[-1])
                self.rejected(drop, "missing required property")

    def test_subject_needs_at_least_one_artifact(self):
        self.rejected(lambda d: d["subject"].update(artifacts=[]))

    def test_overall_result_is_required(self):
        self.rejected(lambda d: d.pop("result"), "'result'")
        self.rejected(lambda d: d["result"].pop("status"), "'status'")

    def test_invalid_enums_are_rejected(self):
        self.rejected(lambda d: d.update(row="freebsd-amd64"))
        self.rejected(lambda d: d["subject"].update(kind="candidate"))
        self.rejected(lambda d: d["result"].update(status="flaky"))
        self.rejected(lambda d: d["result"].update(termination="cancelled"))
        self.rejected(lambda d: d["environment"].update(mode="emulated"))
        self.rejected(lambda d: d["journeys"][0]["steps"][0].update(result="skipped"))
        self.rejected(lambda d: d["journeys"][0]["source"].update(role="n-1"))

    def test_invalid_failure_categories_are_rejected(self):
        failing = fixture("fail-assertion.json")
        for category in ("flaky", "network", "unknown", "Product", ""):
            with self.subTest(category=category):
                document = copy.deepcopy(failing)
                document["result"]["failure_categories"] = [category]
                self.assertTrue(errors(document))
                document = copy.deepcopy(failing)
                step = next(s for s in document["journeys"][0]["steps"] if s["result"] == "fail")
                step["failure_category"] = category
                self.assertTrue(errors(document))

    def test_every_research_category_is_accepted(self):
        for category in ("product", "test_defect", "infrastructure", "external_dependency", "timeout"):
            with self.subTest(category=category):
                document = fixture("fail-assertion.json")
                step = next(s for s in document["journeys"][0]["steps"] if s["result"] == "fail")
                step["failure_category"] = category
                document["result"]["failure_categories"] = [category]
                self.assertEqual(errors(document), [])

    def test_unobservable_failure_cause_stays_null_and_pass_forbids_a_category(self):
        document = fixture("fail-assertion.json")
        step = next(s for s in document["journeys"][0]["steps"] if s["result"] == "fail")
        step["failure_category"] = None
        document["result"]["failure_categories"] = []
        self.assertEqual(errors(document), [])
        self.rejected(lambda d: d["journeys"][0]["steps"][0].update(failure_category="product"))

    def test_a_completed_failure_needs_a_failed_step_or_category(self):
        self.rejected(lambda d: d["result"].update(status="fail", exit_code=1), "needs a failed step or a failure category")

    def test_malformed_digests_are_rejected(self):
        for value in ("A" * 64, "a" * 63, "a" * 65, "sha256:" + "a" * 64, "g" * 64, ""):
            with self.subTest(value=value):
                self.rejected(lambda d, v=value: d["subject"].update(sha256sums_sha256=v))
                self.rejected(lambda d, v=value: d["subject"]["artifacts"][0].update(sha256=v))
                self.rejected(lambda d, v=value: d["inputs"]["fixtures"][0].update(tree_sha256=v))
        self.rejected(lambda d: d["subject"].update(artifact_digest="a" * 64))
        self.rejected(lambda d: d["subject"].update(revision="F4819CCD178B"))

    def test_malformed_timestamps_are_rejected(self):
        for value in ("2026-10-06T15:17:36", "2026-10-06T15:17:36+00:00", "2026-10-06 15:17:36Z",
                      "2026-13-01T00:00:00Z", "2026-02-30T00:00:00Z", "1791299941", "2026-10-06T15:17:36Z\n"):
            with self.subTest(value=value):
                self.rejected(lambda d, v=value: d.update(started_at=v))
        self.rejected(lambda d: d.update(finished_at="2000-01-01T00:00:00Z"), "before started_at")
        self.rejected(lambda d: d["journeys"][0].update(started_at="2000-01-01T00:00:00Z"), "outside the run")

    def test_not_applicable_requires_reason_and_governing_reference(self):
        document = copy.deepcopy(self.document)
        document["journeys"][0]["steps"].append(not_applicable_step())
        document["result"]["counts"]["steps"] += 1
        document["result"]["counts"]["not_applicable"] += 1
        self.assertEqual(errors(document), [])
        for field, value in (("reason", None), ("governing_reference", None), ("reason", "n/a"),
                             ("governing_reference", "because"), ("duration_ms", 5)):
            with self.subTest(field=field, value=value):
                broken = copy.deepcopy(document)
                broken["journeys"][0]["steps"][-1][field] = value
                self.assertTrue(errors(broken))

    def test_not_applicable_is_never_counted_as_pass(self):
        document = copy.deepcopy(self.document)
        document["journeys"][0]["steps"].append(not_applicable_step())
        document["result"]["counts"]["steps"] += 1
        document["result"]["counts"]["passed"] += 1
        self.assertTrue(any("counts.passed" in error for error in errors(document)))

    def test_not_applicable_journey_contains_only_not_applicable_steps(self):
        document = copy.deepcopy(self.document)
        journey = document["journeys"][0]
        journey.update(result="not_applicable", reason="The bounded proxy installs nothing on this host",
                       governing_reference="FR-072")
        self.assertTrue(any("not_applicable journeys" in error for error in errors(document)))
        journey["reason"] = None
        self.assertTrue(errors(document))

    def test_raw_log_and_unbounded_fields_are_not_part_of_the_contract(self):
        for target, key in ((lambda d: d, "log"), (lambda d: d, "environment_variables"),
                            (lambda d: d["result"], "stdout"), (lambda d: d["journeys"][0], "output"),
                            (lambda d: d["journeys"][0]["steps"][0], "stderr"), (lambda d: d["environment"], "env"),
                            (lambda d: d["run"], "token"), (lambda d: d["inputs"], "arguments")):
            with self.subTest(key=key):
                self.rejected(lambda d, t=target, k=key: t(d).update({k: "x"}), "not part of the contract")

    def test_strings_are_bounded_and_single_line(self):
        self.rejected(lambda d: d["journeys"][0].update(id="v1\nstate"))
        self.rejected(lambda d: d["journeys"][0]["classification"].update(after="valid_v1\n"))
        # Defence in depth behind the patterns: no string may carry a control character.
        document = copy.deepcopy(self.document)
        document["journeys"][0]["classification"]["after"] = "valid_v1\r"
        self.assertTrue(any("control character" in error for error in evidence.semantic_errors(document)))
        self.rejected(lambda d: d["environment"].update(go_version="go1.26.1\tlocal"))
        reason = copy.deepcopy(self.document)
        reason["journeys"][0]["steps"].append(not_applicable_step())
        reason["journeys"][0]["steps"][-1]["reason"] = "x" * 241
        self.assertTrue(errors(reason))

    def test_command_never_records_raw_paths_or_values(self):
        for value in ("/home/runner/work/_temp/candidate", "--candidate=/tmp/x", "GITHUB_TOKEN=abc", "{candidate} x"):
            with self.subTest(value=value):
                self.rejected(lambda d, v=value: d["inputs"]["command"].append(v))
        self.rejected(lambda d: d["inputs"]["fixtures"][0].update(path="/abs/path"))

    def test_vacuous_pass_is_rejected(self):
        self.rejected(lambda d: (d.update(journeys=[]), d["result"]["counts"].update(journeys=0, steps=0, passed=0)),
                      "at least one passing step")
        document = copy.deepcopy(self.document)
        journey = document["journeys"][0]
        journey.update(result="not_applicable", reason="The bounded proxy installs nothing on this host",
                       governing_reference="FR-072", steps=[not_applicable_step()])
        document["journeys"] = [journey]
        document["result"]["counts"] = {"journeys": 1, "steps": 1, "passed": 0, "failed": 0, "not_applicable": 1}
        self.assertTrue(any("at least one passing step" in error for error in errors(document)))

    def test_patterns_anchor_at_end_of_input(self):
        self.assertFalse(SCHEMA.accepts(SHA + "\n", SCHEMA.definition("sha256")))
        self.assertFalse(SCHEMA.accepts("ubuntu24\n", SCHEMA.definition("nullable_fact")))

    def test_journey_sources_and_command_placeholders_are_bound(self):
        self.rejected(lambda d: d["journeys"][0]["source"].update(path=["previous-1", "previous-1"]))
        self.rejected(lambda d: d["inputs"]["command"].extend(["--previous", "{previous-9}"]), "bound to nothing")

    def test_subject_carries_the_row_archive(self):
        self.rejected(lambda d: d["subject"]["artifacts"][0].update(name="axiom-1.0.0-linux-amd64.tar.gz"),
                      "no archive of the row")

    def test_pass_binds_every_used_upgrade_source(self):
        used = next(index for index, source in enumerate(self.document["inputs"]["upgrade_sources"])
                    if source["kind"] == "release")
        poc = next(index for index, source in enumerate(self.document["inputs"]["upgrade_sources"])
                   if source["kind"] == "historical_build")
        wrong_row = {"name": "axiom-0.5.0-linux-amd64.tar.gz", "sha256": SHA}
        for label, mutate, expected in (
                ("sums", lambda d: d["inputs"]["upgrade_sources"][used].update(sha256sums_sha256=None), "SHA256SUMS digest"),
                ("artifacts", lambda d: d["inputs"]["upgrade_sources"][used].update(artifacts=[]), "row archive digest"),
                ("other row", lambda d: d["inputs"]["upgrade_sources"][used].update(artifacts=[wrong_row]), "row archive digest"),
                ("historical", lambda d: d["inputs"]["upgrade_sources"][poc].update(artifacts=[]), "executed binary")):
            with self.subTest(label=label):
                self.rejected(mutate, expected)
        self.rejected(lambda d: d["inputs"]["upgrade_sources"][used]["artifacts"][0].update(sha256="A" * 64))

    def test_unused_or_failed_sources_may_stay_partial(self):
        document = copy.deepcopy(self.document)
        document["inputs"]["upgrade_sources"].append({"id": "previous-9", "kind": "release", "version": None, "tag": None,
                                                      "sha256sums_sha256": None, "artifacts": []})
        self.assertEqual(errors(document), [])
        failing = fixture("fail-assertion.json")
        failing["inputs"]["upgrade_sources"][0].update(sha256sums_sha256=None, artifacts=[])
        self.assertEqual(errors(failing), [])

    def test_bounded_collections(self):
        self.rejected(lambda d: d["journeys"][0]["steps"].extend(copy.deepcopy(d["journeys"][0]["steps"][0])
                                                                 for _ in range(64)))
        self.rejected(lambda d: d["subject"].update(artifacts=[{"name": f"a{i}", "sha256": SHA} for i in range(17)]))

    def test_document_size_duplicates_and_constants_are_rejected(self):
        data = json.dumps(self.document).encode()
        with self.assertRaises(evidence.Invalid):
            evidence.load_document(data[:-1] + b" " * evidence.MAX_DOCUMENT_BYTES + b"}")
        with self.assertRaises(evidence.Invalid):
            evidence.load_document(b'{"schema": "axiom-gate-evidence/v1", "schema": "x"}')
        with self.assertRaises(evidence.Invalid):
            evidence.load_document(b'{"exit_code": NaN}')

    def test_local_execution_needs_no_github_actions_identity(self):
        self.assertIsNone(self.document["run"]["ci"])
        self.assertEqual(errors(self.document), [])

    def test_github_actions_identity_is_optional_and_closed(self):
        document = copy.deepcopy(self.document)
        document["run"]["ci"] = {"provider": "github-actions", "repository": "rgomids/axiom",
                                 "workflow_ref": "rgomids/axiom/.github/workflows/ci.yml@refs/pull/240/merge",
                                 "run_id": "123456789", "run_attempt": 2, "job": "upgrade-journeys"}
        document["environment"]["runner"] = {"environment": "github-hosted", "label": "ubuntu-24.04",
                                             "image_os": "ubuntu24", "image_version": "20260928.1.0"}
        self.assertEqual(errors(document), [])
        for change in ({"run_attempt": 0}, {"run_attempt": "2"}, {"provider": "gitlab"}, {"secret": "x"},
                       {"workflow_ref": "https://example.test/ci.yml"}):
            with self.subTest(change=change):
                broken = copy.deepcopy(document)
                broken["run"]["ci"].update(change)
                self.assertTrue(errors(broken))

    def test_prepared_and_published_subjects_require_a_tag(self):
        for kind in ("prepared", "published"):
            with self.subTest(kind=kind):
                self.rejected(lambda d, k=kind: d["subject"].update(kind=k))
                document = copy.deepcopy(self.document)
                document["subject"].update(kind=kind, tag="v1.2.3", version="1.2.3", artifact_id="4242",
                                           artifact_digest="sha256:" + SHA)
                self.assertEqual(errors(document), [])

    def test_result_consistency(self):
        self.rejected(lambda d: d["result"]["counts"].update(steps=1), "counts.steps")
        self.rejected(lambda d: d["result"].update(failure_categories=["product"]))
        self.rejected(lambda d: d["result"].update(termination="aborted"))
        self.rejected(lambda d: d["result"].update(exit_code=1))
        self.rejected(lambda d: d["result"].update(signal="TERM"))
        self.rejected(lambda d: d["journeys"][0].update(completed=False, finished_at=None))
        self.rejected(lambda d: d["journeys"][0]["steps"][0].update(result="fail", failure_category="product"),
                      "pass needs")
        self.rejected(lambda d: d["journeys"][0]["source"].update(path=["previous-9"]), "unknown upgrade source")
        self.rejected(lambda d: d["journeys"].append(copy.deepcopy(d["journeys"][0])), "not unique")

    def test_interrupted_runs_name_their_signal(self):
        document = fixture("fail-interrupted.json")
        document["result"]["signal"] = None
        self.assertTrue(errors(document))
        document = fixture("fail-aborted.json")
        document["result"]["signal"] = "INT"
        self.assertTrue(errors(document))

    def test_a_deterministic_failure_cannot_become_a_pass(self):
        document = fixture("fail-assertion.json")
        document["result"]["status"] = "pass"
        self.assertTrue(errors(document))
        document = fixture("fail-assertion.json")
        document["result"].update(failure_categories=["infrastructure"])
        self.assertTrue(any("missing ['product']" in error for error in errors(document)))

    def test_every_object_is_closed(self):
        def walk(schema, where):
            if isinstance(schema, dict):
                if schema.get("type") == "object" or "properties" in schema and "if" not in schema:
                    if "required" in schema:
                        self.assertIs(schema.get("additionalProperties"), False, where)
                        self.assertEqual(set(schema["required"]), set(schema["properties"]), where)
                for key, value in schema.items():
                    walk(value, f"{where}/{key}")
            elif isinstance(schema, list):
                for index, value in enumerate(schema):
                    walk(value, f"{where}/{index}")
        walk(SCHEMA.root, "#")

    def test_unsupported_schema_keywords_fail_closed(self):
        for extra in ({"format": "date-time"}, {"oneOf": []}, {"additionalProperties": True},
                      {"$ref": "#/$defs/missing"}, {"type": "number"}):
            with self.subTest(extra=extra):
                with self.assertRaises(evidence.SchemaError):
                    evidence.Schema({"type": "object", **extra, "$defs": {}})


def archive(directory, name, metadata):
    path = os.path.join(directory, name)
    data = "".join(f"{key}={value}\n" for key, value in metadata.items()).encode()
    with tarfile.open(path, "w:gz") as bundle:
        info = tarfile.TarInfo(f"{name[:-len('.tar.gz')]}/release-metadata.txt")
        info.size = len(data)
        bundle.addfile(info, io.BytesIO(data))
    return path


class Emitter(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.root = os.path.realpath(self.temporary.name)
        self.candidate = self.directory("candidate", "9999.0.0-acceptance.7", "0123456789ab")
        self.previous = [self.directory("previous-a", "0.5.0", "aaaaaaaaaaaa"),
                         self.directory("previous-b", "0.1.1", "bbbbbbbbbbbb")]
        self.poc = os.path.join(self.root, "poc-lingo")
        with open(self.poc, "wb") as handle:
            handle.write(b"historical build")
        self.repository = os.path.join(self.root, "repository")
        corpus = os.path.join(self.repository, "testdata", "corpus", "state")
        os.makedirs(corpus)
        with open(os.path.join(corpus, "record.json"), "w", encoding="utf-8") as handle:
            handle.write("{}\n")
        self.output_directory = os.path.join(self.root, "evidence")
        os.mkdir(self.output_directory)
        self.output = os.path.join(self.output_directory, "gate-evidence.json")
        self.records = os.path.join(self.root, "records.tsv")
        self.snapshot = self.inventory()

    def tearDown(self):
        self.temporary.cleanup()

    def directory(self, name, version, revision):
        path = os.path.join(self.root, name)
        os.mkdir(path)
        name = f"axiom-{version}-linux-amd64.tar.gz"
        digest = evidence.sha256_file(archive(path, name, {"version": version, "revision": revision}))
        with open(os.path.join(path, "SHA256SUMS"), "w", encoding="utf-8") as handle:
            handle.write(f"{digest}  {name}\n")
        return path

    def inventory(self):
        found = {}
        for base in [self.candidate, *self.previous, self.repository]:
            for directory, _, files in os.walk(base):
                for name in files:
                    path = os.path.join(directory, name)
                    found[path] = evidence.sha256_file(path)
        found[self.poc] = evidence.sha256_file(self.poc)
        return found

    def write_records(self, lines, skip=()):
        """Numbered like the harness; records whose number is in skip are lost."""
        self.record_count = len(lines)
        with open(self.records, "w", encoding="utf-8") as handle:
            handle.write("".join("\t".join((str(number), *fields)) + "\n"
                                 for number, fields in enumerate(lines, start=1) if number not in skip))

    def emit(self, *, exit_code=0, termination="completed", signal="", output=None, env=None, extra=(),
             attempt=ATTEMPT, record_count=None):
        command = [sys.executable, SCRIPT, "upgrade-journeys", "--records", self.records,
                   "--output", output or self.output, "--row", "linux-amd64", "--candidate", self.candidate,
                   "--previous", self.previous[0], "--previous", self.previous[1], "--poc-binary", self.poc,
                   "--fixture", "corpus=testdata/corpus", "--started-at", "2026-10-06T10:00:00Z",
                   "--finished-at", "2026-10-06T10:05:00Z", "--exit-code", str(exit_code),
                   "--termination", termination, "--bash-version", "5.2.37(1)-release",
                   "--clock-resolution-ms", "1", "--work-dir", self.root, "--repository", self.repository,
                   "--attempt-id", attempt, "--record-count", str(self.record_count if record_count is None else record_count),
                   *extra]
        if signal:
            command += ["--signal", signal]
        environment = {key: value for key, value in os.environ.items() if not key.startswith(("GITHUB_", "RUNNER_"))}
        environment.update(env or {})
        return subprocess.run(command, capture_output=True, text=True, env=environment, check=False)

    def document(self):
        with open(self.output, "rb") as handle:
            data = handle.read()
        return evidence.validate_bytes(data, SCHEMA), data.decode()

    def passing_records(self):
        return [
            ("journey", "v1-state", "n", "previous-1", "-", "2026-10-06T10:00:01Z"),
            ("step", "v1-state", "install-previous", "pass", "1200", "-"),
            ("observe", "v1-state", "classification_before", "valid_v1"),
            ("step", "v1-state", "upgrade", "pass", "800", "-"),
            ("observe", "v1-state", "installer_upgrade", "upgraded"),
            ("observe", "v1-state", "classification_after", "valid_v1"),
            ("observe", "v1-state", "installer_rerun", "unchanged"),
            ("journey_end", "v1-state", "2026-10-06T10:02:00Z"),
            ("journey", "recognized-poc", "historical_format", "poc-binary,previous-1", "recognized-poc",
             "2026-10-06T10:02:00Z"),
            ("step", "recognized-poc", "upgrade", "pass", "900", "-"),
            ("observe", "recognized-poc", "preservation_manifest_sha256", SHA),
            ("journey_end", "recognized-poc", "2026-10-06T10:04:00Z"),
        ]

    def test_pass_evidence_is_valid_and_bound_to_digests(self):
        self.write_records(self.passing_records())
        completed = self.emit()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, text = self.document()
        digest = evidence.hashlib.sha256(text.encode()).hexdigest()
        self.assertEqual(completed.stdout, f"evidence_sha256={digest}\n")
        self.assertEqual(document["result"]["status"], "pass")
        self.assertEqual(document["subject"]["kind"], "rebuilt")
        self.assertEqual(document["subject"]["version"], "9999.0.0-acceptance.7")
        self.assertEqual(document["subject"]["sha256sums_sha256"],
                         evidence.sha256_file(os.path.join(self.candidate, "SHA256SUMS")))
        self.assertEqual(document["subject"]["artifacts"][0]["sha256"],
                         evidence.sha256_file(os.path.join(self.candidate, "axiom-9999.0.0-acceptance.7-linux-amd64.tar.gz")))
        self.assertEqual([source["id"] for source in document["inputs"]["upgrade_sources"]],
                         ["previous-1", "previous-2", "poc-binary"])
        self.assertEqual(document["inputs"]["upgrade_sources"][1]["version"], "0.1.1")
        self.assertEqual(document["inputs"]["command"][-2:], ["--evidence", "{evidence}"])
        poc = document["journeys"][1]
        self.assertEqual((poc["source"]["path"], poc["preservation_manifest_sha256"]), (["poc-binary", "previous-1"], SHA))
        self.assertEqual(document["journeys"][0]["installer"], {"upgrade": "upgraded", "rerun": "unchanged"})
        self.assertIsNone(document["run"]["ci"])

    def test_evidence_is_secret_free_and_path_free(self):
        self.write_records(self.passing_records())
        secrets = {"GITHUB_ACTIONS": "true", "GITHUB_TOKEN": "ghs_secretvalue000", "ACTIONS_RUNTIME_TOKEN": "rt-secret",
                   "AWS_SECRET_ACCESS_KEY": "aws-secret", "GITHUB_REPOSITORY": "rgomids/axiom",
                   "GITHUB_RUN_ID": "987654321", "GITHUB_RUN_ATTEMPT": "3", "GITHUB_JOB": "upgrade-journeys",
                   "GITHUB_WORKFLOW_REF": "rgomids/axiom/.github/workflows/ci.yml@refs/heads/main",
                   "RUNNER_ENVIRONMENT": "github-hosted", "ImageOS": "ubuntu24", "ImageVersion": "20260928.1.0",
                   "AXIOM_EVIDENCE_RUNNER_LABEL": "ubuntu-24.04"}
        completed = self.emit(env=secrets)
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, text = self.document()
        for value in ("ghs_secretvalue000", "rt-secret", "aws-secret", self.root, os.path.expanduser("~")):
            self.assertNotIn(value, text)
        self.assertEqual(document["run"]["ci"]["run_attempt"], 3)
        self.assertEqual(document["environment"]["runner"]["label"], "ubuntu-24.04")

    def test_malformed_ci_facts_become_null_not_invented(self):
        self.write_records(self.passing_records())
        completed = self.emit(env={"GITHUB_ACTIONS": "true", "GITHUB_RUN_ATTEMPT": "first",
                                   "GITHUB_REPOSITORY": "not a repository", "ImageOS": "ubuntu 24\nx"})
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        self.assertIsNone(document["run"]["ci"]["run_attempt"])
        self.assertIsNone(document["run"]["ci"]["repository"])
        self.assertIsNone(document["environment"]["runner"]["image_os"])

    def test_observed_failure_category_is_carried_not_inferred(self):
        for category in ("-", "product", "test_defect", "infrastructure", "external_dependency", "timeout"):
            with self.subTest(category=category):
                records = self.passing_records()
                records[3] = ("step", "v1-state", "upgrade", "fail", "800", category)
                self.write_records(records)
                completed = self.emit(exit_code=1)
                self.assertEqual(completed.returncode, 0, completed.stderr)
                document, _ = self.document()
                expected = None if category == "-" else category
                self.assertEqual(document["journeys"][0]["steps"][1]["failure_category"], expected)
                self.assertEqual(document["result"]["failure_categories"], [expected] if expected else [])
                self.assertEqual(document["result"]["status"], "fail")
                os.unlink(self.output)

    def test_observed_product_failure_keeps_exit_status(self):
        records = self.passing_records()
        records[3] = ("step", "v1-state", "upgrade", "fail", "800", "product")
        records[5] = ("observe", "v1-state", "classification_after", "invalid value with spaces")
        self.write_records(records)
        completed = self.emit(exit_code=1)
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        result = document["result"]
        self.assertEqual((result["status"], result["termination"], result["exit_code"], result["failure_categories"]),
                         ("fail", "completed", 1, ["product"]))
        self.assertEqual(document["journeys"][0]["result"], "fail")
        self.assertEqual(document["journeys"][1]["result"], "pass")
        self.assertIsNone(document["journeys"][0]["classification"]["after"])

    def test_abort_marks_the_open_journey_incomplete(self):
        records = self.passing_records()[:4]
        records[1] = ("step", "v1-state", "install-previous", "fail", "1200", "-")
        self.write_records(records)
        completed = self.emit(exit_code=127, termination="aborted")
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        self.assertEqual((document["result"]["status"], document["result"]["termination"]), ("fail", "aborted"))
        self.assertEqual(document["result"]["failure_categories"], [])
        journey = document["journeys"][0]
        self.assertEqual((journey["completed"], journey["result"], journey["finished_at"]), (False, "fail", None))

    def test_interrupt_records_the_signal(self):
        self.write_records(self.passing_records()[:8])
        completed = self.emit(exit_code=143, termination="interrupted", signal="TERM")
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        self.assertEqual((document["result"]["status"], document["result"]["signal"]), ("fail", "TERM"))

    def test_abort_before_any_journey_still_identifies_the_subject(self):
        self.write_records([])
        completed = self.emit(exit_code=1, termination="aborted")
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        self.assertEqual((document["journeys"], document["result"]["status"]), ([], "fail"))

    def test_lost_or_reordered_records_refuse_the_document(self):
        for label, skip, count in (("intermediate", (4,), None), ("last", (12,), None), ("none written", (), 13)):
            with self.subTest(label=label):
                self.write_records(self.passing_records(), skip=skip)
                completed = self.emit(record_count=count)
                self.assertEqual(completed.returncode, 1)
                self.assertIn("evidence capture incomplete", completed.stderr)
                self.assertFalse(os.path.exists(self.output))

    def test_reordered_records_with_a_complete_count_are_refused(self):
        self.write_records(self.passing_records())
        with open(self.records, encoding="utf-8") as handle:
            lines = handle.readlines()
        lines[2], lines[3] = lines[3], lines[2]
        with open(self.records, "w", encoding="utf-8") as handle:
            handle.writelines(lines)
        completed = self.emit()
        self.assertEqual(completed.returncode, 1)
        self.assertIn("record 3 is missing or out of order", completed.stderr)
        self.assertFalse(os.path.exists(self.output))

    def test_attempt_identity_is_an_input_and_reemission_is_deterministic(self):
        self.write_records(self.passing_records())
        first = self.emit()
        self.assertEqual(first.returncode, 0, first.stderr)
        _, text = self.document()
        second_output = os.path.join(self.output_directory, "again.json")
        second = self.emit(output=second_output)
        self.assertEqual(second.returncode, 0, second.stderr)
        with open(second_output, encoding="utf-8") as handle:
            self.assertEqual(handle.read(), text)
        self.assertEqual(first.stdout, second.stdout)
        self.assertEqual(json.loads(text)["run"]["attempt_id"], ATTEMPT)

    def test_invalid_attempt_identity_is_refused(self):
        self.write_records(self.passing_records())
        for attempt in ("", "not-a-uuid", ATTEMPT.upper(), "3c06716f-6202-175b-9390-b20ca00e4877", ATTEMPT + "\n"):
            with self.subTest(attempt=attempt):
                completed = self.emit(attempt=attempt)
                self.assertEqual(completed.returncode, 1)
                self.assertIn("--attempt-id must be a lowercase UUIDv4", completed.stderr)
                self.assertFalse(os.path.exists(self.output))

    def test_source_archive_absent_from_its_sha256sums_is_unbound(self):
        with open(os.path.join(self.previous[0], "SHA256SUMS"), "w", encoding="utf-8") as handle:
            handle.write(f"{SHA}  axiom-0.5.0-linux-amd64.tar.gz\n")
        self.write_records(self.passing_records())
        completed = self.emit()
        self.assertEqual(completed.returncode, 1)
        self.assertIn("row archive digest", completed.stderr)
        self.assertFalse(os.path.exists(self.output))
        records = self.passing_records()
        records[3] = ("step", "v1-state", "upgrade", "fail", "800", "-")
        self.write_records(records)
        completed = self.emit(exit_code=1)
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        self.assertEqual(document["inputs"]["upgrade_sources"][0]["artifacts"], [])

    def test_sha256sums_binding_matches_the_installer(self):
        name = "axiom-0.5.0-linux-amd64.tar.gz"
        digest = evidence.sha256_file(os.path.join(self.previous[0], name))
        for listing in (f"{digest} *{name}\n", f"{digest}  {name}\n{digest}  {name}\n", f"{digest} {name}\n"):
            with self.subTest(listing=listing):
                with open(os.path.join(self.previous[0], "SHA256SUMS"), "w", encoding="utf-8") as handle:
                    handle.write(listing)
                self.write_records(self.passing_records())
                completed = self.emit()
                self.assertEqual(completed.returncode, 1)
                self.assertIn("row archive digest", completed.stderr)

    def test_existing_output_is_never_replaced(self):
        with open(self.output, "w", encoding="utf-8") as handle:
            handle.write("earlier")
        self.write_records(self.passing_records())
        completed = self.emit()
        self.assertEqual(completed.returncode, 1)
        self.assertIn("already exists", completed.stderr)
        with open(self.output, encoding="utf-8") as handle:
            self.assertEqual(handle.read(), "earlier")
        self.assertEqual(sorted(os.listdir(self.output_directory)), ["gate-evidence.json"])

    def test_undecodable_records_are_refused_cleanly(self):
        self.write_records(self.passing_records())
        with open(self.records, "ab") as handle:
            handle.write(b"13\tobserve\tv1-state\tinstaller_rerun\t\xff\n")
        completed = self.emit(record_count=13)
        self.assertEqual(completed.returncode, 1)
        self.assertIn("gate_evidence_error", completed.stderr)
        self.assertNotIn("Traceback", completed.stderr)

    def test_carriage_return_in_an_observed_value_is_not_a_record_break(self):
        records = self.passing_records()
        records[4] = ("observe", "v1-state", "installer_upgrade", "upgraded\rx")
        self.write_records(records)
        completed = self.emit()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        self.assertIsNone(document["journeys"][0]["installer"]["upgrade"])

    def test_subject_archive_absent_from_its_sha256sums_is_insufficient(self):
        with open(os.path.join(self.candidate, "SHA256SUMS"), "w", encoding="utf-8") as handle:
            handle.write(f"{SHA}  axiom-9999.0.0-acceptance.7-linux-amd64.tar.gz\n")
        self.write_records(self.passing_records())
        self.assertEqual(self.emit().returncode, 3)

    def test_inconsistent_state_is_refused_not_written(self):
        self.write_records(self.passing_records())
        completed = self.emit(exit_code=1)
        self.assertEqual(completed.returncode, 1)
        self.assertIn("needs a failed step or a failure category", completed.stderr)
        self.assertFalse(os.path.exists(self.output))

    def test_missing_subject_is_insufficient_state(self):
        os.unlink(os.path.join(self.candidate, "SHA256SUMS"))
        self.write_records(self.passing_records())
        completed = self.emit(exit_code=1, termination="aborted")
        self.assertEqual(completed.returncode, 3)
        self.assertIn("insufficient state", completed.stderr)
        self.assertFalse(os.path.exists(self.output))

    def test_unsafe_observed_values_become_null(self):
        records = self.passing_records()
        records[4] = ("observe", "v1-state", "installer_upgrade", "upgraded\textra")
        self.write_records(records)
        completed = self.emit()
        self.assertEqual(completed.returncode, 0, completed.stderr)
        document, _ = self.document()
        self.assertIsNone(document["journeys"][0]["installer"]["upgrade"])

    def test_malformed_records_are_refused(self):
        journey = ("journey", "j", "n", "previous-1", "-", "2026-10-06T10:00:01Z")
        for records in ([("step", "unknown", "x", "pass", "1", "-")],
                        [journey, ("step", "j", "x", "maybe", "1", "-")],
                        [journey, ("step", "j", "x", "fail", "1", "flaky")],
                        [journey, ("step", "j", "x", "fail", "1", "")],
                        [journey, ("step", "j", "x", "pass", "1", "product")],
                        [journey, ("step", "j", "x", "fail", "1")],
                        [journey, ("observe", "j", "stdout", "x")],
                        [("journey", "j", "n", "previous-9", "-", "2026-10-06T10:00:01Z"), ("step", "j", "x", "pass", "1", "-"),
                         ("journey_end", "j", "2026-10-06T10:00:02Z")]):
            with self.subTest(records=records):
                self.write_records(records)
                completed = self.emit()
                self.assertEqual(completed.returncode, 1, completed.stdout)
                self.assertFalse(os.path.exists(self.output))

    def test_output_never_lands_in_the_subject_sources_or_repository(self):
        self.write_records(self.passing_records())
        for directory in (self.candidate, self.previous[1], self.repository, os.path.join(self.repository, "testdata")):
            with self.subTest(directory=directory):
                completed = self.emit(output=os.path.join(directory, "gate-evidence.json"))
                self.assertEqual(completed.returncode, 1)
                self.assertFalse(os.path.exists(os.path.join(directory, "gate-evidence.json")))
        self.assertEqual(self.emit(output="relative.json").returncode, 1)

    def test_emitter_never_mutates_inputs(self):
        self.write_records(self.passing_records())
        self.assertEqual(self.emit().returncode, 0)
        self.assertEqual(self.inventory(), self.snapshot)

    def test_fixture_paths_stay_inside_the_repository(self):
        self.write_records(self.passing_records())
        for value in ("corpus=../outside", "corpus=/etc", "Bad=testdata/corpus", "corpus=testdata/missing"):
            with self.subTest(value=value):
                command = self.emit(extra=("--fixture", value))
                self.assertEqual(command.returncode, 1)


def capture_block():
    """The harness's own Evidence capture functions, verbatim."""
    with open(HARNESS, encoding="utf-8") as handle:
        text = handle.read()
    begin = text.index("# --- evidence capture: begin")
    return text[begin:text.index("# --- evidence capture: end", begin)]


class Capture(Emitter):
    """Drives the harness capture block (record, check, cleanup, emit_evidence)."""

    def run_block(self, body):
        work = tempfile.mkdtemp(dir=self.root)
        script = f"""set -euo pipefail
{capture_block()}
evidence={self.output!r} work={work!r} evidence_python={sys.executable!r} repository_root={ROOT!r}
row=linux-amd64 candidate={self.candidate!r} previous=({self.previous[0]!r}) poc_binary=
stable_corpus=internal/compatibility/testdata/stable-v1/snapshots/v0.4.0
original_home=$HOME original_path=$PATH termination= signal= record_count=0 capture_lost=0
attempt_id={ATTEMPT} started_at=$(utc_now) failures=0 journey=journey journey_id= clock_resolution_ms=1000
exec 8>&1 9>&2
trap cleanup EXIT
records="$work/evidence.records"
lose() {{ mv "$records" "$work/kept"; mkdir "$records"; }}
restore() {{ rmdir "$records"; mv "$work/kept" "$records"; }}
begin_journey v1-state n previous-1 -
{body}
end_journey
termination=completed
[[ $failures -eq 0 ]] || exit 1
"""
        environment = {key: value for key, value in os.environ.items() if not key.startswith(("GITHUB_", "RUNNER_"))}
        return subprocess.run(["bash", "-c", script], capture_output=True, text=True, env=environment, check=False)

    def test_intact_capture_writes_evidence_for_the_attempt(self):
        completed = self.run_block("check a true\ncheck_observed upgraded b true\ncheck c true")
        self.assertEqual(completed.returncode, 0, completed.stdout + completed.stderr)
        self.assertRegex(completed.stdout, r"evidence_sha256=[0-9a-f]{64}\n$")
        document, _ = self.document()
        self.assertEqual(document["run"]["attempt_id"], ATTEMPT)
        self.assertEqual([step["id"] for step in document["journeys"][0]["steps"]], ["a", "b", "c"])

    def test_record_loss_in_a_passing_run_fails_closed(self):
        completed = self.run_block("check a true\nlose\ncheck b true\nrestore\ncheck c true")
        self.assertEqual(completed.returncode, 70, completed.stdout + completed.stderr)
        self.assertIn("evidence=unavailable", completed.stdout)
        self.assertIn("capture incomplete: 1 of 5 Evidence records not written", completed.stdout)
        self.assertIn("journey=journey step=b result=pass", completed.stdout)
        self.assertFalse(os.path.exists(self.output))
        self.assertEqual(os.listdir(self.output_directory), [])

    def test_record_loss_in_a_failing_run_keeps_the_original_failure(self):
        completed = self.run_block("check a true\nlose\ncheck_observed unchanged b false\nrestore\ncheck c true")
        self.assertEqual(completed.returncode, 1, completed.stdout + completed.stderr)
        self.assertIn("journey=journey step=b result=fail", completed.stdout)
        self.assertIn("capture incomplete", completed.stdout)
        self.assertFalse(os.path.exists(self.output))

    def categories(self, body):
        completed = self.run_block(body)
        self.assertEqual(completed.returncode, 1, completed.stdout + completed.stderr)
        document, _ = self.document()
        os.unlink(self.output)
        return ([step["failure_category"] for step in document["journeys"][0]["steps"]],
                document["result"]["failure_categories"])

    def test_definite_candidate_observation_after_clean_steps_is_product(self):
        self.assertEqual(self.categories("check a true\ncheck_observed unchanged b false"), ([None, "product"], ["product"]))

    def test_unobservable_causes_stay_undetermined(self):
        self.assertEqual(self.categories("check a false"), ([None], []))
        self.assertEqual(self.categories("check_observed '' b false"), ([None], []))

    def test_an_earlier_failure_makes_later_observations_undetermined(self):
        # e.g. a failed upgrade leaves the previous binary answering the version probe
        self.assertEqual(self.categories("check upgrade false\ncheck_observed 0.5.0 version false"), ([None, None], []))

    def test_observed_values_cannot_break_a_record(self):
        completed = self.run_block("check a true\nobserve installer_upgrade $'upgraded\\r7\\tstep\\nx'")
        self.assertEqual(completed.returncode, 0, completed.stdout + completed.stderr)
        document, _ = self.document()
        self.assertIsNone(document["journeys"][0]["installer"]["upgrade"])


def harness_function(name):
    """One function definition from the harness, verbatim."""
    with open(HARNESS, encoding="utf-8") as handle:
        text = handle.read()
    start = text.index(f"\n{name}() {{") + 1
    end = text.index("\n}\n", start) + 3 if not text[start:text.index("\n", start)].rstrip().endswith("}") \
        else text.index("\n", start) + 1
    return text[start:end]


class ObservationHelpers(unittest.TestCase):
    def run_functions(self, body):
        script = "set -euo pipefail\n" + "".join(harness_function(name) for name in
                                                  ("digest", "tree_digest", "strict_tree_digest", "definite_status")) + body
        return subprocess.run(["bash", "-c", script], capture_output=True, text=True, check=False)

    def test_only_completed_installer_outcomes_are_definite(self):
        completed = self.run_functions("for s in installed upgraded unchanged partial success ''; do printf '[%s]' \"$(definite_status \"$s\")\"; done")
        self.assertEqual(completed.stdout, "[installed][upgraded][unchanged][][][]")

    @unittest.skipIf(hasattr(os, "geteuid") and os.geteuid() == 0, "root reads unreadable files")
    def test_strict_tree_digest_fails_where_tree_digest_still_answers(self):
        with tempfile.TemporaryDirectory() as directory:
            for name in ("a", "b"):
                with open(os.path.join(directory, name), "w", encoding="utf-8") as handle:
                    handle.write(name)
            same = self.run_functions(f"[[ $(tree_digest {directory!r}) == $(strict_tree_digest {directory!r}) ]]")
            self.assertEqual(same.returncode, 0, same.stderr)
            os.chmod(os.path.join(directory, "b"), 0)
            try:
                loose = self.run_functions(f"tree_digest {directory!r}")
                strict = self.run_functions(f"strict_tree_digest {directory!r}")
            finally:
                os.chmod(os.path.join(directory, "b"), 0o600)
            self.assertRegex(loose.stdout, r"^[0-9a-f]{64}\n$")
            self.assertNotEqual(strict.returncode, 0)


class Harness(unittest.TestCase):
    def run_harness(self, *arguments):
        return subprocess.run(["bash", HARNESS, *arguments], capture_output=True, text=True, check=False)

    def test_harness_parses(self):
        self.assertEqual(subprocess.run(["bash", "-n", HARNESS], check=False).returncode, 0)

    def test_evidence_argument_contract(self):
        with tempfile.TemporaryDirectory() as temporary:
            candidate = os.path.join(temporary, "candidate")
            previous = os.path.join(temporary, "previous")
            os.mkdir(candidate)
            os.mkdir(previous)
            existing = os.path.join(temporary, "existing.json")
            open(existing, "w", encoding="utf-8").close()
            for target in ("relative.json", os.path.join(temporary, "missing", "e.json"), temporary, existing,
                           os.path.join(candidate, "e.json"), os.path.join(previous, "e.json"),
                           os.path.join(ROOT, "e.json")):
                with self.subTest(target=target):
                    completed = self.run_harness("--candidate", candidate, "--previous", previous, "--evidence", target)
                    self.assertEqual(completed.returncode, 2, completed.stderr)
                    self.assertIn("--evidence", completed.stderr)
                    self.assertEqual(completed.stdout, "")


if __name__ == "__main__":
    unittest.main()
