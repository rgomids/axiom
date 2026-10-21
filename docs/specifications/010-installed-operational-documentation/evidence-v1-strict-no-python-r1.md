# AXM-37 — Strict v1 without Python, r1: blocked preflight

## Authority and preflight

2026-10-21; Decision A, attempt `v1-strict-no-python-r1`, same Draft
[PR #335](https://github.com/rgomids/axiom/pull/335), branch
`docs/axm-37-offline-operational-documentation`. Local/remote/PR HEAD read back
as `593d02f079b1f9757e79e35de36fc85b1ce966bd`; initially clean.
AGENTS.md, orchestration/implementation/review/documentation/security policies,
AXM-37 and recent receipts, ADR-0033/0034, the four named archive files and
[prior diagnostic](evidence-linux-v1-ci-diagnostic-r1.md) were inspected.

Process ancestry identifies this session as the AXM-37 executor of the identical
accepted prompt. Bounded process/cwd inspection found no concrete competing
AXM-37 executor; the other named executor is AXM-36. Not a distributed lease.
No second implementation dispatch, Execution, host or Runtime session created.
No delegation: coupled preflight, no demonstrated strictly inferior hierarchy.
The Linear shared-context endpoint returned "Tool get_agent_context not found";
issue/comments worked. Only task-specific technical observations are reproduced.

**Outcome: BLOCKED before a production correction.** An attractive existing-tool
fix rejects zero padding, but demonstrably fails the stronger requirement to
reject *every* non-gzip tail. Experimental production/test changes were removed
back to the exact clean baseline; this delivery contains Evidence/Tasks only.
No dependencies, v1/v2 defaults, ADRs, accepted history, status or relationships
changed. No implementation HEAD is claimed ready for code review.

## Counterexample to the minimal gzip fix

[GNU's padding documentation](https://www.gnu.org/software/gzip/manual/html_node/Tapes.html)
explains why ordinary `gzip -dc` accepts zeros. GNU gzip 1.12's verbose mode
reports zero padding with warning exit 2. The temporary candidate changed only:

```diff
-if ! gzip -dc "$staged_archive" | head -c 536870912 >"$staged_tar"; then
+if ! gzip -dcv "$staged_archive" 2>"$temporary/gzip-diagnostics" | head -c 536870912 >"$staged_tar"; then
```

The original output bound and pipefail then refuse zero tails and preserve valid
split/empty gzip members. Apple gzip also passes these controls. However GNU
gzip accepts an **empty UNIX compress (.Z) stream after the valid gzip member**:
tail bytes `1f 9d 90`. These bytes are not another gzip member. No additional
tar output is emitted, so physical tar EOF and hashes cannot detect this tail.

An actual checksum-valid Axiom v1 fixture with this tail produced:

```text
environment=linux/arm64 gzip=gzip 1.12
go_accept=false shell_accept=true shell=archive_verify_success: format=1 platform=linux architecture=arm64
invalid archive executed candidate
```

Normal entry reached the harmless candidate sentinel; no persistent bin/receipt
destination was created. The sentinel exits 91, so a later installer error does
not erase candidate execution. Go's existing reader refuses the tail. Apple
gzip refuses it before the sentinel. Thus `-v` alone is **not** a strict gzip
framing verifier. It must not be shipped as satisfying Decision A.

A preliminary appended-witness/double-decode experiment passed its selected
zero/garbage/split controls. It was not adopted: success of an alternate decoder
on transformed bytes does not establish a strict original gzip member grammar
when the decoder also accepts non-gzip formats. No exhaustive proof is claimed.
A full-output witness comparison would be necessary, not merely a suffix check;
the experiment was removed together with the verbose-only prototype.

## Reproducible isolated diagnostic

[Diagnostic overlay patch](v1-strict-no-python-r1-probe.patch) applies to
`internal/install/archive_shell_integrity_test.go` at baseline `593d02f`.
It is an Evidence fixture, **not an active test relaxation**. It selects six
controls per generation, adds .Z, records the actual gzip version and moves the
shell/Go mismatch assertion after normal-entry no-execution checks. Existing
permanent tests and production sources remain byte-identical to baseline.

```bash
probe=$(mktemp -d)
chmod 700 "$probe"
git show 593d02f079b1f9757e79e35de36fc85b1ce966bd:internal/install/archive_shell_integrity_test.go > "$probe/probe_test.go"
patch "$probe/probe_test.go" < docs/specifications/010-installed-operational-documentation/v1-strict-no-python-r1-probe.patch
git show 593d02f079b1f9757e79e35de36fc85b1ce966bd:scripts/install-release.sh > "$probe/install-release.sh"
sed 's/gzip -dc "$staged_archive"/gzip -dcv "$staged_archive"/' "$probe/install-release.sh" > "$probe/verbose.sh"
PROBE="$probe" node -e 'const f=require("fs"),p=process.env.PROBE;f.writeFileSync(p+"/overlay.json",JSON.stringify({Replace:{[process.cwd()+"/internal/install/archive_shell_integrity_test.go"]:p+"/probe_test.go"}}))'
GOPROXY=off GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go test -overlay "$probe/overlay.json" -c ./internal/install -o "$probe/install.test"
docker run --rm --pull=never --network=none --read-only --cap-drop=ALL \
  --security-opt=no-new-privileges --tmpfs /tmp:rw,exec,nosuid,size=768m \
  --mount "type=bind,src=$PWD,dst=/src,readonly" \
  --mount "type=bind,src=$probe,dst=/diag,readonly" \
  --mount "type=bind,src=$probe/verbose.sh,dst=/src/scripts/install-release.sh,readonly" \
  --workdir /src/internal/install gcc:14-bookworm /diag/install.test \
  -test.run '^TestShellArchiveIntegrityBeforeExecution$/^1$' -test.count=1 -test.v -test.timeout=90s
```

Exact executed paths were `/tmp/axm37-v1-strict-r1/overlay.json`,
`strict_probe_test.go`, `probe.test`, and `install-release-verbose.sh`.
The verbosity candidate also captured stderr privately; that does not alter
acceptance. Overlay compilation on Mac used Go 1.26.1 and CGO disabled.
Existing image `gcc:14-bookworm`:
`sha256:5e927c284bf55a7dc796262e311a0703344f62f41f5621eb56843111b1d37e15`.
No pull, dependency install, network, external host or privileged container.

Independent primitive, executed with the same existing container:

```bash
{ printf payload | gzip -c; printf '\037\235\220'; } | gzip -dcv >/dev/null
# GNU gzip 1.12: exit 0 (FAIL strict-tail property).
{ printf payload | gzip -c; printf '\000\000'; } | gzip -dcv >/dev/null
# GNU gzip 1.12: warning, exit 2 (PASS zero-tail refusal).
```

## Results and SHA limits

| Environment/source/check | Result |
| --- | --- |
| Baseline `593d02f` script + selected zero-tail safety regression, local GNU 1.12 arm64 | **FAIL**, candidate sentinel executed |
| Baseline + temporary verbose-only script + diagnostic overlay, local Linux arm64 | **FAIL** .Z tail; five other controls PASS (valid, zero tail, split, empty suffix/prefix) |
| Same verbose-only prototype + six v1 controls on macOS 27.0.1 arm64 / Apple gzip 487.0.1 | **PASS**, 6.110s; no cross-platform strictness claim |
| Verbose prototype, macOS 46 shell/Go integrity cases + corruption/member/Python-prerequisite controls | **PASS**, 42.713s; incomplete suite misses .Z without overlay |
| Preliminary witness prototype, macOS 48 integrity cases + Go/Python controls | **PASS**, 37.968s; withdrawn, not framing proof |
| Preliminary witness prototype, local GNU arm64 24 shell v1 controls and Go controls | **PASS**, withdrawn; not native amd64 qualification |
| Preliminary witness prototype, Mac full install package | **PASS**, 84.460s; not final-head proof |
| Exploratory full Go suite while prototypes changed | **PASS**, cmd/lingo 434.682s, install 109.832s; mutable prototype source, not final-head verification |
| Exploratory Mac vet/build/format/tidy, bash syntax/diff, repository validator | **PASS**, prototype source only |
| Local staticcheck command | **FAIL to start**, binary absent; not an analysis PASS |
| Prototype offline installed-docs harness | Executable checks **PASS**, overall **PARTIAL**; native/source/network controls **NOT_RUN** |
| Final documentation-HEAD CI | Exact HEAD, run URLs and terminal outcomes are read back in the [single callback receipt](https://linear.app/rgomids/issue/AXM-37/distribuir-documentacao-operacional-versionada-junto-a-instalacao-do#comment-98f13feb); this record does not claim pending checks passed |
| Native T13–T15/Q6, AC-08, v0.1.0 | **NOT_RUN / UNVERIFIED**, unchanged; T16 partial |

Additional executed commands:
`GOPROXY=off GOTOOLCHAIN=local go test ./internal/install -count=1 -timeout=240s`,
`go test ./...`, `go vet ./...`, `go build ./...`,
`./scripts/check-go-quality.sh format`, `./scripts/check-go-quality.sh tidy`,
`./scripts/check-go-quality.sh staticcheck`,
`bash -n scripts/install-release.sh`, `git diff --check`,
`./scripts/test-docs-offline.sh`, `./scripts/validate-repository.sh .`,
`./scripts/check-sensitive-files.sh --staged .`,
`gitleaks detect --source . --no-git --redact`.
Go commands used `GOPROXY=off GOTOOLCHAIN=local`. Secret scans PASS; no
prototype PASS is relabeled final-source acceptance.

## Alternatives and next gate

1. **Existing gzip verbose mode alone:** smallest change, demonstrated insufficient;
   rejects zeros but accepts empty .Z. Rejected for the required strict guarantee.
2. **Trusted strict framing helper using an existing library:** reuse Go
   compress/gzip or zlib in a trusted verifier available *before* candidate
   execution. Requires an explicit bootstrap/acquisition/packaging decision;
   candidate execution and a new compiler/interpreter dependency cannot be
   inferred. Recommended direction to evaluate, not an adopted contract.
3. **Custom bounded framing parser within current shell toolchain:** avoids a host
   dependency but must understand DEFLATE block/Huffman framing to locate trailers,
   plus optional headers/member boundaries. This is security-sensitive new parser
   work, materially larger than the demonstrated gzip-mode fix. Requires bounded
   design, differential/adversarial validation and explicit scope disposition.
4. **Python/zlib v1 prerequisite:** reuses the v2 technique with a multistream loop,
   but conflicts with this invocation. Not implemented or installed.

No universal impossibility theorem is claimed. The available existing-decoder
mechanism was falsified; no strict, dependency-neutral replacement has been
demonstrated within this bounded correction. Stop rather than deliver a known
incomplete security guarantee. Next: human choice of verifier strategy/scope,
then implementation, full exact-HEAD CI and actual independent review.
Historical FAILs remain; no new review dispatch is warranted for an unimplemented
correction. Callback must carry `stage.blocked`, this attempt and exact final HEAD.
The [same callback comment](https://linear.app/rgomids/issue/AXM-37/distribuir-documentacao-operacional-versionada-junto-a-instalacao-do#comment-98f13feb)
is updated and reread after documentation push and CI. Stable event ID:
`AXM-37:implement:v1-strict-no-python-r1:stage-blocked`. It owns the final SHA/run
read-back, avoiding a self-referential documentation commit or duplicate callback.
