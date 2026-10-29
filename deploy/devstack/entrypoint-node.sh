#!/bin/sh
# Node container entrypoint (§5, §8, §13). Creates only fv* interfaces inside
# this container. Never prints the enrollment token or the setup key.
set -eu
umask 077
mkdir -p /run/vpn-node /var/lib/vpn-node/state /var/lib/vpn-node/identity \
    /var/lib/vpn-node/update-journal /var/lib/vpn-node/e2e /opt/vpn-node /var/log/netbird /etc/vpn-node
if ! ip link show fvup0 >/dev/null 2>&1; then
    ip link add fvup0 type dummy
    ip addr add 192.0.2.20/32 dev fvup0
    ip link set fvup0 up
fi
if ! ip link show fvmb0 >/dev/null 2>&1; then
    ip link add fvmb0 type dummy
    ip addr add 100.64.0.2/32 dev fvmb0
    ip link set fvmb0 up
fi
if [ -f /etc/vpn-node/ca.pem ]; then
    cp /etc/vpn-node/ca.pem /usr/local/share/ca-certificates/lab-ca.crt
    update-ca-certificates >/dev/null 2>&1 || true
fi
export SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
cat /proc/sys/kernel/random/uuid > /etc/vpn-node/boot-id
epoch=$(cat /proc/sys/kernel/random/uuid)
epoch2=$(cat /proc/sys/kernel/random/uuid)
printf '{"closed_meter_epoch_id":"%s","last_durable_event_seq":"0","central_acked_through":"0","new_meter_epoch_id":"%s"}\n' "$epoch" "$epoch2" > /var/lib/vpn-node/e2e/meter.json
printf '%s\n' '{"stage":"waiting_token"}' > /var/lib/vpn-node/e2e/status.json
while [ ! -s /var/lib/vpn-node/e2e/enrollment.token ]; do
    sleep 1
done
printf '%s\n' '{"stage":"enrolling"}' > /var/lib/vpn-node/e2e/status.json
if ! fvpn-agent enroll -config /etc/vpn-node/agent-enroll.json -token-file /var/lib/vpn-node/e2e/enrollment.token; then
    printf '%s\n' '{"stage":"enroll_failed"}' > /var/lib/vpn-node/e2e/status.json
    exit 1
fi
printf '%s\n' '{"stage":"enrolled"}' > /var/lib/vpn-node/e2e/status.json
fvpn-watchdog run -config /etc/vpn-node/agent-run.json >/var/lib/vpn-node/e2e/watchdog.log 2>&1 &
(
    while true; do
        if [ -f /var/lib/vpn-node/e2e/pause-agent ]; then
            sleep 1
            continue
        fi
        fvpn-agent run -config /etc/vpn-node/agent-run.json >/var/lib/vpn-node/e2e/agent.log 2>&1 &
        pid=$!
        while kill -0 "$pid" 2>/dev/null; do
            if [ -f /var/lib/vpn-node/e2e/pause-agent ]; then
                kill "$pid" 2>/dev/null || true
                wait "$pid" 2>/dev/null || true
                pid=""
                break
            fi
            sleep 1
        done
        if [ -n "${pid:-}" ]; then
            wait "$pid" 2>/dev/null || true
            printf '%s\n' '{"stage":"agent_exited"}' > /var/lib/vpn-node/e2e/agent.json
            sleep 2
        fi
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
        --interface-name fvnb0 --disable-dns --disable-ipv6 \
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
