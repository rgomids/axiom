# Evidence — #369 Plan / attempt-01

2026-11-04. Companion to [Plan](plan-369-pre-spawn-launch.md). These observations
support planning; none establishes that the runtime defect is fixed.

## Source and authority observations

Local/remote/PR HEAD before edits: `bb5c3759a4fce98fb9ee0d19bab4a44d6318f72b`.
Branch fix/369-launch-refused, clean worktree, open draft PR370 targeting main.
Issue body and complete paginated comment timeline were inspected, including
[HD-369-D](https://github.com/rgomids/axiom/issues/369#issuecomment-6621938832).
The accepted Specify bytes remain unchanged. Graphify query (backend none)
returned a truncated map; source inspection, not inferred graph edges, supports
the Plan inventory. Bounded independent agent research inspected persistence
and decoder source; the executor integrated it and reran the isolated proof.
Equivalent capability agents were used for source review, no model test battery.

Observed baseline CI: PR Checks SUCCESS and delivery-metadata SUCCESS on Specify
HEAD (runs 41768332259 and 41768332252). This is not CI for the new Plan commit.
Final publication receipt reports newly observed checks without assuming completion.

## Pinned dependency decision (TECHNICAL_LOCAL TD-369-A)

Select `github.com/BurntSushi/toml v1.6.1-0.20260818200445-d733fc535e4a`, exact
[upstream commit d733fc535e4a9f3c454e421a87b90ade5d49bf31](https://github.com/BurntSushi/toml/commit/d733fc535e4a9f3c454e421a87b90ade5d49bf31).
Module checksum `h1:i5cHSnm+mFd2ykiM2+nL8g1ZrAI4PkNNxrwW7v112p8=`;
go.mod checksum `h1:k/kw8BIRbzIP/cZ/NdyzJ6MIWiYy8zSND2mCu/VjF0s=`.
[COPYING](https://github.com/BurntSushi/toml/blob/d733fc535e4a9f3c454e421a87b90ade5d49bf31/COPYING)
is MIT; README declares TOML1.1; go.mod requires Go1.21, below repository Go1.26.
Use its ordinary Decoder, explicitly `MaxDepth(64)` before Decode into map values.
No dependency was added to the repository in PLAN.

Source inspected from downloaded pinned Go modules:
[decode.go MaxDepth/Decode](https://github.com/BurntSushi/toml/blob/d733fc535e4a9f3c454e421a87b90ade5d49bf31/decode.go#L127),
[parse.go valueArray](https://github.com/BurntSushi/toml/blob/d733fc535e4a9f3c454e421a87b90ade5d49bf31/parse.go#L396),
[valueInlineTable](https://github.com/BurntSushi/toml/blob/d733fc535e4a9f3c454e421a87b90ade5d49bf31/parse.go#L430),
[addContext](https://github.com/BurntSushi/toml/blob/d733fc535e4a9f3c454e421a87b90ade5d49bf31/parse.go#L548),
[iterative lexer nextItem](https://github.com/BurntSushi/toml/blob/d733fc535e4a9f3c454e421a87b90ade5d49bf31/lex.go#L79).
Array depth is checked before recursively parsing values. Inline-table context
length is checked before parsing child values; array depth and table context are
separate counters. Configured64 gives at most64 active array containers plus fewer
than64 table contexts, hence fewer than128 combined containers. Default128 alone
is insufficient to establish a combined128 bound. Dotted/table context is bounded
as well. Axiom still needs its separate32-depth/65536-node semantic walk and1MiB
pre-read bound. The 500000-array fixture is 1000003 bytes, inside1MiB.

Alternative `github.com/pelletier/go-toml/v2 v2.4.3`, commit
071a36c2a57244f2e70369bfc69889fda2a1f60f, MIT, checksum
`h1:GTRvJQutkOSftxIFD5xw9aepkYNuPWmVJpffdDPYVpY=` is rejected.
[GHSA-frc4-6h9q-39fq](https://github.com/pelletier/go-toml/security/advisories/GHSA-frc4-6h9q-39fq)
lists <=2.4.2 affected,2.4.3 patched. Pinned unstable/parser.go:70 sets
maxValueNesting=10000; parseVal applies it without a public limit setting.
The isolated test confirms depth129 is accepted, violating Axiom's bound.
Stable BurntSushi v1.6.0 lacks the applicable bound and is also rejected.
An exact maintained upstream pseudo-version avoids a private fork/custom parser;
trade-off: preserve the exact source proof and fixtures through future upgrades.

Exact-version OSV API queries for both candidates returned `{}` (no advisory
matches observed on this date, not proof that no vulnerability exists): POST
`https://api.osv.dev/v1/query` with package ecosystem Go/name and exact version.
Repeat advisory and checksum review before implementation adoption/release.
No source or dependency install scripts were executed, only Go module download and
synthetic test compilation. No private config was accessed.

## Reproducible isolated parser proof — PASS

Executed in `/tmp/axiom369-toml-proof`: `go test -v -count=1 .`.
An isolated module required only the two exact versions above, leaving repository
go.mod/go.sum unchanged. Recreate outside the worktree with Go1.26 and these exact
requirements, then run the following proof as proof_test.go. It tests dependency
behavior; it is documentation of an experiment, not runtime implementation.
Proof SHA256: `2a089d542c98304b98662208bf0641763f1e91a65a00004e99241eca800a1a57`.

```go
package proof
import (
 "strings"
 "testing"
 b "github.com/BurntSushi/toml"
 p "github.com/pelletier/go-toml/v2"
)
func TestPinnedBounds(t *testing.T) {
 arrays:=func(n int)string{return "a="+strings.Repeat("[",n)+"0"+strings.Repeat("]",n)}
 tables:=func(n int)string{return "a="+strings.Repeat("{k=",n)+"0"+strings.Repeat("}",n)}
 alternating:=func(n int)string{return "a="+strings.Repeat("[{k=",n)+"0"+strings.Repeat("}]",n)}
 for _,c:=range []struct{name,wire string; refuse bool}{
 {"array64",arrays(64),false},{"array65",arrays(65),true},{"array129",arrays(129),true},{"array500000",arrays(500000),true},
 {"inline129",tables(129),true},{"mixed129",alternating(65),true},{"dotted129",strings.Repeat("a.",128)+"a=1",true},
 {"notify","notify=[\"synthetic\",\"arg\"]\nx={a=1}\n",false},{"duplicate","a=1\na=2",true},
 {"multiline","notify=[\n\"x\",\n]\nx=\"\"\"line\nline\"\"\"",false},
 } {t.Run(c.name,func(t *testing.T){var v map[string]any; d:=b.NewDecoder(strings.NewReader(c.wire));d.MaxDepth(64);_,e:=d.Decode(&v);if (e!=nil)!=c.refuse{t.Fatalf("refuse=%v error=%v",c.refuse,e)}})}
 var v map[string]any
 if e:=p.Unmarshal([]byte(arrays(129)),&v);e!=nil{t.Fatal("pelletier unexpectedly rejected129:",e)}
 t.Log("pelletier v2.4.3 accepts depth129; BurntSushi configured64 rejects overbound fixtures")
}
```

PASS: configured64 accepted array64, notify+inline table and multiline forms;
refused array65/129/500000, inline129, mixed65 pairs, dotted129 and duplicates.
Pelletier accepted depth129. These are isolated parser checks; full conformance,
fuzzing and Axiom integration are NOT_RUN.

## Validation executed for PLAN

- PASS: `go test ./internal/runtimeadapter -run 'TestEffortSourcesTOMLNarrowGrammar|TestOperationalEffortConflictsAtEveryFixedSource|TestCheckFrozen' -count=1`.
  This tests existing behavior only, not the proposed parser or fixes.
- PASS: `./scripts/validate-repository.sh .` (repository bootstrap, documentation, ADR/automation/label/security contracts; behavioral scenarios skipped by that script).
- PASS: `./scripts/check-sensitive-files.sh .`; `git diff --check`.
- Staged security and link/marker assertion results are recorded in the Issue receipt.
- NOT_RUN: proposed B/C/D behavior, old-decoder harness against future schema,
  full Go suite/race/platform suite, real provider/AI battery, host reproduction,
  inference, production/E2E. These remain later Tasks/Implement obligations.
- Dedicated gitleaks executable unavailable; required sensitive-file checks still apply.

## Review and limitations

Ponytail-review and Axiom review disciplines applied to documentation against
accepted Spec/HD/ADR and source consumers. No material architecture conflict was
identified by source inspection. Format4 optional fields and downgrade safety are
a proposed design pending executable old-reader tests; do not report them as PASS.
Only Plan/Evidence/index are changed. No runtime code, budget defaults, frozen
corpus or personal config changed. No additional human decision is outstanding.

Independent source/Plan review found no blocking AC-D/strict-codec contradiction.
It specifically identified snapshot isolation for pointer metadata; the Plan now
requires scalar fields or appropriate deep cloning. This is review evidence, not
an executed future schema/crash test.
