#!/usr/bin/env python3
"""Offline trust-boundary regressions; native execution is separate Evidence."""
import importlib.util
import io
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("native", Path(__file__).with_name("accept-native-artifact.py"))
native = importlib.util.module_from_spec(spec)
spec.loader.exec_module(native)


class NativeContract(unittest.TestCase):
    def test_untrusted_baseline_archives(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / "axiom-0.10.0-linux-arm64.tar.gz"
            for name, kind in (("../escape", tarfile.REGTYPE),
                               ("axiom-0.10.0-linux-arm64/link", tarfile.SYMTYPE),
                               ("axiom-0.10.0-linux-arm64/../escape", tarfile.REGTYPE)):
                with self.subTest(name=name):
                    with tarfile.open(archive, "w:gz") as tar:
                        member = tarfile.TarInfo(name)
                        member.type = kind
                        member.size = 1 if kind == tarfile.REGTYPE else 0
                        tar.addfile(member, io.BytesIO(b"x"))
                    (root / "SHA256SUMS").write_text(f"{native.smoke.digest(archive)}  {archive.name}\n")
                    with self.assertRaisesRegex(ValueError, "unsafe archive entry"):
                        native.extract(root, "0.10.0", "linux-arm64", root / "extract")
            (root / "SHA256SUMS").write_text("invalid\n")
            with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                native.extract(root, "0.10.0", "linux-arm64", root / "extract")

    def test_strict_gate_and_no_publication(self):
        root = Path(__file__).resolve().parent.parent
        prepare = (root / ".github/workflows/release-artifacts.yml").read_text()
        native_workflow = (root / ".github/workflows/native-artifact-acceptance.yml").read_text()
        publish = (root / "scripts/verify-prepared-release.sh").read_text()
        qualifier = (root / ".github/workflows/native-acceptance-qualification.yml").read_text()
        self.assertIn("native-acceptance:\n    needs: prepare", prepare)
        self.assertIn("uses: ./.github/workflows/native-artifact-acceptance.yml", prepare)
        self.assertIn("runs-on: ubuntu-24.04-arm", native_workflow)
        self.assertIn("runs-on: windows-2022", native_workflow)
        self.assertIn("--row windows-amd64", native_workflow)
        self.assertIn(".conclusion == \"success\"", publish)
        self.assertIn('test "$PROMOTION" = skipped', qualifier)
        for workflow in (prepare, native_workflow, qualifier):
            self.assertNotIn("continue-on-error", workflow)
            self.assertNotIn("contents: write", workflow)
            self.assertNotIn("secrets.", workflow)
        self.assertNotIn("reject_control:", prepare)
        self.assertNotIn("setup-go", native_workflow)
        self.assertNotIn("build-release-archives", native_workflow)


if __name__ == "__main__":
    unittest.main()
