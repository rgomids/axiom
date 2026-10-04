#!/usr/bin/env python3
"""Validate the durable repository automation registry (Issue #168).

    scripts/check-automation-registry.py [ROOT]

Checks only objective structure defined by
.agents/policies/repository-automation.md: every governed automation surface
is registered, registered paths exist, entries are unique and complete, closed
vocabularies hold, and declared callers, tests and compatibility pins resolve
locally. It runs offline, never executes registered automation and never
evaluates registry values. Exit status is non-zero on any violation.
"""

from __future__ import annotations

import json
import os
import posixpath
import re
import stat
import subprocess
import sys
from pathlib import Path

REGISTRY = "scripts/automation-registry.json"
POLICY_DOC = ".agents/policies/repository-automation.md"

TOP_LEVEL = ("schema_version", "policy", "capabilities", "excluded_scopes", "entries")
FIELDS = (
    "path", "kind", "owner", "purpose", "lifecycle", "disposition", "disposition_ref",
    "runtime", "scope", "trigger", "replay_reason", "callers", "inputs", "outputs",
    "local_effects", "network_effects", "authority", "tests", "evidence",
    "compatibility", "removal_condition",
)
REQUIRED_TEXT = (
    "path", "kind", "owner", "purpose", "lifecycle", "disposition", "inputs",
    "outputs", "local_effects", "network_effects", "authority",
)
NULLABLE_TEXT = ("disposition_ref", "trigger", "replay_reason", "evidence", "removal_condition")

KINDS = {"script", "workflow"}
LIFECYCLES = {"durable", "historical-replay"}
EPHEMERAL_LIFECYCLES = {"ephemeral", "temporary", "one-off", "oneoff", "transient"}
DISPOSITIONS = {"keep", "move", "merge", "replace", "delete"}
RUNTIMES = {"bash", "posix-sh", "python3", "powershell", "go", "github-actions"}
SCOPES = {"public", "maintainer", "ci", "test", "historical"}
GOVERNED_EXTENSIONS = {
    ".sh", ".bash", ".zsh", ".py", ".ps1", ".psm1", ".go", ".rb", ".pl",
    ".js", ".mjs", ".cjs", ".ts",
}
PROTECTED_SCOPES = ("scripts/", ".github/")
CAPABILITY_NAME = re.compile(r"^[a-z][a-z0-9-]*$")
REVISION = re.compile(r"^[0-9a-f]{40}$")


class DuplicateKey(ValueError):
    pass


def reject_duplicate_keys(pairs):
    seen = {}
    for key, value in pairs:
        if key in seen:
            raise DuplicateKey(key)
        seen[key] = value
    return seen


def is_text(value) -> bool:
    return isinstance(value, str) and value.strip() != ""


def safe_relative(value) -> bool:
    """Repository-relative POSIX path with no traversal, absolute or odd segments."""
    if not is_text(value) or value != value.strip():
        return False
    if value.startswith("/") or "\\" in value or "\0" in value:
        return False
    trimmed = value[:-1] if value.endswith("/") else value
    return posixpath.normpath(trimmed) == trimmed and not trimmed.startswith("../") \
        and trimmed not in ("", ".", "..")


def inside(root: Path, relative: str) -> bool:
    real = os.path.realpath(root / relative)
    return real == str(root) or real.startswith(str(root) + os.sep)


def resolves(root: Path, relative: str, want_file: bool) -> bool:
    candidate = root / relative
    if not os.path.lexists(candidate) or not inside(root, relative):
        return False
    return candidate.is_file() if want_file else candidate.exists()


def listed_files(root: Path) -> list[str]:
    result = subprocess.run(
        ["git", "-C", str(root), "ls-files", "-z", "--cached", "--others", "--exclude-standard"],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
    )
    if result.returncode != 0:
        raise RuntimeError("target is not a readable git work tree")
    return sorted({p for p in result.stdout.decode("utf-8", "surrogateescape").split("\0") if p})


def excluded(relative: str, scopes: list[dict]) -> bool:
    suffix = posixpath.splitext(relative)[1]
    return any(relative.startswith(s["path"]) and suffix in s["extensions"] for s in scopes)


def governed(root: Path, relative: str, scopes: list[dict]) -> bool:
    path = root / relative
    if os.path.islink(path) or not path.is_file() or excluded(relative, scopes):
        return False
    if relative.startswith(".github/workflows/") and relative.endswith((".yml", ".yaml")):
        return True
    if relative.startswith(".github/actions/") and posixpath.basename(relative) in ("action.yml", "action.yaml"):
        return True
    if posixpath.splitext(relative)[1] in GOVERNED_EXTENSIONS:
        return True
    if os.stat(path).st_mode & (stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH):
        return True
    with open(path, "rb") as handle:
        return handle.read(2) == b"#!"


def check_scopes(root: Path, scopes, errors: list[str]) -> list[dict]:
    valid: list[dict] = []
    if not isinstance(scopes, list):
        errors.append("excluded_scopes must be a list")
        return valid
    for index, scope in enumerate(scopes):
        label = f"excluded_scopes[{index}]"
        if not isinstance(scope, dict) or set(scope) != {"path", "extensions", "reason"}:
            errors.append(f"{label}: must declare exactly path, extensions and reason")
            continue
        path, extensions = scope["path"], scope["extensions"]
        problems = []
        if not safe_relative(path) or not path.endswith("/") or not (root / path).is_dir() \
                or os.path.islink(root / path.rstrip("/")) or not inside(root, path):
            problems.append("path must be an existing repository directory ending with '/'")
        elif any(path.startswith(p) or p.startswith(path) for p in PROTECTED_SCOPES):
            problems.append(f"path must not cover or be covered by {', '.join(PROTECTED_SCOPES)}")
        if not isinstance(extensions, list) or not extensions \
                or any(e not in GOVERNED_EXTENSIONS for e in extensions) \
                or len(set(extensions)) != len(extensions):
            problems.append("extensions must be a non-empty list of distinct governed extensions")
        if not is_text(scope["reason"]):
            problems.append("reason is required")
        if problems:
            errors.extend(f"{label}: {p}" for p in problems)
        else:
            valid.append(scope)
    return valid


def check_paths(root: Path, label: str, field: str, values, want_file: bool, errors: list[str]) -> None:
    if not isinstance(values, list):
        errors.append(f"{label}: {field} must be a list")
        return
    for value in values:
        if not safe_relative(value):
            errors.append(f"{label}: {field} entry is not a safe repository-relative path: {value!r}")
        elif not resolves(root, value, want_file):
            errors.append(f"{label}: declared {field[:-1]} does not exist: {value}")
    if len(set(map(str, values))) != len(values):
        errors.append(f"{label}: {field} contains duplicates")


def check_vocabulary(label: str, field: str, values, allowed: set, errors: list[str]) -> None:
    if not isinstance(values, list) or not values:
        errors.append(f"{label}: {field} must be a non-empty list")
        return
    for value in values:
        if value not in allowed:
            errors.append(f"{label}: invalid {field} value {value!r} (allowed: {', '.join(sorted(allowed))})")
    if len(set(map(str, values))) != len(values):
        errors.append(f"{label}: {field} contains duplicates")


def check_entry(root: Path, entry, index: int, capabilities: dict, errors: list[str]) -> str | None:
    if not isinstance(entry, dict):
        errors.append(f"entries[{index}]: must be an object")
        return None
    path = entry.get("path")
    label = f"entry {path}" if is_text(path) else f"entries[{index}]"
    unknown = sorted(set(entry) - set(FIELDS))
    missing = [f for f in FIELDS if f not in entry]
    if unknown:
        errors.append(f"{label}: unknown field(s): {', '.join(unknown)}")
    if missing:
        errors.append(f"{label}: missing required field(s): {', '.join(missing)}")
    for field in REQUIRED_TEXT:
        if field in entry and not is_text(entry[field]):
            errors.append(f"{label}: {field} must be a non-empty string")
    for field in NULLABLE_TEXT:
        if field in entry and entry[field] is not None and not is_text(entry[field]):
            errors.append(f"{label}: {field} must be null or a non-empty string")

    if is_text(path):
        if not safe_relative(path) or path.endswith("/"):
            errors.append(f"{label}: path is not a safe repository-relative file path")
            path = None
        elif not os.path.lexists(root / path):
            errors.append(f"{label}: registered path does not exist")
        elif os.path.islink(root / path) or not (root / path).is_file() or not inside(root, path):
            errors.append(f"{label}: registered path must be a regular file inside the repository")

    kind = entry.get("kind")
    if is_text(kind):
        if kind not in KINDS:
            errors.append(f"{label}: invalid kind {kind!r} (allowed: {', '.join(sorted(KINDS))})")
        elif is_text(path):
            workflow_path = path.startswith((".github/workflows/", ".github/actions/"))
            if workflow_path != (kind == "workflow"):
                errors.append(f"{label}: kind 'workflow' is reserved for .github/workflows and .github/actions")

    owner = entry.get("owner")
    if is_text(owner) and owner not in capabilities:
        errors.append(f"{label}: owner {owner!r} is not a declared capability")

    lifecycle = entry.get("lifecycle")
    if is_text(lifecycle):
        if lifecycle.lower() in EPHEMERAL_LIFECYCLES:
            errors.append(f"{label}: lifecycle {lifecycle!r} is ephemeral; ephemeral helpers are not "
                          f"registered: execute, preserve sanitized Evidence, discard (see {POLICY_DOC})")
        elif lifecycle not in LIFECYCLES:
            errors.append(f"{label}: invalid lifecycle {lifecycle!r} (allowed: {', '.join(sorted(LIFECYCLES))})")
        elif lifecycle == "durable":
            if entry.get("trigger") is None and "trigger" in entry:
                errors.append(f"{label}: durable automation must declare its recurring trigger")
            if entry.get("replay_reason") is not None:
                errors.append(f"{label}: durable automation must not declare a replay_reason; "
                              f"use lifecycle 'historical-replay' for replay-only tooling")
        else:
            for field in ("replay_reason", "removal_condition"):
                if entry.get(field) is None and field in entry:
                    errors.append(f"{label}: historical-replay automation must declare {field}")

    disposition = entry.get("disposition")
    if is_text(disposition):
        if disposition not in DISPOSITIONS:
            errors.append(f"{label}: invalid disposition {disposition!r} "
                          f"(allowed: {', '.join(sorted(DISPOSITIONS))})")
        elif disposition != "keep":
            for field in ("disposition_ref", "removal_condition"):
                if entry.get(field) is None and field in entry:
                    errors.append(f"{label}: disposition {disposition!r} requires {field}")

    if "runtime" in entry:
        check_vocabulary(label, "runtime", entry["runtime"], RUNTIMES, errors)
    if "scope" in entry:
        check_vocabulary(label, "scope", entry["scope"], SCOPES, errors)
    if "callers" in entry:
        check_paths(root, label, "callers", entry["callers"], True, errors)
        if isinstance(entry["callers"], list) and path in entry["callers"]:
            errors.append(f"{label}: an entry cannot declare itself as a caller")
    if "tests" in entry:
        check_paths(root, label, "tests", entry["tests"], False, errors)

    compatibility = entry.get("compatibility")
    if "compatibility" in entry and not isinstance(compatibility, list):
        errors.append(f"{label}: compatibility must be a list")
    for pin in compatibility if isinstance(compatibility, list) else []:
        if not isinstance(pin, dict) or not set(pin) <= {"relation", "revision", "path"} \
                or not is_text(pin.get("relation")) or not ({"revision", "path"} & set(pin)):
            errors.append(f"{label}: compatibility pin must declare relation and a revision and/or path: {pin!r}")
            continue
        if "revision" in pin and not (isinstance(pin["revision"], str) and REVISION.match(pin["revision"])):
            errors.append(f"{label}: compatibility revision must be a full 40-character lowercase SHA: "
                          f"{pin['revision']!r}")
        if "path" in pin and not (safe_relative(pin["path"]) and resolves(root, pin["path"], False)):
            errors.append(f"{label}: compatibility path does not exist: {pin['path']!r}")

    return path if is_text(path) else None


def validate(root: Path) -> tuple[list[str], int]:
    errors: list[str] = []
    registry_path = root / REGISTRY
    if os.path.islink(registry_path) or not registry_path.is_file():
        return [f"registry is missing or not a regular file: {REGISTRY}"], 0
    try:
        data = json.loads(registry_path.read_text(encoding="utf-8"), object_pairs_hook=reject_duplicate_keys)
    except DuplicateKey as error:
        return [f"registry declares duplicate JSON key: {error}"], 0
    except (ValueError, UnicodeDecodeError) as error:
        return [f"registry is not valid UTF-8 JSON: {error}"], 0
    if not isinstance(data, dict) or set(data) != set(TOP_LEVEL):
        return [f"registry must declare exactly: {', '.join(TOP_LEVEL)}"], 0
    if data["schema_version"] != 1:
        errors.append("schema_version must be 1")
    if data["policy"] != POLICY_DOC or not resolves(root, POLICY_DOC, True):
        errors.append(f"policy must reference the existing {POLICY_DOC}")

    capabilities = data["capabilities"]
    if not isinstance(capabilities, dict) or not capabilities:
        errors.append("capabilities must be a non-empty object")
        capabilities = {}
    for name, description in capabilities.items():
        if not CAPABILITY_NAME.match(name) or not is_text(description):
            errors.append(f"capability {name!r} needs a kebab-case name and a description")

    scopes = check_scopes(root, data["excluded_scopes"], errors)

    entries = data["entries"]
    if not isinstance(entries, list):
        return errors + ["entries must be a list"], 0
    registered: dict[str, int] = {}
    previous = None
    for index, entry in enumerate(entries):
        path = check_entry(root, entry, index, capabilities, errors)
        if path is None:
            continue
        if path in registered:
            errors.append(f"duplicate registration: {path}")
        elif previous is not None and path < previous:
            errors.append(f"entries must be sorted by path: {path} follows {previous}")
        registered[path] = registered.get(path, 0) + 1
        previous = path

    try:
        files = listed_files(root)
    except RuntimeError as error:
        return errors + [str(error)], 0
    surfaces = {f for f in files if governed(root, f, scopes)}
    for path in sorted(surfaces - set(registered)):
        errors.append(f"unregistered automation surface: {path} (register it in {REGISTRY}, "
                      f"or discard it if it is a one-off helper; see {POLICY_DOC})")
    for path in sorted(set(registered) - surfaces):
        if os.path.lexists(root / path) and (root / path).is_file() and not os.path.islink(root / path):
            errors.append(f"registered path is not a governed automation surface "
                          f"(ignored, excluded or not automation): {path}")
    return errors, len(surfaces)


def main(argv: list[str]) -> int:
    if len(argv) > 2 or (len(argv) == 2 and argv[1].startswith("-")):
        print("usage: check-automation-registry.py [ROOT]", file=sys.stderr)
        return 2
    target = Path(argv[1] if len(argv) == 2 else ".")
    if not target.is_dir():
        print(f"FAIL: target directory does not exist: {target}", file=sys.stderr)
        return 1
    root = Path(os.path.realpath(target))
    errors, count = validate(root)
    for error in errors:
        print(f"FAIL: {error}", file=sys.stderr)
    if errors:
        return 1
    print(f"PASS: automation registry covers all {count} governed automation surfaces")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
