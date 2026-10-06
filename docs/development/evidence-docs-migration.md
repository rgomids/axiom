# Notion → GitHub documentation migration — publication

## Status and authority

Prepared on 2026-10-05 against repository `5ad602c2327b6a6380d7fc43effc3d44858667f1`
and Wiki `aaa1480239e3d5f6ee2b28256648a31fd318b6da`.
Both local branches: `agent/notion-github-docs-migration`.
At preparation time, neither change was published or committed. The user then
explicitly authorized Wiki publication and a repository PR on 2026-10-05.
Wiki publication/read-back is recorded below. Repository integration and human
acceptance remain separate gates. Original Notion remains untouched; unresolved
local drafts are temporary review artifacts, not an additional canonical source.

The requested target is Repository for technical contracts and code-coupled
references; Wiki for public product explanations, user navigation and historical
discovery clearly marked as such. Exact installation and CLI references stay
versioned; Wiki links rather than reproduces their contracts.

## Inventory and map

Scope: linked Axiom page tree under source ID
`3b4e01f22626810791b4f9d016ab5979`. Eighteen pages fetched and classified
before conversion; nine historical children plus their archive index.
The full original locations, last-edited values, classification, corresponding
artifacts, duplication and conflict/action fields are in the local review packet.

| Source title / ID | Class | Destination / action |
|---|---|---|
| 🧭 Axiom — Plataforma de Desenvolvimento Assistido por IA / `3b4e01f22626810791b4f9d016ab5979` | WIKI | Home — Replace stale S8/S9 status with navigation to canonical lifecycle; ownership Notion→Wiki. |
| 🔬 3. Discovery & Research / `3dee01f2262681b6b247ff55baa0b463` | REVIEW_REQUIRED | review/Discovery-and-Research.md — Split technical research into repository; preserve dated third-party claims pending validation. No adoption or prices asserted current. |
| 📸 Snapshot da página raiz — 13/09/2026 / `3dee01f2262681538f21d01575a1d6f5` | ARCHIVE | Archive-1 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| 🧭 Visão do Produto e Proposta Atual / `3b4e01f2262681b3a54bdf9c58a989a1` | ARCHIVE | Archive-2 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| 🧩 Domínios e Capacidades do Produto / `3b4e01f226268124a9e8e64f7d03ab2a` | ARCHIVE | Archive-3 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| ❓ Decisões Estratégicas em Aberto / `3b4e01f22626819eb2bcf4accaddd401` | ARCHIVE | Archive-4 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| 🧠 Contexto, Memória e Rastreabilidade / `3b4e01f2262681afa36bd5c6be30972b` | ARCHIVE | Archive-5 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| 🏗️ Arquitetura Conceitual Inicial / `3b4e01f2262681c9a313ef5b11e35f67` | ARCHIVE | Archive-6 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| 🎯 MVP e Estratégia de Evolução / `3b4e01f226268105b11edecd509d9a69` | ARCHIVE | Archive-7 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| 📦 Versão Anterior — Harness SDD Gerado por GPT / `3b4e01f226268121b3f5f1b9f845e8e5` | ARCHIVE | Archive-8 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| ✅ POC E2E — Aceite e transição para MVP — 20/09/2026 / `3e1e01f2262681f7b0ecf76f1c8db276` | ARCHIVE | Archive-9 — Historical under Archive; preserve dated content, never promote old choices to current architecture. |
| 🧭 1. Product Brief / `3dee01f2262681bc8165cf4b27d446fb` | WIKI | Product-Vision — Overlap foundation; product intent retained; direction and success criteria are goals, not acceptance. |
| 🔄 2. Product Flows / `3dee01f226268190ba4cfdc77e8cf5ac` | WIKI | Product-Flows — Detailed normative markers/output/authority duplicate Specification; user overview links to contract. |
| ❓ 4. Open Questions / `3dee01f22626814685e7ccbed40d4335` | REVIEW_REQUIRED | review/Open-Questions.md — Many runtime, authority, summary, provenance questions now specified. Reconcile question-by-question before current publication. |
| 🛠️ 5. Engineering / `3dee01f2262681d4bdbaf5a8669e54cd` | DUPLICATE | Technical-Documentation — Navigation already in README/docs indexes; convert to navigation only; ownership table superseded. |
| 📦 6. Archive / `3dee01f2262681b5982fc5ae464fd667` | ARCHIVE | Archive — Historical index; preserve original historical content with explicit dated warnings, no Notion mutation. |
| 🏷️ Política de Versionamento e Canais de Instalação — S9 / `3e9e01f2262681fe8389fb09b3c847b1` | REVIEW_REQUIRED | review/Versioning-and-Installation.md — CONFLICT: floating --channel rc removed; current candidates require --version. Release workflow/skill future tense stale. Link canonical technical docs, do not copy obsolete commands. |
| 🎨 Design System — direção, repository boundary e bootstrap / `3e7e01f226268132be8cfa46fe7b104a` | REVIEW_REQUIRED | review/Design-System.md — Separate repository canonical; technology proposals and 26/09 issue/project snapshots need reconciliation there; no cross-repository changes. |

Repository inventory covers all 245 tracked files under `docs/`:
239 retained technical/history/assets, four user-facing candidates and two mixed
documents requiring scoped extraction. Nothing removed or relocated.
Specifications (189 files), ADRs (20), Architecture (3), Development (6),
Research (4), Security (1), canonical assets (8), constitution, dogfooding and
landing-page Evidence retain repository authority. Installation and Commands
retain exact technical references. Product summaries link to public Wiki.

## Wiki preparation

Eight current/navigation pages: Home, Product-Vision, Product-Flows, Concepts,
Roadmap, Installation, Getting-Started, Technical-Documentation.
Historical area: Archive plus Archive-1 through Archive-9, each explicitly
non-current. Sidebar links current navigation and Archive. Eighteen content pages
and one sidebar, all content reachable from Home. No blank placeholder pages.
Four uncertain sources are converted separately for review; not included in
current Wiki content. All fetched originals and full conversions preserved locally.

## Conflicts and remaining review

- Source root reports S8/S9 reconciliation as pending; current GitHub artifacts
  govern lifecycle. Home links rather than reproduces the dated statement.
- Notion installation describes floating `--channel rc`; current installation
  reference requires exact `--version`. No obsolete command copied into current Wiki.
- Open Questions contains authority, Runtime, summary/provenance and metadata
  questions already addressed by versioned contracts. Needs item-level reconciliation.
- Discovery mixes technical research with dated third-party claims and product
  hypotheses. Requires separation and validation before current publication.
- Design System owns another repository; its proposals and operational snapshot
  require reconciliation there. No changes to that repository.
- Two remaining Notion URLs in research are historical provenance, not execution
  dependencies. Provider-boundary and security mentions describe generic providers;
  they are not operational dependencies on the old documentation workspace.
- Do not merge this repository proposal before matching Wiki pages are published
  and read back; local governance describes the proposed destination.

## Executed validation

- `./scripts/validate-repository.sh .`: PASS, including documentation routing,
  maintainer structure, ADR, automation registry, label-policy and security checks.
  Runtime behavioral scenarios explicitly SKIPPED/UNVERIFIED.
- `git diff --check` in Repository and Wiki: PASS.
- Local migration validator: PASS for internal links, Home reachability,
  20 repository target paths, balanced fences, full converted-source code-block
  preservation, absence of Notion block residue/URLs in Wiki and identical page bodies.
- Four Mermaid blocks retained; public rendering verified after publication below.
- No image/attachment blocks observed in returned source content:
  assets NOT_APPLICABLE within observed scope.
- Gitleaks directory scans of Wiki and review packet: PASS, no leaks.
- Product content/manual diff reviewed; no technical contract or accepted ADR rewritten.

## Unverified and retirement gates

Connector responses describe September snapshots and omit `truncated` and
`unknown_block_count` metadata. Full workspace search/access discovery controls
are unavailable in this session. Linked-tree coverage is verified; unlinked
pages, omitted blocks and October 5 source freshness are not.
Exhaustive page-by-page visual checks and external HTTP link health remain
unverified; sampled Markdown and all Mermaid render checks are recorded below.
Do not claim lossless workspace migration or retire Notion on this evidence.

Next: confirm complete Axiom source export/inventory and freshness, resolve four
review sources, review/merge repository PR separately, then verify
zero operational Notion dependency. Deletion/archive of Notion requires separate
explicit authority; migration itself does not grant it.

## Authorized Wiki publication — 2026-10-05

Published to the existing Wiki Git repository without rewriting its initial history:

- `2a6f2e7938a7b9f88106ab64b3b95cf6e20546c4` — historical discovery archive.
- `ef1525f6ce214b78fcea88047e050b595565d009` — product and user navigation.

`git ls-remote origin refs/heads/master` read back the second SHA;
`git fetch origin master` and `git diff --exit-code HEAD FETCH_HEAD` confirmed
all published Markdown matches the local candidate. Nineteen Markdown files:
18 content pages plus sidebar. Local migration validator passed again after
publication. No source page was deleted, moved, archived or edited in Notion.

Public [Wiki Home](https://github.com/rgomids/axiom/wiki) rendered with the
18-page count, current navigation and sidebar. Home → Product-Flows navigation
worked; its lifecycle Mermaid rendered with Intent through Reconcile nodes.
Archive-2 rendered its historical Mermaid; Archive-6 rendered both historical
diagrams. All four Mermaid blocks produced diagram output in the public browser.
This public browser check supplements static and Git read-back checks;
it does not establish exhaustive visual or external-link validation.

## CI metadata correction and English Wiki — 2026-10-05

The first PR description omitted the required `Related-Issues` and
`Completes-Issues` fields. The failed `delivery-metadata` run was
`37396375584`; its log reported those missing fields. The description was
reconciled with the repository PR template, using `none` for both fields:
no Issue is associated with this migration, and no Issue is completed.
`./scripts/delivery-issues.sh check-pr --title <title-file> --body <body-file>`
returned `metadata=declared`, `related=none`, `completes=none`.
The corrected remote metadata run `37396800181` passed on the original PR head.
No workflow, validator behavior, required check or protection was weakened.

The user also requested translation of the entire Wiki into English. All
18 content pages and `_Sidebar.md` were translated, including historical prose,
headings, list/table labels and Mermaid display labels. Historical facts and
warnings remain historical; this translation does not resolve the four source
review items or establish full Notion retirement.

- Translation commit: `c705142a44f8e6849ef39d1e944860ac0ad80e16`.
- Concurrent remote commit `3db4778ef0d000a492494ec98413d86433b6c75c` added
  an English `_Footer.md` logo. It was preserved unchanged.
- Published integrated Wiki head: `455b74cbea228d370edb52ba5d38fb757fb41186`.

Validation compared all 20 Markdown files with remote pre-translation state:
19 translated files and one unchanged English footer. Link destinations,
inline technical literals, executable fenced code, historical warnings/dates
and Mermaid identifiers/edges were preserved. Text-only diagram structure was
checked; diagram display labels were translated. No Portuguese prose was found
by the residual-language scan or manual review. All 18 content pages remain
reachable from Home, with 20 repository destinations validated. Staged
sensitive-file checks, Gitleaks and whitespace checks passed for the Wiki.

Publication used a normal push after incorporating the concurrent footer;
remote fetch and full-tree comparison confirmed the published candidate.
Public Home and sidebar rendered in English with the 18-page count.
