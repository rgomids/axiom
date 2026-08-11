# Commands Executed

Run from the Axiom repository root unless noted. Paths below preserve the actual
workflow while using variables instead of volatile temporary suffixes.

## Integrity and tests

```sh
cd experiments/speckit-evaluation/scenarios/002-analyze-converge-reuse
shasum -a 256 -c scenario/checksums.sha256
shasum -a 256 -c protocol/freeze.sha256
shasum -a 256 -c protocol/prototype-freeze.sha256
bash -n scripts/*.sh adapters/speckit/*.sh
scripts/test-tools.sh
adapters/speckit/test-failures.sh
```

## Current Axiom baseline and native prototype

Each run used a separate `mktemp -d` workspace containing the current Axiom
harness, frozen scenario, contract/schema, and only the applicable prompt and
capability. The oracle was not copied.

```sh
/Applications/ChatGPT.app/Contents/Resources/codex exec \
  -C "$WORKSPACE" --ignore-user-config --sandbox read-only --ephemeral --json \
  -m gpt-5.6-sol -c 'model_reasoning_effort="high"' \
  --output-schema protocol/findings.schema.json \
  -o result.json - < protocol/APPROACH-prompt.md > events.jsonl
```

## B adapter

```sh
adapters/speckit/prepare-workspace.sh \
  "$SCENARIO_ROOT" "$WORKSPACE" v0.16.2

(cd "$WORKSPACE" && \
  .specify/scripts/bash/check-prerequisites.sh \
    --json --require-tasks --include-tasks)

adapters/speckit/run-capability.sh \
  "$WORKSPACE" events.jsonl raw-result.json time.txt gpt-5.6-sol

adapters/speckit/normalize-result.sh \
  scripts/validate-findings.sh raw-result.json result.json
```

The preparation script executed the pinned one-shot command:

```sh
uvx --from git+https://github.com/github/spec-kit.git@v0.16.2 \
  specify init --here --force --integration codex --script sh \
  --ignore-agent-tools
```

## Version and upgrade evidence

```sh
gh api repos/github/spec-kit/releases/latest \
  --jq '{tag_name,published_at,html_url}'
git ls-remote https://github.com/github/spec-kit.git \
  'refs/tags/v0.16.2*'
uvx --from git+https://github.com/github/spec-kit.git@v0.16.2 \
  specify version
```

No persistent install, Axiom dependency, provider mutation, or production
workspace was used.
