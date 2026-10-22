# Concepts

Axiom is a local development control plane. Lingo is its CLI and domain
application surface. The names describe related parts of one product, but the
objects and capabilities below have distinct owners.

- **Project** is a logical product context. **Repository** is a version-control
  boundary that may be associated with a Project. A Project is not a Repository;
  current directory or Git metadata does not silently choose a Project.
- **Repository Role** describes how an associated Repository participates in a
  Project. It is not a model role or an execution identity.
- **Model Execution** is work performed through an explicitly resolved Runtime,
  Model Profile and execution authority. A configured model name is intent; it
  does not prove support, authentication, billing or successful inference.
- **Agent Provider** is an external capability source. A **Transport** is a
  mechanism for reaching a Provider, such as MCP, native API, CLI or HTTP.
  **Integration** is configuration connecting capabilities and context; it is
  not synonymous with MCP.
- **MCP skills** are instruction entrypoints for supported agent runtimes. They
  guide interaction with Axiom; they are not authoritative Project, Repository,
  configuration or execution state.

AI may propose intent. Axiom validates it, and an explicit authority step gates
applicable local or external effects. A preview describes proposed effects;
only the exact returned preview digest and required revision/authority values
authorize its corresponding apply. Any changed input or observed state requires
a fresh preview.

Continue to [Project](project.md) or the [command reference](cli.md).
