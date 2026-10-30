# Ready-to-post Linear comment (AXM-42)

`linear_published: false` — Linear MCP `needsAuth` in this environment; no `LINEAR_API_KEY`. Paste onto the same AXM-42 issue after authentication.

---

## AXM-42 Specify — investigation + Proposed Spec 015 (attempt-01)

**Outcome:** COMPLETED (research + Proposed Specification delivered locally).  
**HEAD:** `440e44a68638e08e901fd3f87f5b15cd395a6365`  
**Artifacts:**  
- https://github.com/rgomids/axiom/blob/440e44a68638e08e901fd3f87f5b15cd395a6365/docs/research/axm-42-minimum-viable-workflow-investigation.md *(paths exist in worktree; commit/PR not created — paste after authorized commit or attach worktree paths)*  
- `docs/research/axm-42-minimum-viable-workflow-investigation.md`  
- `docs/specifications/015-minimum-viable-operational-workflow/spec.md`

### Executive finding

First real dispatch is blocked primarily by **Axiom product gaps/ceremony**, not by ADR-0032 certification (already removed) and not by CI/release:

1. **S0** — `workflow stage run` requires `--validator-policy <digest>` (`internal/cli/workflow_stage.go:238`; load `cmd/lingo/stage_dispatch.go:57-58`) but **no public CLI** publishes policies (tests seed `ValidatorPolicyService.Apply` only).  
2. **S0** — Manual `--plan` JSON + authority ceiling required for run.  
3. **S1** — Ready path is SDD-shaped (r2 = 10 stages; configured Executions need manual `advance`; humanGates on plan/implementation/review/completion).  
4. **S1** — Declarative Runtime allowlists/v4 admission remain (selection, not certification).  
5. **S2** — Preview/apply + dual authority; WI/Provider before start.

Exploratory 29/10 claims: Plan + missing Validator publisher **confirmed**; `reasoning-effort-low` as required capability **unsupported** at HEAD (string absent); TOML/`configuration_invalid` and rebind **not reproduced** here. Live Codex/Claude E2E **NOT_RUN**.

### Recommendations (Proposed)

| ID | Class | Action |
|----|-------|--------|
| A | INTERNALIZE/AUTO | Project-default / empty policy for run; public author or keep digest only when integrate runs commands |
| B | INTERNALIZE/AUTO | Derive single-agent Plan from Workflow binding |
| C | PRESERVE_MINIMAL | Keep allowlists; do not restore certification |
| D | PRESERVE external; INTERNALIZE local coalesce where safe | Keep `--authorize-external` for spawn |
| E | OPTIONAL full SDD; ship minimal default + auto-advance | |
| F–L | See investigation matrix | Preserve process safety, delivery auth, child inbox |

**Next gate:** human MATERIAL review of Spec 015 (especially A/B/E) before any Implement of removals. AXM-12 remains blocked outbound until this lands as accepted product direction.

### Verification

| Check | Status |
|-------|--------|
| Source inspection @ 440e44a | PASS |
| `go test ./internal/stageexecution/ -run 'TestValidatorPolicy\|TestAdmit'` | PASS |
| Live Codex/Claude | NOT_RUN |
| Linear API publish from this session | NOT_RUN / REPORT_PENDING |

>>> Dados estruturados do evento — abrir JSON

AWF-EVENT:v1
<!-- axiom-workflow-event:v1 -->
```json
{
  "schema_version": 1,
  "event_id": "AXM-42:specify:attempt-01:stage.completed:440e44a68638e08e901fd3f87f5b15cd395a6365",
  "work_item": "AXM-42",
  "stage": "specify",
  "type": "stage.completed",
  "attempt": "attempt-01",
  "outcome": "completed",
  "publisher": "runtime-executor",
  "verification": "reported",
  "artifact_revision": "440e44a68638e08e901fd3f87f5b15cd395a6365",
  "technical_decisions": [
    {
      "id": "TD-01",
      "summary": "Deliver Proposed Spec 015 + research under docs/; no Implement of gate removals",
      "rationale": "Specify authorization is research-only; MATERIAL architecture/product changes need human decision"
    },
    {
      "id": "TD-02",
      "summary": "Classify A/B as INTERNALIZE/AUTO rather than REMOVE",
      "rationale": "Counterargument: unreviewed integrate argv and unbounded effects; preserve safety via defaults"
    }
  ],
  "deferred_findings": [
    {
      "id": "DF-01",
      "summary": "Builtin embed vs listed r2 default drift (r1 wire vs 10-stage r2)",
      "evidence": "internal/workflowdefinition/definition.go Builtin(); default-sdd-r2.json"
    }
  ],
  "evidence": [
    {
      "kind": "source_inspection",
      "status": "PASS",
      "url": "https://github.com/rgomids/axiom/tree/440e44a68638e08e901fd3f87f5b15cd395a6365"
    },
    {
      "kind": "unit_test",
      "status": "PASS",
      "detail": "go test ./internal/stageexecution/ -run TestValidatorPolicy|TestAdmit"
    },
    {
      "kind": "live_runtime_e2e",
      "status": "NOT_RUN"
    },
    {
      "kind": "linear_comment",
      "status": "NOT_RUN",
      "detail": "MCP needsAuth; ready-to-post in linear-comment-ready.md"
    }
  ],
  "next_gate": "specification_review_and_material_architecture_decision"
}
```
