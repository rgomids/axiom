# AXM-38 T03 captures — 2026-10-22 / attempt-02

See [qualification report](../evidence-qualification.md) before interpreting these
files. **SYNTHETIC source; real Code Desktop output; AI observer, no human witness.**
No captured file alone establishes complete host qualification.

- `session-manifest.json`: pre-session grant and original input/skill hashes.
- `artifact-manifest.json`: final file hashes, capture sizes and input references.
- `semantic-evaluation.json`: selected actual SVG/AX facts compared independently
  with expected Stage IDs, gates and exact predecessor lists. Matrix verification
  tests every cell center against the actual SVG polygon; expected edges are the
  independent literal rule row > column. Missing-edge control must fail equality.
  This evaluation is not a product format, renderer or proof of every field.
- `c01*-prompt.txt` through `c04-prompt.txt`: exact requests. C02's one diagnostic
  continuation is retained in `c02-actual-ax.txt.gz`.
- `*-actual-ax.txt.gz`: real host accessibility snapshots, possibly including previous
  responses. Lossless gzip retains the original AX bytes, including whitespace; decode with
  `gzip -dc <file>`. Trees can normalize/truncate text and do not prove exact code points.
- `custom-unselected-concurrency1.json`, `custom-divergent.json`, `dense-32.json`:
  additional synthetic wrappers prepared via existing Go Encode/JCS, not live CLI
  results. Original inputs remain under `internal/codexruntime/testdata/rendered-workflow`.
- `*.svg`: actual model-generated static transient presentation outputs, copied
  unchanged from the isolated scratch after completion. They are Evidence, never
  executable product assets or a new presentation source of truth.

Unmodified UI screenshots:

| File | What is actually visible |
| --- | --- |
| `c01-literal-mermaid.png` | Retained graphical failure: literal Mermaid source. |
| `c01-table-clipping.png` | Retained cramped/clipped table. |
| `c01-agent-detail.png` | Custom reviewer declared configuration, independent text. |
| `c01b-inline-top.png`, `c01b-inline-dag.png` | Actual custom SVG inline with Browser pane closed; top identity/Stages and lower gates/independent join. |
| `c02-inline-default-top.png`, `c02-inline-default-gates.png` | Default SVG inline, ordered Stages and gates; Browser closed. |
| `c02-selection-detail.png` | Distinct inspected r3/selected r2 references and cap-1 independent predecessors. |
| `c03-inline-overview-top.png`, `c03-inline-gates.png` | Actual dense overview's two segments with continuation/identity and four gates. |
| `c03-inline-matrix.png` | Full 32×32 matrix, orientation, identity and counts. |
| `c03-fallback.png` | Actual Agent detail plus explicitly simulated failure-condition notes. |
| `c04-text-retrieval.png` | Real text-only supplement/retrieval and model's exact-copy limitation. |

No screenshot is synthesized or used to replace a failed observation. Vertical
scrolling is required; some viewport captures show only an explicitly named part.
Images/AX retain only this public synthetic session, with unrelated sidebars hidden.
The scratch session is left reviewable; permissions restored to Manual. No live
installation, native terminal graphic, remote renderer or human acceptance.
