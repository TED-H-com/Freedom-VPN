# Post-upgrade re-enrollment diagnostic

TEST ONLY read-only `fvpn-node-install check --json` after the failed
`installer-reenroll` row in this run. Captured 2026-10-31T10:52:29.126858+00:00

No seed retry, tree repair or metadata edit was performed.
Owner: installer / updater (`packaging/installer/baseline.go` and updater seed).

```json
{
  "exit": 2,
  "baseline": {
    "status": "failed",
    "manifest_path": "/var/lib/vpn-node/update-staging/01a19192-b8f2-7424-864b-9b781677ba18.manifest.json",
    "tree_path": "/opt/vpn-node/releases/01a19192-b8f2-7424-864b-9b781677ba18",
    "probe": {
      "exit_code": 64,
      "stderr": "Usage of seed:\n  -config string\n    \tupdater config JSON (paths only, no keys)\n  -manifest string\n    \tretained signed release envelope\n  -pending string\n    \tinstaller pending-release.json to apply\n  -tree string\n    \tresolved current release directory to verify\n"
    },
    "seed": {
      "exit_code": 1,
      "stderr": "fvpn-updater: updater: HASH_MISMATCH: seed tree differs from the signed artifact\n"
    }
  }
}
```
