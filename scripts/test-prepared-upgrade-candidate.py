#!/usr/bin/env python3
"""Deterministic tests of the prepared candidate's filesystem/identity boundary."""
import argparse
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest import mock

sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location("prepared_candidate", Path(__file__).with_name("prepared-upgrade-candidate.py"))
CANDIDATE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CANDIDATE)
GATE_SPEC = importlib.util.spec_from_file_location("gate_evidence", Path(__file__).with_name("gate-evidence.py"))
GATE = importlib.util.module_from_spec(GATE_SPEC)
GATE_SPEC.loader.exec_module(GATE)


class PreparedCandidateTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        # macOS /var and /tmp are symlink aliases; fixtures intentionally use their
        # physical location while individual rejection tests introduce links.
        self.root = Path(self.temporary.name).resolve()
        self.source = self.root / "prepared"
        self.output = self.root / "output"
        self.source_root = self.root / "checkout"
        for directory in (self.source, self.output, self.source_root):
            directory.mkdir(mode=0o700)
        self.args = argparse.Namespace(prepared_set=str(self.source), output=str(self.output),
                                       source_root=str(self.source_root), tag="v1.2.3", revision="a" * 40,
                                       row="linux-amd64", sha256sums_sha256="")
        self.make_set()
        self.verifier = mock.patch.object(CANDIDATE.subprocess, "run", side_effect=self.verified)
        self.run = self.verifier.start()
        self.addCleanup(self.verifier.stop)
        self.addCleanup(self.temporary.cleanup)

    def verified(self, command, **kwargs):
        self.assertEqual(command[0], "bash")
        self.assertIn("--skip-executable-smoke", command)
        self.assertEqual(command[command.index("--source-root") + 1], str(self.source_root))
        self.assertEqual(command[command.index("--dir") + 1], str(self.output / "artifacts"))
        self.assertTrue(kwargs["check"])
        return subprocess.CompletedProcess(command, 0, b"publication=none\nresult=structural_pass\n", b"")

    def make_set(self):
        for row, (platform, _, architecture) in CANDIDATE.ROWS.items():
            bundle = f"axiom-1.2.3-{platform}-{architecture}"
            binary = "axiom.exe" if row == "windows-amd64" else "axiom"
            identity = {"version": "1.2.3", "revision": "a" * 40, "row": row}
            entries = {binary: b"original harmless binary", "release-metadata.txt": CANDIDATE.metadata_bytes(identity),
                       "MANIFEST.sha256": b"fixture manifest\n", "skills/example/SKILL.md": b"fixture skill\n"}
            with tarfile.open(self.source / (bundle + ".tar.gz"), "w:gz") as archive:
                for directory in (bundle, bundle + "/skills", bundle + "/skills/example"):
                    member = tarfile.TarInfo(directory)
                    member.type = tarfile.DIRTYPE
                    member.mode = 0o700
                    archive.addfile(member)
                for name, data in entries.items():
                    member = tarfile.TarInfo(bundle + "/" + name)
                    member.size = len(data)
                    member.mode = 0o700 if name == binary else 0o600
                    archive.addfile(member, io.BytesIO(data))
        self.update_pin()

    def update_pin(self):
        sums = b"".join(hashlib.sha256((self.source / name).read_bytes()).hexdigest().encode() + b"  " + name.encode() + b"\n"
                        for name in sorted(CANDIDATE.archive_names("1.2.3")))
        (self.source / "SHA256SUMS").write_bytes(sums)
        self.args.sha256sums_sha256 = hashlib.sha256(sums).hexdigest()

    def prepare(self):
        return CANDIDATE.prepare(self.args)

    def recheck(self):
        return CANDIDATE.verify_materialized(self.output / "identity.json", self.args.row, self.args.sha256sums_sha256)

    def gate_arguments(self):
        return argparse.Namespace(candidate=str(self.output / "artifacts"), row=self.args.row,
                                  prepared_identity=str(self.output / "identity.json"),
                                  prepared_tag=self.args.tag, prepared_revision=self.args.revision,
                                  prepared_sha256sums=self.args.sha256sums_sha256)

    def extract_source(self):
        arguments = argparse.Namespace(extract_source=str(self.source), output=str(self.output), row=self.args.row)
        return Path(CANDIDATE.extract_source(arguments))

    def set_selected_members(self, members):
        path = self.source / "axiom-1.2.3-linux-amd64.tar.gz"
        with tarfile.open(path, "w:gz") as archive:
            for member in members:
                archive.addfile(member, io.BytesIO(b"x") if member.size else None)
        self.update_pin()

    def assert_rejected(self):
        with self.assertRaises((CANDIDATE.CandidateError, OSError, ValueError)):
            self.prepare()
        self.run.assert_not_called()

    def test_complete_snapshot_and_materialized_identity(self):
        identity = self.prepare()
        self.assertEqual(identity, self.recheck())
        self.assertEqual(identity["revision"], self.args.revision)
        self.assertEqual(identity["tag"], self.args.tag)
        self.assertEqual(set((self.output / "artifacts").iterdir()),
                         {self.output / "artifacts" / name for name in {"SHA256SUMS", *CANDIDATE.archive_names("1.2.3")}})
        self.assertEqual((self.output / "bundle/axiom").read_bytes(), b"original harmless binary")
        self.assertEqual((self.output / "bundle/axiom").stat().st_mode & 0o777, 0o500)
        self.assertEqual((self.output / "identity.json").stat().st_mode & 0o777, 0o400)
        self.run.assert_called_once()

    def test_original_substitution_after_snapshot_does_not_change_materialized_bytes(self):
        def substitute(command, **kwargs):
            for path in self.source.iterdir():
                path.unlink()
            self.source.rmdir()
            self.source.symlink_to(self.source_root, target_is_directory=True)
            return self.verified(command, **kwargs)
        self.run.side_effect = substitute
        identity = self.prepare()
        self.assertEqual((self.output / "bundle/axiom").read_bytes(), b"original harmless binary")
        self.assertEqual(identity, self.recheck())

    def test_source_directory_replacement_during_snapshot_uses_anchored_original(self):
        snapshot = CANDIDATE.snapshot_file

        def replace_after_checksums(source, destination, name):
            snapshot(source, destination, name)
            if name == "SHA256SUMS":
                self.source.rename(self.root / "original")
                self.source.mkdir(mode=0o700)
                for original in (self.root / "original").iterdir():
                    (self.source / original.name).write_bytes(b"substituted")

        with mock.patch.object(CANDIDATE, "snapshot_file", side_effect=replace_after_checksums):
            identity = self.prepare()
        self.assertEqual((self.output / "bundle/axiom").read_bytes(), b"original harmless binary")
        self.assertEqual(identity, self.recheck())

    def test_symlink_substitution_just_before_source_open_is_rejected(self):
        snapshot = CANDIDATE.snapshot_file

        def replace_before_open(source, destination, name):
            if name == "axiom-1.2.3-linux-amd64.tar.gz":
                (self.source / name).rename(self.root / "original-archive")
                (self.source / name).symlink_to(self.root / "original-archive")
            snapshot(source, destination, name)

        with mock.patch.object(CANDIDATE, "snapshot_file", side_effect=replace_before_open):
            self.assert_rejected()

    def test_selected_archive_traversal_and_link_are_rejected_without_extraction(self):
        name = "axiom-1.2.3-linux-amd64.tar.gz"
        archive = self.source / name
        for entry, kind in (("axiom-1.2.3-linux-amd64/../../escaped", tarfile.REGTYPE),
                            ("axiom-1.2.3-linux-amd64/axiom", tarfile.SYMTYPE)):
            with self.subTest(entry=entry):
                with tarfile.open(archive, "w:gz") as target:
                    member = tarfile.TarInfo(entry)
                    member.type = kind
                    member.linkname = "/foreign"
                    target.addfile(member)
                self.update_pin()
                # The independent extraction check fails even when the authoritative
                # verifier is mocked as successful for this boundary test.
                with self.assertRaises(CANDIDATE.CandidateError):
                    self.prepare()
                self.assertFalse((self.root / "escaped").exists())
                self.assertFalse((self.output / "identity.json").exists())
                for file in (self.output / "artifacts").iterdir():
                    file.unlink()
                (self.output / "artifacts").rmdir()
                (self.output / "bundle").rmdir()

    def test_self_consistent_substituted_set_fails_external_pin(self):
        pin = self.args.sha256sums_sha256
        (self.source / "axiom-1.2.3-windows-amd64.tar.gz").write_bytes(b"different candidate")
        self.update_pin()
        self.args.sha256sums_sha256 = pin
        self.assert_rejected()

    def test_nonselected_archive_checksum_is_required(self):
        (self.source / "axiom-1.2.3-windows-amd64.tar.gz").write_bytes(b"corrupt")
        self.assert_rejected()

    def test_closed_set_rejects_missing_and_extra_files(self):
        (self.source / "unexpected").write_text("extra")
        self.assert_rejected()

    def test_missing_archive(self):
        (self.source / "axiom-1.2.3-linux-arm64.tar.gz").unlink()
        self.assert_rejected()

    def test_symlink_artifact(self):
        archive = self.source / "axiom-1.2.3-linux-arm64.tar.gz"
        archive.rename(self.root / "real-archive")
        archive.symlink_to(self.root / "real-archive")
        self.assert_rejected()

    def test_symlink_source_directory(self):
        link = self.root / "alias"
        link.symlink_to(self.source, target_is_directory=True)
        self.args.prepared_set = str(link)
        self.assert_rejected()

    def test_symlink_ancestor(self):
        link = self.root / "alias"
        link.symlink_to(self.root, target_is_directory=True)
        self.args.prepared_set = str(link / "prepared")
        self.assert_rejected()

    def test_symlink_output(self):
        link = self.root / "alias"
        link.symlink_to(self.output, target_is_directory=True)
        self.args.output = str(link)
        self.assert_rejected()

    def test_nonregular_artifact_does_not_block(self):
        archive = self.source / "axiom-1.2.3-linux-arm64.tar.gz"
        archive.unlink()
        os.mkfifo(archive)
        self.assert_rejected()

    def test_output_must_be_private_and_empty(self):
        self.output.chmod(0o755)
        self.assert_rejected()

    def test_existing_output_is_preserved(self):
        existing = self.output / "existing"
        existing.write_text("preserve")
        self.assert_rejected()
        self.assertEqual(existing.read_text(), "preserve")

    def test_duplicate_checksum_names(self):
        checksums = (self.source / "SHA256SUMS").read_bytes().splitlines(keepends=True)
        data = checksums[0] * 4
        (self.source / "SHA256SUMS").write_bytes(data)
        self.args.sha256sums_sha256 = hashlib.sha256(data).hexdigest()
        self.assert_rejected()

    def test_revision_and_tag_are_explicit(self):
        self.args.revision = "HEAD"
        self.assert_rejected()

    def test_verifier_failure_prevents_materialization(self):
        self.run.side_effect = subprocess.CalledProcessError(1, "verifier")
        with self.assertRaises(subprocess.CalledProcessError):
            self.prepare()
        self.assertEqual(list((self.output / "bundle").iterdir()), [])
        self.assertFalse((self.output / "identity.json").exists())

    def test_verifier_must_confirm_structural_only_result(self):
        self.run.side_effect = lambda command, **kwargs: subprocess.CompletedProcess(
            command, 0, b"publication=none\nresult=pass\n", b"")
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "complete validation"):
            self.prepare()
        self.assertEqual(list((self.output / "bundle").iterdir()), [])
        self.assertFalse((self.output / "identity.json").exists())

    def test_snapshot_mutation_during_verification_is_rejected(self):
        def mutate(command, **kwargs):
            archive = self.output / "artifacts/axiom-1.2.3-linux-amd64.tar.gz"
            archive.chmod(0o600)
            archive.write_bytes(b"mutated")
            archive.chmod(0o400)
            return self.verified(command, **kwargs)
        self.run.side_effect = mutate
        with self.assertRaises(CANDIDATE.CandidateError):
            self.prepare()
        self.assertFalse((self.output / "identity.json").exists())

    def test_revalidation_detects_extracted_binary_mutation(self):
        self.prepare()
        binary = self.output / "bundle/axiom"
        binary.chmod(0o700)
        binary.write_bytes(b"changed")
        binary.chmod(0o500)
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "materialized bundle changed"):
            self.recheck()

    def test_revalidation_detects_snapshot_archive_mutation(self):
        self.prepare()
        archive = self.output / "artifacts/axiom-1.2.3-windows-amd64.tar.gz"
        archive.chmod(0o600)
        archive.write_bytes(b"changed")
        archive.chmod(0o400)
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "snapshot archive changed"):
            self.recheck()

    def test_revalidation_requires_external_row_and_pin(self):
        self.prepare()
        for row, pin in (("linux-arm64", self.args.sha256sums_sha256), (self.args.row, "0" * 64)):
            with self.assertRaises(CANDIDATE.CandidateError):
                CANDIDATE.verify_materialized(self.output / "identity.json", row, pin)

    def test_revalidation_rejects_forged_binary_path(self):
        self.prepare()
        identity_path = self.output / "identity.json"
        identity = json.loads(identity_path.read_text())
        identity["binary"]["path"] = str(self.root / "foreign")
        identity_path.chmod(0o600)
        identity_path.write_text(json.dumps(identity))
        identity_path.chmod(0o400)
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "identity file changed"):
            self.recheck()

    def test_revalidation_rejects_malformed_identity_row_as_validation_error(self):
        self.prepare()
        path = self.output / "identity.json"
        identity = json.loads(path.read_text())
        identity["row"] = []
        path.chmod(0o600)
        path.write_text(json.dumps(identity))
        path.chmod(0o400)
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "identity row changed"):
            self.recheck()

    def test_revalidation_rejects_extracted_symlink(self):
        self.prepare()
        binary = self.output / "bundle/axiom"
        binary.unlink()
        binary.symlink_to(self.root / "foreign")
        with self.assertRaises(OSError):
            self.recheck()

    def test_gate_subject_binds_all_archives_pin_and_full_revision(self):
        identity = self.prepare()
        subject = GATE.Builder(self.gate_arguments(), None).subject()
        self.assertEqual(subject["kind"], "prepared")
        self.assertEqual(subject["tag"], self.args.tag)
        self.assertEqual(subject["revision"], self.args.revision)
        self.assertEqual(subject["sha256sums_sha256"], self.args.sha256sums_sha256)
        self.assertEqual(subject["artifacts"], [{"name": name, "sha256": digest}
                                               for name, digest in sorted(identity["archives"].items())])

    def test_gate_rejects_wrong_expected_pin_tag_revision_and_row(self):
        self.prepare()
        for key, value in (("prepared_sha256sums", "0" * 64), ("prepared_tag", "v1.2.4"),
                           ("prepared_revision", "b" * 40), ("row", "linux-arm64")):
            with self.subTest(key=key):
                arguments = self.gate_arguments()
                setattr(arguments, key, value)
                with self.assertRaises(GATE.Invalid):
                    GATE.Builder(arguments, None).subject()

    def test_gate_rejects_mutated_extracted_binary(self):
        self.prepare()
        binary = self.output / "bundle/axiom"
        binary.chmod(0o700)
        binary.write_bytes(b"changed")
        binary.chmod(0o500)
        with self.assertRaises(GATE.Invalid):
            GATE.Builder(self.gate_arguments(), None).subject()

    def test_gate_command_records_candidate_acceptance_only_for_that_gate(self):
        arguments = self.gate_arguments()
        arguments.previous, arguments.poc_binary = [], ""
        self.assertNotIn("--candidate-acceptance", GATE.Builder(arguments, None).command())
        arguments.gate = "candidate-acceptance"
        command = GATE.Builder(arguments, None).command()
        self.assertEqual(command[command.index("{subject-sha256sums}") + 1], "--candidate-acceptance")

    def test_gate_rejects_prepared_claims_without_identity(self):
        arguments = self.gate_arguments()
        arguments.prepared_identity = ""
        with self.assertRaisesRegex(GATE.Invalid, "verified materialization"):
            GATE.Builder(arguments, None).subject()

    def test_source_extraction_accepts_historical_three_row_set(self):
        sums = self.source / "SHA256SUMS"
        sums.write_bytes(b"".join(line for line in sums.read_bytes().splitlines(keepends=True) if b"windows" not in line))
        (self.source / "axiom-1.2.3-windows-amd64.tar.gz").unlink()
        bundle = self.extract_source()
        self.assertEqual(bundle, self.output / "axiom-1.2.3-linux-amd64")
        self.assertEqual((bundle / "axiom").read_bytes(), b"original harmless binary")
        self.assertEqual((bundle / "axiom").stat().st_mode & 0o777, 0o500)
        self.assertFalse(any(path.name.startswith(".source-snapshot-") for path in self.output.iterdir()))
        self.run.assert_not_called()

    def test_source_extraction_refuses_output_inside_input(self):
        arguments = argparse.Namespace(extract_source=str(self.source), output=str(self.source), row=self.args.row)
        before = {p.name: p.read_bytes() for p in self.source.iterdir()}
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "disjoint"):
            CANDIDATE.extract_source(arguments)
        self.assertEqual({p.name: p.read_bytes() for p in self.source.iterdir()}, before)

    def test_source_extraction_rejects_file_ancestor_collision_before_writing(self):
        prefix = "axiom-1.2.3-linux-amd64"
        archive_path = self.source / (prefix + ".tar.gz")
        with tarfile.open(archive_path, "w:gz") as archive:
            for path in (prefix + "/skills", prefix + "/skills/example/SKILL.md"):
                member = tarfile.TarInfo(path)
                member.size = 1
                archive.addfile(member, io.BytesIO(b"x"))
        self.update_pin()
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "file and directory"):
            self.extract_source()
        self.assertEqual(list(self.output.iterdir()), [])

    def test_source_extraction_rejects_checksum_corruption(self):
        (self.source / "axiom-1.2.3-linux-amd64.tar.gz").write_bytes(b"changed")
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "checksum mismatch"):
            self.extract_source()
        self.assertEqual(list(self.output.iterdir()), [])

    def test_source_extraction_rejects_duplicate_row_archives(self):
        (self.source / "axiom-1.2.4-linux-amd64.tar.gz").write_bytes(b"other row")
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "exactly one archive"):
            self.extract_source()
        self.assertEqual(list(self.output.iterdir()), [])

    def test_source_extraction_preserves_existing_bundle(self):
        bundle = self.output / "axiom-1.2.3-linux-amd64"
        bundle.mkdir(mode=0o700)
        (bundle / "axiom").write_bytes(b"preserve")
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "already exists"):
            self.extract_source()
        self.assertEqual((bundle / "axiom").read_bytes(), b"preserve")

    def test_source_extraction_rejects_source_symlink(self):
        archive = self.source / "axiom-1.2.3-linux-amd64.tar.gz"
        archive.rename(self.root / "original")
        archive.symlink_to(self.root / "original")
        with self.assertRaises(OSError):
            self.extract_source()
        self.assertEqual(list(self.output.iterdir()), [])

    def test_source_extraction_rejects_ancestor_symlink(self):
        link = self.root / "alias"
        link.symlink_to(self.root, target_is_directory=True)
        arguments = argparse.Namespace(extract_source=str(link / "prepared"), output=str(self.output), row=self.args.row)
        with self.assertRaises(OSError):
            CANDIDATE.extract_source(arguments)
        self.assertEqual(list(self.output.iterdir()), [])

    def test_source_extraction_rejects_unsafe_members_before_writing_bundle(self):
        prefix = "axiom-1.2.3-linux-amd64"
        for path, kind in ((prefix + "/../../prepared/bundle/axiom", tarfile.REGTYPE),
                           ("/absolute", tarfile.REGTYPE), (prefix + "//axiom", tarfile.REGTYPE),
                           (prefix + "/axiom", tarfile.SYMTYPE), (prefix + "/axiom", tarfile.LNKTYPE),
                           (prefix + "/fifo", tarfile.FIFOTYPE)):
            with self.subTest(path=path, kind=kind):
                member = tarfile.TarInfo(path)
                member.type = kind
                member.linkname = "/foreign"
                self.set_selected_members([member])
                with self.assertRaises(CANDIDATE.CandidateError):
                    self.extract_source()
                self.assertEqual(list(self.output.iterdir()), [])
                self.assertFalse((self.root / "prepared/bundle/axiom").exists())

    def test_source_extraction_rejects_duplicate_entries(self):
        member = tarfile.TarInfo("axiom-1.2.3-linux-amd64/axiom")
        member.size = 1
        self.set_selected_members([member, member])
        with self.assertRaisesRegex(CANDIDATE.CandidateError, "duplicate archive entry"):
            self.extract_source()
        self.assertEqual(list(self.output.iterdir()), [])

    def test_source_extraction_creates_missing_safe_directories(self):
        member = tarfile.TarInfo("axiom-1.2.3-linux-amd64/skills/example/SKILL.md")
        member.size = 1
        self.set_selected_members([member])
        bundle = self.extract_source()
        self.assertEqual((bundle / "skills/example/SKILL.md").read_bytes(), b"x")


if __name__ == "__main__":
    unittest.main()
