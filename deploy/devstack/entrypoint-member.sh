#!/bin/sh
# Member container (§8). Joins NetBird with the embedded IdP user flow when
# e2e/user-auth appears. That file is only a trigger. The up log can contain
# a short-lived device URL; the harness reads it and deletes it.
set -eu
umask 077
mkdir -p /var/lib/netbird /var/lib/vpn-node/e2e /var/log/netbird
if [ -f /etc/vpn-node/ca.pem ]; then
    cp /etc/vpn-node/ca.pem /usr/local/share/ca-certificates/lab-ca.crt
    update-ca-certificates >/dev/null 2>&1 || true
fi
export SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
printf '%s\n' '{"stage":"waiting_user_auth"}' > /var/lib/vpn-node/e2e/status.json
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
    if [ -f /var/lib/vpn-node/e2e/user-auth ]; then
        rm -f /var/lib/vpn-node/e2e/user-auth
        printf '%s\n' '{"stage":"user_auth_started"}' > /var/lib/vpn-node/e2e/status.json
        if netbird up --management-url https://netbird.example.invalid:443 \
            --hostname fvpn-member-e2e \
            --interface-name fvnb0 --disable-dns --disable-ipv6 --no-browser \
            --log-level info --log-file console > /var/lib/vpn-node/e2e/netbird-up.log 2>&1; then
            printf '%s\n' '{"stage":"netbird_up"}' > /var/lib/vpn-node/e2e/status.json
        else
            printf '%s\n' '{"stage":"netbird_failed"}' > /var/lib/vpn-node/e2e/status.json
        fi
    fi
    sleep 1
done
