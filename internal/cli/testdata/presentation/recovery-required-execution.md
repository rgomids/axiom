### Partially completed — recovery required (`partial`)

Provider labels published; local projection not recorded

- **Next:** Run workflow reconcile again with the exact Execution revision
- **References:**
  - `execution:exec-7`
  - `github:acme/app#42`
- **Details:** `artifact:123e4567-e89b-42d3-a456-426614174000`

#### workflow

- **executionId:** `exec-7`
- **workflowVersion:** `1`
- **status:** `interrupted`
- **currentGate:** `implementation`
- **revision:** `4`
- **repositoryKey:** `app`
- **workItem:**
  - **projectId:** `123e4567-e89b-42d3-a456-426614174000`
  - **repositoryKey:** `app`
  - **provider:** `github`
  - **resource:** `acme/app`
  - **externalId:** `42`
  - **url:** `https://github.com/acme/app/issues/42`
  - **state:** `closed`
- **runtimeId:** `codex`
- **transitions:**
  - revision `4` · from `plan` · to `implementation` · outcome `advanced` · committedAt `2026-10-09T12:00:00Z`

**Provenance:** product `Axiom` · version `development` · revision `abc123def456` · sourceState `clean`
