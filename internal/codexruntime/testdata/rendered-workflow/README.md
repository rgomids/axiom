# AXM-38 offline presentation evaluation

All four `*.json` completion envelopes are **SYNTHETIC**, never captured CLI or
model responses. Source commit: `4bf046f79d7d5006fea25ff2b5dc4dafc8a48fdc`.
Definition bytes originate from Spec 007 `examples/{default-sdd-r1,default-sdd-r2,
custom-r1,custom-r2}.json`; existing Go `workflowdefinition.Decode` supplies JCS
identity. `manifest.json` pins envelope SHA-256 and original full reference.
`custom-*` is `reviewed-delivery`, never an observed `teste-sdd` installation.
Project UUID and Project revision are synthetic. Operation context is read-only
`axiom --json project workflow show`, category `inspected`, status `success`.
The actual completion has no operation field: command context identifies show.
CLI tests compare these wrappers with the real show serializer and native text.
No live installation or Project/Workflow/Execution store is accessed.

## Independently checked expected facts

Stage order: intake, specification, clarification, plan, tasks, implementation,
review, evidence, reconciliation, completion. Gate boundaries: planning_authority
before plan; implementation_authority before implementation; review_started before
review; human_acceptance after completion. All are declarations, never satisfied.
Default r1/r2: one owner per Stage, sequential, concurrency 1, no Agent edges.
Custom r1/r2: implementation DAG concurrency 2; implementer and reviewer are roots;
integrator depends on both. Exactly those two edges; other Stages have owner only.
Implementation eligibility: implementer codex; reviewer claude; integrator both.
`policy-default` is declared; schema 1 retains effort; schema 2 omits it.
The hand-checked oracle is in `workflow_presentation_test.go`, separately from
projection. Negative controls remove/add/reverse edges, alter identity/order,
gates/constraints and remove fallback facts. They must fail equality.

## Specimen provenance and limits

`*.fallback.md` are independently constructed, synthetic full-detail specimens,
with Workflow overview, Stage detail, scoped Agent detail and readable predecessors.
Their labelled JSON-pointer rows are a test-only comparison notation, not a product
wire format or mandated host output. Canonical leaf values, empty arrays and source
ordering remain recoverable. Go tests traverse the canonical envelope independently
and compare every row. This checks specimen fidelity, never model compliance.

`custom-dag.candidate.mmd` is an unrendered, synthetic relationship specimen only:
complete Stage progression plus one implementation DAG. Its restricted node/edge
syntax is checked against the hand oracle; full gate/configuration facts remain in
the fallback specimens. It neither adopts Mermaid nor qualifies a surface. There
is no executed renderer, screenshot, native host output or manual witness here.
Escape candidates are test-only; unsupported exact-host escaping must fall back to
inert text. `presentation_bounds_test.go` constructs valid sequential extra-edge,
single-Agent DAG, concurrency-1 DAG, dense 496-edge and boundary inputs in memory.

Host matrix: Codex Desktop, Codex CLI/IDE, Claude Code terminal/Desktop and all
other surfaces remain **UNVERIFIED** for AXM-38. Text is the safe conversational
route while capability is unknown. Terminal contract stays native. T03 requires
separate operational grant and real visible/manual qualification against final
skill bytes, including all three levels, accessibility and failed-render fallbacks.
