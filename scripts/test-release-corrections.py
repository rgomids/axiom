#!/usr/bin/env python3
"""Offline integration checks for pinned, append-only correction inputs."""
import hashlib
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class CorrectionsTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="axiom-corrections-")
        self.root = Path(self.temp.name)
        (self.root / "scripts").mkdir()
        shutil.copyfile(Path(__file__).with_name("release-corrections.sh"),
                        self.root / "scripts/release-corrections.sh")
        (self.root / ".github").mkdir()
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("config", "user.name", "Fixture")
        (self.root / "tracked").write_text("initial\n")
        self.target = self.commit()
        self.manifest = self.root / ".release-please-manifest.json"
        self.manifest.write_text('{".":"0.1.0"}\n')
        self.original_row = f"{self.target} related=1 completes=none\n"
        self.file = self.root / ".github/delivery-corrections.txt"
        self.file.write_text(self.original_row)
        self.source = self.commit()
        (self.root / "tracked").write_text("source\n")
        self.new_target = self.commit()
        self.manifest.write_text('{".":"0.2.0"}\n')
        self.source = self.commit()
        self.valid = self.original_row + f"{self.new_target} related=2,3 completes=2\n"
        self.correction = self.change(self.valid)

    def tearDown(self):
        self.temp.cleanup()

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.root), *args],
                                       stderr=subprocess.DEVNULL, text=True).strip()

    def commit(self):
        self.git("add", ".")
        self.git("commit", "-qm", "fixture", "--allow-empty")
        return self.git("rev-parse", "HEAD")

    def change(self, text):
        self.file.write_text(text)
        return self.commit()

    def invoke(self, text=None, source=None, correction=None, digest=None, main="main", extra=()):
        if text is not None:
            correction = self.change(text)
        correction = correction or self.correction
        content = self.git("show", f"{correction}:.github/delivery-corrections.txt") + "\n"
        digest = digest or hashlib.sha256(content.encode()).hexdigest()
        return subprocess.run([
            "bash", str(self.root / "scripts/release-corrections.sh"),
            "--source-revision", source or self.source,
            "--corrections-revision", correction,
            "--corrections-digest", digest, "--main-ref", main, *extra,
        ], capture_output=True, text=True)

    def rejects(self, message, **kwargs):
        result = self.invoke(**kwargs)
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertIn(message, result.stderr)

    def test_pins_exact_committed_bytes(self):
        output = self.root / "verified.txt"
        self.file.write_text("uncommitted bytes must be ignored\n")
        result = self.invoke(extra=("--file-output", str(output)))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(output.read_text(), self.valid)
        self.assertIn(f"corrections_revision={self.correction}\n", result.stdout)
        self.assertIn(hashlib.sha256(self.valid.encode()).hexdigest(), result.stdout)

    def test_bad_pins_and_refs(self):
        self.rejects("full source", source="main")
        self.rejects("unknown source", source="0" * 40)
        self.rejects("unknown main", main="missing")
        self.rejects("digest mismatch", digest="0" * 64)
        self.rejects("does not descend", source=self.correction, correction=self.source)

    def test_unmerged_correction(self):
        self.git("checkout", "-qb", "side", self.source)
        side = self.change(self.valid + "# side\n")
        self.rejects("first-parent main", correction=side)

    def test_append_only(self):
        self.rejects("changed or removed", text=self.valid.replace("related=1", "related=9"))
        self.rejects("changed or removed", text=self.valid.splitlines()[1] + "\n")

    def test_full_file_validation(self):
        for suffix, error in [
            (self.original_row, "duplicate commit"),
            ("bad row\n", "malformed"),
            (f"{'1' * 40} related=0 completes=none\n", "invalid Issue"),
            (f"{'1' * 40} related=2,2 completes=none\n", "duplicate Issue"),
            (f"{'1' * 40} related=2 completes=3\n", "missing from related"),
            (f"{'1' * 40} related=2 completes=none\n", "unknown commit"),
        ]:
            with self.subTest(error=error):
                self.rejects(error, text=self.valid + suffix)

    def test_new_target_after_source(self):
        self.rejects("outside source release range", text=self.valid +
                     f"{self.correction} related=4 completes=none\n")

    def test_pre_boundary_target_rejected(self):
        self.rejects("outside source release range", text=self.valid +
                     f"{self.git('rev-parse', self.new_target + '^1')} related=4 completes=none\n")

    def test_source_release_commit_cannot_be_corrected(self):
        self.rejects("outside source release range", text=self.valid +
                     f"{self.source} related=4 completes=none\n")

    def test_source_must_introduce_stable_version(self):
        self.rejects("not the commit", source=self.new_target)

    def test_missing_file(self):
        self.file.unlink()
        correction = self.commit()
        result = subprocess.run([
            "bash", str(self.root / "scripts/release-corrections.sh"),
            "--source-revision", self.source, "--corrections-revision", correction,
            "--corrections-digest", "0" * 64, "--main-ref", "main",
        ], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("file missing", result.stderr)

    def test_committed_symlink_rejected(self):
        self.file.unlink()
        self.file.symlink_to("../tracked")
        correction = self.commit()
        self.rejects("not regular", correction=correction)

    def test_output_symlink_rejected(self):
        output = self.root / "output"
        output.symlink_to(self.root / "tracked")
        self.rejects("regular output", extra=("--file-output", str(output)))


if __name__ == "__main__":
    unittest.main()
