# AXM-37 — Linux v1 CI diagnostic r1

## Scope, authority and preflight

2026-10-21; same Implement attempt `linux-v1-ci-diagnostic-r1`, branch
`docs/axm-37-offline-operational-documentation`, Draft [PR #335](https://github.com/rgomids/axiom/pull/335).
All executable checks below target source HEAD
`630eaf7555aa10cdad494b083bf77a1cc1cbc6b6`; documentation-only successor commits
do not retroactively qualify their CI. Local/remote/PR heads agreed; tree clean
before this diagnostic. AGENTS.md, security/documentation policies, ADR-0033/0034,
the relevant archive code/tests, prior Evidence and Linear discussion were read.

The first preflight stopped at an unresolved accepted dispatch. Rafael's
subsequent decision A confirms this Codex-on-Mac session is the sole authorized
executor and that the accepted dispatch initiated this execution. That gate is
resolved; accepted dispatch alone is not evidence of competition. Bounded local
process/cwd inspection identified this AXM-37 executor and a separate AXM-36
executor, with no concrete competing AXM-37 edit/executor. This is not a
distributed lease. No new dispatch, Execution, PR or agent was created.
Sequential work: tightly coupled reproduction did not benefit from delegation.

Authority covers diagnosis, bounded tests and non-behavior-changing Evidence/Tasks
in this PR, including diagnostic commit/push. It does **not** cover a v1 contract
or prerequisite change, default activation, native hosts/sessions, merge,
release, status/relationship mutation or acceptance. Provider discussion was
used for scope; only task-specific technical observations are published here.

## Exact CI and causal finding

[CI run 39657000010](https://github.com/rgomids/axiom/actions/runs/39657000010)
is **FAIL**, and [delivery metadata 39657000042](https://github.com/rgomids/axiom/actions/runs/39657000042)
is **PASS** for the source HEAD above, verified through commit workflow lookup.
[Job 119419810222](https://github.com/rgomids/axiom/actions/runs/39657000010/job/119419810222)
checked out synthetic merge `5d7d03db47ca30232099b1d8e58ae4fe8c90f2cf`,
combining that source HEAD with base `0f1073ac6a3af615d48b651340d8fe895827080a`.
Runner: Ubuntu 24 image `20261018.339`, Go 1.26.0 linux/amd64.

Only reported failing subcase:
`TestShellArchiveIntegrityBeforeExecution/1/zero_trailing`.
Its log is `go_accept=false shell_accept=true`, with shell verification success
for format 1 / linux / amd64 and Go `upgrade: archive_invalid`.
Installed-documentation journey and repository checks were skipped after Tests
failed. They are **NOT_RUN** in this run, not implicitly PASS.
`internal/executiongraph` is **PASS** here (`13.817s`); the historical isolated
failure is not the cause of this CI failure.

The fixture appends **16 zero bytes after a completed gzip member** and
recomputes SHA256SUMS. This is different from `zero_tar_tail`, which puts 512
zero bytes inside the compressed tar stream after tar EOF.

1. Shell verifies SHA256, then uses `gzip -dc | head -c 536870912` with
   `pipefail`. GNU gzip 1.12 returns 0 for the outside-member zero padding;
   Apple gzip 487.0.1 emits "trailing garbage ignored" and returns 2.
2. Both decoders emit exactly the original tar bytes. The subsequent physical
   tar EOF walk, tar listing, hashes and metadata therefore cannot detect the
   ignored compressed bytes. This is a gzip-boundary discrepancy, not a tar
   EOF/checksum/CRC bypass.
3. Go first reads with `Multistream(false)`, observes unconsumed input, then
   retries legacy multistream. Its next gzip header is zeros: `gzip: invalid
   header`, propagated by the trailer drain as archive invalid. Valid split
   members and empty prefix/suffix members still work in v1.
4. Only detected v2 runs the additional Python/zlib single-member/no-tail check.
   That check refuses the same compressed padding; it must not be copied to v1
   unchanged because v1 explicitly accepts multiple members.

Independent inert tar probes confirmed the decoder mechanism with identical
input bytes on Mac and local Linux. Base wire SHA256:
`591e55b4d3dc9e7cfe951ffa8eb826f23a0ad7c92b8384283d81f30dcb6ce9b4`;
base plus 16 zeros:
`43c0fcfa0ec8a02ea0eae70cff3d8c61c5bbf62d0c67c41917f55db43c395b8f`.
These primitive probes contain only `payload.txt`, not an installable bundle.
Actual acceptance tests separately use complete, checksum-valid Axiom fixtures.

**Security boundary FAIL reproduced:** a temporary diagnostic overlay (below)
continues past the permanent test's early mismatch assertion. On local Linux,
verify-only succeeds without executing anything; normal entry invokes the
harmless candidate `version` sentinel and creates its temporary marker despite
Go refusal. The sentinel exits 91. Shell then reports a non-executable-workspace
error; that message does not negate the observed execution. No bin/receipt
destination is created, and no real Axiom/Runtime session runs. The permanent
test aborts at the mismatch before its normal-entry sentinel check, so its CI
log alone did not establish this additional observation.

## Environments and test matrix

Mac: Darwin 27.0.0 arm64, macOS 27.0.1, Go 1.26.1; Apple gzip 487.0.1,
bsdtar 3.5.3 / libarchive 3.7.4 / zlib 1.2.12; Python 3.14.4 / zlib 1.2.12.
Local Linux: existing Docker image `gcc:14-bookworm`, Debian 12.11 arm64,
GNU gzip 1.12, GNU tar 1.34; Go test executable cross-compiled with Go 1.26.1,
CGO disabled. Image ID:
`sha256:5e927c284bf55a7dc796262e311a0703344f62f41f5621eb56843111b1d37e15`.
No pull, dependency installation, network or external host provisioning.
Container root and checkout read-only; temporary files in disposable tmpfs.

| Input/control | Go v1 / v2 | Mac shell v1 / v2 | Local Linux shell v1 |
| --- | --- | --- | --- |
| Valid single member | accept / accept | accept / accept | accept |
| 16 zeros after member | refuse / refuse | refuse / refuse | **accept: FAIL** |
| 512 zeros inside tar tail | accept / accept | accept / accept | accept |
| Split member at byte 17 | accept / refuse | accept / refuse | accept |
| Empty suffix/prefix members | accept / refuse | accept / refuse | accept |
| Concatenated complete tars; nonzero tar tail | refuse / refuse | refuse / refuse | refuse |
| Nonzero compressed garbage; CRC corruption | refuse / refuse | refuse / refuse | refuse |

Local Linux Go-only v1/v2 gzip controls also **PASS**. Full local Linux shell v2
matrix is **NOT_RUN**: this image has no Python. No interpreter was installed.
Direct execution of the unchanged v2 Python block on Mac independently rejects
outside-member zeros, garbage, split and empty prefix/suffix; accepts valid
single member and zero tar tail. This is a decoder check, not native Linux v2
qualification. CI v2 had no reported failing subcase; CI suite remains FAIL.

| Command/check at source HEAD | Result |
| --- | --- |
| `GOPROXY=off GOTOOLCHAIN=local go test ./internal/install -count=1 -timeout=180s` (Mac) | **PASS**, 53.220s |
| Mac `TestShellArchiveIntegrityBeforeExecution -count=1 -v`, 34 subcases | **PASS**, including seven repaired framing cases and legacy controls |
| Linux existing integrity subset below, 10 v1 cases | **FAIL**: exactly zero_trailing; other nine PASS |
| Linux `TestLegacyGzipMembers`, `TestGzipCorruptionRefused`, `TestV2SplitGzipRefused` | **PASS** |
| Temporary overlay observation, Mac v1/v2 and Linux v1 | Observation **PASS**; Linux no-execution safety property **FAIL**, marker=true |
| Raw identical-byte decoder matrix | Observation **PASS**: Mac gzip rc=2, GNU gzip rc=0, Go invalid header for outside-member zeros; tar rc=0 in both |
| Isolated executiongraph command below (Mac) | **FAIL**, 2/3 iterations lost result |
| Whole-repo Go suite/vet/build, release/bootstrap pipelines | **NOT_RUN** in this diagnostic; no behavior changed |
| Native T13–T15/Q6, v0.1.0 qualification | **NOT_RUN / UNVERIFIED**, unchanged |
| `./scripts/validate-repository.sh .` on diagnostic docs | **PASS**; runtime behavioral scenarios explicitly skipped/UNVERIFIED |
| `gitleaks detect --source . --no-git --redact` | **PASS**, no leaks |
| Staged sensitive-file check and complete diagnostic diff review | **PASS** |

Isolated command:
`GOPROXY=off GOTOOLCHAIN=local go test ./internal/executiongraph -run
'^TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation$' -count=3 -timeout=30s`.
Two failures returned ExitCode=-1, empty output, Stopped=true,
OutcomeUnknown=true; one iteration passed. No executiongraph source changed.
No race or scheduling mechanism is diagnosed from timing. Prior isolated FAIL,
prior full-suite PASS and current CI package PASS remain separate observations.

Initial Linux container run omitted executable tmpfs and failed five otherwise
valid sentinel controls as well as zero_trailing. It is **FAIL / unsuitable for
normal-entry qualification**. Repeating with `/tmp:rw,exec,nosuid` restored all
five controls; only zero_trailing remains FAIL. No production behavior was
changed to obtain that result.

## Compatibility decision and smallest safe next action

ADR-0033 and ADR-0034 are Accepted at their named human revisions. They require
explicit format/migration boundaries and preserve historical receipts; neither
authorizes new v1 gzip acceptance or a new v1 tool prerequisite.
`TestLegacyGzipMembers`, existing empty-member controls and current scope
require v1 header/body/metadata splits plus empty-member compatibility.
Multistream is not equivalent to arbitrary bytes after the last valid member.

**Recommendation, not authorization:** align shell v1 with existing Go refusal
of every non-member compressed tail, while preserving valid legacy multistream
and zero padding inside tar. Use a bounded parser of the complete member
sequence before candidate execution. Reusing Python/zlib is a candidate, but
would extend its prerequisite to v1; the existing v2 single-member test cannot
be applied unchanged. This narrows GNU-shell acceptance and needs explicit
human approval even though Go already refuses the package.

| Option | Compatibility/security trade-off | Authority now |
| --- | --- | --- |
| A: strict shell v1 member sequence, reject outside-member tail; explicitly approve Python >=3.6/stdlib zlib for v1 if chosen | Preserves Go refusal and legacy multistream; removes pre-probe mismatch. GNU-shell-padded archives lose acceptance; v1 hosts may gain prerequisite. Historical published-package qualification still needed | Decision required |
| B: define zero padding after gzip as legal v1 and change Go plus shell consistently | Broadens Go; Apple shell also needs coordinated handling. Risks accepting unvalidated data if generalized beyond zeros; does not repair pre-probe equivalence by changing Go alone | Decision required; not recommended |
| C: defer patch and retain truthful failing CI/diagnostic gate | No contract/prerequisite change; safety mismatch and red CI remain. No waiver or test relaxation | No behavior change |

Precise next decision: **authorize A's bounded v1 compressed-tail refusal and
choose whether Python/zlib may become a v1 bootstrap prerequisite**, or retain
the gate and request another bounded parser design. Approval must preserve
split/empty legacy members, checksums/CRC, both tar EOF blocks, all seven
repaired framing refusals, quotas, identity, symlink/path and lock controls;
require Mac/Linux acceptance and no-execution sentinel regressions.
No patch is authorized or implemented by this Evidence. Default remains v1,
v2 opt-in. T13–T15/Q6 deferred to AXM-12; T16 partial; AC-08/v0.1.0 unproven.

## Reproduction without repository code edits

Run from the source HEAD. Use the existing image only if already present;
otherwise classify local Linux checks NOT_RUN rather than pull/install.
Copy the temporary overlay source below to `$diag/diagnostic_overlay_test.go`.

```bash
diag=$(mktemp -d /tmp/axm37-diagnostic.XXXXXX)
# Place the appendix source in "$diag/diagnostic_overlay_test.go".
python3 - "$diag" "$PWD" <<'PY'
import json, pathlib, sys
d, root = map(pathlib.Path, sys.argv[1:])
(d / "overlay.json").write_text(json.dumps({"Replace": {
    str(root / "internal/install/axm37_diagnostic_overlay_test.go"):
    str(d / "diagnostic_overlay_test.go")}}))
PY
GOPROXY=off GOTOOLCHAIN=local go test -overlay "$diag/overlay.json" \
  ./internal/install -run '^TestAXM37ZeroTailDiagnostic$' -count=1 -v
GOPROXY=off GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go test -c ./internal/install -o "$diag/install.test"
GOPROXY=off GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go test -overlay "$diag/overlay.json" -c ./internal/install -o "$diag/observe.test"
# Existing integrity subset: expected FAIL only for zero_trailing.
docker run --rm --pull=never --network=none --read-only --cap-drop=ALL \
  --security-opt=no-new-privileges --tmpfs /tmp:rw,exec,nosuid,size=512m \
  --mount "type=bind,src=$PWD,dst=/src,readonly" \
  --mount "type=bind,src=$diag,dst=/diag,readonly" \
  --workdir /src/internal/install gcc:14-bookworm /diag/install.test \
  -test.run '^TestShellArchiveIntegrityBeforeExecution$/^1$/(valid|zero_trailing|zero_tar_tail|split|extra_empty|prefix_empty|trailing|crc|complete_tars|tar_trailing)$' \
  -test.count=1 -test.v -test.timeout=90s
# Repeat that container command with /diag/observe.test and:
# -test.run '^TestAXM37ZeroTailDiagnostic$/^1$' -test.count=1 -test.v
# Pure-Go controls use /diag/install.test and:
# -test.run '^(TestLegacyGzipMembers|TestGzipCorruptionRefused|TestV2SplitGzipRefused)$'
```

The observation harness returns PASS when observations complete; its log's
`marker=true` is the safety FAIL, not acceptance of the mismatch. It does not
replace or relax any committed test. Sentinel paths vary; archive checksums are
recalculated per fixture, so outcome is deterministic rather than path-byte identity.

```go
package install

import (
 "bytes"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "testing"
)

// Temporary diagnostic only. No existing test or production source is changed.
func TestAXM37ZeroTailDiagnostic(t *testing.T) {
 for _, version := range []int{1,2} {
  t.Run(string(rune('0'+version)),func(t *testing.T) {
   work:=t.TempDir()
   marker:=filepath.Join(work,"probe")
   sentinel:=[]byte("#!/bin/sh\nprintf probe > '"+marker+"'\nexit 91\n")
   var archive,sums string
   if version==1 {
    b:=newBundle("1.1.0",sentinel)
    if runtime.GOOS=="linux" {
     meta:=string(b.contents()["release-metadata.txt"])
     meta=strings.ReplaceAll(meta,"platform=macos-27","platform=linux")
     meta=strings.ReplaceAll(meta,"goos=darwin","goos=linux")
     meta=strings.ReplaceAll(meta,"architecture=arm64","architecture="+runtime.GOARCH)
     b.files["release-metadata.txt"]=[]byte(meta)
    }
    archive,sums=b.write(t,work)
   } else {archive,sums=newV2ArchiveFor(t,string(sentinel),"").write(t)}
   wire:=append(gzipMembers(t,tarBytes(t,archive)),make([]byte,16)...)
   replaceArchive(t,archive,sums,wire)
   _,ge:=LoadCandidate(archive,sums)
   script,_:=filepath.Abs("../../scripts/install-release.sh")
   args:=[]string{"--archive",archive,"--checksums",sums,"--bin-dir",filepath.Join(work,"bin"),"--receipt-dir",filepath.Join(work,"receipts")}
   out,se:=exec.Command("bash",append([]string{script,"--verify-only"},args...)...).CombinedOutput()
   _,me:=os.Stat(marker)
   t.Logf("verify sha256=%s go=%v shell=%v output=%s marker=%t",digest(wire),ge,se,bytes.TrimSpace(out),me==nil)
   if me==nil {t.Fatal("verify executed candidate")}
   out,se=exec.Command("bash",append([]string{script},args...)...).CombinedOutput()
   _,me=os.Stat(marker)
   _,be:=os.Stat(filepath.Join(work,"bin"))
   _,re:=os.Stat(filepath.Join(work,"receipts"))
   t.Logf("normal shell=%v output=%s marker=%t bin=%t receipts=%t",se,bytes.TrimSpace(out),me==nil,be==nil,re==nil)
   // Observe mismatch without replacing the permanent refusal assertion.
   if be==nil || re==nil {t.Fatal("persistent destination created")}
  })
 }
}
```
