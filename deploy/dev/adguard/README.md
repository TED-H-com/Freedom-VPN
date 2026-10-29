# Local Dev AdGuard Home Resolvers (`deploy/dev/adguard`)

This directory provides a self-contained Docker Compose stack for developing and testing the Freedom VPN DNS policy engine and AdGuard Home backend (§12, DNS01–DNS04, SEC02, AT20–AT22).

## Components

1. **`upstream` (`coredns/coredns:1.12.0`)**:
   - Local authoritative test DNS server serving `example.com`, `example.net`, `.test`, and `notexample.com` with `A`, `AAAA`, multi-hop `CNAME` chains, and `HTTPS`/`SVCB` records (using RFC 5737 `192.0.2.0/24` and RFC 3849 `2001:db8::/32` documentation prefixes).
   - Configured as a closed authoritative server (returns `NXDOMAIN` for any other zone) so tests never send queries to third-party DNS servers.
2. **`resolver1` and `resolver2` (`adguard/adguardhome:v0.107.59`)**:
   - Two controlled AdGuard Home resolvers configured with `upstream:5353` as their only upstream (`fallback_dns: []`).
   - Enforces data minimization (§12.7, SEC02): `querylog.enabled: false`, `querylog.file_enabled: false`, `statistics.enabled: false`, `anonymize_client_ip: true`, `log.verbose: false`.
   - Enforces `blocking_mode: nxdomain` and a short default `blocked_response_ttl: 60` (§12.5).

## Host Port Bindings (127.0.0.1 only)

All ports bind strictly to `127.0.0.1` and can be overridden via environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `FVPN_DNS_PREFIX` | `fvpn-dev-adguard` | Compose project, container, and network name prefix |
| `FVPN_DNS_UPSTREAM_PORT` | `15350` | Authoritative test DNS server (UDP/TCP) |
| `FVPN_DNS_RESOLVER1_DNS_PORT` | `15351` | Resolver 1 DNS listener (UDP/TCP) |
| `FVPN_DNS_RESOLVER1_HTTP_PORT` | `18081` | Resolver 1 management HTTP API |
| `FVPN_DNS_RESOLVER2_DNS_PORT` | `15352` | Resolver 2 DNS listener (UDP/TCP) |
| `FVPN_DNS_RESOLVER2_HTTP_PORT` | `18082` | Resolver 2 management HTTP API |

## Bootstrap and Credentials

Running `./bootstrap.sh` (or `make -f mk/dns.mk dns-up`) generates random `TEST ONLY` admin credentials and initial `AdGuardHome.yaml` files inside the gitignored `deploy/dev/adguard/.state/` directory:

- `deploy/dev/adguard/.state/credentials.json`
- `deploy/dev/adguard/.state/credentials.env`
- `deploy/dev/adguard/.state/resolver1/conf/AdGuardHome.yaml`
- `deploy/dev/adguard/.state/resolver2/conf/AdGuardHome.yaml`

## Usage

From the repository root:

```bash
# Start stack
make -f mk/dns.mk dns-up

# Run integration suite against the running stack
make -f mk/dns.mk dns-test-integration

# Stop stack and remove containers, networks, volumes, and .state
make -f mk/dns.mk dns-down
```
