# Confined Spec-Kit Adapter Prototype

This prototype translates neutral Scenario 002 artifacts into a disposable,
pinned Spec-Kit workspace. It is experiment-only and is not an Axiom provider,
runtime dependency, or official state format.

## Boundary

```text
neutral artifacts
  -> exact-copy/path translation
  -> disposable Spec-Kit feature workspace
  -> official analyze then converge
  -> schema validation
  -> normalized neutral findings
```

The adapter exposes upstream failures and preserves raw events. It never copies
`.specify/` back into Axiom.

## Translation

| Neutral input | Disposable target | Treatment |
|---|---|---|
| `scenario/specification.md` | `specs/002-analyze-converge-reuse/spec.md` | exact content copy |
| `scenario/plan.md` | `specs/002-analyze-converge-reuse/plan.md` | exact content copy |
| `scenario/tasks.md` | `specs/002-analyze-converge-reuse/tasks.md` | exact initial copy; converge may append |
| `scenario/implementation/src/` | `src/` | exact content copy |
| `scenario/implementation/tests/` | `tests/` | exact content copy |
| `scenario/decisions.md` | `.experiment/decisions.md` | preserved, but official analyze/converge has no decision-artifact input |

Information invented only for tool execution: disposable feature directory,
feature state, generated Spec-Kit scaffold, and default unfilled constitution.
These are adapter metadata, not Axiom state. The principal translation loss is
the decision artifact; official capability outputs also use different severity
and category vocabularies, normalized by the experiment prompt.
