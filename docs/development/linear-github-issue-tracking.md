# Linear and GitHub issue tracking

This guide defines **where Axiom work is tracked**, how a product deliverable maps to
technical work, and how to preserve delivery authority across Linear and GitHub.
It applies to the **AXM** team in Linear and the `rgomids/axiom` repository.

This is a **governance rule**, not evidence that a provider integration is installed,
that any synchronization is running, or that implementation is authorized.

This policy separates **operational backlog authority** (Linear), **technical
repository authority** (GitHub), and **canonical Axiom Execution/acceptance
contracts**. A workflow status is neither execution authority nor technical
acceptance.

## Ownership

| Concern | Authority |
|---|---|
| Product roadmap, priorities, release targets, user outcomes, product epics and stories | Linear (AXM) |
| Operational backlog of Linear-managed work: status, assignment, priority, sequencing and `Blocks` / `Blocked by` relationships (including the already mapped MVP work) | Linear (AXM) |
| Technical-only work items, defects, bounded engineering tasks, technical research and technical issue taxonomy | GitHub Issues |
| Specifications, architecture decisions, code, PRs, validation and evidence | Versioned GitHub repository |
| Technical delivery and stable publication | GitHub Release and its traceable evidence |
| Product acceptance and delivery status | Linear, reconciled against technical evidence and required human decisions |
| Public editorial product explanations | GitHub Wiki; not an operational tracker |

A Linear story may map to **multiple** GitHub Issues, and an engineering Issue
may have no Linear card. Do not force a one-to-one relationship, duplicate
technical specifications into Linear, or treat Linear projections as the
canonical Axiom Execution/workflow state.

The existing GitHub Issue taxonomy and Axiom-managed `axiom:*` projections
remain governed by [CONTRIBUTING.md](../../CONTRIBUTING.md) and approved
Specifications. Linear uses its native issue statuses instead of duplicating
GitHub `status:*` labels.

**Transition boundary:** For already mapped MVP items, the corresponding AXM
card is the operational source of truth for status, assignee, scheduling and
blocking/unblocking. Related GitHub Issues retain technical history and links;
their `status:*` labels or Project columns must not be used as a competing
product backlog status or to unblock an AXM successor. The companion label-policy migration makes `status:*` an **optional legacy**
GitHub editorial field: Issue Forms and Actions no longer seed or require it.
The existing mapped MVP Issues will have these labels removed only **after**
the migration PR is merged and the new default-branch workflow is active.
GitHub-only technical Issues keep existing labels unless separately reviewed.
`type:*`, `area:*`, `platform:*` and Axiom-managed `axiom:*` are unaffected.

## Where a new request starts

| Request | Create in | When to create a link elsewhere |
|---|---|---|
| User-facing capability, product epic, story, committed outcome or product discovery work | Linear | Create a bounded GitHub technical Issue only when required for distinct engineering implementation, Specification or research; do not mirror every Linear story. |
| Reproducible bug, regression or engineering defect | GitHub | Link a Linear deliverable only when the bug affects a product commitment, priority or date. |
| Refactor, CI/CD, maintenance, technical documentation or internal engineering improvement | GitHub | Link Linear only for a product-delivery dependency or material roadmap impact. |
| Specification, ADR or technical investigation | GitHub | Link the impacted Linear outcome when it is a prerequisite. |
| Product research requiring a bounded engineering experiment | Linear (outcome), then GitHub (technical work) | Record references to the investigation; keep claims of evidence and authority in their owning artifacts. |

Security vulnerabilities requiring non-public disclosure go through
[SECURITY.md](../../SECURITY.md), **not** a public GitHub Issue or a public
Linear/GitHub link.

For every request, search for existing work before creating another record.
Product epics in Linear do not replace GitHub technical epics where engineering
coordination is genuinely required.

## Linking product outcomes to engineering work

1. When there is a product commitment, add the URL of the related GitHub
   Issue(s) to the Linear card and the Linear reference to the relevant GitHub
   technical issue/PR context. Existing records may be linked instead of copied.
2. Keep product priority, assignee, operational status, target date, milestone,
   sequencing, `Blocks` / `Blocked by` relations and user-facing scope in Linear.
   Keep technical acceptance criteria, Specifications, ADRs, implementation
   constraints, evidence, PR metadata and release facts in GitHub. GitHub Issue
   relationships may be retained for traceability, but do not independently
   drive the operational state of a mapped Linear card.
3. A single Linear card may depend on several GitHub Issues: do not complete
   the product card until **all** necessary technical work and its own product
   acceptance conditions are satisfied.
4. Preserve the original GitHub Issue numbers and Linear identifiers when
   reconciling existing work. Do not enable a bulk issue sync that could create
   duplicate work items without an explicit migration plan.

### Pull request references

Every PR **must** still follow the existing
[delivery metadata contract](../../CONTRIBUTING.md#pull-request-issue-metadata).
Its two authoritative lines accept **GitHub Issue numbers only**:

```text
Related-Issues: #123
Completes-Issues: none

Relates to AXM-123
```

The numbers above are illustrative.

- `Related-Issues` records engineering Issues advanced by the PR.
- `Completes-Issues` names only GitHub Issues for which the PR satisfies all
  remaining technical acceptance criteria (or `none` for partial work). This
  is **not** an authorization to close the Issue on merge.
- `Relates to AXM-123` is the Linear PR-linking **relation** keyword: when
  the integration is configured, it attaches the PR without automatically
  changing Linear status.
- Never put a Linear ID in the GitHub `Related-Issues` or `Completes-Issues`
  fields. Do not use GitHub closing keywords or Development-sidebar
  closing links; the repository explicitly forbids closing Issues at merge.
- Do not assume PR linking is active until its installation and configuration
  have been verified. The Linear reference is optional for GitHub-only
  engineering work with no product counterpart.

Use relation-only linking to avoid silently advancing or closing product
work. A GitHub PR may link several relevant Linear cards, but add only
justified product relationships.

## Delivery states and acceptance

| Linear state | Meaning |
|---|---|
| `Backlog` / `Todo` | Discovered / prioritized for execution; neither grants execution authority. |
| `In Progress` | Work actually started under the required authority. |
| `In Review` | Implementation or delivery under technical/human review. |
| `Awaiting Release` | All technical work necessary for this Linear outcome has been integrated and satisfies pre-publication conditions; stable publication is pending. |
| `Done` | All necessary GitHub Issues were delivered by a stable release, traceable evidence exists, and the Linear outcome meets its acceptance conditions, including any required human decision. |
| `Canceled` / `Duplicate` | Alternatives to delivery as appropriate, not a generic blocked status. |

Use Linear `Blocks` / `Blocked by` relationships for operational dependencies,
including already mapped MVP activities. A completed predecessor should be
reconciled against actual acceptance evidence before its successor is treated
as unblocked. GitHub Issue relationships may remain as technical references,
not as a second unblock mechanism. Legacy `status:blocked` is optional on
GitHub-only technical Issues, not a replacement for Linear relations or an
extra Linear workflow column.

A PR opening, review approval, green CI or squash merge **never** proves a
stable release or human acceptance. Per the repository's existing
[delivery tracking](../../CONTRIBUTING.md#delivery-tracking), GitHub Issues
stay open until published in a stable GitHub Release. Do not infer product
`Done` from PR completion or the closure of one of several related GitHub
Issues.

If the technical and product systems disagree, record the discrepancy, use
their respective authoritative evidence, and require reconciliation before
performing a destructive, external or acceptance-changing action.

## Integration rollout: explicit future work

**Adopted policy:** link PRs to relevant Linear issues using `Relates to`,
and keep product status transitions intentional. Defining Linear as the
operational authority does **not** enable automatic transitions or equate
PR creation, approval or merge with completion.

**Not authorized by this document:**

- Two-way or indiscriminate one-way GitHub Issues Sync;
- Automatic creation of Linear issues from every GitHub bug/task/PR;
- Automatic Linear `Done` on PR merge or commit reaching `main`;
- Automatically closing GitHub Issues on merge;
- Automatically publishing releases, advancing Execution gates or recording
  human acceptance.

Recommended rollout, **only when separately authorized**:

1. Configure the native Linear/GitHub **PR integration** for the intended
   repository and AXM team; verify personal/account mapping where relevant.
2. Audit the team's PR/commit workflow automations. Avoid any merge rule
   that marks `Done` or bypasses the Axiom stable-release acceptance rule.
3. Test one existing mapped GitHub Issue / Linear card with a **relation-only**
   PR reference, and verify both sides without changing workflow state.
4. Consider a later **GitHub stable release -> Linear status projection**
   with explicit source-ID mapping, idempotency, audit trail, duplicate
   prevention, human acceptance gates and reconciliation/retry rules.
   Define its requirements and authority before implementing an API/webhook.
5. Treat historical GitHub Issues and already-manually-mapped Linear cards
   as a migration/reconciliation problem, not as permission for bulk
   two-way synchronization.

The provider's [GitHub integration documentation](https://linear.app/docs/github)
describes PR linking, relation keywords, issue sync and configurable
automations. Provider behavior is not evidence that the Axiom integration has
been enabled.

## Maintenance

This versioned document is the **canonical contributor-facing cross-tool
routing policy**. [CONTRIBUTING.md](../../CONTRIBUTING.md) points here;
[documentation governance](../documentation.md) defines the owning sources.
Linear's product-facing explanation should summarize this rule and link to
the reviewed repository version, without copying volatile operational status.
Any future change to syncing semantics must update the applicable owner
artifacts and be reviewed through a normal PR.
