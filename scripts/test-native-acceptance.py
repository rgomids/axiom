#!/usr/bin/env python3
"""Offline trust-boundary regressions; native execution is separate Evidence."""
import importlib.util
import io
import os
import re
import subprocess
import textwrap
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest
from unittest import mock

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("native", Path(__file__).with_name("accept-native-artifact.py"))
native = importlib.util.module_from_spec(spec)
spec.loader.exec_module(native)


class NativeContract(unittest.TestCase):
    def test_unsupported_rows_refused(self):
        for row in ("windows-amd64", "macos-27-arm64"):
            with self.subTest(row=row), self.assertRaises(SystemExit), \
                    mock.patch.object(sys, "argv", ["accept-native-artifact.py", "--dir", "/x", "--version", "0.10.0",
                                                    "--revision", "0" * 40, "--row", row, "--log", "/y"]), \
                    mock.patch.object(sys, "stderr", io.StringIO()):
                native.main()

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

    def test_upgrade_harness_revision_boundary(self):
        root = Path(__file__).resolve().parent.parent
        workflow = (root / ".github/workflows/native-suite.yml").read_text()
        prepare = (root / ".github/workflows/release-artifacts.yml").read_text()
        source_jobs, journeys = workflow.split("\n  upgrade-journeys:\n")
        self.assertEqual(source_jobs.count("ref: ${{ inputs.ref }}"), 2)
        self.assertNotIn("inputs.journey_ref", source_jobs)
        self.assertIn("default: ''", source_jobs)
        self.assertIn("journey_ref: ${{ github.sha }}", prepare)
        self.assertIn("ref: ${{ inputs.revision }}", prepare)
        self.assertNotIn("validation_revision", prepare)
        self.assertIn("needs: prepare", prepare.split("\n  native-suite:\n")[1])
        for caller in ("nightly.yml", "native-acceptance-qualification.yml"):
            self.assertNotIn("journey_ref:", (root / ".github/workflows" / caller).read_text())
        self.assertIn("if: inputs.journey_ref != ''", journeys)
        self.assertIn("ref: ${{ inputs.journey_ref }}", journeys)
        self.assertIn("path: harness", journeys)
        self.assertIn("working-directory: ${{ inputs.journey_ref != '' && 'harness' || '.' }}", journeys)
        self.assertLess(journeys.index("Build the historical POC binary"), journeys.index("Checkout upgrade harness"))
        self.assertEqual(journeys.count("ref: ${{ inputs.ref }}"), 1)
        self.assertNotIn("continue-on-error", journeys)
        self.assertNotIn("if:", journeys.split("      - name: Upgrade journeys\n")[1])

        def command(name):
            step = journeys.split("      - name: " + name + "\n", 1)[1].split("      - ", 1)[0]
            run = re.search(r"^        run: (.*)$", step, re.M)
            return textwrap.dedent(step[run.end() + 1:]) if run[1] == "|" else run[1]

        # Execute the actual workflow build/journey commands with two Git
        # revisions. Tiny fixture scripts expose checkout identity and failure
        # propagation; real native journeys are separate acceptance Evidence.
        with tempfile.TemporaryDirectory() as temporary:
            tmp = Path(temporary)
            origin, checkout = tmp / "origin", tmp / "source"
            origin.mkdir()

            def git(*args, cwd=origin):
                return subprocess.check_output(["git", *args], cwd=cwd, text=True, stderr=subprocess.DEVNULL).strip()

            git("init", "-q")
            git("config", "user.name", "Fixture")
            git("config", "user.email", "fixture@example.invalid")
            scripts = origin / "scripts"
            scripts.mkdir()
            builder = scripts / "build-release-archives.sh"
            builder.write_text('#!/bin/bash\nset -eu\nmkdir -p "$4"\ngit rev-parse HEAD > "$4/source"\n')
            harness = scripts / "test-upgrade-journeys.sh"
            harness.write_text('#!/bin/bash\nexit 17\n')
            builder.chmod(0o755)
            harness.chmod(0o755)
            git("add", ".")
            git("commit", "-qm", "release")
            release = git("rev-parse", "HEAD")
            harness.write_text('#!/bin/bash\nset -eu\ntest "$(cat "$2/source")" = "$EXPECTED_SOURCE"\ngit rev-parse HEAD\nexit "${FAIL_JOURNEY:-0}"\n')
            git("add", ".")
            git("commit", "-qm", "fixed harness")
            fixed = git("rev-parse", "HEAD")
            git("merge-base", "--is-ancestor", release, fixed)
            self.assertNotEqual(release, fixed)
            git("clone", "-q", str(origin), str(checkout))
            git("checkout", "-q", release, cwd=checkout)
            env = dict(os.environ, RUNNER_TEMP=str(tmp), GITHUB_RUN_NUMBER="1",
                       GITHUB_STEP_SUMMARY=str(tmp / "summary"), PREVIOUS_LATEST="/unused",
                       PREVIOUS_v0_1_1="/unused", EXPECTED_SOURCE=release)

            def run(cmd, cwd, extra=None):
                return subprocess.run(["bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", cmd],
                                      cwd=cwd, env=dict(env, **(extra or {})), capture_output=True, text=True)

            built = run(command("Build the candidate release archives"), checkout)
            self.assertEqual(built.returncode, 0, built.stderr)
            self.assertEqual((tmp / "candidate/source").read_text().strip(), release)
            # Omitted override executes the original checkout (including failure).
            self.assertEqual(run(command("Upgrade journeys"), checkout).returncode, 17)
            git("clone", "-q", str(origin), str(checkout / "harness"))
            git("checkout", "-q", fixed, cwd=checkout / "harness")
            corrected = run(command("Upgrade journeys"), checkout / "harness")
            self.assertEqual(corrected.returncode, 0, corrected.stderr)
            self.assertEqual(corrected.stdout.strip(), fixed)
            self.assertEqual(run(command("Upgrade journeys"), checkout / "harness",
                                 {"FAIL_JOURNEY": "23"}).returncode, 23)
            self.assertEqual(git("rev-parse", "HEAD", cwd=checkout), release)

    def test_strict_gate_and_no_publication(self):
        root = Path(__file__).resolve().parent.parent
        prepare = (root / ".github/workflows/release-artifacts.yml").read_text()
        native_workflow = (root / ".github/workflows/native-artifact-acceptance.yml").read_text()
        publish = (root / "scripts/verify-prepared-release.sh").read_text()
        qualifier = (root / ".github/workflows/native-acceptance-qualification.yml").read_text()
        self.assertIn("native-acceptance:\n    needs: prepare", prepare)
        self.assertIn("uses: ./.github/workflows/native-artifact-acceptance.yml", prepare)
        self.assertIn("runs-on: ubuntu-24.04-arm", native_workflow)
        self.assertIn("--row linux-arm64", native_workflow)
        # Windows is not a supported platform: no Windows runner or row is provisioned.
        for workflow in (native_workflow, qualifier):
            self.assertNotIn("windows", workflow.lower())
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
