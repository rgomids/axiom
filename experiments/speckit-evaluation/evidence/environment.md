# Environment

- Host OS: macOS 26.5.2, build 25F84
- Architecture: arm64
- Time zone: America/Sao_Paulo
- Git: 2.53.0
- Python: 3.14.4
- uv/uvx: 0.11.6
- Codex CLI: 0.147.0-alpha.6.5
- Spec-Kit CLI: 0.16.2 through pinned one-shot `uvx`
- Source repository: public Axiom checkout
- Provider mutations during the experiment: none
- Persistent dependencies introduced: none

## Tool availability limits

- `gitleaks`: unavailable
- `lychee`: unavailable in the Spec-Kit execution
- `markdown-link-check`: unavailable in the Spec-Kit execution
- Existing Axiom sensitive-file scanner: available to the Axiom execution and
  later used against the final staged experiment

No global tool was installed to remove these differences. Both approaches used
document-specific deterministic fallbacks and reported scanner limits.

## Isolation

Each approach ran in its own Git-initialized copy of the same seed repository.
Operational workspaces and the shallow upstream clone lived only under
`experiments/speckit-evaluation/evidence/` during execution. They were removed
after curated public evidence was copied into the approach directories.

Removal from the repository workspace was recoverable: exact scratch targets
were moved to the local macOS Trash after an `rm` attempt was rejected before
execution.

Nested Codex loaded the same user-level memory environment in both runs. This
was not part of the frozen scenario and is a contamination risk: the Spec-Kit
run saw high-level Axiom repository memory even though it did not read or use
Axiom workflow files. Raw transcripts were not committed because they contained
environment-owned memory and noisy tool diagnostics rather than necessary
public evidence.
