Usando somente `speckit-constitution`, `speckit-specify`, `speckit-clarify`, nessa ordem. Primeiro: ler brief, critérios, instruções oficiais completas e estado; depois executar cada fronteira; parar antes de qualquer resposta inventada ou planejamento.
Contexto confirmado: mudança documental, comandos existentes intactos, cinco decisões humanas materiais. Nenhum arquivo de workflow Axiom será lido. Agora executo fronteira `speckit-constitution`; skill restringe escrita somente à constituição.
Constituição concluída: v1.0.0, sem placeholders ou hooks. Fronteira `speckit-specify` inicia agora com a citação inicial exata; brief completo serve somente como contexto fornecido.
Especificação criada em `specs/001-command-deprecation-process`: checklist 14/16, duas pendências deliberadas por decisões humanas. `speckit-clarify` agora verifica pré-requisitos uma única vez e para sem integrar respostas inexistentes.
`spelkit-clarify` encontrou cinco ambiguidades materiais: gatilho, ciclo/aprovação, localização/nome, campos obrigatórios, aviso/exceção. Skill pausa antes de escrever sem respostas; nenhuma resposta será presumida. Faço só revisão final de escopo.
Status:

- `speckit-constitution`: complete, v1.0.0 — [constitution.md](official/constitution.md)
- `speckit-specify`: complete — [spec.md](official/specs/001-command-deprecation-process/spec.md)
- Quality checklist: 14/16 — [requirements.md](official/specs/001-command-deprecation-process/checklists/requirements.md)
- `speckit-clarify`: paused at human decision point; 5 asked, 0 answered.
- No sample documentation, command behavior, plan, tasks, implementation, commit, provider, or network changed.

### Q1

**Question:** Which command changes must require a deprecation record? (FR-001)

Why it matters: This defines which proposals reviewers must block until a record exists.

**Recommended:** Option A — Covers user-impacting lifecycle changes while excluding harmless editorial corrections.

| Option | Description |
|--------|-------------|
| A | Any planned rename, removal, replacement, or incompatible behavior, output, or interface change; editorial corrections excluded. |
| B | Renames and removals only. |
| C | Removals only. |
| D | Every command-documentation change. |
| Short | Provide a different short answer (≤5 words). |

### Q2

**Question:** Which lifecycle stages and approval authority must govern a command deprecation? (FR-005)

Why it matters: Reviewers need explicit states and accountable human approval gates.

**Recommended:** Option A — Lightweight lifecycle with separate approval before public deprecation and final removal.

| Option | Description |
|--------|-------------|
| A | Proposed → Deprecated → Removed; one designated repository maintainer approves entry into Deprecated and Removed. |
| B | Proposed → Deprecated → Removed; two repository maintainers approve both transitions. |
| C | Proposed → Deprecated → Removed; the record owner approves all transitions. |
| D | Proposed → Removed; one repository maintainer approves removal, with no formal Deprecated stage. |
| Short | Provide a different short answer (≤5 words). |

### Q3

**Question:** Which repository path and filename convention must deprecation records use? (FR-003)

Why it matters: A deterministic convention makes records discoverable and prevents conflicting identities.

**Recommended:** Option A — Date plus command name provides stable history and avoids collisions.

| Option | Description |
|--------|-------------|
| A | `docs/deprecations/YYYY-MM-DD-<command-name>.md`, using proposal date and kebab-case command name. |
| B | `docs/deprecations/<command-name>.md`. |
| C | `docs/commands/deprecations/<command-name>.md`. |
| D | Store all records in one `docs/deprecations.md` file. |
| Short | Provide a different short answer (≤5 words). |

### Q4

**Question:** Which fields must every deprecation record require? (FR-004)

Why it matters: The chosen field set determines whether migration, approval, and rollback reviews are testable.

**Recommended:** Option A — Smallest set covering every safety and governance concern from the brief.

| Option | Description |
|--------|-------------|
| A | Command, status, rationale, owner, approver, lifecycle dates, notice, user impact, replacement or none, migration, rollback or explicit impossibility, and exceptions. |
| B | Command, reason, removal date, migration, rollback, and approver. |
| C | Option A plus alternatives, risk rating, affected versions, communication log, and sign-off history. |
| D | Only command and owner mandatory; reviewers select remaining fields per record. |
| Short | Provide a different short answer (≤5 words). |

### Q5

**Question:** What minimum notice and exception policy must apply before command removal? (FR-006)

Why it matters: A deterministic notice rule protects users while defining how urgent removals proceed.

**Recommended:** Option A — Thirty calendar days remains measurable without requiring a release cadence.

| Option | Description |
|--------|-------------|
| A | 30 calendar days; designated approval authority may shorten it for urgent risk with written rationale, migration, and rollback treatment. |
| B | One published release cycle; same documented exception requirements. |
| C | 90 calendar days; emergency exception requires two maintainers. |
| D | No fixed minimum; approval authority chooses notice per record. |
| Short | Provide a different short answer (≤5 words). |

Reply once, for example: `Q1: A, Q2: A, Q3: A, Q4: A, Q5: A`.
