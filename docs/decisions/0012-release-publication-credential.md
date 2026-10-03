# ADR-0012 — Environment-scoped release publication credential

## Status

Proposed for human review on 2026-10-03. The maintainer configured the dedicated
secret and requested workflow integration. Merge accepts the repository change;
publication and approval of the `release` environment remain separate gates.

## Context

The v0.3.0 recovery preserves original source revision
`b79d3bf8cbf21247ca30cae06ff000e7f89a5adf` under ADR-0010. Publication run
[37123636525](https://github.com/rgomids/axiom/actions/runs/37123636525) passed
prepared-set verification and envelope authorization, then failed creating the
draft with HTTP 403. The source differs from main in workflow files; the job's
`GITHUB_TOKEN` had Contents write but cannot receive Workflows write required
by GitHub's [release API](https://docs.github.com/en/rest/releases/releases#create-a-release).

## Decision

Use an expiring, repository-scoped fine-grained PAT stored as
`AXIOM_RELEASE_PUBLISH_TOKEN` in the protected `release` environment. Bind it
only to `GH_TOKEN` in the publication script step, after the human environment
gate. Missing secret refuses before scripts run, with no token fallback.

The PAT needs Contents write and Workflows write for release publication
and Actions read for prepared-run verification. As amended by ADR-0013, Issue
records and PR labels use the job's `GITHUB_TOKEN` with Issues write and Pull
requests write, preserving the trusted `github-actions[bot]` record author.
Preflight, artifact download and checkout retain read-only permissions;
checkouts never persist credentials.
The Project token retains its independent GraphQL-only role.

This changes authentication, not publication authority. Source revision,
recovery pins, prepared bytes, digest matching, immutable releases, tag
protection, human environment approval and delivery effects remain unchanged.
Operational configuration lives in [repository security](../security/repository-security.md#publication-credential).

## Alternatives considered

- Keep `GITHUB_TOKEN`: insufficient for the original historical workflow files.
- Change source revision or create a tag manually: violates original source
  provenance or bypasses the guarded workflow.
- Dedicated GitHub App installation token: viable and preferable for long-lived
  automation, but requires new App lifecycle and installation management.
- Expiring fine-grained PAT: bounded integration using the maintainer-configured
  environment secret; requires manual renewal and remains tied to its owner.

## Consequences

Publication can authenticate for the preserved source without rebuilding or
changing recovery inputs. The credential can modify workflows within this
repository, so it is available only to reviewed publication code behind the
protected environment; code review and that human gate remain required.
Maintainers must renew/revoke the token and keep its repository scope narrow.
Local tests verify binding and missing-secret refusal, not the remote token's
permissions or successful publication. A failed run requires fresh status and
publication authorization before retry, even if its digest is unchanged.

## Revisit when

Adopt a dedicated GitHub App if automated rotation, multiple maintainers or
long-lived service identity becomes necessary. Revisit if GitHub changes the
release API permission requirement or the publication script's API inventory.
