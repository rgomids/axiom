# Reproduction Commands

Run from the Axiom repository root. These commands record what was executed;
they are not a request to adopt Spec-Kit.

## Repository preparation

```sh
git status -sb
git pull --ff-only
git switch -c experiment/speckit-evaluation
```

## Upstream resolution

```sh
gh api repos/github/spec-kit/releases/latest \
  --jq '{tag_name,published_at,html_url}'
git ls-remote https://github.com/github/spec-kit.git \
  refs/heads/main 'refs/tags/v0.16.2*'
```

The evaluated source was inspected from a shallow tag checkout inside the
experiment evidence workspace and removed after curation.

## Scenario integrity

```sh
cd experiments/speckit-evaluation/scenario
shasum -a 256 -c checksums.sha256
```

## Pinned one-shot Spec-Kit initialization

From a clean copy of `scenario/seed/`:

```sh
uvx --from git+https://github.com/github/spec-kit.git@v0.16.2 \
  specify init --here --force --integration codex --script sh \
  --ignore-agent-tools

uvx --from git+https://github.com/github/spec-kit.git@v0.16.2 \
  specify version
```

Observed initialization wall time: 0.63 seconds. Generated integration and
shared infrastructure were isolated to the Spec-Kit workspace.

## Controlled Codex execution

Both approaches used:

```sh
/Applications/ChatGPT.app/Contents/Resources/codex exec \
  --approve-for-me --json \
  -m gpt-5.6-sol \
  -c 'model_reasoning_effort="high"'
```

Exact phase prompts are retained in:

- `../axiom/prompts/phase-1.md`
- `../axiom/prompts/phase-2.md`
- `../speckit/prompts/phase-1.md`
- `../speckit/prompts/phase-2.md`

The same frozen operator answer was copied only after each approach had
recorded its clarification questions.

## Validation pointers

- Axiom exact validation commands and outcomes:
  `../axiom/workflow/validation.md`
- Spec-Kit commands and recorded outcomes:
  `../speckit/official/specs/001-command-deprecation-process/quickstart.md`
- Final repository validation before commit:

```sh
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh --staged .
git diff --check
git diff --cached --check
```
