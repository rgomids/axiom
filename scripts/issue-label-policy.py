#!/usr/bin/env python3
"""Deterministic Issue label policy for rgomids/axiom.

Pure planner used by .github/workflows/issue-label-policy.yml. It never calls
the network: the workflow passes GitHub JSON in and applies the printed plan.

Subcommands:
  plan --issue FILE --action ACTION
      FILE is a REST Issue object. Prints {"skip", "add", "violations",
      "notices", "report"}. "add" only ever contains canonical labels; the
      planner never removes or renames a label and ignores axiom:* entirely.
  catalog-plan --labels FILE
      FILE is the repository label list (REST). Prints the create/update
      operations that make every canonical label exist with its color and
      description. Never deletes; never touches a non-canonical label.
"""

import argparse
import json
import re
import sys

RESERVED_PREFIX = "axiom:"
REPORT_MARKER = "<!-- issue-label-policy -->"
POLICY_LINK = "https://github.com/rgomids/axiom/blob/main/CONTRIBUTING.md#issue-labels"

TYPE_COLOR = "8250df"
AREA_COLOR = "0969da"
PLATFORM_COLOR = "bf8700"

# name -> (color, description). The single source of the canonical catalog.
CATALOG = {
    "type:epic": (TYPE_COLOR, "Editorial grouping of related Work Items, created deliberately."),
    "type:story": (TYPE_COLOR, "User-facing outcome or value."),
    "type:task": (TYPE_COLOR, "Bounded technical activity with a concrete result."),
    "type:bug": (TYPE_COLOR, "Defect or incorrect observable behavior."),
    "type:research": (TYPE_COLOR, "Question to answer before a decision or implementation."),
    "area:cli": (AREA_COLOR, "Work primarily related to the CLI commands and their contracts."),
    "area:skills": (AREA_COLOR, "Work primarily related to maintainer or runtime skills."),
    "area:work-item": (AREA_COLOR, "Work primarily related to Work Item intake, classification or projection."),
    "area:workflow": (AREA_COLOR, "Work primarily related to the workflow lifecycle and its stages."),
    "area:runtime": (AREA_COLOR, "Work primarily related to Runtime integration or resolution."),
    "area:execution": (AREA_COLOR, "Work primarily related to Execution and Evidence."),
    "area:installer": (AREA_COLOR, "Work primarily related to installation, bootstrap or upgrade."),
    "area:ci-cd": (AREA_COLOR, "Work primarily related to CI, release or delivery automation."),
    "area:governance": (AREA_COLOR, "Work primarily related to project governance, policies or process."),
    "status:planned": ("6e7781", "Legacy optional GitHub-only editorial stage; not automatically seeded."),
    "status:active": ("1a7f37", "Legacy optional GitHub-only editorial stage; Linear owns mapped work."),
    "status:blocked": ("cf222e", "Legacy optional GitHub-only editorial blocker; Linear owns dependencies."),
    "platform:windows": (PLATFORM_COLOR, "Specific to or impacting Windows."),
    "platform:linux": (PLATFORM_COLOR, "Specific to or impacting Linux."),
    "platform:macos": (PLATFORM_COLOR, "Specific to or impacting macOS."),
}

# Issue Form option -> label. The forms must offer exactly these options.
FORM_AREAS = {
    "CLI": "area:cli",
    "Skills": "area:skills",
    "Work Item": "area:work-item",
    "Workflow": "area:workflow",
    "Runtime": "area:runtime",
    "Execution": "area:execution",
    "Installer": "area:installer",
    "CI/CD": "area:ci-cd",
    "Governance": "area:governance",
}
FORM_PLATFORMS = {
    "Windows": "platform:windows",
    "Linux": "platform:linux",
    "macOS": "platform:macos",
}
NOT_PLATFORM_SPECIFIC = "Not platform-specific"
AREA_HEADING = "Area"
PLATFORM_HEADING = "Platform"
NO_RESPONSE = "_No response_"
MAX_AREAS = 2

# Retired vocabulary outside the governed families; release/phase is a milestone.
# status:* is retained for backward-compatible reading, never seeded or required.
RETIRED = {"scope:mvp", "bug", "enhancement", "cross-cutting"}
RETIRED_PREFIXES = ("slice:",)

FAMILIES = ("type", "area", "status", "platform")


def canonical(family):
    return sorted(name for name in CATALOG if name.startswith(family + ":"))


def family_of(label):
    head, sep, _ = label.partition(":")
    return head if sep and head in FAMILIES else None


def form_value(body, heading):
    """Value of the first `### heading` section of an Issue Form body, or None."""
    lines = (body or "").replace("\r\n", "\n").split("\n")
    target = "### " + heading
    for index, line in enumerate(lines):
        if line.rstrip() != target:  # form headings start at column 0
            continue
        value = []
        for following in lines[index + 1:]:
            if following.startswith("### "):
                break
            value.append(following)
        text = "\n".join(value).strip()
        return None if text in ("", NO_RESPONSE) else text
    return None


def form_choices(answer):
    """Options of a multi-select answer, whether GitHub renders them separated by
    commas, by lines or as a bullet list. Canonical options contain none of these
    separators (enforced by the tests), so splitting can never cut an option."""
    choices = []
    for line in answer.split("\n"):
        line = re.sub(r"^\s*[-*]\s+", "", line)
        choices += [part.strip() for part in line.split(",") if part.strip()]
    return choices


def code(text):
    return "`" + text.replace("`", "'") + "`"


def plan(issue, action):
    result = {"skip": None, "add": [], "violations": [], "notices": [], "report": ""}
    if issue.get("pull_request"):
        result["skip"] = "pull request"
        return result
    if issue.get("state") != "open":
        result["skip"] = "Issue is not open"
        return result

    labels = []
    for entry in issue.get("labels") or []:
        name = entry.get("name") if isinstance(entry, dict) else entry
        if isinstance(name, str) and name and not name.startswith(RESERVED_PREFIX):
            labels.append(name)
    present = set(labels)
    body = issue.get("body") or ""
    add = []
    violations = []
    notices = []

    def members(family):
        return sorted(label for label in present | set(add) if family_of(label) == family)

    for label in sorted(present):
        family = family_of(label)
        if family and label not in CATALOG:
            violations.append(f"{code(label)} is not a canonical `{family}:*` label.")
        elif label in RETIRED or label.startswith(RETIRED_PREFIXES):
            violations.append(f"{code(label)} is retired; use `type:*` for the nature and a milestone for release/phase.")

    # area: fill only an empty family from the form answer.
    area_answer = form_value(body, AREA_HEADING)
    if not members("area") and area_answer is not None:
        if area_answer in FORM_AREAS:
            add.append(FORM_AREAS[area_answer])
        else:
            notices.append(f"Area answer {code(area_answer[:80])} is not a canonical option.")

    # platform: optional; seeded from the form only when the Issue is opened.
    platform_answer = form_value(body, PLATFORM_HEADING)
    if action == "opened" and platform_answer is not None:
        choices = form_choices(platform_answer)
        unknown = [choice for choice in choices if choice not in FORM_PLATFORMS and choice != NOT_PLATFORM_SPECIFIC]
        if unknown:
            notices.append(f"Platform answer {code(platform_answer[:80])} has non-canonical options; no platform applied.")
        elif NOT_PLATFORM_SPECIFIC in choices and len(choices) > 1:
            notices.append(f"Platform answer combines {code(NOT_PLATFORM_SPECIFIC)} with platforms; no platform applied.")
        else:
            for choice in choices:
                label = FORM_PLATFORMS.get(choice)
                if label and label not in present and label not in add:
                    add.append(label)

    types = [label for label in members("type") if label in CATALOG]
    if not members("type"):
        violations.append("Missing `type:*`: exactly one of " + ", ".join(map(code, canonical("type"))) + " is required.")
    elif len(types) > 1:
        violations.append("Conflicting types " + ", ".join(map(code, types)) + ": exactly one `type:*` is required.")

    statuses = [label for label in members("status") if label in CATALOG]
    if len(statuses) > 1:
        violations.append("Conflicting legacy statuses " + ", ".join(map(code, statuses)) + ": at most one optional `status:*` is allowed.")

    areas = [label for label in members("area") if label in CATALOG]
    if not members("area") and types != ["type:epic"]:
        violations.append("Missing `area:*`: select one primary area (an Epic may omit it).")
    elif len(areas) > MAX_AREAS:
        violations.append("Too many areas " + ", ".join(map(code, areas)) + f": use one primary area, at most {MAX_AREAS}.")

    result["add"] = add
    result["violations"] = violations
    result["notices"] = notices
    result["report"] = report(violations)
    return result


def report(violations):
    if not violations:
        return REPORT_MARKER + "\n**Issue label policy:** satisfied.\n"
    lines = [
        REPORT_MARKER,
        "**Issue label policy:** this Issue does not satisfy the [label taxonomy](" + POLICY_LINK + ").",
        "No label was removed or replaced automatically; a maintainer should resolve:",
        "",
    ]
    lines += ["- " + violation for violation in violations]
    return "\n".join(lines) + "\n"


def catalog_plan(existing):
    current = {}
    for entry in existing:
        if isinstance(entry, dict) and isinstance(entry.get("name"), str):
            current[entry["name"]] = entry
    operations = []
    for name in sorted(CATALOG):
        color, description = CATALOG[name]
        label = current.get(name)
        if label is None:
            operations.append({"method": "create", "name": name, "color": color, "description": description})
        elif (label.get("color") or "").lower() != color or (label.get("description") or "") != description:
            operations.append({"method": "update", "name": name, "color": color, "description": description})
    return operations


def read_json(path):
    with open(path, encoding="utf-8") as handle:
        return json.load(handle)


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    commands = parser.add_subparsers(dest="command", required=True)
    plan_parser = commands.add_parser("plan")
    plan_parser.add_argument("--issue", required=True)
    plan_parser.add_argument("--action", required=True)
    catalog_parser = commands.add_parser("catalog-plan")
    catalog_parser.add_argument("--labels", required=True)
    arguments = parser.parse_args(argv)

    if arguments.command == "plan":
        if not re.fullmatch(r"[a-z_]{1,32}", arguments.action):
            parser.error("invalid --action")
        issue = read_json(arguments.issue)
        if not isinstance(issue, dict):
            parser.error("--issue must contain a JSON object")
        output = plan(issue, arguments.action)
    else:
        labels = read_json(arguments.labels)
        if not isinstance(labels, list):
            parser.error("--labels must contain a JSON array")
        output = catalog_plan(labels)
    json.dump(output, sys.stdout, indent=2, sort_keys=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
