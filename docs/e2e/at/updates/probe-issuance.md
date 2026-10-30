# D-063 / D-064 probe issuance and traversal — follow-up 8

Run `71f2aa37-b0b9-4568-abe0-2ba7665377e4`, source `bc93ce430ebe8a50e5dfdaf4121352047fe89249` (on `main` `52dc77026cb168df5684ae2ccd5cbcadb4dbaeff`), evidence `2026-10-30T20:59:15Z`. Prefix `w4-updates`, port base 42000.

Raw evidence: [probe-issuance.json](probe-issuance.json), [AT23.json](AT23.json), [AT25.json](AT25.json).

## Exported probe peer setup (#204)

After `node` (`01a18e94-1ede-7da3-b7c1-ae4f674321c0`, incarnation `01a18e94-2123-7ae8-9820-7a80fca3fe42`) reaches its ready unpublished baseline, `TestUpdatesFollowup` calls `stack.SetupUpdaterProbe(ctx, "node", EnrolledNode{NodeID, IncarnationID})` (exported in PR #204, closing F5-02):

- `probe_not_member`: `true` (`member_devices` count for the setup-key peer is `0`).
- `probe_route_selected`: `true` (dedicated route `192.0.7.10/32` via `node` selected on `fvnb0` inside `fvpn-w4-updates-node-probe`).
- `probe_config_scope`: `"one dedicated setup-key peer /32; one isolated origin /32 UDP 4443; no member device or grant"`.
- `updater.json` receives `agent_config: "/etc/vpn-node/agent-run.json"` and `probe_peer: {"destinations":[{"addr_v4":"192.0.7.10/32","port":4443,"proto":"udp"}],"peer_id":"[redacted]","source_v4":"100.109.39.46/32"}`.
- The raw `peer_id` is compared in memory against the live challenge's `probe` object (`sameJSON(ch["probe"], rawProbePeer)`) and redacted to `"[redacted]"` before writing any public evidence file.
- Challenge and attestation files inside `/var/lib/vpn-node/update-journal` (`0700`) are observed through a read-only `docker exec` stream inside the node container; journal permissions and ownership are never modified.

## AT23 — candidate health check with D-063 issuance and D-064 traversal (`pass`)

During the `1.0.1` (`01a18e94-0630-7535-a302-8ebf65284d54`) update transaction `01a18e95-5453-70ca-bf1d-ab4a17ab0bc5`:

- **In-node challenge stream**:
  - `has_probe_peer`: `true`
  - `agent_config_exists`: `true` (`/etc/vpn-node/agent-run.json`)
  - `configured_probe_issued`: `true` (fresh challenge `nonce: "cf7cf54c..."` carries the exact configured `probe` object)
  - `omission_logs`: `[]` (`0` `"health challenge probe omitted"` diagnostics)
  - `candidate_attestations`: `1` fresh attestation for `txn_id: "01a18e95-5453-70ca-bf1d-ab4a17ab0bc5"` with `internet_only=passed`, `dns_policy=passed`, `tunnel_probe=passed` (`overlay_up`).
- **Exported observer (`UpdaterProbe.Observe(ctx)`)**:
  - `verified`: `true`
  - `probe_traffic`: `{"attempts":644,"authenticated_udp_replies":14}`
  - `probe_window`: `{"samples":539,"grant_samples":4,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1078,"probe_down_bytes":1078,"member_bytes_before":0,"member_bytes_after":0,"elements_after":0}`
  - `probe_origin_receipt`: `{"probe_id":"b5500e72-5d00-48ab-a8f9-5a64ababba56","requests":14,"source_address":"192.0.7.20"}`

## AT25 — byte-tampered archive rejected at preflight before health check (`pass`)

During the byte-tampered `1.0.2` (`01a18e94-0df0-7894-9874-5ef3b6736565`) batch `01a18e95-f57c-7be0-a91b-44499ca0ce8e`:

- `has_probe_peer`: `true`, `agent_config_exists`: `true`, `omission_logs`: `[]`.
- `fvpn-updater` rejects the tampered archive at preflight with `HASH_MISMATCH` (`stage=""`) before creating a transaction, installing a candidate executable, or entering `stageCheck`.
- `fresh_challenges`: `[]` (`0` challenges issued), `configured_probe_issued`: `false`.
- `probe_observer`: `verified=false`, `grant_samples=0`, `source_elements=0`, `destination_elements=0`, `authenticated_udp_replies=0`, `requests=0`, `elements_after=0`.
- **Conclusion**: Probe scope applies only to candidate/rollback health checks (`stageCheck`); it does not apply to an archive that never passes preflight, and the observer confirms the D-064 exception set remained empty throughout preflight rejection.
