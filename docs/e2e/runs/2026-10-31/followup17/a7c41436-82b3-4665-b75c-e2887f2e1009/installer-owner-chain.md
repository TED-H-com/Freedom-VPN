# Installer-owned marker blocks the unmodified enforcement verifier

TEST ONLY live read-back, 2026-10-31T08:13:00.997337+00:00.
No table or chain was changed to obtain a serving result.

`docker exec fvpn-e2e-installer-node nft -j list chain inet fvpn owner`:

```json
{
  "chains": [
    {
      "family": "inet",
      "table": "fvpn",
      "name": "owner",
      "handle": 1,
      "comment": "managed-by=fvpn-node-install"
    }
  ],
  "rule_count": 0
}
```

Actual agent journal:

```text
time=2026-10-31T08:12:47.107Z level=WARN msg=apply err=HOOK_PRIORITY_MISMATCH
time=2026-10-31T08:12:52.102Z level=WARN msg=apply err=HOOK_PRIORITY_MISMATCH
time=2026-10-31T08:12:57.113Z level=WARN msg=apply err=HOOK_PRIORITY_MISMATCH
```

Static correlation on this source: `packaging/installer/isolation.go`
`ensureTable` creates this empty non-hook `owner` chain.
`node/enforce/verify.go` `verifyChainSet` requires exactly four chains
(`forward`, `input`, `output`, `postrouting`), and rejects the fifth chain
with `HOOK_PRIORITY_MISMATCH`. The agent preserves the installer marker.
This is an installer/enforcement integration blocker outside this lane's
owned paths. It prevents the real serving member prerequisite and thus
nonzero same-slot usage/restored-traffic proof; it is not a reason to
remove the marker or weaken the check in the harness.
