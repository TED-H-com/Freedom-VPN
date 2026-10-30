# at23-arm64

arm64 artifact and operator hardware

Status: needs_operator_run

Spec: §3.2, §13.4, §19 AT23, NODE01

At: 2026-10-30T00:00:16Z

Expected: arm64 is packaged and manifest-listed. A physical arm64 node and the three join sources need an operator.

Actual: arm64 updater cross-built; node image in this lab is amd64 only

Git: 433f621cbaf427c3a31fd375899b73591e41d668

Images: fvpn-w4-updates-node:local bd7f05b30cf7; fvpn-w4-updates-worker:local 2aec0292ecca; fvpn-w4-updates-control:local a16f12ce181e; fvpn-w4-updates-origin:local 54a5efcd9fd3; fvpn-w4-updates-payment:local 0c1a30526707; fvpn-w4-updates-idp:local abc41c5800a0; fvpn-w4-updates-netbird:0.82.0 7b5b53b2dc8b; fvpn-w4-updates-postgres:17 31ba9c94a03a; fvpn-w4-updates-nginx:1.30.0 55d1fb0944aa; fvpn-w4-updates-adguard:v0.107.59 5d5e3aef39a8; fvpn-w4-updates-coredns:1.12.0 40384aa1f5ea

Commands:

- GOOS=linux GOARCH=arm64 go build ./node/updater/cmd/fvpn-updater
- see docs/e2e/at/updates/operator-run.md

- updater_arm64_sha256: 5e7c9402aa3794a9cffe33c8e23a6bb36c4ad081e17e27c6c6f3fbbd2ff711f7
- emulation: Unable to find image 'ubuntu:24.04' locally
24.04: Pulling from library/ubuntu
e2d2bb84d022: Pulling fs layer
2238dc1b4376: Download complete
e2d2bb84d022: Download complete
e2d2bb84d022: Pull complet… exit status 255
- hardware: needs operator run; this host did not execute the arm64 binary
- sources: preinstalled box, cloud Ubuntu and coordinated third-party Ubuntu are operator-run.md; the lab container is one Ubuntu 24.04 amd64 source

