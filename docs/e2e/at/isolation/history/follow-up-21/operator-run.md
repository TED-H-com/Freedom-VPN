# Operator run: platforms this host cannot exercise

Status: needs operator run. Do not mark these steps passed from the lab or the dev stack.

The dev stack has one Linux member container. It does not boot Windows, macOS, iOS or Android, and this host is not the candidate production machine. Physical arm64 and a second physical NAT are also absent.

Run the same checks on each platform after a node is serving (publication observed active, forward generation at least 1, the device binding present). Record UTC time, build, image digest, policy revision and node incarnation. Do not include tokens, setup keys or home paths.

1. Install the member client and connect to the published exit.
2. Confirm the default route for probed destinations is the tunnel interface. A hit on the local LAN interface does not count.
3. From that tunnel, attempt TCP to an RFC1918 host, a CGNAT address that is not the member, the metadata address 169.254.169.254, and the node's own management addresses. Each must fail. The controlled Internet endpoint must succeed.
4. While those probes run at least ten times a second, add and remove a dummy interface on the node. No probe may succeed during the change.
5. On a dual-stack network, repeat for IPv6 in the selected mode. v1 on the node is block (§8.3). ULA, link-local, loopback and IPv4-mapped addresses must not bypass that mode.
6. With two members and two devices of one member, confirm peer addresses, SSH, the agent API and any port other than 53 on the designated resolvers fail. TCP and UDP 53 to those resolvers succeed. A spoofed source must not use another member's mapping.
7. Hold a long TCP and UDP flow. Publish a revocation. The flow must stop and its conntrack entry must disappear. Repeat the forbidden probes across agent SIGKILL, a NetBird restart, a node reboot and an updater switch. Forwarding must stay off until a valid policy is held.

Expected evidence: the connection matrix, packet captures, nft ruleset, conntrack listing and a timeline. Store it next to the other pages in this directory, with secrets removed.
