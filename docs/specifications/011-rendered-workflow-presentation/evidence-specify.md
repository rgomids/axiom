# AXM-38 — Specify Evidence, revision 1

Date: 2026-10-22. Scope: documentary analysis and local proposal only.
Base HEAD: `2e53228bfd27cff206c5c955f363bfff18fe745a` (detached, initially clean).
This is not renderer, installed-CLI, native-session or product acceptance Evidence.

## Source and repository observations

- Linear AXM-38 description/relations and all four comments read through the
  authenticated integration, `hasNextPage: false`. Gate UUID
  `1c7a61cd-01b9-4386-a57f-e6ba0711280f` says `READY_FOR_specify`; no blockedBy;
  AXM-39/AXM-12 downstream. No metadata/dependency changes made.
- AXM-32 read as Done operationally. Historical Spec 008 Proposed header is
  reconciled by its accepted-revision index; ADR-0030 has an Accepted header.
- GitHub `search_prs` for AXM-38 initially returned incidental mentions in unrelated
  PRs #335/#318. Exact `repo:rgomids/axiom "AXM-38" in:title,head` returned none;
  `search_branches` query `axm-38` returned an empty list/no cursor. No linked PR
  identified in card attachments/comments. No matching branch/PR observed; Linear
  `gitBranchName` is merely suggested metadata. No duplicate PR created.
- GitHub `fetch_file` on main and local `git hash-object` matched complete-file
  blob identities: Spec 007 `d39aac703d715b27ce116de23fd77c8ed2850e48`;
  ADR-0030 `25899eb789d311619ba661bc4f52c2619bfdc36d`. Remote responses returned
  first 16 lines plus complete-file blob SHA; matching hash establishes same file.
- CLI source confirms leading `--json`, required Project, exact show reference,
  default source `project`; report fields and producer expose both default revisions.
  Syntax verified by source, **not** CLI execution. No installed Project read/write.
- Python standard-library JSON inspection of four existing examples: default r1
  schema 1, default r2 schema 2, custom r1/r2 schema 1; each has 10 Stages. Custom
  implementation Stage is DAG/concurrency 2: implementer and reviewer have no
  predecessors; integrator depends on both. Inspection is not fixture validation.
- Current definition code selects schemas 1/2; both bound Stages/Agents to 32.
  Accepted ADR-0024 explains differing effort ownership. No schema file modified.
- Linear shared-notes lookup was attempted; server returned tool-not-found.
  Work Item, comments and repository sources remained available; no blocker inferred.

## Host documentation inventory

Documentation fetched via web search/open on 2026-10-22, never a real host session.
No screenshot or native diagram rendering has been observed. `UNVERIFIED` means
empirical support unknown, not proven unsupported.

| Ref | Official page opened | Bounded finding |
|---|---|---|
| H1 | [OpenAI app features](https://developers.openai.com/codex/app/features) → [current features](https://learn.chatgpt.com/docs/features) | Redirected documentation does not identify an exact Codex Desktop build or guarantee diagram syntax. |
| H2 | [OpenAI Visualizations](https://learn.chatgpt.com/docs/visualizations) | Describes diagrams for eligible ChatGPT surfaces; desktop preview availability varies. Explicitly excludes Codex CLI/IDE from Visualizations. No Mermaid/SVG format guarantee found. This does not qualify Codex Desktop for Axiom. |
| H3 | [OpenAI CLI](https://developers.openai.com/codex/cli/features) → [current CLI](https://learn.chatgpt.com/docs/codex/cli) | Documents terminal operation; no native Workflow rendering proof. |
| H4 | [Claude Code interactive mode](https://code.claude.com/docs/en/interactive-mode) | Terminal interaction and Markdown handling do not establish a rendered diagram format. |
| H5 | [Claude Code Desktop](https://code.claude.com/docs/en/desktop) | Distinct desktop surface. Mermaid/SVG guarantees not established by inspected page. |
| H6 | [Claude custom visuals](https://support.claude.com/en/articles/13979539-custom-visuals-in-chat-and-cowork) | Describes inline diagrams on chat/Cowork web/desktop. SVG/HTML download differs from display capability. Does not prove Code terminal rendering or Axiom integration. |
| H7 | [Claude artifacts](https://support.claude.com/en/articles/17153992-what-are-artifacts-and-how-do-i-use-them) | Describes separate artifacts and capability prerequisites. Availability does not certify a Workflow diagram in any particular session. |

Reproduce discovery with exact URLs above and searches:
`site:developers.openai.com/codex "Mermaid"`,
`site:code.claude.com mermaid terminal`,
`site:support.claude.com artifacts SVG diagrams`.
Inspect page bodies, redirects, supported-surface wording and format guarantees;
absence of a search match is not proof of impossibility. Official documentation
can change: recheck at later host qualification.

Future empirical protocol, **NOT_RUN / AUTHORIZATION_REQUIRED**: after an explicit
bounded external-session authorization, record exact supported host/build/settings,
canonical sequential/DAG event inputs and identity; render via its available
surface; capture actual visible output and manual witness; exercise Stage/Agent
detail and forced fallback. Retain viewport/zoom, failures, source digest and
renderer revision. No new plugin, service, upload, billing or support promise is
implicitly authorized. If required, propose those concrete effects first.

## Documentary validation

Validation results are recorded in `completion-receipt.json` after actual checks.
Checks concern whitespace, local links, requirement/acceptance coverage, JSON receipt
syntax and sensitive content. No durable checker, renderer test or runtime fixture
is introduced. No build, behavioral tests, CI or external smoke is claimed.

Manual coherence review: preserved ordered Stages, intra-Stage dependencies,
identity/selection separation, runtime eligibility versus resolved facts, effort
ownership, gate declarations versus authority and ADR-0030 transport/presentation
ownership. Proposed refinements remain labelled. No accepted file was rewritten.
