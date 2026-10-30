# update-good-finding

First install ends recovery_required / ROLLBACK_FAILED

Status: blocked

Spec: §13.9, §13.11, §19 AT23

At: 2026-10-30T00:00:16Z

Expected: A first install with no last-good release records ROLLBACK_FAILED after health checks fail, without a symlink restore.

Git: 433f621cbaf427c3a31fd375899b73591e41d668

Images: fvpn-w4-updates-node:local bd7f05b30cf7; fvpn-w4-updates-worker:local 2aec0292ecca; fvpn-w4-updates-control:local a16f12ce181e; fvpn-w4-updates-origin:local 54a5efcd9fd3; fvpn-w4-updates-payment:local 0c1a30526707; fvpn-w4-updates-idp:local abc41c5800a0; fvpn-w4-updates-netbird:0.82.0 7b5b53b2dc8b; fvpn-w4-updates-postgres:17 31ba9c94a03a; fvpn-w4-updates-nginx:1.30.0 55d1fb0944aa; fvpn-w4-updates-adguard:v0.107.59 5d5e3aef39a8; fvpn-w4-updates-coredns:1.12.0 40384aa1f5ea


