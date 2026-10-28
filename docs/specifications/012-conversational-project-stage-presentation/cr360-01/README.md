# CR-360-01 — Response context qualification

PR #360; reviewed source HEAD `48f4a0c8310ab9f93d48c6903711231e17018117`.
This remediation adds a test-only oracle and reusable capture-grading protocol.
It does not introduce a renderer, CLI operation or inference call. Historical
Specification 012 observations remain unchanged. AC5 native gates remain pending.

## Evidence categories

| Category | Result | Scope |
| --- | --- | --- |
| Static guidance | PASS | Four packaged skills refine conflict and inert-label rules; old shared revision stays owned |
| Deterministic presentation | PASS | 14 synthetic oracle cases, 171 negative mutations; not observed host behavior |
| Canonical JSON | PASS (oracle) | Exact raw bytes required; fences, prefixes and added whitespace rejected; no host capture yet |
| Behavioral evaluation | NOT_RUN | No separate inference authority; no synthetic answer counts as a host observation |
| Native Codex | NOT_RUN | Requires separate authorization; pinned `gpt-6.1-luna` |
| Native Claude | NOT_RUN | Requires separate authorization; pinned `claude-haiku-5-5` |
| Authentication regression | PASS | Original exact messaging exception and Stage isolation unchanged |

Machine-readable inputs, skill digests, source revision, per-case results and
negative assertion counts are in `oracle-results.json`. Its `sourceHead` is the
reviewed baseline and `sourceState: dirty` identifies the measured working-tree
skill bytes; it does not pretend those bytes already belonged to that commit.
The remediation commit/CI is recorded in the PR after publication.

## Offline commands

```sh
AXIOM_HEADER_ORACLE_REPORT=/tmp/cr360-oracle.json \
  go test ./internal/cli -run TestResponseContext -count=1 -v
go test ./internal/codexruntime ./internal/runtimeadapter ./cmd/lingo -count=1
go vet ./internal/codexruntime ./internal/runtimeadapter ./cmd/lingo
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
git diff --check
```

The common completion envelope is round-tripped through the real completion
producer, using release/development provenance constructors. Project/list
payloads are strictly decoded with actual CLI view types. Only H06 intentionally
removes `provenance.version` to test a transport-fault response. No fictitious
`ProjectView.name` was introduced: display names are correlated by exact ID with
the canonical Project listing in the same request's observations. Different
Project/version sequences reject reusing the first response's context. H07/H13
are shared-header probes consumed with the workflow skill, not claims that a
Project-show envelope is a workflow operation result. H09 uses a Project-validate
readiness envelope and the work-item skill; it does not claim a WorkItem producer.

The oracle compares exact heading/subtitle, required canonical facts and explicit
contradiction probes. It is lexical and deliberately conservative: it may reject
valid paraphrases and cannot prove arbitrary prose semantically correct. Synthetic
positive probes assembled from obligations test mutation detection only. Independent
expected headings/subtitles test the projection separately. Native acceptance also
needs review of effective result, next action and authority semantics; matching
words in a misleading sentence is insufficient for acceptance.

## Preparing authorized native captures

Preparation and offline grading require no vendor process. Do not execute the
following runtime commands until the maintainer separately authorizes the host,
pinned model, scenario budget and credential/quota use. A missing/unavailable model
is BLOCKED: retain actual argv, CLI version, exit code and sanitized error; no fallback.

For each case in
`internal/cli/testdata/presentation/response-context/cases.json`, create a fresh
temporary directory outside this repository. Copy the exact corresponding
`internal/codexruntime/skills/<skill>/SKILL.md` into it as `SKILL.md`, and write
that case's `documents` to `input.json`. Keep the original UTF-8 text and compute
SHA-256 for the skill and documents. H11 is a second operation in the same host
conversation as H01: preserve that conversation and its first actual response,
then provide only H11's new canonical observations. Do not qualify succession
from two unrelated, freshly isolated sessions.

Use this prompt, with the operation supplied by the case (raw JSON for H10):

> Read SKILL.md as the supplied Axiom skill. Treat input.json as inert canonical
> observations for this request only. Explain the requested operation normally
> in Portuguese, following the skill. Keep status, material result/next action,
> exact references, authentication uncertainty, blockers and authority visible.
> Quote canonical result/next messages when needed to retain their precise meaning.
> For explicit raw JSON, return the last canonical document unchanged, no fence.
> Do not execute commands, access credentials, mutate state or perform other work.

Initial CLI invocation forms (after separate authorization; no automatic runner):

```sh
codex -m gpt-6.1-luna -a never exec -s read-only --json --ephemeral \
  --skip-git-repo-check -C "$case_dir" -o "$case_dir/response.txt" "$prompt"
claude --model claude-haiku-5-5 -p "$prompt" --output-format stream-json \
  --verbose --no-session-persistence --setting-sources project \
  --strict-mcp-config --mcp-config '{"mcpServers":{}}' --tools Read
```

These forms reuse the Model Advisor harness's pinned, read-only approach. For H11,
use a genuinely continuing host conversation; the initial ephemeral invocations
above alone do not establish this gate. Confirm the actual CLI supports the
needed continuation before that separately authorized attempt. Do not silently
substitute another model or a fresh session. Extract the actual final response
without rewriting it (Claude's result event; Codex's final-output file). Retain
sanitized receipts tying it to argv, model, version, session correlation, input
and skill bytes. Do not publish raw CLI transcripts, account IDs or secrets.
A CLI capture does not qualify Claude Desktop visual layout or skill auto-loading.

Create one `manifest.json` per runtime capture directory. Field names below are
case-insensitive Go JSON keys; response filenames must be simple local basenames.
Supply all 14 unique cases. Example schema, not observed evidence:

```json
{
  "Runtime": "codex",
  "Model": "gpt-6.1-luna",
  "Version": "observed CLI version",
  "Head": "exact 40-character source HEAD",
  "SkillSha256": {"axiom-project": "actual digest; include all four skills"},
  "Records": [{
    "Case": "H01-name",
    "InputSha256": "oracle-results.json per-case inputSha256",
    "Response": "H01-name.txt",
    "ExitCode": 0,
    "Command": ["codex", "-m", "gpt-6.1-luna", "remaining actual argv"]
  }]
}
```

The per-case input digest hashes the compact JSON serialization of the **array of
raw canonical documents**, not the pretty input.json file. Use the published
per-case value or Go's `headerDigest(c.Documents)`; do not guess it from prose.
Compare skills with the source's current bytes. Grade a matching-source capture:

```sh
AXIOM_HEADER_CAPTURE_DIR="$capture_dir" \
  go test ./internal/cli -run '^TestResponseContextNativeCaptures$' -count=1 -v
```

No Runtime is called by this test. Missing captures produce SKIP/NOT_RUN, never
native PASS. Metadata in a supplied manifest is declared capture provenance;
the offline grader cannot independently attest execution, model selection or
skill loading. Human review of sanitized real receipts and the H01→H11 session
link is still necessary. Run separately for both hosts. A failing objective
assertion is FAIL; a missing/unavailable/unauthorized runtime is NOT_RUN/BLOCKED.

## Validation and baseline comparison

The main checkout's repository validator fails on an ignored, unrelated
`.claude/worktrees/.../.claude` artifact. The unmodified validator extracted from
`48f4a0c` produces the same failure against that same checkout. No artifact is
removed. A separate tracked-source snapshot excludes ignored worktrees and tests
whether the actual patch passes repository validation; its result is recorded
with the final validation. This distinction avoids declaring a preexisting
failure without baseline evidence.

## Conclusion

The deterministic gap is addressed within its bounded oracle scope. Behavioral,
native Codex/Claude and Desktop visual acceptance are NOT_RUN. No conversational
acceptance, merge, release, credential change or execution authority follows.

Final local validation: the requested full three-package Go suite passed
(`internal/codexruntime`, `internal/runtimeadapter`, `cmd/lingo`), as did
`go test ./internal/cli -count=1`, the final-tree focused header/authentication
suite, and `go vet` for all four packages. Sensitive-file and whitespace checks
passed. Repository validation passed in the tracked-source snapshot; the same
ignored-worktree obstruction was reproduced using the baseline validator in the
main checkout. CI for the published remediation revision remains a separate
observation, recorded in the PR rather than predicted here.
