#!/usr/bin/env python3
"""Offline black-box tests of check-automation-registry.py; fixtures never touch the network."""
import copy
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

CHECK = Path(__file__).resolve().with_name("check-automation-registry.py")
REGISTRY = "scripts/automation-registry.json"


def entry(path, **overrides):
    value = {
        "path": path,
        "kind": "workflow" if path.startswith(".github/") else "script",
        "owner": "repository-validation",
        "purpose": "Fixture automation.",
        "lifecycle": "durable",
        "disposition": "keep",
        "disposition_ref": None,
        "runtime": ["bash"],
        "scope": ["ci"],
        "trigger": "every fixture validation",
        "replay_reason": None,
        "callers": [],
        "inputs": "none",
        "outputs": "exit status",
        "local_effects": "none",
        "network_effects": "none",
        "authority": "none",
        "tests": [],
        "evidence": None,
        "compatibility": [],
        "removal_condition": None,
    }
    value.update(overrides)
    return value


VALID = {
    "schema_version": 1,
    "policy": ".agents/policies/repository-automation.md",
    "capabilities": {
        "repository-validation": "Fixture repository checks.",
        "release": "Fixture release automation.",
    },
    "excluded_scopes": [
        {"path": "internal/", "extensions": [".go"], "reason": "Fixture product code."},
        {"path": "site/", "extensions": [".js"], "reason": "Fixture site code."},
    ],
    "entries": [
        entry(".github/workflows/ci.yml", runtime=["github-actions"], callers=[]),
        entry("scripts/test-tool.sh", scope=["ci", "test"], callers=[".github/workflows/ci.yml"]),
        entry("scripts/tool.sh", owner="release", callers=["scripts/test-tool.sh"],
              tests=["scripts/test-tool.sh"],
              compatibility=[{"relation": "pinned source", "revision": "0" * 40},
                             {"relation": "reads", "path": "docs/notes.md"}]),
    ],
}


class RegistryCheckTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="axiom-automation-registry-")
        self.root = Path(self.temp.name)
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)
        self.write(".agents/policies/repository-automation.md", "# Policy\n")
        self.write("scripts/tool.sh", "#!/usr/bin/env bash\necho tool\n", executable=True)
        self.write("scripts/test-tool.sh", "#!/usr/bin/env bash\nscripts/tool.sh\n", executable=True)
        self.write(".github/workflows/ci.yml", "on: push\n")
        self.write("internal/app/main.go", "package main\n")
        self.write("site/app.js", "console.log('site');\n")
        self.write("docs/notes.md", "notes\n")
        self.registry = copy.deepcopy(VALID)
        self.save()

    def tearDown(self):
        self.temp.cleanup()

    def write(self, relative, content, executable=False):
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        path.chmod(0o755 if executable else 0o644)

    def save(self, raw=None):
        (self.root / REGISTRY).write_text(raw if raw is not None else json.dumps(self.registry, indent=2) + "\n",
                                          encoding="utf-8")

    def entry(self, path):
        return next(e for e in self.registry["entries"] if e["path"] == path)

    def run_check(self):
        result = subprocess.run([sys.executable, str(CHECK), str(self.root)],
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=False)
        return result.returncode, result.stdout, result.stderr

    def assert_passes(self):
        code, stdout, stderr = self.run_check()
        self.assertEqual(code, 0, stderr)
        self.assertIn("PASS: automation registry covers all", stdout)

    def assert_fails(self, message):
        self.save()
        code, _, stderr = self.run_check()
        self.assertEqual(code, 1, f"expected failure containing {message!r}")
        self.assertIn(message, stderr)

    def test_valid_registry_passes(self):
        self.assert_passes()

    def test_excluded_product_and_ignored_files_are_not_governed(self):
        self.write(".gitignore", "tmp/\n")
        self.write("tmp/scratch-helper.sh", "#!/usr/bin/env bash\n", executable=True)
        self.assert_passes()

    def test_unregistered_helper_fails(self):
        self.write("scripts/one-off-fix.py", "print('fix')\n")
        self.assert_fails("unregistered automation surface: scripts/one-off-fix.py")

    def test_unregistered_executable_without_extension_fails(self):
        self.write("docs/run-me", "echo hi\n", executable=True)
        self.assert_fails("unregistered automation surface: docs/run-me")

    def test_unregistered_shebang_file_fails(self):
        self.write("docs/collect", "#!/bin/sh\necho hi\n")
        self.assert_fails("unregistered automation surface: docs/collect")

    def test_shell_helper_inside_excluded_product_scope_is_governed(self):
        self.write("internal/app/regenerate.sh", "echo regenerate\n")
        self.assert_fails("unregistered automation surface: internal/app/regenerate.sh")

    def test_registered_missing_path_fails(self):
        self.registry["entries"].append(entry("scripts/zz-removed.sh"))
        self.assert_fails("entry scripts/zz-removed.sh: registered path does not exist")

    def test_missing_owner_fails(self):
        del self.entry("scripts/tool.sh")["owner"]
        self.assert_fails("entry scripts/tool.sh: missing required field(s): owner")

    def test_undeclared_owner_fails(self):
        self.entry("scripts/tool.sh")["owner"] = "alice"
        self.assert_fails("owner 'alice' is not a declared capability")

    def test_missing_purpose_fails(self):
        self.entry("scripts/tool.sh")["purpose"] = "  "
        self.assert_fails("entry scripts/tool.sh: purpose must be a non-empty string")

    def test_unknown_field_fails(self):
        self.entry("scripts/tool.sh")["purpse"] = "typo"
        self.assert_fails("unknown field(s): purpse")

    def test_invalid_lifecycle_fails(self):
        self.entry("scripts/tool.sh")["lifecycle"] = "sometimes"
        self.assert_fails("invalid lifecycle 'sometimes'")

    def test_ephemeral_lifecycle_is_rejected(self):
        self.entry("scripts/tool.sh")["lifecycle"] = "ephemeral"
        self.assert_fails("lifecycle 'ephemeral' is ephemeral; ephemeral helpers are not registered")

    def test_durable_without_trigger_fails(self):
        self.entry("scripts/tool.sh")["trigger"] = None
        self.assert_fails("durable automation must declare its recurring trigger")

    def test_durable_with_replay_reason_fails(self):
        self.entry("scripts/tool.sh")["replay_reason"] = "reproduces a past incident"
        self.assert_fails("durable automation must not declare a replay_reason")

    def test_historical_replay_requires_rationale_and_removal(self):
        tool = self.entry("scripts/tool.sh")
        tool.update(lifecycle="historical-replay", trigger=None)
        self.assert_fails("historical-replay automation must declare replay_reason")
        self.assert_fails("historical-replay automation must declare removal_condition")
        tool.update(replay_reason="Reproduces the pinned fixture.", removal_condition="When the fixture retires.")
        self.save()
        self.assert_passes()

    def test_non_keep_disposition_requires_reference_and_removal(self):
        self.entry("scripts/tool.sh")["disposition"] = "move"
        self.assert_fails("disposition 'move' requires disposition_ref")
        self.assert_fails("disposition 'move' requires removal_condition")

    def test_invalid_scope_and_runtime_fail(self):
        tool = self.entry("scripts/tool.sh")
        tool["scope"] = ["everyone"]
        tool["runtime"] = []
        self.assert_fails("invalid scope value 'everyone'")
        self.assert_fails("runtime must be a non-empty list")

    def test_duplicate_registration_fails(self):
        self.registry["entries"].append(entry("scripts/tool.sh"))
        self.assert_fails("duplicate registration: scripts/tool.sh")

    def test_duplicate_json_key_fails(self):
        raw = json.dumps(self.registry, indent=2).replace('"schema_version": 1,', '"schema_version": 1, "schema_version": 1,')
        self.save(raw)
        code, _, stderr = self.run_check()
        self.assertEqual(code, 1)
        self.assertIn("duplicate JSON key: schema_version", stderr)

    def test_unsorted_entries_fail(self):
        self.registry["entries"].reverse()
        self.assert_fails("entries must be sorted by path")

    def test_dangling_local_caller_fails(self):
        self.entry("scripts/tool.sh")["callers"] = ["scripts/deleted-caller.sh"]
        self.assert_fails("declared caller does not exist: scripts/deleted-caller.sh")

    def test_dangling_test_fails(self):
        self.entry("scripts/tool.sh")["tests"] = ["scripts/test-missing.sh"]
        self.assert_fails("declared test does not exist: scripts/test-missing.sh")

    def test_invalid_compatibility_revision_fails(self):
        self.entry("scripts/tool.sh")["compatibility"] = [{"relation": "pinned source", "revision": "abc123"}]
        self.assert_fails("compatibility revision must be a full 40-character lowercase SHA")

    def test_invalid_compatibility_path_and_shape_fail(self):
        self.entry("scripts/tool.sh")["compatibility"] = [{"relation": "reads", "path": "docs/missing.md"},
                                                         {"relation": "no target"}]
        self.assert_fails("compatibility path does not exist: 'docs/missing.md'")
        self.assert_fails("compatibility pin must declare relation and a revision and/or path")

    def test_path_traversal_and_absolute_paths_fail(self):
        tool = self.entry("scripts/tool.sh")
        tool["callers"] = ["../outside.sh", "/etc/passwd"]
        self.assert_fails("callers entry is not a safe repository-relative path: '../outside.sh'")
        self.assert_fails("callers entry is not a safe repository-relative path: '/etc/passwd'")

    def test_symlinked_registered_path_fails(self):
        os.symlink("tool.sh", self.root / "scripts/alias.sh")
        self.registry["entries"].append(entry("scripts/alias.sh"))
        self.registry["entries"].sort(key=lambda e: e["path"])
        self.assert_fails("entry scripts/alias.sh: registered path must be a regular file inside the repository")

    def test_workflow_kind_is_reserved(self):
        self.entry("scripts/tool.sh")["kind"] = "workflow"
        self.assert_fails("kind 'workflow' is reserved for .github/workflows and .github/actions")

    def test_excluded_scope_cannot_cover_governed_roots(self):
        self.registry["excluded_scopes"].append({"path": "scripts/", "extensions": [".sh"], "reason": "hide"})
        self.assert_fails("excluded_scopes[2]: path must not cover or be covered by scripts/, .github/")

    def test_registering_non_automation_fails(self):
        self.registry["entries"].insert(0, entry("docs/notes.md"))
        self.assert_fails("registered path is not a governed automation surface")

    def test_registry_values_are_never_executed(self):
        marker = self.root / "executed"
        self.entry("scripts/tool.sh")["inputs"] = f"$(touch {marker})"
        self.entry("scripts/tool.sh")["purpose"] = f"`touch {marker}`"
        self.save()
        self.assert_passes()
        self.assertFalse(marker.exists())


if __name__ == "__main__":
    stream = io.StringIO()
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(RegistryCheckTests)
    result = unittest.TextTestRunner(stream=stream, verbosity=2).run(suite)
    if not result.wasSuccessful():
        sys.stderr.write(stream.getvalue())
        sys.exit(1)
    print(f"PASS: automation registry checker behaves as specified ({result.testsRun} cases)")
