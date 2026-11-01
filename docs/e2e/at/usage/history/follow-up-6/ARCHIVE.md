# Follow-up 6 archive (D-098)

Run `71b1a160` passed AT14–AT20 on 2026-10-31. Its 19 root evidence
files are copied byte-identically from baseline `de6d2e0246892b5744f68864c3d6d2854945c40a` into
[71b1a160/](71b1a160/). [SHA256SUMS](71b1a160/SHA256SUMS) pins every copied file.
The run used harness/image `7ccaabade8efc5074cfde5dcae84c6b4a27fd8c0` and
product baseline `fb5e002549f79e8ef61bebcf42cce1678ab65cb1`.

Verify from the repository root:

```bash
(cd docs/e2e/at/usage/history/follow-up-6/71b1a160 && sha256sum -c SHA256SUMS)
```

The interrupted [68542eb7](68542eb7/README.md) attempt and measured
[cb434530](cb434530/README.md) diagnostic remain unchanged beside the final run.
The archive preserves original text and relative links; follow references in
the original repository context. New evidence belongs in the living root.

Handover claims already read immutable snapshots, including their own pinned
`71b1a160` copies. The living AT20 path occurs only in D-109 negative checks;
no handover reader or pin needs moving for this archive.
