# TEST ONLY NetBird CE lab

Pinned server/client 0.82.0, dashboard 2.96.0 and nginx 1.30.0; exact upstream
index digests live in the four Dockerfiles and [VERSIONS](../../../docs/netbird/VERSIONS.md).
Go test/build scratch uses the disk-backed `.agent/netbird-tmp/` directory
through Make; do not use a tmpfs TMPDIR. Requires Go 1.27, Python 3 standard library, Docker Compose, Linux `/dev/net/tun`
and permission to run containers with NET_ADMIN/NET_RAW. No host networking
commands or privileged host setup are part of this lab.

Run from the repository root:

```sh
export FVPN_NB_PREFIX=fvpn-w1-netbird
make -f mk/netbird.mk netbird-up netbird-bootstrap
# Prove the bootstrap is safe to repeat:
python3 -B deploy/dev/netbird/bootstrap.py
make -f mk/netbird.mk netbird-test netbird-vet netbird-test-integration
make -f mk/netbird.mk netbird-reset
```

`FVPN_NB_PREFIX` defaults to `fvpn-dev-netbird`. On a shared host, set your task's
prefix explicitly. All locally built images, containers, networks and volumes
use that prefix. `FVPN_NB_API_PORT` defaults to 18080; `FVPN_NB_STUN_PORT` to
13478. Both published sockets bind **127.0.0.1 only**. Configure ports before
`netbird-up`; after a port change reset this disposable stack to regenerate the
issuer/redirect configuration. Nothing binds port 80 or 443 on the host.

The combined server owns management, signal, relay, STUN and the embedded Dex
IdP. It and all peers live on an internal Docker network without external
egress. A small nginx gateway is also on a frontend network for loopback port
publication; its upstreams are fixed to the local server/dashboard. nginx
proxies STUN as well as HTTP/gRPC/WebSocket relay. QUIC relay transport is not
exposed by this dev gateway. Geolocation, telemetry, client auto-update and JWT
auto-group propagation are explicitly disabled. The lab exercises only its own
overlay peers; it never browses through an exit or probes third-party services.

## Bootstrap and state

`prepare.py` generates `.state/config.yaml` with random TEST ONLY encryption and
relay secrets. `bootstrap.py` uses only HTTP APIs to create the owner, remove
All-to-All, disable IPv6 allocation, create the adapter service user/PAT and two
ordinary local users. Passwords are random and are never printed. The adapter
service has no auto-groups. Its built-in `admin` role is needed for setup-key
creation and peer deletion; `network_admin` cannot perform both in this version.

The owner PAT expires in one day and the service PAT in 30 days. All secret files
are mode 0600 inside mode-0700 `.state/`, ignored by the repository's existing
rule. Do not copy these files into reports, CI artifacts, shell history or Git.
The generated prefix marker prevents accidentally resetting another prefix's
state. NetBird's databases live in its own named volume; these scripts never
open them. Peer private keys/state live in one named volume per client.

Only after policy removal does the Make target start clients. The client
entrypoint supplies an empty private-key template and the private management
URL before the daemon starts. `login-members.py` completes the actual embedded
IdP device-authorization flow for member-a and member-b. These are real user
peers, **not setup-key substitutes**. It resolves the documentation hostname to
the published loopback port inside that Python process only; no host DNS or
hosts-file changes occur. Login codes/passwords are never printed. The exit
joins with the adapter's one-off key during integration; `replay` tests rejection
of its second use. Plaintext keys travel over stdin into temporary mode-0600
container files and are deleted after the command.

## Browser access (optional)

The browser must resolve `netbird.example.invalid` to the loopback gateway; it is
intentionally not a public DNS name. On a workstation with Chromium installed,
use a **disposable browser profile** and a process-local resolver override:

```sh
chromium --user-data-dir=./deploy/dev/netbird/.state/browser \
  --host-resolver-rules='MAP netbird.example.invalid 127.0.0.1' \
  http://netbird.example.invalid:18080
```

Set the browser's secure-DNS feature off in that disposable profile so the local
mapping is used. If running on a remote dev host, forward its loopback TCP port
to the workstation using your existing approved access method. The username is
`admin@example.invalid`; its generated password is in `.state/admin-password`.
Use the embedded login, then inspect Peers, Policies and DNS. Browser usability
and admin MFA were not exercised by the machine run. This HTTP/password-only
lab is not a production configuration and is not suitable for native phone
acceptance; that needs a reachable test URL with trusted TLS and enforced MFA.

## Stop, reset and failure recovery

`netbird-down` stops/removes containers and networks while preserving state
volumes and secret files. `netbird-up netbird-bootstrap` resumes the same lab.
`netbird-reset` explicitly removes its containers, networks, volumes, locally
named images and `.state/`; it is destructive to this disposable lab only.
Use reset followed by up/bootstrap for a fresh one-off-key test run.

Bootstrap retries adopt named users instead of creating duplicates. If a token
creation response is lost, bootstrap revokes the unusable named PAT and replaces
it. If the **initial owner** response is lost or its one-day PAT has expired,
use the saved admin password in the local dashboard to create a replacement
owner PAT and write it to `.state/owner-token` with mode 0600; alternatively
reset the lab. Do not create another owner. If a member creation response is
lost before its generated password is saved, reset the lab; bootstrap stops
rather than inventing another member identity.

If login fails, diagnostics stay in private `.state/*-login.log`; never paste
those files (they may contain expiring device codes). Confirm the API socket,
internal gateway DNS and gRPC proxy configuration. A freshly started client
must point at the private URL from its first launch. Restoring old cloud-default
client volumes can leave an ongoing cloud connection attempt; reset the lab.

## Tests and recordings

The build tag is `netbird_integration`. Direct `go test -tags netbird_integration`
skips if `FVPN_NETBIRD_API_URL` or `FVPN_NETBIRD_TOKEN_FILE` is unset. The URL must
end in `/api`. The Make integration target sets both to this lab by default and
sets `FVPN_NETBIRD_CLIENT_TESTS=1`; override them only for your own disposable
instance. The CLI suite requires the matching Compose clients and explicit
prefix. It does not execute into any other containers.

To regenerate API recordings, add `FVPN_NETBIRD_RECORD=1` to the integration
command, then run `python3 -B deploy/dev/netbird/scrub-recordings.py`. Review the
five JSON fixtures under `internal/netbird/testdata` before committing. Raw
recordings remain private/ignored. No setup-key or token response is recorded.
The fake's setup-key response is synthetic and explicitly TEST ONLY.

Results and coverage limits: [G1 evidence](../../../docs/poc/G1-netbird.md),
[manual apps checklist](../../../docs/poc/G1-app-checklist.md),
[adapter contract](../../../docs/netbird/ADAPTER.md).
