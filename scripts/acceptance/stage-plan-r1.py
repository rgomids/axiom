#!/usr/bin/env python3
"""Manual, offline #275 Lane S collector. No native Runtime or Provider calls."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import tempfile
import threading
import uuid

HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
VERSION = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?\Z")
JOURNEY = "TestWorkflowStagePlanPublicJourneySingleAndMixedRuntimeGraph"
EXECUTABLE = "TestWorkflowStagePlanExecutableFailsClosedAndNeverMutates"
AUTHORING = "TestExecutableWorkflowDefaultCustomEditSelectAndRetire"
PACKAGES = {
    "./cmd/lingo": (JOURNEY, EXECUTABLE, AUTHORING, "TestReadPlanDocumentRefusesNonRegularFiles"),
    "./internal/workflowcompiler": ("TestCompileRejectsInvalidTopologyControlsAndReferences", "TestCompileSequentialConcurrencyAndUnsafeOverlap", "TestPlanStageFailsClosedWithSpecificationCategories"),
    "./internal/cli": ("TestUnsupportedDomainOperationsFailSafely",),
}
TESTS = {key: [JOURNEY] for key in "ACDE"}
TESTS["B"] = [AUTHORING]
TESTS["F"] = [JOURNEY, EXECUTABLE, *PACKAGES["./cmd/lingo"][3:], *PACKAGES["./internal/workflowcompiler"], *PACKAGES["./internal/cli"]]
TESTS["G"] = [JOURNEY, EXECUTABLE]
TESTS["H"] = []
REFUSALS = {"runtime_unresolvable", "stale_execution_revision", "stage_prerequisite_missing", "stage_not_found", "execution_selector_conflict", "invalid_stage_plan", "authority_denied", "unsupported_effort", "invalid_stage_topology", "stale_authority", "invalid_definition", "prior_revision_required", "workflow_exists", "revision_in_use", "workflow_not_found", "reference_inventory_unknown"}
LIMITATIONS = ["G-1", "G-2", "G-3", "G-4", "G-5"]
SKILLS = {"axiom-project", "axiom-work-item", "axiom-workflow"}
SCHEMA = Path(__file__).with_name("stage-plan-r1.schema.json")


def validate_schema(value, schema=None, root=None):
    """Validate the closed, dependency-free subset used by the bundled schema."""
    schema = json.loads(SCHEMA.read_text()) if schema is None else schema
    root = schema if root is None else root
    if "$ref" in schema:
        node = root
        for part in schema["$ref"].split("/")[1:]:
            node = node[part]
        return validate_schema(value, node, root)
    kinds = {"object": dict, "array": list, "string": str, "boolean": bool, "null": type(None)}
    if "type" in schema:
        names = schema["type"] if isinstance(schema["type"], list) else [schema["type"]]
        if not any(type(value) is kinds[name] for name in names):
            raise ValueError("schema_type")
    if "const" in schema and value != schema["const"] or "enum" in schema and value not in schema["enum"]:
        raise ValueError("schema_value")
    if isinstance(value, str) and "pattern" in schema and re.fullmatch(schema["pattern"], value) is None:
        raise ValueError("schema_pattern")
    if isinstance(value, dict):
        properties = schema.get("properties", {})
        if not set(schema.get("required", ())).issubset(value):
            raise ValueError("schema_required")
        if schema.get("additionalProperties") is False and set(value) - set(properties):
            raise ValueError("schema_unknown")
        for key, child in value.items():
            if key in properties:
                validate_schema(child, properties[key], root)
    if isinstance(value, list):
        if len(value) < schema.get("minItems", 0):
            raise ValueError("schema_items")
        if schema.get("uniqueItems") and len({json.dumps(v, sort_keys=True) for v in value}) != len(value):
            raise ValueError("schema_unique")
        for child in value:
            validate_schema(child, schema.get("items", {}), root)
    for child in schema.get("allOf", []):
        validate_schema(value, child, root)
    if "if" in schema:
        try:
            validate_schema(value, schema["if"], root)
        except ValueError:
            pass
        else:
            validate_schema(value, schema.get("then", {}), root)


def command(argv, source, env, timeout=180, bound=2 * 1024 * 1024):
    """Bound allocation and time; kill the entire subprocess group on refusal."""
    chunks, exceeded = [], threading.Event()
    try:
        process = subprocess.Popen(argv, cwd=source, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, start_new_session=os.name != "nt")
    except OSError:
        return "unavailable", b""

    def stop():
        try:
            if os.name != "nt":
                os.killpg(process.pid, signal.SIGKILL)
            else:
                process.kill()
        except ProcessLookupError:
            pass

    def read():
        count = 0
        while True:
            data = process.stdout.read(4096)
            if not data:
                return
            count += len(data)
            if count > bound:
                exceeded.set()
                stop()
                return
            chunks.append(data)

    reader = threading.Thread(target=read, daemon=True)
    reader.start()
    try:
        code = process.wait(timeout=timeout)
    except subprocess.TimeoutExpired:
        stop()
        process.wait()
        code = "timeout"
    reader.join(timeout=1)
    if reader.is_alive():
        stop()
        reader.join(timeout=1)
        code = "timeout"
    process.stdout.close()
    return ("output_bound" if exceeded.is_set() else code), b"".join(chunks)


def report(target):
    return {"axiomVersion": "unreleased", "versionProof": "none", "targetRevision": target if HEX40.fullmatch(target) else None, "observedRevision": None, "sourceClean": None, "goVersion": None, "runtimeObservation": "controlled", "skillSet": {"version": None, "skills": []}, "sandboxId": "r1-" + uuid.uuid4().hex, "lane": "S", "class": "synthetic", "provenance": {"source": "git", "revisionVerified": False, "versionVerified": False}, "scenarios": {key: {"result": "blocked", "tests": names[:], "refusalCategories": [], "canonicalDigests": {}} for key, names in TESTS.items()}, "r1": "blocked", "r2": "deferred_to_278", "r3": "deferred_to_278", "limitations": LIMITATIONS[:], "handoff": "https://github.com/rgomids/axiom/issues/278#AXM-12"}


def refuse(value, reason, result="blocked"):
    value["r1"] = result
    for scenario in value["scenarios"].values():
        scenario.update(result=result, refusalCategories=[reason], canonicalDigests={})
    return value


def provenance_gate(value, source, version, binary, env, run):
    if value["targetRevision"] is None:
        return "target_unverifiable"
    code, wire = run(["git", "rev-parse", "HEAD"], source, env)
    observed = wire.decode("ascii", "ignore").strip()
    if code != 0 or not HEX40.fullmatch(observed):
        return "target_unverifiable"
    value["observedRevision"] = observed
    if observed != value["targetRevision"]:
        return "revision_mismatch"
    code, wire = run(["git", "status", "--porcelain", "--untracked-files=all"], source, env)
    if code != 0:
        return "target_unverifiable"
    value["sourceClean"] = not bool(wire.strip())
    if not value["sourceClean"]:
        return "dirty_source"
    code, wire = run(["go", "version"], source, env)
    go = wire.decode("ascii", "ignore").strip()
    if code != 0 or not re.fullmatch(r"go version go[0-9]+\.[0-9]+(?:\.[0-9]+)? [a-z0-9]+/[a-z0-9]+", go):
        return "toolchain_missing"
    value["goVersion"] = go
    value["provenance"]["revisionVerified"] = True
    if version is None:
        return None
    if not VERSION.fullmatch(version):
        return "version_unverifiable"
    code, wire = run(["git", "rev-parse", "--verify", "v" + version + "^{commit}"], source, env)
    tag = wire.decode("ascii", "ignore").strip()
    mismatch = code == 0 and HEX40.fullmatch(tag) and tag != value["targetRevision"]
    if code == 0 and tag == value["targetRevision"]:
        value.update(axiomVersion=version, versionProof="tag")
    elif binary:
        code, wire = run([str(Path(binary).resolve()), "--json", "version"], source, env)
        try:
            claimed = json.loads(wire)["provenance"]
        except (ValueError, KeyError, TypeError, RecursionError):
            return "version_unverifiable"
        if code != 0:
            return "version_unverifiable"
        if not isinstance(claimed, dict):
            return "version_unverifiable"
        if claimed.get("product") != "Axiom" or claimed.get("version") != version or claimed.get("revision") != value["targetRevision"] or claimed.get("sourceState") != "clean":
            return "version_mismatch"
        value.update(axiomVersion=version, versionProof="release-binary")
    else:
        return "version_mismatch" if mismatch else "version_unverifiable"
    value["provenance"]["versionVerified"] = True
    return None


def parse_events(wire, expected, package):
    outcomes, markers, pending = {}, [], {}
    for line in wire.splitlines():
        try:
            event = json.loads(line)
        except (ValueError, TypeError, RecursionError):
            continue
        if not isinstance(event, dict) or event.get("Package") != package:
            continue
        test = event.get("Test", "")
        if test not in expected:
            continue
        if event.get("Action") in ("pass", "fail", "skip"):
            outcomes[test] = event["Action"]
        output = event.get("Output", "")
        if event.get("Action") != "output" or not isinstance(output, str):
            continue
        # test2json splits long log lines into 1024-byte Output events.
        # Reassemble only that test's bounded line, never an unbounded transcript.
        pending[test] = pending.get(test, "") + output
        if len(pending[test]) > 128 * 1024:
            pending[test] = ""
            continue
        while "\n" in pending[test]:
            logged, pending[test] = pending[test].split("\n", 1)
            match = re.fullmatch(r"\s*(?:[^\r\n]*\.go:\d+: )?r1-evidence (\{[^\r\n]*\})\s*", logged)
            if match:
                try:
                    marker = json.loads(match[1])
                    if isinstance(marker, dict) and marker.get("scenario") in TESTS and isinstance(marker.get("canonical"), dict):
                        markers.append((test, marker))
                except (ValueError, TypeError, RecursionError):
                    pass
    return outcomes, markers


def plan_digests(plan):
    try:
        value = {"workflowRefDigest": plan["workflowRef"]["digest"], "planDigest": plan["digest"], "planDocumentDigest": plan["planDocumentDigest"], "stageInputRefDigest": plan["stageInputRef"]["digest"]}
        if plan.get("executionKind") == "graph":
            value["graphProposalRefDigest"] = plan["graphProposalRef"]["digest"]
        if not all(isinstance(d, str) and HEX64.fullmatch(d) for d in value.values()):
            return None
        return value
    except (KeyError, TypeError):
        return None


def collect_results(value, outcomes, markers):
    required = {name for names in TESTS.values() for name in names}
    if any(outcomes.get(name) == "fail" for name in required):
        return refuse(value, "test_failed", "failed")
    if any(name not in outcomes or outcomes[name] == "skip" for name in required):
        return refuse(value, "test_missing")
    for key, scenario in value["scenarios"].items():
        if key == "H":
            continue
        missing = any(name not in outcomes or outcomes[name] == "skip" for name in scenario["tests"])
        failed = any(outcomes.get(name) == "fail" for name in scenario["tests"])
        scenario["result"] = "failed" if failed else "blocked" if missing else "passed"
        scenario["refusalCategories"] = ["test_failed"] if failed else ["test_missing"] if missing else []
    singles, graphs, previews, references, categories = [], [], [], [], set()
    for test, marker in markers:
        key, canonical = marker["scenario"], marker["canonical"]
        if key == "A" and test == JOURNEY and canonical.get("status") == "success" and isinstance(canonical.get("plan"), dict):
            plan = canonical["plan"]
            if plan.get("executionKind") == "single":
                singles.append(plan_digests(plan))
            elif plan.get("executionKind") == "graph":
                graphs.append(plan_digests(plan))
        if key == "B" and test == AUTHORING and canonical.get("category") == "previewed":
            authoring = canonical.get("workflowAuthoring", {})
            if not isinstance(authoring, dict):
                continue
            previews.append(authoring.get("previewDigest"))
            ref = authoring.get("reference")
            if isinstance(ref, dict):
                references.append(ref.get("digest"))
        if key == "F" and test in (JOURNEY, EXECUTABLE):
            category = canonical.get("category")
            if isinstance(category, str) and category in REFUSALS:
                categories.add(category)
    # Digests are copied from canonical results; never hash, substitute or repair.
    if len(singles) < 3 or not all(singles[:3]) or not graphs or not graphs[0] or not previews or not references or not all(isinstance(d, str) and HEX64.fullmatch(d) for d in previews + references):
        return refuse(value, "canonical_digest_missing")
    if singles[0]["planDigest"] != singles[1]["planDigest"] or singles[0]["planDigest"] == singles[2]["planDigest"]:
        return refuse(value, "determinism_mismatch", "failed")
    for key in "ACE":
        value["scenarios"][key]["canonicalDigests"] = singles[0].copy()
    value["scenarios"]["D"]["canonicalDigests"] = graphs[0]
    value["scenarios"]["B"]["canonicalDigests"] = {"workflowRefDigest": references[0], "previewDigests": list(dict.fromkeys(previews))}
    value["scenarios"]["E"]["canonicalDigests"].update(repeatedPlanDigests=[singles[0]["planDigest"], singles[1]["planDigest"]], driftedPlanDigest=singles[2]["planDigest"])
    value["scenarios"]["F"]["refusalCategories"] += sorted(categories)
    if not categories:
        return refuse(value, "canonical_digest_missing")
    results = [s["result"] for k, s in value["scenarios"].items() if k != "H"]
    value["r1"] = "failed" if "failed" in results else "blocked" if "blocked" in results else "passed"
    value["scenarios"]["H"]["result"] = value["r1"]
    return value


def collect(source, target, version=None, binary=None, run=command):
    value = report(target)
    # Never inherit credentials, Go hooks/flags, vendor configuration or prompts.
    env = {key: os.environ[key] for key in ("PATH", "SYSTEMROOT", "WINDIR", "TMPDIR", "TEMP", "TMP") if key in os.environ}
    env.update(GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull)
    gate_env = dict(env)
    if "HOME" in os.environ:
        gate_env["HOME"] = os.environ["HOME"]
    reason = provenance_gate(value, source, version, binary, gate_env, run)
    if reason:
        return refuse(value, reason)
    with tempfile.TemporaryDirectory(prefix="axiom-r1-") as directory:
        scratch = Path(directory)
        env.update(HOME=directory, USERPROFILE=directory, LINGO_STATE_ROOT=str(scratch / "state"), AXIOM_CODEX_SKILLS_ROOT=str(scratch / "skills"), CLAUDE_CONFIG_DIR=str(scratch / "claude"))
        code, wire = run(["go", "env", "-json", "GOMODCACHE", "GOCACHE", "GOROOT"], source, gate_env)
        if code in ("timeout", "output_bound"):
            return refuse(value, code, "failed")
        try:
            caches = json.loads(wire)
            if code != 0 or not all(isinstance(caches[key], str) and caches[key] for key in ("GOMODCACHE", "GOCACHE", "GOROOT")):
                raise ValueError()
            env.update({key: caches[key] for key in ("GOMODCACHE", "GOCACHE", "GOROOT")})
        except (ValueError, KeyError, TypeError, RecursionError):
            return refuse(value, "toolchain_missing")
        binary_path = str(scratch / ("axiom.exe" if os.name == "nt" else "axiom"))
        code, _ = run(["go", "build", "-o", binary_path, "./cmd/lingo"], source, env)
        if code != 0:
            return refuse(value, code if code in ("timeout", "output_bound") else "build_failed", "failed")
        # Status on an EMPTY root inspects embedded skill bytes without installation,
        # vendor discovery/authentication or any native Runtime process.
        code, wire = run([binary_path, "--json", "runtime", "codex", "status"], source, dict(env, PATH=str(scratch)))
        if code in ("timeout", "output_bound"):
            return refuse(value, code, "failed")
        try:
            runtime = json.loads(wire)["runtime"]
            skills = runtime["skills"]
            if code not in (0, 1) or not re.fullmatch(r"[0-9]+", runtime["skillSetVersion"]) or not skills or not all(s["name"] in SKILLS and HEX64.fullmatch(s["sha256"]) and s["state"] == "missing" for s in skills) or {s["name"] for s in skills} != SKILLS or len(skills) != len(SKILLS):
                raise ValueError()
            value["skillSet"] = {"version": runtime["skillSetVersion"], "skills": [{"name": s["name"], "sha256": s["sha256"]} for s in skills]}
        except (ValueError, KeyError, TypeError, RecursionError):
            return refuse(value, "skill_set_unverifiable")
        outcomes, markers = {}, []
        for package, names in PACKAGES.items():
            code, wire = run(["go", "test", "-json", "-count=1", "-run", "^(" + "|".join(names) + ")$", package], source, env)
            if code in ("timeout", "output_bound"):
                return refuse(value, code, "failed")
            if code == "unavailable":
                return refuse(value, "toolchain_missing")
            observed, evidence = parse_events(wire, names, "github.com/rgomids/axiom/" + package[2:])
            outcomes.update(observed)
            markers.extend(evidence)
            if code != 0 and not any(v == "fail" for v in observed.values()):
                return refuse(value, "test_failed", "failed")
        # Close the race: source drift during the run invalidates all Evidence.
        reason = provenance_gate(value, source, version, binary, gate_env, run)
        if reason:
            return refuse(value, reason)
        return collect_results(value, outcomes, markers)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target-revision", required=True)
    parser.add_argument("--target-version")
    parser.add_argument("--release-binary")
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args(argv)
    try:
        if args.output.exists() or args.output.is_symlink() or args.output.resolve().is_relative_to(args.source.resolve()):
            raise ValueError()
        value = collect(args.source, args.target_revision, args.target_version, args.release_binary)
        validate_schema(value)
        # Exclusive creation never replaces a previous receipt (including symlinks).
        with args.output.open("x", encoding="utf-8") as stream:
            json.dump(value, stream, sort_keys=True, indent=2)
            stream.write("\n")
    except (OSError, ValueError):
        print("r1_evidence_error=output_or_schema_refused")
        return 2
    print("r1=" + value["r1"])
    return {"passed": 0, "failed": 1, "blocked": 78}[value["r1"]]


if __name__ == "__main__":
    raise SystemExit(main())
