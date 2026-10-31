#!/bin/sh
# Node container entrypoint (§5, §8, §13). Creates only fv* interfaces inside
# this container. Never prints the enrollment token or the setup key.
set -eu
umask 077
mkdir -p /run/vpn-node /var/lib/vpn-node/state /var/lib/vpn-node/identity \
    /var/lib/vpn-node/update-journal /var/lib/vpn-node/e2e /opt/vpn-node /var/log/netbird /etc/vpn-node
# Rename only this container's origin-facing interface. The real NetBird
# interface is fvmb0; no dummy interface can prove member forwarding.
if ! ip link show fvup0 >/dev/null 2>&1; then
    uplink=$(ip -o -4 addr show | awk '$4 ~ /^192[.]0[.]/ {print $2; exit}')
    test -n "$uplink"
    ip link set "$uplink" down
    ip link set "$uplink" name fvup0
    ip link set fvup0 up
fi
# Docker's embedded DNS installs iptables-nft xt DNAT/SNAT for 127.0.0.11 in
# this netns. Whole-ruleset read-back treats an xt target reachable from
# postrouting as FLOWTABLE_BYPASS. inet fvpn is a different table and is left
# alone. AdGuard's upstream is CoreDNS by address, so this netns does not
# need 127.0.0.11. Pin the control-plane names before enrollment. A hostname
# already in /etc/hosts wins. The clone's lab-hosts names the presentation
# /24; this loop must not replace that line with the lab address. This file
# is the process container's hosts file. --network container: does not share
# it with the netns container. compose extra_hosts and docker --add-host
# both conflict with that network mode, so neither is used.
nft delete table ip nat >/dev/null 2>&1 || true
if [ -s /etc/vpn-node/lab-hosts ]; then
    while read -r ip name _; do
        [ -n "${name:-}" ] || continue
        case "$ip" in \#*) continue ;; esac
        if ! awk -v n="$name" 'BEGIN{f=1} {for (i=2;i<=NF;i++) if ($i==n) {f=0; exit}} END{exit f}' /etc/hosts; then
            printf '%s %s\n' "$ip" "$name" >> /etc/hosts
        fi
    done < /etc/vpn-node/lab-hosts
fi
if [ -f /etc/vpn-node/ca.pem ]; then
    cp /etc/vpn-node/ca.pem /usr/local/share/ca-certificates/lab-ca.crt
    update-ca-certificates >/dev/null 2>&1 || true
fi
export SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
# D-061 refuses credentials unless the regular file is root-owned mode 0600.
chown 0:0 /etc/vpn-node/resolver-credentials.json
chmod 600 /etc/vpn-node/resolver-credentials.json
# TEST ONLY durable OS identity: distinct fresh slots; cloned images preserve it.
if [ ! -s /etc/vpn-node/machine-id ]; then
    tr -d '-' < /proc/sys/kernel/random/uuid > /etc/vpn-node/machine-id
fi
cp /etc/vpn-node/machine-id /etc/machine-id
cat /proc/sys/kernel/random/uuid > /etc/vpn-node/boot-id
epoch=$(cat /proc/sys/kernel/random/uuid)
epoch2=$(cat /proc/sys/kernel/random/uuid)
printf '{"closed_meter_epoch_id":"%s","last_durable_event_seq":"0","central_acked_through":"0","new_meter_epoch_id":"%s"}\n' "$epoch" "$epoch2" > /var/lib/vpn-node/e2e/meter.json
# A durable identity resumes through the normal fail-closed runtime. Do not
# request or consume another installer token on a container restart. A partial
# identity is an error, never permission to silently enroll a second incarnation.
if [ -s /var/lib/vpn-node/identity/node.json ]; then
    if [ ! -s /var/lib/vpn-node/identity/cert.pem ] || [ ! -s /var/lib/vpn-node/identity/key.pem ]; then
        printf '%s\n' '{"stage":"identity_incomplete"}' > /var/lib/vpn-node/e2e/status.json
        exit 1
    fi
    printf '%s\n' '{"stage":"resuming"}' > /var/lib/vpn-node/e2e/status.json
else
    printf '%s\n' '{"stage":"waiting_token"}' > /var/lib/vpn-node/e2e/status.json
    while [ ! -s /var/lib/vpn-node/e2e/enrollment.token ]; do
        sleep 1
    done
    printf '%s\n' '{"stage":"enrolling"}' > /var/lib/vpn-node/e2e/status.json
    if ! fvpn-agent enroll -config /etc/vpn-node/agent-enroll.json -token-file /var/lib/vpn-node/e2e/enrollment.token; then
        printf '%s\n' '{"stage":"enroll_failed"}' > /var/lib/vpn-node/e2e/status.json
        exit 1
    fi
fi
# PIDs belong to the old container process tree, not the durable identity.
rm -f /var/lib/vpn-node/e2e/agent.pid
fvpn-e2e-nodeconfig
# TEST ONLY loopback mTLS transport; corrupts only when the private local marker is armed.
fvpn-e2e-nodeconfig resolver-proxy >/var/lib/vpn-node/e2e/resolver-proxy.log 2>&1 &
printf '%s\n' '{"stage":"enrolled"}' > /var/lib/vpn-node/e2e/status.json
fvpn-watchdog run -config /etc/vpn-node/agent-run.json >/var/lib/vpn-node/e2e/watchdog.log 2>&1 &
# TEST ONLY supervisor. This loop is the only owner of fvpn-agent. Public
# agent-service.sh stop/start only toggle pause-agent and wait. Poll is 0.2s
# so a stop/start cycle does not wait out the old 2s period; a slower poll
# still converges because the public commands wait across more than one poll.
# Every iteration reaps while paused and respawns while running, including
# after a crash. --supervisor must not call the public wait path.
(
    while true; do
        if [ -f /var/lib/vpn-node/e2e/pause-agent ]; then
            agent-service.sh --supervisor stop || true
        else
            selected=/usr/local
            if [ -f /var/lib/vpn-node/e2e/agent-selected ]; then
                selected=$(cat /var/lib/vpn-node/e2e/agent-selected)
            fi
            agent-service.sh --supervisor start "$selected" || true
        fi
        sleep 0.2
    done
) &
(
    netbird service run --log-level warn --log-file console >/var/lib/vpn-node/e2e/netbird-service.log 2>&1 &
    ready=0
    i=0
    while [ "$i" -lt 40 ]; do
        if netbird status --check live >/dev/null 2>&1; then
            ready=1
            break
        fi
        i=$((i + 1))
        sleep 1
    done
    if [ "$ready" -ne 1 ]; then
        printf '%s\n' '{"stage":"netbird_daemon_failed"}' > /var/lib/vpn-node/e2e/netbird.json
        exit 0
    fi
    while [ ! -s /var/lib/vpn-node/identity/setup.key ]; do
        sleep 1
    done
    if netbird up --setup-key-file /var/lib/vpn-node/identity/setup.key \
        --management-url https://netbird.example.invalid:443 \
        --interface-name fvmb0 --disable-dns --disable-ipv6 --disable-firewall \
        --log-level warn --log-file console >/var/lib/vpn-node/e2e/netbird-up.log 2>&1; then
        printf '%s\n' '{"stage":"netbird_up"}' > /var/lib/vpn-node/e2e/netbird.json
    else
        printf '%s\n' '{"stage":"netbird_failed"}' > /var/lib/vpn-node/e2e/netbird.json
    fi
) &
(
    while true; do
        if [ -f /var/lib/vpn-node/e2e/updater.json ]; then
            set +e
            fvpn-updater run -config /var/lib/vpn-node/e2e/updater.json >/var/lib/vpn-node/e2e/updater.out 2>&1
            code=$?
            set -e
            printf '{"exit_code":%s}\n' "$code" > /var/lib/vpn-node/e2e/updater-result.json
            mv /var/lib/vpn-node/e2e/updater.json /var/lib/vpn-node/e2e/updater.json.done
        fi
        sleep 2
    done
) &
wait
