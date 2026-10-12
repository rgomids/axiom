---
name: axiom-runtime
description: Inspect, configure and recover machine-local Axiom Runtime bindings and Model Profiles through Lingo.
---

# Axiom Runtime

Run `axiom skill inspect axiom-runtime` before collecting configuration inputs.
Present its `skill` payload and stop when inspection was requested. The binary
owns executable commands, argument forms and authority metadata. Use only
operations returned by that inspection; never maintain another registry.

Use canonical `runtime binding` and `runtime profile` commands for local
configuration. Read exact state before preparing changes. Missing state leads
to setup; format 1 requires explicit migration; unsafe/pending state requires
scoped recovery. Do not write configuration JSON, receipts or vendor settings.
Never collect credentials, login files, tokens, sessions or API keys.

## Guided setup

Passively discover installed supported executables. Installed, integrated,
authenticated, model available and effort supported are separate facts. Passive
inspection runs no vendor subprocess or installer. Ask which detected Runtime
the operator wants; detection never selects one. One explicitly chosen Profile
per included Runtime is sufficient. Both Runtimes may be configured separately.
Existing valid configuration must remain intact.

Collect exact operator Profile ID, Runtime, model intent, capabilities and
explicit effort mode/value. Effort belongs exclusively to the Model Profile.
`runtime-default` with value `default` requests no Axiom override; it makes no
claim about a vendor's concrete internal effort. Unknown model/effort facts can
be saved as intent, but operational use remains blocked pending exact proof.
Recommend options only with canonical provenance; never use this chat's model
or infer model access from authentication. Auth probes and inference need their
own consent and never authorize local apply.

Optional templates describe intent only:

- `fast`: short, bounded work with prompt feedback.
- `balanced`: routine engineering with proportionate verification.
- `deep`: difficult investigation with explicit verification needs.

Template names imply no Runtime, concrete model, effort token, ranking or
cross-vendor equivalence. Ask for each choice. Templates are optional; arbitrary
operator names and more than three Profiles per Runtime are supported.

## Exact confirmation

Prepare typed CLI flags from explicit choices. Execute without mutation authority
first. Present full effects, candidate digest, known dependency limits, opaque
snapshot and preview digest. Ask for confirmation of that exact candidate.
Apply only with returned `--expected-revision`, `--preview-digest` and
`--authorize-local`. Changed inputs or state require a new preview and new
confirmation. Successful replay is stale. Inspect read-back and report confirmed
or uncertain publication truthfully. Never substitute a choice or silently retry.

Removal changes local eligibility only. Final removal requires an explicit
compound cascade and publishes a tombstone; no vendor authentication, binary,
receipt, Project, Workflow or immutable Execution is deleted. Recovery uses only
the canonical scoped recovery command and its separate exact confirmation.
Project policy edits belong to axiom-project; Workflow definitions and migrations
belong to axiom-workflow; executing Work Items belongs to axiom-work-item.
Skill guidance and Evidence grant no authority.

## Canonical presentation

Invoke only `axiom` for Axiom domain operations. Preserve `status`, `result`,
`references`, `next`, `details` and `provenance` exactly.
Present that Markdown exactly as returned by the CLI;
never present an outcome as more
successful than its canonical status.
Run `axiom --json` only when the user
explicitly asks for machine-readable output.
Never derive, synthesize, or reinterpret completion fields.
Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`.
Operation-specific payloads: `runtimeAuthoring`.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `binding.list` | - | `axiom runtime binding list` | read-only | none | allowed |
| `binding.show` | - | `axiom runtime binding show` | read-only | none | allowed |
| `binding.create` | - | `axiom runtime binding create` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `binding.edit` | - | `axiom runtime binding edit` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `binding.enable` | - | `axiom runtime binding enable` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `binding.disable` | - | `axiom runtime binding disable` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `binding.remove` | - | `axiom runtime binding remove` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `binding.migrate` | - | `axiom runtime binding migrate` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `binding.validate-candidate` | - | `axiom runtime binding validate-candidate` | read-only | none | allowed |
| `binding.recover` | - | `axiom runtime binding recover` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `profile.list` | - | `axiom runtime profile list` | read-only | none | allowed |
| `profile.show` | - | `axiom runtime profile show` | read-only | none | allowed |
| `profile.create` | - | `axiom runtime profile create` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `profile.edit` | - | `axiom runtime profile edit` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `profile.enable` | - | `axiom runtime profile enable` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `profile.disable` | - | `axiom runtime profile disable` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `profile.remove` | - | `axiom runtime profile remove` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `profile.migrate` | - | `axiom runtime profile migrate` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |
| `profile.validate-candidate` | - | `axiom runtime profile validate-candidate` | read-only | none | allowed |
| `profile.recover` | - | `axiom runtime profile recover` | local mutation | preview first; exact snapshot, digest and explicit local confirmation (`--expected-revision`, `--preview-digest`, `--authorize-local`) | only for unambiguous intent |

Supported domain operations are `binding.list`, `binding.show`, `binding.create`, `binding.edit`, `binding.enable`, `binding.disable`, `binding.remove`, `binding.migrate`, `binding.validate-candidate`, `binding.recover`, `profile.list`, `profile.show`, `profile.create`, `profile.edit`, `profile.enable`, `profile.disable`, `profile.remove`, `profile.migrate`, `profile.validate-candidate`, and `profile.recover`.
