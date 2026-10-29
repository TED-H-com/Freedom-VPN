#!/bin/sh
set -eu
umask 077
# TEST ONLY. Never start a pristine client against the public cloud defaults.
if [ ! -f /var/lib/netbird/default.json ]; then
    cat > /var/lib/netbird/default.json <<JSON
{"ManagementURL":{"Scheme":"http","Host":"netbird.example.invalid:${FVPN_NB_API_PORT}"},"AdminURL":{"Scheme":"http","Host":"netbird.example.invalid:${FVPN_NB_API_PORT}"},"PrivateKey":"","WgIface":"fvnb0","DisableDNS":true,"DisableIPv6":true}
JSON
fi
exec netbird service run
