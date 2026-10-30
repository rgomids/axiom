# AXM-42 — Investigation: minimum viable Axiom workflow

Status: **Research / Proposed** (not product acceptance).  
HEAD inspected: `440e44a68638e08e901fd3f87f5b15cd395a6365` (matches prompt baseline).  
Installed CLI observed: Axiom `0.22.3` (`revision: 6ccfbfc10a89`) — provenance only; not treated as proof of HEAD binary.  
Authority: GitHub source at HEAD = technical; Linear AXM-42 Intake = product/operational baseline.  
Scope: read-only Specify. No implementation, ADR rewrite, PR, merge or live inference.

Related: [AXM-42](https://linear.app/rgomids/issue/AXM-42), [AXM-12](https://linear.app/rgomids/issue/AXM-12), [AXM-29](https://linear.app/rgomids/issue/AXM-29).  
Proposed contract: [Spec 015 (Proposed)](../specifications/015-minimum-viable-operational-workflow/spec.md).

---

## 1. Executive finding

### Root causes that block a first real dispatch (confirmed at HEAD)

| Sev | Cause | Evidence | Owner |
|-----|-------|----------|-------|
| **S0** | `workflow stage run` **requires** `--validator-policy <digest>`, but there is **no public CLI** to author/publish a Validator Policy. Tests seed via `ValidatorPolicyService.Apply` only. | `internal/cli/workflow_stage.go:238`; `cmd/lingo/stage_dispatch.go:57-58` → `ErrValidatorPolicy`; seed only in `cmd/lingo/stage_journey_blackbox_test.go:257-269`; help lists flag as required. | Axiom product gap |
| **S0** | `workflow stage run` **requires** a manual `--plan` JSON (authority ceiling / agents) before preview/apply. | `internal/cli/workflow_stage.go:238`; `docs/commands.md:1868-1918`; compile ceiling `internal/workflowcompiler/compiler.go` (graph/single). | Axiom UX / authority model |
| **S1** | Configured Executions do **not** auto-advance between stages; each stage needs plan→run→(integrate)→`advance --stage-result`, plus human `fact` gates on SDD stages. | `internal/workflow/gates.go:48-54`; default SDD r2 = **10 stages**, 4 `humanGates` (`internal/workflowdefinition/default-sdd-r2.json`). | Product ritual vs orchestration |
| **S1** | Runtime selection still uses **declarative** allowlists (runtime/profile/role/complexity/capabilities) and v4 frozen Profile admission — distinct from removed certification (ADR-0032). Misaligned Profile → `no_allowed_match` / admission errors. | `internal/runtimeapplication/policy_v4.go:79-176`; `internal/stageexecution/admission.go:56-132`; ADR-0032 accepted. | Axiom-owned selection |
| **S2** | Preview/apply + `--authorize-local` (+ `--authorize-external` for vendor spawn) on every mutating stage step. Correct for external consent; high ceremony for local-only ops. | `docs/commands.md:1912-1922`; `internal/stageexecution/run.go:71-85`. | Safety protocol |
| **S2** | `workflow start` requires a linked Work Item / Provider identity before stage work. | `docs/commands.md:1967-1985`; WI input in SDD intake. | Product coupling |

### Confirmed non-causes (do not blame these for first-dispatch failure)

- **CI/release/native-suite** do not gate local `stage run` (J).
- **Capability certification / CapabilityProven** removed from admission per ADR-0032; `Admit` is proof-free (`admission.go:56-57`).
- Literal string `reasoning-effort-low` as a **required Profile capability**: **not present** in HEAD tree (grep empty). Compiler may synthesize `reasoning-effort-*` tokens for adapters; Profile must not declare them (`violations.go`). Exploratory 29/10 claim → **reported/unverified** as current required capability gate.

### Unsupported / unverified (exploratory 29/10 govsm)

| Claim | Status at HEAD |
|-------|----------------|
| Repository binding mismatch forced rebind | Plausible (readiness `repository_binding_missing` / install binding rules) — **not reproduced** in this session |
| Valid Codex TOML → `configuration_invalid` | Parser/effort inspection can refuse unsafe/invalid TOML (`effort_v4`, auth preflight) — **not reproduced** here |
| Manual Plan required | **Confirmed** |
| Validator Policy digest required, no public publisher | **Confirmed** |
| No real dispatch occurred | Session did **not** invoke live Codex/Claude; journey tests seed policy artificially — live E2E **NOT_RUN** |

---

## 2. Current vs proposed user sequence

Scenario: Project with **two** Repositories, one Work Item, default Workflow, Codex + Claude messages, rework, optional PR.

### Current (configured path, HEAD)

1. `project configure` / bindings / Runtime Profiles (v4) + Project runtime policy — preview/`--authorize-local`.
2. `project workflow select` (builtin r1 embed or listed r2) — digest tuple.
3. `work-item create|select` (Provider).
4. `workflow start` ×2 with `--runtime-preview`.
5. Per stage (×10 on SDD r2): optional `fact` → **`stage plan --plan plan.json`** → **`stage run` preview** (`--plan`, `--validator-policy`, `--workspace-root`) → apply (`--preview-digest --authorize-local --authorize-external`) → multi-agent: `stage integrate` → `advance --stage-result`.
6. Cross-Runtime: children publish into attempt inbox under supervisor context; host `coordination publish` outside child context fails (`child_context_required`).
7. `delivery packet` → preview/apply (`--authorize-push`/`--authorize-pr`) → `delivery decision --authorize-human` → optional `delivery rework`.

### Proposed minimum (Spec 015 — Proposed until human MATERIAL decision)

1. Configure Project + 2 repo bindings + Codex/Claude Profiles once.
2. Select or accept a **minimal default Workflow** (≤3 agent stages or single-stage task path); SDD remains available as optional template.
3. Link or create Work Item (Provider for create; allow already-linked WI).
4. `workflow start` once; runtime choice from Project defaults.
5. **First slice:** single-agent `stage run` without operator-authored Plan JSON or operator-published Validator Policy digest — Axiom derives plan from Workflow stage binding and uses Project-default/empty validators until integrate needs commands.
6. Auto-advance eligible stages; stop only for humanGates / missing input / external auth.
7. Multi-agent / dual-Runtime: structured inbox exchange under dispatch (slice 2).
8. Multi-repo outcome + optional authorized PR (slice 3). Preserve: secrets out of manifests, no unapproved destructive/external effects, no fabricated evidence, attempt identity/logs, no blind retry of uncertain mutations.

---

## 3. Decision matrix (A–L)

| ID | Journey step | Current gate | Evidence | Why | Owner | Impact | Class | Simplest alternative | Risk if wrong | Conf. |
|----|--------------|--------------|----------|-----|-------|--------|-------|----------------------|---------------|-------|
| A | stage run / integrate | `--validator-policy` digest required; no public publish | `workflow_stage.go:238`; `stage_dispatch.go:57-58`; Apply only in tests | Separate reviewed test argv from agent/plan (PD-7) | Axiom | **Blocks first run** | **INTERNALIZE/AUTO** for run; PRESERVE digest at integrate when commands non-empty; add public author OR Project default empty policy | Unreviewed argv could run at integrate | HIGH |
| B | stage plan / run | Manual Plan JSON + authorityCeiling | `workflow_stage.go:238`; stage plan CLI; compiler ceiling | Explicit effect ceiling before spawn | Axiom | Blocks first run | **INTERNALIZE/AUTO** single-agent from Workflow binding; OPTIONAL explicit plan for multi-agent/advanced | Over-broad effects if ceiling wrong | HIGH |
| C | runtime select / admit | Allowlist runtime/profile/capabilities/complexity; v4 frozen; **no** certification | `policy_v4.go:79-176`; `admission.go:90-132`; ADR-0032 | Selection without vendor proof | Axiom | Onboarding friction | **PRESERVE_MINIMAL** allowlist+v4; OPTIONAL complexity; do not restore certification | Ambiguous/unsafe Profile match | HIGH |
| D | mutations | preview + digest + `--authorize-local`; external for spawn | `run.go:71-85`; commands.md | CAS + consent | Axiom / external | Ceremony | **PRESERVE_MINIMAL** external+costly; **INTERNALIZE** coalesce local preview/apply for stage integrate/cancel where safe | Stale apply / unpaid inference | HIGH |
| E | workflow shape / advance | Default SDD 10 stages; no auto-advance when configured | `default-sdd-r2.json` (10); `gates.go:48-54`; Builtin embed = r1 wire | SDD completeness | Product | Ritual cost | **OPTIONAL/ADVANCED** full SDD; ship **minimal ready Workflow** + auto-advance eligible | Skipping material human gates | HIGH |
| F | before start | Linked Work Item / Provider | commands.md start blockers | Traceability | Product+Provider | Blocks start | **PRESERVE_MINIMAL** link; do not require create-from-Axiom for local-only tasks if WI already exists | Orphan executions | MED |
| G | persistence | Graph3 + single_stage ledger + companions | ADR-0009/0026/0027 | Durable dispatch | Axiom | Complexity | **PRESERVE_MINIMAL** dual format; hide from happy path | Broken recovery | HIGH |
| H | retry / crash | Receipts, leases, epochs, reconcile, process group | ADR-0028; retry docs | Process safety | Axiom | Ops | **PRESERVE_MINIMAL**; no speculative guarantees beyond observed | Double mutation / orphan processes | HIGH |
| I | delivery | Packet → publication → decision/rework | commands.md delivery; #277 | Human accept ≠ technical complete | Axiom+Provider | Post-MVP path | **OPTIONAL** until PR desired; PRESERVE auth boundaries | Accidental push/PR | HIGH |
| J | CI/release | validate-repository / native-suite | `.github/workflows/*` | Repo hygiene / release | Maintainer | Does **not** block local dispatch | **PRESERVE_MINIMAL** | Weakening release gates | HIGH |
| K | config / identity | TOML/effort diagnostics; repo binding; project context | Spec 013; readiness; install bindings | Honest diagnostics | Axiom+vendor files | Friction | **PRESERVE_MINIMAL** diagnostics; improve messages (013) | Silent wrong repo/runtime | MED |
| L | first dispatch / cross-Runtime | Child-context coordination; per-Runtime auth | stage_publish / coordination_ingest; frozen_stage auth | Inbox integrity | Axiom | Multi-agent | **PRESERVE_MINIMAL** child inbox; ensure dual Runtime auth before graph | Cross-talk / forged publish | HIGH |

Counterarguments (do not remove as “overengineering” alone): A integrate commands without review; B unbounded effects; C silent Profile substitution; D replay after drift; H blind retry of uncertain Git/remote; I push without human; L cross-child inbox forgery.

---

## 4. Spec / ADR impact map (Proposed — do not rewrite historical ADRs)

| Artifact | Conflict with MVP slim | Proposed handling |
|----------|------------------------|-------------------|
| Spec 007 | Stage plan + validator policy as operator-facing gates; SDD-shaped defaults | Amend **after** MATERIAL approval; 015 proposes superseding *operator ceremony*, not graph safety |
| ADR-0032 | Aligned (no certification) | Keep |
| ADR-0009/0026/0027/0028 | Keep internals | Preserve; simplify UX façade |
| ADR-0020/0003/0005 | Keep | Keep |
| ADR-0025 fragments already limited by 0032 | — | No further silent supersession |
| Persisted Executions | format/bindings | Migration: historical remain inspect/cancel/reconcile; new starts use slim path; no silent rewrite of ledgers |

**MATERIAL human decisions required before Implement of removals:** (1) Validator Policy public author vs Project-default auto; (2) auto-derived Plan for single-agent; (3) minimal default Workflow vs keep SDD-10 as default; (4) auto-advance policy; (5) whether local-only work may start without Provider create.

---

## 5. Incremental vertical slices (Proposed)

| Slice | Observable AC | Deps | Regression intent |
|-------|---------------|------|-------------------|
| **S1** | Single-agent stage dispatch completes attempt with logs/status **without** manual Plan file or operator-published policy digest | MATERIAL decisions 1–2 | Existing journey tests updated; forbid secret leakage |
| **S2** | Two Runtimes exchange structured inbox records under one multi-agent stage | S1 + dual auth | coordination_ingest + blackbox |
| **S3** | Two-repo Project produces reviewed outcome + optional authorized PR | S1–S2 + delivery auth | delivery packet/decision unchanged boundaries |

---

## 6. E2E qualification (NOT synthetic substitute for R2/R3)

| Check | Result |
|-------|--------|
| Source inspection HEAD `440e44a…` | PASS |
| `go test ./internal/stageexecution/ -run TestValidatorPolicy\|TestAdmit` | PASS |
| `go test ./internal/cli/ -run TestWorkflowStage…` | SKIP (no matching tests) |
| Live Codex inference | **NOT_RUN** (out of Specify authority) |
| Live Claude inference | **NOT_RUN** |
| Dual-Runtime structured exchange on real repos | **NOT_RUN** |
| Exploratory 29/10 repro | **NOT_RUN** / treated as reported |
| Linear comment publish | **NOT_RUN** (MCP needsAuth; no API key in environment) |

---

## 7. Technical decisions (Orchestrator, reversible)

1. Artifacts placed under `docs/research/` + Proposed Spec 015 — research/proposed only.  
2. Delegation: three explore inventories (A/B/D, C/K, E–I) + one (J/L); Orchestrator spot-checked CLI help, admission, policy_v4, gates, validator CLI absence.  
3. Classification bias: prefer INTERNALIZE/AUTO over REMOVE for safety-bearing mechanisms.

---

## Next gate

`specification_review_and_material_architecture_decision` — human review of Spec 015 + matrix A/B/E especially; then authorized Plan. No Implement of gate removals under this Specify authorization.
