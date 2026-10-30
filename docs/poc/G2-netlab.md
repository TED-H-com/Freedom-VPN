# Gate G2: node data path (lab evidence)

PoC for §8.2 and §10.2. The hook, priority, namespace boundary, SNAT position and classification below are the choices this lab's packets support. Raw rulesets, counters and logs are in `docs/poc/G2-evidence/`. Every address in that directory is from a documentation or lab range (`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`, `2001:db8::/32`, `100.64.0.0/10`, RFC1918, link-local).

The lab is one privileged container with its own network namespace. `fv-fwd` stands in for the NetBird routing peer. The container root namespace is the node host. The simulated Internet never leaves the container.

## Topology

```text
member ns          fv-fwd (NetBird stand-in)         root netns (node host)          fv-inet / fv-lan
-----------        --------------------------         ----------------------          ----------------
fv-a1  100.64.1.1
fv-a2  100.64.1.2   kernel WG :51820
fv-b1  100.64.2.1   table inet fvnb                   table inet fvpn
fv-rogue 100.64.9.9  (accept-all, optional masq)      filter forward priority -150
fv-probe 100.64.8.1  dedicated test peer, not a member
        |                 |                           nat postrouting priority 100
        +---- underlay ---+---- veth fv-ff / fv-fh ---+---- veth fv-ih / fv-ii ---- fv-inet
        192.0.2.0/24 /30s                              tc HTB egress fv-ih (upload)
                                                       tc HTB egress fv-fh (download)
                                                       veth fv-lh / fv-ll -------- fv-lan
```

| Namespace | Role |
|---|---|
| `fv-a1`, `fv-a2` | Member A, two devices. Same member, two overlay addresses. |
| `fv-b1` | Member B, one device. |
| `fv-rogue` | Tunnel with no binding. |
| `fv-probe` | Dedicated synthetic-probe peer (§13.9). Not a member. Overlay `100.64.8.1/32`. `100.64.8.2/32` is a second address on the same tunnel so a same-/24 source reaches nft. |
| `fv-fwd` | Terminates kernel WireGuard. Overlay `100.64.0.0/10`. Inner table `inet fvnb` only. |
| root | Node host. SSH-like `:2222` and agent-API `:8443`. Table `inet fvpn` only. |
| `fv-inet` | Simulated Internet `203.0.113.10` and `203.0.113.20`, resolver `198.51.100.53` (DNS 53 and a non-DNS port). |
| `fv-lan` | RFC1918 services, `169.254.169.254`, link-local `169.254.1.50`, public-prefix private host `192.0.2.200` (§8.1), IPv6 ULA `fd00:db8:ffff::10`, documentation `2001:db8::10`, link-local `fe80::10`. |

Policy routing in `fv-fwd` stops hairpin: traffic from the host is routed to the matching peer, and traffic from a tunnel is routed to the host. Peer allowed-ips are only that peer's own addresses, so a spoof of another device's overlay address does not leave the tunnel. The unbound source `100.64.7.7` injected on `fv-ff` is the nft unknown-source case.

`203.0.113.0/24` is removed from the lab deny set only because that prefix is the simulated Internet. Production must keep `SpecDeniedV4()` intact. The resolver prefix stays denied except TCP/UDP 53.

## Chosen data path (§8.2, §10.2)

Outer enforcement is the host namespace, table `inet fvpn` only. NetBird's simulated rules live in `fv-fwd` as `inet fvnb` and never write `inet fvpn`.

| Direction | Namespace | Interface | Table | Chain | Hook | Priority | NAT | Classification |
|---|---|---|---|---|---|---|---|---|
| Upload admission | root | ingress `fv-fh` | `inet fvpn` | `forward` | filter forward | **-150** (after conntrack -200, before filter 0 and srcnat 100) | none yet | source must be in `allowed_v4` (kernel set with timeout) |
| Upload count and class | root | `fv-fh` | `inet fvpn` | `forward` | same | -150 | before SNAT | per-device counter, then `meta priority set` to the member classid |
| Upload SNAT | root | egress `fv-ih` | `inet fvpn` | `postrouting` | nat postrouting | **100** | `snat` to `192.0.2.37` | class already stored in `skb->priority` |
| Upload shape | root | egress `fv-ih` | tc | HTB `1:` | egress qdisc | n/a | after SNAT | HTB selects the leaf whose classid equals `skb->priority` |
| Download admission | root | egress toward `fv-fh` | `inet fvpn` | `forward` | filter forward | -150 | conntrack has already un-SNATed | destination must still be in `allowed_v4` **before** any established/related accept |
| Download count and class | root | `fv-fh` | `inet fvpn` | `forward` | same | -150 | after reverse NAT | per-device download counter, then `meta priority set` |
| Download shape | root | egress `fv-fh` | tc | HTB `1:` | egress qdisc | n/a | inner addresses | same classid |

Forward policy is `drop`. Input from `fv-fh` and from `100.64.0.0/10` is dropped, so a member cannot open the node's own listeners (§8.1, §17.1). Output policy is `accept`, so the control channel is not part of the member path.

Serving forward rules name both interfaces. Upload matches `iifname` `fv-fh` and `oifname` `fv-ih`. Download matches `iifname` `fv-ih` and `oifname` `fv-fh`. The boot baseline still drops member ingress and member egress without requiring the uplink, so a packet that is not on that pair is the forward policy drop. Named deny and `ctv6` counters count only the pair.

Classification does not write `skb->mark` or the conntrack mark. NetBird's mark range `0x1BD00–0x1BDFF` is left alone. No tc filter is installed (`cls_fw`, `cls_u32`, `cls_flower` and `cls_matchall` are not required). `skb->priority` is `(1<<16)|minor`. Minor 0 is HTB direct, so the host default leaf is minor `0xFFFE` (class `1:fffe`). A lab table `inet fvprobe` (not `inet fvpn`) set mark `0x1bd11` and conntrack mark `0x1bd22` and read them again after forward: `mark_pre=5 mark_post=5`.

There is no flowtable and no `flow add` offload (§10.3). GRO, GSO and TSO are off on the lab veths. The forward counter records the inner IPv4 total length, once, before the shaper. A later qdisc drop can already have been counted, so these counters are admission bytes, not delivered bytes (§11.7 uses the HTB class byte counter for rate).

IPv6 mode is `block` (§8.3). Member IPv6 on the member-uplink pair is dropped and counted on `ctv6`. There is no IPv6 forward path and no `::/0` accept.

Lease elements carry an nft `timeout`. That timeout is in jiffies. It is installed before any established/related accept, so a stuck agent does not keep an expired flow (§16.3). The deadline is the earliest of the offline cap (`signed_server_time + max_offline`), the service end, and the node lease end. A zero signed server time or a non-positive offline cap is not admitted. Re-adding an element restarts the timer, so the agent must pass a non-increasing `TimeoutCeilSeconds`. Revoke deletes the conntrack entry for that address after the new ruleset is in place (§8.2).

A ruleset change is one `nft -f` transaction: delete and add of our chains happen in that transaction, and existing named counters are not recreated. TC is applied first, then nft. If verification fails, the boot baseline is installed with a nil probe: member forward stays drop, the probe sets are removed, and output stays accept (§8.2, §17.1).

A health check that must prove the data path while members stay blocked calls `ApplyBaselineModel` with `Model.Probe` (§13.9, D-063). `ApplyBaseline`, which the agent and the watchdog call, is that function with a nil probe. The exception is one `/32` source, at most `ProbeMaxDestinations` (4) exact destination/protocol/port elements that do not overlap a member binding, and optionally TCP/UDP port 53 to the resolver. The upload tuple is behind the invalid, IPv6, deny, and host drops. Resolver DNS is before the deny drop, because the resolver prefix is inside that set, and every other port to the resolver is dropped. The download accept is `ct state established,related` to the probe address. Elements carry a kernel timeout of at most `ProbeMaxSeconds` (300). `TimeoutCeilSeconds`, and the remaining lifetime `State` read from the live elements, stop a later render from extending that timeout. While the probe sets already exist and no remaining lifetime was recorded, a fresh 300 s window is not written back. `Verify` rejects a prefix wider than `/32`, an element it cannot parse, an original timeout above `ProbeMaxSeconds`, or a remaining lifetime past the model window (`ReasonProbeUnbounded`). `probe_up` and `probe_down` are diagnostic counters and are not member usage. A nil probe deletes the sets after the chains are replaced.

## Results

Short windows. `FVPN_NETLAB_FULL=1` is the §11.7 window (30 s warm-up, 120 s, three repetitions) and was not run. QoS rates below are TEST values, not product plan values (§11.3, §11.7).

| Case | Result |
|---|---|
| AT00 | 5 WireGuard handshakes after the probe peer was added, forwarding on only after the baseline, baseline present. The earlier evidence file still records the 4-peer topology. |
| AT12 / ISO01 / ISO04 | 17 deny probes, all timeout or dropped. DNS UDP replied and DNS TCP connected, and that traffic stayed in `dns_service_v1`. Member TCP, UDP and ICMP to `203.0.113.10` connected. Target captures empty. Diagnostic counters on the member-uplink pair moved by 8. |
| AT11 | Native IPv6 to global `2001:db8:203::10`, ULA, `2001:db8::10` and `fe80::10` timed out. IPv4-mapped targets were unreachable. Raw IPv6 was sent and both target captures stayed empty. `ctv6` rose by 3. |
| AT14 / MTR01 / MTR02 | Upload and download deviation 0.0000 against the inner IP length, for UDP and TCP. DNS and host-originated traffic did not move `forward_admission_v1`. Three off-uplink denies did not move usage or `ctdeny`. A counter reset was reported as `METER_EPOCH_CHANGE`. |
| AT18 / FUP02 | Two devices of one member shared one 20 Mbit class at ratio 1.000 on upload, download and both at once. Two members on a 20 Mbit parent: 0.500 and 0.500, sum 1.000. Drops 0. Class minor survived a rate change (`1:2f`, `1:30`). Removing member B deleted class `1:30`. |
| AT10 / AT13 | 100 regenerations, one LAN interface add/remove, revoke stopped the live flow in 1.608 s and removed its conntrack entry, `kill -9` of the apply process left generation 2100 in place, then the boot baseline denied every member. The deny loop completed 9 passes and recorded no allowed probe. Target captures empty. |
| AT29 | Forwarding stopped 8.763 s after the short service end, with the apply process already gone. It stayed denied, then resumed only after generation 30. The offline cap and the backward-clock ceil were checked in process. The kernel clock step stayed skipped. |
| NetBird | Inner accept-all did not bypass member, LAN, metadata or rogue denies. Masquerade to `192.0.2.34` was dropped as an unknown source (`ctunku`) and `meter-path` returned `METER_PATH_INVALID`. Marks preserved. |
| Probe (§13.9) | On the boot baseline, `100.64.8.1` reached `203.0.113.10:80` and was SNATed to `192.0.2.37`. `probe_up` rose by 5 packets and `probe_down` by 3. A same-/24 source, an unlisted destination, a different port, and the node address on TCP 22 were dropped. After the 8 s element timeout the same flow was dropped and the sets no longer held the addresses. Resolver DNS admitted only port 53. Member usage counters did not move. |

### AT14 byte identity (§10.3, §10.7)

Reference is `tcpdump -v` on the member tunnel device `fv-wg0`, summing IPv4 total length. That is the same layer as the forward counter.

| Flow | nft upload | inner upload | nft download | inner download | deviation |
|---|---:|---:|---:|---:|---:|
| UDP 400 × 1200 payload | 491200 | 491200 | 491200 | 491200 | 0 |
| TCP 80 × 1000 payload | 85528 | 85528 | 88584 | 88584 | 0 |

`491200 = 400 × 1228` (20 byte IPv4 header + 8 byte UDP header + 1200 payload). The same blast on `fv-wg0` link stats was `505600 = 400 × 1264`. The extra 36 bytes per datagram are WireGuard ciphertext overhead (4 bytes of pad-to-16, 16 byte header, 16 byte tag), not usage. Underlay `fv-a1u` TX was 522400, strictly above the inner length, and is not usage. Member `netdev` egress and `inet` output both miss some 52-byte TCP ACKs on this kernel, so they are not a valid inner reference. The 2% bar was not loosened.

DNS upload for the resolver probes increased `dns_service_v1` by 381 bytes and left `forward_admission_v1` unchanged. Three denied probes to LAN and metadata addresses did not increase usage. `ctdeny` stayed put because those destinations do not egress `fv-ih`; the forward policy dropped them. A host-originated TCP connection to `203.0.113.10` did not increase any member counter.

### AT18 (§11.4, §11.7)

Measured on `tc -s class` bytes during the steady window, which is the shaper, not the admission counter. Short window: 5 s warm-up, 20 s measure, 1 repetition. R = 20 Mbit, contention C = 20 Mbit.

| Window | Ratio of R or C | Drops | CPU |
|---|---:|---:|---:|
| Shared upload (a1+a2) | 1.000 (20.00 Mbit) | 0 | 9.2% |
| Shared download | 1.000 (20.00 Mbit) | 0 | 30.8% |
| Shared both, each direction | 1.000 / 1.000 (20.00 Mbit) | 0 | — |
| Contention, 8 flows vs 1 flow | 0.500 and 0.500, sum 1.000 | 0 | 7.7% |

The parent ceil is what caps the two members. Host class `1:fffe` stayed at 0.00 Mbit on the shared upload window. After a rate change the same minors remained (`1:2f` for member A, `1:30` for member B) and the class byte counter did not go backwards (177922189 bytes kept). Dropping member B removed class `1:30` and its leaf qdisc on `fv-ih` and `fv-fh`. Member A stayed on `1:2f`.

### AT29 (§16.3)

Service end was 8 s and the offline lease was 1 h. The apply process exited. Forwarding still worked at +1 s and was denied at 8.763 s, which is the kernel set timeout and not a userspace delete. A new model at generation 30 resumed it.

In process, an offline cap of 30 s is earlier than a 24 h node lease and a 48 h service end, and a zero signed server time or a zero offline cap is rejected. A process clock stepped back two hours, with `TimeoutCeilSeconds` 45, rendered `timeout 45s`. That check does not call `date -s`.

`unshare --time` creates a new time-namespace inode, and `date -s` inside that child still moves the container clock. The test compares the inodes, records `clock_step=skipped`, and does not call `date -s`. nft timeouts follow jiffies, so a wall-clock step would not have been the enforcement mechanism anyway.

### Synthetic probe (§13.9, D-063)

The probe scenario keeps member forwarding on the boot ruleset. The only forward accepts are the declared exception. `100.64.8.1` to `203.0.113.10` TCP 80 connected. The capture on the simulated Internet showed source `192.0.2.37` and did not show `100.64.8.1`. `probe_up` increased by 5 packets and `probe_down` by 3. No `u_`, `d_`, `nu_`, or `nd_` counter changed.

| Case | Flow | Observed |
|---|---|---|
| Positive | `100.64.8.1` to `203.0.113.10:80` | connected, SNAT applied |
| Negative A | `100.64.8.2` (same /24) to that socket | timeout |
| Negative B | `100.64.8.1` to `203.0.113.20:80`, and to `203.0.113.10:9001` | timeout |
| Negative C | `100.64.8.1` to `192.0.2.37:22` (listener on the uplink address) | timeout |
| Negative D | the positive flow after the 8 s `ExpiresAt` | timeout; the probe sets no longer contained the elements |
| Resolver DNS | UDP and TCP 53 to `198.51.100.53`; TCP 80 to that address; port 53 to `203.0.113.10` | reply and connected; the other three timed out |

The suite that includes this scenario and AT00 printed `SUMMARY PROBE snat=1 up=5 down=3 neg=4 expired=1 dns_only_53=1` and `ok github.com/TED-H-com/Freedom-VPN/tests/netlab 168.128s`. The rows above this section, other than AT00's handshake count, are the earlier measured windows. This run did not replace them. `docs/poc/G2-evidence/SUMMARY.txt` still says `AT00 handshakes=4` from that earlier capture.

## Decisions

- D-019: usage is the inner IPv4 total length admitted at the forward hook (§10.2). Bytes later dropped by HTB are not subtracted. "Delivered bytes" is not a ledger rule. The lab counters are the meter definition for the node agent.
- Serving forward rules bind the member interface and the uplink. Upload uses `oifname` uplink. Download uses `iifname` uplink. The baseline drop does not.
- D-063: the synthetic probe is an optional `Model.Probe`. Nil removes it through the same render path. The kernel element timeout is the authorization, capped by `ProbeMaxSeconds` and by `TimeoutCeilSeconds` or the remaining lifetime already in the kernel. At most four destinations, none overlapping a member binding. `Verify` fails closed on a wide prefix, an unparsed element, an original timeout above the ceiling, or a remaining lifetime past the model window. `probe_up` and `probe_down` are not member usage. The agent authenticates the test peer before it supplies the `/32`.
- Clock-step expiry (§16.3) stays under 待實測. The kernel set-timeout evidence is accepted for Gate G2. This lab does not call `date -s` and does not change the host clock.
- §11.7 production windows remain a recommended starting point. The short window is what CI runs.

## 待實測

This lab does not claim any of the following:

- A real NetBird routing peer, its real nftables table, fwmark routing, or a version upgrade that reorders hooks (§10.3).
- Real member apps and their DNS, QUIC and private-DNS behavior (§12 is a different task).
- arm64 hardware. `GOARCH=arm64 go build` is a cross-compile only.
- The relay path, and MTU, PMTU, TCP MSS and QUIC on real underlays (§8.3). The lab underlay MTU is 9000 so the outer packet is not fragmented. Inner WireGuard MTU is 1420.
- Suspend, resume, VM restore, and a clock step on a kernel where a time namespace is actually isolated (§16.3).
- Userspace NetBird or a netstack path. These counters are kernel forward-hook counters only (§10.3).
- The §11.7 long window (30 s / 120 s / 3 repetitions) and a line whose unshaped capacity is at least 1.5× R. The short window passed at the TEST rates. It is not a product rate.
- Agent watchdog, SQLite outbox, sampling loop, meter-epoch UUID, and usage upload. The library reports `METER_EPOCH_CHANGE` and leaves the epoch to the agent (§10.5, §10.7).
- IPv6 forwarding. v1 mode is block only (§8.3).
