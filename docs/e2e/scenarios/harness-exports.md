# Shared harness exports (follow-up 12)

Run `TestE2E_HarnessExports` on a fresh prepared stack. Records go to
`runs/2026-10-30/followup12/<prefix>/`; they do not replace full-suite evidence.
Every record carries the source commit and one run ID. This task uses `e2e`
and then the temporary `e2e-exports` prefix, sequentially on its own base 18000.

The checks require real signed boot/candidate releases and API-created policy,
enrollment, publication and grants. No projector-owned rows are written.

| Check | Required observation |
|---|---|
| update-good | Shared probe setup and its non-secret notes; real updater issuance with `configured_probe_issued` true; timed exception elements, authenticated origin replies, nonzero probe counters, unchanged member counters, elements removed |
| update-bad | Real tampered candidate rejected before health; no challenge requirement for this earlier failure |
| export-native-streams | Two independent native joins; bounded final per-stream inventories, mode 0600/root, one recognized device-flow key line, successful deletion; foreign container refused |
| export-enrollment | Two harness keys race one code; one success/one rejection; successful response deliberately discarded; byte-identical replay completes one-time mTLS bootstrap and native join; complete correlated peer readback count one |
| export-clone | Consistent stopped-source snapshot, clone created before boot, original resumed; all three public identity hashes equal; both real entrypoints run |
| clone-conflict | Product conflict and mapping freeze actually observed; otherwise blocked with exact rows and owning domain |

The clone fixture is a fault in disposable storage, not a new authorized exit.
A successful clone setup does not prove AT07 safety. Complete incarnation
history still needs the ENR-03 admin read surface. The enrollment protocol
fixture uses a waiting node slot and does not fabricate agent SQLite state or
claim node readiness; ordinary serving nodes continue to use `EnrollAt`.
`clone-conflict` stays blocked: `ObserveEndpoint` has no production caller.

## Member-ingress injection (follow-up 13)

`Stack.InjectMemberIngress` and `Stack.ObserveMemberIngress` are the devstack
side of `InjectMemberIngress` in package `tests/netlab`. The acceptance
scenario calls `ObserveMemberIngress` after a real device withdrawal. Members
(AT02) and isolation (AT13) call the same exports; they do not exec
`fvpn-netlab` themselves.

| Export | What the caller gets |
|---|---|
| `InjectMemberIngress(ctx, "fvmb0", source, dest, 9)` | `MemberIngressResult`. Empty mechanism, so type 65534 selects `tun-mirred`. `deltas["ctunku"].packets` is 1 for a source that is not admitted. `deltas["u_<32 hex>"]` is the admitted device's upload counter. |
| `ObserveMemberIngress(ctx, source, dest)` | `MemberIngressProof`: that result, plus `fvup0` and origin-namespace tcpdump text for UDP port 9. |

The hook copies a linux/amd64 `fvpn-netlab` into the node container, reads
`nft -j list table inet fvpn`, and removes `fv-inj0` before it returns. It
does not restore a revoked peer, group, or binding. The node container is
already privileged, so this hook adds no capability.
