#!/usr/bin/env bash
# Wrapper entrypoint for the Freedom VPN PostgreSQL 17 + pgBackRest image (§15.4, §17.2).
# Prepares pgBackRest directories, renders /etc/postgresql/conf.d/50-fvpn-archiving.conf
# (honoring TEST ONLY lab overrides or isolated restore mode), and delegates to
# the upstream docker-entrypoint.sh.
set -euo pipefail

STANZA="${FVPN_PGBACKREST_STANZA:-fvpn}"
if [[ ! "$STANZA" =~ ^[a-zA-Z0-9_-]+$ ]]; then
	echo "fvpn-postgres-entrypoint: invalid FVPN_PGBACKREST_STANZA" >&2
	exit 2
fi

ARCHIVE_TIMEOUT="${FVPN_PG_ARCHIVE_TIMEOUT:-300s}"
if [[ ! "$ARCHIVE_TIMEOUT" =~ ^[1-9][0-9]*(ms|s|min)$ ]]; then
	echo "fvpn-postgres-entrypoint: invalid FVPN_PG_ARCHIVE_TIMEOUT (expected e.g. 300s)" >&2
	exit 2
fi

ISOLATED_RESTORE="${FVPN_PG_ISOLATED_RESTORE:-0}"
PG_USER="${POSTGRES_USER:-postgres}"
PG_DB="${POSTGRES_DB:-$PG_USER}"
PGDATA_DIR="${PGDATA:-/var/lib/postgresql/data}"

if [[ "$(id -u)" == "0" ]]; then
	install -d -o postgres -g postgres -m 0750 \
		/var/log/pgbackrest \
		/var/spool/pgbackrest \
		/etc/pgbackrest \
		/etc/postgresql/conf.d
	# Only chown /var/lib/pgbackrest when not mounted read-only (isolated restores mount :ro).
	if [[ "$ISOLATED_RESTORE" != "1" ]] && [[ -w /var/lib/pgbackrest ]]; then
		chown -R postgres:postgres /var/lib/pgbackrest
		chmod 0750 /var/lib/pgbackrest
	fi
fi

ARCHIVING_CONF="/etc/postgresql/conf.d/50-fvpn-archiving.conf"
if [[ -w /etc/postgresql/conf.d ]]; then
	if [[ "$ISOLATED_RESTORE" == "1" ]]; then
		cat >"$ARCHIVING_CONF" <<EOF
# Isolated PITR verification instance (§17.2, AT30).
# Archiving is disabled so promoted verification timelines never write to the backup repository.
wal_level = replica
archive_mode = off
archive_timeout = 0
archive_command = ''
max_wal_senders = 5
EOF
	else
		cat >"$ARCHIVING_CONF" <<EOF
# Freedom VPN PostgreSQL 17 WAL Archiving Configuration (§3.1, §15.4, §17.2)
wal_level = replica
archive_mode = on
archive_timeout = ${ARCHIVE_TIMEOUT}
archive_command = 'pgbackrest --stanza=${STANZA} archive-push %p'
max_wal_senders = 5
EOF
	fi
	if [[ "$(id -u)" == "0" ]]; then
		chown postgres:postgres "$ARCHIVING_CONF"
		chmod 0644 "$ARCHIVING_CONF"
	fi
fi

# Ensure existing PGDATA postgresql.conf includes /etc/postgresql/conf.d.
if [[ -f "${PGDATA_DIR}/postgresql.conf" ]] && [[ -w "${PGDATA_DIR}/postgresql.conf" ]]; then
	if ! grep -qF "/etc/postgresql/conf.d" "${PGDATA_DIR}/postgresql.conf"; then
		printf "\ninclude_dir = '/etc/postgresql/conf.d'\n" >>"${PGDATA_DIR}/postgresql.conf"
	fi
fi

# Render default local POSIX pgBackRest config unless caller mounted a custom config
# and set FVPN_PGBACKREST_CUSTOM_CONFIG=1.
PGBACKREST_CONF="/etc/pgbackrest/pgbackrest.conf"
if [[ "${FVPN_PGBACKREST_CUSTOM_CONFIG:-0}" != "1" ]] && [[ -w /etc/pgbackrest ]]; then
	CIPHER_LINES=""
	if [[ -n "${FVPN_PGBACKREST_CIPHER_PASS:-}" ]]; then
		CIPHER_LINES=$'repo1-cipher-type=aes-256-cbc\nrepo1-cipher-pass='"${FVPN_PGBACKREST_CIPHER_PASS}"
	fi
	cat >"$PGBACKREST_CONF" <<EOF
# Freedom VPN pgBackRest container configuration (§17.2)
[global]
repo1-type=posix
repo1-path=/var/lib/pgbackrest
repo1-retention-full-type=count
repo1-retention-full=7
repo1-retention-diff=7
process-max=2
log-level-console=warn
log-level-file=info
log-path=/var/log/pgbackrest
spool-path=/var/spool/pgbackrest
start-fast=y
archive-async=n
compress-type=zst
${CIPHER_LINES}

[${STANZA}]
pg1-path=${PGDATA_DIR}
pg1-port=5432
pg1-user=${PG_USER}
pg1-database=${PG_DB}
pg1-socket-path=/var/run/postgresql
EOF
	if [[ "$(id -u)" == "0" ]]; then
		chown postgres:postgres "$PGBACKREST_CONF"
		chmod 0640 "$PGBACKREST_CONF"
	fi
fi

exec /usr/local/bin/docker-entrypoint.sh "$@"
