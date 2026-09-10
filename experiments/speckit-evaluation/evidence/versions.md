# Evaluated Versions

## Axiom

- Repository: `rgomids/axiom`
- Base branch: `main`
- Base commit after `git pull --ff-only`:
  `173f05faef01702b63410aa4f39d7f6751bf29be`
- Experiment branch: `experiment/speckit-evaluation`
- Base date observed: 2026-08-10/11
- Workflow source: the base commit's `AGENTS.md`, `.agents/`, constitution,
  architecture, decisions, research, and specifications

## GitHub Spec-Kit

- Official repository: <https://github.com/github/spec-kit>
- Latest stable release verified before and after execution: `v0.16.2`
- Release published: 2026-08-10T19:46:01Z
- Annotated tag object: `10e42d94a392c69316ecfa8e36a5c26ac17f8d1a`
- Evaluated commit: `4871b485f97c7fa452ec58eba325d87536c55c34`
- Upstream `main` observed during the experiment:
  `9d15554c08ac5d01dc669dbd1a161a9638bc673b`
- CLI-reported version: `0.16.2`
- License at evaluated commit: MIT
- Release: <https://github.com/github/spec-kit/releases/tag/v0.16.2>
- Commit: <https://github.com/github/spec-kit/commit/4871b485f97c7fa452ec58eba325d87536c55c34>

The stable tag, not moving `main`, was executed. The one-shot install command
resolved the tag to the evaluated commit and did not install a persistent Axiom
dependency.

## Codex executor

- Executor for both approaches: Codex CLI
- Binary: `/Applications/ChatGPT.app/Contents/Resources/codex`
- CLI version: `codex-cli 0.147.0-alpha.6.5`
- Model explicitly fixed for both runs: `gpt-5.6-sol`
- Reasoning effort explicitly fixed for both runs: `high`

The `codex` command on `PATH` was unusable because its npm package referenced a
missing native binary. The ChatGPT application-bundled Codex binary was used for
both executions, so this environment difference did not vary by approach.
