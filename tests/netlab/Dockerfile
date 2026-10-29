# Network lab image for Gate G2. Tools only; the topology is built at runtime
# inside this container's network namespaces. Never used with --network host.
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        nftables \
        iproute2 \
        wireguard-tools \
        conntrack \
        iperf3 \
        tcpdump \
        ethtool \
        python3 \
        iputils-ping \
        procps \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# The host AppArmor profile "wg" attaches to the path /usr/bin/wg
# (flags=attach_disconnected) even inside a privileged container, and it
# allows fopen of key files only under /etc/wireguard. A copy at another
# path is not attached to that profile. topology.sh uses this binary.
RUN cp -a /usr/bin/wg /usr/local/bin/wg

COPY scripts/ /opt/netlab/scripts/

CMD ["sleep", "infinity"]
