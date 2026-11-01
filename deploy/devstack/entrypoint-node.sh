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
# Containers restart processes, not the kernel. Use the same boot identity as
# the packaged updater; a process restart must not impersonate an OS reboot.
cat /proc/sys/kernel/random/boot_id > /etc/vpn-node/boot-id
# The real agent writes state/meter.checkpoint after the updater's D-125
# withdrawal barrier. The fixture never fabricates a meter checkpoint.
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
    # TEST ONLY: a retained snapshot may already contain a real pending key,
    # CSR and incarnation after an offline CLI attempt. Keep the API-issued
    # token unconsumed until its mid-run boot resumes that same request.
    if [ -f /var/lib/vpn-node/e2e/hold-enrolling ]; then
        printf '%s\n' '{"stage":"enrolling_snapshot"}' > /var/lib/vpn-node/e2e/status.json
        while [ -f /var/lib/vpn-node/e2e/hold-enrolling ]; do sleep 1; done
    fi
    printf '%s\n' '{"stage":"enrolling"}' > /var/lib/vpn-node/e2e/status.json
    if ! fvpn-agent enroll -config /etc/vpn-node/agent-enroll.json -token-file /var/lib/vpn-node/e2e/enrollment.token; then
        printf '%s\n' '{"stage":"enroll_failed"}' > /var/lib/vpn-node/e2e/status.json
        exit 1
    fi
fi
# PIDs belong to the old container process tree, not the durable identity.
# The previous container may have killed the waiter too. Keep historical exit
# receipts, but never make a stop before this boot's first start wait on that ID.
rm -f /var/lib/vpn-node/e2e/agent.pid /var/lib/vpn-node/e2e/agent-process-id
fvpn-e2e-nodeconfig
# TEST ONLY loopback mTLS transport; corrupts only when the private local marker is armed.
fvpn-e2e-nodeconfig dns-relay >/var/lib/vpn-node/e2e/dns-relay.log 2>&1 &
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
    # TEST ONLY diagnostic, armed before enrollment by Followup25. Scope this
    # environment setting to the lab daemon; the ordinary kernel path is unchanged.
    if [ -f /var/lib/vpn-node/e2e/test-only-userspace-wireguard ]; then
        # Consume the opt-in: a later ordinary container boot uses the baseline.
        rm -f /var/lib/vpn-node/e2e/test-only-userspace-wireguard
        NB_WG_KERNEL_DISABLED=true netbird service run --log-level warn --log-file console >/var/lib/vpn-node/e2e/netbird-service.log 2>&1 &
    else
        netbird service run --log-level warn --log-file console >/var/lib/vpn-node/e2e/netbird-service.log 2>&1 &
    fi
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
