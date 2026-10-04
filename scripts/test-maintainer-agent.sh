#!/usr/bin/env bash
# Cross-runtime maintainer-agent evaluation (Issue #174, ADR-0014).
#
#   scripts/test-maintainer-agent.sh                    # deterministic checks only
#   scripts/test-maintainer-agent.sh --runtime claude --runtime codex [--require-runtimes]
#
# Without --runtime, runtime behavioral scenarios are reported as SKIP
# (UNVERIFIED). A requested runtime that is missing or cannot answer is
# UNVERIFIED, never PASS; --require-runtimes turns UNVERIFIED into failure.
# Runtimes run read-only: no files change, nothing is published.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

python3 "$ROOT/scripts/test_maintainer_agent_eval.py" >/dev/null 2>&1 \
  || { python3 "$ROOT/scripts/test_maintainer_agent_eval.py"; exit 1; }
printf 'PASS: maintainer evaluation invariants and status handling are valid\n'

python3 "$ROOT/scripts/maintainer-agent-eval.py" --root "$ROOT" "$@"
