# Work Item classification — issue #136

## Scope and authority

Bounded implementation requested by the maintainer for
[issue #136](https://github.com/rgomids/axiom/issues/136). This amendment adds
creation-time delivery intent and provider classification to the existing draft
contract. It does not establish human acceptance, authorize publication, or
depend on the interview/body-formatting follow-ups.

## Behavior

- Every complete draft has the provider-neutral type `story`, `bug`, or `task`.
  `--type` supplies it explicitly; guided creation asks for it, and the runtime
  skill can propose it from the conversation. Legacy non-interactive callers
  without a type normalize to a visible `task`.
- A story requires separately authored beneficiary and concrete user/product
  value, in addition to the existing seven sections. Implementation activity
  belongs in scope. The runtime asks for value or proposes task when only
  implementation is described. Missing story value cannot produce a draft.
  Semantic adequacy remains part of human review of the complete draft; the CLI
  validates required content and provenance rather than pretending to judge
  arbitrary natural language.
- Repeatable `--classification` supplies provider-neutral classification intent.
  The GitHub adapter interprets these values as existing label names. Explicit
  values override inference and unknown values fail before creation.
- Without explicit classifications, the GitHub adapter prefers `type:<type>`,
  then the plain type, with `enhancement` as a story fallback. It uses canonical
  names from the observed catalog and does not infer area or lifecycle labels
  from arbitrary prose. No suitable label produces an empty reviewed label set
  and `no_existing_label_for_item_type` notice.
- Type, story value, and explicit classification intent participate in recovery
  identity. The complete adapter document, selected labels and notices
  participate in the final preview digest. Changed proposals require review.
- The catalog is paginated (100 per page, at most 10 pages). Malformed,
  duplicated or incomplete catalogs fail explicitly. The adapter checks selected
  labels again before POST and never creates a label for this operation.
- Labels travel in the single issue-create JSON payload through stdin. Dropped
  labels produce a partial result with the existing Issue reference. Confirmed
  retries and ambiguous-create reconciliation verify classification read-only;
  they never create another Issue to repair metadata.
- GitHub field names, label catalogs, mapping and verification remain inside
  `internal/githubissues`. The domain transports opaque adapter metadata in
  the reviewed provider document; no GitHub label concept enters local state.

## Plan and compatibility

Extend the Work Item draft, CLI flags/guided prompts, GitHub document adapter,
and embedded runtime skill. Preserve existing authority and durable attempt
fences. No new dependency, credential handling, manifest migration, or release
is required. Historical embedded skill digests remain recognized by both
supported runtime integrations.

## Acceptance evidence

| Issue criterion | Executable boundary |
|---|---|
| Every created Work Item has an explicit type | `TestTypesAndStoryDeliveryValue`; typed GitHub body; CLI transport tests |
| Stories express value beyond implementation | Missing beneficiary/value rejection; rendered story value; guided/skill contract |
| Labels visible before creation | Adapter metadata in complete preview; digest-change authority tests |
| Only existing/authorized labels; explicit degradation | Catalog selection, pre-POST validation, dropped-label and retry tests |
| Provider metadata behind adapter boundary | Neutral classification/opaque document contracts; GitHub-owned mapping and payload |

Validation results are recorded in [classification evidence](evidence-136.md).

## External sources

The [GitHub issue-create API](https://docs.github.com/en/rest/issues/issues#create-an-issue)
documents that labels can be silently dropped without push access. That is why
creation and recovery verify applied classification and report partial results.
The [repository label API](https://docs.github.com/en/rest/issues/labels#list-labels-for-a-repository)
defines catalog discovery and pagination. These sources do not authorize
creating labels or expanding provider permissions.
