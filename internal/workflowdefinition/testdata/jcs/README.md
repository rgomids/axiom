# RFC 8785 conformance vectors

Source: https://github.com/cyberphone/json-canonicalization/tree/19d51d7fe467/testdata

Copied byte-for-byte from the pinned module's `testdata/input` and `testdata/output`.
The upstream Apache-2.0 license is retained in `LICENSE`. These six vectors test
the general canonicalizer, including numeric serialization, nested structures,
Unicode and UTF-16 key sorting. Axiom's definition schema separately refuses
floats/null and applies its own bounds and semantic validation.
