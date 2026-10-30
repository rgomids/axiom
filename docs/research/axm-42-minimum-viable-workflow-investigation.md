# AXM-42 — Investigation: minimum viable Axiom workflow

Status: **Research / Proposed** (not product acceptance).  
**Research snapshot HEAD** (code inspection baseline for matrix A–L): `440e44a68638e08e901fd3f87f5b15cd395a6365`.  
**Specify PR head** (artifact first appeared / analytical review): `74b668460ef8f53de7323fce54f2f0adc61d646d` — [PR #366](https://github.com/rgomids/axiom/pull/366). File paths under `docs/research/` did **not** exist at `440e44a…` (blob 404); they exist from the Specify commit onward. Matrix evidence lines refer to the **snapshot** tree unless noted.  
Installed CLI observed at snapshot time: Axiom `0.22.3` (`revision: 6ccfbfc10a89`) — provenance only; not treated as proof of snapshot or PR-head binary.  
Authority: GitHub source at inspected SHA = technical; Linear AXM-42 Intake = product/operational baseline ([accepted baseline comment](https://linear.app/rgomids/issue/AXM-42#comment-274fd7e0)).  
Scope: read-only Specify (+ doc correction). No implementation, accepted-ADR rewrite, merge or live inference.

Related: [AXM-42](https://linear.app/rgomids/issue/AXM-42), [AXM-12](https://linear.app/rgomids/issue/AXM-12), [AXM-29](https://linear.app/rgomids/issue/AXM-29).  
Proposed contract: [Spec 015 (Proposed)](../specifications/015-minimum-viable-operational-workflow/spec.md).  
Prior Specify completion event (published): [Linear comment 4ee399c9](https://linear.app/rgomids/issue/AXM-42#comment-4ee399c9) — do not overwrite.

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

### Proposed minimum (Spec 015 — Intake baseline locked; Proposed until Spec acceptance)

1. Configure Project + 2 repo bindings + Codex/Claude Profiles once (defaults or user selection; no extraneous ceremony).
2. Accept the **minimal default Workflow** (Intake-settled); SDD remains an optional template.
3. Link or create Work Item (Provider for create; allow already-linked WI).
4. From the linked Work Item, start the default Workflow in **one natural user action**; Runtime from Project defaults/permissions.
5. **First slice:** single-agent dispatch **without** operator-authored Plan, manually published Validator Policy/digest, Project-default policy object, or per-stage manual graph preview — Axiom derives plan/graph **internally**; vendor auth/compat failures surface on real execution.
6. Auto-advance eligible stages (Intake-settled); stop only for humanGates / missing Axiom-owned input / explicit external/sensitive auth.
7. Multi-agent / dual-Runtime: structured inbox exchange under dispatch (slice 2); graph may stay INTERNAL.
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

### 4.1 Intake-settled (not open product options)

Do **not** reopen as mandatory human gates: minimal default Workflow; auto-advance of eligible stages; no manual Plan / policy digest / public policy ceremony on the happy path; no external certification. Encoding is Orchestrator-delegated within baseline.

### 4.2 Accepted-artifact tension

| Artifact | Conflict with MVP slim | Proposed handling |
|----------|------------------------|-------------------|
| Spec 007 | Stage plan + validator policy as operator-facing gates; SDD-shaped journey language | **Proposed supersession of operator ceremony only** after explicit Spec 015 / amendment acceptance; preserve graph safety, ceilings, humanGates, delivery auth. 015 must not silently rewrite 007. |
| ADR-0032 | Aligned (no certification) | Keep |
| ADR-0009/0026/0027/0028 | Keep internals | Preserve; simplify UX façade; migration/effect boundaries = open (4.3) |
| ADR-0020/0003/0005 | Keep | Keep |
| ADR-0025 fragments already limited by 0032 | — | No further silent supersession |
| Persisted Executions | format/bindings | Historical remain inspect/cancel/reconcile; new starts may use slim path; no silent ledger rewrite |

### 4.3 Truly open (implementation / compatibility)

1. Schema migration / own-effect boundaries when slim path coexists with historical Executions.  
2. Optional local/stub WI identity without Provider create — only if it changes Provider product contracts.  
3. Exact Spec 007 (and any ADR) amendment text — human accepts package before historical rewrite.

**What human acceptance of Spec 015 would decide:** authorize Plan/Implement of the slim path + precise amendment package — **not** re-litigate Intake product direction.

---

## 5. Incremental vertical slices (Proposed)

Normative product AC live in [Spec 015 § Acceptance criteria B](../specifications/015-minimum-viable-operational-workflow/spec.md) (P-AC1–P-AC6). Summary:

| Slice | Observable AC | Deps | Regression intent |
|-------|---------------|------|-------------------|
| **S1** | Single-agent dispatch with logs/status **without** manual Plan / policy digest / ceremonial transition (P-AC1, P-AC2, P-AC5) | Intake baseline (settled) | Journey tests updated; forbid secret leakage |
| **S2** | Two Runtimes structured Q/A; unblock; auto-advance without “continue” (P-AC3) | S1 + dual Runtime available | coordination_ingest + blackbox; live R2 **NOT_RUN** until authorized |
| **S3** | Two-repo Project outcomes + optional authorized PR (P-AC4–P-AC6) | S1–S2 + delivery auth | delivery packet/decision unchanged boundaries; live R3 **NOT_RUN** until authorized |

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

`re_review_PR_366` — analytical re-review of Spec 015 correction (F1–F4) on the same PR. Spec acceptance / ADR amendment / Plan / Implement / live R2–R3 remain separately authorized. No Implement of gate removals under this Specify correction authorization.
