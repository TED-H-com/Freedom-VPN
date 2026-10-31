**Offline installation:** initial and same-release installs retain both the verified
envelope and `/var/lib/vpn-node/update-staging/<artifact-sha>.tar.zst` before
invoking seed. The installer copies the supplied archive through a private mode
0600 file, rechecks it against the signed manifest, then renames and syncs it
([install.go](../../packaging/installer/install.go),
[archive.go](../../packaging/installer/archive.go)). Upgrade also stages the
verified archive. With that cache present, seed verifies a private copy and the
whole installed tree without fetching the artifact; this describes seed's
offline capability, not offline enrollment.

**Missing or invalid retained archive:** [seed.go](../../node/updater/seed.go)
downloads the manifest's signed artifact URL only when the cache is absent,
using the URL guards, TLS trust, bounded retries and signed size/hash checks.
Updates 11's [uncached probe](../e2e/at/updates/artifact-origin.json) measured
this fallback with unchanged generated config; it added only a verified
`seeded_baseline`. If the origin is unreachable after retries, seed exits 1
with `updater: STORE_UNAVAILABLE: download` (an exhausted attempt timeout is
`updater: DOWNLOAD_TIMEOUT: attempt`). An existing cache that fails verification
does **not** fall back to downloading: wrong hash is
`updater: HASH_MISMATCH: staged artifact`, wrong size is
`updater: SIZE_MISMATCH: staged artifact`, and an unreadable/non-regular cache is
`updater: ARTIFACT_INVALID: staged artifact is unreadable`. These failures leave
the current pointer and existing baseline evidence intact. Restore the original
signed package and rerun the installer for the same release to reverify and
repair the retained archive before seed, or restore access/trust to the signed
URL for the absent-cache fallback and rerun seed. Never substitute the live tree
or a locally computed hash for the signed artifact. No unverified download is
accepted, and a corrupt retained archive is never silently replaced by seed.
See the [follow-up 19 acceptance record](node-followup19-2026-10-31.md) for F11-02.

The verified release, current pointer, state-schema marker and baseline evidence commit in one SQLite transaction, with no schema migration. `status` exposes `seeded_baseline` for the current release: `release_id`, `manifest_sha256`, `tree_hash`, `boot_id`, `seeded_at`. A baseline is installation evidence; it does not assert health, create an update job/result, promote `last_good`, or change publication/forwarding authorization. Seed performs no switch, service restart or rollback.

Seed exit 0 means verified and recorded. The same canonical envelope and tree are re-verified on replay, preserving the original seed timestamp and boot id, with no second release or baseline row. An established journal with a different current release, manifest, tree or state schema is refused; open work is refused too. Exit 1 reports the reason and leaves existing journal evidence and the current pointer intact. Usage/unknown commands exit 64; `seed -h`/`--help` print `-manifest` and `-tree` help with exit 64. The installer recognizes an older unsupported updater only from exit 64 with `unknown command "seed"`; other probe/seed failures become installer exit 1 and check drift. Probe and seed exits/stderr are retained in `install/baseline.json` and `check --json`.

#