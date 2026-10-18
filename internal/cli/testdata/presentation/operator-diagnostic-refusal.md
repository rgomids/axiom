### Validation failed (`validation_failure`)

Project edit refused

- **Next:** Inspect canonical Projects

#### diagnostic

- **schemaVersion:** `1`
- **operation:** `project configure edit`
- **category:** `edit_project_not_found`
- **target:** _empty_
- **observedRevision:** `""`
- **observed:**
  - `Project selector was not resolved`
- **notObserved:**
  - `Exact target and revision`
- **hypotheses:** _none_
- **missingPrerequisites:**
  - `Canonical Project identity`
- **capability:** `unknown`
- **authority:** `not established`
- **confinement:** `unknown`
- **confirmedEffects:** _none_
- **uncertainEffects:** _none_
- **options:**
  - **[1]:**
    - **id:** `inspect`
    - **availability:** `available`
    - **prerequisites:** _none_
    - **effects:**
      - `read-only`
    - **requiredAuthority:** _none_
    - **operation:** `project list`
    - **arguments:** _empty_
    - **missingArguments:** _none_
    - **purpose:** `Resolve the missing Project`
    - **verification:** operation `project list` · expectedFact `Exact Project identity observed`
- **verification:** operation `project list` · expectedFact `Exact Project identity observed`

**Provenance:** product `Axiom` · version `development` · revision `abc123def456` · sourceState `clean`
