# Tasks 013

**Proposed — revisão 3, 2026-10-26.** Decomposição do [Plan](plan.md).

| ID | Task | Requisitos | Estado |
|---|---|---|---|
| T01 | Reproduzir `candidate_invalid` sem capabilities em teste | RD-03, A1 | Done |
| T02 | `ProfileViolations` + teste de concordância com `Validate` | RD-08 | Done |
| T03 | Diagnósticos em `invalid_input`, `candidate_invalid` e recusas de `editCandidate`; `next` declarativo; `candidate_valid` não certifica | RD-01–07, RD-09 | Done |
| T04 | Jornada E2E `govsm` Codex+Claude e Project existente incompleto | A2–A8 | Done |
| T05 | Skill `axiom-project` “Complete guided setup”; `axiom-runtime` “Refusal diagnostics” | WZ-01–07 | Done |
| T06 | Registrar revisão v0.22.0 e testar upgrade Codex/Claude | WZ-08, A9 | Done |
| T07 | Documentação operacional e referência de comandos | RD-02, WZ-01 | Done |
| T09 | CR-356-01: jornada a partir de máquina vazia só com operações públicas; skill e docs reordenados | WZ-02, A10 | Done |
| T10 | CR-356-02: diagnóstico `--enabled` em Profile, unit + CLI JSON/humano | RD-10, A11 | Done |
| T11 | Model Advisor em `axiom-runtime`; oferta na etapa 4 de `axiom-project`; reconciliar linhas de recomendação | MA-R01–R09 | Done |
| T12 | Fixtures MA-01…MA-10, ancoragem e guarda contra catálogo estático | MA-R10, MA-01–10 | Done |
| T13 | Avaliação comportamental real dos cenários prioritários (MA-356-01) | MA-01, 02, 05, 08, 09, 10 | Claude: executado (PASS após correção, ver Evidence); Codex: BLOCKED (modelo fixado recusado pela conta) |
| T14 | Jornada nativa completa do wizard com o Advisor e pesquisa web real | MA-10, WZ-01–05 | NOT_RUN |
| T08 | Conversa nativa real Codex/Claude com o wizard | WZ-01–05 | NOT_RUN — exige ambiente/autorização próprios |
