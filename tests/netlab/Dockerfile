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

COPY scripts/ /opt/netlab/scripts/

CMD ["sleep", "infinity"]
