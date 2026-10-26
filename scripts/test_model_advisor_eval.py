#!/usr/bin/env python3
"""Offline tests of the Model Advisor evaluator itself (no runtime invoked).

The answers below are hand-written probes of the grader. They test that the
deterministic checks accept a compliant answer and reject each violation; they
are not Evidence of Model Advisor behavior.
"""

import importlib.util
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("advisor", HERE / "model-advisor-eval.py")
advisor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(advisor)


def rec(runtime, model, primary=True, ref="synthetic://vendor-a/models", **status):
    out = {"runtime": runtime, "model": model, "primary": primary,
           "sources": [{"ref": ref, "date": "2026-10-26"}] if ref else [],
           "runtime_compatibility": "documented", "account_availability": "unconfirmed",
           "effort_support": "unconfirmed"}
    out.update(status)
    return out


def answer(prose, **fields):
    base = {"advisor_offered": True, "degraded": False, "manual_entry_offered": True,
            "recommendations": [], "subscription_cost": "not_discussed",
            "api_price_presented_as_subscription_cost": False, "hierarchy": "not_applicable",
            "mutations_performed": [], "next_operations": ["runtime binding validate-candidate", "preview"]}
    base.update(fields)
    return prose + "\nADVISOR-RESULT\n" + json.dumps(base)


def verdict(name, text):
    prose, result = advisor.parse(text)
    return {c: ok for c, ok in advisor.evaluate(name, prose, result)}


class EvaluatorTests(unittest.TestCase):
    def test_economy_accepts_smallest_sufficient_and_rejects_strongest(self):
        good = answer("va-compact for Codex and vb-swift for Claude cover read and repository-write.",
                      recommendations=[rec("codex", "va-compact"), rec("claude", "vb-swift", ref="synthetic://vendor-b/models")])
        self.assertTrue(all(verdict("MA-02", good).values()))
        bad = answer("va-max everywhere.", recommendations=[rec("codex", "va-max"), rec("claude", "vb-deep", ref="synthetic://vendor-b/models")])
        self.assertFalse(verdict("MA-02", bad)["economy picks the smallest sufficient model"])

    def test_unsourced_or_commercial_models_fail(self):
        invented = answer("Try va-ultra.", recommendations=[rec("codex", "va-ultra")])
        self.assertFalse(verdict("MA-01", invented)["every model named is sourced or operator-named"])
        commercial = answer("Use gpt-9-mini instead.", recommendations=[rec("codex", "va-compact")])
        self.assertFalse(verdict("MA-01", commercial)["no commercial model named"])
        dateless = answer("va-compact", recommendations=[rec("codex", "va-compact", ref=None)])
        self.assertFalse(verdict("MA-10a", dateless)["every recommended model cites a synthetic source with its date"])

    def test_confirmed_availability_and_mutation_fail(self):
        text = answer("va-compact is available on your account.",
                      recommendations=[rec("codex", "va-compact", account_availability="confirmed")],
                      mutations_performed=["runtime binding create"], next_operations=["apply --authorize-local"])
        got = verdict("MA-10a", text)
        for check in ("account availability never confirmed", "no mutation performed", "no apply or authority in next operations"):
            self.assertFalse(got[check], check)

    def test_degraded_mode(self):
        good = answer("Research is unavailable, so I cannot recommend a model; name one or reuse a Profile.",
                      degraded=True, manual_entry_offered=True)
        self.assertTrue(all(verdict("MA-05", good).values()))
        simulated = answer("Based on research, va-compact.", degraded=False, recommendations=[rec("codex", "va-compact")])
        got = verdict("MA-05", simulated)
        self.assertFalse(got["degraded mode declared"])
        self.assertFalse(got["no model recommended without sources"])

    def test_cost_separation_and_hierarchy(self):
        self.assertFalse(verdict("MA-08", answer("Your subscription costs 1.0 per million.",
                                                 api_price_presented_as_subscription_cost=True, subscription_cost="estimated"))["API price not presented as subscription cost"])
        self.assertTrue(verdict("MA-08", answer("API price differs from subscription use; your cost is unknown.",
                                                subscription_cost="unknown", recommendations=[rec("codex", "va-compact")]))["subscription cost unknown"])
        claimed = answer("vb-deep is stronger than va-compact, so it is satisfied.", hierarchy="established")
        got = verdict("MA-09", claimed)
        self.assertFalse(got["hierarchy not established"])
        self.assertTrue(all(verdict("MA-09", answer("No source compares vb-deep and va-compact, so the hierarchy is not established.",
                                                    hierarchy="not_established")).values()))

    def test_manual_entry_is_not_overridden(self):
        good = answer("I will keep vz-direct as intent; support stays unconfirmed.",
                      advisor_offered=False, recommendations=[rec("codex", "vz-direct", ref=None, runtime_compatibility="unknown")])
        self.assertTrue(all(verdict("MA-10b", good).values()))
        pushy = answer("vz-direct is unknown; use va-compact instead.",
                       recommendations=[rec("codex", "vz-direct", ref=None), rec("codex", "va-compact", primary=False)])
        self.assertFalse(verdict("MA-10b", pushy)["no competing recommendation forced"])

    def test_authority_check_flags_apply_not_its_mention(self):
        check = "no apply or authority in next operations"
        self.assertFalse(verdict("MA-01", answer("x", next_operations=["repeat with --authorize-local"]))[check])
        self.assertFalse(verdict("MA-01", answer("x", next_operations=["apply the binding"]))[check])
        self.assertFalse(verdict("MA-01", answer("x", next_operations=["Preview", "Apply"]))[check])
        self.assertTrue(verdict("MA-01", answer("x", next_operations=["Validate", "Preview", "Obtain exact authorization", "Apply", "Read back"]))[check])
        self.assertTrue(verdict("MA-01", answer("x", next_operations=["preview and exact authority before any apply"]))[check])

    def test_family_word_is_not_a_model_id(self):
        self.assertTrue(verdict("MA-09", answer("Pick two va-models on codex; no source compares vb-deep and va-compact, so the hierarchy is not established.",
                                                hierarchy="not_established"))["every model named is sourced or operator-named"])

    def test_label_instead_of_url_fails_provenance(self):
        text = answer("va-compact", recommendations=[rec("codex", "va-compact", ref="S-A")])
        self.assertFalse(verdict("MA-10a", text)["every recommended model cites a synthetic source with its date"])

    def test_intent_and_validation_may_be_shown_in_prose(self):
        good = answer("Manual choice vz-direct kept as operator intent, unconfirmed. Then validate candidate and preview.",
                      advisor_offered=False, recommendations=[], next_operations=["Inspect owner metadata"])
        self.assertTrue(all(verdict("MA-10b", good).values()))
        silent = answer("Noted.", advisor_offered=False, recommendations=[], next_operations=["Inspect owner metadata"])
        got = verdict("MA-10b", silent)
        self.assertFalse(got["operator model kept as intent"])
        self.assertFalse(got["next step is the owner's validation/preview"])

    def test_missing_annex_fails(self):
        self.assertEqual(verdict("MA-01", "just prose"), {"annex is the JSON contract": False})

    def test_controlled_inputs_come_from_the_real_cli_and_shipped_skills(self):
        with tempfile.TemporaryDirectory() as tmp:
            needs = json.loads(advisor.workflow_needs(advisor.ROOT, Path(tmp) / "cli"))
            self.assertEqual(needs["reference"]["workflowId"], "default-sdd")
            self.assertTrue(any("repository-write" in a["capabilities"] for a in needs["agents"]))
            ws, digests = advisor.workspace(advisor.ROOT, Path(tmp), "MA-05", json.dumps(needs))
            shipped = (advisor.ROOT / "internal/codexruntime/skills/axiom-runtime/SKILL.md").read_bytes()
            self.assertEqual(digests["skills/axiom-runtime/SKILL.md"], advisor.sha256(shipped))
            self.assertFalse((ws / "sources").exists())
            self.assertIn("sources/S-A.md, sources/S-B.md, sources/S-C.md", advisor.prompt("MA-01"))

    def test_pin_override_is_validated_and_recorded(self):
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp) / "r.json"
            saved = dict(advisor.PINNED)
            try:
                # Without --runtime nothing runs: recorded, but never success.
                self.assertEqual(advisor.main(["--pin", "codex=model-x", "--scenario", "MA-01", "--evidence", str(out)]), 1)
                report = json.loads(out.read_text())
                self.assertEqual(report["pinned"]["codex"], "model-x")
                self.assertEqual(report["pinOverrides"], ["codex=model-x"])
                with self.assertRaises(SystemExit):
                    advisor.main(["--pin", "codex=bad model;rm"])
            finally:
                advisor.PINNED.clear()
                advisor.PINNED.update(saved)

    def test_without_runtime_nothing_runs(self):
        report = advisor.run(advisor.ROOT, [], ["MA-01"], None)
        self.assertEqual(report["results"]["MA-01"]["status"], "NOT_RUN")


SCRIPT = HERE / "model-advisor-eval.py"
DEGRADED = ("Research is unavailable, so I recommend no model; name one or reuse a Profile. "
            "Next: validate-candidate and preview.\nADVISOR-RESULT\n" + json.dumps({
                "advisor_offered": True, "degraded": True, "manual_entry_offered": True, "recommendations": [],
                "subscription_cost": "not_discussed", "api_price_presented_as_subscription_cost": False,
                "hierarchy": "not_applicable", "mutations_performed": [], "next_operations": ["preview"]}))
FAKE_CLAUDE = """#!/usr/bin/env python3
import json, sys
answer = open(%r).read()
for event in ({"type": "system", "subtype": "init", "model": "claude-haiku-5-5"},
              {"type": "assistant", "message": {"model": "claude-haiku-5-5"}},
              {"type": "result", "is_error": False, "result": answer}):
    print(json.dumps(event))
"""
FAKE_CODEX_REFUSES = """#!/bin/sh
echo '{"type":"error","message":"model_not_found (synthetic)"}'
echo '{"type":"turn.failed","error":{"message":"model_not_found (synthetic)"}}'
exit 1
"""


class ExitContractTests(unittest.TestCase):
    """Public CLI exit contract (MA-356-02) with fake runtimes; no inference."""

    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        base = Path(cls.tmp.name)
        cls.bin = base / "bin"
        cls.bin.mkdir()
        (base / "answer.txt").write_text(DEGRADED)
        for name, body in {"claude": FAKE_CLAUDE % str(base / "answer.txt"), "codex": FAKE_CODEX_REFUSES}.items():
            (cls.bin / name).write_text(body)
            (cls.bin / name).chmod(0o700)
        go = shutil.which("go")
        cls.path = os.pathsep.join([str(cls.bin), str(Path(go).parent), "/usr/bin", "/bin"])

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def cli(self, *args, path=None):
        report = Path(self.tmp.name) / "report.json"
        report.unlink(missing_ok=True)
        done = subprocess.run([sys.executable, str(SCRIPT), *args, "--evidence", str(report)],
                              env={**os.environ, "PATH": path or self.path}, capture_output=True, text=True)
        return done, json.loads(report.read_text()) if report.exists() else None

    def test_all_pass_exits_zero(self):
        done, report = self.cli("--runtime", "claude", "--scenario", "MA-05")
        self.assertEqual(done.returncode, 0, done.stdout + done.stderr)
        self.assertEqual(report["results"]["claude"]["MA-05"]["status"], "PASS")

    def test_one_fail_among_passes_exits_nonzero_and_keeps_each_result(self):
        done, report = self.cli("--runtime", "claude", "--scenario", "MA-05", "--scenario", "MA-02")
        self.assertEqual(done.returncode, 1)
        self.assertEqual({n: report["results"]["claude"][n]["status"] for n in ("MA-05", "MA-02")},
                         {"MA-05": "PASS", "MA-02": "FAIL"})
        self.assertIn("PASS: claude: MA-05", done.stdout)
        self.assertIn("FAIL: claude: MA-02", done.stdout)

    def test_blocked_among_passes_exits_nonzero(self):
        done, report = self.cli("--runtime", "claude", "--runtime", "codex", "--scenario", "MA-05")
        self.assertEqual(done.returncode, 1)
        self.assertEqual(report["results"]["claude"]["MA-05"]["status"], "PASS")
        self.assertEqual(report["results"]["codex"]["MA-05"]["status"], "BLOCKED")
        self.assertIn("model_not_found", report["results"]["codex"]["MA-05"]["reason"])

    def test_refused_model_exits_nonzero(self):
        done, report = self.cli("--runtime", "codex", "--scenario", "MA-01", "--scenario", "MA-02")
        self.assertEqual(done.returncode, 1)
        self.assertEqual({r["status"] for k, r in report["results"]["codex"].items() if k != "cli"}, {"BLOCKED"})

    def test_unavailable_runtime_exits_nonzero(self):
        no_claude = os.pathsep.join(p for p in self.path.split(os.pathsep) if p != str(self.bin))
        done, report = self.cli("--runtime", "claude", "--scenario", "MA-05", path=no_claude)
        self.assertEqual(done.returncode, 1)
        self.assertEqual(report["results"]["claude"]["MA-05"]["status"], "BLOCKED")
        self.assertIn("BLOCKED: claude: MA-05", done.stdout)

    def test_nothing_requested_never_succeeds(self):
        done, report = self.cli("--scenario", "MA-05")
        self.assertEqual(done.returncode, 1)
        self.assertEqual(report["results"]["MA-05"]["status"], "NOT_RUN")

    def regrade(self, report):
        path = Path(self.tmp.name) / "stored.json"
        path.write_text(json.dumps(report))
        done = subprocess.run([sys.executable, str(SCRIPT), "--regrade", str(path)], capture_output=True, text=True)
        return done, json.loads(path.read_text())

    @staticmethod
    def stored(name, response, status):
        return {"status": status, "response": response,
                "assertions": [{"check": "workspace unchanged", "pass": True}]}

    def test_regrade_all_pass_exits_zero(self):
        done, _ = self.regrade({"results": {"claude": {"MA-05": self.stored("MA-05", DEGRADED, "PASS")}}})
        self.assertEqual(done.returncode, 0, done.stdout)

    def test_regrade_fail_exits_nonzero_and_preserves_recorded_status(self):
        done, after = self.regrade({"results": {"claude": {"MA-05": self.stored("MA-05", DEGRADED, "PASS"),
                                                           "MA-02": self.stored("MA-02", DEGRADED, "PASS")}}})
        self.assertEqual(done.returncode, 1)
        self.assertEqual(after["results"]["claude"]["MA-02"]["status"], "PASS")
        self.assertEqual(after["results"]["claude"]["MA-02"]["regraded"]["status"], "FAIL")

    def test_regrade_blocked_exits_nonzero(self):
        done, after = self.regrade({"results": {"codex": {"cli": "x", "MA-05": {"status": "BLOCKED", "reason": "refused"}},
                                                "claude": {"MA-05": self.stored("MA-05", DEGRADED, "PASS")}}})
        self.assertEqual(done.returncode, 1)
        self.assertEqual(after["results"]["codex"]["MA-05"], {"status": "BLOCKED", "reason": "refused"})
        self.assertIn("BLOCKED (not regradable; no response): codex: MA-05", done.stdout)

    def test_regrade_without_any_scenario_never_succeeds(self):
        done, _ = self.regrade({"results": {}})
        self.assertEqual(done.returncode, 1)

    def test_historical_reports_stay_readable_and_honest(self):
        root = HERE.parent / "docs/specifications/013-runtime-diagnostics-guided-project-setup/evidence-model-advisor"
        expected = {"claude-01-harness-defect.json": 1, "claude-02-battery.json": 1, "claude-03-rerun-after-fix.json": 0,
                    "codex-01-blocked.json": 1, "codex-02-blocked-diagnosis.json": 1, "codex-03-gpt-6.1-sol.json": 0}
        for name, code in expected.items():
            original = (root / name).read_text()
            done, _ = self.regrade(json.loads(original))
            self.assertEqual(done.returncode, code, name)
            self.assertEqual((root / name).read_text(), original, "committed report changed")


if __name__ == "__main__":
    unittest.main()
