#!/usr/bin/env python3
"""Offline black-box tests of check-adr-governance.py (ADR-0018); fixtures never touch the network.

Each negative case mutates one invariant of a valid fixture and asserts the
exact violation set, so a failure proves the intended cause rather than an
unrelated earlier check.
"""
import io
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve()
CHECK = HERE.with_name("check-adr-governance.py")
REPOSITORY = HERE.parent.parent
D = "docs/decisions"

ADRS = {
    # Partially superseded by 0002: keeps Accepted and annotates the fragment in place.
    "0001-alpha.md": """# ADR-0001 — Alpha

## Status

Accepted on 2026-01-01.

## Decision

The runtime uses ~~X~~ Y.

> Superseded by [ADR-0002 — Beta](0002-beta.md#decision), accepted 2026-02-01.
> Only the struck fragment is superseded; the rest remains in force.

## Consequences

Kept.
""",
    "0002-beta.md": """# ADR-0002 — Beta

## Status

**Accepted** on 2026-02-01.

## Decision

The runtime uses Z.

## Supersedes

| Artifact | Superseded fragment |
|---|---|
| [ADR-0001](0001-alpha.md#decision) | "X" |
""",
    # Fully superseded by 0004.
    "0003-gamma.md": """# ADR-0003 — Gamma

## Status

Superseded by [ADR-0004 — Delta](0004-delta.md), accepted 2026-04-01.

Accepted on 2026-03-01.

## Decision

Gamma.
""",
    "0004-delta.md": """# ADR-0004 — Delta

## Status

Status: Accepted
Accepted: 2026-04-01

## Decision

Delta.

## Supersedes

- [ADR-0003](0003-gamma.md), entirely.
""",
    "0005-epsilon.md": """# ADR-0005 — Epsilon

## Status

Proposed for human review on 2026-05-01.

## Decision

Epsilon.
""",
    # Two ADRs share a number prefix; identity is the path.
    "0007-eta-one.md": """# ADR-0007 — Eta one

## Status

Accepted on 2026-07-01.

## Decision

Eta one. See [ADR-0007 — Eta two](0007-eta-two.md).

```markdown
> Superseded by an example inside a fence is not an annotation.
```
""",
    "0007-eta-two.md": """# ADR-0007 — Eta two

## Status

Accepted on 2026-07-02.

## Decision

Eta two.
""",
}

INDEX = """# ADRs

## Index

- [ADR-0001 — Alpha](0001-alpha.md) — Accepted, 2026-01-01; Partially superseded by [ADR-0002](0002-beta.md).
- [ADR-0002 — Beta](0002-beta.md) — **Accepted, 2026-02-01**; supersedes a fragment of ADR-0001.
- [ADR-0003 — Gamma](0003-gamma.md) — Superseded by [ADR-0004](0004-delta.md), 2026-04-01.
- [ADR-0004 — Delta](0004-delta.md) — Accepted, 2026-04-01.
- [ADR-0005 — Epsilon](0005-epsilon.md) — Proposed, 2026-05-01.

- [ADR-0007 — Eta one](0007-eta-one.md) — Accepted, 2026-07-01; mentions `Partially superseded by` only as code.
- [ADR-0007 — Eta two](0007-eta-two.md) — Accepted, 2026-07-02.

## Notes

Free text outside the index.
"""


class ADRGovernanceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="axiom-adr-governance-")
        self.root = Path(self.temp.name)
        (self.root / D).mkdir(parents=True)
        for name, text in ADRS.items():
            self.write(name, text)
        self.write("README.md", INDEX)

    def tearDown(self):
        self.temp.cleanup()

    def path(self, name):
        return self.root / D / name

    def write(self, name, text):
        self.path(name).write_text(text, encoding="utf-8")

    def edit(self, name, old, new):
        text = self.path(name).read_text(encoding="utf-8")
        self.assertEqual(text.count(old), 1, f"fixture edit must be unambiguous: {old!r}")
        self.write(name, text.replace(old, new))

    def append(self, name, text):
        self.write(name, self.path(name).read_text(encoding="utf-8") + text)

    def run_check(self, *args):
        argv = args if args else (str(self.root),)
        return subprocess.run([sys.executable, str(CHECK), *argv], capture_output=True, text=True, timeout=60)

    def assert_passes(self):
        result = self.run_check()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("PASS: ADR governance structure is valid", result.stdout)

    def assert_fails(self, *expected):
        """Exactly the expected violations (as substrings, one per FAIL line) are reported."""
        result = self.run_check()
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        lines = [line for line in result.stderr.splitlines() if line.startswith("FAIL: ")]
        self.assertEqual(len(lines), len(expected), result.stderr)
        for fragment in expected:
            self.assertTrue(any(fragment in line for line in lines), f"{fragment!r} not in:\n{result.stderr}")

    # Positive cases.

    def test_valid_fixture_passes(self):
        """Ordinary accepted, proposed, partial and full supersession, shared number prefix."""
        self.assert_passes()

    def test_real_adr_set_passes(self):
        """The repository ADRs, including the ADR-0010 -> ADR-0015 baseline, pass."""
        result = self.run_check(str(REPOSITORY))
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_adr_0010_0015_baseline_requires_backlink(self):
        shutil.rmtree(self.root / D)
        shutil.copytree(REPOSITORY / D, self.root / D)
        self.edit("0015-installer-host-eligibility-os-family-architecture.md",
                  "| [ADR-0010](0010-windows-native-filesystem-boundary.md#decision) |", "| ADR-0010 |")
        self.assert_fails("missing backlink: 0015-installer-host-eligibility-os-family-architecture.md has no "
                          "'## Supersedes' section linking 0010-windows-native-filesystem-boundary.md")

    def test_adr_0010_0015_baseline_requires_index_note(self):
        shutil.rmtree(self.root / D)
        shutil.copytree(REPOSITORY / D, self.root / D)
        self.edit("README.md", "Partially superseded by [ADR-0015]", "Superseded in part by [ADR-0015]")
        self.assert_fails("0010-windows-native-filesystem-boundary.md is partially superseded by "
                          "0015-installer-host-eligibility-os-family-architecture.md")
        self.assert_fails("but its index entry lacks 'Partially superseded by'")

    # Lifecycle status.

    def test_partially_superseded_status_is_rejected(self):
        self.edit("0005-epsilon.md", "Proposed for human review", "Partially Superseded")
        self.assert_fails("lifecycle status 'Partially' is not one of Proposed, Accepted, Superseded, Rejected; "
                          "partial supersession keeps 'Accepted' (ADR-0018)")

    def test_invented_status_is_rejected(self):
        self.edit("0005-epsilon.md", "Proposed for human review", "**Deprecated**")
        self.assert_fails("lifecycle status 'Deprecated' is not one of")

    def test_rejected_status_is_allowed(self):
        self.edit("0005-epsilon.md", "Proposed for human review", "Rejected")
        self.edit("README.md", "— Proposed, 2026-05-01.", "— Rejected, 2026-05-01.")
        self.assert_passes()

    def test_multiple_status_sections_are_ambiguous(self):
        self.append("0007-eta-two.md", "\n## Status\n\nRejected.\n")
        self.assert_fails("expected exactly one '## Status' section, found 2")

    def test_missing_status_section_fails(self):
        self.edit("0007-eta-two.md", "## Status", "## State")
        self.assert_fails("expected exactly one '## Status' section, found 0")

    # Identity.

    def test_title_number_must_match_filename(self):
        self.edit("0007-eta-two.md", "# ADR-0007 — Eta two", "# ADR-0008 — Eta two")
        self.assert_fails("H1 number ADR-0008 does not match the filename prefix 0007")

    def test_duplicate_titles_are_ambiguous(self):
        self.edit("0007-eta-two.md", "# ADR-0007 — Eta two", "# ADR-0007 — Eta one")
        self.assert_fails("ambiguous identity: H1 title duplicates 0007-eta-one.md",
                          "index label 'ADR-0007 — Eta two' does not match the H1 title of 0007-eta-two.md")

    def test_invalid_filename_fails(self):
        self.write("0009_Bad.md", "# ADR-0009 — Bad\n")
        self.assert_fails("0009_Bad.md: ADR filename must be 'NNNN-<lowercase-slug>.md'")

    def test_shared_prefix_label_resolves_by_path(self):
        self.edit("README.md", "[ADR-0007 — Eta two](0007-eta-two.md)", "[ADR-0007 — Eta one](0007-eta-two.md)")
        self.assert_fails("index label 'ADR-0007 — Eta one' does not match the H1 title of 0007-eta-two.md")

    # References.

    def test_nonexistent_annotation_target_fails(self):
        self.append("0007-eta-one.md",
                    "\n> Superseded by [ADR-0009 — Missing](0009-missing.md), accepted 2026-09-01.\n")
        self.assert_fails("partial-supersession annotation references nonexistent ADR 0009-missing.md")

    def test_nonexistent_adr_link_fails(self):
        self.append("0007-eta-two.md", "\nSee [ADR-0009](0009-missing.md).\n")
        self.assert_fails("link '0009-missing.md' references nonexistent ADR 0009-missing.md")

    def test_self_referential_annotation_fails(self):
        self.append("0007-eta-one.md",
                    "\n> Superseded by [ADR-0007 — Eta one](0007-eta-one.md), accepted 2026-07-01.\n")
        self.assert_fails("self-referential partial-supersession annotation: 0007-eta-one.md cannot supersede itself")

    def test_self_referential_full_supersession_fails(self):
        self.edit("0007-eta-two.md", "Accepted on 2026-07-02.",
                  "Superseded by [ADR-0007 — Eta two](0007-eta-two.md), accepted 2026-07-02.")
        self.edit("README.md", "— Accepted, 2026-07-02.", "— Superseded by [ADR-0007](0007-eta-two.md).")
        self.assert_fails("self-referential full supersession")

    def test_self_referential_supersedes_link_fails(self):
        self.append("0007-eta-two.md", "\n## Supersedes\n\n- [itself](0007-eta-two.md)\n")
        self.assert_fails("self-referential '## Supersedes' link to 0007-eta-two.md")

    def test_label_must_match_target_title(self):
        self.edit("0001-alpha.md", "[ADR-0002 — Beta]", "[ADR-0002 — Beta rule]")
        self.assert_fails("partial-supersession annotation label 'ADR-0002 — Beta rule' does not match the H1 "
                          "title of 0002-beta.md: 'ADR-0002 — Beta'")

    def test_anchor_must_exist(self):
        self.edit("0001-alpha.md", "0002-beta.md#decision", "0002-beta.md#nowhere")
        self.assert_fails("anchor '#nowhere' does not match a heading in 0002-beta.md")

    def test_date_must_be_target_acceptance_date(self):
        self.edit("0001-alpha.md", "accepted 2026-02-01.", "accepted 2026-02-02.")
        self.assert_fails("annotation date 2026-02-02 is not the acceptance date 2026-02-01 of 0002-beta.md")

    def test_date_must_be_valid(self):
        self.edit("0001-alpha.md", "accepted 2026-02-01.", "accepted 2026-02-30.")
        self.assert_fails("date 2026-02-30 is not a valid date")

    def test_other_status_date_is_not_the_acceptance_date(self):
        """CR-001: a date elsewhere in the target's Status is not its acceptance date."""
        self.edit("0002-beta.md", "**Accepted** on 2026-02-01.\n",
                  "**Accepted** on 2026-02-01.\nOriginally proposed on 2026-01-15.\n")
        self.edit("0001-alpha.md", "accepted 2026-02-01.", "accepted 2026-01-15.")
        self.assert_fails("partial-supersession annotation date 2026-01-15 is not the acceptance date 2026-02-01 "
                          "of 0002-beta.md")

    def test_full_supersession_date_must_be_acceptance_date(self):
        self.edit("0004-delta.md", "Accepted: 2026-04-01\n", "Accepted: 2026-04-01\nProposed: 2026-03-20\n")
        self.edit("0003-gamma.md", "accepted 2026-04-01.", "accepted 2026-03-20.")
        self.assert_fails("full supersession date 2026-03-20 is not the acceptance date 2026-04-01 of 0004-delta.md")

    def test_supported_acceptance_statements_parse(self):
        for statement in (
            "Accepted on 2026-02-01.",
            "**Accepted** on 2026-02-01.\nOriginally proposed on 2026-01-15.",
            "**Accepted — human approval recorded on 2026-02-01.**",
            "**Accepted for the requested implementation** on 2026-02-01. Review stays separate.",
            "Accepted direction — explicit human authorization to implement the proposed\nrecovery recorded on 2026-02-01.",
            "Accepted on 2026-02-01 by the decision in\n[Issue #1](https://example.invalid/1) (\"Human decision\nrecorded — 2026-02-01\").",
            "Status: Accepted\nAccepted: 2026-02-01",
            "Status: Accepted\nAccepted: 2026-02-01\nProposed: 2026-01-15",
        ):
            with self.subTest(statement=statement):
                self.tearDown()
                self.setUp()
                self.edit("0002-beta.md", "**Accepted** on 2026-02-01.", statement)
                self.assert_passes()

    def test_undeterminable_acceptance_date_fails(self):
        sentence = "the first sentence of its acceptance statement must give exactly one YYYY-MM-DD date"
        labelled = "the 'Accepted: YYYY-MM-DD' line under 'Status: Accepted' must give exactly one YYYY-MM-DD date"
        for statement, reason in (
            ("Accepted after review.\nRecorded on 2026-02-01.", f"{sentence} (found 0)"),
            ("Accepted on 2026-02-01 after the 2026-01-15 proposal.", f"{sentence} (found 2)"),
            ("Status: Accepted\nProposed: 2026-02-01", f"{labelled} (found 0)"),
        ):
            with self.subTest(statement=statement):
                self.tearDown()
                self.setUp()
                self.edit("0002-beta.md", "**Accepted** on 2026-02-01.", statement)
                self.assert_fails(f"cannot determine the acceptance date of 0002-beta.md: {reason}")

    def test_superseded_target_uses_preserved_acceptance_date(self):
        """A Superseded target's acceptance date is its preserved statement, not the 'Superseded by' date."""
        for date, valid in (("2026-03-01", True), ("2026-04-01", False), ("2026-03-01", None)):
            with self.subTest(date=date, valid=valid):
                self.tearDown()
                self.setUp()
                if valid is None:
                    self.edit("0003-gamma.md", "Accepted on 2026-03-01.", "Previously accepted on 2026-03-01.")
                self.append("0007-eta-two.md",
                            f"\n> Superseded by [ADR-0003 — Gamma](0003-gamma.md), accepted {date}.\n")
                self.append("0003-gamma.md", "\n## Supersedes\n\n- [ADR-0007](0007-eta-two.md), fragment.\n")
                self.edit("README.md", "— Accepted, 2026-07-02.",
                          "— Accepted, 2026-07-02; Partially superseded by [ADR-0003](0003-gamma.md).")
                if valid:
                    self.assert_passes()
                elif valid is None:
                    self.assert_fails("cannot determine the acceptance date of 0003-gamma.md: no preserved 'Accepted' "
                                      "statement in the paragraph after its 'Superseded by' line")
                else:
                    self.assert_fails("date 2026-04-01 is not the acceptance date 2026-03-01 of 0003-gamma.md")

    def test_full_supersession_target_rejects_anchor(self):
        self.edit("0003-gamma.md", "](0004-delta.md), accepted", "](0004-delta.md#decision), accepted")
        self.assert_fails("full supersession target '0004-delta.md#decision' must be the ADR file without an anchor")

    def test_target_must_be_accepted(self):
        self.edit("0002-beta.md", "**Accepted** on 2026-02-01.", "Proposed on 2026-02-01.")
        self.edit("README.md", "**Accepted, 2026-02-01**", "Proposed, 2026-02-01")
        self.assert_fails("target 0002-beta.md has status 'Proposed'; only an Accepted ADR supersedes another one",
                          "a 'Proposed' ADR cannot supersede")

    def test_proposed_adr_cannot_carry_annotations(self):
        self.append("0005-epsilon.md", "\nX.\n\n> Superseded by [ADR-0002 — Beta](0002-beta.md), accepted 2026-02-01.\n")
        self.append("0002-beta.md", "- [ADR-0005](0005-epsilon.md)\n")
        self.edit("README.md", "— Proposed, 2026-05-01.", "— Proposed, 2026-05-01; Partially superseded by [ADR-0002](0002-beta.md).")
        self.assert_fails("a 'Proposed' ADR cannot carry supersession annotations")

    # Annotation syntax.

    def test_invalid_annotations_fail(self):
        for line in (
            "> **Superseded by** [ADR-0002 — Beta](0002-beta.md), accepted 2026-02-01.",
            "> Superseded by [ADR-0002 — Beta](0002-beta.md).",
            "> Superseded by [ADR-0002 — Beta](0002-beta.md), accepted 2026-2-1.",
            "> Superseded by ADR-0002, accepted 2026-02-01.",
            "> superseded by [ADR-0002 — Beta](0002-beta.md), accepted 2026-02-01.",
            "> Partially superseded by [ADR-0002 — Beta](0002-beta.md), accepted 2026-02-01.",
            "> Superseded by [ADR-0002 — Beta](../decisions/0002-beta.md), accepted 2026-02-01.",
        ):
            with self.subTest(line=line):
                self.tearDown()
                self.setUp()
                self.append("0007-eta-one.md", f"\nText.\n\n{line}\n")
                if "../decisions/" in line:
                    self.assert_fails("must be a sibling ADR filename")
                else:
                    self.assert_fails("non-canonical supersession annotation")

    def test_annotation_inside_code_is_ignored(self):
        self.append("0007-eta-one.md", "\n> `Superseded by` is quoted as code here.\n")
        self.assert_passes()

    # Backlinks and full supersession.

    def test_missing_backlink_fails(self):
        self.edit("0002-beta.md", "| [ADR-0001](0001-alpha.md#decision) |", "| ADR-0001 |")
        self.assert_fails("missing backlink: 0002-beta.md has no '## Supersedes' section linking 0001-alpha.md")

    def test_backlink_outside_supersedes_section_does_not_count(self):
        self.edit("0002-beta.md", "## Supersedes", "## Related")
        self.assert_fails("missing backlink: 0002-beta.md has no '## Supersedes' section linking 0001-alpha.md")

    def test_one_sided_supersedes_claim_fails(self):
        self.append("0004-delta.md", "- [ADR-0007](0007-eta-two.md), fragment.\n")
        self.assert_fails("declares it supersedes 0007-eta-two.md, but 0007-eta-two.md has neither a "
                          "'Superseded by' status nor a partial-supersession annotation pointing to 0004-delta.md")

    def test_superseded_status_without_canonical_link_fails(self):
        self.edit("0003-gamma.md", "Superseded by [ADR-0004 — Delta](0004-delta.md), accepted 2026-04-01.",
                  "Superseded by ADR-0004.")
        self.assert_fails("full supersession status 'Superseded by ADR-0004.' must be exactly",
                          "declares it supersedes 0003-gamma.md, but 0003-gamma.md has neither")

    def test_full_supersession_requires_backlink(self):
        self.edit("0004-delta.md", "- [ADR-0003](0003-gamma.md), entirely.", "- ADR-0003, entirely.")
        self.assert_fails("missing backlink: 0004-delta.md has no '## Supersedes' section linking 0003-gamma.md")

    def test_superseded_index_entry_requires_superseding_link(self):
        self.edit("README.md", "— Superseded by [ADR-0004](0004-delta.md), 2026-04-01.", "— Superseded, 2026-04-01.")
        self.assert_fails("index entry for superseded 0003-gamma.md must start its status with "
                          "'Superseded by [ADR-NNNN](0004-delta.md)'")

    def test_superseded_index_entry_label_must_be_number(self):
        self.edit("README.md", "— Superseded by [ADR-0004](0004-delta.md)", "— Superseded by [ADR-0005](0004-delta.md)")
        self.assert_fails("index entry for superseded 0003-gamma.md: label 'ADR-0005' must be 'ADR-0004'")

    def test_superseded_index_entry_rejects_anchor(self):
        self.edit("README.md", "— Superseded by [ADR-0004](0004-delta.md)", "— Superseded by [ADR-0004](0004-delta.md#decision)")
        self.assert_fails("index entry for superseded 0003-gamma.md must link exactly '0004-delta.md' without an "
                          "anchor, found '0004-delta.md#decision'")

    def test_superseded_index_entry_rejects_other_path(self):
        self.edit("README.md", "— Superseded by [ADR-0004](0004-delta.md)", "— Superseded by [ADR-0004](./0004-delta.md)")
        self.assert_fails("index entry for superseded 0003-gamma.md must link exactly '0004-delta.md' without an "
                          "anchor, found './0004-delta.md'")

    # Index reconciliation.

    def test_index_status_divergence_fails(self):
        self.edit("README.md", "— Accepted, 2026-04-01.", "— Proposed, 2026-04-01.")
        self.assert_fails("index status 'Proposed' for 0004-delta.md diverges from its lifecycle status 'Accepted'")

    def test_index_missing_entry_fails(self):
        self.edit("README.md", "- [ADR-0007 — Eta two](0007-eta-two.md) — Accepted, 2026-07-02.\n", "")
        self.assert_fails("0007-eta-two.md is missing from the index")

    def test_index_duplicate_entry_fails(self):
        self.edit("README.md", "\n## Notes", "- [ADR-0004 — Delta](0004-delta.md) — Accepted.\n\n## Notes")
        self.assert_fails("0004-delta.md is listed more than once")

    def test_index_entry_for_nonexistent_adr_fails(self):
        self.edit("README.md", "\n## Notes", "- [ADR-0009 — Missing](0009-missing.md) — Accepted.\n\n## Notes")
        self.assert_fails("link '0009-missing.md' references nonexistent ADR 0009-missing.md",
                          "index entry target '0009-missing.md' is not an existing ADR file")

    def test_index_free_text_line_fails(self):
        self.edit("README.md", "\n## Notes", "ADR-0008 is planned.\n\n## Notes")
        self.assert_fails("index line 'ADR-0008 is planned.' must be")

    def test_index_missing_partial_note_fails(self):
        self.edit("README.md", "; Partially superseded by [ADR-0002](0002-beta.md).", ".")
        self.assert_fails("0001-alpha.md is partially superseded by 0002-beta.md but its index entry lacks")

    def test_index_extra_partial_note_fails(self):
        self.edit("README.md", "— Accepted, 2026-07-02.", "— Accepted, 2026-07-02; Partially superseded by [ADR-0002](0002-beta.md).")
        self.assert_fails("index entry claims 0007-eta-two.md is partially superseded by 0002-beta.md, but no annotation")

    def test_index_non_canonical_partial_note_fails(self):
        self.edit("README.md", "Partially superseded by [ADR-0002](0002-beta.md)", "Partially superseded by ADR-0002")
        self.assert_fails("non-canonical partial-supersession note for 0001-alpha.md",
                          "0001-alpha.md is partially superseded by 0002-beta.md but its index entry lacks")

    def test_index_partial_note_label_must_be_number(self):
        self.edit("README.md", "Partially superseded by [ADR-0002](0002-beta.md)", "Partially superseded by [Beta](0002-beta.md)")
        self.assert_fails("partial-supersession note label 'Beta' must be 'ADR-0002'")

    def test_index_partial_note_rejects_anchor(self):
        self.edit("README.md", "Partially superseded by [ADR-0002](0002-beta.md)",
                  "Partially superseded by [ADR-0002](0002-beta.md#decision)")
        self.assert_fails("partial-supersession note link '0002-beta.md#decision' must be the ADR file without an anchor")

    def test_missing_index_section_fails(self):
        self.edit("README.md", "## Index", "## List")
        self.assert_fails("expected exactly one '## Index' section, found 0")

    # Contract of the command.

    def test_output_is_bounded(self):
        self.append("0007-eta-one.md", "".join(f"\n> Superseded by item {n}.\n" for n in range(60)))
        result = self.run_check()
        self.assertEqual(result.returncode, 1)
        lines = result.stderr.splitlines()
        self.assertEqual(len(lines), 51, result.stderr)
        self.assertEqual(lines[-1], "FAIL: 10 more violation(s) not shown")

    def test_usage_error(self):
        self.assertEqual(self.run_check("a", "b").returncode, 2)
        self.assertEqual(self.run_check(str(self.root / "missing")).returncode, 2)

    def test_missing_decisions_directory_fails(self):
        shutil.rmtree(self.root / D)
        self.assert_fails("ADR directory is missing")

    def test_symlinked_adr_fails(self):
        self.path("0007-eta-two.md").unlink()
        (self.path("0007-eta-two.md")).symlink_to(self.path("0007-eta-one.md"))
        result = self.run_check()
        self.assertEqual(result.returncode, 1)
        self.assertIn("0007-eta-two.md: must be a regular file, not a symlink", result.stderr)


if __name__ == "__main__":
    stream = io.StringIO()
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(ADRGovernanceTests)
    result = unittest.TextTestRunner(stream=stream, verbosity=2).run(suite)
    if not result.wasSuccessful():
        sys.stderr.write(stream.getvalue())
        sys.exit(1)
    print(f"PASS: ADR governance checker behaves as specified ({result.testsRun} cases)")
