# Remaining operator runs

Every procedure below is **needs operator run**. It has not passed. Complete
SECOPS-09 and prove counted member traffic before attempting outage assertions. Use the shared `deploy/devstack`
implementation with an orchestrator-assigned instance; do not fork it. On this
branch the only assigned instance is `w4-secops`, base `30000`. The automated logical restore uses a separate network-none container; no second
networked restore slot or production hardware is allocated. Record UTC start/end, source commit,
binary and image digests, OS/kernel/app builds, policy revision, incarnation,
commands, expected/actual, and verdict for each row. All test rates are TEST ONLY.

Protect generated cookies, tokens, curl configs, CA material, backup data and raw
logs in `.agent` with mode 0600. No secrets belong in command-line arguments or
published evidence. For this extended manual matrix, prepare protected
curl configs for `admin`, `member-a`, `member-b`, `node-a`, `node-b`, `expired` and
`revoked`, plus valid request bodies/IDs. The automated Go probe creates in-memory sessions/certificates directly from
the existing IdP and enrollment hooks. It does not export these operator curl
files; prepare them privately for the additional manual steps.

## AT28

1. Export fixture IDs and protected curl config paths under
   `.agent/secops/operator/`; configs include CA, session cookie or node client
   key paths, CSRF and Origin where appropriate. Set `PUBLIC_URL` and `AGENT_URL`
   to the assigned stack listeners. Keep all transport inside the controlled lab.
2. Enumerate every operation from `contracts/openapi.yaml`. For each admin
   resource family (nodes, enrollments, members/grants, policies/revisions,
   releases, batches/targets, operations, audit), issue the same valid request as
   admin (positive control), both members and node identities (403/401). Record
   request ID, status and error code only. For member self, devices, exits,
   usage, checkout, billing portal and operations, attempt member B's IDs/query/
   body fields using A's session. Expect schema refusal or A's own data; device
   and operation lookups must not disclose B's resources (404). Also check A's
   positive resource before/after every refusal.
3. Examples after the fixture hook exists:

   ```sh
   curl --config .agent/secops/operator/member-a.curl --silent \
     --output .agent/secops/operator/refusal.json \
     --write-out '%{http_code}\n' "$PUBLIC_URL/api/v1/admin/nodes"
   curl --config .agent/secops/operator/node-a.curl --silent \
     --header "X-Node-ID: $NODE_B_ID" \
     --output .agent/secops/operator/desired.json \
     "$AGENT_URL/api/v1/agent/desired-state"
   ```

   Verify desired state's signed node/incarnation remains A. Test every agent
   POST using valid bodies with B's node/incarnation, nested usage event or job
   identity: expect 403/contract refusal and unchanged B state. Repeat with
   `X-Client-Cert`, no client cert, expired and revoked certs. Expired TLS may
   terminate before any HTTP status. Revoked database/journal identities must
   remain refused even when the X.509 validity period is current.
4. OIDC: mutate issuer, audience, signature, expiry, nonce and state one at a
   time; remove MFA for admin. Omit/wrong-value CSRF and Origin on every mutating
   browser command. Test role selection with both cookies. No state change or
   wrong-role session is permitted. Send oversized bodies at configured cap+1,
   malformed UUIDs/decimal strings, unknown fields and duplicate JSON keys.
5. Replay the same command concurrently and after restart with one UUID
   idempotency key: same resource/operation, one effect. Change payload with the
   same key: 409. Test missing If-Match (428) and stale If-Match (412). Enrollment
   retry must not repeat the one-time secret. Rate tests must show 429 and
   Retry-After; one principal must not evict another's allowance.
6. Capture **every** stack service log, node agent/watchdog/updater/NetBird logs,
   access/error logs and all audit pages from run start to end. Inventory stream
   name, UTC bounds, byte count, collection success and truncation; an omitted,
   failed or truncated stream blocks this step. Scan runtime-issued enrollment,
   setup, session, CSRF, provider and signing secrets (including encoded forms)
   against logs/audit and wrong-role responses. Export only finding type and
   location, never matched bytes. Preserve request-ID/audit correlation and
   secret-bearing response `Cache-Control: no-store`. A generic regex scan alone
   is insufficient.

## AT29

1. The automated fixture sets a TEST ONLY node lease ceiling of 90 s through
   fleet configuration input. Once serving works, create source inputs (API
   grants and fake-provider payment/grace events), never projected entitlement
   or device rows. Wait for projections. Use a node lease of 90 s, a paid member service end at +30 s, a
   finite grace member ending at +45 s and a finite grant member ending at +60 s.
   Persist and compare original `paid_through`, grant/grace source deadlines and
   entitlement revisions before/after every renewal. Obtain verified policy and
   lease, and confirm nonzero generation and member accept rules. Require
   **successful** continuous TCP upload/download and UDP echo before the fault.
2. Capture node heartbeat/boot ID and monotonic time, signed server time,
   `effective_service_end`, member/node lease ends and kernel element timeout:

   ```sh
   docker exec fvpn-w4-secops-node nft -j list table inet fvpn
   docker exec fvpn-w4-secops-node cat /run/vpn-node/heartbeat
   docker stop fvpn-w4-secops-control
   docker exec fvpn-w4-secops-node touch /var/lib/vpn-node/e2e/pause-agent
   ```

   The marker makes the existing entrypoint stop and keep the agent paused;
   confirm its process exits. Keep per-packet timestamp/sequence results on the
   controlled sink. Existing and newly attempted flows must stop no later than
   the earliest signed bound, or earlier if the watchdog fails closed for stale
   liveness. Control connectivity must remain permitted by node rules.
3. Re-establish a fresh valid policy and repeat with agent **and watchdog**
   stopped inside the owned node. Only the already armed kernel timeout remains;
   verify established TCP/UDP stop. Record actual timeout set membership at the
   boundary, not just a process death or an empty table before traffic starts.
4. Repeat on the disposable **guest VM console**, never the shared host or a
   Docker container's shared wall clock. The guest operator rolls wall time back
   by one hour while keeping monotonic time/packet collection. Also restart the
   watchdog, replay the exact lease, reboot the guest and perform suspend/resume.
   Expected: no added lifetime; reboot or loss of anchor stays closed until a
   fresh valid lease. Preserve BOOTTIME/monotonic and UTC traces around each
   action. No host `sudo`, `date`, `sysctl`, nft or interface changes are allowed.
5. Restore control and resume the existing entrypoint:

   ```sh
   docker start fvpn-w4-secops-control
   docker exec fvpn-w4-secops-node unlink /var/lib/vpn-node/e2e/pause-agent
   ```

   The second command releases the existing entrypoint pause marker. Do not
   claim recovery until new signed policy/lease generation and applied ACK are observed
   and controlled traffic succeeds. Expired paid/grace/grant members stay closed
   unless a new independently authorized entitlement exists.

## AT30

This run must use a newly allocated isolated restore copy. Do not point any
restore command at `FVPN_TEST_DATABASE_URL`, the live lab or another task's DB.
The runner has passed a real isolated logical dump/replay subset (see
[restore.json](restore.json)). The steps below extend that evidence to WAL/PITR,
original node archives and coordinated external reconciliation; they are unrun.

1. First use the owning `w4-ops-backups` pgBackRest procedure in
   `deploy/backup/` and `docs/ops/backups.md` after it lands; these paths are absent
   on this follow-up checkout. Do not create a parallel physical backup path.
   On the isolated source copy, use a valid node archive retaining at least
   72 hours. Generate original events 1–400, persist exact hashes/totals and
   archive a base backup plus WAL through that commit. Record backup ID,
   manifest digest, system identifier, timeline, start/stop LSN and UTC commit
   watermark. Continue through event 700 and verify central ACK=700 and node
   retention of all 700 original events. Record post-backup payment, manual
   suspension, revoked identity, disabled publication and newer applied release.
2. Freeze ingest and pause ledger/entitlement workers. Stop all external access
   on the restore copy using the shared isolated-stack hook. Restore PostgreSQL
   base backup/WAL to the 400-event point; restore config/secrets/revocation
   journal from their independent sources. Validate schema and actual receipt
   coverage; never infer highest contiguous sequence from `max(event_seq)`.
   Read back database name/system identifier before any write. Restore images
   and recovery config must refer exclusively to this copy.
3. Build the existing CLI and operate only on the protected copy config:

   ```sh
   python3 tests/e2e/at/secops/diskexec.py go build \
     -o .agent/secops/fvpn-usage-admin ./internal/usage/cmd/fvpn-usage-admin
   .agent/secops/fvpn-usage-admin -config .agent/secops/restore/control.json \
     freeze -actor "$RECOVERY_ACTOR_UUID" -reason 'TEST ONLY isolated restore'
   .agent/secops/fvpn-usage-admin -config .agent/secops/restore/control.json \
     begin-recovery -actor "$RECOVERY_ACTOR_UUID" \
     -restored-to "$RESTORED_UTC" -affected-since "$BACKUP_UTC" \
     -reason 'TEST ONLY restore to acknowledged event 400'
   ```

   Verify new ledger epoch and the separate ledger-epoch signature; nodes receive
   only the recovery-purpose signed checkpoint. Old-epoch batch: 409. Frozen
   ingest: 503 and no ACK. Same-epoch lower ACK: refused. New authenticated epoch
   must lower ACK to 400 and request replay 401–700 despite prior ACK=700.
4. Replay original seq/hash/meter epoch/interval bytes from node retention twice.
   Verify exactly 700 durable receipts, original exact member totals and one
   projection per receipt; a same-key changed payload is `conflict` in the 200
   per-event response, not a whole-request 409 (binding contract). Test an
   unavailable retained interval separately: visible gap, no invented bytes.
5. Reconcile the fake provider's **current** invoices/refunds/periods and NetBird
   peer/routes/groups using the existing worker jobs. Restore newer suspension
   and grant evidence independently; payments must not clear suspension. Retry
   each job and compare stable external IDs. Revoked identities remain revoked;
   disabled nodes remain disabled; a newer safe node release is not downgraded
   merely because the DB snapshot is older. Unexplained intents stay fail closed.
6. Complete only after usage, entitlement, identity, publication and update
   evidence agree:

   ```sh
   .agent/secops/fvpn-usage-admin -config .agent/secops/restore/control.json \
     complete-recovery -actor "$RECOVERY_ACTOR_UUID"
   ```

   Resume workers and run isolated login, authorization, DNS, usage, suspension
   and version probes. RPO = incident cutoff minus latest recoverable committed
   evidence; list any remaining permanent losses after replay. RTO = start of
   recovery through all probes and reconciliation complete, not restore-command
   duration. Targets are 15 minutes and 4 hours; report actuals and failed steps.
   Destroy only the allocated copy and verify no objects remain.

## AT31

1. Fix [load-profile.json](load-profile.json) before execution. Use 20 separate
   OIDC subjects, 60 centrally authorized and bound devices, two serving nodes,
   controlled sink/resolvers and verified relay path. Record the dev host as
   **not candidate hardware**; record CPU model/count, architecture, RAM, disk,
   kernel, container limits and background work without private hostnames.
2. Warm up 60 seconds, then measure 300 seconds once. Start 20 sign-ins together;
   each member performs one authenticated portal read/s. Heartbeat every 30 s,
   sampling every 10 s, policy changes at +60 and +180 s. Each device maintains
   TCP up, TCP down and UDP echo. Run direct and forced-relay windows separately;
   NetBird reported relay path plus controlled connection failure of direct
   paths must confirm relay use. Never target a third-party endpoint.
3. Use monotonic request timers; save all status codes (including 429/timeouts),
   p50/p95/p99 and error rate for each route. Measure policy commit to final
   eligible node applied ACK **and actual traffic behavior**; sampling interval
   end to the correct visible member total. Record stopped/overlimit/suspended
   membership probes while load runs. Match every submitted sequence/hash to a
   durable receipt and verify duplicates do not add bytes.
4. Collect each owned container's resource snapshots during the window:

   ```sh
   docker stats --no-stream --format '{{json .}}' \
     fvpn-w4-secops-control fvpn-w4-secops-worker \
     fvpn-w4-secops-postgres fvpn-w4-secops-node fvpn-w4-secops-member
   ```

   Repeat at one-second intervals using the operator collector. Export CPU,
   memory, block IO, network bytes and timestamped sink throughput, plus queue
   depth and errors. Inspect no other task's containers.
5. Evaluate strict read p95 <500 ms, policy <=60 s and usage <=120 s. Report
   actual missed targets, pool/CPU/IO/limiter/relay constraints and their capacity
   consequence. Do not raise limits, reduce work or discard errors after seeing
   the result. Repeat the fixed profile on candidate hardware and physical arm64
   as separate operator evidence; never extrapolate a cross-country Mbps SLA.

## AT32

1. Read README, portal zh-TW/en texts and the member guide against
   [scope-review.json](scope-review.json). For each DOC finding, record corrected
   file/line, owner and the acceptance evidence or explicit limitation. Check
   payment pending/applied, usage freshness, lease and client kill-switch language.
2. On the same release, enroll preinstalled Ubuntu, cloud Ubuntu and a general
   Ubuntu host with installer/preinstalled token delivery. Use the same node
   lifecycle actions and API roles. Record OIDC actor, incarnation, provisioning,
   published/observed state, desired/applied generation and actual release for
   every source. Include one physical arm64 source and the second NAT.
3. Change only notes to each source label and a fictional ownership label.
   Re-run identical role, publishing, serving and all_nodes update checks. No
   capability or extra approval may change because of the notes. Review schema,
   authorization, updater and installer branches alongside these traces; no
   source trust, partner portal, remote shell or maintenance-ticket workflow is
   allowed. An operator checklist alone is not a product workflow defect.
4. On Windows, macOS, iOS and Android, record actual OS/app build, management URL
   setup, SSO, automatic device authorization, exit selection, revoked/removed
   exit, sleep, Wi-Fi/mobile network switch, reconnect and DNS path. Record
   system DNS, external DNS, Private DNS enabled/disabled and DoH separately.
   Confirm text against observed behavior and preserve client fallback caveats.
5. Publish the three-source comparison, four-platform matrix and all resolved
   documentation requests. Any unrun row remains **needs operator run**.
