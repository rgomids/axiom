#!/usr/bin/env python3
"""Offline refusal/security contracts; real prepared bytes run in pipeline test."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("smoke", Path(__file__).with_name("smoke-prepared-artifact.py"))
smoke = importlib.util.module_from_spec(spec)
spec.loader.exec_module(smoke)


class SmokeContract(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.revision = "a" * 40
        self.bundle = "axiom-1.2.3-linux-amd64"
        self.archive = self.root / f"{self.bundle}.tar.gz"
        self.binary = b"#!/bin/sh\nexit 1\n"
        self.pack()

    def pack(self, name=None, mode=0o700, kind=tarfile.REGTYPE, archive_format="1"):
        with tarfile.open(self.archive, "w:gz") as archive:
            member = tarfile.TarInfo(name or f"{self.bundle}/axiom")
            member.mode = mode
            member.type = kind
            member.linkname = "/outside"
            member.size = len(self.binary) if kind == tarfile.REGTYPE else 0
            archive.addfile(member, io.BytesIO(self.binary) if member.size else None)
            metadata = f"formatVersion={archive_format}\n".encode()
            member = tarfile.TarInfo(f"{self.bundle}/release-metadata.txt")
            member.mode = 0o600
            member.size = len(metadata)
            archive.addfile(member, io.BytesIO(metadata))
        (self.root / "SHA256SUMS").write_text(f"{smoke.digest(self.archive)}  {self.archive.name}\n")

    def invoke(self, *extra, responder=None):
        output = io.StringIO()
        argv = ["smoke", "--dir", str(self.root), "--version", "1.2.3", "--revision", self.revision,
                "--row", "linux-amd64", *extra]
        with patch.object(sys, "argv", argv), contextlib.redirect_stdout(output), \
                patch.object(smoke.platform, "system", return_value="Linux"), \
                patch.object(smoke.platform, "machine", return_value="x86_64"), \
                patch.object(smoke.platform, "platform", return_value="fixture"):
            if responder:
                with patch.object(smoke.subprocess, "run", side_effect=responder):
                    code = smoke.main()
            else:
                code = smoke.main()
        return code, json.loads(output.getvalue())

    def event(self, command, **kwargs):
        self.assertEqual(kwargs["timeout"], 20)
        self.assertEqual(kwargs["stdin"], subprocess.DEVNULL)
        self.assertNotIn("GH_TOKEN", kwargs["env"])
        self.assertNotIn("CLAUDE_CONFIG_DIR", kwargs["env"])
        self.assertEqual(list(Path(kwargs["env"]["PATH"]).iterdir()), [])
        event = {"status": "success"}
        if command[2] == "version":
            event["provenance"] = {"product": "Axiom", "version": "1.2.3",
                                   "revision": self.revision[:12], "sourceState": "clean"}
        elif command[3] == "configure":
            event["setup"] = {"projectId": "id", "digest": "digest"}
            if "--authorize-local" in command:
                Path(kwargs["env"]["LINGO_PROJECTS_ROOT"]).mkdir()
                Path(kwargs["env"]["LINGO_STATE_ROOT"]).mkdir()
        elif command[3] == "show":
            event["project"] = {"slug": "release-smoke"}
        elif command[3] == "list":
            event["projects"] = [{"slug": "release-smoke"}]
        kwargs["stdout"].write(json.dumps(event).encode())
        return subprocess.CompletedProcess(command, 0)

    def test_success_and_exact_input_identity(self):
        before = smoke.inventory(self.root)
        code, summary = self.invoke(responder=self.event)
        self.assertEqual(code, 0)
        self.assertEqual(summary["result"], "pass")
        self.assertEqual(summary["archiveSha256"], smoke.digest(self.archive))
        self.assertTrue(summary["inputsUnchanged"])
        self.assertEqual(before, smoke.inventory(self.root))

    def test_v2_smoke_and_compression_refusals_before_execution(self):
        import gzip
        for suffix in (b"", gzip.compress(b""), b"garbage"):
            self.pack(archive_format="2")
            with self.archive.open("ab") as stream:
                stream.write(suffix)
            (self.root / "SHA256SUMS").write_text(f"{smoke.digest(self.archive)}  {self.archive.name}\n")
            calls = []
            def respond(command, **kwargs):
                calls.append(command)
                return self.event(command, **kwargs)
            code, _ = self.invoke("--format", "2", responder=respond)
            self.assertEqual(code, 1 if suffix else 0)
            self.assertEqual(bool(calls), not bool(suffix))

    def test_generation_token_and_metadata_refused(self):
        for extra in (("--format", "2"), ("--format", "2", "--row", "macos-27-arm64")):
            with patch.object(smoke.subprocess, "run") as execute:
                code, _ = self.invoke(*extra)
                self.assertEqual(code, 1)
                execute.assert_not_called()

    def test_wrong_version_and_revision(self):
        for field, value in (("version", "9.9.9"), ("revision", "b" * 12), ("sourceState", "dirty")):
            def wrong(command, **kwargs):
                result = self.event(command, **kwargs)
                if command[2] == "version":
                    event = {"status": "success", "provenance": {
                        "product": "Axiom", "version": "1.2.3", "revision": self.revision[:12], "sourceState": "clean"}}
                    event["provenance"][field] = value
                    kwargs["stdout"].seek(0)
                    kwargs["stdout"].truncate()
                    kwargs["stdout"].write(json.dumps(event).encode())
                return result
            with self.subTest(field=field):
                code, summary = self.invoke(responder=wrong)
                self.assertEqual(code, 1)
                self.assertIn("version/revision mismatch", summary["error"])

    def test_non_executable_and_command_failure(self):
        code, summary = self.invoke()
        self.assertEqual(code, 1)
        self.assertIn("CLI command failed", summary["error"])
        self.pack(mode=0o600)
        code, summary = self.invoke()
        self.assertEqual(code, 1)
        self.assertIn("not executable", summary["error"])

    def test_ambiguous_checksum_refused_before_execution(self):
        with (self.root / "SHA256SUMS").open("a") as stream:
            stream.write(f"{'0' * 64}  {self.archive.name}\n")
        with patch.object(smoke.subprocess, "run") as execute:
            code, summary = self.invoke()
            self.assertEqual(code, 1)
            self.assertIn("checksum mismatch", summary["error"])
            execute.assert_not_called()

    def test_corrupted_archive(self):
        with self.archive.open("ab") as stream:
            stream.write(b"changed")
        code, summary = self.invoke()
        self.assertEqual(code, 1)
        self.assertIn("checksum mismatch", summary["error"])

    def test_archive_paths_and_links(self):
        for name, kind in (("../outside", tarfile.REGTYPE),
                           (f"{self.bundle}/../outside", tarfile.REGTYPE),
                           (f"/{self.bundle}/axiom", tarfile.REGTYPE),
                           (f"{self.bundle}/axiom", tarfile.SYMTYPE),
                           (f"{self.bundle}/axiom", tarfile.LNKTYPE)):
            with self.subTest(name=name, kind=kind):
                self.pack(name=name, kind=kind)
                code, summary = self.invoke()
                self.assertEqual(code, 1)
                self.assertIn("unsafe prepared archive", summary["error"])

    def test_mutated_inputs_fail_even_after_cli_failure(self):
        def mutate(command, **kwargs):
            (self.root / "SHA256SUMS").write_text("tamper")
            return subprocess.CompletedProcess(command, 1)
        code, summary = self.invoke(responder=mutate)
        self.assertEqual(code, 1)
        self.assertFalse(summary["inputsUnchanged"])
        self.assertIn("digests changed", summary["error"])

    def test_timeout_is_failure_with_summary(self):
        def timeout(command, **kwargs):
            raise subprocess.TimeoutExpired(command, kwargs["timeout"])
        code, summary = self.invoke(responder=timeout)
        self.assertEqual(code, 1)
        self.assertEqual(summary["error"], "TimeoutExpired")
        self.assertTrue(summary["inputsUnchanged"])

    def test_lifecycle_failure_cannot_pass(self):
        def fail(command, **kwargs):
            if command[2:4] == ["project", "show"]:
                return subprocess.CompletedProcess(command, 1)
            return self.event(command, **kwargs)
        code, summary = self.invoke(responder=fail)
        self.assertEqual(code, 1)
        self.assertIn("project show", summary["error"])

    def test_foreign_host_refused(self):
        code, summary = self.invoke("--row", "macos-27-arm64")
        self.assertEqual(code, 1)
        self.assertIn("native host", summary["error"])

    def test_missing_project_and_binary_mutation_refused(self):
        for failure in ("missing-project", "changed-binary"):
            def incomplete(command, **kwargs):
                result = self.event(command, **kwargs)
                if command[2:4] == ["project", "list"]:
                    if failure == "missing-project":
                        kwargs["stdout"].seek(0)
                        kwargs["stdout"].truncate()
                        kwargs["stdout"].write(b'{"status":"success","projects":[]}')
                    else:
                        with Path(command[0]).open("ab") as stream:
                            stream.write(b"tamper")
                return result
            with self.subTest(failure=failure):
                code, summary = self.invoke(responder=incomplete)
                self.assertEqual(code, 1)
                self.assertTrue(summary["inputsUnchanged"])
                self.assertIn("failed to list" if failure == "missing-project" else "binary changed", summary["error"])


if __name__ == "__main__":
    unittest.main()
