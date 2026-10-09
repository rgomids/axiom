# Issue #271 — Documentation validation

## Scope and provenance

Date: 2026-10-08 (America/Sao_Paulo). Source base:
`73dce6df0ab590b37df0e5a19b1483737bf1ca1e`; Epic audit base:
`c7260797aad7865419f555eea94a2eef7ae78285`. The delivery contains Specification
007, Proposed ADR-0020, JSON schema/default/custom/boundary examples, frozen
digests, an offline fixture validator and index/roadmap reconciliation. The
automation registry entry is validation metadata required by repository policy,
not product implementation or CI/CD redesign.

No product code, generated runtime skills, old specifications/ADRs, historical
Txx/Evidence, credentials or local user changes are modified. No live Runtime,
Provider delivery, merge, release or human acceptance was exercised.

## Card acceptance review

| #271 input criterion | Review result | Concrete evidence |
|---|---|---|
| Every Epic capability has proven-contract scope or uniquely owned follow-up | PASS (input coverage) | Specification WF-001–WF-014 and full Epic C/K1–K12 coverage matrix; existing proof limitations retained; #230/#133/#232 reused |
| Versioned default/custom examples, sequential and independent Codex/Claude, one-agent stage | PASS (document fixtures) | Three strict JSON definitions: default single-agent stages; custom independent implementer/reviewer, explicit dependent integrator; R1/R2 frozen digests |
| Specific schemas/DTOs/compatibility/projection/failures for concurrent consumers | PASS (contract review) | Specification portable/local boundaries, format-1 compatibility, format-2 admission, stage ledger, ten lifecycle values, operation/DTO/result tables and failure matrix; #274/#275 ownership separated |
| Open decisions have options/recommendation/trade-offs/owner/downstream gate | PASS (proposal completeness) | HD-001–HD-004 and Proposed ADR-0020; approval ledger explicitly pending; human approval must be recorded before implementation relies on choices |

These PASS labels validate the requested planning inputs. They do not approve
Specification 007/ADR-0020 or satisfy downstream implementation acceptance.

## Deterministic checks

Reproduce the dependency-free documentation checks from the repository root:

```text
python docs/specifications/007-configurable-workflows/validate_examples.py
python scripts/check-adr-governance.py .
python scripts/check-automation-registry.py .
git diff --check
```

- Fixture validator: PASS, three examples; 28 negative mutations including
  duplicate JSON keys, schema/type/limit/reference failures, protected gates,
  cycles, dangling dependencies and disconnected integration. Frozen hashes,
  R1/R2 binding fixtures and acyclic follow-up Issue DAG pass. This is not a
  simulation of the product's resume or authority checks.
- Full JSON Schema Draft 2020-12 meta-schema and all three examples: PASS using
  `jsonschema 4.26.0`; actual RFC 8785 canonicalization/SHA-256 agrees with every
  frozen digest using `rfc8785 0.1.4`. Installed only in an external temporary
  validation directory; no product/repository dependency added.
- ADR governance: PASS, 21 ADR files including Proposed ADR-0020; accepted
  history unchanged. Structure checks do not establish semantic acceptance.
- Automation registry: PASS, all 75 governed surfaces registered. Local Markdown
  references/anchors: PASS (118 checked at the review checkpoint).
- Whitespace and sensitive-file checks: PASS. Optional `gitleaks` was unavailable
  on the Windows PATH; no global scanner dependency was installed.

For the optional full schema/JCS check, make those two packages available in a
temporary Python environment, then validate `workflow.schema.json` with
`jsonschema.Draft202012Validator.check_schema`; call the same validator's
`validate` on each example. Compute `sha256(rfc8785.dumps(example)).hexdigest()`
and compare to `example-digests.json`. No external schema reference is fetched.

## Host limitations and repository validation

Windows 11 / PowerShell 7.6 / Python 3.12 / Git Bash:
`scripts/validate-repository.sh` stops on a baseline Claude skill symlink
materialized as a file. Git records that path as mode 120000 and this delivery
has no change under `.claude` or `CLAUDE.md`. The standalone ADR regression suite
has 55 passes and one environment error (`WinError 1314` creating a symlink);
no test assertion failed. The older user checkout also fails the bootstrap check
on its CLAUDE.md line-ending representation; it is not validation of this branch.
These failures are reported, not waived or patched into unrelated harness code.

Linux/WSL Ubuntu validation on a temporary copy with real symlinks: PASS. The copy
uses `git archive HEAD` with this delivery's changed files overlaid, preserving
tracked executable modes/symlinks. It is a validation workspace, not a supported
host installation or Runtime acceptance run.

`./scripts/validate-repository.sh .` passed: maintainer harness, 52 Claude bootstrap
cases, maintainer evaluation invariants, 31 automation-registry cases, all 75
registered surfaces, 29 label-policy tests, 56 ADR-governance tests, sensitive-file
tests and scans, and whitespace validation. The offline Specification 007 validator
also passed in that Linux copy. Runtime behavioral scenarios were explicitly
SKIPPED/UNVERIFIED by the harness because no live Runtime run was requested.
Product Go/browser tests are UNEXECUTED: the card changes no product code or UI
and expressly requires documentation/reference/schema validation only.

To reproduce the platform-neutral governance check with correct links, use a
Linux checkout of this branch and run the two commands above, or unpack a Git
archive with current changed documents into a temporary Linux directory,
initialize a temporary Git index and run the validators there. Windows' copied
symlink representation does not justify altering accepted harness contracts.

## Proportional review and remaining gates

Specification, architecture, verification and security review found no blocking
contract inconsistency after making dependency-output inputs explicit, preserving
legacy codecs, keeping one-agent execution possible and separating rejection from
terminal acceptance. All external references are primary sources, cited in the
Specification; no framework/dependency is adopted by reference. Human review of
the proposed trade-offs remains required. Accepted ADR-0003/0004/0008/0009 texts
and historical evidence are preserved.

The downstream ownership matrix is a bounded implementation contract proposal,
not additional Issue creation or status changes. Operational proof belongs to
#272/#276/#278 and exact publication/rework implementation to #277. Those checks
remain UNEXECUTED here by the card's documentation-only boundary. #15 and #271
human acceptance remain pending; no automated closure is implied.
