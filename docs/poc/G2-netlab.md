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

## Special-purpose deny list

`SpecDeniedV4()` follows the IANA IPv4 Special-Purpose Address Registry (last updated 2025-10-09). Globally Reachable means a destination in that block is forwardable beyond a specified administrative domain. A block is denied when that cell is not True. Multicast `224.0.0.0/4` is not a row of that registry and stays denied.

| Prefix | Name | Globally Reachable | Decision |
|---|---|---|---|
| `0.0.0.0/8` | This network (covers `0.0.0.0/32`) | False | deny |
| `10.0.0.0/8` | Private-Use | False | deny |
| `100.64.0.0/10` | Shared Address Space | False | deny |
| `127.0.0.0/8` | Loopback | False | deny |
| `169.254.0.0/16` | Link Local | False | deny |
| `172.16.0.0/12` | Private-Use | False | deny |
| `192.0.0.0/24` | IETF Protocol Assignments | False | deny. Not usable unless a more specific reservation says so. Covers `192.0.0.0/29`, `192.0.0.8/32`, and `192.0.0.170/32`, `192.0.0.171/32` |
| `192.0.0.9/32` | Port Control Protocol Anycast | True | denied by the parent `192.0.0.0/24`. No hole |
| `192.0.0.10/32` | TURN Anycast | True | denied by the parent `192.0.0.0/24`. No hole |
| `192.0.2.0/24` | Documentation (TEST-NET-1) | False | deny |
| `192.31.196.0/24` | AS112-v4 | True | not denied |
| `192.52.193.0/24` | AMT | True | not denied |
| `192.88.99.0/24` | Deprecated 6to4 Relay Anycast (RFC 7526, allocation terminated 2015-03) | blank | deny. This is the prefix the list was missing |
| `192.88.99.2/32` | 6a44-relay anycast | False | denied by `192.88.99.0/24` |
| `192.168.0.0/16` | Private-Use | False | deny |
| `192.175.48.0/24` | Direct Delegation AS112 | True | not denied |
| `198.18.0.0/15` | Benchmarking | False | deny |
| `198.51.100.0/24` | Documentation (TEST-NET-2) | False | deny |
| `203.0.113.0/24` | Documentation (TEST-NET-3) | False | deny |
| `240.0.0.0/4` | Reserved | False | deny |
| `255.255.255.255/32` | Limited Broadcast | False | deny |

The member path and the updater probe validator both call `SpecDeniedV4()`. A rendered ruleset puts `192.88.99.0/24` in `@denied_v4` and does not put the three globally reachable /24s there.

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

D-085: the serving gate fails closed with `TC_FOREIGN_FILTER` when live tc read-back finds a foreign qdisc, filter or action on either the member interface or uplink. The agent still installs no tc filter. See `TC serving read-back` below for the measured redirect and the limit of nft baseline rollback.

There is no flowtable and no `flow add` offload (§10.3). GRO, GSO and TSO are off on the lab veths. The forward counter records the inner IPv4 total length, once, before the shaper. A later qdisc drop can already have been counted, so these counters are admission bytes, not delivered bytes (§11.7 uses the HTB class byte counter for rate).

IPv6 mode is `block` (§8.3). Member IPv6 on the member-uplink pair is dropped and counted on `ctv6`. There is no IPv6 forward path and no `::/0` accept.

Lease elements carry an nft `timeout`. That timeout is in jiffies. It is installed before any established/related accept, so a stuck agent does not keep an expired flow (§16.3). The deadline is the earliest of the offline cap (`signed_server_time + max_offline`), the service end, and the node lease end. A zero signed server time or a non-positive offline cap is not admitted. Re-adding an element restarts the timer, so the agent must pass a non-increasing `TimeoutCeilSeconds`. Revoke deletes the conntrack entry for that address after the new ruleset is in place (§8.2).

`conntrack` is an enforcement prerequisite. `RequireConntrack` checks `LookPath` through the `Executor` before the agent starts forwarding. The binary deletes NAT state after a revocation. The ruleset does not depend on that delete to stop the flow: upload drops `ip saddr != @allowed_v4` and download drops `ip daddr != @allowed_v4` before any established or related accept. A `conntrack` failure after a verified ruleset therefore leaves the serving ruleset installed. `Apply` returns `ISOLATION_POLICY_INVALID` with the fixed detail `conntrack delete failed after the serving ruleset was verified` (no address in the message). The next `Apply` that still lists the address retries the delete. It does not install the baseline, so a host that lacks the binary does not flap between the serving ruleset and the boot ruleset.

The agent distinguishes that outcome from a verification failure without comparing detail text. `ApplyResult.Verified` is true only when the serving ruleset and QoS were verified and left installed (the success return and the conntrack-failure return) and false when the baseline was installed or nothing was installed. The conntrack failure is a `*ConntrackDeleteError`: `Pending` is how many revoke addresses were not deleted, and `Unwrap` is the `*CheckError` above, so `errors.As` to `*CheckError` still sees `ISOLATION_POLICY_INVALID` and the error string is unchanged. `IsConntrackDeleteFailure` is that type. `RequireConntrack` is not it. That pair is the contract for the agent.

`VerifyServing` accepts a forward `accept` only when it is a rule `Render` emits. That is member DNS to `@resolver_v4` on TCP or UDP port 53 after the upload `ip saddr != @allowed_v4` drop, a per-device upload or download accept after the `@denied_v4` and `@host_v4` drops, and a download accept only after `ip daddr != @allowed_v4` and the non-established drop. A probe adds only its verified accepts. `ct state related` alone, a mark match, an unconditional accept, or an extra interface match fails in serving mode the same way the baseline rejects an accept outside the probe. The upload and download allow-set drops are tracked separately and both must precede accepts of that direction. When the model has resolvers or bindings, `@resolver_v4` must equal those resolvers (a prefix fails) and the ruleset must drop `@denied_v4`, `@host_v4`, and every non-53 port to `@resolver_v4` (`ctbadns`, after the port-53 accepts and before the deny drop). The only chains in `inet fvpn` are `forward`, `input`, `output`, and `postrouting` with the rendered hook and priority. `ParseRuleset` keys chains by family, table, and name and rejects a duplicate instead of merging it. No input accept of the member ingress or of `100.64.0.0/10` may precede those two drops. `allowed_v4`, `probe_v4`, and `probe_dst_v4` may carry only the `timeout` flag the renderer emits, and `resolver_v4` may carry none.

`ExtraDeniedV4` is where the caller lists the public on-link uplink prefix and the gateway. `SpecDeniedV4` does not contain that prefix, so a member can reach other hosts on the uplink LAN until those addresses are added. The lab removes `203.0.113.0/24` from the deny list so the simulated Internet stays reachable. A deployment whose uplink LAN is a public prefix supplies that prefix and gateway itself.

A ruleset change is one `nft -f` transaction: delete and add of our chains happen in that transaction, and existing named counters are not recreated. TC is applied first, then nft. If verification fails, the boot baseline is installed with a nil probe: member forward stays drop, the probe sets are removed, and output stays accept (§8.2, §17.1).

A health check that must prove the data path while members stay blocked calls `ApplyBaselineModel` with `Model.Probe` (§13.9, D-063). `ApplyBaseline`, which the agent and the watchdog call, is that function with a nil probe. The exception is one `/32` source, at most `ProbeMaxDestinations` (4) exact destination/protocol/port elements that do not overlap a member binding, and optionally TCP/UDP port 53 to the resolver. The upload tuple is behind the invalid, IPv6, deny, and host drops. Resolver DNS is before the deny drop, because the resolver prefix is inside that set, and every other port to the resolver is dropped. That DNS accept is only `iif` member ingress, `oif` uplink, `saddr @probe_v4`, `daddr @resolver_v4`, one of TCP or UDP, destination port 53. `@probe_v4` holds the probe source `/32` for the element lifetime, and `@resolver_v4` must equal the configured resolver addresses. `Verify` fails closed (`ISOLATION_POLICY_INVALID`) when an accept names the probe and is not that pair or the destination tuple, when the two DNS accepts are not one TCP and one UDP, or when `@resolver_v4` is not exactly those addresses. The download accept is `ct state established,related` to the probe address. Elements carry a kernel timeout of at most `ProbeMaxSeconds` (300). `TimeoutCeilSeconds`, and the remaining lifetime `State` read from the live elements, stop a later render from extending that timeout. While the probe sets already exist and no remaining lifetime was recorded, a fresh 300 s window is not written back. `Verify` rejects a prefix wider than `/32`, an element it cannot parse, an original timeout above `ProbeMaxSeconds`, or a remaining lifetime past the model window (`ReasonProbeUnbounded`). `probe_up` and `probe_down` are diagnostic counters and are not member usage. A nil probe deletes the sets after the chains are replaced.

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
| Probe DNS bound | Live ruleset: one TCP and one UDP accept, `saddr @probe_v4` to `daddr @resolver_v4` port 53. `@denied_v4` contains `192.88.99.0/24` and not the three globally reachable /24s. `@resolver_v4` is `198.51.100.53`. UDP and TCP 53 replied. Port 80 and 853 to the resolver, `10.1.2.10` on ports 53 and 9, and `192.88.99.1` on TCP 53 and UDP 80 timed out. Captures on the uplink and the LAN were empty. A same-/24 source was denied. After the 8 s element timeout, port 53 to the resolver timed out and `probe_v4` no longer held the source. |
| Revoke without conntrack delete | An established UDP flow for `100.64.1.1` was left in conntrack (`Revoke` empty). Generation 4101 stayed the serving ruleset. In the next window the uplink capture was 0 packets and the return path capture was 0. `ctunkd` rose by 4, matching 4 injected replies. `ctunku` rose with the stream's own uploads (5 on the follow-up 5 run; 4 on the follow-up 8 run, with `ctunkd` 4, injected 4, uplink 0, and return 0). The other device of that member still connected. Decision: the ruleset alone stops the flow, so a later `conntrack` failure keeps the serving ruleset. |

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

## Member-ingress injection

Contract for devstack follow-up 13. The function is `InjectMemberIngress` in package `github.com/TED-H-com/Freedom-VPN/tests/netlab`. The command, run inside the node network namespace, is:

```text
fvpn-netlab inject -iface fvmb0 -src <old-device-ipv4> -dst <origin-ipv4> -port 9
```

`-mechanism` stays empty. The interface type selects the transport: `/sys/class/net/<iface>/type` 1 (`ARPHRD_ETHER`, netlab `fv-fh`) uses `veth-mirred`; type 65534 (`ARPHRD_NONE`, devstack `fvmb0`, tcpdump link-type RAW) uses `tun-mirred`. stdout is one JSON object, `IngressObservation`: `mechanism`, `link_type`, `member_iface`, `source`, `dest`, `source_port` (40000), `dest_port`, and `before` / `after` maps of absolute named counters from `nft -j list table inet fvpn`. The datagram is one UDP packet with payload `x`.

The hook creates lab devices `fv-inj0` and, for ethernet, `fv-inj1`, installs a clsact ingress `mirred redirect` onto the member interface, sends one frame, and deletes those devices before it returns. It reads table `inet fvpn` and does not write it. It does not call `wg`, does not add a peer, and does not change the member interface address, qdisc, or routes. It needs `CAP_NET_ADMIN`. The ethernet transport also needs `CAP_NET_RAW`. The raw transport opens `/dev/net/tun`.

Devstack follow-up 13 copies this lab binary into the node container (`fvpn-<prefix>-node`) and execs it there after the agent has applied the post-revoke ruleset. The revoked device source is `-src`. `-dst` is an origin address beyond the node. Compare `before` and `after` for `ctunku` and the device `u_` / `d_` counters. Leave the peer, group, and binding withdrawn.

`TestIngress_InjectedMemberFrames` is the netlab proof. On ethernet `fv-fh` the hook is `veth-mirred`. A non-admitted source `100.64.9.9` increments `ctunku` and the uplink capture on `fv-ih` stays empty. An admitted source `100.64.1.1` increments `u_019a00000000700080000000000000a1` and the uplink capture shows the forwarded datagram. `nft -s list table inet fvpn`, the `fv-fh` address, and the `fv-fwd` WireGuard peer identity are unchanged.

Mechanism comparison in that same test:

| Transport | Where the frame is delivered | Lab result |
|---|---|---|
| `veth-mirred` (auto on type 1) | Member ingress of `fv-fh` | `ctunku` +1 for `100.64.9.9`; device upload +1 for `100.64.1.1` |
| `tun-mirred` (auto on type 65534) | Forward hook of a lab WireGuard device `fv-raw0` with no peers, counted in side table `inet fvpinj` | Tuple counter +1. `ctunku` on `fv-fh` stays put |
| `veth-mirred` forced onto type 65534 | Same `fv-raw0` forward hook | Tuple counter +1 on this kernel. The hook still selects `tun-mirred` for type 65534 so the injected frame has no ethernet header, matching `fvmb0` |
| `AF_PACKET` send bound to `fv-fh` | Egress toward `fv-ff` | Capture on `fv-ff` shows the frame. `ctunku` +0 |
| Lab WireGuard peer on `fv-lab0` | Ingress of `fv-lab0` only | Side counter +2 for source `100.64.50.2`. `ctunku` on `fv-fh` +0. Usable when a netlab run sets `Datapath.MemberIngress` to that interface. The devstack agent already bound `fvmb0`, and this transport does not place a frame there |

The measured line is `SUMMARY INGRESS fv-fh=veth-mirred type=1 nonadmitted_ctunku=1 nonadmitted_uplink=0 admitted_u_019a00000000700080000000000000a1=1 admitted_uplink=1 afpacket_ctunku=0 raw_tun=1 raw_veth=1 wg_separate=2 wg_ctunku=0`.

## Tables outside `inet fvpn`

`nft -j list ruleset` is the listing `VerifyServing` reads. `loadState` still reads `nft -j list table inet fvpn`, and the renderer still writes only that table. A chain in another table or family is not merged into the `inet fvpn` model.

Every base chain of one hook runs, in priority order. `accept` and `continue` end the current chain only. `drop` and `reject` end the hook. An earlier base chain that only matches, counts, logs, accepts, or continues does not stop a later `inet fvpn` drop, so `VerifyServing` tolerates that chain at any priority. `input` and `output` are host hooks; forwarded member traffic does not cross them, and a drop or a rewrite there is tolerated. Family `arp` is not member IP traffic and is tolerated. An `ingress` or `egress` chain is tolerated when it names a device that is neither the member ingress nor the uplink.

On `prerouting`, `forward`, and `postrouting` (families `ip`, `ip6`, `inet`, and `bridge`), and on `ingress` or `egress` of the member ingress or the uplink (including a chain with no device), any other statement fails closed. That covers a payload rewrite (`ip saddr set`, `ip6 saddr set`), `meta mark` or `ct mark` set, `notrack`, NAT, `queue`, `drop`, `reject`, and a `jump` or `goto` into a chain that does any of those. A base-chain policy of `drop` or `reject` is the same failure. The priority does not matter: an earlier rewrite changes the address `fvpn` matches, and a later drop discards a packet `fvpn` must forward. Both would let a verifier that only saw `inet fvpn` call the node serving.

The reason is one of the existing codes. A mark write is `MARK_SEMANTICS_VIOLATED`. A `flow` or `xt` statement, or a flowtable hooked to member traffic, is `FLOWTABLE_BYPASS`. Anything else in the list is `ISOLATION_POLICY_INVALID`. The detail names the base chain that brings the packet into that table:

```text
table fvobs family inet hook prerouting priority -200 chain pre statement mangle
```

`Apply` installs the `inet fvpn` baseline and returns that error. It does not delete the other table.

The lab proof is `TestForeign_TableBeforeFvpn`. Table `inet fvobs` forward priority -200 policy accept, with one accept rule, left `fvpn-netlab verify` at exit 0. A non-admitted inject of `100.64.9.9` to `203.0.113.10` UDP 39998 incremented `ctunku` and the `fv-ih` capture stayed empty, which is the `inet fvpn` drop still applying. Adding prerouting priority -200 `ip saddr 100.64.9.9 ip saddr set 100.64.1.1` in that same table made verify and apply exit 3 with `ISOLATION_POLICY_INVALID` and the detail above. The serving generation was not left installed (`fvpn_baseline=1`). `inet fvobs` was still present. Deleting that table and applying again returned exit 0.

The measured line is `SUMMARY FOREIGN accept_ctunku=1 accept_uplink=0 verdict=ISOLATION_POLICY_INVALID mangle_exit=3 table_kept=1 recovered=1`.

## Loopback-scoped foreign rules

A foreign rule whose native `match` restricts the packet to loopback cannot act on forwarded member traffic. Forwarded packets do not carry loopback addresses, and the kernel drops a loopback-addressed packet that arrives on a non-loopback interface. `VerifyServing` classifies that rule as `loopback_scoped` and does not fail closed on it.

The proof is read from the ruleset JSON. A native `payload` match on `ip saddr`, `ip daddr`, `ip6 saddr`, or `ip6 daddr` proves scope when the right-hand side is an address, a prefix, a range, or an anonymous set and every address in it is inside `127.0.0.0/8`, or, for IPv6, equal to `::1`. `127.0.0.0/8`, `127.0.0.11`, and `::1` prove scope. `127.0.0.0/7` and `0.0.0.0/0` do not. `meta l4proto` and port matches neither prove scope nor cancel a proof. An `xt` match is an opaque extension and proves nothing. A rule with no native address match proves nothing. Chain and table names prove nothing, including `DOCKER_OUTPUT` and `DOCKER_POSTROUTING`.

`LoopbackEvidence` records each such rule: class `loopback_scoped`, table, family, chain, hook, rule handle, and the proving match. A base chain whose every rule is loopback-scoped is tolerated on a hook member traffic crosses. A `jump` or `goto` is tolerated when every rule of the target is loopback-scoped or otherwise safe under the accept, continue, counter, and log rule above. The jump is still walked when the jumping rule itself is loopback-scoped, so an unscoped rule in the target stays a bypass and the verdict names that rule's handle. An `xt` target, or any other opaque statement, reached through a rule that is not loopback-scoped is still `FLOWTABLE_BYPASS` or the follow-up 10 reason for that statement.

Docker's embedded DNS NAT is this case. On a user-defined network the lab node container has `table ip nat`. Base chain `OUTPUT` (hook `output`, priority -100, policy accept) jumps to `DOCKER_OUTPUT`, whose rules match `ip daddr 127.0.0.11` and carry an `xt` target `DNAT`. Base chain `POSTROUTING` (hook `postrouting`, priority 100, policy accept) matches `ip daddr 127.0.0.11` and jumps to `DOCKER_POSTROUTING`, whose rules match `ip saddr 127.0.0.11` and carry an `xt` target `SNAT`. The lab keeps that NAT. `netlab-up` attaches the container to the `$(FVPN_LAB_PREFIX)-dns` bridge, and `topology.sh` removes the default route that bridge adds. `TestDockerDNS_LoopbackScopedServes` publishes a serving generation with those rules present. A non-admitted inject still increments `ctunku` and leaves the uplink empty.

Production exit nodes must not run Docker, or any other NAT, for non-loopback traffic. A `MASQUERADE` or `xt` `SNAT` of a container subnet is not loopback-scoped and stays `FLOWTABLE_BYPASS`.

The measured line is `SUMMARY LOOPBACK dns=1 serving=1 class=loopback_scoped postrouting_saddr=2 postrouting_daddr=1 ctunku=1 uplink=0`.

## TC serving read-back

`VerifyServing(ctx, executor, model, assignments)` reads live state on every call. Both `Apply` and `fvpn-netlab verify` use it. The nft-only checker is internal; a saved nft listing cannot establish serving. Read-back includes `tc -j qdisc show dev <iface>`, `tc -j class show dev <iface>` and `tc -j filter show dev <iface> ingress` / `egress` on the member interface and uplink. It also reads the root block and every listed qdisc and HTB class filter block, across all chains. The bare filter command only reads the root block; filters attached to an HTB class or fq_codel leaf need their own parent read.

Only the rendered HTB root `1:` and the assigned member fq_codel leaves are owned. Kernel-default root qdiscs (`noqueue`, `pfifo_fast`, `fq_codel`, `fq`, `mq`, `cake`, handle `0:`) without filters are not foreign; this includes `mq`'s default queues under `:N`. Default mq classes and fq_codel flow classes are not parsed as HTB classes. The separate QoS check still requires the serving HTB classes. Every filter is foreign, including `matchall`, `flower`, `u32` and `bpf`, regardless of action. This covers redirect/mirror `mirred`, `pedit`, `nat`, `skbedit`, `bpf` and future filter kinds without an action allowlist. Empty `clsact` and `ingress` qdiscs also fail. Unreadable or malformed tc evidence fails closed.

| Read-back code | Trigger | Result |
|---|---|---|
| `TC_FOREIGN_FILTER` | Foreign qdisc/filter/action, or an incomplete tc read | Serving verification fails; detail names interface, parent/handle and kind. `Apply` returns `Verified=false` and installs the nft baseline. |
| `FLOWTABLE_BYPASS` | Flowtable/offload or unsafe foreign nft extension | Serving verification fails; `Apply` installs the nft baseline. |
| `QOS_CLASS_MISSING` | Missing or mismatched serving HTB class/leaf | Serving verification fails; `Apply` installs the nft baseline. |

`ApplyResult.Evidence` and the lab commands' JSON record tc attachment identities and action kinds, including evidence on rejection. `ip -j -4 rule show` and `ip -j -6 rule show` are recorded as family, rule count and scan status only. Rule selectors, destinations and action parameters are omitted. Policy rules and policy-scan failures are informational; the policy-routing control below still passes admission checks.

`TestReviewR15_TcMirredAndPolicyRoutingBypass/tc_clsact_mirred` measured a `clsact`/`matchall`/`mirred egress redirect` from `fv-fh` to `fv-ih`. Before the fix, verify exited 0 while one non-admitted UDP packet reached the uplink and `ctunku` stayed zero. After the fix, the same trigger remains observable but verify and apply exit 3 with `TC_FOREIGN_FILTER`, naming `fv-fh`; apply records `verified=false` and the nft baseline is present. The held `ip_rule` subtest adds a source rule and route in table 100: verify exits 0, the rule scan is recorded, `ctunku` increases by one and the uplink capture is empty. `TestReviewR15_TCFilterAttachments` also rejects filters on root, HTB parent/host/member classes, fq_codel leaves and egress on both interfaces, including nonzero filter chain 7.

This is a serving-gate check, not continuous surveillance. A privileged process can change tc after a read. The existing nft baseline cannot stop a retained ingress redirect that bypasses nft, and this change does not delete foreign `clsact` state. The agent-core owner must decide the re-verification cadence before a serving heartbeat and how to quarantine a path with a retained foreign redirect. Until then, the next serving apply detects it; no between-apply protection is claimed. The current agent observation maps the new reason to code `HEALTH_CHECK_FAILED` with message `TC_FOREIGN_FILTER` until the HTTP contract gains the enum.

## Decisions

- D-019: usage is the inner IPv4 total length admitted at the forward hook (§10.2). Bytes later dropped by HTB are not subtracted. "Delivered bytes" is not a ledger rule. The lab counters are the meter definition for the node agent.
- Serving forward rules bind the member interface and the uplink. Upload uses `oifname` uplink. Download uses `iifname` uplink. The baseline drop does not.
- A base chain outside `inet fvpn` that only accepts, continues, counts, or logs does not bypass the `fvpn` drop. A chain on a hook member traffic crosses that rewrites, marks, offloads, or drops fails `VerifyServing` closed. The agent still writes only `inet fvpn`. The measurement is `## Tables outside inet fvpn`.
- D-079: a foreign rule whose native match restricts the packet to `127.0.0.0/8` or `::1` is `loopback_scoped` and is not a bypass. An `xt` match does not prove scope. Production exit nodes must not run Docker or any other NAT for non-loopback traffic. A container-subnet `MASQUERADE` stays `FLOWTABLE_BYPASS`. The measurement is `## Loopback-scoped foreign rules`.
- D-063: the synthetic probe is an optional `Model.Probe`. Nil removes it through the same render path. The kernel element timeout is the authorization, capped by `ProbeMaxSeconds` and by `TimeoutCeilSeconds` or the remaining lifetime already in the kernel. At most four destinations, none overlapping a member binding. `Verify` fails closed on a wide prefix, an unparsed element, an original timeout above the ceiling, or a remaining lifetime past the model window. `probe_up` and `probe_down` are not member usage. The agent authenticates the test peer before it supplies the `/32`.
- Clock-step expiry (§16.3) stays under 待實測. The kernel set-timeout evidence is accepted for Gate G2. This lab does not call `date -s` and does not change the host clock.
- §11.7 production windows remain a recommended starting point. The short window is what CI runs.

## 待實測

The local `clsact` mirred bypass and its D-085 serving rejection are now measured, along with the held local policy-routing case, in `TestReviewR15_TcMirredAndPolicyRoutingBypass`. This does not establish real NetBird routing compatibility or continuous detection between applies.

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
