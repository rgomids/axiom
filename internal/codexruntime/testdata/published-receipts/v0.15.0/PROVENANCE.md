# v0.15.0 receipt provenance

Tag: `v0.15.0`; commit: `37662730c64e077d615f110744ea86d2e897c890`.
Official asset: `axiom-0.15.0-macos-27-arm64.tar.gz` from
https://github.com/rgomids/axiom/releases/tag/v0.15.0.
Asset SHA-256: `3ca48fb132a9163e03a42bbd4cc617254833a66666ec98ef6765ffad4b2e4e6f`.
Official SHA256SUMS SHA-256:
`66e3a69aad3c0d9cf7c6c26cd15321e3ff3b1e2555e98dcab9d9e90883c4ccee`.
Both downloaded read-only through `gh release download v0.15.0`.
The archive digest matches both official SHA256SUMS and GitHub asset digest.
Archive paths/types validated before extraction; skill bytes compare exactly
with `git show v0.15.0:internal/codexruntime/skills/<name>/SKILL.md`.

Fixtures reproduced by the verified archive's own binary, with isolated empty
HOME, `AXIOM_CODEX_SKILLS_ROOT` and `CLAUDE_CONFIG_DIR`, using only
`axiom --json runtime codex install` and `axiom --json runtime claude install`.
Its `--json version` reports version `0.15.0`, revision `37662730c64e`, source
state `clean`. No vendor process or authentication probe invoked.
The written Codex receipt is copied unchanged; the Claude receipt changes only
the temporary `skillsRoot` value to the established `{{ROOT}}` fixture token.
No receipt or digest was fabricated. Historical fixtures remain immutable.
