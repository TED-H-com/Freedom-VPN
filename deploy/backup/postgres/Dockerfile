# Freedom VPN Central PostgreSQL 17 + pgBackRest Image (§3.1, §15.4, §17.2, SECOPS-05, AT30)
#
# Drop-in replacement for the central PostgreSQL 17 container image with
# pgBackRest pinned and continuous WAL archiving enabled (`archive_mode=on`,
# default production `archive_timeout=300s`).
FROM postgres@sha256:31ba9c94a03ada9988f2bb493a22a75eef3a185884533e5cdbcd498ea835b08f

ARG PGBACKREST_VERSION=2.60.0-1.pgdg13+1

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        "pgbackrest=${PGBACKREST_VERSION}" \
        ca-certificates; \
    rm -rf /var/lib/apt/lists/*; \
    install -d -o postgres -g postgres -m 0750 \
        /var/lib/pgbackrest \
        /var/log/pgbackrest \
        /var/spool/pgbackrest \
        /etc/pgbackrest \
        /etc/postgresql/conf.d; \
    if ! grep -qF "include_dir = '/etc/postgresql/conf.d'" /usr/share/postgresql/postgresql.conf.sample; then \
        printf "\ninclude_dir = '/etc/postgresql/conf.d'\n" >> /usr/share/postgresql/postgresql.conf.sample; \
    fi

COPY --chmod=0644 fvpn-archiving.conf /etc/postgresql/conf.d/50-fvpn-archiving.conf
COPY --chmod=0755 fvpn-postgres-entrypoint.sh /usr/local/bin/fvpn-postgres-entrypoint.sh
COPY --chmod=0755 10-pgbackrest-stanza.sh /docker-entrypoint-initdb.d/10-pgbackrest-stanza.sh

ENTRYPOINT ["/usr/local/bin/fvpn-postgres-entrypoint.sh"]
CMD ["postgres"]
