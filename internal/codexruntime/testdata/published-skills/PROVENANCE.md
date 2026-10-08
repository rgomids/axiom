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
`axiom-project` and `axiom-work-item`.
