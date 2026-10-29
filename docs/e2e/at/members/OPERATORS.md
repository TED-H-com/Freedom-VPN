# Physical and native-client runs

Status for every row below: **needs operator run**. No platform, physical arm64
or second-NAT capability is claimed by the live Linux lab tests.
References: §19 AT01, AT05, AT09, AT21; §7.3, §8.3, §12.6.

## Inputs and preparation

Use a reachable controlled TLS test environment with the same candidate builds
and policy as the lab, two exits, two filtering resolvers, controlled HTTP echo,
TCP/UDP/QUIC endpoints and controlled DoT/DoH endpoints. Do not expose this
loopback devstack on the host's public interfaces. The operator records the
private endpoint mapping separately; public evidence uses `example.invalid`
and documentation address aliases. Never send tests to public IP-echo or DNS
services.

For each run record UTC, tester pseudonym, device pseudonym, OS/build/CPU, App
version/build and embedded core, control/worker/node/server/resolver image
digests, source commit, actual applied policy revision and node incarnation
pseudonyms. Use TEST ONLY accounts A/B and clean App settings. Pin the actual
candidate version and photograph its About screen; if the UI differs from the
repository [G1 script](../../../poc/G1-app-checklist.md), record the difference
and actual control used. Do not assume a newer store version has been tested.

## AT01/AT09 per-platform script

Run steps 1–12 independently on Windows, macOS, iOS and Android. Give each step
a UTC time, expected/actual result and redacted screenshot/trace identifier.

| Platform | Server configuration | Exit control | Sleep/network action |
|---|---|---|---|
| Windows | Clean Self-hosted profile, Settings/About evidence | Main/tray Exit Nodes selector | Start → Power → Sleep for 2 minutes; wake; change between two controlled Wi-Fi networks. |
| macOS | Clean Self-hosted profile, Settings/About evidence | Main/menu-bar Exit Nodes selector | Sleep for 2 minutes; wake; change between two controlled Wi-Fi networks. |
| iOS | Settings → Change Server; enter controlled URL, no setup key | Connection → Exit node | Lock for 2 minutes, then wake; change Wi-Fi networks. Record lock/wake as mobile suspension, not proof of desktop-style sleep. |
| Android | Settings → Profiles → add; Management Server → Self-hosted; no setup key | Home → Exit node | Lock/power-save for 2 minutes, wake; change Wi-Fi networks; record battery optimization settings. |

1. Confirm the candidate OS/App/core versions and fresh configuration. Enter
   the controlled management URL and connect using browser SSO as A. Verify
   issuer and client audience in private evidence without publishing tokens.
2. Log into Portal as A through its separate OAuth client. The operator must
   verify the stable cross-client identity proof and server member/device ↔
   actual NetBird user/peer mapping. Equal emails/subjects alone do not pass.
3. Before device admission, prove the controlled endpoint is healthy by a
   separate authorized control, while this peer cannot forward. Then wait for
   device quota admission, paid-group readback and matching node policy ACK.
4. Record two exits visible with neither automatically selected. Select exit A.
   Read controlled echo in the normal browser and compare actual egress with A.
   Record native selected exit, server `/member/exits` view and device mapping.
5. Switch manually to B; verify echo egress B. Deselect exit and record original
   route behaviour. Do not describe exit order as a speed measurement.
6. Select A; sign out of Portal, verify session ended and authenticated API
   access denied. Record native session separately. Sign out of native App,
   reconnect by SSO, and verify unchanged correct ownership/quota semantics.
7. With an exit selected, perform the platform sleep/lock action in the table.
   On wake recheck selected exit, echo, DNS and mapping; record recovery seconds.
8. Change to the second controlled network and back. Verify mapping, selected
   exit, DNS and echo after both transitions. Keep the application open, then
   repeat with background suspension. Record any unsupported required step.
9. While A is selected, operator stops A in the isolated test environment.
   Record UI error, route behaviour, existing/new TCP/UDP outcomes and any
   original-network fallback. Restore A, explicitly select B and verify egress.
10. Repeat A/B over verified direct and forced-relay paths and across the
    second physical NAT. Save per-peer connection type, NAT layout, underlay,
    RTT/loss and fixed-duration controlled TCP/UDP/QUIC throughput. Probe
    1200/1280/1360/1400/1472-byte payloads, PMTU feedback and MSS; record the
    limits instead of configuring one guessed MTU. Missing fixture = blocked.
11. Operator renames/regroups/disables the selected exit via API/UI. Record
    pending then applied evidence, lost access for the old group, actual flow
    denial, and disabled behaviour after App and node restart.
12. Repeat first login with B and prove no A-only exit or A-device data leaks.
    Remove the TEST ONLY profile and restore the operator's original settings.

Submit a JSON and short page per platform referencing these step IDs. Aggregate
AT01/AT09 cannot pass while any required platform row is missing or failed.
Screenshots are redacted before sharing; no raw debug bundles are published.

## AT05 physical arm64 script

Use three owner-provided clean Ubuntu 24.04 arm64 machines/images (preinstalled,
cloud-init and operator-run installer). Record actual `uname -m`, OS release,
hardware pseudonym, boot image and installer/release digest on each. If any
host is emulated, label it and leave physical hardware acceptance pending.

1. Verify the preinstalled/cloud master contains no generated Agent/NetBird
   private identity or reusable token. Record presence/absence only.
2. Issue a different single-use enrollment through the same admin endpoint for
   each machine. Inject each token through the documented secret-file channel;
   do not put it in command history or evidence. Record default 7-day
   preinstalled or 15-minute installer expiry as appropriate.
3. Boot preinstalled, run the cloud-init template, and run the documented
   installer on the third machine. Record agent/package hashes and the same
   enrollment/bootstrap/observation sequence. Do not invent an installer command
   if the candidate package has no supported arm64 installation instructions.
4. Wait for registered/ready/unpublished and prove no forwarding. Check that
   enrollment needed no second human admission action. Replay the same request
   only through the test hook: same incarnation, no duplicate peer; bootstrap
   already consumed. Private key must stay on its machine.
5. Classify and publish all three through identical Admin UI/API actions.
   Record 202/pending, matching policy ACK, NetBird route readback and real
   controlled forwarding for each. Revoke test enrollment/identity and clean up
   according to the isolated test environment's operator procedure.

## AT21 Private DNS / DoH script

Android baseline: record original system Private DNS setting and browser secure
DNS setting. Set Android Settings → Network & internet → Private DNS → Off
(record manufacturer-specific path), and disable custom browser DoH. Resolve a
unique synthetic blocked name through the normal browser; require the controlled
filter's response and verify the intended resolver path with scoped counters.

Set Private DNS provider hostname to the **controlled** DoT fixture. Repeat and
record successful bypass or connection failure, actual path and UI message.
Restore Off and verify the blocked baseline again. Separately enable custom
browser DoH using the controlled DoH fixture; repeat, record path/outcome and
restore. Repeat browser DoH on Windows/macOS/iOS where the candidate supports
it; unsupported configuration must be recorded, not assumed equivalent.

Stop each controlled filtering resolver in turn, then both, and separately
interrupt policy sync. Record system resolution, browser resolution and
fallback destination. Restore each fault; require only the other controlled
resolver as fallback, and declared failure if both unavailable. Do not use a
public resolver for a negative control. Save resolver hashes/partial publication
status and the guidance presented to the member.

Proposed member guidance (zh-TW; acceptance draft, not a claim already verified):

> 本服務提供網域層級的 DNS 防護，不解密 HTTPS，也不能依網址路徑過濾內容。
> Android「私人 DNS」或瀏覽器自訂 DoH 可能繞過 VPN 指派的 DNS，或造成
> 解析失敗。使用本服務的網域防護時，請先將私人 DNS 設為「關閉」，並關閉
> 瀏覽器自訂安全 DNS，再依測試指引確認解析路徑。介面位置依系統版本而異。
> 服務的備援僅使用受控解析器；全部無法使用時可能暫停解析，不會默默切換
> 至未過濾的公共 DNS。未受管裝置上的所有 DoH、自訂隧道及直接 IP 連線
> 無法保證全面攔截。節點的會員隔離也不代表裝置在隧道中斷時具有完整
> kill switch。服務保留必要資格、用量與安全紀錄，不宣稱完全 no-log。

English equivalent: Domain protection operates at DNS level, without HTTPS
decryption or URL-path inspection. Custom Android Private DNS or browser DoH
may bypass VPN DNS or fail to resolve. Disable conflicting settings and verify
the resolver path. Only controlled resolvers provide fallback; if all are
unavailable, resolution can fail. Unmanaged devices can use paths this protection
cannot fully control. Server isolation is not an endpoint kill switch. Necessary
entitlement, usage and security records remain; this is not a complete no-log
service.
