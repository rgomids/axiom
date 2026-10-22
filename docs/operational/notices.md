# Notices and limits

- Axiom supports macOS arm64 and Linux amd64/arm64 for published installation.
  Other host rows are refused. Exact validated environments remain release
  evidence, not a guarantee for every filesystem or host configuration.
- Checksums establish observed byte integrity against the selected release
  metadata; they do not independently authenticate the publisher.
- The installed `LICENSE` is part of the Axiom release bundle. Preserve it with
  the binary and consult that exact file for the distribution's license terms;
  this guide does not reproduce or redefine those terms.
- The installed corpus is an explicit, bounded set of operational pages and a
  generated CLI reference. It excludes Specifications, ADRs, Evidence,
  maintainer instructions and skills, development-only docs, personal paths,
  secrets and private Provider content.
- Documentation discovery describes verified installed documentation binding
  only. Paths are locators and consumers must revalidate selected content. It
  does not prove which man page a reader selected or which skill a Runtime
  loaded.
- `man axiom` selection is session-only. Keep `axiom help` and this Markdown
  guide available when no reader exists or the selected page cannot be verified.
- A successful configuration, login observation, preview, plan or dispatch
  acknowledgement does not certify inference, billing, Provider outcome or
  human acceptance.
- Axiom does not automatically roll back confirmed changes, downgrade an owned
  installation or retry an uncertain operation. Use the explicit diagnostic,
  preview and recovery protocol for the owning command.
- Examples use placeholders such as `<slug-or-id>`, `<plan-digest>` and
  `$verified_man_root`; replace them with values obtained from the current
  command/result. Do not treat placeholder text as a literal identifier.

Continue to [Troubleshooting](troubleshooting.md) or return to the
[operational guide](index.md).
