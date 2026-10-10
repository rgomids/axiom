# Historical Runtime skill bytes

`v0.6.0` contains exactly the six operation-specific skills from the published
`v0.6.0` tag. `v0.10.0` contains exactly the eight skills from the published
`v0.10.0` tag. Each SKILL.md was compared byte for byte with:

```sh
git show <tag>:internal/codexruntime/skills/<skill>/SKILL.md
```

These fixtures establish historical ownership and migration behavior. They are
not embedded, distributed, installed, or exposed as Runtime entrypoints.
Published tags were not modified. Current distribution contains only
`axiom-project`, `axiom-work-item` and `axiom-workflow`.


`v0.15.0` contains the two skills extracted from its verified official
`axiom-0.15.0-macos-27-arm64.tar.gz` asset, SHA-256
`3ca48fb132a9163e03a42bbd4cc617254833a66666ec98ef6765ffad4b2e4e6f`.
Each file matches its tag bytes. Reproduced historical receipt provenance:
[../published-receipts/v0.15.0/PROVENANCE.md](../published-receipts/v0.15.0/PROVENANCE.md).
