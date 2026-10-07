# Generated Work Items read as concise native Markdown

<!-- axiom:work-item-draft:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa -->
<!-- axiom:provenance:Axiom:development:abc123def456:clean -->

**Work Item type:** `task`

## Problem

\# heading
\<!-- comment -->
- list item
**bold**
`code`

## Desired outcome

Generated Work Items read as concise native Markdown.

## Context

> [!NOTE]
> Quoted note with a [link](https://example.com) and <https://example.com/docs>.

- [ ] open task
- [x] done task

#135 stays an Issue reference; 2 < 3 stays text.

## Scope

Run:

 ```sh
 # not a heading
 go test ./internal/githubissues
 <!-- literal in code -->
 ```

 ~~~
 ## also code
 ~~~

## Constraints

Keep correlation markers machine-readable.

## Non-goals

No lifecycle or classification change.

## Acceptance expectations

Golden tests cover representative Issues.

---

_Structure authored by Axiom `development` (revision `abc123def456`, clean source); section content keeps its declared authorship._
_All section content is user-authored._
