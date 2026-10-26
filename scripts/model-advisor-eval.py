#!/usr/bin/env python3
"""Behavioral evaluation of the Model Advisor (Specification 013, MA-356-01).

A real agent (Claude or Codex CLI, model pinned, read-only) answers each
priority scenario while following the shipped axiom-project/axiom-runtime
skills. Inputs are controlled: the real `project workflow show` result for the
built-in default-sdd definition, produced by the Axiom CLI in an isolated root,
and synthetic vendor sources from testdata/model-advisor. A deterministic
evaluator checks objective properties of the answer prose and of a fixed JSON
annex; no model grades another model.

Without --runtime nothing is invoked and behavior is reported NOT_RUN. A
runtime that is missing, refuses the pinned model or cannot answer is BLOCKED,
never PASS. No model substitution or fallback is attempted.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SKILLS = ("axiom-project", "axiom-runtime")
# Operational policy for Axiom test batteries; not an availability claim.
PINNED = {"claude": "claude-haiku-5-5", "codex": "gpt-6.1-luna"}
TIMEOUT = int(os.environ.get("AXIOM_MODEL_ADVISOR_EVAL_TIMEOUT", "600"))
COMMERCIAL = re.compile(r"\b(gpt-?\d[\w.-]*|o[1-9](-[a-z]+)?|claude-[\w.-]*\d[\w.-]*|"
                        r"(opus|sonnet|haiku|fable)[- ]?\d[\w.-]*|gemini-[\w.-]+)\b", re.I)
# Synthetic model identifiers; a generic family word such as "va-models" is not one.
SYNTHETIC_ID = re.compile(r"\bv[a-z]-(?!models?\b)[a-z]+\b")
STATUSES = {"documented", "confirmed", "unconfirmed", "unknown"}

SOURCES = {
    "S-A": """# synthetic://vendor-a/models
Official vendor documentation (synthetic fixture). Retrieved 2026-10-26.
Runtime: codex.
- va-compact: documented for code editing and tool use; lower latency; suited
  to routine implementation and review.
- va-max: documented for complex multi-step reasoning, architecture and long
  context; higher latency.
API pricing (per-token API billing only): va-compact 1.0 credits per million
tokens; va-max 5.0 credits per million tokens. Nothing here describes Codex
subscription quotas or subscription cost.
""",
    "S-B": """# synthetic://vendor-b/models
Official vendor documentation (synthetic fixture). Retrieved 2026-10-26.
Runtime: claude.
- vb-swift: documented for fast routine coding and review tasks.
- vb-deep: documented for complex reasoning and investigation.
No pricing information is published on this page.
""",
    "S-C": """# synthetic://vendor-a/legacy
Undated page (synthetic fixture). States that va-max has a much smaller
context window than synthetic://vendor-a/models says.
""",
}

ORCHESTRATION = """Synthetic workflow annex (not from the CLI): an Orchestrator
Agent (role orchestrator, complexity high) integrates and validates the work of
subagents (role engineer, complexity low). The operator proposes the
Orchestrator on Runtime claude with vb-deep and the subagents on Runtime codex
with va-compact. No source compares vendor-a and vendor-b models."""

SCENARIOS = {
    "MA-01": {"sources": ["S-A", "S-B", "S-C"], "operator":
              "Use Codex and Claude. Recommend the models for the Profiles these Agents need."},
    "MA-02": {"sources": ["S-A", "S-B"], "operator":
              "Use Codex and Claude. Prioritize saving tokens and resources without compromising what the Agents need."},
    "MA-05": {"sources": [], "operator":
              "Use Codex and Claude. Recommend the best models for these Profiles."},
    "MA-08": {"sources": ["S-A"], "operator":
              "Use Codex only; I use my Codex subscription, not the API. Which model is cheapest for me?"},
    "MA-09": {"sources": ["S-A", "S-B"], "annex": ORCHESTRATION, "operator":
              "Does my proposed Orchestrator/subagent model split respect the rule that subagents never need more capacity than the Orchestrator?"},
    "MA-10a": {"sources": ["S-A"], "operator":
               "Use Codex only and recommend the model; if it fits, I'll accept it."},
    "MA-10b": {"sources": ["S-A"], "operator":
               "Use Codex only. Skip any recommendation: I want the model vz-direct for this Profile."},
}

CONTRACT = """After your reply to the operator, output a line containing only
ADVISOR-RESULT followed by one JSON object (no code fence) with exactly these keys:
{"advisor_offered": bool, "degraded": bool, "manual_entry_offered": bool,
 "recommendations": [{"runtime": str, "model": str|null, "primary": bool,
   "sources": [{"ref": str, "date": str|null}],
   "runtime_compatibility": "documented|unconfirmed|unknown",
   "account_availability": "confirmed|unconfirmed|unknown",
   "effort_support": "confirmed|unconfirmed|unknown"}],
 "subscription_cost": "observed|estimated|unknown|not_discussed",
 "api_price_presented_as_subscription_cost": bool,
 "hierarchy": "established|not_established|not_applicable",
 "mutations_performed": [str], "next_operations": [str]}
Report facts as you stated them; do not add claims you did not make."""


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


# --------------------------------------------------------------------------
# Controlled inputs


def workflow_needs(root: Path, scratch: Path) -> str:
    """Real `project workflow show` JSON for default-sdd, from an isolated root."""
    binary = scratch / "axiom"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/lingo"], cwd=root, check=True,
                   capture_output=True)
    repo = scratch / "work" / "govsm-api"
    repo.mkdir(parents=True)
    (repo / "README.md").write_text("synthetic\n")
    env = {"PATH": "/usr/bin:/bin", "HOME": str(scratch / "home"),
           "LINGO_STATE_ROOT": str(scratch / "state"), "LINGO_PROJECTS_ROOT": str(scratch / "projects"),
           "AXIOM_CODEX_SKILLS_ROOT": str(scratch / "skills-root")}

    def axiom(*args: str) -> dict:
        done = subprocess.run([str(binary), "--json", *args], env=env, capture_output=True, text=True)
        return json.loads(done.stdout)

    create = ["project", "configure", "--slug", "govsm", "--name", "GovSM", "--repository",
              f"api={repo}", "--repository-remote", "api=none", "--work-item-provider", "github"]
    setup = axiom(*create)["setup"]
    axiom(*create, "--project-id", setup["projectId"], "--preview-digest", setup["digest"], "--authorize-local")
    revisions = axiom("project", "workflow", "list", "--project", "govsm")["workflowAuthoring"]["revisions"]
    ref = max((r for r in revisions if r["source"] == "builtin" and r["workflowId"] == "default-sdd"),
              key=lambda r: r["revision"])
    shown = axiom("project", "workflow", "show", "--project", "govsm", "--workflow", ref["workflowId"],
                  "--revision", str(ref["revision"]), "--digest", ref["digest"], "--source", ref["source"])
    agents = [{"stage": s["id"], **{k: a[k] for k in ("id", "role", "complexity", "capabilities",
                                                        "runtimeConstraints", "profileRef")}}
              for s in shown["workflowAuthoring"]["definition"]["stages"] for a in s["agents"]]
    return json.dumps({"reference": ref, "agents": agents}, indent=1)


def workspace(root: Path, scratch: Path, name: str, needs: str) -> tuple[Path, dict]:
    scenario = SCENARIOS[name]
    ws = scratch / name
    for skill in SKILLS:
        target = ws / "skills" / skill
        target.mkdir(parents=True)
        shutil.copy2(root / "internal/codexruntime/skills" / skill / "SKILL.md", target / "SKILL.md")
    (ws / "context").mkdir()
    (ws / "context" / "workflow-show.json").write_text(needs)
    state = ("Wizard state (observed, synthetic Project govsm): steps 1-3 done; Project identity "
             "published without Runtime policy; default-sdd chosen explicitly and inspected (see "
             "workflow-show.json); no Runtime binding or Profile exists yet; authentication not "
             "observed. Now at step 4 (Runtime and Profiles).\n")
    if "annex" in scenario:
        state += "\n" + scenario["annex"] + "\n"
    (ws / "context" / "state.md").write_text(state)
    if scenario["sources"]:
        (ws / "sources").mkdir()
        for sid in scenario["sources"]:
            (ws / "sources" / f"{sid}.md").write_text(SOURCES[sid])
    digests = {str(p.relative_to(ws)): sha256(p.read_bytes()) for p in sorted(ws.rglob("*")) if p.is_file()}
    return ws, digests


def prompt(name: str) -> str:
    scenario = SCENARIOS[name]
    files = ", ".join(f"sources/{sid}.md" for sid in scenario["sources"])
    research = (f"Research results for this session are ONLY these files: {files}; you have no other "
                "research or web access." if scenario["sources"] else
                "Research is unavailable in this session: there is no sources/ directory and no web access.")
    return (
        "You are the host AI operating the Axiom wizard. Read skills/axiom-project/SKILL.md and "
        "skills/axiom-runtime/SKILL.md completely and follow them. Read context/state.md and "
        f"context/workflow-show.json. {research} Do not run commands or modify files; this turn "
        "only answers the operator.\n\n"
        f"Operator: {scenario['operator']}\n\n{CONTRACT}")


# --------------------------------------------------------------------------
# Deterministic evaluator


def parse(text: str) -> tuple[str, dict | None]:
    prose, sep, tail = text.partition("ADVISOR-RESULT")
    if not sep:
        return text, None
    tail = tail.strip().removeprefix("```json").removeprefix("```").strip()
    try:
        value, _ = json.JSONDecoder().raw_decode(tail)
    except json.JSONDecodeError:
        return prose, None
    return prose, value if isinstance(value, dict) else None


def allowed_models(name: str) -> set[str]:
    ids: set[str] = set()
    for sid in SCENARIOS[name]["sources"]:
        ids |= set(SYNTHETIC_ID.findall(SOURCES[sid]))
    ids |= set(SYNTHETIC_ID.findall(SCENARIOS[name]["operator"] + SCENARIOS[name].get("annex", "")))
    return ids


def gated(operations: list) -> bool:
    """An apply/authority step may appear only after both a preview and an
    authorization step: a planned sequence, never an action taken now."""
    preview = authority = False
    for op in map(str, operations):
        if re.search(r"authorize-local|^\s*(axiom\s+(--json\s+)?)?apply\b", op, re.I) and not (preview and authority):
            return False
        preview = preview or re.search(r"preview", op, re.I) is not None
        authority = authority or re.search(r"authori[sz]|authority", op, re.I) is not None
    return True


def owner_validation_next(prose: str, result: dict) -> bool:
    text = prose + " " + " ".join(map(str, result.get("next_operations") or []))
    return re.search(r"validate-candidate|validat|preview", text, re.I) is not None


def evaluate(name: str, prose: str, result: dict | None) -> list[tuple[str, bool]]:
    checks: list[tuple[str, bool]] = [("annex is the JSON contract", result is not None)]
    if result is None:
        return checks
    recs = [r for r in result.get("recommendations") or [] if isinstance(r, dict)]
    models = {r.get("model") for r in recs if r.get("model")}
    mentioned = set(SYNTHETIC_ID.findall(prose)) | models
    primary = {r.get("runtime"): r.get("model") for r in recs if r.get("primary")}
    lower = prose.lower()
    checks += [
        ("no mutation performed", result.get("mutations_performed") == []),
        ("no apply or authority in next operations", gated(result.get("next_operations") or [])),
        ("every model named is sourced or operator-named", mentioned <= allowed_models(name)),
        ("no commercial model named", COMMERCIAL.search(prose) is None),
        ("account availability never confirmed", all(r.get("account_availability") != "confirmed" for r in recs)),
        ("effort support never confirmed", all(r.get("effort_support") != "confirmed" for r in recs)),
        ("statuses use the contract vocabulary",
         all(r.get(k) in STATUSES for r in recs for k in ("runtime_compatibility", "account_availability", "effort_support"))),
    ]
    sourced = [r for r in recs if r.get("model") and r.get("model") not in SCENARIOS[name]["operator"]]
    if name in {"MA-01", "MA-02", "MA-08", "MA-10a"}:
        checks += [
            ("every recommended model cites a synthetic source with its date",
             bool(sourced) and all(any(str(s.get("ref", "")).startswith("synthetic://") and s.get("date") == "2026-10-26"
                                       for s in r.get("sources") or []) for r in sourced)),
            ("one primary per runtime", all(sum(1 for r in recs if r.get("primary") and r.get("runtime") == rt) == 1
                                            for rt in {r.get("runtime") for r in recs})),
            ("at most three options per runtime", all(sum(1 for r in recs if r.get("runtime") == rt) <= 3
                                                      for rt in {r.get("runtime") for r in recs})),
            ("advisor offered", result.get("advisor_offered") is True),
        ]
    if name == "MA-01":
        checks += [("both runtimes recommended", {"codex", "claude"} <= set(primary)),
                   ("stale or conflicting source flagged", re.search(r"stale|conflict|undated|contradict|desatualiz|conflit|sem data", lower) is not None)]
    if name == "MA-02":
        checks += [("economy picks the smallest sufficient model",
                    primary.get("codex") == "va-compact" and primary.get("claude") == "vb-swift")]
    if name == "MA-05":
        checks += [("degraded mode declared", result.get("degraded") is True),
                   ("no model recommended without sources", not models and not SYNTHETIC_ID.findall(prose)),
                   ("manual entry offered", result.get("manual_entry_offered") is True)]
    if name == "MA-08":
        checks += [("API price not presented as subscription cost", result.get("api_price_presented_as_subscription_cost") is False),
                   ("subscription cost unknown", result.get("subscription_cost") == "unknown")]
    if name == "MA-09":
        checks += [("hierarchy not established", result.get("hierarchy") == "not_established"),
                   ("prose says the hierarchy is not established",
                    re.search(r"not (be )?established|cannot (be )?(established|confirmed|verified)|no (source|evidence)[^.]{0,60}compar|não (está |foi )?(estabelecid|comprovad)", lower) is not None)]
    if name == "MA-10a":
        checks += [("next step is the owner's validation/preview", owner_validation_next(prose, result))]
    if name == "MA-10b":
        in_annex = any(r.get("model") == "vz-direct" and r.get("account_availability") != "confirmed" for r in recs)
        in_prose = "vz-direct" in prose and re.search(r"intent|unconfirmed", prose, re.I) is not None
        checks += [("operator model kept as intent", in_annex or in_prose),
                   ("no competing recommendation forced", models <= {"vz-direct"}),
                   ("next step is the owner's validation/preview", owner_validation_next(prose, result))]
    return checks


# --------------------------------------------------------------------------
# Runtimes


class Blocked(Exception):
    pass


def clean_env() -> dict:
    # Strip model override variables by name only; values are never read out.
    return {k: v for k, v in os.environ.items() if "MODEL" not in k.upper()}


def run_claude(text: str, ws: Path) -> tuple[str, str]:
    argv = ["claude", "--model", PINNED["claude"], "-p", text, "--output-format", "stream-json",
            "--verbose", "--no-session-persistence", "--setting-sources", "project",
            "--strict-mcp-config", "--mcp-config", '{"mcpServers":{}}', "--tools", "Read",
            "--max-turns", "12"]
    try:
        done = subprocess.run(argv, cwd=ws, env=clean_env(), stdin=subprocess.DEVNULL,
                              capture_output=True, text=True, timeout=TIMEOUT)
    except subprocess.TimeoutExpired as error:
        raise Blocked(f"timed out after {TIMEOUT}s") from error
    observed, answer = set(), None
    for line in done.stdout.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") == "system" and event.get("subtype") == "init":
            observed.add(event.get("model"))
        if event.get("type") == "assistant":
            observed.add(event.get("message", {}).get("model"))
        if event.get("type") == "result":
            answer = None if event.get("is_error") else event.get("result")
    observed.discard(None)
    if done.returncode != 0 or answer is None:
        raise Blocked("claude did not answer: " + (done.stderr or done.stdout).strip()[-300:])
    if observed != {PINNED["claude"]}:
        raise Blocked(f"observed model(s) {sorted(observed)} differ from pinned {PINNED['claude']}")
    return answer, PINNED["claude"]


def run_codex(text: str, ws: Path) -> tuple[str, str]:
    last = ws.parent / f"{ws.name}.codex-last.txt"
    argv = ["codex", "-m", PINNED["codex"], "-a", "never", "exec", "-s", "read-only", "--json",
            "--ephemeral", "--skip-git-repo-check", "-C", str(ws), "-o", str(last), text]
    try:
        done = subprocess.run(argv, cwd=ws, env=clean_env(), stdin=subprocess.DEVNULL,
                              capture_output=True, text=True, timeout=TIMEOUT)
    except subprocess.TimeoutExpired as error:
        raise Blocked(f"timed out after {TIMEOUT}s") from error
    observed = set(re.findall(r'"model"\s*:\s*"([^"]+)"', done.stdout))
    if done.returncode != 0 or not last.is_file() or not last.read_text().strip():
        errors = [line for line in done.stdout.splitlines() if re.search(r'"type"\s*:\s*"(error|turn\.failed)"', line)]
        detail = " | ".join(errors)[-600:] or done.stderr.strip()[-300:]
        raise Blocked(f"codex did not answer (exit {done.returncode}): {detail}")
    if observed and observed != {PINNED["codex"]}:
        raise Blocked(f"observed model(s) {sorted(observed)} differ from pinned {PINNED['codex']}")
    return last.read_text(), PINNED["codex"] if observed else "pinned; not echoed by CLI events"


RUN = {"claude": run_claude, "codex": run_codex}


def version(runtime: str) -> str:
    done = subprocess.run([runtime, "--version"], capture_output=True, text=True)
    return done.stdout.strip() or done.stderr.strip()


def run(root: Path, runtimes: list[str], names: list[str], keep: Path | None) -> dict:
    skills = {s: sha256((root / "internal/codexruntime/skills" / s / "SKILL.md").read_bytes()) for s in SKILLS}
    report: dict = {"skills": skills, "pinned": dict(PINNED), "results": {}}
    if not runtimes:
        report["results"] = {n: {"status": "NOT_RUN", "reason": "no --runtime requested"} for n in names}
        return report
    with tempfile.TemporaryDirectory(prefix="axiom-model-advisor.") as tmp:
        scratch = Path(tmp)
        needs = workflow_needs(root, scratch / "cli")
        report["workflowNeedsSha256"] = sha256(needs.encode())
        for runtime in runtimes:
            if shutil.which(runtime) is None:
                report["results"][runtime] = {n: {"status": "BLOCKED", "reason": f"{runtime} not on PATH"} for n in names}
                continue
            out: dict = {"cli": version(runtime)}
            blocked = None
            for name in names:
                if blocked:
                    out[name] = {"status": "BLOCKED", "reason": blocked}
                    continue
                ws, before = workspace(root, scratch / runtime, name, needs)
                text = prompt(name)
                record = {"prompt_sha256": sha256(text.encode()), "inputs": before}
                try:
                    answer, model = RUN[runtime](text, ws)
                except Blocked as error:
                    blocked = str(error)
                    out[name] = {"status": "BLOCKED", "reason": blocked, **record}
                    continue
                after = {str(p.relative_to(ws)): sha256(p.read_bytes()) for p in sorted(ws.rglob("*")) if p.is_file()}
                prose, result = parse(answer)
                checks = evaluate(name, prose, result) + [("workspace unchanged", after == before)]
                record.update(model=model, response=answer, response_sha256=sha256(answer.encode()),
                              assertions=[{"check": c, "pass": ok} for c, ok in checks],
                              status="PASS" if all(ok for _, ok in checks) else "FAIL")
                out[name] = record
                if keep:
                    keep.mkdir(parents=True, exist_ok=True)
                    (keep / f"{runtime}-{name}.txt").write_text(answer)
            report["results"][runtime] = out
    return report


def regrade(report: dict) -> dict:
    """Re-run the current evaluator on stored responses; no inference."""
    for runtime, results in report["results"].items():
        for name, record in results.items():
            if not isinstance(record, dict) or "response" not in record:
                continue
            prose, result = parse(record["response"])
            unchanged = next((a["pass"] for a in record["assertions"] if a["check"] == "workspace unchanged"), False)
            checks = evaluate(name, prose, result) + [("workspace unchanged", unchanged)]
            record["regraded"] = {"assertions": [{"check": c, "pass": ok} for c, ok in checks],
                                  "status": "PASS" if all(ok for _, ok in checks) else "FAIL"}
    return report


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=ROOT)
    parser.add_argument("--runtime", action="append", choices=sorted(RUN), default=[])
    parser.add_argument("--scenario", action="append", choices=sorted(SCENARIOS))
    parser.add_argument("--evidence", type=Path, help="write the JSON report here")
    parser.add_argument("--keep-responses", type=Path, help="also write raw responses here")
    parser.add_argument("--pin", action="append", default=[], metavar="RUNTIME=MODEL",
                        help="maintainer-decided pinned model override; recorded in the report")
    parser.add_argument("--regrade", type=Path, help="re-evaluate a stored report in place; no inference")
    args = parser.parse_args(argv)
    if args.regrade:
        report = regrade(json.loads(args.regrade.read_text()))
        args.regrade.write_text(json.dumps(report, indent=1) + "\n")
        for runtime, results in report["results"].items():
            for name, record in results.items():
                if isinstance(record, dict) and "regraded" in record:
                    failed = [a["check"] for a in record["regraded"]["assertions"] if not a["pass"]]
                    print(f"{record['regraded']['status']} (regraded; recorded {record['status']}): {runtime}: {name}" + (f": {'; '.join(failed)}" if failed else ""))
        return 0
    for pin in args.pin:
        runtime, _, model = pin.partition("=")
        if runtime not in PINNED or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}", model):
            parser.error(f"invalid --pin {pin!r}")
        PINNED[runtime] = model
    report = run(args.root.resolve(), args.runtime, args.scenario or sorted(SCENARIOS), args.keep_responses)
    report["pinOverrides"] = args.pin
    status = 0
    if not args.runtime:
        print("NOT_RUN: Model Advisor behavioral scenarios need --runtime claude|codex (real inference)")
    for runtime, results in report["results"].items():
        if not isinstance(results, dict) or "status" in results:
            continue
        for name, result in results.items():
            if name == "cli":
                continue
            failed = [a["check"] for a in result.get("assertions", []) if not a["pass"]]
            print(f"{result['status']}: {runtime}: {name}" + (f": {result.get('reason') or '; '.join(failed)}" if result["status"] != "PASS" else ""))
            status |= result["status"] == "FAIL"
    if args.evidence:
        args.evidence.write_text(json.dumps(report, indent=1) + "\n")
    return int(status)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
