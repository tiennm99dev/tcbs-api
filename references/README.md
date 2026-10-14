# References

These files are TCBS's API specifications, kept here so the clients can be
built and checked against them. They belong to Techcom Securities (TCBS). They
are not covered by this repository's Apache-2.0 license, and they will be
removed if TCBS asks.

| File | Source | Added | Changes |
|---|---|---|---|
| `tcbs-openapi-v1.0.0.json` | TCBS developer portal, `https://developers.tcbs.com.vn/specs/openapi-v1.0.0.json` | 2026-10-14 | none |
| `openapi.json` | Earlier edition of the iFlash Open API spec published by TCBS (its contact block points to `https://developers.tcbs.com.vn`); the hand-written `tcbs` package follows it | 2026-03-18 | line endings normalized |
| `tcbs-operation-ids.json` | Written for this repository: maps each `METHOD /path` of the portal spec to an operation ID named after its portal docs page | 2026-10-14 | n/a (Apache-2.0) |

`tcbsgen/openapi.json` is generated from `tcbs-openapi-v1.0.0.json` by
`tools/specfix` and carries the same terms as its source.

See [docs/spec-and-codegen.md](../docs/spec-and-codegen.md) for how the
specs differ and how to refresh them.
