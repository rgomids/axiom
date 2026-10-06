#!/usr/bin/env python3
"""axiom-gate-evidence/v1: validate release-gate Evidence and build it for the
upgrade journeys (Issue #234; Specification 004 FR-075/AC-56; ADR-0019).

    scripts/gate-evidence.py validate FILE...
    scripts/gate-evidence.py upgrade-journeys --records FILE --output ABS_FILE \
      --row ROW --candidate DIR [--previous DIR ...] [--poc-binary FILE] \
      --fixture ID=RELATIVE_PATH ... --started-at TS --finished-at TS \
      --exit-code N --termination completed|aborted|interrupted [--signal NAME] \
      --bash-version V --clock-resolution-ms 1|1000 --work-dir DIR --repository DIR

Validation is deterministic and offline. The closed JSON Schema
scripts/schemas/axiom-gate-evidence-v1.schema.json is interpreted by the strict
subset below (an unsupported keyword is an error, never ignored), followed by
the cross-field rules JSON Schema cannot express. The emitter reads only the
harness's step records, the subject and source archives (read-only), the
fixture trees and a fixed allowlist of host and CI facts. It never reads logs,
never dumps the environment, and prints only the written document's digest.

Exit status: 0 valid or written; 1 invalid document or emitter error; 2 usage;
3 not enough subject state to produce a document.
"""

from __future__ import annotations

import argparse
import datetime
import hashlib
import json
import os
import platform
import re
import subprocess
import sys
import tarfile
import tempfile
import uuid

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCHEMA_PATH = os.path.join(ROOT, "scripts", "schemas", "axiom-gate-evidence-v1.schema.json")
SCHEMA_ID = "axiom-gate-evidence/v1"
MAX_DOCUMENT_BYTES = 256 * 1024
MAX_STRING = 512
CONTROL = re.compile(r"[\x00-\x1f\x7f]")
TIMESTAMP_FORMATS = ("%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%dT%H:%M:%S.%fZ")

GATE = "merge-regression"
SUITE = "upgrade-journeys"
SCRIPT = "scripts/test-upgrade-journeys.sh"
# A failed harness assertion is a deterministic observation, so it is recorded
# as `product`: never a retryable category (Research #154). Re-attributing it to
# `test_defect` is a triage decision recorded by a new attempt, not here.
ASSERTION_FAILURE_CATEGORY = "product"
OBSERVATIONS = {
    "installer_upgrade": ("installer", "upgrade"),
    "installer_rerun": ("installer", "rerun"),
    "classification_before": ("classification", "before"),
    "classification_after": ("classification", "after"),
    "preservation_manifest_sha256": ("preservation_manifest_sha256", None),
}
ARCHITECTURES = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}


class Invalid(Exception):
    pass


class Insufficient(Exception):
    pass


class SchemaError(Exception):
    pass


# --- strict JSON ----------------------------------------------------------


def _reject_duplicates(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise Invalid(f"duplicate key {key!r}")
        result[key] = value
    return result


def _reject_constant(name):
    raise Invalid(f"non-JSON number {name}")


def load_document(data: bytes):
    if len(data) > MAX_DOCUMENT_BYTES:
        raise Invalid(f"document exceeds {MAX_DOCUMENT_BYTES} bytes")
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError as error:
        raise Invalid("document is not UTF-8") from error
    try:
        return json.loads(text, object_pairs_hook=_reject_duplicates, parse_constant=_reject_constant)
    except json.JSONDecodeError as error:
        raise Invalid(f"not JSON: {error.msg} at line {error.lineno}") from error


# --- strict JSON Schema subset -------------------------------------------

SUPPORTED_KEYWORDS = {
    "$schema", "$id", "$defs", "$comment", "title", "description",
    "type", "enum", "const", "pattern", "minLength", "maxLength", "minimum", "maximum",
    "minItems", "maxItems", "uniqueItems", "required", "properties",
    "additionalProperties", "items", "$ref", "allOf", "anyOf", "if", "then", "else",
}
TYPES = {"null", "boolean", "integer", "string", "array", "object"}


def _canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def _equal(left, right):
    return type(left) is type(right) and _canonical(left) == _canonical(right)


def _is_type(value, name):
    if name == "null":
        return value is None
    if name == "boolean":
        return isinstance(value, bool)
    if name == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if name == "string":
        return isinstance(value, str)
    if name == "array":
        return isinstance(value, list)
    return isinstance(value, dict)


def _pattern(source):
    """ECMA-262 semantics for a final `$`: end of input, never before a newline."""
    if source.endswith("$") and not source.endswith("\\$"):
        source = source[:-1] + r"\Z"
    return re.compile(source)


class Schema:
    def __init__(self, document):
        self.root = document
        self._check(document, "#")

    @classmethod
    def load(cls, path=SCHEMA_PATH):
        with open(path, "rb") as handle:
            return cls(load_document(handle.read()))

    def _check(self, schema, where):
        if not isinstance(schema, dict):
            raise SchemaError(f"{where}: schema must be an object")
        unknown = set(schema) - SUPPORTED_KEYWORDS
        if unknown:
            raise SchemaError(f"{where}: unsupported keyword(s) {sorted(unknown)}")
        types = schema.get("type")
        if types is not None:
            for name in types if isinstance(types, list) else [types]:
                if name not in TYPES:
                    raise SchemaError(f"{where}: unsupported type {name!r}")
        if "pattern" in schema:
            _pattern(schema["pattern"])
        if "additionalProperties" in schema and schema["additionalProperties"] is not False:
            raise SchemaError(f"{where}: only additionalProperties: false is supported")
        if "$ref" in schema:
            self.resolve(schema["$ref"])
        for key in ("$defs", "properties"):
            for name, child in schema.get(key, {}).items():
                self._check(child, f"{where}/{key}/{name}")
        for key in ("items", "if", "then", "else"):
            if key in schema:
                self._check(schema[key], f"{where}/{key}")
        for key in ("allOf", "anyOf"):
            for index, child in enumerate(schema.get(key, [])):
                self._check(child, f"{where}/{key}/{index}")

    def resolve(self, reference):
        if not reference.startswith("#/$defs/"):
            raise SchemaError(f"unsupported $ref {reference!r}")
        try:
            return self.root["$defs"][reference[len("#/$defs/"):]]
        except KeyError as error:
            raise SchemaError(f"unresolved $ref {reference!r}") from error

    def definition(self, name):
        return self.resolve(f"#/$defs/{name}")

    def accepts(self, value, schema):
        return not self.errors(value, schema)

    def errors(self, value, schema=None, path="$"):
        schema = self.root if schema is None else schema
        found = []
        if "$ref" in schema:
            found += self.errors(value, self.resolve(schema["$ref"]), path)
        types = schema.get("type")
        if types is not None:
            names = types if isinstance(types, list) else [types]
            if not any(_is_type(value, name) for name in names):
                return found + [f"{path}: expected {'/'.join(names)}"]
        if "const" in schema and not _equal(value, schema["const"]):
            found.append(f"{path}: must be {_canonical(schema['const'])}")
        if "enum" in schema and not any(_equal(value, option) for option in schema["enum"]):
            found.append(f"{path}: not an allowed value")
        if isinstance(value, str):
            if "minLength" in schema and len(value) < schema["minLength"]:
                found.append(f"{path}: shorter than {schema['minLength']}")
            if "maxLength" in schema and len(value) > schema["maxLength"]:
                found.append(f"{path}: longer than {schema['maxLength']}")
            if "pattern" in schema and not _pattern(schema["pattern"]).search(value):
                found.append(f"{path}: does not match the required format")
        if _is_type(value, "integer"):
            if "minimum" in schema and value < schema["minimum"]:
                found.append(f"{path}: below {schema['minimum']}")
            if "maximum" in schema and value > schema["maximum"]:
                found.append(f"{path}: above {schema['maximum']}")
        if isinstance(value, list):
            if "minItems" in schema and len(value) < schema["minItems"]:
                found.append(f"{path}: fewer than {schema['minItems']} items")
            if "maxItems" in schema and len(value) > schema["maxItems"]:
                found.append(f"{path}: more than {schema['maxItems']} items")
            if schema.get("uniqueItems") and len({_canonical(item) for item in value}) != len(value):
                found.append(f"{path}: items are not unique")
            if "items" in schema:
                for index, item in enumerate(value):
                    found += self.errors(item, schema["items"], f"{path}[{index}]")
        if isinstance(value, dict):
            for name in schema.get("required", []):
                if name not in value:
                    found.append(f"{path}: missing required property {name!r}")
            properties = schema.get("properties", {})
            if schema.get("additionalProperties") is False:
                for name in value:
                    if name not in properties:
                        found.append(f"{path}: property {name!r} is not part of the contract")
            for name, child in properties.items():
                if name in value:
                    found += self.errors(value[name], child, f"{path}.{name}")
        for child in schema.get("allOf", []):
            found += self.errors(value, child, path)
        if "anyOf" in schema and not any(self.accepts(value, child) for child in schema["anyOf"]):
            found.append(f"{path}: matches no allowed alternative")
        if "if" in schema:
            branch = schema.get("then") if self.accepts(value, schema["if"]) else schema.get("else")
            if branch is not None:
                found += self.errors(value, branch, path)
        return found


# --- cross-field rules ----------------------------------------------------


def _timestamp(value):
    for layout in TIMESTAMP_FORMATS:
        try:
            return datetime.datetime.strptime(value, layout)
        except ValueError:
            continue
    raise Invalid(f"invalid UTC timestamp {value!r}")


def _strings(value, path="$"):
    if isinstance(value, str):
        yield path, value
    elif isinstance(value, list):
        for index, item in enumerate(value):
            yield from _strings(item, f"{path}[{index}]")
    elif isinstance(value, dict):
        for key, item in value.items():
            yield f"{path}.{key}", key
            yield from _strings(item, f"{path}.{key}")


def semantic_errors(document):
    found = []
    for path, text in _strings(document):
        if CONTROL.search(text):
            found.append(f"{path}: control character")
        if len(text) > MAX_STRING:
            found.append(f"{path}: longer than {MAX_STRING}")
    try:
        started = _timestamp(document["started_at"])
        finished = _timestamp(document["finished_at"])
    except Invalid as error:
        return found + [str(error)]
    if finished < started:
        found.append("$.finished_at: before started_at")

    def unique(values, path):
        if len(set(values)) != len(values):
            found.append(f"{path}: identifiers are not unique")

    unique([item["name"] for item in document["subject"]["artifacts"]], "$.subject.artifacts")
    sources = [item["id"] for item in document["inputs"]["upgrade_sources"]]
    unique(sources, "$.inputs.upgrade_sources")
    unique([item["id"] for item in document["inputs"]["fixtures"]], "$.inputs.fixtures")
    unique([item["id"] for item in document["journeys"]], "$.journeys")
    for placeholder in document["inputs"]["command"]:
        if placeholder.startswith("{") and placeholder[1:-1] not in ("candidate", "evidence", *sources):
            found.append(f"$.inputs.command: placeholder {placeholder} is bound to nothing")

    totals = {"pass": 0, "fail": 0, "not_applicable": 0}
    failed_categories = set()
    failed_journeys = 0
    for index, journey in enumerate(document["journeys"]):
        where = f"$.journeys[{index}]"
        unique([step["id"] for step in journey["steps"]], f"{where}.steps")
        for source in journey["source"]["path"]:
            if source not in sources:
                found.append(f"{where}.source.path: unknown upgrade source {source!r}")
        try:
            journey_started = _timestamp(journey["started_at"])
            if not started <= journey_started <= finished:
                found.append(f"{where}.started_at: outside the run")
            if journey["finished_at"] is not None:
                journey_finished = _timestamp(journey["finished_at"])
                if not journey_started <= journey_finished <= finished:
                    found.append(f"{where}.finished_at: outside the journey or run")
        except Invalid as error:
            found.append(f"{where}: {error}")
        results = [step["result"] for step in journey["steps"]]
        for step in journey["steps"]:
            totals[step["result"]] += 1
            if step["result"] == "fail":
                failed_categories.add(step["failure_category"])
        if journey["result"] == "pass" and (not journey["completed"] or "pass" not in results or "fail" in results):
            found.append(f"{where}.result: pass needs a completed journey with passing and no failing steps")
        if journey["result"] == "not_applicable" and any(result != "not_applicable" for result in results):
            found.append(f"{where}.result: not_applicable journeys contain only not_applicable steps")
        if journey["result"] == "fail":
            failed_journeys += 1
            if journey["completed"] and "fail" not in results:
                found.append(f"{where}.result: a completed failing journey needs a failing step")

    result = document["result"]
    counts = result["counts"]
    expected = {
        "journeys": len(document["journeys"]),
        "steps": sum(totals.values()),
        "passed": totals["pass"],
        "failed": totals["fail"],
        "not_applicable": totals["not_applicable"],
    }
    for name, value in expected.items():
        if counts[name] != value:
            found.append(f"$.result.counts.{name}: {counts[name]} does not match the journeys ({value})")
    categories = result["failure_categories"]
    if categories != sorted(categories):
        found.append("$.result.failure_categories: must be sorted")
    missing = failed_categories - set(categories)
    if missing:
        found.append(f"$.result.failure_categories: missing {sorted(missing)} of failed steps")
    if result["status"] == "pass" and (failed_journeys or result["termination"] != "completed"):
        found.append("$.result.status: pass contradicts a failed or incomplete journey")
    if result["status"] == "pass" and not totals["pass"]:
        found.append("$.result.status: pass needs at least one passing step; not_applicable is never a pass")
    if result["status"] == "fail" and not categories and result["termination"] == "completed":
        found.append("$.result.status: a completed failing run needs a failure category")
    if result["termination"] != "completed" and result["status"] != "fail":
        found.append("$.result.status: an aborted or interrupted run is a failure")
    return found


def validate(document, schema=None):
    schema = schema or Schema.load()
    found = schema.errors(document)
    if not found:
        found = semantic_errors(document)
    if found:
        raise Invalid("; ".join(found[:20]))


def validate_bytes(data, schema=None):
    document = load_document(data)
    validate(document, schema)
    return document


# --- emitter: upgrade journeys -------------------------------------------


def sha256_file(path):
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for block in iter(lambda: handle.read(1 << 20), b""):
            digest.update(block)
    return digest.hexdigest()


def tree_sha256(root):
    lines = []
    for directory, directories, files in os.walk(root):
        directories.sort()
        for name in files:
            path = os.path.join(directory, name)
            if os.path.isfile(path) and not os.path.islink(path):
                relative = os.path.relpath(path, root).replace(os.sep, "/")
                lines.append((relative, f"{sha256_file(path)}  {relative}\n"))
    if not lines:
        raise Insufficient(f"fixture {root!r} has no files")
    return hashlib.sha256("".join(line for _, line in sorted(lines)).encode()).hexdigest()


def row_archive(directory, row):
    suffix = f"-{row}.tar.gz"
    try:
        names = sorted(name for name in os.listdir(directory)
                       if name.startswith("axiom-") and name.endswith(suffix)
                       and os.path.isfile(os.path.join(directory, name)))
    except OSError:
        return None
    return os.path.join(directory, names[0]) if len(names) == 1 else None


def archive_metadata(archive):
    """Reads release-metadata.txt from the archive without extracting it."""
    try:
        with tarfile.open(archive, "r:gz") as bundle:
            for member in bundle.getmembers():
                parts = member.name.split("/")
                if member.isfile() and len(parts) == 2 and parts[1] == "release-metadata.txt":
                    handle = bundle.extractfile(member)
                    text = handle.read(16384).decode("utf-8", "replace") if handle else ""
                    return dict(line.split("=", 1) for line in text.splitlines() if "=" in line)
    except (OSError, tarfile.TarError, EOFError):
        return {}
    return {}


class Builder:
    def __init__(self, arguments, schema):
        self.arguments = arguments
        self.schema = schema

    def observed(self, value, definition):
        """Keeps an observed fact only when it fits the contract; otherwise null."""
        subschema = definition if isinstance(definition, dict) else self.schema.definition(definition)
        return value if value is not None and self.schema.accepts(value, subschema) else None

    def artifact(self, path):
        name = os.path.basename(path)
        if not self.schema.accepts(name, self.schema.definition("artifact")["properties"]["name"]):
            return None
        return {"name": name, "sha256": sha256_file(path)}

    def subject(self):
        candidate = self.arguments.candidate
        archive = row_archive(candidate, self.arguments.row)
        sums = os.path.join(candidate, "SHA256SUMS")
        if archive is None or not os.path.isfile(sums):
            raise Insufficient("candidate row archive or SHA256SUMS is missing")
        metadata = archive_metadata(archive)
        version = self.observed(metadata.get("version"), "version")
        revision = self.observed(metadata.get("revision"), self.schema.definition("subject")["properties"]["revision"])
        artifact = self.artifact(archive)
        if version is None or revision is None or artifact is None:
            raise Insufficient("candidate release metadata is missing or malformed")
        return {
            "kind": "rebuilt",
            "tag": None,
            "version": version,
            "revision": revision,
            "sha256sums_sha256": sha256_file(sums),
            "artifacts": [artifact],
            "artifact_id": None,
            "artifact_digest": None,
        }

    def upgrade_sources(self):
        sources = []
        for index, directory in enumerate(self.arguments.previous, start=1):
            archive = row_archive(directory, self.arguments.row)
            sums = os.path.join(directory, "SHA256SUMS")
            artifact = self.artifact(archive) if archive else None
            sources.append({
                "id": f"previous-{index}",
                "kind": "release",
                "version": self.observed(archive_metadata(archive).get("version") if archive else None, "nullable_version"),
                "tag": None,
                "sha256sums_sha256": sha256_file(sums) if os.path.isfile(sums) else None,
                "artifacts": [artifact] if artifact else [],
            })
        if self.arguments.poc_binary:
            path = self.arguments.poc_binary
            artifact = self.artifact(path) if os.path.isfile(path) else None
            sources.append({
                "id": "poc-binary",
                "kind": "historical_build",
                "version": None,
                "tag": None,
                "sha256sums_sha256": None,
                "artifacts": [artifact] if artifact else [],
            })
        return sources

    def fixtures(self):
        fixtures = []
        repository = os.path.realpath(self.arguments.repository)
        for item in self.arguments.fixture:
            identifier, _, relative = item.partition("=")
            definition = self.schema.definition("fixture")["properties"]
            if not (self.schema.accepts(identifier, definition["id"]) and self.schema.accepts(relative, definition["path"])) \
                    or ".." in relative.split("/"):
                raise Invalid(f"invalid --fixture {item!r}")
            root = os.path.realpath(os.path.join(repository, relative))
            if os.path.commonpath([repository, root]) != repository or not os.path.isdir(root):
                raise Invalid(f"fixture {relative!r} is not a repository directory")
            fixtures.append({"id": identifier, "path": relative, "tree_sha256": tree_sha256(root)})
        return fixtures

    def command(self):
        command = [SCRIPT, "--candidate", "{candidate}"]
        for index in range(1, len(self.arguments.previous) + 1):
            command += ["--previous", f"{{previous-{index}}}"]
        if self.arguments.poc_binary:
            command += ["--poc-binary", "{poc-binary}"]
        return command + ["--evidence", "{evidence}"]

    def repository(self):
        def git(*arguments):
            try:
                completed = subprocess.run(["git", "-C", self.arguments.repository, *arguments],
                                           capture_output=True, text=True, timeout=30, check=False)
            except (OSError, subprocess.TimeoutExpired):
                return None
            return completed.stdout if completed.returncode == 0 else None

        revision = git("rev-parse", "--verify", "HEAD")
        status = git("status", "--porcelain")
        return {
            "revision": self.observed(revision.strip() if revision else None,
                                      self.schema.definition("inputs")["properties"]["repository"]["properties"]["revision"]),
            "state": None if status is None else ("dirty" if status.strip() else "clean"),
        }

    def records(self):
        journeys, order = {}, []
        with open(self.arguments.records, encoding="utf-8") as handle:
            for number, line in enumerate(handle, start=1):
                fields = line.rstrip("\n").split("\t", 3 if line.startswith("observe\t") else -1)
                kind = fields[0]
                if kind == "journey" and len(fields) == 6:
                    _, identifier, role, path, generation, started_at = fields
                    if identifier in journeys:
                        raise Invalid(f"records:{number}: journey {identifier!r} repeated")
                    journeys[identifier] = {
                        "id": identifier,
                        "source": {"role": role, "path": path.split(","),
                                   "generation": None if generation == "-" else generation, "baseline": None},
                        "completed": False,
                        "result": "fail",
                        "reason": None,
                        "governing_reference": None,
                        "installer": {"upgrade": None, "rerun": None},
                        "classification": {"before": None, "after": None},
                        "preservation_manifest_sha256": None,
                        "started_at": started_at,
                        "finished_at": None,
                        "steps": [],
                    }
                    order.append(identifier)
                    continue
                journey = journeys.get(fields[1]) if len(fields) > 1 else None
                if journey is None:
                    raise Invalid(f"records:{number}: unknown or missing journey")
                if kind == "step" and len(fields) == 5 and fields[3] in ("pass", "fail") and fields[4].isdigit():
                    failed = fields[3] == "fail"
                    journey["steps"].append({
                        "id": fields[2],
                        "result": fields[3],
                        "failure_category": ASSERTION_FAILURE_CATEGORY if failed else None,
                        "reason": None,
                        "governing_reference": None,
                        "duration_ms": int(fields[4]),
                    })
                elif kind == "observe" and len(fields) == 4 and fields[2] in OBSERVATIONS:
                    section, key = OBSERVATIONS[fields[2]]
                    definition = self.schema.definition("journey")["properties"][section]
                    if key is None:
                        journey[section] = self.observed(fields[3] or None, definition)
                    else:
                        journey[section][key] = self.observed(fields[3] or None, definition["properties"][key])
                elif kind == "journey_end" and len(fields) == 3:
                    journey["completed"] = True
                    journey["finished_at"] = fields[2]
                else:
                    raise Invalid(f"records:{number}: malformed record")
        for identifier in order:
            journey = journeys[identifier]
            results = {step["result"] for step in journey["steps"]}
            journey["result"] = "pass" if journey["completed"] and "fail" not in results else "fail"
        return [journeys[identifier] for identifier in order]

    def build(self):
        arguments = self.arguments
        journeys = self.records()
        steps = [step for journey in journeys for step in journey["steps"]]
        failed = [step for step in steps if step["result"] == "fail"]
        status = "pass" if arguments.termination == "completed" and arguments.exit_code == 0 and not failed else "fail"
        return {
            "schema": SCHEMA_ID,
            "gate": GATE,
            "suite": SUITE,
            "row": arguments.row,
            "subject": self.subject(),
            "run": {"attempt_id": str(uuid.uuid4()), "ci": ci_identity(self)},
            "environment": environment(self),
            "inputs": {
                "upgrade_sources": self.upgrade_sources(),
                "fixtures": self.fixtures(),
                "command": self.command(),
                "repository": self.repository(),
            },
            "journeys": journeys,
            "result": {
                "status": status,
                "termination": arguments.termination,
                "exit_code": arguments.exit_code,
                "signal": arguments.signal or None,
                "failure_categories": sorted({step["failure_category"] for step in failed}),
                "counts": {
                    "journeys": len(journeys),
                    "steps": len(steps),
                    "passed": sum(step["result"] == "pass" for step in steps),
                    "failed": len(failed),
                    "not_applicable": sum(step["result"] == "not_applicable" for step in steps),
                },
            },
            "started_at": arguments.started_at,
            "finished_at": arguments.finished_at,
        }


def _run(command, **options):
    try:
        completed = subprocess.run(command, capture_output=True, text=True, timeout=30, check=False, **options)
    except (OSError, subprocess.TimeoutExpired):
        return None
    return completed.stdout.strip() if completed.returncode == 0 else None


def ci_identity(builder):
    """GitHub Actions identity from an allowlist of variables; null locally."""
    if os.environ.get("GITHUB_ACTIONS") != "true":
        return None
    definition = builder.schema.definition("ci")["properties"]
    attempt = os.environ.get("GITHUB_RUN_ATTEMPT", "")
    return {
        "provider": "github-actions",
        "repository": builder.observed(os.environ.get("GITHUB_REPOSITORY"), definition["repository"]),
        "workflow_ref": builder.observed(os.environ.get("GITHUB_WORKFLOW_REF"), definition["workflow_ref"]),
        "run_id": builder.observed(os.environ.get("GITHUB_RUN_ID"), definition["run_id"]),
        "run_attempt": builder.observed(int(attempt) if attempt.isdigit() else None, definition["run_attempt"]),
        "job": builder.observed(os.environ.get("GITHUB_JOB"), definition["job"]),
    }


def filesystem_type(path):
    """Type of the filesystem holding path, by device identity."""
    mounts = []
    if os.path.isfile("/proc/self/mounts"):
        with open("/proc/self/mounts", encoding="utf-8", errors="replace") as handle:
            for line in handle:
                fields = line.split()
                if len(fields) >= 3:
                    mountpoint = re.sub(r"\\([0-7]{3})", lambda match: chr(int(match.group(1), 8)), fields[1])
                    mounts.append((mountpoint, fields[2]))
    else:
        output = _run(["/sbin/mount"]) or ""
        for line in output.splitlines():
            match = re.match(r"^.+? on (.+) \(([^,()]+)", line)
            if match:
                mounts.append((match.group(1), match.group(2)))
    try:
        device = os.stat(path).st_dev
    except OSError:
        return None
    found = None
    for mountpoint, kind in mounts:
        try:
            if os.stat(mountpoint).st_dev == device:
                found = kind
        except OSError:
            continue
    return found


def os_release():
    values = {}
    try:
        with open("/etc/os-release", encoding="utf-8", errors="replace") as handle:
            for line in handle:
                key, _, value = line.strip().partition("=")
                values[key] = value.strip().strip('"')
    except OSError:
        pass
    return values


def environment(builder):
    arguments = builder.arguments
    family = platform.system().lower()
    if family == "darwin":
        distribution, version = "macos", _run(["/usr/bin/sw_vers", "-productVersion"])
    else:
        release = os_release()
        distribution, version = release.get("ID"), release.get("VERSION_ID")
    go_version = _run(["go", "env", "GOVERSION"], cwd="/", env={**os.environ, "GOTOOLCHAIN": "local"})
    runner = None
    if os.environ.get("GITHUB_ACTIONS") == "true":
        runner_environment = os.environ.get("RUNNER_ENVIRONMENT")
        runner = {
            "environment": runner_environment if runner_environment in ("github-hosted", "self-hosted") else None,
            "label": builder.observed(os.environ.get("AXIOM_EVIDENCE_RUNNER_LABEL"), "nullable_fact"),
            "image_os": builder.observed(os.environ.get("ImageOS"), "nullable_fact"),
            "image_version": builder.observed(os.environ.get("ImageVersion"), "nullable_fact"),
        }
    definition = builder.schema.definition("environment")["properties"]
    architecture = ARCHITECTURES.get(platform.machine().lower())
    if family not in ("darwin", "linux", "windows") or architecture is None:
        raise Insufficient("unsupported host family or architecture")
    return {
        "mode": "native",
        "os": {
            "family": family,
            "kernel_release": builder.observed(platform.release(), "nullable_fact"),
            "distribution": builder.observed(distribution, definition["os"]["properties"]["distribution"]),
            "version": builder.observed(version, "nullable_fact"),
        },
        "architecture": architecture,
        "filesystem": builder.observed(filesystem_type(arguments.work_dir), definition["filesystem"]),
        "runner": runner,
        "go_version": builder.observed(go_version, definition["go_version"]),
        "runtime": {
            "bash": builder.observed(arguments.bash_version, definition["runtime"]["properties"]["bash"]),
            "python": builder.observed(platform.python_version(), "nullable_fact"),
            "clock_resolution_ms": arguments.clock_resolution_ms,
        },
    }


def _inside(path, directory):
    path, directory = os.path.realpath(path), os.path.realpath(directory)
    return os.path.commonpath([path, directory]) == directory


def write_document(document, output, schema):
    data = (json.dumps(document, indent=2, ensure_ascii=True) + "\n").encode()
    validate_bytes(data, schema)
    directory = os.path.dirname(output)
    handle, temporary = tempfile.mkstemp(prefix=".gate-evidence.", dir=directory)
    try:
        with os.fdopen(handle, "wb") as stream:
            stream.write(data)
        os.replace(temporary, output)
    except BaseException:
        if os.path.exists(temporary):
            os.unlink(temporary)
        raise
    return hashlib.sha256(data).hexdigest()


def upgrade_journeys(arguments):
    output = arguments.output
    if not os.path.isabs(output) or not os.path.isdir(os.path.dirname(output)) or os.path.isdir(output):
        raise Invalid("--output must be an absolute file path in an existing directory")
    for directory in [arguments.candidate, *arguments.previous, arguments.repository]:
        if _inside(os.path.dirname(output), directory):
            raise Invalid("--output must be outside the subject, upgrade sources and repository")
    schema = Schema.load()
    document = Builder(arguments, schema).build()
    print(f"evidence_sha256={write_document(document, output, schema)}")


def parse(argv):
    parser = argparse.ArgumentParser(prog="gate-evidence.py")
    commands = parser.add_subparsers(dest="command", required=True)
    check = commands.add_parser("validate")
    check.add_argument("files", nargs="+")
    emit = commands.add_parser("upgrade-journeys")
    emit.add_argument("--records", required=True)
    emit.add_argument("--output", required=True)
    emit.add_argument("--row", required=True, choices=["linux-amd64", "linux-arm64", "macos-27-arm64"])
    emit.add_argument("--candidate", required=True)
    emit.add_argument("--previous", action="append", default=[])
    emit.add_argument("--poc-binary", default="")
    emit.add_argument("--fixture", action="append", default=[])
    emit.add_argument("--started-at", required=True)
    emit.add_argument("--finished-at", required=True)
    emit.add_argument("--exit-code", required=True, type=int)
    emit.add_argument("--termination", required=True, choices=["completed", "aborted", "interrupted"])
    emit.add_argument("--signal", default="", choices=["", "HUP", "INT", "TERM"])
    emit.add_argument("--bash-version", default="")
    emit.add_argument("--clock-resolution-ms", required=True, type=int, choices=[1, 1000])
    emit.add_argument("--work-dir", required=True)
    emit.add_argument("--repository", default=ROOT)
    return parser.parse_args(argv)


def main(argv=None):
    arguments = parse(sys.argv[1:] if argv is None else argv)
    try:
        if arguments.command == "validate":
            schema = Schema.load()
            for path in arguments.files:
                with open(path, "rb") as handle:
                    validate_bytes(handle.read(), schema)
                print(f"gate_evidence=valid {os.path.basename(path)}")
        else:
            upgrade_journeys(arguments)
    except Insufficient as error:
        print(f"gate_evidence_error: insufficient state: {error}", file=sys.stderr)
        return 3
    except (Invalid, SchemaError, OSError) as error:
        print(f"gate_evidence_error: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
