# ADR-0019 — Windows default onboarding and consent-gated permission repair

## Status

Accepted for the requested implementation, 2026-10-08: the requester asked for
working default Windows onboarding and explicitly selected diagnosis followed
by authorization to repair only necessary directories. Merge, human acceptance
of executable Evidence and release publication remain separate.

The requester separately authorized automatic persistent user PATH setup.

## Context

Issue #265 reproduced default storage refusals at AppData ancestors and standard
Runtime skill roots. Isolated roots can install a binary but cannot provide
normal Runtime discovery. A command that reports binary installation without
checking Runtime integration does not establish usable onboarding.

## Decision

Fresh Windows installations use private profile storage at
`%USERPROFILE%\.axiom\windows\{bin,install,state}`. Existing default executable
or receipt locations under LocalAppData are retained; existing LocalAppData
state is also retained. Explicit destinations and state overrides remain
authoritative. There is no automatic migration, ACL change on AppData, or
redirection around an unsafe explicit/legacy location.

Before binary publication, the verified native installer inspects required
Runtime roots. An explicit repair exception is available only for current-owned
standard `.agents`, `.agents/skills`, `.claude` and `.claude/skills` directories.
Parents receive replacement protection; skill roots receive private access.
The operation only subtracts rejected permissions from untrusted allow ACEs,
retains trusted/deny ACEs and owner/group, and protects the resulting DACL.
Nonstandard overrides and unsafe ancestors outside that allowlist are refused.

Every repair requires an exact preview digest, owner/object identity and ACL
revalidation, durable private backup before effects, and postcondition checks.
No consent, no repair. Updates operate on pinned handles without descendant
propagation. Failure attempts rollback; the saved backup remains available for
explicit recovery. Recovery refuses replaced objects or later ACL changes.
Windows may clear `DACL_AUTO_INHERITED` bookkeeping on a direct update; recovery
preserves ACEs, including inherited flags, and DACL protection, rather than
claiming bit-for-bit preservation of that bookkeeping flag.

The native handle operation uses `NtSetSecurityObject`; `SetSecurityInfo`
normalizes inherited ACEs and can propagate to unrelated children. Existing
NTFS handle/object-identity confinement remains in force. Native tests must
verify no propagation, backup failure, denied authority and recovery.

The online bootstrap verifies the installed executable, runs `first-run` and
adds the selected directory to current process and persistent user PATH only after successful
onboarding. Existing Runtime skill conflicts remain preserved and are reported
as setup failures. The requester explicitly authorized persistent user PATH:
preserve raw entries and registry type, deduplicate and provide `-SessionOnly`.
System PATH, Runtime installation and credentials remain
outside installer authority. `-SkipRuntimeSetup` supports explicit binary-only
validation and must not report full onboarding success.

## Alternatives considered

1. Documentation-only isolated roots: useful diagnostics, but no default Runtime
   discovery or usable installation contract.
2. Automatically rewrite existing ACLs without consent: rejected by the
   requester; could affect other tools and sandbox access.
3. Trust additional sandbox/application principals or disable checks: rejected;
   would weaken the accepted Windows filesystem boundary.
4. Consent-gated bounded repair plus private fresh storage: selected; adds a
   prompt on affected profiles while preserving ordinary fail-closed behavior.

## Consequences

Safe fresh profiles need no manual root overrides. Affected Runtime roots need
one reviewable consent decision. Existing child skills and AppData/profile ACLs
are preserved. Legacy installations, foreign/modified skills, nonstandard roots,
policy-denied writes or locked directories can still require operator action;
universal success on arbitrary Windows machines is not a supported guarantee.

The bootstrap and native release must be published together: an older release
does not gain the new root resolution or repair behavior merely because the
bootstrap source changed. Existing release artifacts remain immutable.

## Revisit when

Supporting nonstandard Runtime roots, migrating legacy storage, extending the
repair allowlist, modifying existing Axiom child artifacts, or changing system
PATH becomes an approved requirement.
