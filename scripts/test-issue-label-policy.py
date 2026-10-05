#!/usr/bin/env python3
"""Offline tests for scripts/issue-label-policy.py, the Issue Forms and the workflow."""

import importlib.util
import json
import os
import re
import subprocess
import sys
import tempfile
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCRIPT = os.path.join(ROOT, "scripts", "issue-label-policy.py")
TEMPLATES = os.path.join(ROOT, ".github", "ISSUE_TEMPLATE")
WORKFLOW = os.path.join(ROOT, ".github", "workflows", "issue-label-policy.yml")
FORMS = {"bug.yml": "type:bug", "story.yml": "type:story", "task.yml": "type:task", "research.yml": "type:research"}

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("issue_label_policy", SCRIPT)
policy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(policy)


def read(path):
    with open(path, encoding="utf-8") as handle:
        return handle.read()


def form_labels(text):
    match = re.search(r"^labels:\n((?:  - .+\n)+)", text, re.MULTILINE)
    return [line.strip()[2:] for line in match.group(1).splitlines()] if match else []


def form_fields(text):
    """(id, type, label, options) of every body element, in order."""
    fields = []
    current = None
    in_options = False
    for line in text.splitlines():
        element = re.match(r"^  - type: (\S+)$", line)
        if element:
            current = {"type": element.group(1), "id": None, "label": None, "options": []}
            fields.append(current)
            in_options = False
            continue
        if current is None:
            continue
        if re.match(r"^    id: ", line):
            current["id"] = line.split(":", 1)[1].strip()
        elif re.match(r"^      label: ", line):
            current["label"] = line.split(":", 1)[1].strip()
        elif re.match(r"^      options:$", line):
            in_options = True
        elif in_options and re.match(r"^        - ", line):
            current["options"].append(line.strip()[2:])
        elif not re.match(r"^        ", line):
            in_options = False
    return fields


def render(sections):
    """Issue body as GitHub renders an Issue Form submission."""
    return "\n\n".join(f"### {heading}\n\n{value}" for heading, value in sections)


def issue(labels, body="", state="open", **extra):
    value = {"state": state, "body": body, "labels": [{"name": name} for name in labels]}
    value.update(extra)
    return value


class PlanTest(unittest.TestCase):
    def test_bug_form_gets_area_and_platforms(self):
        body = render([("Area", "Installer"), ("Platform", "Windows, macOS"), ("Description", "x")])
        result = policy.plan(issue(["type:bug", "status:planned"], body), "opened")
        self.assertEqual(result["add"], ["area:installer", "platform:windows", "platform:macos"])
        self.assertEqual(result["violations"], [])

    def test_every_area_option_maps(self):
        for option, label in policy.FORM_AREAS.items():
            body = render([("Area", option), ("Platform", "Linux")])
            result = policy.plan(issue(["type:task", "status:planned"], body), "opened")
            self.assertEqual(result["add"], [label, "platform:linux"], option)

    def test_platform_only_seeded_when_opened(self):
        body = render([("Area", "CLI"), ("Platform", "Windows")])
        result = policy.plan(issue(["type:bug", "status:planned", "area:cli"], body), "unlabeled")
        self.assertEqual(result["add"], [])

    def test_not_platform_specific(self):
        alone = policy.plan(issue(["type:bug", "status:planned", "area:cli"], render([("Platform", "Not platform-specific")])), "opened")
        self.assertEqual((alone["add"], alone["notices"]), ([], []))
        mixed = policy.plan(issue(["type:bug", "status:planned", "area:cli"], render([("Platform", "Not platform-specific, Linux")])), "opened")
        self.assertEqual(mixed["add"], [])
        self.assertEqual(len(mixed["notices"]), 1)

    def test_existing_area_is_never_replaced(self):
        body = render([("Area", "CLI")])
        result = policy.plan(issue(["type:story", "status:active", "area:runtime"], body), "opened")
        self.assertEqual(result["add"], [])
        self.assertEqual(result["violations"], [])

    def test_first_area_section_wins_over_injected_heading(self):
        body = render([("Area", "Skills"), ("Description", "### Area\n\nRuntime")])
        self.assertEqual(policy.plan(issue(["type:task", "status:planned"], body), "opened")["add"], ["area:skills"])

    def test_unknown_area_answer_is_a_notice_not_a_label(self):
        result = policy.plan(issue(["type:task", "status:planned"], render([("Area", "area:anything")])), "opened")
        self.assertEqual(result["add"], [])
        self.assertTrue(result["notices"])
        self.assertTrue(any("Missing `area:*`" in violation for violation in result["violations"]))

    def test_missing_status_defaults_to_planned(self):
        result = policy.plan(issue(["type:task", "area:cli"]), "labeled")
        self.assertEqual(result["add"], ["status:planned"])
        self.assertEqual(result["violations"], [])

    def test_non_canonical_status_is_reported_not_replaced(self):
        result = policy.plan(issue(["type:task", "area:cli", "status:done"]), "reopened")
        self.assertEqual(result["add"], [])
        self.assertEqual(len(result["violations"]), 1)

    def test_conflicts_are_reported_without_choosing(self):
        result = policy.plan(issue(["type:bug", "type:task", "status:planned", "status:active", "area:cli"]), "labeled")
        self.assertEqual(result["add"], [])
        self.assertTrue(any("Conflicting types" in violation for violation in result["violations"]))
        self.assertTrue(any("Conflicting statuses" in violation for violation in result["violations"]))

    def test_missing_type_is_reported(self):
        result = policy.plan(issue(["status:planned", "area:cli"]), "opened")
        self.assertTrue(any("Missing `type:*`" in violation for violation in result["violations"]))

    def test_epic_may_omit_area(self):
        self.assertEqual(policy.plan(issue(["type:epic", "status:active"]), "labeled")["violations"], [])
        self.assertTrue(policy.plan(issue(["type:story", "status:active"]), "labeled")["violations"])

    def test_area_limit(self):
        two = policy.plan(issue(["type:task", "status:planned", "area:cli", "area:runtime"]), "labeled")
        self.assertEqual(two["violations"], [])
        three = policy.plan(issue(["type:task", "status:planned", "area:cli", "area:runtime", "area:skills"]), "labeled")
        self.assertEqual(len(three["violations"]), 1)

    def test_retired_and_non_canonical_labels_are_reported(self):
        labels = ["type:task", "status:planned", "area:cli", "scope:mvp", "bug", "enhancement", "cross-cutting", "slice:s8", "type:slice", "platform:bsd"]
        result = policy.plan(issue(labels), "labeled")
        self.assertEqual(result["add"], [])
        self.assertEqual(len(result["violations"]), 7)

    def test_axiom_namespace_is_ignored(self):
        labels = ["axiom:stage:specifying", "axiom:blocked", "axiom:type:weird", "type:task", "status:planned", "area:cli"]
        result = policy.plan(issue(labels, render([("Area", "CLI")])), "opened")
        self.assertEqual(result["violations"], [])
        self.assertFalse(any(label.startswith("axiom:") for label in result["add"]))
        self.assertNotIn("axiom:", result["report"].replace("<!-- issue-label-policy -->", ""))

    def test_foreign_labels_are_preserved(self):
        result = policy.plan(issue(["type:task", "status:planned", "area:cli", "documentation", "good first issue", "autorelease: pending"]), "labeled")
        self.assertEqual((result["add"], result["violations"]), ([], []))
        self.assertNotIn("remove", result)

    def test_skips(self):
        self.assertEqual(policy.plan(issue([], state="closed"), "labeled")["skip"], "Issue is not open")
        self.assertEqual(policy.plan(issue([], pull_request={"url": "x"}), "opened")["skip"], "pull request")

    def test_report(self):
        self.assertTrue(policy.report([]).startswith(policy.REPORT_MARKER))
        report = policy.plan(issue(["status:planned", "area:cli", "type:`x`"]), "labeled")["report"]
        self.assertTrue(report.startswith(policy.REPORT_MARKER + "\n"))
        self.assertIn("`type:'x'`", report)


class CatalogTest(unittest.TestCase):
    def test_catalog_shape(self):
        families = {"type": 5, "area": 9, "status": 3, "platform": 3}
        for family, count in families.items():
            self.assertEqual(len(policy.canonical(family)), count, family)
        self.assertEqual(len(policy.CATALOG), sum(families.values()))
        for name, (color, description) in policy.CATALOG.items():
            self.assertRegex(color, r"^[0-9a-f]{6}$", name)
            self.assertTrue(0 < len(description) <= 100, name)
            self.assertFalse(name.startswith("axiom:"))
        self.assertNotIn("status:done", policy.CATALOG)
        self.assertEqual(set(policy.FORM_AREAS.values()), set(policy.canonical("area")))
        self.assertEqual(set(policy.FORM_PLATFORMS.values()), set(policy.canonical("platform")))

    def test_catalog_plan_creates_updates_and_never_deletes(self):
        color, description = policy.CATALOG["type:bug"]
        existing = [
            {"name": "type:bug", "color": color.upper(), "description": description},
            {"name": "area:cli", "color": "ededed", "description": ""},
            {"name": "axiom:stage:reviewing", "color": "5319e7", "description": "Axiom current workflow stage"},
            {"name": "scope:mvp", "color": "ededed", "description": ""},
        ]
        operations = policy.catalog_plan(existing)
        by_name = {operation["name"]: operation["method"] for operation in operations}
        self.assertNotIn("type:bug", by_name)
        self.assertEqual(by_name["area:cli"], "update")
        self.assertEqual(by_name["platform:macos"], "create")
        self.assertEqual(len(operations), len(policy.CATALOG) - 1)
        self.assertTrue(all(operation["method"] in ("create", "update") for operation in operations))
        self.assertTrue(all(operation["name"] in policy.CATALOG for operation in operations))


class FormsTest(unittest.TestCase):
    def test_form_set(self):
        self.assertEqual(sorted(os.listdir(TEMPLATES)), sorted(list(FORMS) + ["config.yml"]))
        self.assertRegex(read(os.path.join(TEMPLATES, "config.yml")), r"(?m)^blank_issues_enabled: false$")

    def test_forms_match_policy(self):
        for name, type_label in FORMS.items():
            text = read(os.path.join(TEMPLATES, name))
            self.assertEqual(form_labels(text), [type_label, "status:planned"], name)
            fields = form_fields(text)
            self.assertEqual(fields[0]["type"], "markdown", name)
            area = fields[1]
            # Classification precedes free text, so the first section heading is the real answer.
            self.assertEqual((area["type"], area["id"], area["label"]), ("dropdown", "area", policy.AREA_HEADING), name)
            self.assertEqual(area["options"], list(policy.FORM_AREAS), name)
            platform = [field for field in fields if field["id"] == "platform"]
            if name == "research.yml":
                self.assertEqual(platform, [], name)
            else:
                self.assertIs(platform[0], fields[2], name)
                self.assertEqual(platform[0]["label"], policy.PLATFORM_HEADING, name)
                self.assertEqual(platform[0]["options"], [policy.NOT_PLATFORM_SPECIFIC] + list(policy.FORM_PLATFORMS), name)
            ids = [field["id"] for field in fields if field["id"]]
            self.assertEqual(len(ids), len(set(ids)), name)
            self.assertIn("confirmations", ids, name)

    def test_submissions_produce_canonical_labels(self):
        for name, type_label in FORMS.items():
            sections = [("Area", "Runtime")]
            if name != "research.yml":
                sections.append(("Platform", "Linux"))
            sections.append(("Context", "synthetic"))
            result = policy.plan(issue(form_labels(read(os.path.join(TEMPLATES, name))), render(sections)), "opened")
            expected = ["area:runtime"] + (["platform:linux"] if name != "research.yml" else [])
            self.assertEqual(result["add"], expected, name)
            self.assertEqual(result["violations"], [], name)


class WorkflowTest(unittest.TestCase):
    def test_workflow_contract(self):
        text = read(WORKFLOW)
        self.assertRegex(text, r"(?m)^permissions: \{\}$")
        self.assertNotIn("pull_request_target", text)
        self.assertNotIn("issue_comment", text)
        self.assertRegex(text, r"(?m)^    types: \[opened, reopened, labeled, unlabeled\]$")
        for uses in re.findall(r"uses: (\S+)", text):
            self.assertRegex(uses, r"@[0-9a-f]{40}$")
        # Untrusted Issue text never reaches an expression.
        for field in ("body", "title", "labels", "user"):
            self.assertNotIn("github.event.issue." + field, text)
        self.assertEqual(re.findall(r"(?m)^      (\w[\w-]*): (\w+)$", text.split("permissions: {}", 1)[1]).count(("issues", "write")), 3)
        self.assertNotIn("DELETE", text)
        self.assertIn("persist-credentials: false", text)


class CommandLineTest(unittest.TestCase):
    def run_script(self, *arguments):
        return subprocess.run([sys.executable, SCRIPT, *arguments], capture_output=True, text=True, check=False)

    def test_plan_and_catalog_commands(self):
        with tempfile.TemporaryDirectory() as directory:
            issue_path = os.path.join(directory, "issue.json")
            labels_path = os.path.join(directory, "labels.json")
            with open(issue_path, "w", encoding="utf-8") as handle:
                json.dump(issue(["type:bug", "status:planned"], render([("Area", "CI/CD"), ("Platform", "Linux")])), handle)
            with open(labels_path, "w", encoding="utf-8") as handle:
                json.dump([], handle)
            planned = self.run_script("plan", "--issue", issue_path, "--action", "opened")
            self.assertEqual(planned.returncode, 0, planned.stderr)
            self.assertEqual(json.loads(planned.stdout)["add"], ["area:ci-cd", "platform:linux"])
            catalog = self.run_script("catalog-plan", "--labels", labels_path)
            self.assertEqual(catalog.returncode, 0, catalog.stderr)
            self.assertEqual(len(json.loads(catalog.stdout)), len(policy.CATALOG))
            refused = self.run_script("plan", "--issue", issue_path, "--action", "opened; rm")
            self.assertEqual(refused.returncode, 2)


if __name__ == "__main__":
    result = unittest.main(exit=False, verbosity=1).result
    if not result.wasSuccessful():
        sys.exit(1)
    print("PASS: issue label policy tests")
