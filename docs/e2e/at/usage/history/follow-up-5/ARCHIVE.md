# Follow-up 5 archive (D-098)

2026-10-31: root records from follow-up 5 (#324), run `170ea3e3`, are preserved
byte-for-byte in [170ea3e3](170ea3e3/). The retained diagnostic runs
[eba86d44](eba86d44/README.md), [64865721](64865721/README.md) and
[014daad8](014daad8/README.md) are unchanged; their failures remain failures.
[SHA256SUMS](SHA256SUMS) indexes every file in all four snapshots.

Verify from this directory with `sha256sum -c SHA256SUMS`.
The handover claims already read immutable `tests/handover/testdata/170ea3e3-AT*.json`
and `351d59aa-AT*.json` fixtures. No pins moved and no handover test changed.
The living-path AT20 string in its D-109 test is a negative source-policy test,
not a reader of the root AT20 record.
