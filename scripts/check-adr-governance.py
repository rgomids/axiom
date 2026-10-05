#!/usr/bin/env python3
"""Validate the structural ADR evolution and supersession contract (ADR-0018).

    scripts/check-adr-governance.py [ROOT]

Checks only the structure defined by
docs/decisions/0018-adr-evolution-and-supersession-governance.md: ADR identity,
lifecycle statuses, canonical partial- and full-supersession syntax, existing,
accepted and non-self-referential targets, titles, dates and anchors,
reciprocal `## Supersedes` backlinks, and reconciliation of the ADR index in
docs/decisions/README.md. It never decides whether two decisions semantically
conflict; that is a review responsibility. It runs offline, reads only
docs/decisions/*.md and never evaluates their content. Exit status is 0 when
valid, 1 on violations and 2 on usage error.
"""

from __future__ import annotations

import datetime
import posixpath
import re
import sys
from pathlib import Path

DECISIONS = "docs/decisions"
INDEX = "README.md"
STATUSES = ("Proposed", "Accepted", "Superseded", "Rejected")
SUPERSEDING_STATUSES = {"Accepted", "Superseded"}
MAX_BYTES = 1 << 20
MAX_FAILURES = 50
CLIP = 120

ADR_FILE = re.compile(r"^(\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$")
TITLE = re.compile(r"^# (ADR-(\d{4}) — \S.*)$")
HEADING = re.compile(r"^(#{1,6})[ \t]+(.+?)[ \t]*#*[ \t]*$")
FENCE = re.compile(r"^[ \t]{0,3}(`{3,}|~{3,})")
INLINE_CODE = re.compile(r"(`+).+?\1")
LINK = re.compile(r"\[[^\]]*\]\(([^)\s]+)\)")
TARGET = r"\[(?P<label>[^\]]+)\]\((?P<target>[^)\s]+)\)"
DATE = r"(?P<date>\d{4}-\d{2}-\d{2})"
ANNOTATION = re.compile(rf"^[ \t]*> Superseded by {TARGET}, accepted {DATE}\.$")
FULL_STATUS = re.compile(rf"^Superseded by {TARGET}, accepted {DATE}\.$")
ENTRY = re.compile(r"^- \[(?P<label>[^\]]+)\]\((?P<target>[^)\s]+)\) — (?P<rest>\S.*)$")
INDEX_FULL = re.compile(r"^(?:\*\*)?Superseded by \[[^\]]+\]\((?P<target>[^)\s]+)\)")
INDEX_LINK = r"\[(?P<label{n}>[^\]]+)\]\((?P<target{n}>[^)\s]+)\)"
PARTIAL_CLAUSE = re.compile(r"Partially superseded by ((?:\[[^\]]+\]\([^)\s]+\))(?:(?:, | and )\[[^\]]+\]\([^)\s]+\))*)")
CLAUSE_LINK = re.compile(r"\[(?P<label>[^\]]+)\]\((?P<target>[^)\s]+)\)")

failures: list[str] = []


def fail(where: str, line: int | None, message: str) -> None:
    failures.append(f"{where}:{line}: {message}" if line else f"{where}: {message}")


def clip(text: str) -> str:
    text = text.strip()
    return text if len(text) <= CLIP else text[: CLIP - 1] + "…"


def slug(text: str) -> str:
    """GitHub-style heading anchor."""
    text = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", text).strip().lower()
    return re.sub(r"[^\w\- ]", "", text).replace(" ", "-")


def split_target(target: str) -> tuple[str, str | None]:
    path, _, anchor = target.partition("#")
    return path, (anchor if "#" in target else None)


def adr_link(target: str) -> str | None:
    """Filename of the sibling ADR a relative link resolves to, or None for other links."""
    path, _ = split_target(target)
    if not path or re.match(r"^[a-zA-Z][a-zA-Z0-9+.-]*:", path) or path.startswith("/"):
        return None
    resolved = posixpath.normpath(posixpath.join(DECISIONS, path))
    if posixpath.dirname(resolved) != DECISIONS or not resolved.endswith(".md"):
        return None
    name = posixpath.basename(resolved)
    return None if name == INDEX else name


class Doc:
    def __init__(self, name: str, text: str):
        self.name = name
        self.where = f"{DECISIONS}/{name}"
        self.raw = text.split("\n")
        self.masked: list[str] = []
        fence = None
        for line in self.raw:
            match = FENCE.match(line)
            if fence:
                if match and match.group(1)[0] == fence[0] and len(match.group(1)) >= len(fence):
                    fence = None
                self.masked.append("")
            elif match:
                fence = match.group(1)
                self.masked.append("")
            else:
                self.masked.append(INLINE_CODE.sub("", line))
        self.slugs: set[str] = set()
        self.sections: dict[str, list[int]] = {}
        seen: dict[str, int] = {}
        current = None
        for number, line in enumerate(self.masked, 1):
            match = HEADING.match(line)
            if match:
                base = slug(match.group(2))
                count = seen.get(base, 0)
                seen[base] = count + 1
                self.slugs.add(base if count == 0 else f"{base}-{count}")
                if len(match.group(1)) <= 2:
                    current = match.group(2) if len(match.group(1)) == 2 else None
                    if current is not None:
                        self.sections.setdefault(current, []).append(number)
                    continue
            if current is not None:
                self.sections.setdefault(f"\0{current}", []).append(number)

    def section(self, heading: str) -> list[int]:
        return self.sections.get(f"\0{heading}", [])

    def links(self, numbers=None):
        for number in numbers if numbers is not None else range(1, len(self.masked) + 1):
            for match in LINK.finditer(self.masked[number - 1]):
                yield number, match.group(1)


class ADR(Doc):
    def __init__(self, name: str, text: str):
        super().__init__(name, text)
        self.title = None
        self.status = None
        self.status_line = None
        self.full = None
        self.annotations: list[tuple[int, re.Match]] = []
        first = next((n for n, line in enumerate(self.raw, 1) if line.strip()), None)
        match = TITLE.match(self.raw[first - 1]) if first else None
        if not match:
            fail(self.where, first, "first line must be the H1 '# ADR-NNNN — <title>'")
        elif match.group(2) != name[:4]:
            fail(self.where, first, f"H1 number ADR-{match.group(2)} does not match the filename prefix {name[:4]}")
        else:
            self.title = match.group(1)
        self.parse_status()
        for number, line in enumerate(self.masked, 1):
            content = re.sub(r"^[ \t]*(>[ \t]*)+", "", line)
            if content == line:
                continue
            word = content.lstrip("*_ \t").lower()
            if not (word.startswith("superseded") or word.startswith("partially superseded")):
                continue
            annotation = ANNOTATION.match(self.raw[number - 1].rstrip())
            if annotation:
                self.annotations.append((number, annotation))
            else:
                fail(self.where, number, "non-canonical supersession annotation "
                     f"'{clip(self.raw[number - 1])}'; expected '> Superseded by "
                     "[ADR-NNNN — <title>](NNNN-<slug>.md#<section>), accepted YYYY-MM-DD.' (ADR-0018)")

    def parse_status(self):
        headings = self.sections.get("Status", [])
        if len(headings) != 1:
            fail(self.where, headings[1] if headings else None,
                 f"expected exactly one '## Status' section, found {len(headings)}")
            return
        body = [n for n in self.section("Status") if self.raw[n - 1].strip()]
        if not body:
            fail(self.where, headings[0], "'## Status' section is empty")
            return
        self.status_line = body[0]
        line = self.raw[body[0] - 1].strip()
        text = re.sub(r"^(?:\*\*)?(?:Status:[ \t]*)?(?:\*\*)?", "", line)
        word = re.match(r"[A-Za-z]*", text).group(0)
        if word not in STATUSES:
            hint = "; partial supersession keeps 'Accepted' (ADR-0018)" if word.lower() == "partially" else ""
            fail(self.where, body[0], f"lifecycle status '{clip(word or line)}' is not one of "
                 f"{', '.join(STATUSES)}{hint}")
            return
        self.status = word
        if word == "Superseded":
            match = FULL_STATUS.match(line)
            if match:
                self.full = (body[0], match)
            else:
                fail(self.where, body[0], f"full supersession status '{clip(line)}' must be exactly "
                     "'Superseded by [ADR-NNNN — <title>](NNNN-<slug>.md), accepted YYYY-MM-DD.' (ADR-0018)")

    def status_text(self) -> str:
        return "\n".join(self.raw[n - 1] for n in self.section("Status"))


def check_anchor(doc: Doc, number: int, target: ADR, anchor: str | None) -> None:
    if anchor is not None and anchor not in target.slugs:
        fail(doc.where, number, f"anchor '#{clip(anchor)}' does not match a heading in {target.name}")


def check_reference(adr: ADR, number: int, match: re.Match, adrs: dict[str, ADR], kind: str) -> ADR | None:
    path, anchor = split_target(match.group("target"))
    if not ADR_FILE.match(path):
        fail(adr.where, number, f"{kind} target '{clip(match.group('target'))}' must be a sibling ADR filename")
        return None
    if path == adr.name:
        fail(adr.where, number, f"self-referential {kind}: {adr.name} cannot supersede itself")
        return None
    target = adrs.get(path)
    if target is None:
        fail(adr.where, number, f"{kind} references nonexistent ADR {path}")
        return None
    if target.title is not None and match.group("label") != target.title:
        fail(adr.where, number, f"{kind} label '{clip(match.group('label'))}' does not match the H1 title of "
             f"{path}: '{clip(target.title)}'")
    check_anchor(adr, number, target, anchor)
    date = match.group("date")
    try:
        datetime.date.fromisoformat(date)
    except ValueError:
        fail(adr.where, number, f"{kind} date {date} is not a valid date")
    else:
        if date not in target.status_text():
            fail(adr.where, number, f"{kind} date {date} is not the acceptance date recorded in the "
                 f"'## Status' section of {path}")
    if target.status is not None and target.status not in SUPERSEDING_STATUSES:
        fail(adr.where, number, f"{kind} target {path} has status '{target.status}'; only an Accepted ADR "
             "supersedes another one")
    if not any(adr_link(link) == adr.name for _, link in target.links(target.section("Supersedes"))):
        fail(adr.where, number, f"missing backlink: {path} has no '## Supersedes' section linking {adr.name}")
    return target


def check_adrs(adrs: dict[str, ADR]) -> None:
    titles: dict[str, str] = {}
    for adr in adrs.values():
        if adr.title in titles:
            fail(adr.where, 1, f"ambiguous identity: H1 title duplicates {titles[adr.title]}")
        elif adr.title:
            titles[adr.title] = adr.name
        reported = {number for number, _ in adr.annotations} | ({adr.full[0]} if adr.full else set())
        for number, link in adr.links():
            name = adr_link(link)
            if name is not None and name not in adrs and number not in reported:
                fail(adr.where, number, f"link '{clip(link)}' references nonexistent ADR {name}")
        if adr.full:
            check_reference(adr, adr.full[0], adr.full[1], adrs, "full supersession")
        if adr.annotations and adr.status not in SUPERSEDING_STATUSES | {None}:
            fail(adr.where, adr.annotations[0][0], f"a '{adr.status}' ADR cannot carry supersession "
                 "annotations; only an Accepted or Superseded ADR can")
        for number, match in adr.annotations:
            check_reference(adr, number, match, adrs, "partial-supersession annotation")
        headings = adr.sections.get("Supersedes", [])
        if len(headings) > 1:
            fail(adr.where, headings[1], "ambiguous metadata: more than one '## Supersedes' section")
        if headings and adr.status is not None and adr.status not in SUPERSEDING_STATUSES:
            fail(adr.where, headings[0], f"a '{adr.status}' ADR cannot supersede; only an Accepted ADR can")
        for number, link in adr.links(adr.section("Supersedes")):
            name = adr_link(link)
            if name is None or name not in adrs:
                continue
            if name == adr.name:
                fail(adr.where, number, f"self-referential '## Supersedes' link to {adr.name}")
                continue
            old = adrs[name]
            check_anchor(adr, number, old, split_target(link)[1])
            points_back = any(split_target(m.group("target"))[0] == adr.name
                              for _, m in old.annotations + ([old.full] if old.full else []))
            if not points_back:
                fail(adr.where, number, f"declares it supersedes {name}, but {name} has neither a "
                     f"'Superseded by' status nor a partial-supersession annotation pointing to {adr.name}")


def check_index(index: Doc, adrs: dict[str, ADR]) -> None:
    headings = index.sections.get("Index", [])
    if len(headings) != 1:
        fail(index.where, headings[1] if headings else None,
             f"expected exactly one '## Index' section, found {len(headings)}")
        return
    for number, link in index.links():
        name = adr_link(link)
        if name is not None and name not in adrs:
            fail(index.where, number, f"link '{clip(link)}' references nonexistent ADR {name}")
    listed: dict[str, int] = {}
    for number in index.section("Index"):
        raw = index.raw[number - 1]
        if not raw.strip():
            continue
        entry = ENTRY.match(raw)
        if not entry:
            fail(index.where, number, f"index line '{clip(raw)}' must be '- [ADR-NNNN — <title>](NNNN-<slug>.md) "
                 "— <Status> ...'")
            continue
        path, anchor = split_target(entry.group("target"))
        adr = adrs.get(path)
        if anchor is not None or adr is None:
            fail(index.where, number, f"index entry target '{clip(entry.group('target'))}' is not an existing ADR file")
            continue
        if path in listed:
            fail(index.where, number, f"{path} is listed more than once (first on line {listed[path]})")
            continue
        listed[path] = number
        if adr.title is not None and entry.group("label") != adr.title:
            fail(index.where, number, f"index label '{clip(entry.group('label'))}' does not match the H1 title of "
                 f"{path}: '{clip(adr.title)}'")
        rest = entry.group("rest")
        word = re.match(r"(?:\*\*)?([A-Za-z]*)", rest).group(1)
        if adr.status is not None and word != adr.status:
            fail(index.where, number, f"index status '{clip(word or rest)}' for {path} diverges from its lifecycle "
                 f"status '{adr.status}'")
        if adr.full:
            match = INDEX_FULL.match(rest)
            superseding = split_target(adr.full[1].group("target"))[0]
            if not match or split_target(match.group("target"))[0] != superseding:
                fail(index.where, number, f"index entry for superseded {path} must start its status with "
                     f"'Superseded by [ADR-NNNN]({superseding})'")
        expected = {split_target(m.group("target"))[0] for _, m in adr.annotations} & set(adrs) - {path}
        claimed: set[str] = set()
        note = INLINE_CODE.sub("", rest)
        clauses = list(PARTIAL_CLAUSE.finditer(note))
        if len(clauses) != len(re.findall(r"partially superseded", note, re.IGNORECASE)):
            fail(index.where, number, f"non-canonical partial-supersession note for {path}; expected "
                 "'Partially superseded by [ADR-NNNN](NNNN-<slug>.md)' (ADR-0018)")
        for clause in clauses:
            for link in CLAUSE_LINK.finditer(clause.group(1)):
                target = split_target(link.group("target"))[0]
                if link.group("label") != f"ADR-{target[:4]}":
                    fail(index.where, number, f"partial-supersession note label '{clip(link.group('label'))}' must "
                         f"be 'ADR-{target[:4]}'")
                claimed.add(target)
        missing = sorted(expected - claimed)
        extra = sorted(claimed - expected)
        if missing:
            fail(index.where, number, f"{path} is partially superseded by {', '.join(missing)} but its index "
                 "entry lacks 'Partially superseded by' for it")
        if extra:
            fail(index.where, number, f"index entry claims {path} is partially superseded by {', '.join(extra)}, "
                 "but no annotation in it points there")
    for name in sorted(set(adrs) - set(listed)):
        fail(index.where, headings[0], f"{name} is missing from the index")


def read(path: Path, where: str) -> str | None:
    if path.is_symlink() or not path.is_file():
        fail(where, None, "must be a regular file, not a symlink")
        return None
    if path.stat().st_size > MAX_BYTES:
        fail(where, None, f"exceeds {MAX_BYTES} bytes")
        return None
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        fail(where, None, "is not valid UTF-8")
        return None


def main(argv: list[str]) -> int:
    if len(argv) > 2:
        print("usage: check-adr-governance.py [ROOT]", file=sys.stderr)
        return 2
    root = Path(argv[1] if len(argv) == 2 else ".")
    if not root.is_dir():
        print(f"usage: ROOT is not a directory: {root}", file=sys.stderr)
        return 2
    directory = root / DECISIONS
    if not directory.is_dir() or directory.is_symlink():
        fail(DECISIONS, None, "ADR directory is missing")
    else:
        adrs: dict[str, ADR] = {}
        for path in sorted(directory.iterdir(), key=lambda p: p.name):
            if path.name == INDEX or not path.name.endswith(".md"):
                continue
            where = f"{DECISIONS}/{path.name}"
            if not ADR_FILE.match(path.name):
                fail(where, None, "ADR filename must be 'NNNN-<lowercase-slug>.md'")
                continue
            text = read(path, where)
            if text is not None:
                adrs[path.name] = ADR(path.name, text)
        check_adrs(adrs)
        text = read(directory / INDEX, f"{DECISIONS}/{INDEX}")
        if text is not None:
            check_index(Doc(INDEX, text), adrs)
    if failures:
        for message in failures[:MAX_FAILURES]:
            print(f"FAIL: {message}", file=sys.stderr)
        if len(failures) > MAX_FAILURES:
            print(f"FAIL: {len(failures) - MAX_FAILURES} more violation(s) not shown", file=sys.stderr)
        return 1
    print(f"PASS: ADR governance structure is valid ({len(adrs)} ADRs)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
