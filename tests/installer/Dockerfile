# Privileged systemd lab for fvpn-node-install. TEST ONLY. Not a product image.
# systemd is PID 1 so the installer can enable and start the real units (§6.3).
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV container=docker

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        systemd \
        systemd-sysv \
        nftables \
        iproute2 \
        conntrack \
        ca-certificates \
        curl \
        zstd \
        tar \
        gpg \
        dbus \
        procps \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && printf '0123456789abcdef0123456789abcdef\n' > /etc/machine-id \
    && systemctl mask \
        systemd-resolved.service \
        systemd-networkd-wait-online.service \
        apt-daily.service \
        apt-daily.timer \
        apt-daily-upgrade.service \
        apt-daily-upgrade.timer \
        unattended-upgrades.service \
        fstrim.service \
        fstrim.timer \
    && systemctl set-default multi-user.target

# TEST ONLY stand-in for the pinned NetBird client. The real package is
# installed only when this binary is absent (see packaging/installer/netbird.go).
COPY netbird-stub.sh /usr/bin/netbird
RUN chmod 0755 /usr/bin/netbird

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
