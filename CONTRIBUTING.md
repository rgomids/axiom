# Contributing to Axiom

## Project maturity

Axiom is in early development. The Codex harness, tested Go foundations, and a
bounded local Lingo control plane are available. See [project status](README.md#project-status)
for stable capability context and the [Specifications index](docs/specifications/README.md)
for authoritative scope and lifecycle state. The roadmap expresses direction,
not implementation authorization.

## Before contributing

- Search [existing Issues](https://github.com/rgomids/axiom/issues) before opening a new one.
- Open or discuss an Issue before a significant change. Identify any affected Specification, ADR, or contract.
- Do not implement Tasks without explicit authority. An Issue or roadmap entry is not approval.
- Read [AGENTS.md](AGENTS.md), [documentation governance](docs/documentation.md), and the [Code of Conduct](CODE_OF_CONDUCT.md).
- Never include secrets, credentials, or private data in Issues, commits, PRs, or evidence. Sanitize reproductions and logs.
- Report vulnerabilities through the private process in [SECURITY.md](SECURITY.md), never in a public Issue. See [SUPPORT.md](SUPPORT.md) for other requests.

## Issue labels

Open Issues use one label taxonomy. Release or phase is never a label.

| Label | Meaning | Rule for an open Issue |
|---|---|---|
| `type:*` (`epic`, `story`, `task`, `bug`, `research`) | nature of the Work Item | exactly one |
| `area:*` (`cli`, `skills`, `work-item`, `workflow`, `runtime`, `execution`, `installer`, `ci-cd`, `governance`) | primary area | one; a second only when it improves classification; an Epic may have none |
| `status:*` (`planned`, `active`, `blocked`) | editorial backlog state | exactly one; closing the Issue means done |
| `platform:*` (`windows`, `linux`, `macos`) | platform restriction or impact | optional, any number |
| `axiom:*` | reserved for Axiom | never edited by hand |
| milestone | release or phase | e.g. `MVP` |

`axiom:*` labels are system-managed and must not be manually repurposed: they
are Axiom's projection of workflow state (see
[Work Item lifecycle governance](#work-item-lifecycle-governance)).

Open Issues through the [Issue Forms](.github/ISSUE_TEMPLATE/) (Bug, Story,
Task, Research); blank Issues are disabled. Epics are created deliberately by
maintainers, without a form. Each form applies its `type:*` and
`status:planned`; [`issue-label-policy.yml`](.github/workflows/issue-label-policy.yml)
maps the selected Area and Platform answers to labels. It fills only an empty
`status:*` (`status:planned`) or `area:*` (the form answer), never removes or
replaces a label, and reports any other violation (missing or conflicting
type, conflicting status, missing area, non-canonical or retired labels such
as `scope:mvp`, `slice:*`, `bug` or `enhancement`) in one comment that it edits
as the Issue changes. Platform answers are applied once, when the Issue is
opened.

Work Items created with `axiom work-item create` follow the same taxonomy:
declare both families explicitly with `--classification type:<type>
--classification area:<area>` (for example `type:story` and `area:cli`); the
policy then adds `status:planned`. Axiom never infers an area, and explicit
classification replaces type inference, so omitting either value leaves a
reported violation. The rules and the canonical catalog (names, colors, descriptions) live
in [`scripts/issue-label-policy.py`](scripts/issue-label-policy.py); a change
to it on `main` creates or normalizes the catalog labels, and never deletes
one.

## Development flow

Axiom uses [GitHub Flow](https://docs.github.com/en/get-started/using-github/github-flow).
`main` is the only long-lived branch and is always releasable; there is no
`develop` branch and no GitFlow.

```text
feature/fix branch -> Pull Request -> required CI -> review + approval -> squash merge -> main
```

Requirements and setup details live in [Getting Started](docs/development/getting-started.md). Git, Bash, standard POSIX utilities, and Go 1.26 or later are required for the checks below. The first Go run may download the dependency pinned in `go.mod`.

```bash
git clone https://github.com/rgomids/axiom.git
cd axiom
./scripts/validate-repository.sh .
go test ./...
```

1. Branch from the current `main` with one small, focused scope, following the branch naming convention below.
2. Consult affected contracts; add or adjust tests appropriate to the change, using test-first development where useful.
3. Keep commits coherent. Do not mix unrelated refactors or changes.
4. Update only affected documentation and include verifiable Evidence: commands, outcomes, and known limits. Raw claims of completion are insufficient.
5. Run the relevant checks below, inspect the diff, and open a Pull Request against `main`. Address review feedback within the agreed scope.
6. After required CI passes, approval is given and every review thread is resolved, the PR is squash-merged. Nobody pushes directly to `main`.

A merge to `main` never starts, versions or publishes a release: merges
integrate code; [`$axiom-release`](#release-flow) starts releases.

## Work Item lifecycle governance

The Issue #94 amendment, approved by human review on 2026-09-24, establishes the
following rules for Axiom-managed Work Items. Approval of the amendment does not by
itself authorize T26–T29 implementation; S6 implementation remains separately gated.

- Repository artifacts remain technical source of truth; local Execution/workflow
  state remains canonical workflow truth; Provider metadata is a durable
  projection and recovery signal only.
- GitHub lifecycle projection uses exactly one canonical `axiom:stage:*` marker
  from the closed Specification 004 set. `axiom:blocked`,
  `axiom:needs-decision`, `axiom:needs-approval`, and
  `axiom:recovery-required` are independent flags, never synthetic stages.
- Do not manually treat labels, Issue closure, merge, green CI, or PR review as
  authority to advance local workflow or record acceptance.
- Transition comments must be bounded and reference Specifications, Plans/Tasks,
  PRs, Evidence, next actions, or blockers instead of copying dense documents,
  logs, or chat.
- Apply Project metadata policy deterministically. Ask only for required Work
  Item or Pull Request metadata still unresolved after policy/context resolution;
  keep concrete GitHub fields in the adapter boundary.
- If local workflow truth is unavailable, inspect and reconcile. Never reconstruct
  an Execution, advance a canonical gate, or infer a derived stage from
  Provider/Repository state; contradictory or insufficient facts require
  `recovery_required` and human decision.

PRs affecting this lifecycle must state the current and target canonical Execution
gates, the current and expected derived lifecycle stages, prerequisites and
authority, auxiliary flags, Provider effects, recovery impact, and whether a human
acceptance decision remains pending. `Not applicable` is valid only with an
objective reason.

## Branch names

Use this format:

```text
<category>/<concise-kebab-case-description>
```

Use a lowercase category that identifies the kind of work or its bounded delivery context. Prefer the commit types below when they fit. Categories such as `agent`, `issue`, `impl`, `release`, and `poc` are also acceptable when they make the workflow context clearer.

Examples:

```text
agent/s3-intent-work-item
docs/spec-004-tasks-approval
issue/62-mvp-specification
fix/pages-deployment-artifact
security/filesystem-destination-ownership
```

Use lowercase ASCII letters, digits, and hyphens; do not use spaces or underscores. Include a relevant Issue, Specification, Task, or Slice identifier when it improves traceability, but do not invent one when none applies. Avoid vague names such as `changes`, `update`, `wip`, or `my-branch`. Each branch must represent one coherent scope, and its name never grants implementation or external-effect authorization. Branches named `release-please--*` belong to the Release PR automation; do not push to them.

## Commit messages

Axiom uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```text
<type>(<optional-scope>)<optional !>: <concise description>
```

Accepted types are `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`, `ci`, `chore`, `security`, and `revert`. The scope is optional; use it only when it improves understanding.

Examples:

```text
feat(web): add bootstrap landing page
fix(pages): correct deployment artifact path
docs(contributing): define commit message requirements
ci(pages): deploy static site to GitHub Pages
security(filesystem): reject unsafe destination ownership
feat(cli)!: rename the configure command
```

Because PRs are squash-merged, **the PR title becomes the commit on `main`**
and must follow this format; the squash body may carry footers.
The `delivery-metadata` check validates the title on PR opens, edits and
updates, using the validator from the base revision. Require this check in
the repository ruleset; a passing check cannot establish whether the chosen
type describes the change correctly.

Versions come from Release Please's default versioning strategy as pinned by
[`release-please.yml`](.github/workflows/release-please.yml)
(`release-please-action` v5.0.0, which bundles release-please 17.6.0) and
configured in [`release-please-config.json`](release-please-config.json)
(`bump-minor-pre-major: true`, `bump-patch-for-minor-pre-major: false`). The
next version is the largest bump among the commits since the last release:

| Commits since the last release | Next version while `0.x` | Next version from `1.0.0` |
|---|---|---|
| any breaking change (`!` or `BREAKING CHANGE:` footer, any type) | minor | major |
| `feat` | minor | minor |
| any other type (`fix`, `security`, `perf`, `revert`, `docs`, `test`, `refactor`, `build`, `ci`, `chore`, ...) | patch | patch |

The first release has no previous release and uses `initial-version` (`0.1.0`).
A `Release-As: X.Y.Z` footer forces the version.
[`scripts/release-plan.sh`](scripts/release-plan.sh) applies this table in the
release preflight; Release Please computes the version of the Release PR, and
a Release PR whose version differs from the plan is refused (no required
checks, `release.sh start` fails).

Whether a release is releasable is a separate rule: Release Please skips a
release whose changelog would be empty. Only `feat`, `fix`, `security`,
`perf` and `revert` have visible changelog sections (Features, Bug Fixes,
Security, Performance, Reverts); `docs`, `test`, `refactor`, `build`, `ci` and
`chore` are hidden in the changelog, not non-releasable. So:

- hidden types alone are not releasable: the plan reports `next_action=none`;
- a breaking change or a `Release-As` footer is always listed, whatever its
  type, and so is releasable;
- hidden commits still count toward the bump above (they can never exceed
  patch) and are left out of the changelog text.

Each commit must represent one coherent unit of change. Its subject must explain the observable purpose, not merely identify modified files. Generic messages such as `wip`, `update`, `changes`, `fix`, `misc`, `added stuff`, or equivalents are not acceptable in final history. Do not group unrelated changes in one commit.

Temporary `fixup!`, `squash!`, or WIP commits may exist on a branch; the squash merge replaces them with the reviewed PR title. Add a body when the subject alone does not make the context or rationale clear.

## Validation

Run commands directly from the repository root. The [command reference](docs/commands.md) explains their scope and offline options.

```bash
./scripts/validate-repository.sh .
./scripts/validate-agent-package.sh .
./scripts/check-sensitive-files.sh .
go test ./...
go vet ./...
go build ./...
go mod verify
git diff --check
```

Repository validation also enforces the [repository automation policy](.agents/policies/repository-automation.md): every durable script or workflow must be registered in [`scripts/automation-registry.json`](scripts/automation-registry.json), and one-off helpers stay out of the tracked tree.

Before every commit, inspect staged paths and content and run `./scripts/check-sensitive-files.sh --staged .`. Follow [repository security](docs/security/repository-security.md), including the dedicated secret scanner when available. Report unavailable checks explicitly. Harness and documentation checks do not prove unimplemented product behavior.

### Required CI

The [CI workflow](.github/workflows/ci.yml) runs on every Pull Request and every
push to `main`. Its jobs are the required checks of the `main` ruleset
([desired state](.github/rulesets/main.json)):

| Required check | Runs |
|---|---|
| `verify (linux)`, `verify (macos)` | `go test -race ./...`, `go vet ./...`, `go build ./...`, `go mod verify`, `./scripts/validate-repository.sh .`, `./scripts/dogfood-poc.sh` |
| `release-contract` | `./scripts/test-release-pipeline.sh`, `./scripts/test-release-flow.sh` |

CI has a read-only token and never publishes anything. Renaming a job changes a
required check: update the ruleset file and the repository ruleset together.

## Pull requests

Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md). A PR without a description is not ready for review. Replace every template placeholder with real information or an objective explanation of why that item is not applicable. The description must provide:

- context and the problem being solved;
- scope and non-goals;
- related Issue, Specification, or ADR when applicable;
- validations executed, their results, and reproducible Evidence;
- documentation and security impact;
- known limitations;
- confirmation that unrelated changes are absent.

CI results do not replace Evidence or the context required in the PR description. Before requesting review, the author must inspect the PR title, description, and commit history.

### Ready for review

A PR is ready for review only when, at minimum:

1. its description is complete;
2. its scope is coherent;
3. relevant Issue, Specification, and ADR references are linked;
4. applicable validations and Evidence are recorded;
5. known limitations are declared;
6. temporary or WIP commits are cleaned up; and
7. the diff contains no unrelated changes.

### Merge policy

`main` is protected by a repository ruleset: changes arrive only through a Pull
Request with at least one approval, code owner review ([CODEOWNERS](.github/CODEOWNERS)),
all review threads resolved and all required checks green. Squash merge is the
only merge method; force-push and deletion of `main` are blocked. Only an
authorized maintainer merges, and an agent never merges its own Pull Request
without explicit human authorization.

A technical merge does not establish human acceptance. Acceptance does not automatically authorize the next Task. Contributions may be declined, split, or reformulated during review. Required explicit authorization must be recorded before work proceeds.

Accepted submissions are licensed under [Apache-2.0](LICENSE), unless explicitly stated otherwise. Contributors must hold the rights necessary to submit their content; identify any different licensing explicitly for review.

## Delivery tracking

Issues stay open until the functionality they describe is published in a
stable GitHub Release. Merging a Pull Request never closes an Issue.

### Pull Request Issue metadata

Every Pull Request declares its Issue relationship in its description, which
the squash merge copies into the commit on `main`:

```text
Related-Issues: #132
Completes-Issues: none
```

- `Related-Issues`: every Issue the PR advances, or `none`.
- `Completes-Issues`: only the Issues whose remaining acceptance criteria this
  PR completes, or `none`. Each one must also be in `Related-Issues`.
- One line per key, at the start of a line, outside code blocks and HTML
  comments, and at most 72 characters long (GitHub wraps the squash commit
  body near that width). Values
  are `none` or same-repository references separated by commas (`#12, #34`).
  Each key appears exactly once. Duplicates, other repositories, URLs,
  near-miss keys (`completes-issues:`, `Completes-Issue:`) or more than 20
  Issues are refused.
- A partial PR (for example one Task of a multi-Task Issue) lists the parent
  Issue only in `Related-Issues`. Only the PR that completes the Issue lists it
  in `Completes-Issues`. A mention, review, label, green CI or merge never
  implies completion.
- Do not use GitHub closing keywords (`close`, `fix` or `resolve` and their
  forms, followed by an Issue reference) in the title or description, and do
  not link Issues in the PR's Development sidebar: GitHub would close the
  Issue at merge time. Both are refused.

The [`delivery-metadata`](.github/workflows/delivery-metadata.yml) check
validates this with
[`scripts/delivery-issues.sh check-pr`](scripts/delivery-issues.sh), using the
base revision's validator. It also requires that the metadata parse the same
after a 72-column wrap, and that GitHub's `closingIssuesReferences` for the PR
is empty. Release PRs deliver no Issue and are not validated. Release Please
(dispatched only by `$axiom-release`) updates them with `GITHUB_TOKEN`, which
starts no `pull_request` run, so `release-please.yml` dispatches the check on
the Release PR branch, next to CI. That dispatch runs the default branch's script and passes only for the
head of the open bot-authored Release PR dispatched by the bot, and only when
that Release PR would close no Issue at merge; any other dispatch fails and
is never skipped. The integrity of this check, like that of every required
check, rests on code-owner review of `.github/workflows/**`: a workflow file
always comes from the PR or dispatched ref, only the script comes from the
base or default branch. The desired `main` ruleset makes
`delivery-metadata` a required check; applying that ruleset is a separate,
authorized administrator action (see
[repository security](docs/security/repository-security.md#delivery-tracking)).

The check cannot observe a Development-sidebar link added after its last run,
so delivery sync has a post-merge fail-safe. When GitHub closes an Issue at
the merge that completes it, before the stable release that delivers that
merge recorded it, sync writes a bounded `axiom-delivery:reopened` record,
reopens the Issue and continues the normal `Awaiting Release` projection. It
reopens only when the Issue's last close event names exactly that merged PR
(number from the squash subject, with that merge commit) or that commit as
closer, and the reason is `completed`. It never reopens when a bot release
record was written after that closure (a publication saw it), whatever window
the run covers, and it re-checks this just before the reopen. A record written
before the closure belongs to an earlier delivery of a reopened Issue and does
not block it. A manual
close, another PR or commit, or an unreadable closer is reported and left
alone. Limits: it covers only Issues the merge completes; GitHub closes
linked Issues asynchronously, so a closure that lands after that push's sync
is repaired by the next push to `main` (at the latest the Release PR merge
of the release that contains it), which re-scans the window.
Keep the metadata intact when editing the squash commit message. If a merged
commit's metadata is missing or malformed, the release preflight
(`$axiom-release`) stops before any Release PR exists, naming the commit. The
fix is a normal reviewed PR to `main` that adds a line to
[`.github/delivery-corrections.txt`](.github/delivery-corrections.txt) of the
form `<full-sha> related=<n,m|none> completes=<n,m|none>`, with its reason
recorded above it. The file is read as committed at the revision being
resolved, so by default a correction counts only for releases whose release
commit contains it; a correction merged before the release starts is always
contained. History is never rewritten. Only a release whose release commit
already exists with inconsistent history (v0.3.0) needs the exception of
[pinned metadata recovery](docs/development/release-recovery.md)
under ADR-0010: a reviewed correction/control SHA and the committed file digest
are additional immutable inputs, bound into the preparation and publication
envelope. Source revision, archives, changelog, range and Project configuration
remain those of the original release commit. No floating-ref fallback exists.

### Migration boundary (v0.2.0)

Commits merged before this contract carry no metadata and deliver no Issue.
Old closing keywords are not reinterpreted. The only exceptions are two
reviewed records in `.github/delivery-corrections.txt`:

- PR #143 (`f04dbf3`) explicitly implements #129;
- PR #148 (`fc6cdf4`) completes #147; it closed that Issue with a keyword at
  merge.

These records let the `v0.2.0` envelope record #129 and #147 as delivered by
`v0.2.0`. Their effect is `comment`, plus `project` when the projection is
enabled. Both Issues were already closed by a keyword, and closed is not
Released, so they are recorded but not re-closed. #132 is not completed: #145
delivered only `I132-T01`. From `v0.2.0` on, only `Completes-Issues` delivers
an Issue. `v0.2.0` is also the legacy boundary for undeclared commits: in the
range of every later stable release, each ordinary first-parent commit must
declare metadata or have a reviewed correction (which may be
`related=none completes=none`). Only that release's own Release Please commit
may stay undeclared. An unexpected undeclared commit makes release resolution
fail closed; its closing keywords are never interpreted. Any Project #5 snapshot or mutation digest taken before these
records is stale and must be refreshed by read-back before a Project
migration. This applies only if this contract reaches `main` before Release PR
#146 is merged, so that Release Please regenerates its release commit on top
of it. If #146 merges first, `v0.2.0` publishes with the earlier scripts, and
recording #129 needs a separately authorized manual action.

### Lifecycle

```text
Planned -> In Progress -> In Review -> Awaiting Release -> Released (Issue closed)
```

| Event | Effect | Authority |
|---|---|---|
| Issue added to the Project | Status `Planned` (Issues only; Pull Requests stay in their own view) | maintainer or a verified built-in workflow |
| Work starts, PR opened | Status `In Progress`, then `In Review` | maintainer, manually |
| A PR with `Completes-Issues: none` merges | nothing changes; the commit keeps its `Related-Issues` | PR merge |
| A PR with `Completes-Issues: #N` merges | bounded comment on #N; Status `Awaiting Release`; `Target Release` cleared because the version is not yet known; #N stays open; if GitHub closed #N at exactly this merge before any stable release, it is reopened with a bounded record; any other closed #N is reported and left alone | PR merge; [`delivery-sync.yml`](.github/workflows/delivery-sync.yml), which re-scans from the latest release range on each push |
| The Release PR merges (release commit) | `Target Release = vX.Y.Z` for every Issue that release delivers | Release PR merge; `delivery-sync.yml` |
| A release candidate is published | nothing changes: an RC delivers no Issue and closes none | authorized envelope |
| The stable release is published and read back | for each delivered Issue: bounded comment linking the release, Status `Released`, `Target Release = vX.Y.Z`, closed as completed | the same authorized publication envelope |
| Projection enabled after a release published while it was disabled (or the Project drifted) | every closed/completed Issue whose latest `axiom-delivery:released` record names its release is repaired to `Released` and that `Target Release`; consistent items are not written; no record, a foreign record or a later record for another tag means no repair | `delivery-sync.yml` on each push, from `v0.2.0` on |

The Issues a stable release delivers are the `Completes-Issues` of the
first-parent commits after the previous release commit, up to and including
its own release commit. They are deduplicated across PRs, so consecutive
releases never share an Issue. The release notes list them under
`### Issues delivered`. Each title is sanitized and rendered as a code span,
so an Issue title cannot mention users or link references from a release.

The Project status, `Target Release` and the delivery comments are a delivery
projection for human visibility. They are not Axiom Execution truth and not
the `axiom:stage:*` lifecycle of Axiom-managed Work Items (see
[Work Item lifecycle governance](#work-item-lifecycle-governance)). They
create no local fact and record no human acceptance. Blocked, decision,
approval and recovery conditions stay out of the Status field.

The Project is the existing `Axiom Base Line` #5, which is being evolved
into `Axiom Delivery`; no second Project is created (see
[repository security](docs/security/repository-security.md#delivery-tracking)).
`.github/delivery-project.json` binds it and stays at
`"projection": "disabled"` until the separately authorized migration is done.
While disabled, comments and closure still happen and the envelope says
`delivery_project=users/rgomids/projects/5 projection=disabled`. The Issue and
its release record stay canonical; the Project is reconstructed from them once
the projection is enabled. `Legacy Done`
is a temporary migration status for historical `Done` items. It is never
`Released`, and the scripts never set or read it.

## Release flow

Merges integrate code. `$axiom-release` starts releases
([ADR-0011](docs/decisions/0011-command-driven-release-start.md)).

```text
normal development:  PR -> review -> squash merge -> main -> CI (-> delivery projection) -> done

release:  human intent -> $axiom-release
     -> preflight: every commit since the last release (Conventional Commits,
        delivery metadata) -> SemVer plan (planned_version, planned_tag; no tag)
     -> dispatch Release Please -> Release PR -> STOP
     -> human review + approval + squash merge (the release commit)
     -> $axiom-release: prepare (build + verify the release commit) -> publication envelope -> STOP
     -> explicit human authorization of exactly that preview_digest
     -> publish the same bytes: tag vX.Y.Z[-rc.N] + GitHub Release -> read-back verify
     -> stable only: release, record and close the delivered Issues
```

GitHub Releases is the initial distribution channel. Starting a release,
preparing a versioned state, building/verifying artifacts and publishing are
separate steps, and publication always requires explicit human authority.
The tag is created only by the publication.

### Versioning

Versions follow [SemVer](https://semver.org/). Public tags are:

- stable: `vMAJOR.MINOR.PATCH`, published as a normal GitHub Release, and the
  `latest` release when it is the highest stable version;
- release candidate: `vMAJOR.MINOR.PATCH-rc.N`, published as a GitHub
  prerelease, never `latest`, installable only by its exact tag.

Tags are never moved or deleted, and a published release is never replaced.
[`scripts/release-tag-version.sh`](scripts/release-tag-version.sh) and
[`scripts/release-preflight.sh`](scripts/release-preflight.sh) implement the
tag and ordering rules.

### Release PR

[Release Please](https://github.com/googleapis/release-please)
([workflow](.github/workflows/release-please.yml),
[config](release-please-config.json)) opens the Release PR against `main`
only when `$axiom-release` (`scripts/release.sh start`) dispatches it; a push
or merge never runs it. Before that dispatch,
[`scripts/release-plan.sh`](scripts/release-plan.sh) validates every
first-parent commit since the last release commit: a Conventional Commit
subject and declared (or reviewed, committed) delivery metadata. It also
requires the tag of the previous release at its release commit and the
planned tag to be absent and newer than every stable tag; `release.sh` also
requires that previous release to be a published (non-draft) GitHub Release
bound to that tag and commit, so a tag alone never lets a new release start. One inconsistent
commit stops the release, for example
`release_error: ... commit <sha> has no delivery metadata`, before any Release
PR, artifact, tag or release exists. The workflow requires `main` to still be
the planned SHA at dispatch time, re-runs the plan on it, then runs Release
Please, which proposes the version, groups the commits into a new
`CHANGELOG.md` section and updates
[`.release-please-manifest.json`](.release-please-manifest.json). The Release
PR gets its required checks only when it records the planned version and is
built on the validated SHA (or an ancestor of it). It is configured with
`skip-github-release`: it never creates tags or releases. While the Release
PR is open, `status` (and `status --tag vX.Y.Z`, which considers only the
Release PR of exactly that version) re-validates the current `main`: an
inconsistent commit
blocks it, a releasable commit merged after it was built reports
`refresh_release_pr` (`$axiom-release` re-validates and refreshes it), and
validated hidden commits only need the branch update that the ruleset
requires before merge.

Merging the Release PR is a normal reviewed squash merge. It records the
versioned state on `main` (that squash commit is the *release commit*) and
nothing else. Until that stable version is published, no new release can
start: the preflight refuses it. Curated dated entries in `CHANGELOG.md` continue as before; the
Release PR inserts the version heading above the entries it releases, and only
the Release PR edits those version headings.

### Preparation and publication

Publication is split in two phases with human authority between them:

```text
PREPARE  release-artifacts.yml: preflight -> build -> verify -> notes -> retained workflow artifact
         release.sh: re-verify that artifact at the revision -> publication envelope + preview_digest
ACCEPT   stable only: release-candidate acceptance on those exact prepared bytes (manual until automated)
AUTHORITY  a maintainer authorizes that exact preview_digest
PUBLISH  publish-release.yml: same artifact -> re-verify -> envelope == authorized digest
         -> draft -> upload -> read-back -> publish -> read-back -> delivered Issues (stable)
```

1. [`release-artifacts.yml`](.github/workflows/release-artifacts.yml), dispatched
   from `main` with the tag and full revision, checks them with
   `release-preflight.sh`, builds the set with `build-release-archives.sh` from a
   clean checkout, verifies it with `verify-release-artifacts.sh` (closed
   artifact set, `SHA256SUMS`, provenance, exact revision), renders the release
   notes and retains exactly those files as the workflow artifact
   `axiom-release-<tag>`. It has a read-only token and publishes nothing.
2. `scripts/release.sh` downloads that artifact, re-verifies it with
   [`verify-prepared-release.sh`](scripts/verify-prepared-release.sh) in a clean
   checkout of the revision, and prints the **publication envelope**: repository,
   tag, version, channel, prerelease, revision, `make_latest`, prepared run,
   release-notes and `SHA256SUMS` digests, each artifact with its SHA-256, the
   relevant remote state and the external effects. Its SHA-256 is the
   `preview_digest`.
3. A maintainer authorizes that `preview_digest`.
4. [`publish-release.yml`](.github/workflows/publish-release.yml), dispatched
   from `main` with the tag, revision, prepared run and authorized digest, waits
   for approval of the protected `release` environment, downloads the same
   artifact, re-verifies it, and recomputes the envelope from those bytes and the
   current remote state. If anything differs (an artifact, a digest, the notes,
   the revision, the `latest` decision, the remote state), it stops with
   `preview changed; review and authorize again` before any effect. Otherwise it
   creates or reconciles one **draft** bound to the revision, uploads exactly the
   envelope's files, reads back each digest, publishes once and reads back the
   release, the tag and the `latest` pointer. Nothing is rebuilt after
   authorization.
5. For a stable release, the same envelope also lists the
   [delivered Issues](#delivery-tracking), each Issue's remote state
   (`delivery_issue.N`) and its exact effects (`effect.issue.N`: comment,
   Project status, close). They run only after the release reads back
   published, and only if the Issue state still equals the authorized one. A
   configured Project that cannot be resolved stops the run before the draft.
   If an Issue effect fails after publication, the remote state has changed:
   the next envelope lists only the remaining effects and needs a new
   authorization. A release candidate's envelope says
   `delivery_issues=not_applicable`.

**Release-candidate acceptance.** A stable release is published only from a
prepared set that passed release-candidate acceptance on those exact bytes
([Specification 004 FR-068–FR-076](docs/specifications/004-mvp-v1-baseline/spec.md#release-candidate-acceptance-and-supported-upgrade-sources),
[ADR-0019](docs/decisions/0019-release-candidate-acceptance-prepared-bytes-generation-upgrade-sources.md)).
That acceptance covers:
- fresh install, first run, a representative Project and workflow;
- upgrade from N, from each persisted-generation baseline and from declared
  historical formats, with non-empty state;
- reinstall no-op and downgrade refusal.

It runs on every release row. For Windows it runs on the bounded Windows Server
proxy, which covers identity, provenance, direct `axiom.exe` behaviour and the
installers' fail-closed Server refusal. The proxy claims no Windows client
install, upgrade or reinstall. Until automated acceptance exists, the maintainer performs it
manually on the downloaded prepared set and records Evidence bound to its
exact digests, before authorizing the envelope. Acceptance Evidence is not
publication authority.

For the native macOS arm64 / Linux rows, Slice 3 provides the shared upgrade
journey harness on a complete downloaded prepared set:

```bash
./scripts/test-upgrade-journeys.sh --prepared-set /absolute/prepared \
  --tag vVERSION --revision FULL_SOURCE_SHA --row HOST_ROW \
  --sha256sums-sha256 EXPECTED_SHA256SUMS_DIGEST \
  --previous /absolute/published-N --previous /absolute/published-baseline \
  --poc-binary /absolute/poc-lingo --evidence /absolute/evidence/prepared.json
```

Use physical absolute input paths; symlink ancestors are rejected.
Obtain the expected identity and checksum-file digest from the preparation
record, independently of the downloaded directory. The harness snapshots the
closed four-row set into private temporary storage, verifies its digests,
metadata, manifests, embedded provenance and exact clean local source revision
before executing the selected native row, then rechecks materialized bytes
before emitting `axiom-gate-evidence/v1` with `subject.kind = prepared`.
The local repository must contain that source commit; no build or network
request occurs. The complete set is bound in `subject.artifacts`; unavailable
provider artifact id/digest remain `null`. Invocation placeholders
`{subject-row}`, `{subject-tag}`, `{subject-revision}` and
`{subject-sha256sums}` refer to those document fields. Prepared input, earlier
releases and fixtures are read-only inputs; extraction and installs use only
temporary homes. `--candidate` retains rebuilt PR regression behavior.
This supplies upgrade Evidence only: release-boundary acceptance wiring and
publication-envelope binding remain Slice 4; Windows proxy acceptance remains
a separate slice.

A stable release must be published from its release commit. A release
candidate may be published from any `main` revision that does not record a
newer version (typically `main` before the Release PR, or the release commit
itself). Reruns converge: a consistent published release is a no-op, and any
duplicate, foreign asset, moved tag or inconsistent release fails closed
without changes. Every draft write carries the full release identity (tag,
name, revision, channel) and is checked on its response and on an independent
read-back before publication. A release that belongs to the candidate but is
not bound to its tag (for example `untagged-*`) is reported as
`publication_state=orphan_conflict`: `status` blocks, no envelope is offered
and nothing is created until a recorded human decision resolves it. Because an interrupted publication changes the remote state
(for example to a partial draft), completing it needs a new envelope and a new
authorization.

### Authority

| Action | Who |
|---|---|
| merge a feature/fix PR | maintainer with merge authority; it starts no release |
| start a release (preflight, then dispatch Release Please) | maintainer, by invoking `$axiom-release` |
| review, approve and merge the Release PR | maintainer; merging does not publish |
| prepare and verify an artifact set (no publication) | maintainer or agent |
| authorize the exact publication envelope and dispatch it | maintainer, explicitly |
| approve the `release` environment | maintainer, in GitHub |
| repository settings, rulesets, environments | repository administrator |
| migrating Project #5, its credential, environments and enabling `.github/delivery-project.json` | repository administrator; the file through a reviewed PR |

Green CI, a merged Release PR, a PR comment, file content or an earlier
approval never authorizes a publication. Agents may prepare, verify and report, and dispatch publication
only after an explicit human authorization of the exact envelope digest; they never
approve the environment, create tags or edit releases by hand.

### `$axiom-release`

The maintainer skill [`axiom-release`](.agents/skills/axiom-release/SKILL.md)
conducts this flow for humans and agents. Codex invokes it as `$axiom-release`
and Claude Code as `/axiom-release` (same arguments); both load the same
canonical `SKILL.md`
([ADR-0014](docs/decisions/0014-canonical-maintainer-skills-runtime-discovery.md)):

```text
$axiom-release              # discover the state and do the next step (also: continue)
$axiom-release v0.2.0-rc.1  # conduct that release candidate
$axiom-release v0.2.0       # conduct that stable release
```

It runs [`scripts/release.sh`](scripts/release.sh) as a state machine
(`status` prints `state` and `next_action`; `start`, `prepare`, `publish`,
`verify`):

```text
state=no_release_in_progress         -> preflight -> start (Release PR) -> STOP
state=release_pr_open                -> report the Release PR -> STOP (human merge)
state=release_pr_merged              -> prepare -> publication envelope -> STOP (human authority)
state=awaiting_publication_authority -> after an explicit yes: publish -> verify
state=published                      -> verify
```

Each run rediscovers the release in progress (Release PRs, in-flight runs, the
newest verified prepared run), so re-running it in any intermediate state
reports the same state and never duplicates a dispatch; the maintainer never
supplies SHAs, run ids or intermediate digests, only the authorization of the
`preview_digest` they reviewed. Detailed commands, settings and
Evidence are in the [command reference](docs/commands.md#release-flow) and
[repository security](docs/security/repository-security.md#release-and-branch-protection).

### Merge versus release

| | Merge to `main` | Release |
|---|---|---|
| Trigger | squash merge of a reviewed PR | `$axiom-release`: preflight, Release PR merge, prepared set, authorized envelope, dispatch of `publish-release.yml` |
| Effects | commit on `main`; CI; delivery projection (`Awaiting Release`); no versioning | Release PR; then tag, GitHub Release, assets; delivered Issues released and closed (stable only) |
| Authority | PR approval and required CI | authority over the exact envelope digest and `release` environment approval |
| Reversible | by a new PR | never silently: tags and published releases are immutable |

## AI-assisted contributions

Codex and Claude Code are both supported maintainer runtimes. They follow the
same canonical policy ([AGENTS.md](AGENTS.md)) and the same maintainer skills
in `.agents/skills/`, which Claude Code discovers through the
`.claude/skills/` symlinks; see the [agent harness](docs/agent-harness.md#maintainer-runtimes-and-canonical-skills).
Do not add other Claude- or Codex-specific configuration; validators reject it.

AI-assisted contributions are allowed. The human contributor remains responsible for every submission and must verify code, licenses, sources, and claims. Prompts or raw model outputs do not replace Evidence. Do not submit private content or material whose licensing does not permit its inclusion.
