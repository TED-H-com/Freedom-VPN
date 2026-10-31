# Current full-case blockers

The authoritative outcomes are [summary.md](summary.md) and the per-AT JSON.
Owner, location, observed reproduction and requested hook are in
[requests.md](requests.md).

| Request | Cases | Remaining condition |
|---|---|---|
| F4-03 | AT01, AT04, AT09 | Real platforms/network moves; member UDP and forced-relay/QUIC/PMTU/physical NAT matrix |
| F4-04 | AT04 | Member UDP withdrawal and rendered suspended/expired screens |
| F4-05 | AT20, AT21 | Native DNS is disabled; automatic fallback between the two controlled filters is unproved |
| F4-06 | AT22 | Remaining retention/file/backup/URL surfaces and commercial feed approval |
| F4-07 | AT20 | Complete controlled RR, cache-expiry and rebinding fixtures |
| F4-10 | AT05 | Preinstalled/cloud-init boot sources and physical arm64 |

- **AT02 strict post-withdrawal ingress** is resolved by devstack follow-up 13 (#219) and verified in [AT02.json](AT02.json): `ObserveMemberIngress` on `fvmb0` (`tun-mirred`, link type `65534`, port `40000` -> `192.0.4.10:9`) proves that the revoked source (`100.104.160.29`) increments `ctunku` by `+1` packet (`+29` bytes) with `0` forwarded bytes and empty `fvup0`/`origin` captures, while the admitted sibling (`100.104.242.187`) increments `u_01a18ef9ae937a1cb93420bed403bf2f` by `+1` packet and reaches `fvup0` and `origin`.
- **F4-01** (AT09 cold namespace restart empty same-generation member lease) is resolved by PR #193 (D-067), PR #213 (contract revision 11), and PR #216 (agent follow-up 18), reporting `awaiting_member_grants` and `error_code: "AWAITING_MEMBER_GRANTS"` on `lease`, `qos` and `source_binding` checks (`stage=lease` count `1`, never `lease_expired`) during the provisional window and restoring the 7-grant generation-`23` lease and counted `fvmb0` TCP traffic in `41.695 s` from container start (`36.976 s` to serving-ready).

Earlier runs remain under [history/follow-up-5](history/follow-up-5/RUN.md),
[history/follow-up-4](history/follow-up-4/RUN.md) and
[history/follow-up-3](history/follow-up-3/RUN.md). Operator-only rows remain
**needs operator run** in [OPERATORS.md](OPERATORS.md).
