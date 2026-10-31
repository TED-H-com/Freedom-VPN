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

# Exercise the real pinned daemon and its Unix socket under the product unit.
# Only `up` is intercepted: this suite has a fake enrollment control and no
# NetBird management server. Full peer enrollment belongs to the devstack lab.
RUN curl -fsSL https://pkgs.netbird.io/debian/public.key -o /tmp/netbird.asc \
    && gpg --batch --dearmor -o /usr/share/keyrings/netbird.gpg /tmp/netbird.asc \
    && echo 'deb [signed-by=/usr/share/keyrings/netbird.gpg] https://pkgs.netbird.io/debian stable main' > /etc/apt/sources.list.d/netbird.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends netbird=0.82.0 \
    && systemctl disable netbird.service \
    && systemctl mask netbird.service \
    && mkdir -p /usr/lib/fvpn-tests \
    && mv /usr/bin/netbird /usr/lib/fvpn-tests/netbird \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/netbird.asc
COPY netbird-stub.sh /usr/bin/netbird
RUN chmod 0755 /usr/bin/netbird

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
