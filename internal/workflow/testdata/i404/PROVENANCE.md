# Issue #404 — pre-change canonical format-4 corpus

Frozen before any codec change in IMPLEMENT-404-01 on 2026-11-24.
Generator source repository SHA: `15d0f87911b758636a922cd9ab9777ed5d05046f`.
Observed worktree HEAD: `045c4072d386469c64322564e4baa6b90b4ff22e`.
`git diff --exit-code 15d0f87911b758636a922cd9ab9777ed5d05046f HEAD -- internal/workflow/execution.go internal/workflow/execution_transitions.go`
returned 0 before generation. Generation nevertheless ran in a **git archive of
15d0f879**, never against the future amended encoder.

`freeze.go` is the complete literal generator. All IDs, times, invocation,
repositories and process identities are synthetic constants; no host/provider
state was read. Cases are single, independent DAG (a/b roots, join dependency),
terminal attempt, prepared, running, and multi-active historical a/b attempts.
Each `*.json` Execution is a complete encoder output, including the encoder's final newline.
Each historical `*.digest-input.json` is the exact ordered digest input emitted
by the unchanged types. `manifest.json` records SHA-256 of every literal file;
its hash is pinned in `TestI404FrozenFormat4` independently of the current writer.
Workflow digests equal `sha256:` plus their digest-input file hashes.

Reproduction (use an empty temporary archive, output to a separate temporary
directory; never overwrite this corpus):

```sh
archive=$(mktemp -d)
output=$(mktemp -d)
git archive 15d0f87911b758636a922cd9ab9777ed5d05046f | tar -x -C "$archive"
mkdir -p "$archive/internal/i404freeze"
cp internal/workflow/testdata/i404/freeze.go "$archive/internal/i404freeze/main.go"
(cd "$archive" && go run ./internal/i404freeze "$output")
```

The generator independently decodes/re-encodes every output and compares bytes
using the pinned old reader/writer. Its observed command exited 0. Each file was
then hashed using Python `hashlib.sha256` and validated by:

```sh
go test ./internal/workflow -run '^TestI404FrozenFormat4$' -count=1
```

Observed exit 0 at `045c4072` before production edits; full literal bytes,
independent ordered input hashes, decode/re-encode and clone digest all PASS.

Explicit-1 and explicit-2 digest inputs were pinned before T2 without invoking
any amended encoder: take literal `dag.digest-input.json`, append respectively
`,"concurrency":1` or `,"concurrency":2` immediately before the final Stage
object's closing brace, preserving all other bytes, and hash with hashlib.
Their hashes are respectively
`ed5a5f6a5b48e655c20c9c788df6cbee23db9ce9bf9404b42881982bc07db7c7` and
`f7d4042a2f4da83d5c55964bac2c3c8ccd4da224ec1a6ab86f8c754073e0af60`.
They are future digest oracles, not old-reader compatible Executions.

Released stable-v1 and i369 corpora are unchanged. This issue corpus is additive,
not a regeneration or relaxation of existing historical expectations. Never
regenerate expected values with the amended encoder.

## Pinned old-reader probe (T2)

`new_writer.go` produces explicit-1/2 full canonical Executions using the new
writer. It is distinct from `freeze.go` and never writes this frozen corpus.
Reuse the existing i369 old-reader spy probe, compiled **inside an archive of
15d0f879**, to verify refusal through the real old decoder/store/engine/reconcile
before resolver/launcher/collector/runner effects, without directory/lease/byte
changes:

```sh
probe=$(mktemp -d)
go run internal/workflow/testdata/i404/new_writer.go "$probe"
old=$(mktemp -d)
git archive 15d0f87911b758636a922cd9ab9777ed5d05046f | tar -x -C "$old"
mkdir -p "$old/internal/i404reader"
cp internal/workflow/testdata/i369/old_reader.go "$old/internal/i404reader/main.go"
(cd "$old" && go run ./internal/i404reader "new:$probe/explicit-1.json" "new:$probe/explicit-2.json")
```

Observed T2 command exit 0: `explicit-1.json PASS actual new writer rejected
before effects`; `explicit-2.json PASS actual new writer rejected before effects`.
`go test ./internal/workflow ./internal/strictjson -count=1` also exited 0
(workflow 1.584s, strictjson 0.283s) against the additive codec working diff.
