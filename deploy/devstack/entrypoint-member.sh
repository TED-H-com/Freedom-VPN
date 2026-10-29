#!/bin/sh
# Member container (§8). Brings up a NetBird client when a setup-key file appears.
# The key file is never printed.
set -eu
umask 077
mkdir -p /var/lib/netbird /var/lib/vpn-node/e2e /var/log/netbird
if [ -f /etc/vpn-node/ca.pem ]; then
    cp /etc/vpn-node/ca.pem /usr/local/share/ca-certificates/lab-ca.crt
    update-ca-certificates >/dev/null 2>&1 || true
fi
export SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
printf '%s\n' '{"stage":"waiting_setup_key"}' > /var/lib/vpn-node/e2e/status.json
netbird service run --log-level warn --log-file console >/var/lib/vpn-node/e2e/netbird-service.log 2>&1 &
i=0
while [ "$i" -lt 40 ]; do
    if netbird status --check live >/dev/null 2>&1; then
        break
    fi
    i=$((i + 1))
    sleep 1
done
while true; do
    if [ -s /var/lib/vpn-node/e2e/setup.key ]; then
        if netbird up --setup-key-file /var/lib/vpn-node/e2e/setup.key \
            --management-url https://netbird.example.invalid:443 \
            --interface-name fvnb0 --disable-dns --disable-ipv6 \
            --log-level warn --log-file console >/var/lib/vpn-node/e2e/netbird-up.log 2>&1; then
            printf '%s\n' '{"stage":"netbird_up"}' > /var/lib/vpn-node/e2e/status.json
        else
            printf '%s\n' '{"stage":"netbird_failed"}' > /var/lib/vpn-node/e2e/status.json
        fi
        rm -f /var/lib/vpn-node/e2e/setup.key
    fi
    sleep 2
done
