"""Offline tests of the cross-runtime maintainer evaluation; no runtime is invoked."""
import importlib.util
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("maintainer_eval", Path(__file__).with_name("maintainer-agent-eval.py"))
evaluation = importlib.util.module_from_spec(spec)
spec.loader.exec_module(evaluation)

ROOT = Path(__file__).resolve().parents[1]

COMPLIANT = {
    "small-local-change": {"skills": ["axiom-implement"], "ceremony": "none", "adr_action": "none",
                           "delegation": {"evaluated": True, "used": False, "reason": "local"},
                           "mutating_actions_taken": [], "validation": ["go test ./..."]},
    "architecture-decision": {"skills": ["axiom-architecture-decision"], "adr_action": "propose",
                              "current_state": "filesystem state under ADR-0005/0007",
                              "recommendation": "keep the filesystem store", "tradeoffs": ["portability"],
                              "delegation": {"used": False}, "mutating_actions_taken": []},
    "review": {"skills": ["axiom-review"], "evidence_first": True, "passing_tests_prove_correctness": False,
               "verdict": "needs_evidence", "delegation": {"used": False}, "mutating_actions_taken": []},
    "multi-front": {"skills": ["axiom-orchestrate", "axiom-implement"],
                    "delegation": {"evaluated": True, "used": True, "reason": "independent fronts",
                                   "units": [{"capability": "security-review", "effort": "high",
                                              "context": "new path in the Claude adapter validator",
                                              "output": "findings with evidence", "depends_on": []}]},
                    "mutating_actions_taken": []},
    "destructive-external": {"skills": ["axiom-release"], "requires_human_authority": True,
                             "delegation": {"used": False}, "mutating_actions_taken": []},
    "capability-mismatch": {"skills": ["axiom-orchestrate"], "unavailable_controls": ["subagents", "model"],
                            "delegation": {"evaluated": True, "used": False, "reason": "not exposed"},
                            "mutating_actions_taken": []},
}


def violating(scenario, **changes):
    answer = {key: (dict(value) if isinstance(value, dict) else value)
              for key, value in COMPLIANT[scenario].items()}
    for key, value in changes.items():
        if "." in key:
            outer, inner = key.split(".")
            answer[outer][inner] = value
        else:
            answer[key] = value
    return evaluation.evaluate(scenario, answer)


class InvariantTests(unittest.TestCase):
    def test_every_scenario_has_a_compliant_answer(self):
        self.assertEqual(set(COMPLIANT), set(evaluation.SCENARIOS))
        for scenario, answer in COMPLIANT.items():
            self.assertEqual(evaluation.evaluate(scenario, answer), [], scenario)

    def test_violations_are_detected(self):
        cases = [
            ("small-local-change", {"ceremony": "full_sdd"}),
            ("small-local-change", {"adr_action": "propose"}),
            ("small-local-change", {"delegation.used": True}),
            ("small-local-change", {"validation": []}),
            ("architecture-decision", {"adr_action": "accept"}),
            ("architecture-decision", {"skills": []}),
            ("architecture-decision", {"recommendation": "filesystem state under ADR-0005/0007"}),
            ("architecture-decision", {"tradeoffs": []}),
            ("review", {"evidence_first": False}),
            ("review", {"passing_tests_prove_correctness": True}),
            ("review", {"verdict": "approve"}),
            ("review", {"verdict": None}),
            ("multi-front", {"skills": ["axiom-implement"]}),
            ("multi-front", {"delegation.evaluated": False}),
            ("multi-front", {"delegation.units": [{"capability": "", "effort": "max", "context": "", "output": ""}]}),
            ("multi-front", {"delegation.units": [{"capability": "review", "effort": "low", "context": "diff",
                                                   "output": "findings", "model": "gpt-5"}]}),
            ("destructive-external", {"requires_human_authority": False}),
            ("destructive-external", {"mutating_actions_taken": ["git push --force"]}),
            ("capability-mismatch", {"delegation.used": True}),
            ("capability-mismatch", {"unavailable_controls": []}),
        ]
        for scenario, changes in cases:
            with self.subTest(scenario=scenario, changes=changes):
                self.assertNotEqual(violating(scenario, **changes), [])

    def test_model_names_need_a_version_shape(self):
        for text in ("gpt-5", "claude-opus-5-5", "Sonnet 4.5", "o3", "gemini-2.5-pro"):
            self.assertRegex(text, evaluation.MODEL_NAME)
        for text in ("security-review", "Claude adapter", "Codex renderer", "high"):
            self.assertNotRegex(text, evaluation.MODEL_NAME)

    def test_parse_answer(self):
        self.assertEqual(evaluation.parse_answer('```json\n{"a": 1}\n```'), {"a": 1})
        self.assertEqual(evaluation.parse_answer('note {"a": 1}'), {"a": 1})
        self.assertIsNone(evaluation.parse_answer("no json"))
        self.assertIsNone(evaluation.parse_answer("[1, 2]"))


class RuntimeStatusTests(unittest.TestCase):
    def test_missing_runtime_is_unverified_not_passed(self):
        with patch.object(evaluation.shutil, "which", return_value=None):
            report = evaluation.run_runtime("codex", ROOT, ["review"])
        self.assertEqual({r["status"] for r in report.values()}, {"unverified"})
        self.assertIn("canonical-equivalence", report)

    def test_runtime_error_is_unverified_not_passed(self):
        def fail(*_args, **_kwargs):
            raise evaluation.Unverified("usage limit")
        with patch.object(evaluation.shutil, "which", return_value="/bin/true"), \
                patch.dict(evaluation.ASK, {"codex": fail}):
            report = evaluation.run_runtime("codex", ROOT, ["review"])
        self.assertEqual(report["review"], {"status": "unverified", "reason": "usage limit"})
        self.assertEqual(report["canonical-equivalence"]["status"], "unverified")
        self.assertEqual(report["native-discovery"]["status"], "unverified")

    def test_unrequested_runtimes_are_reported_unverified(self):
        with patch("builtins.print") as printed:
            self.assertEqual(evaluation.main([]), 0)
        self.assertTrue(any("SKIP" in str(call) for call in printed.call_args_list))

    def test_required_runtimes_fail_when_unverified(self):
        with patch.object(evaluation.shutil, "which", return_value=None), patch("builtins.print"):
            self.assertEqual(evaluation.main(["--runtime", "codex", "--scenario", "review"]), 0)
            self.assertEqual(evaluation.main(["--runtime", "codex", "--scenario", "review",
                                              "--require-runtimes"]), 1)

    def test_tree_mutation_and_wrong_contract_fail(self):
        states = iter(["", " M AGENTS.md\n"])
        with patch.object(evaluation.shutil, "which", return_value="/bin/true"), \
                patch.object(evaluation, "worktree_state", lambda _root: next(states)), \
                patch.dict(evaluation.ASK, {"codex": lambda *a, **k: "I approved it."}):
            report = evaluation.run_runtime("codex", ROOT, ["review"])
        self.assertEqual(report["review"]["status"], "fail")
        self.assertIn("the working tree changed", report["review"]["violations"])
        self.assertIn("answer is not the JSON contract", report["review"]["violations"])

    def test_equivalence_requires_the_canonical_nonce(self):
        seen = {}

        def answer(prompt, cwd, **_kwargs):
            if prompt.startswith("$axiom-release"):
                text = (Path(cwd) / ".claude/skills/axiom-release/SKILL.md").read_text()
                seen["adapter_is_link"] = (Path(cwd) / ".claude/skills/axiom-release").is_symlink()
                return text.split("Evaluation nonce: ", 1)[1].split()[0]
            return '{"skills": ["axiom-review"]}'
        with patch.object(evaluation.shutil, "which", return_value="/bin/true"), \
                patch.dict(evaluation.ASK, {"codex": answer}):
            report = evaluation.run_runtime("codex", ROOT, [])
        self.assertTrue(seen["adapter_is_link"])
        self.assertEqual(report["canonical-equivalence"]["status"], "pass")
        with patch.object(evaluation.shutil, "which", return_value="/bin/true"), \
                patch.dict(evaluation.ASK, {"codex": lambda *a, **k: "nonce-000000000000"}):
            report = evaluation.run_runtime("codex", ROOT, [])
        self.assertEqual(report["canonical-equivalence"]["status"], "fail")


class StructuralTests(unittest.TestCase):
    def test_repository_structure_passes(self):
        failures, evidence = evaluation.structural(ROOT)
        self.assertEqual(failures, [])
        release = evidence["skills"]["axiom-release"]
        self.assertEqual(release["resolved"], ".agents/skills/axiom-release/SKILL.md")

    def test_links_that_only_resolve_physically_fail(self):
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch) / "repo"
            shutil.copytree(ROOT / ".agents", root / ".agents", symlinks=True)
            shutil.copytree(ROOT / ".claude", root / ".claude", symlinks=True)
            shutil.copy(ROOT / "AGENTS.md", root / "AGENTS.md")
            shutil.copy(ROOT / "CLAUDE.md", root / "CLAUDE.md")
            (root / "scripts").mkdir()
            shutil.copy(ROOT / "scripts/check-claude-bootstrap.sh", root / "scripts")
            # Repository-root targets (CONTRIBUTING.md, docs/) are absent from this fixture.
            baseline = set(evaluation.structural(root)[0])
            self.assertFalse(any("axiom-review" in f for f in baseline), baseline)
            skill = root / ".agents/skills/axiom-review/SKILL.md"
            # Physically ../../policies reaches .agents/policies from both entries;
            # lexically, from .claude/skills/axiom-review, it reaches .claude/policies.
            self.assertTrue((root / ".agents/policies/security.md").is_file())
            skill.write_text(skill.read_text() + "\nSee [policy](../../policies/security.md).\n")
            added = set(evaluation.structural(root)[0]) - baseline
            self.assertEqual(added, {"axiom-review: link ../../policies/security.md does not resolve "
                                     "to one file from both entry paths"})


if __name__ == "__main__":
    unittest.main()
