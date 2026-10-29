# Parallel stacks

Two prefixes ran at the same time on 2026-10-29. `stack-health` for each
prefix is the control `/readyz`, the IdP `/healthz`, the origin `/healthz`,
and a password login to that prefix's published Postgres port.

| Prefix | Port base | Control | IdP | Origin | Postgres |
|---|---:|---|---|---|---|
| `e2e` | 18000 | 18443 → 200 | 18444 → 200 | 18445 → 200 | 23432 authenticated |
| `side` | 24000 | 24443 → 200 | 24444 → 200 | 24445 → 200 | 29432 authenticated |

`e2e-down` for prefix `side` removed only `fvpn-side` containers, networks,
volumes, and images. Prefix `e2e` was still listed in `docker ps -a`. A later
`e2e-down` for prefix `e2e` removed only that prefix.

Afterwards these listings were empty:

- `docker ps -a` names matching `fvpn-e2e-` or `fvpn-side-`
- `docker network ls` names matching `fvpn-e2e` or `fvpn-side`
- `docker volume ls` names matching `fvpn-e2e-` or `fvpn-side-`
- `docker images` repositories matching `fvpn-e2e-` or `fvpn-side-`

The unrelated container `fvpn-pg` was still present.
