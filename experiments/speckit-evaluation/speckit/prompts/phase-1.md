# Spec-Kit execution — phase 1

Use only the official Spec-Kit `v0.16.2` Codex skills installed in this
workspace. Do not consult or use Axiom workflow files.

Read `.experiment/brief.md` and `.experiment/completion-criteria.md`. Run these
official skills in order, preserving each official artifact and command
boundary:

1. `$speckit-constitution` with these scenario-derived principles:
   documentation-only scope; preserve existing command behavior; durable
   repository-local Markdown; provider-neutral offline use; explicit human
   approval for command lifecycle changes; deterministic validation where
   practical; no application code, executable validator, dependency, CI,
   provider, credential, or network requirement.
2. `$speckit-specify` using exactly the initial user input in
   `.experiment/brief.md`, with that file's problem, outcome, constraints, known
   context, unknowns, and non-goals as supplied context.
3. `$speckit-clarify` over all material unknowns before planning.

Clarification control:

- Operator answers are not available yet.
- Do not invent answers or use Axiom conventions.
- Ask the official clarification questions, preserve them in the final response,
  and stop at the human decision point.
- Do not run plan, checklist, tasks, analyze, implement, or converge.
- Do not modify sample documentation except official Spec-Kit infrastructure,
  constitution, feature-state, and specification artifacts required by these
  three commands.
- Do not commit, push, or access a provider.

Final response: concise command status plus exact questions awaiting one batched
human response.
