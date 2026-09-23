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

## Commit messages

Use this format for commits in the final review history:

```text
<type>(<optional-scope>): <concise description>
```

Accepted types are `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`, `ci`, `chore`, and `security`. The scope is optional; use it only when it improves understanding.

Examples:

```text
feat(web): add bootstrap landing page
fix(pages): correct deployment artifact path
docs(contributing): define commit message requirements
ci(pages): deploy static site to GitHub Pages
security(filesystem): reject unsafe destination ownership
```

Each commit must represent one coherent unit of change. Its subject must explain the observable purpose, not merely identify modified files. Generic messages such as `wip`, `update`, `changes`, `fix`, `misc`, `added stuff`, or equivalents are not acceptable in final history. Do not group unrelated changes in one commit.

Temporary `fixup!`, `squash!`, or WIP commits may exist during local development, but squash or reword them before requesting review. Add a body when the subject alone does not make the context or rationale clear; small, self-explanatory commits do not need one.

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

Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md). A PR without a description is not ready for review. Replace every template placeholder with real information or an objective explanation of why that item is not applicable. The description must provide:

- context and the problem being solved;
- scope and non-goals;
- related Issue, Specification, or ADR when applicable;
- validations executed, their results, and reproducible Evidence;
- documentation and security impact;
- known limitations;
- confirmation that unrelated changes are absent.

CI results do not replace Evidence or the context required in the PR description. Before requesting review, the author must inspect the PR title, description, and commit history.

### Ready for review

A PR is ready for review only when, at minimum:

1. its description is complete;
2. its scope is coherent;
3. relevant Issue, Specification, and ADR references are linked;
4. applicable validations and Evidence are recorded;
5. known limitations are declared;
6. temporary or WIP commits are cleaned up; and
7. the diff contains no unrelated changes.

A technical merge does not establish human acceptance. Acceptance does not automatically authorize the next Task. Contributions may be declined, split, or reformulated during review. Required explicit authorization must be recorded before work proceeds.

Accepted submissions are licensed under [Apache-2.0](LICENSE), unless explicitly stated otherwise. Contributors must hold the rights necessary to submit their content; identify any different licensing explicitly for review.

## AI-assisted contributions

AI-assisted contributions are allowed. The human contributor remains responsible for every submission and must verify code, licenses, sources, and claims. Prompts or raw model outputs do not replace Evidence. Do not submit private content or material whose licensing does not permit its inclusion.
