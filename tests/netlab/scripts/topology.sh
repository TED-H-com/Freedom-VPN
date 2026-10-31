#!/bin/bash
# Gate G2 lab topology (§8.2). Runs only inside the lab container.
# Names start with fv. Forwarding stays off until the boot baseline is installed.
set -euo pipefail

log() { printf 'topology: %s\n' "$*" >&2; }

nsdo() {
	local ns="$1"
	shift
	if [[ "$ns" == root ]]; then
		"$@"
	else
		ip netns exec "$ns" "$@"
	fi
}

rp_off() {
	local ns="$1"
	nsdo "$ns" sh -c 'for i in /proc/sys/net/ipv4/conf/*/rp_filter; do echo 0 > "$i"; done'
}

offload_off() {
	local ns="$1" dev="$2"
	nsdo "$ns" ethtool -K "$dev" gro off gso off tso off >/dev/null 2>&1 || true
}

# ns_a if_a addr_a ns_b if_b addr_b mtu
veth_pair() {
	local nsa="$1" ifa="$2" addra="$3" nsb="$4" ifb="$5" addrb="$6" mtu="$7"
	ip link add "$ifa" type veth peer name "$ifb"
	if [[ "$nsa" != root ]]; then ip link set "$ifa" netns "$nsa"; fi
	if [[ "$nsb" != root ]]; then ip link set "$ifb" netns "$nsb"; fi
	nsdo "$nsa" ip link set "$ifa" up mtu "$mtu"
	nsdo "$nsa" ip addr add "$addra" dev "$ifa"
	nsdo "$nsb" ip link set "$ifb" up mtu "$mtu"
	nsdo "$nsb" ip addr add "$addrb" dev "$ifb"
	offload_off "$nsa" "$ifa"
	offload_off "$nsb" "$ifb"
}

SCRATCH=/scratch
mkdir -p "$SCRATCH/keys" "$SCRATCH/log"
chmod 700 "$SCRATCH/keys"
umask 077

# Host AppArmor profile "wg" attaches to /usr/bin/wg and denies reading key
# files outside /etc/wireguard. /usr/local/bin/wg is the same binary, unconfined.
if [[ ! -x /usr/local/bin/wg ]]; then
	cp -a /usr/bin/wg /usr/local/bin/wg
fi
WG=/usr/local/bin/wg

log "forwarding off before any path exists"
sysctl -w net.ipv4.ip_forward=0 net.ipv6.conf.all.forwarding=0 net.ipv6.conf.default.forwarding=0 >/dev/null

for ns in fv-fwd fv-a1 fv-a2 fv-b1 fv-rogue fv-probe fv-inet fv-lan; do
	ip netns add "$ns"
	ip netns exec "$ns" ip link set lo up
	ip netns exec "$ns" sysctl -w net.ipv4.ip_forward=0 net.ipv6.conf.all.forwarding=0 >/dev/null
done

log "underlay and transit links"
# Member underlay /30s use 192.0.2.0/24 documentation space. MTU 9000 so the
# encrypted outer packet is not fragmented. Inner WG MTU is 1420.
veth_pair fv-a1 fv-a1u 192.0.2.2/30 fv-fwd fv-fa1 192.0.2.1/30 9000
veth_pair fv-a2 fv-a2u 192.0.2.6/30 fv-fwd fv-fa2 192.0.2.5/30 9000
veth_pair fv-b1 fv-b1u 192.0.2.10/30 fv-fwd fv-fb1 192.0.2.9/30 9000
veth_pair fv-rogue fv-rgu 192.0.2.14/30 fv-fwd fv-frg 192.0.2.13/30 9000
# Dedicated synthetic-probe peer. Not a member. 100.64.8.2 is the same /24
# and is routed here so the nft source match, not WireGuard, is what drops it.
veth_pair fv-probe fv-pru 192.0.2.18/30 fv-fwd fv-fpr 192.0.2.17/30 9000
# Host side of the routing-peer link. Upload arrives here.
veth_pair root fv-fh 192.0.2.33/30 fv-fwd fv-ff 192.0.2.34/30 1500
nsdo root ip -6 addr add fd00:db8:ff::33/64 dev fv-fh nodad
nsdo fv-fwd ip -6 addr add fd00:db8:ff::34/64 dev fv-ff nodad
# Internet side. SNAT address is 192.0.2.37.
veth_pair root fv-ih 192.0.2.37/30 fv-inet fv-ii 192.0.2.38/30 1500
nsdo root ip -6 addr add fd00:db8:17::37/64 dev fv-ih nodad
nsdo fv-inet ip -6 addr add fd00:db8:17::38/64 dev fv-ii nodad
nsdo fv-inet ip addr add 203.0.113.10/32 dev fv-ii
nsdo fv-inet ip addr add 203.0.113.20/32 dev fv-ii
nsdo fv-inet ip addr add 198.51.100.53/32 dev fv-ii
nsdo fv-inet ip -6 addr add 2001:db8:203::10/128 dev fv-ii nodad
# Host LAN, including a public-prefix private address (§8.1).
veth_pair root fv-lh 192.168.0.1/24 fv-lan fv-ll 192.168.0.2/24 1500
nsdo root ip -6 addr add fd00:db8:ffff::1/64 dev fv-lh nodad
nsdo fv-lan ip -6 addr add fd00:db8:ffff::10/64 dev fv-ll nodad
nsdo fv-lan ip -6 addr add fe80::10/64 dev fv-ll nodad
nsdo fv-lan ip addr add 192.168.10.10/32 dev fv-ll
nsdo fv-lan ip addr add 10.1.2.10/32 dev fv-ll
nsdo fv-lan ip addr add 172.16.5.10/32 dev fv-ll
nsdo fv-lan ip addr add 169.254.169.254/32 dev fv-ll
nsdo fv-lan ip addr add 169.254.1.50/32 dev fv-ll
nsdo fv-lan ip addr add 192.0.2.200/32 dev fv-ll
nsdo fv-lan ip -6 addr add 2001:db8::10/128 dev fv-ll nodad

log "routes"
# A user-defined network installs a default route via eth0 and Docker's
# embedded DNS NAT for 127.0.0.11. The NAT rules stay. The default route is
# not a lab path.
while ip route show default | grep -q .; do
	ip route del default
done
while ip -6 route show default | grep -q .; do
	ip -6 route del default
done
ip route add 100.64.0.0/10 via 192.0.2.34 dev fv-fh
ip route add 203.0.113.0/24 via 192.0.2.38 dev fv-ih
ip route add 198.51.100.53/32 via 192.0.2.38 dev fv-ih
ip route add 192.168.10.10/32 via 192.168.0.2 dev fv-lh
ip route add 10.1.2.10/32 via 192.168.0.2 dev fv-lh
ip route add 172.16.5.10/32 via 192.168.0.2 dev fv-lh
ip route add 169.254.169.254/32 via 192.168.0.2 dev fv-lh
ip route add 169.254.1.50/32 via 192.168.0.2 dev fv-lh
ip route add 192.0.2.200/32 via 192.168.0.2 dev fv-lh
ip -6 route add 2001:db8:203::10/128 via fd00:db8:17::38 dev fv-ih
ip -6 route add 2001:db8::10/128 via fd00:db8:ffff::10 dev fv-lh
ip -6 route add fd00:db8:a1::/64 via fd00:db8:ff::34 dev fv-fh

nsdo fv-inet ip route add default via 192.0.2.37 dev fv-ii
nsdo fv-inet ip -6 route add default via fd00:db8:17::37 dev fv-ii
nsdo fv-lan ip route add default via 192.168.0.1 dev fv-ll
nsdo fv-lan ip -6 route add default via fd00:db8:ffff::1 dev fv-ll

log "wireguard keys in /scratch (not the repo)"
for name in fwd a1 a2 b1 rogue probe; do
	"$WG" genkey | tee "$SCRATCH/keys/$name" | "$WG" pubkey >"$SCRATCH/keys/$name.pub"
done
chmod 600 "$SCRATCH/keys"/*

nsdo fv-fwd ip link add fv-wg0 type wireguard
nsdo fv-fwd "$WG" set fv-wg0 listen-port 51820 private-key "$SCRATCH/keys/fwd" \
	peer "$(cat "$SCRATCH/keys/a1.pub")" allowed-ips 100.64.1.1/32,fd00:db8:a1::1/128 \
	peer "$(cat "$SCRATCH/keys/a2.pub")" allowed-ips 100.64.1.2/32 \
	peer "$(cat "$SCRATCH/keys/b1.pub")" allowed-ips 100.64.2.1/32 \
	peer "$(cat "$SCRATCH/keys/rogue.pub")" allowed-ips 100.64.9.9/32 \
	peer "$(cat "$SCRATCH/keys/probe.pub")" allowed-ips 100.64.8.1/32,100.64.8.2/32
nsdo fv-fwd ip addr add 100.64.0.1/32 dev fv-wg0
nsdo fv-fwd ip -6 addr add fd00:db8:f::1/128 dev fv-wg0 nodad
nsdo fv-fwd ip link set fv-wg0 up mtu 1420

setup_member() {
	local ns="$1" key="$2" endpoint="$3" addr="$4" v6="${5:-}"
	nsdo "$ns" ip link add fv-wg0 type wireguard
	nsdo "$ns" "$WG" set fv-wg0 private-key "$SCRATCH/keys/$key" \
		peer "$(cat "$SCRATCH/keys/fwd.pub")" \
		endpoint "${endpoint}:51820" \
		allowed-ips 0.0.0.0/0,::/0 \
		persistent-keepalive 0
	nsdo "$ns" ip addr add "$addr" dev fv-wg0
	if [[ -n "$v6" ]]; then
		nsdo "$ns" ip -6 addr add "$v6" dev fv-wg0 nodad
	fi
	nsdo "$ns" ip link set fv-wg0 up mtu 1420
	nsdo "$ns" ip route add default dev fv-wg0
	nsdo "$ns" ip -6 route add default dev fv-wg0
	offload_off "$ns" fv-wg0
}

setup_member fv-a1 a1 192.0.2.1 100.64.1.1/32 fd00:db8:a1::1/128
setup_member fv-a2 a2 192.0.2.5 100.64.1.2/32
setup_member fv-b1 b1 192.0.2.9 100.64.2.1/32
setup_member fv-rogue rogue 192.0.2.13 100.64.9.9/32
setup_member fv-probe probe 192.0.2.17 100.64.8.1/32
nsdo fv-probe ip addr add 100.64.8.2/32 dev fv-wg0

# Member-originated packets go to the host. Packets from the host go back
# into the matching tunnel. No hairpin inside the routing namespace (§8.2).
nsdo fv-fwd ip route add 100.64.1.1/32 dev fv-wg0 table 100
nsdo fv-fwd ip route add 100.64.1.2/32 dev fv-wg0 table 100
nsdo fv-fwd ip route add 100.64.2.1/32 dev fv-wg0 table 100
nsdo fv-fwd ip route add 100.64.9.9/32 dev fv-wg0 table 100
nsdo fv-fwd ip route add 100.64.8.1/32 dev fv-wg0 table 100
nsdo fv-fwd ip route add 100.64.8.2/32 dev fv-wg0 table 100
nsdo fv-fwd ip -6 route add fd00:db8:a1::1/128 dev fv-wg0 table 100
nsdo fv-fwd ip route add default via 192.0.2.33 dev fv-ff table 200
nsdo fv-fwd ip route add 100.64.0.0/10 via 192.0.2.33 dev fv-ff table 200
nsdo fv-fwd ip -6 route add default via fd00:db8:ff::33 dev fv-ff table 200
nsdo fv-fwd ip rule add iif fv-wg0 lookup 200 priority 50
nsdo fv-fwd ip rule add iif fv-ff lookup 100 priority 50
nsdo fv-fwd ip -6 rule add iif fv-wg0 lookup 200 priority 50
nsdo fv-fwd ip -6 rule add iif fv-ff lookup 100 priority 50
nsdo fv-fwd ip route add default via 192.0.2.33 dev fv-ff
nsdo fv-fwd ip -6 route add default via fd00:db8:ff::33 dev fv-ff

for ns in root fv-fwd fv-a1 fv-a2 fv-b1 fv-rogue fv-probe fv-inet fv-lan; do
	rp_off "$ns"
done

log "boot baseline before forwarding (§8.2)"
/usr/local/bin/fvpn-netlab baseline

sysctl -w net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1 net.ipv6.conf.default.forwarding=1 >/dev/null
nsdo fv-fwd sysctl -w net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1 net.ipv6.conf.default.forwarding=1 >/dev/null
# Per-interface IPv6 forwarding follows "all" only for interfaces that exist now.
nsdo root sysctl -w net.ipv6.conf.fv-fh.forwarding=1 net.ipv6.conf.fv-ih.forwarding=1 net.ipv6.conf.fv-lh.forwarding=1 >/dev/null
nsdo fv-fwd sysctl -w net.ipv6.conf.fv-ff.forwarding=1 net.ipv6.conf.fv-wg0.forwarding=1 >/dev/null

log "listeners"
nsdo fv-inet python3 /opt/netlab/scripts/services.py inet >/scratch/log/inet.out 2>&1 &
echo $! >/scratch/inet.pid
nsdo fv-lan python3 /opt/netlab/scripts/services.py lan >/scratch/log/lan.out 2>&1 &
echo $! >/scratch/lan.pid
nsdo root python3 /opt/netlab/scripts/services.py host >/scratch/log/host.out 2>&1 &
echo $! >/scratch/host.pid
for ns in fv-a1 fv-a2 fv-b1; do
	nsdo "$ns" python3 /opt/netlab/scripts/services.py member >/scratch/log/"$ns".out 2>&1 &
	echo $! >/scratch/"$ns".pid
done
for port in 5201 5202 5203 5204; do
	nsdo fv-inet iperf3 -s -p "$port" -D
done

for i in 1 2 3 4 5 6 7 8 9 10; do
	if [[ -f /scratch/svc-inet.ready && -f /scratch/svc-lan.ready && -f /scratch/svc-host.ready ]]; then
		break
	fi
	sleep 0.2
done
[[ -f /scratch/svc-inet.ready && -f /scratch/svc-lan.ready && -f /scratch/svc-host.ready ]]

log "wireguard handshakes"
ready=0
for attempt in 1 2 3 4 5 6 7 8; do
	nsdo fv-a1 ping -c 1 -W 1 100.64.0.1 >/dev/null 2>&1 || true
	nsdo fv-a2 ping -c 1 -W 1 100.64.0.1 >/dev/null 2>&1 || true
	nsdo fv-b1 ping -c 1 -W 1 100.64.0.1 >/dev/null 2>&1 || true
	nsdo fv-rogue ping -c 1 -W 1 100.64.0.1 >/dev/null 2>&1 || true
	nsdo fv-probe ping -c 1 -W 1 100.64.0.1 >/dev/null 2>&1 || true
	count=$(nsdo fv-fwd "$WG" show fv-wg0 latest-handshakes | awk '$2 > 0 { n++ } END { print n+0 }')
	if [[ "$count" == 5 ]]; then
		ready=1
		break
	fi
	sleep 0.3
done
if [[ "$ready" != 1 ]]; then
	log "handshake failed"
	nsdo fv-fwd "$WG" show fv-wg0 | awk '{ print $1 }' >&2 || true
	exit 1
fi

date -u +%Y-%m-%dT%H:%M:%SZ >/scratch/topology.ok
log "ready"
