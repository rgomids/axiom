# Contributing to Axiom

## Project maturity

Axiom is in early development. The Codex harness and tested Go foundations are available. Lingo CLI, operational filesystem persistence, runtime adapters, and orchestration do not exist yet. See [project status](README.md#project-status) and the [Specifications index](docs/specifications/README.md) for authoritative scope and lifecycle references. The roadmap expresses direction, not implementation authorization.

## Before contributing

- Search [existing Issues](https://github.com/rgomids/axiom/issues) before opening a new one.
- Open or discuss an Issue before a significant change. Identify any affected Specification, ADR, or contract.
- Do not implement Tasks without explicit authority. An Issue or roadmap entry is not approval.
- Read [AGENTS.md](AGENTS.md), [documentation governance](docs/documentation.md), and the [Code of Conduct](CODE_OF_CONDUCT.md).
- Never include secrets, credentials, or private data in Issues, commits, PRs, or evidence. Sanitize reproductions and logs.
- Report vulnerabilities through the private process in [SECURITY.md](SECURITY.md), never in a public Issue. See [SUPPORT.md](SUPPORT.md) for other requests.

## Development workflow

Requirements and setup details live in [Getting Started](docs/development/getting-started.md). Git, Bash, standard POSIX utilities, and Go 1.26 or later are required for the checks below. The first Go run may download the dependency pinned in `go.mod`.

```bash
git clone https://github.com/rgomids/axiom.git
cd axiom
./scripts/validate-repository.sh .
go test ./...
```

1. Create a small, focused branch for the discussed scope.
2. Consult affected contracts; add or adjust tests appropriate to the change, using test-first development where useful.
3. Keep commits coherent. Do not mix unrelated refactors or changes.
4. Update only affected documentation and include verifiable Evidence: commands, outcomes, and known limits. Raw claims of completion are insufficient.
5. Run the relevant checks below, inspect the diff, and submit a PR for review. Address review feedback within the agreed scope.

## Validation

Run commands directly from the repository root. The [command reference](docs/commands.md) explains their scope and offline options.

```bash
./scripts/validate-repository.sh .
./scripts/validate-agent-package.sh .
./scripts/check-sensitive-files.sh .
go test ./...
go vet ./...
go build ./...
go mod verify
git diff --check
```

Before every commit, inspect staged paths and content and run `./scripts/check-sensitive-files.sh --staged .`. Follow [repository security](docs/security/repository-security.md), including the dedicated secret scanner when available. Report unavailable checks explicitly. Harness and documentation checks do not prove unimplemented product behavior.

## Pull requests

Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md) and provide:

- context and the problem being solved;
- scope and non-goals;
- related Issue, Specification, or ADR when applicable;
- validations executed and their results;
- documentation and security impact;
- known limitations;
- confirmation that unrelated changes are absent.

A technical merge does not establish human acceptance. Acceptance does not automatically authorize the next Task. Contributions may be declined, split, or reformulated during review. Required explicit authorization must be recorded before work proceeds.

Accepted submissions are licensed under [Apache-2.0](LICENSE), unless explicitly stated otherwise. Contributors must hold the rights necessary to submit their content; identify any different licensing explicitly for review.

## AI-assisted contributions

AI-assisted contributions are allowed. The human contributor remains responsible for every submission and must verify code, licenses, sources, and claims. Prompts or raw model outputs do not replace Evidence. Do not submit private content or material whose licensing does not permit its inclusion.
