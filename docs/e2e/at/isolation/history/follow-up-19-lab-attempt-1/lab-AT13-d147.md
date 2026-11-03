# lab-AT13-d147

Mechanism: D-147 RELATED IPv4 fragmentation-needed per source set and overlap

Status: fail

Run: 0b5e3b1e-eedf-4014-ad29-5bb92e9296f1; started UTC: 2026-11-03T22:03:08Z; source dirty: false

Spec: §19 AT13, ISO01, ISO02, MTR01, D-146, D-147

UTC: 2026-11-03T22:03:09Z

Build: 150c090fa35f2d62d456613f92fb9283d56d6e16

Image digests:

- fvpn-w4-isolation-lab:lab sha256:830c7fc8f9ac635063bc720a850c9443ffc08e1d6b667c635d17e7c87657c7a9

Policy revision: generation 4200

Node incarnation: mechanism-lab

Expected: Genuine router IPv4 ICMP fragmentation-needed (type 3, code 4, MTU 1000) from resolver_v4-only, denied_v4-only, host_v4-only, and denied_v4+host_v4 overlap sources reaches the admitted member when conntrack marks it RELATED, is counted once in d_<device> at the device's assigned priority (1:2f), and matches independent captures; time-exceeded is dropped at ctdenyd, unsolicited fragmentation-needed at ctinv, and member upload to that source is dropped with 0 upload usage.

Actual: d147 runner failed: exit status 1; see lab-AT13-d147-detail.json; stderr=Traceback (most recent call last):
  File "<stdin>", line 876, in <module>
  File "<stdin>", line 853, in main
  File "<stdin>", line 612, in run_row
  File "<stdin>", line 267, in

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`
