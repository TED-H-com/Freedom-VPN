#!/usr/bin/env bash
# Freedom VPN Central PostgreSQL Backup Operator Tool (`deploy/backup/backup.sh`)
# Spec references: §3.1, §15.4, §16.2, §16.4, §17.2, AT30 (SECOPS-05)
#
# Usage:
#   deploy/backup/backup.sh [--stanza fvpn] [--container <name>] [--db-user fvpn] [--db-name fvpn] <command>
#
# Commands:
#   stanza-create   Create/upgrade the pgBackRest stanza and verify WAL archiving
#   check           Verify archive_mode=on and run pgBackRest archive check
#   full            Execute a physical full base backup
#   diff            Execute a physical differential base backup
#   wal-switch      Force a PostgreSQL WAL segment switch and verify archiving
#   info            Print pgBackRest repository inventory as JSON
set -euo pipefail

STANZA="${FVPN_PGBACKREST_STANZA:-fvpn}"
CONTAINER="${FVPN_BACKUP_CONTAINER:-}"
DB_USER="${FVPN_BACKUP_DB_USER:-}"
DB_NAME="${FVPN_BACKUP_DB_NAME:-fvpn}"
ACTION=""

usage() {
	cat >&2 <<'EOF'
Usage: backup.sh [options] <stanza-create|check|full|diff|wal-switch|info>

Options:
  --stanza <name>       pgBackRest stanza name (default: fvpn)
  --container <name>    Execute inside the named PostgreSQL Docker container
  --db-user <user>      PostgreSQL user (default: fvpn in container, postgres on host)
  --db-name <db>        PostgreSQL database name (default: fvpn)
  --type <full|diff>    Backup type alias for full or diff
  -h, --help            Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--stanza)
		STANZA="${2:-}"
		shift 2
		;;
	--container)
		CONTAINER="${2:-}"
		shift 2
		;;
	--db-user)
		DB_USER="${2:-}"
		shift 2
		;;
	--db-name)
		DB_NAME="${2:-}"
		shift 2
		;;
	--type)
		ACTION="${2:-}"
		shift 2
		;;
	stanza-create | check | full | diff | wal-switch | info)
		ACTION="$1"
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "backup.sh: unknown argument: $1" >&2
		usage
		exit 2
		;;
	esac
done

if [[ -z "$ACTION" ]]; then
	echo "backup.sh: action is required" >&2
	usage
	exit 2
fi

if [[ ! "$STANZA" =~ ^[a-zA-Z0-9_-]+$ ]]; then
	echo "backup.sh: invalid stanza name: $STANZA" >&2
	exit 2
fi

if [[ -n "$CONTAINER" ]] && [[ ! "$CONTAINER" =~ ^[a-zA-Z0-9_.-]+$ ]]; then
	echo "backup.sh: invalid container name: $CONTAINER" >&2
	exit 2
fi

if [[ -z "$DB_USER" ]]; then
	if [[ -n "$CONTAINER" ]]; then
		DB_USER="fvpn"
	else
		DB_USER="postgres"
	fi
fi

run_pgbackrest() {
	if [[ -n "$CONTAINER" ]]; then
		docker exec --user postgres "$CONTAINER" pgbackrest --stanza="$STANZA" "$@"
	else
		pgbackrest --stanza="$STANZA" "$@"
	fi
}

run_psql() {
	if [[ -n "$CONTAINER" ]]; then
		docker exec --user postgres "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 "$@"
	else
		psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 "$@"
	fi
}

assert_archive_mode_on() {
	local mode
	mode="$(run_psql -tAc "SHOW archive_mode;" | tr -d '[:space:]')"
	if [[ "$mode" != "on" ]] && [[ "$mode" != "always" ]]; then
		echo "backup.sh: refusing to proceed: PostgreSQL archive_mode is '${mode}' (expected 'on')" >&2
		exit 1
	fi
}

case "$ACTION" in
stanza-create)
	assert_archive_mode_on
	run_pgbackrest stanza-create
	run_pgbackrest check
	;;
check)
	assert_archive_mode_on
	run_pgbackrest check
	;;
full)
	assert_archive_mode_on
	run_pgbackrest --type=full backup
	run_pgbackrest verify
	run_pgbackrest --output=json info
	;;
diff)
	assert_archive_mode_on
	run_pgbackrest --type=diff backup
	run_pgbackrest verify
	run_pgbackrest --output=json info
	;;
wal-switch)
	assert_archive_mode_on
	run_psql -tAc "SELECT pg_switch_wal();" >/dev/null
	run_pgbackrest check
	;;
info)
	run_pgbackrest --output=json info
	;;
*)
	echo "backup.sh: unsupported action: $ACTION" >&2
	exit 2
	;;
esac
